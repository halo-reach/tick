package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/tickplatform/tick/internal/credential"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/hook"
	"github.com/tickplatform/tick/internal/repo"
)

const TypeTrigger = "task:trigger"

type TriggerPayload struct {
	TaskID       string             `json:"task_id"`
	TenantID     string             `json:"tenant_id"`
	TriggerTime  time.Time          `json:"trigger_time"`
	IsMakeup     bool               `json:"is_makeup,omitempty"`
	ExecutionID  string             `json:"execution_id,omitempty"`
	RetryBackoff domain.RetryBackoff `json:"retry_backoff,omitempty"`
}

type Handler struct {
	taskRepo     *repo.TaskRepo
	targetRepo   *repo.TargetRepo
	execRepo     *repo.ExecutionRepo
	secretRepo   *repo.SecretRepo
	variableRepo *repo.VariableRepo
	rdb          *redis.Client
	executor     *HTTPExecutor
	guard        *ConcurrencyGuard
	enqueuer     *Enqueuer
	resolver     *credential.Resolver
	hookEngine   *hook.Engine
}

func NewHandler(
	taskRepo *repo.TaskRepo,
	targetRepo *repo.TargetRepo,
	execRepo *repo.ExecutionRepo,
	secretRepo *repo.SecretRepo,
	variableRepo *repo.VariableRepo,
	rdb *redis.Client,
	guard *ConcurrencyGuard,
	enqueuer *Enqueuer,
	resolver *credential.Resolver,
	hookEngine *hook.Engine,
) *Handler {
	return &Handler{
		taskRepo:     taskRepo,
		targetRepo:   targetRepo,
		execRepo:     execRepo,
		secretRepo:   secretRepo,
		variableRepo: variableRepo,
		rdb:          rdb,
		executor:     NewHTTPExecutor(),
		guard:        guard,
		enqueuer:     enqueuer,
		resolver:     resolver,
		hookEngine:   hookEngine,
	}
}

func (h *Handler) HandleTrigger(ctx context.Context, t *asynq.Task) error {
	var p TriggerPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	task, err := h.taskRepo.GetByID(ctx, p.TaskID, p.TenantID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if task.Status != domain.TaskActive {
		slog.Info("skipping inactive task", "task_id", p.TaskID)
		return nil
	}

	target, err := h.targetRepo.GetByID(ctx, task.TargetID, task.TenantID)
	if err != nil {
		return fmt.Errorf("get target: %w", err)
	}

	secret, _ := h.secretRepo.GetActivByTenant(ctx, p.TenantID)
	signingSecret := ""
	if secret != nil {
		signingSecret = secret.Secret
	}

	// Execute pre-hooks
	vc := hook.NewVariableContext()
	vc.SetBuiltins(task.ID, task.Name, p.TenantID, p.TriggerTime.Format(time.RFC3339), p.ExecutionID)

	var hookInjections []domain.CredentialInjection
	var preHookResults []domain.HookResultEntry
	if h.hookEngine != nil && len(task.PreHooks) > 0 {
		var preHooks []domain.PreHook
		if err := json.Unmarshal(task.PreHooks, &preHooks); err == nil && len(preHooks) > 0 {
			preResults, hookInj, hookErr := h.hookEngine.ExecutePreHooks(ctx, preHooks, vc, p.TenantID)
			preHookResults = preResults
			if hookErr != nil {
				slog.Error("pre-hook failed", "task_id", p.TaskID, "error", hookErr)
				exec := &domain.Execution{
					TaskID:      p.TaskID,
					TenantID:    p.TenantID,
					TriggerTime: p.TriggerTime,
					Attempt:     1,
					Status:      domain.ExecFailed,
					ErrorMsg:    "pre-hook failed: " + hookErr.Error(),
					IsMakeup:    p.IsMakeup,
					CreatedAt:   time.Now(),
				}
				_ = h.execRepo.Create(ctx, exec)
				_ = h.taskRepo.IncrementExecutions(ctx, p.TaskID)
				if h.guard != nil {
					_ = h.guard.Release(ctx, p.TaskID)
					h.processQueue(ctx, task)
				}
				return fmt.Errorf("pre-hook failed: %w", hookErr)
			}
			hookInjections = hookInj
		}
	}

	var injections []domain.CredentialInjection
	injections = append(injections, hookInjections...)

	// Auto-resolve credential placeholders from target config
	if h.resolver != nil {
		var cfg domain.HTTPTargetConfig
		if err := json.Unmarshal(target.Config, &cfg); err == nil {
			// Resolve credential_ids into injections
			for _, credID := range cfg.CredentialIDs {
				resolved, err := h.resolver.Resolve(ctx, credID, p.TenantID)
				if err != nil {
					slog.Warn("resolve credential by id failed", "id", credID, "error", err)
					continue
				}
				injections = append(injections, credential.BuildInjections(resolved)...)
			}

			var scanSources []string
			scanSources = append(scanSources, cfg.URL)
			for _, v := range cfg.Headers {
				scanSources = append(scanSources, v)
			}
			if cfg.Body != nil {
				scanSources = append(scanSources, string(cfg.Body))
			}
			for _, src := range scanSources {
				for _, match := range hook.ExtractPlaceholders(src) {
					if _, exists := vc.Get(match); !exists {
						slog.Info("resolving credential placeholder", "code", match, "tenant_id", p.TenantID)
						resolved, err := h.resolver.ResolveByCode(ctx, match, p.TenantID)
						if err != nil {
							slog.Error("credential resolve failed, will retry", "code", match, "error", err)
							return fmt.Errorf("credential resolve failed for %q: %w", match, err)
						}
						value := credentialValueWithPrefix(resolved)
						vc.Set(match, value)
						slog.Info("credential placeholder resolved", "code", match)
					}
				}
			}
		} else {
			slog.Error("unmarshal target config for placeholder scan", "error", err)
		}
	}

	// Load tenant variables into VariableContext (Redis cached 60s)
	if h.variableRepo != nil {
		variables, err := h.loadVariables(ctx, p.TenantID)
		if err == nil {
			for k, v := range variables {
				if _, exists := vc.Get(k); !exists {
					vc.Set(k, v)
				}
			}
		}
	}

	result := h.executor.Execute(ctx, task, target, p.TriggerTime, signingSecret, p.ExecutionID, injections, vc)

	// Execute post-hooks
	var postHookResults []domain.HookResultEntry
	if h.hookEngine != nil && len(task.PostHooks) > 0 {
		var postHooks []domain.PostHook
		if err := json.Unmarshal(task.PostHooks, &postHooks); err == nil && len(postHooks) > 0 {
			vc.Set("execution_status", string(result.Status))
			vc.Set("status_code", fmt.Sprintf("%d", result.StatusCode))
			vc.Set("duration_ms", fmt.Sprintf("%d", result.DurationMs))
			vc.Set("error_msg", result.ErrorMsg)
			respBody := result.ResponseBody
			if len(respBody) > 4096 {
				respBody = respBody[:4096]
			}
			vc.Set("response_body", respBody)
			postHookResults = h.hookEngine.ExecutePostHooks(ctx, postHooks, vc, string(result.Status))
		}
	}

	// Build hooks_result
	var hooksResultJSON json.RawMessage
	hooksResult := domain.HooksResult{PreHooks: preHookResults, PostHooks: postHookResults}
	if len(hookInjections) > 0 {
		var injected []domain.CredentialInjected
		for _, inj := range hookInjections {
			injected = append(injected, domain.CredentialInjected{CredentialID: "", InjectKey: inj.Key, Status: "success"})
		}
		hooksResult.CredentialsInjected = injected
	}
	if hooksResult.PreHooks != nil || hooksResult.PostHooks != nil || hooksResult.CredentialsInjected != nil {
		hooksResultJSON, _ = json.Marshal(hooksResult)
	}

	retryCount, _ := asynq.GetRetryCount(ctx)

	exec := &domain.Execution{
		TaskID:       p.TaskID,
		TenantID:     p.TenantID,
		TriggerTime:  p.TriggerTime,
		Attempt:      retryCount + 1,
		Status:         result.Status,
		StatusCode:     result.StatusCode,
		DurationMs:     result.DurationMs,
		RequestHeaders: result.RequestHeaders,
		RequestBody:    result.RequestBody,
		ResponseBody:   result.ResponseBody,
		ErrorMsg:       result.ErrorMsg,
		IsMakeup:     p.IsMakeup,
		HooksResult:  hooksResultJSON,
		CreatedAt:    time.Now(),
	}
	if err := h.execRepo.Create(ctx, exec); err != nil {
		slog.Error("save execution", "error", err)
	}
	_ = h.taskRepo.IncrementExecutions(ctx, p.TaskID)

	// Release concurrency slot
	if h.guard != nil {
		_ = h.guard.Release(ctx, p.TaskID)
		// Process queued triggers
		h.processQueue(ctx, task)
	}

	if result.Status != domain.ExecSuccess {
		return fmt.Errorf("execution failed: %s", result.ErrorMsg)
	}
	return nil
}

func (h *Handler) processQueue(ctx context.Context, task *domain.Task) {
	if task.ConcurrencyPolicy != domain.ConcurrencyQueue {
		return
	}
	data, err := h.guard.QueuePop(ctx, task.ID)
	if err != nil || data == nil {
		return
	}
	qt, err := UnmarshalQueuedTrigger(data)
	if err != nil {
		return
	}
	if h.enqueuer != nil {
		h.enqueuer.EnqueueDirect(task, qt.TriggerTime, qt.ExecutionID)
	}
}

func credentialValueWithPrefix(resolved *credential.ResolvedCredential) string {
	if resolved.Secret != "" {
		return resolved.Secret
	}
	return resolved.Token
}


func (h *Handler) loadVariables(ctx context.Context, tenantID string) (map[string]string, error) {
	cacheKey := "tick:variables:" + tenantID
	if h.rdb != nil {
		cached, err := h.rdb.HGetAll(ctx, cacheKey).Result()
		if err == nil && len(cached) > 0 {
			return cached, nil
		}
	}
	vars, err := h.variableRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(vars))
	for _, v := range vars {
		result[v.Key] = v.Value
	}
	if h.rdb != nil && len(result) > 0 {
		pipe := h.rdb.Pipeline()
		pipe.Del(ctx, cacheKey)
		for k, v := range result {
			pipe.HSet(ctx, cacheKey, k, v)
		}
		pipe.Expire(ctx, cacheKey, 60*time.Second)
		_, _ = pipe.Exec(ctx)
	}
	return result, nil
}

package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/tickplatform/tick/internal/api/middleware"
	"github.com/tickplatform/tick/internal/credential"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/hook"
	"github.com/tickplatform/tick/internal/repo"
	"github.com/tickplatform/tick/internal/scheduler"
	"github.com/tickplatform/tick/internal/worker"
)

type TaskHandler struct {
	taskRepo     *repo.TaskRepo
	targetRepo   *repo.TargetRepo
	tenantRepo   *repo.TenantRepo
	execRepo     *repo.ExecutionRepo
	secretRepo   *repo.SecretRepo
	credRepo     *repo.CredentialRepo
	variableRepo *repo.VariableRepo
	scheduler    *scheduler.Scheduler
	resolver     *credential.Resolver
}

func NewTaskHandler(tr *repo.TaskRepo, tgr *repo.TargetRepo, tnr *repo.TenantRepo, er *repo.ExecutionRepo, sr *repo.SecretRepo, cr *repo.CredentialRepo, vr *repo.VariableRepo, s *scheduler.Scheduler, resolver *credential.Resolver) *TaskHandler {
	return &TaskHandler{taskRepo: tr, targetRepo: tgr, tenantRepo: tnr, execRepo: er, secretRepo: sr, credRepo: cr, variableRepo: vr, scheduler: s, resolver: resolver}
}

type CreateTaskRequest struct {
	Name                   string            `json:"name" binding:"required"`
	ScheduleType           string            `json:"schedule_type" binding:"required"`
	CronExpr               string            `json:"cron_expr"`
	IntervalValue          int               `json:"interval_value"`
	IntervalUnit           string            `json:"interval_unit"`
	OnceAt                 *string           `json:"once_at"`
	TargetID               string            `json:"target_id"`
	URL                    string            `json:"url"`
	Method                 string            `json:"method"`
	Headers                map[string]string `json:"headers"`
	Body                   json.RawMessage   `json:"body"`
	ContentType            string            `json:"content_type"`
	CredentialIDs          []string          `json:"credential_ids"`
	TimeoutSecs            int               `json:"timeout_secs"`
	RetryCount             int               `json:"retry_count"`
	RetryBackoff           string            `json:"retry_backoff"`
	ConcurrencyPolicy      string            `json:"concurrency_policy"`
	MaxConcurrency         int               `json:"max_concurrency"`
	ExecutionRetentionDays int               `json:"execution_retention_days"`
	MissedPolicy           string            `json:"missed_policy"`
	PreHooks               json.RawMessage   `json:"pre_hooks"`
	PostHooks              json.RawMessage   `json:"post_hooks"`
}

var cronParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func (h *TaskHandler) Create(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	tenantID := middleware.GetTenantID(c)

	if req.TargetID == "" && req.URL == "" {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Either target_id or url is required")
		return
	}
	if req.TargetID == "" && req.URL != "" {
		method := req.Method
		if method == "" {
			method = "POST"
		}
		cfg, _ := json.Marshal(domain.HTTPTargetConfig{
			URL:           req.URL,
			Method:        method,
			Headers:       req.Headers,
			Body:          req.Body,
			ContentType:   req.ContentType,
			CredentialIDs: req.CredentialIDs,
		})
		now := time.Now()
		target := &domain.Target{
			ID:        generateID("tgt_", 12),
			TenantID:  tenantID,
			Name:      req.Name + " target",
			Type:      domain.TargetHTTP,
			Config:    cfg,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := h.targetRepo.Create(c.Request.Context(), target); err != nil {
			log.Printf("[Create] targetRepo.Create failed: %v", err)
			RespondError(c, http.StatusInternalServerError, ErrInternal)
			return
		}
		req.TargetID = target.ID
	}

	count, err := h.tenantRepo.CountTasksByTenant(c.Request.Context(), tenantID)
	if err != nil {
		log.Printf("[Create] CountTasksByTenant failed: %v", err)
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		log.Printf("[Create] GetByID tenant failed: %v", err)
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	if count >= tenant.QuotaMaxTasks {
		RespondError(c, http.StatusForbidden, ErrQuotaExceeded)
		return
	}

	if _, err := h.targetRepo.GetByID(c.Request.Context(), req.TargetID, tenantID); err != nil {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Target not found")
		return
	}

	now := time.Now()
	task := &domain.Task{
		ID:                     generateID("t_", 11),
		TenantID:               tenantID,
		Name:                   req.Name,
		ScheduleType:           domain.ScheduleType(req.ScheduleType),
		CronExpr:               req.CronExpr,
		TargetID:               req.TargetID,
		TimeoutSecs:            req.TimeoutSecs,
		RetryCount:             req.RetryCount,
		RetryBackoff:           domain.RetryBackoff(req.RetryBackoff),
		ConcurrencyPolicy:      domain.ConcurrencyPolicy(req.ConcurrencyPolicy),
		MaxConcurrency:         req.MaxConcurrency,
		ExecutionRetentionDays: req.ExecutionRetentionDays,
		MissedPolicy:           domain.MissedPolicy(req.MissedPolicy),
		Status:                 domain.TaskActive,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if task.TimeoutSecs == 0 {
		task.TimeoutSecs = 30
	}
	if task.RetryCount == 0 {
		task.RetryCount = 3
	}
	if task.RetryBackoff == "" {
		task.RetryBackoff = domain.BackoffExponential
	}
	if task.ConcurrencyPolicy == "" {
		task.ConcurrencyPolicy = domain.ConcurrencySkip
	}
	if task.MaxConcurrency <= 0 {
		task.MaxConcurrency = 1
	}
	if task.ExecutionRetentionDays <= 0 {
		task.ExecutionRetentionDays = 30
	}
	if task.ExecutionRetentionDays > 90 {
		task.ExecutionRetentionDays = 90
	}
	if task.MissedPolicy == "" {
		task.MissedPolicy = domain.MissedSkip
	}

	switch task.ScheduleType {
	case domain.ScheduleCron:
		if task.CronExpr == "" {
			RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "cron_expr required for cron schedule")
			return
		}
		// 兼容 5 位 cron（无秒），自动补 "0 " 前缀
		expr := task.CronExpr
		if len(strings.Fields(expr)) == 5 {
			expr = "0 " + expr
			task.CronExpr = expr
		}
		sched, err := cronParser.Parse(task.CronExpr)
		if err != nil {
			RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid cron expression")
			return
		}
		next := sched.Next(now)
		task.NextTriggerAt = &next
	case domain.ScheduleInterval:
		task.IntervalValue = req.IntervalValue
		task.IntervalUnit = domain.IntervalUnit(req.IntervalUnit)
		if task.IntervalValue <= 0 {
			RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "interval_value must be positive")
			return
		}
		next := now.Add(intervalDuration(task.IntervalValue, task.IntervalUnit))
		task.NextTriggerAt = &next
	case domain.ScheduleOnce:
		if req.OnceAt == nil {
			RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "once_at required for once schedule")
			return
		}
		t, err := time.Parse(time.RFC3339, *req.OnceAt)
		if err != nil {
			RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid once_at format (RFC3339)")
			return
		}
		task.OnceAt = &t
		task.NextTriggerAt = &t
	default:
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid schedule_type")
		return
	}

	if errMsg := h.validateHooks(c, tenantID, req.PreHooks, req.PostHooks); errMsg != "" {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", errMsg)
		return
	}
	task.PreHooks = req.PreHooks
	task.PostHooks = req.PostHooks

	if err := h.taskRepo.Create(c.Request.Context(), task); err != nil {
		log.Printf("[Create] taskRepo.Create failed: %v", err)
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	if h.scheduler != nil {
		h.scheduler.AddTask(task)
	}

	RespondData(c, http.StatusCreated, task)
}

func (h *TaskHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tasks, total, err := h.taskRepo.ListByTenant(c.Request.Context(), tenantID, status, limit, offset)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks, "total": total})
}

func (h *TaskHandler) Get(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	task, err := h.taskRepo.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}
	RespondData(c, http.StatusOK, h.enrichTask(c, task, tenantID))
}

func (h *TaskHandler) Delete(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	if err := h.taskRepo.SoftDelete(c.Request.Context(), id, tenantID); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	if h.scheduler != nil {
		h.scheduler.RemoveTask(id)
	}
	RespondData(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *TaskHandler) Pause(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	if err := h.taskRepo.UpdateStatus(c.Request.Context(), id, tenantID, domain.TaskPaused); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	if h.scheduler != nil {
		h.scheduler.RemoveTask(id)
	}
	RespondData(c, http.StatusOK, gin.H{"paused": true})
}

func (h *TaskHandler) Resume(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	if err := h.taskRepo.UpdateStatus(c.Request.Context(), id, tenantID, domain.TaskActive); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	task, err := h.taskRepo.GetByID(c.Request.Context(), id, tenantID)
	if err == nil && h.scheduler != nil {
		h.scheduler.AddTask(task)
	}
	RespondData(c, http.StatusOK, gin.H{"resumed": true})
}

func (h *TaskHandler) Trigger(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	task, err := h.taskRepo.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}
	if task.Status == domain.TaskDeleted {
		RespondErrorMsg(c, http.StatusConflict, "CONFLICT", "task is deleted")
		return
	}

	target, err := h.targetRepo.GetByID(c.Request.Context(), task.TargetID, tenantID)
	if err != nil {
		RespondErrorMsg(c, http.StatusInternalServerError, "INTERNAL_ERROR", "target not found")
		return
	}

	var secret string
	if h.secretRepo != nil {
		if s, err := h.secretRepo.GetActivByTenant(c.Request.Context(), tenantID); err == nil {
			secret = s.Secret
		}
	}

	triggerTime := time.Now()
	executor := worker.NewHTTPExecutor()
	executionID := generateID("exec_", 16)

	// Build variable context and credential injections
	vc := hook.NewVariableContext()
	vc.SetBuiltins(task.ID, task.Name, tenantID, triggerTime.Format(time.RFC3339), executionID)

	var injections []domain.CredentialInjection
	var cfg domain.HTTPTargetConfig
	if err := json.Unmarshal(target.Config, &cfg); err == nil {
		// Resolve credential_ids into injections
		if h.resolver != nil {
			for _, credID := range cfg.CredentialIDs {
				resolved, err := h.resolver.Resolve(c.Request.Context(), credID, tenantID)
				if err != nil {
					log.Printf("[Trigger] resolve credential %s failed: %v", credID, err)
					continue
				}
				injections = append(injections, credential.BuildInjections(resolved)...)
			}

			// Auto-resolve {{placeholder}} patterns
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
						resolved, err := h.resolver.ResolveByCode(c.Request.Context(), match, tenantID)
						if err != nil {
							continue
						}
						vc.Set(match, credentialValueWithPrefix(resolved))
					}
				}
			}
		}
	}

	// Load tenant variables
	if h.variableRepo != nil {
		vars, err := h.variableRepo.ListByTenant(c.Request.Context(), tenantID)
		if err == nil {
			for _, v := range vars {
				if _, exists := vc.Get(v.Key); !exists {
					vc.Set(v.Key, v.Value)
				}
			}
		}
	}

	result := executor.Execute(c.Request.Context(), task, target, triggerTime, secret, executionID, injections, vc)

	exec := &domain.Execution{
		TaskID:       task.ID,
		TenantID:     tenantID,
		TriggerTime:  triggerTime,
		Attempt:      1,
		Status:       result.Status,
		StatusCode:   result.StatusCode,
		DurationMs:     result.DurationMs,
		RequestHeaders: result.RequestHeaders,
		RequestBody:    result.RequestBody,
		ResponseBody:   result.ResponseBody,
		ErrorMsg:     result.ErrorMsg,
		IsManual:     true,
		CreatedAt:    time.Now(),
	}
	h.execRepo.Create(c.Request.Context(), exec)

	RespondData(c, http.StatusOK, gin.H{"message": "task triggered", "status": result.Status, "duration_ms": result.DurationMs})
}

func (h *TaskHandler) History(c *gin.Context) {
	id := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	execs, err := h.execRepo.ListByTask(c.Request.Context(), id, limit, offset)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, execs)
}

type UpdateTaskRequest struct {
Name                   *string           `json:"name"`
	ScheduleType           *string           `json:"schedule_type"`
	CronExpr               *string           `json:"cron_expr"`
	IntervalValue          *int              `json:"interval_value"`
	IntervalUnit           *string           `json:"interval_unit"`
	OnceAt                 *string           `json:"once_at"`
	TimeoutSecs            *int              `json:"timeout_secs"`
	RetryCount             *int              `json:"retry_count"`
	RetryBackoff           *string           `json:"retry_backoff"`
	ConcurrencyPolicy      *string           `json:"concurrency_policy"`
	MaxConcurrency         *int              `json:"max_concurrency"`
	ExecutionRetentionDays *int              `json:"execution_retention_days"`
	URL                    *string           `json:"url"`
	Method                 *string           `json:"method"`
	Headers                map[string]string `json:"headers"`
	Body                   json.RawMessage   `json:"body"`
	ContentType            *string           `json:"content_type"`
	CredentialIDs          []string          `json:"credential_ids"`
	PreHooks               json.RawMessage   `json:"pre_hooks"`
	PostHooks              json.RawMessage   `json:"post_hooks"`
}

func (h *TaskHandler) Update(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	task, err := h.taskRepo.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.TimeoutSecs != nil {
		task.TimeoutSecs = *req.TimeoutSecs
	}
	if req.RetryCount != nil {
		task.RetryCount = *req.RetryCount
	}
	if req.RetryBackoff != nil {
		task.RetryBackoff = domain.RetryBackoff(*req.RetryBackoff)
	}
	if req.ConcurrencyPolicy != nil {
		task.ConcurrencyPolicy = domain.ConcurrencyPolicy(*req.ConcurrencyPolicy)
	}
	if req.MaxConcurrency != nil && *req.MaxConcurrency >= 1 {
		task.MaxConcurrency = *req.MaxConcurrency
	}
	if req.ExecutionRetentionDays != nil {
		days := *req.ExecutionRetentionDays
		if days < 1 {
			days = 1
		}
		if days > 90 {
			days = 90
		}
		task.ExecutionRetentionDays = days
	}

	now := time.Now()
	reschedule := false

	if req.ScheduleType != nil {
		task.ScheduleType = domain.ScheduleType(*req.ScheduleType)
	}

	switch task.ScheduleType {
	case domain.ScheduleCron:
		if req.CronExpr != nil {
			sched, err := cronParser.Parse(*req.CronExpr)
			if err != nil {
				RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid cron expression")
				return
			}
			task.CronExpr = *req.CronExpr
			next := sched.Next(now)
			task.NextTriggerAt = &next
			reschedule = true
		}
	case domain.ScheduleInterval:
		if req.IntervalValue != nil || req.IntervalUnit != nil {
			if req.IntervalValue != nil {
				task.IntervalValue = *req.IntervalValue
			}
			if req.IntervalUnit != nil {
				task.IntervalUnit = domain.IntervalUnit(*req.IntervalUnit)
			}
			next := now.Add(intervalDuration(task.IntervalValue, task.IntervalUnit))
			task.NextTriggerAt = &next
			reschedule = true
		}
	case domain.ScheduleOnce:
		if req.OnceAt != nil {
			t, err := time.Parse(time.RFC3339, *req.OnceAt)
			if err != nil {
				RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid once_at time")
				return
			}
			task.OnceAt = &t
			task.NextTriggerAt = &t
			reschedule = true
		}
	}

	if req.ScheduleType != nil {
		reschedule = true
	}

	if req.PreHooks != nil || req.PostHooks != nil {
		if errMsg := h.validateHooks(c, tenantID, req.PreHooks, req.PostHooks); errMsg != "" {
			RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", errMsg)
			return
		}
		if req.PreHooks != nil {
			task.PreHooks = req.PreHooks
		}
		if req.PostHooks != nil {
			task.PostHooks = req.PostHooks
		}
	}

	task.UpdatedAt = now

	if req.URL != nil || req.Method != nil || req.Headers != nil || req.Body != nil || req.ContentType != nil || req.CredentialIDs != nil {
		target, err := h.targetRepo.GetByID(c.Request.Context(), task.TargetID, tenantID)
		if err == nil && target.Type == domain.TargetHTTP {
			var cfg domain.HTTPTargetConfig
			_ = json.Unmarshal(target.Config, &cfg)
			if req.URL != nil {
				cfg.URL = *req.URL
			}
			if req.Method != nil {
				cfg.Method = *req.Method
			}
			if req.Headers != nil {
				cfg.Headers = req.Headers
			}
			if req.Body != nil {
				cfg.Body = req.Body
			}
			if req.ContentType != nil {
				cfg.ContentType = *req.ContentType
			}
			if req.CredentialIDs != nil {
				cfg.CredentialIDs = req.CredentialIDs
			}
			target.Config, _ = json.Marshal(cfg)
			target.UpdatedAt = now
			_ = h.targetRepo.Update(c.Request.Context(), target)
		}
	}

	if err := h.taskRepo.Update(c.Request.Context(), task); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	if reschedule && h.scheduler != nil {
		h.scheduler.RemoveTask(id)
		h.scheduler.AddTask(task)
	}

	RespondData(c, http.StatusOK, task)
}

func intervalDuration(value int, unit domain.IntervalUnit) time.Duration {
	switch unit {
	case domain.UnitSeconds:
		return time.Duration(value) * time.Second
	case domain.UnitMinutes:
		return time.Duration(value) * time.Minute
	case domain.UnitHours:
		return time.Duration(value) * time.Hour
	case domain.UnitDays:
		return time.Duration(value) * 24 * time.Hour
	default:
		return time.Duration(value) * time.Second
	}
}

type TaskResponse struct {
	*domain.Task
	TargetType  string            `json:"target_type,omitempty"`
	URL         string            `json:"url,omitempty"`
	Method      string            `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        json.RawMessage   `json:"body,omitempty"`
	ContentType   string            `json:"content_type,omitempty"`
	CredentialIDs []string          `json:"credential_ids,omitempty"`
}

func (h *TaskHandler) enrichTask(c *gin.Context, task *domain.Task, tenantID string) *TaskResponse {
	resp := &TaskResponse{Task: task}
	target, err := h.targetRepo.GetByID(c.Request.Context(), task.TargetID, tenantID)
	if err != nil {
		log.Printf("[enrichTask] failed to get target %s for tenant %s: %v", task.TargetID, tenantID, err)
		return resp
	}
	resp.TargetType = string(target.Type)
	if target.Type == domain.TargetHTTP {
		var cfg domain.HTTPTargetConfig
		if err := json.Unmarshal(target.Config, &cfg); err != nil {
			log.Printf("[enrichTask] failed to unmarshal config for target %s: %v", task.TargetID, err)
			return resp
		}
		resp.URL = cfg.URL
		resp.Method = cfg.Method
		resp.Headers = cfg.Headers
		resp.Body = cfg.Body
		resp.ContentType = cfg.ContentType
		resp.CredentialIDs = cfg.CredentialIDs
	}
	return resp
}

func (h *TaskHandler) validateHooks(c *gin.Context, tenantID string, preRaw, postRaw json.RawMessage) string {
	if preRaw != nil {
		var hooks []domain.PreHook
		if err := json.Unmarshal(preRaw, &hooks); err != nil {
			return "pre_hooks 格式无效"
		}
		if len(hooks) > 5 {
			return "pre_hooks 最多 5 个"
		}
		for _, h := range hooks {
			if h.Type != domain.HookTypeCredential && h.Type != domain.HookTypeHTTP {
				return "pre_hook 类型须为 credential 或 http"
			}
			if h.Type == domain.HookTypeCredential && h.CredentialID == "" {
				return "credential 类型的 pre_hook 须指定 credential_id"
			}
		}
		for _, hook := range hooks {
			if hook.Type == domain.HookTypeCredential && hook.CredentialID != "" {
				cred, err := h.credRepo.GetByID(c.Request.Context(), hook.CredentialID, tenantID)
				if err != nil {
					return "pre_hook 引用的凭证不存在：" + hook.CredentialID
				}
				if cred.Status != domain.CredStatusActive {
					return "pre_hook 引用的凭证未激活：" + hook.CredentialID
				}
			}
		}
	}
	if postRaw != nil {
		var hooks []domain.PostHook
		if err := json.Unmarshal(postRaw, &hooks); err != nil {
			return "post_hooks 格式无效"
		}
		if len(hooks) > 5 {
			return "post_hooks 最多 5 个"
		}
		for _, h := range hooks {
			if h.Type != domain.HookTypeHTTP && h.Type != domain.HookTypeFeishu {
				return "post_hook 类型须为 http 或 feishu"
			}
			if h.When != domain.HookWhenSuccess && h.When != domain.HookWhenFailure && h.When != domain.HookWhenAlways {
				return "post_hook when 须为 success、failure 或 always"
			}
		}
	}
	return ""
}

func credentialValueWithPrefix(resolved *credential.ResolvedCredential) string {
	if resolved.Secret != "" {
		return resolved.Secret
	}
	return resolved.Token
}


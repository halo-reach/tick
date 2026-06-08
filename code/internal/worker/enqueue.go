package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/repo"
)

type Enqueuer struct {
	client   *asynq.Client
	guard    *ConcurrencyGuard
	execRepo *repo.ExecutionRepo
	taskRepo *repo.TaskRepo
}

func NewEnqueuer(redisAddr, redisPassword string, db int, guard *ConcurrencyGuard, execRepo *repo.ExecutionRepo, taskRepo *repo.TaskRepo) *Enqueuer {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword, DB: db})
	return &Enqueuer{client: client, guard: guard, execRepo: execRepo, taskRepo: taskRepo}
}

func (e *Enqueuer) Close() error {
	return e.client.Close()
}

func (e *Enqueuer) Enqueue(task *domain.Task, triggerTime time.Time) {
	ctx := context.Background()

	if e.taskRepo != nil {
		current, err := e.taskRepo.GetStatus(ctx, task.ID)
		if err == nil && current != domain.TaskActive {
			slog.Info("enqueue skipped: task not active", "task_id", task.ID, "status", string(current))
			return
		}
	}

	executionID := uuid.New().String()

	if task.ConcurrencyPolicy != domain.ConcurrencyAllow && e.guard != nil {
		ttl := time.Duration(task.TimeoutSecs*(task.RetryCount+1)*2) * time.Second
		acquired, err := e.guard.Acquire(ctx, task.ID, task.MaxConcurrency, ttl)
		if err != nil {
			slog.Error("concurrency acquire", "task_id", task.ID, "error", err)
			return
		}
		if !acquired {
			switch task.ConcurrencyPolicy {
			case domain.ConcurrencySkip:
				e.recordSkipped(ctx, task, triggerTime, executionID)
				return
			case domain.ConcurrencyQueue:
				qt := QueuedTrigger{
					TaskID:      task.ID,
					TenantID:    task.TenantID,
					TriggerTime: triggerTime,
					ExecutionID: executionID,
				}
				ok, err := e.guard.QueuePush(ctx, task.ID, MarshalQueuedTrigger(qt))
				if err != nil || !ok {
					e.recordSkipped(ctx, task, triggerTime, executionID)
				}
				return
			}
		}
	}

	if err := e.enqueueTask(task, triggerTime, executionID); err != nil {
		if e.guard != nil && task.ConcurrencyPolicy != domain.ConcurrencyAllow {
			_ = e.guard.Release(ctx, task.ID)
		}
	}
}

// EnqueueDirect enqueues without concurrency check (used for queue drain).
func (e *Enqueuer) EnqueueDirect(task *domain.Task, triggerTime time.Time, executionID string) {
	_ = e.enqueueTask(task, triggerTime, executionID)
}

func (e *Enqueuer) enqueueTask(task *domain.Task, triggerTime time.Time, executionID string) error {
	payload := TriggerPayload{
		TaskID:       task.ID,
		TenantID:     task.TenantID,
		TriggerTime:  triggerTime,
		ExecutionID:  executionID,
		RetryBackoff: task.RetryBackoff,
	}
	b, _ := json.Marshal(payload)

	t := asynq.NewTask(TypeTrigger, b)
	_, err := e.client.EnqueueContext(context.Background(), t,
		asynq.Queue("default"),
		asynq.MaxRetry(task.RetryCount),
		asynq.Timeout(time.Duration(task.TimeoutSecs)*time.Second),
		asynq.TaskID(task.ID+":"+triggerTime.Format("20060102150405")),
		asynq.Unique(time.Duration(task.TimeoutSecs)*time.Second),
	)
	if err != nil {
		slog.Error("enqueue task", "task_id", task.ID, "error", err)
		return err
	}
	return nil
}

func (e *Enqueuer) recordSkipped(ctx context.Context, task *domain.Task, triggerTime time.Time, executionID string) {
	slog.Info("task skipped by concurrency policy", "task_id", task.ID, "policy", task.ConcurrencyPolicy)
	if e.execRepo != nil {
		exec := &domain.Execution{
			TaskID:      task.ID,
			TenantID:    task.TenantID,
			TriggerTime: triggerTime,
			Attempt:     0,
			Status:      domain.ExecSkipped,
			ErrorMsg:    "skipped: concurrency limit reached",
			CreatedAt:   time.Now(),
		}
		_ = e.execRepo.Create(ctx, exec)
	}
}

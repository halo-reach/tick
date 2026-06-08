package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type TaskRepo struct {
	pool *pgxpool.Pool
}

func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

func (r *TaskRepo) Create(ctx context.Context, t *domain.Task) error {
	preHooks := nullableJSON(t.PreHooks)
	postHooks := nullableJSON(t.PostHooks)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tasks (id, tenant_id, name, schedule_type, cron_expr, interval_value, interval_unit,
		 once_at, target_id, timeout_secs, retry_count, retry_backoff, concurrency_policy, max_concurrency,
		 execution_retention_days, missed_policy, status, next_trigger_at,
		 total_executions, created_at, updated_at, pre_hooks, post_hooks)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)`,
		t.ID, t.TenantID, t.Name, t.ScheduleType, t.CronExpr, t.IntervalValue, t.IntervalUnit,
		t.OnceAt, t.TargetID, t.TimeoutSecs, t.RetryCount, t.RetryBackoff, t.ConcurrencyPolicy, t.MaxConcurrency,
		t.ExecutionRetentionDays, t.MissedPolicy, t.Status, t.NextTriggerAt,
		t.TotalExecutions, t.CreatedAt, t.UpdatedAt, preHooks, postHooks)
	return err
}

func (r *TaskRepo) GetByID(ctx context.Context, id, tenantID string) (*domain.Task, error) {
	t := &domain.Task{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, schedule_type, cron_expr, interval_value, interval_unit,
		 once_at, target_id, timeout_secs, retry_count, retry_backoff, concurrency_policy, max_concurrency,
		 execution_retention_days, missed_policy, status, next_trigger_at,
		 total_executions, created_at, updated_at, deleted_at,
		 COALESCE(pre_hooks, '[]'), COALESCE(post_hooks, '[]')
		 FROM tasks WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, id, tenantID).
		Scan(&t.ID, &t.TenantID, &t.Name, &t.ScheduleType, &t.CronExpr, &t.IntervalValue, &t.IntervalUnit,
			&t.OnceAt, &t.TargetID, &t.TimeoutSecs, &t.RetryCount, &t.RetryBackoff, &t.ConcurrencyPolicy, &t.MaxConcurrency,
			&t.ExecutionRetentionDays, &t.MissedPolicy, &t.Status, &t.NextTriggerAt,
			&t.TotalExecutions, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt,
			&t.PreHooks, &t.PostHooks)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TaskRepo) ListByTenant(ctx context.Context, tenantID string, status string, limit, offset int) ([]*domain.Task, int, error) {
	countQuery := `SELECT COUNT(*) FROM tasks WHERE tenant_id = $1 AND deleted_at IS NULL`
	listQuery := `SELECT id, tenant_id, name, schedule_type, cron_expr, interval_value, interval_unit,
		 once_at, target_id, timeout_secs, retry_count, retry_backoff, concurrency_policy, max_concurrency,
		 execution_retention_days, missed_policy, status, next_trigger_at,
		 total_executions, created_at, updated_at,
		 COALESCE(pre_hooks, '[]'), COALESCE(post_hooks, '[]')
		 FROM tasks WHERE tenant_id = $1 AND deleted_at IS NULL`

	args := []any{tenantID}
	if status != "" {
		countQuery += ` AND status = $2`
		listQuery += ` AND status = $2`
		args = append(args, status)
	}
	listQuery += ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		t := &domain.Task{}
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.ScheduleType, &t.CronExpr, &t.IntervalValue, &t.IntervalUnit,
			&t.OnceAt, &t.TargetID, &t.TimeoutSecs, &t.RetryCount, &t.RetryBackoff, &t.ConcurrencyPolicy, &t.MaxConcurrency,
			&t.ExecutionRetentionDays, &t.MissedPolicy, &t.Status, &t.NextTriggerAt,
			&t.TotalExecutions, &t.CreatedAt, &t.UpdatedAt,
			&t.PreHooks, &t.PostHooks); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	return tasks, total, nil
}

func (r *TaskRepo) UpdateStatus(ctx context.Context, id, tenantID string, status domain.TaskStatus) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4 AND deleted_at IS NULL`,
		status, now, id, tenantID)
	return err
}

func (r *TaskRepo) SoftDelete(ctx context.Context, id, tenantID string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET status = 'deleted', deleted_at = $1, updated_at = $1 WHERE id = $2 AND tenant_id = $3`,
		now, id, tenantID)
	return err
}

func (r *TaskRepo) UpdateNextTrigger(ctx context.Context, id string, nextTriggerAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET next_trigger_at = $1, updated_at = NOW() WHERE id = $2`, nextTriggerAt, id)
	return err
}

func (r *TaskRepo) IncrementExecutions(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET total_executions = total_executions + 1, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *TaskRepo) GetStatus(ctx context.Context, id string) (domain.TaskStatus, error) {
	var status domain.TaskStatus
	err := r.pool.QueryRow(ctx, `SELECT status FROM tasks WHERE id = $1`, id).Scan(&status)
	return status, err
}

func (r *TaskRepo) LoadActive(ctx context.Context) ([]*domain.Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, schedule_type, cron_expr, interval_value, interval_unit,
		 once_at, target_id, timeout_secs, retry_count, retry_backoff, concurrency_policy, max_concurrency,
		 execution_retention_days, missed_policy, status, next_trigger_at,
		 total_executions, created_at, updated_at,
		 COALESCE(pre_hooks, '[]'), COALESCE(post_hooks, '[]')
		 FROM tasks WHERE status = 'active' AND deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		t := &domain.Task{}
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.ScheduleType, &t.CronExpr, &t.IntervalValue, &t.IntervalUnit,
			&t.OnceAt, &t.TargetID, &t.TimeoutSecs, &t.RetryCount, &t.RetryBackoff, &t.ConcurrencyPolicy, &t.MaxConcurrency,
			&t.ExecutionRetentionDays, &t.MissedPolicy, &t.Status, &t.NextTriggerAt,
			&t.TotalExecutions, &t.CreatedAt, &t.UpdatedAt,
			&t.PreHooks, &t.PostHooks); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func nullableJSON(data json.RawMessage) []byte {
	if data == nil {
		return []byte("[]")
	}
	return data
}

func (r *TaskRepo) Update(ctx context.Context, t *domain.Task) error {
	preHooks := nullableJSON(t.PreHooks)
	postHooks := nullableJSON(t.PostHooks)
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET name = $1, cron_expr = $2, interval_value = $3, interval_unit = $4,
timeout_secs = $5, retry_count = $6, retry_backoff = $7, concurrency_policy = $8,
		 max_concurrency = $9, execution_retention_days = $10,
		 next_trigger_at = $11, updated_at = $12, schedule_type = $13, once_at = $14,
		 pre_hooks = $15, post_hooks = $16
		 WHERE id = $17 AND tenant_id = $18`,
		t.Name, t.CronExpr, t.IntervalValue, t.IntervalUnit,
		t.TimeoutSecs, t.RetryCount, t.RetryBackoff, t.ConcurrencyPolicy,
		t.MaxConcurrency, t.ExecutionRetentionDays,
		t.NextTriggerAt, t.UpdatedAt, t.ScheduleType, t.OnceAt,
		preHooks, postHooks,
		t.ID, t.TenantID)
	return err
}

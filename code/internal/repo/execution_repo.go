package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type ExecutionRepo struct {
	pool *pgxpool.Pool
}

func NewExecutionRepo(pool *pgxpool.Pool) *ExecutionRepo {
	return &ExecutionRepo{pool: pool}
}

func (r *ExecutionRepo) Create(ctx context.Context, e *domain.Execution) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO executions (task_id, tenant_id, trigger_time, attempt, status, status_code,
		 duration_ms, request_headers, request_body, response_body, error_msg, is_makeup, is_manual, triggered_by, hooks_result, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		e.TaskID, e.TenantID, e.TriggerTime, e.Attempt, e.Status, e.StatusCode,
		e.DurationMs, e.RequestHeaders, e.RequestBody, e.ResponseBody, e.ErrorMsg, e.IsMakeup, e.IsManual, e.TriggeredBy, e.HooksResult, e.CreatedAt)
	return err
}

func (r *ExecutionRepo) ListByTask(ctx context.Context, taskID string, limit, offset int) ([]*domain.Execution, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, task_id, tenant_id, trigger_time, attempt, status, status_code,
		 duration_ms, request_headers, request_body, response_body, error_msg, is_makeup, is_manual, triggered_by, hooks_result, created_at
		 FROM executions WHERE task_id = $1 ORDER BY trigger_time DESC LIMIT $2 OFFSET $3`,
		taskID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []*domain.Execution
	for rows.Next() {
		e := &domain.Execution{}
		if err := rows.Scan(&e.ID, &e.TaskID, &e.TenantID, &e.TriggerTime, &e.Attempt, &e.Status,
			&e.StatusCode, &e.DurationMs, &e.RequestHeaders, &e.RequestBody, &e.ResponseBody, &e.ErrorMsg, &e.IsMakeup, &e.IsManual,
			&e.TriggeredBy, &e.HooksResult, &e.CreatedAt); err != nil {
			return nil, err
		}
		execs = append(execs, e)
	}
	return execs, nil
}

func (r *ExecutionRepo) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Execution, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, task_id, tenant_id, trigger_time, attempt, status, status_code,
		 duration_ms, request_headers, request_body, response_body, error_msg, is_makeup, is_manual, triggered_by, hooks_result, created_at
		 FROM executions WHERE tenant_id = $1 ORDER BY trigger_time DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []*domain.Execution
	for rows.Next() {
		e := &domain.Execution{}
		if err := rows.Scan(&e.ID, &e.TaskID, &e.TenantID, &e.TriggerTime, &e.Attempt, &e.Status,
			&e.StatusCode, &e.DurationMs, &e.RequestHeaders, &e.RequestBody, &e.ResponseBody, &e.ErrorMsg, &e.IsMakeup, &e.IsManual,
			&e.TriggeredBy, &e.HooksResult, &e.CreatedAt); err != nil {
			return nil, err
		}
		execs = append(execs, e)
	}
	return execs, nil
}

func (r *ExecutionRepo) DeleteExpired(ctx context.Context, taskID string, before time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM executions WHERE task_id = $1 AND created_at < $2`, taskID, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

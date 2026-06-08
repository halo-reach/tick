package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type TargetRepo struct {
	pool *pgxpool.Pool
}

func NewTargetRepo(pool *pgxpool.Pool) *TargetRepo {
	return &TargetRepo{pool: pool}
}

func (r *TargetRepo) Create(ctx context.Context, t *domain.Target) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO targets (id, tenant_id, name, type, config, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		t.ID, t.TenantID, t.Name, t.Type, t.Config, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *TargetRepo) GetByID(ctx context.Context, id, tenantID string) (*domain.Target, error) {
	t := &domain.Target{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, type, config, created_at, updated_at
		 FROM targets WHERE id = $1 AND tenant_id = $2`, id, tenantID).
		Scan(&t.ID, &t.TenantID, &t.Name, &t.Type, &t.Config, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TargetRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Target, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, type, config, created_at, updated_at
		 FROM targets WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []*domain.Target
	for rows.Next() {
		t := &domain.Target{}
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Type, &t.Config, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func (r *TargetRepo) Update(ctx context.Context, t *domain.Target) error {
	t.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE targets SET name = $1, type = $2, config = $3, updated_at = $4
		 WHERE id = $5 AND tenant_id = $6`,
		t.Name, t.Type, t.Config, t.UpdatedAt, t.ID, t.TenantID)
	return err
}

func (r *TargetRepo) Delete(ctx context.Context, id, tenantID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM targets WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *TargetRepo) HasCredentialReference(ctx context.Context, credentialID, tenantID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM targets WHERE tenant_id = $1 AND config::jsonb->'credential_ids' ? $2`,
		tenantID, credentialID).Scan(&count)
	return count > 0, err
}

func (r *TargetRepo) HasActiveTasks(ctx context.Context, id, tenantID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tasks WHERE target_id = $1 AND tenant_id = $2 AND status = 'active'`,
		id, tenantID).Scan(&count)
	return count > 0, err
}

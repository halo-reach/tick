package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type VariableRepo struct {
	pool *pgxpool.Pool
}

func NewVariableRepo(pool *pgxpool.Pool) *VariableRepo {
	return &VariableRepo{pool: pool}
}

func (r *VariableRepo) Create(ctx context.Context, v *domain.Variable) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO variables (id, tenant_id, key, value, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		v.ID, v.TenantID, v.Key, v.Value, v.CreatedAt, v.UpdatedAt)
	return err
}

func (r *VariableRepo) GetByID(ctx context.Context, id, tenantID string) (*domain.Variable, error) {
	v := &domain.Variable{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, key, value, created_at, updated_at
		 FROM variables WHERE id = $1 AND tenant_id = $2`, id, tenantID).
		Scan(&v.ID, &v.TenantID, &v.Key, &v.Value, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *VariableRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Variable, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, key, value, created_at, updated_at
		 FROM variables WHERE tenant_id = $1 ORDER BY key ASC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vars []*domain.Variable
	for rows.Next() {
		v := &domain.Variable{}
		if err := rows.Scan(&v.ID, &v.TenantID, &v.Key, &v.Value, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		vars = append(vars, v)
	}
	return vars, nil
}

func (r *VariableRepo) Update(ctx context.Context, v *domain.Variable) error {
	v.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE variables SET key = $1, value = $2, updated_at = $3
		 WHERE id = $4 AND tenant_id = $5`,
		v.Key, v.Value, v.UpdatedAt, v.ID, v.TenantID)
	return err
}

func (r *VariableRepo) Delete(ctx context.Context, id, tenantID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM variables WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *VariableRepo) ExistsByKey(ctx context.Context, tenantID, key string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM variables WHERE tenant_id = $1 AND key = $2`,
		tenantID, key).Scan(&count)
	return count > 0, err
}

package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type TenantRepo struct {
	pool *pgxpool.Pool
}

func NewTenantRepo(pool *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{pool: pool}
}

func (r *TenantRepo) Create(ctx context.Context, t *domain.Tenant) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenants (id, name, username, password_hash, must_change_password, status, quota_max_tasks, quota_max_rps, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		t.ID, t.Name, t.Username, t.PasswordHash, t.MustChangePassword, t.Status, t.QuotaMaxTasks, t.QuotaMaxRPS, t.CreatedAt)
	return err
}

func (r *TenantRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, username, password_hash, must_change_password, status, quota_max_tasks, quota_max_rps, created_at
		 FROM tenants WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Username, &t.PasswordHash, &t.MustChangePassword, &t.Status, &t.QuotaMaxTasks, &t.QuotaMaxRPS, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TenantRepo) GetByUsername(ctx context.Context, username string) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, username, password_hash, must_change_password, status, quota_max_tasks, quota_max_rps, created_at
		 FROM tenants WHERE username = $1`, username).
		Scan(&t.ID, &t.Name, &t.Username, &t.PasswordHash, &t.MustChangePassword, &t.Status, &t.QuotaMaxTasks, &t.QuotaMaxRPS, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TenantRepo) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tenants SET password_hash = $1, must_change_password = false WHERE id = $2`,
		passwordHash, id)
	return err
}

func (r *TenantRepo) UpdateStatus(ctx context.Context, id string, status domain.TenantStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE tenants SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *TenantRepo) UpdateName(ctx context.Context, id, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE tenants SET name = $1 WHERE id = $2`, name, id)
	return err
}

func (r *TenantRepo) CountTasksByTenant(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tasks WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID).Scan(&count)
	return count, err
}

type ApiKeyRepo struct {
	pool *pgxpool.Pool
}

func NewApiKeyRepo(pool *pgxpool.Pool) *ApiKeyRepo {
	return &ApiKeyRepo{pool: pool}
}

func (r *ApiKeyRepo) Create(ctx context.Context, k *domain.ApiKey) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO api_keys (id, tenant_id, name, key_hash, key_prefix, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		k.ID, k.TenantID, k.Name, k.KeyHash, k.KeyPrefix, k.Status, k.CreatedAt)
	return err
}

func (r *ApiKeyRepo) FindByHash(ctx context.Context, hash string) (*domain.ApiKey, error) {
	k := &domain.ApiKey{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key_hash, key_prefix, status, created_at, revoked_at
		 FROM api_keys WHERE key_hash = $1 AND status = 'active'`, hash).
		Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Status, &k.CreatedAt, &k.RevokedAt)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (r *ApiKeyRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.ApiKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, key_prefix, status, created_at, revoked_at
		 FROM api_keys WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.ApiKey
	for rows.Next() {
		k := &domain.ApiKey{}
		if err := rows.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyPrefix, &k.Status, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *ApiKeyRepo) CountActiveByTenant(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM api_keys WHERE tenant_id = $1 AND status = 'active'`, tenantID).Scan(&count)
	return count, err
}

func (r *ApiKeyRepo) Revoke(ctx context.Context, id, tenantID string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET status = 'revoked', revoked_at = $1 WHERE id = $2 AND tenant_id = $3`,
		now, id, tenantID)
	return err
}

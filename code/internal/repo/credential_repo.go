package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type CredentialRepo struct {
	pool *pgxpool.Pool
}

func NewCredentialRepo(pool *pgxpool.Pool) *CredentialRepo {
	return &CredentialRepo{pool: pool}
}

func (r *CredentialRepo) Create(ctx context.Context, c *domain.Credential) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO credentials (id, tenant_id, name, code, type, config, config_preview, timeout_secs, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		c.ID, c.TenantID, c.Name, c.Code, c.Type, c.Config, c.ConfigPreview, c.TimeoutSecs, c.Status, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *CredentialRepo) GetByID(ctx context.Context, id, tenantID string) (*domain.Credential, error) {
	c := &domain.Credential{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, code, type, config, config_preview, timeout_secs, status, created_at, updated_at
		 FROM credentials WHERE id = $1 AND tenant_id = $2`, id, tenantID).
		Scan(&c.ID, &c.TenantID, &c.Name, &c.Code, &c.Type, &c.Config, &c.ConfigPreview, &c.TimeoutSecs, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *CredentialRepo) ListByTenant(ctx context.Context, tenantID string, status string, limit, offset int) ([]*domain.Credential, int, error) {
	countQuery := `SELECT COUNT(*) FROM credentials WHERE tenant_id = $1`
	listQuery := `SELECT id, tenant_id, name, code, type, config_preview, timeout_secs, status, created_at, updated_at
		 FROM credentials WHERE tenant_id = $1`

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

	var creds []*domain.Credential
	for rows.Next() {
		c := &domain.Credential{}
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Code, &c.Type, &c.ConfigPreview, &c.TimeoutSecs, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		creds = append(creds, c)
	}
	return creds, total, nil
}

func (r *CredentialRepo) Update(ctx context.Context, c *domain.Credential) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE credentials SET name = $1, config = $2, config_preview = $3, timeout_secs = $4, updated_at = $5
		 WHERE id = $6 AND tenant_id = $7`,
		c.Name, c.Config, c.ConfigPreview, c.TimeoutSecs, c.UpdatedAt, c.ID, c.TenantID)
	return err
}

func (r *CredentialRepo) UpdateStatus(ctx context.Context, id, tenantID string, status domain.CredentialStatus) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE credentials SET status = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4`,
		status, now, id, tenantID)
	return err
}

func (r *CredentialRepo) Delete(ctx context.Context, id, tenantID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM credentials WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *CredentialRepo) GetByCode(ctx context.Context, code, tenantID string) (*domain.Credential, error) {
	c := &domain.Credential{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, code, type, config, config_preview, timeout_secs, status, created_at, updated_at
		 FROM credentials WHERE code = $1 AND tenant_id = $2 AND status = 'active'`, code, tenantID).
		Scan(&c.ID, &c.TenantID, &c.Name, &c.Code, &c.Type, &c.Config, &c.ConfigPreview, &c.TimeoutSecs, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *CredentialRepo) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM credentials WHERE tenant_id = $1 AND status != 'deleted'`, tenantID).Scan(&count)
	return count, err
}

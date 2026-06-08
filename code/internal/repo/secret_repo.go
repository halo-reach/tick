package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type SecretRepo struct {
	pool *pgxpool.Pool
}

func NewSecretRepo(pool *pgxpool.Pool) *SecretRepo {
	return &SecretRepo{pool: pool}
}

func (r *SecretRepo) Create(ctx context.Context, s *domain.SigningSecret) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO signing_secrets (id, tenant_id, secret, status, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		s.ID, s.TenantID, s.Secret, s.Status, s.CreatedAt)
	return err
}

func (r *SecretRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.SigningSecret, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, status, created_at, revoked_at
		 FROM signing_secrets WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*domain.SigningSecret
	for rows.Next() {
		s := &domain.SigningSecret{}
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Status, &s.CreatedAt, &s.RevokedAt); err != nil {
			return nil, err
		}
		secrets = append(secrets, s)
	}
	return secrets, nil
}

func (r *SecretRepo) GetActivByTenant(ctx context.Context, tenantID string) (*domain.SigningSecret, error) {
	s := &domain.SigningSecret{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, secret, status, created_at
		 FROM signing_secrets WHERE tenant_id = $1 AND status = 'active'
		 ORDER BY created_at DESC LIMIT 1`, tenantID).
		Scan(&s.ID, &s.TenantID, &s.Secret, &s.Status, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *SecretRepo) Revoke(ctx context.Context, id, tenantID string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE signing_secrets SET status = 'revoked', revoked_at = $1 WHERE id = $2 AND tenant_id = $3`,
		now, id, tenantID)
	return err
}

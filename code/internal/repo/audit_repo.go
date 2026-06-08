package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type AuditRepo struct {
	pool *pgxpool.Pool
}

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

func (r *AuditRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_logs (tenant_id, actor, action, resource_type, resource_id, payload, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		log.TenantID, log.Actor, log.Action, log.ResourceType, log.ResourceID, log.Payload, log.CreatedAt)
	return err
}

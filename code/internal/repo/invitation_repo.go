package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type InvitationRepo struct {
	pool *pgxpool.Pool
}

func NewInvitationRepo(pool *pgxpool.Pool) *InvitationRepo {
	return &InvitationRepo{pool: pool}
}

func (r *InvitationRepo) Create(ctx context.Context, inv *domain.Invitation) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO invitations (id, tenant_id, code, created_by, role, max_uses, used_count, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		inv.ID, inv.TenantID, inv.Code, inv.CreatedBy, inv.Role, inv.MaxUses, inv.UsedCount, inv.ExpiresAt, inv.CreatedAt)
	return err
}

func (r *InvitationRepo) GetByCode(ctx context.Context, code string) (*domain.Invitation, error) {
	inv := &domain.Invitation{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, code, created_by, role, max_uses, used_count, expires_at, created_at
		 FROM invitations WHERE code = $1`, code).
		Scan(&inv.ID, &inv.TenantID, &inv.Code, &inv.CreatedBy, &inv.Role, &inv.MaxUses, &inv.UsedCount, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *InvitationRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Invitation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, code, created_by, role, max_uses, used_count, expires_at, created_at
		 FROM invitations WHERE tenant_id = $1 AND expires_at > NOW() ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []*domain.Invitation
	for rows.Next() {
		inv := &domain.Invitation{}
		if err := rows.Scan(&inv.ID, &inv.TenantID, &inv.Code, &inv.CreatedBy, &inv.Role, &inv.MaxUses, &inv.UsedCount, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}
	return invitations, nil
}

func (r *InvitationRepo) Delete(ctx context.Context, id, tenantID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM invitations WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *InvitationRepo) IncrementUsedCount(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE invitations SET used_count = used_count + 1 WHERE id = $1`, id)
	return err
}

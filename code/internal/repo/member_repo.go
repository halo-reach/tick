package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type UserTenantInfo struct {
	TenantID   string            `json:"id"`
	TenantName string            `json:"name"`
	Role       domain.MemberRole `json:"role"`
}

type MemberRepo struct {
	pool *pgxpool.Pool
}

func NewMemberRepo(pool *pgxpool.Pool) *MemberRepo {
	return &MemberRepo{pool: pool}
}

func (r *MemberRepo) Create(ctx context.Context, m *domain.TenantMember) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenant_members (id, tenant_id, user_id, role, joined_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		m.ID, m.TenantID, m.UserID, m.Role, m.JoinedAt)
	return err
}

func (r *MemberRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.TenantMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, user_id, role, joined_at
		 FROM tenant_members WHERE tenant_id = $1 ORDER BY joined_at`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*domain.TenantMember
	for rows.Next() {
		m := &domain.TenantMember{}
		if err := rows.Scan(&m.ID, &m.TenantID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *MemberRepo) ListByUser(ctx context.Context, userID string) ([]*UserTenantInfo, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tm.tenant_id, t.name, tm.role
		 FROM tenant_members tm
		 JOIN tenants t ON t.id = tm.tenant_id
		 WHERE tm.user_id = $1 ORDER BY tm.joined_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []*UserTenantInfo
	for rows.Next() {
		info := &UserTenantInfo{}
		if err := rows.Scan(&info.TenantID, &info.TenantName, &info.Role); err != nil {
			return nil, err
		}
		tenants = append(tenants, info)
	}
	return tenants, nil
}

func (r *MemberRepo) GetByTenantAndUser(ctx context.Context, tenantID, userID string) (*domain.TenantMember, error) {
	m := &domain.TenantMember{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, role, joined_at
		 FROM tenant_members WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID).
		Scan(&m.ID, &m.TenantID, &m.UserID, &m.Role, &m.JoinedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *MemberRepo) Delete(ctx context.Context, tenantID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM tenant_members WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID)
	return err
}

func (r *MemberRepo) UpdateRole(ctx context.Context, tenantID, userID string, role domain.MemberRole) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tenant_members SET role = $1 WHERE tenant_id = $2 AND user_id = $3`,
		role, tenantID, userID)
	return err
}

func (r *MemberRepo) CountOwners(ctx context.Context, tenantID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tenant_members WHERE tenant_id = $1 AND role = 'owner'`, tenantID).Scan(&count)
	return count, err
}

type MemberWithUser struct {
	UserID      string            `json:"user_id"`
	Username    string            `json:"username"`
	DisplayName string            `json:"display_name"`
	Role        domain.MemberRole `json:"role"`
	JoinedAt    time.Time         `json:"joined_at"`
}

func (r *MemberRepo) ListMembersWithUser(ctx context.Context, tenantID string) ([]*MemberWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tm.user_id, u.username, u.display_name, tm.role, tm.joined_at
		 FROM tenant_members tm JOIN users u ON u.id = tm.user_id
		 WHERE tm.tenant_id = $1 ORDER BY tm.joined_at`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*MemberWithUser
	for rows.Next() {
		m := &MemberWithUser{}
		if err := rows.Scan(&m.UserID, &m.Username, &m.DisplayName, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

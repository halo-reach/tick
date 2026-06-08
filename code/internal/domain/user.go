package domain

import "time"

type UserStatus string

const (
	UserActive    UserStatus = "active"
	UserSuspended UserStatus = "suspended"
	UserLocked    UserStatus = "locked"
)

type User struct {
	ID             string     `json:"id"`
	Username       string     `json:"username"`
	PasswordHash   string     `json:"-"`
	DisplayName    string     `json:"display_name,omitempty"`
	Email          *string    `json:"email,omitempty"`
	Status         UserStatus `json:"status"`
	FailedAttempts int        `json:"-"`
	LockedUntil    *time.Time `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
}

type MemberRole string

const (
	RoleOwner  MemberRole = "owner"
	RoleMember MemberRole = "member"
)

type TenantMember struct {
	ID       string     `json:"id"`
	TenantID string     `json:"tenant_id"`
	UserID   string     `json:"user_id"`
	Role     MemberRole `json:"role"`
	JoinedAt time.Time  `json:"joined_at"`
}

type Invitation struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	Code      string     `json:"code"`
	CreatedBy string     `json:"created_by"`
	Role      MemberRole `json:"role"`
	MaxUses   int        `json:"max_uses"`
	UsedCount int        `json:"used_count"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

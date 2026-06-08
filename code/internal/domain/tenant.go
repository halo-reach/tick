package domain

import "time"

type TenantStatus string

const (
	TenantActive    TenantStatus = "active"
	TenantSuspended TenantStatus = "suspended"
)

type Tenant struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	Username           *string      `json:"username,omitempty"`
	PasswordHash       *string      `json:"-"`
	MustChangePassword bool         `json:"must_change_password,omitempty"`
	Status             TenantStatus `json:"status"`
	QuotaMaxTasks      int          `json:"quota_max_tasks"`
	QuotaMaxRPS        int          `json:"quota_max_rps"`
	CreatedAt          time.Time    `json:"created_at"`
}

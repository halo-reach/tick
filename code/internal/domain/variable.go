package domain

import "time"

type Variable struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

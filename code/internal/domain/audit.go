package domain

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID           int64           `json:"id"`
	TenantID     string          `json:"tenant_id"`
	Actor        string          `json:"actor"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type KeyStatus string

const (
	KeyActive  KeyStatus = "active"
	KeyRevoked KeyStatus = "revoked"
)

type ApiKey struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	Name      string     `json:"name"`
	KeyHash   string     `json:"-"`
	KeyPrefix string     `json:"key_prefix"`
	Status    KeyStatus  `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type SecretStatus string

const (
	SecretActive  SecretStatus = "active"
	SecretRevoked SecretStatus = "revoked"
)

type SigningSecret struct {
	ID        string       `json:"id"`
	TenantID  string       `json:"tenant_id"`
	Secret    string       `json:"-"`
	Status    SecretStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	RevokedAt *time.Time   `json:"revoked_at,omitempty"`
}

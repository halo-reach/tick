package domain

import (
	"encoding/json"
	"time"
)

type CredentialType string

const (
	CredTypeBearer       CredentialType = "bearer"
	CredTypeBasic        CredentialType = "basic"
	CredTypeOAuth2CC     CredentialType = "oauth2_cc"
	CredTypeDynamic      CredentialType = "dynamic"
	CredTypeHMAC         CredentialType = "hmac"
	CredTypeCustomHeader CredentialType = "custom_header"
)

type CredentialStatus string

const (
	CredStatusActive   CredentialStatus = "active"
	CredStatusDisabled CredentialStatus = "disabled"
	CredStatusDeleted  CredentialStatus = "deleted"
)

type Credential struct {
	ID            string           `json:"id"`
	TenantID      string           `json:"tenant_id"`
	Name          string           `json:"name"`
	Code          string           `json:"code"`
	Type          CredentialType   `json:"type"`
	Config        []byte           `json:"-"`
	ConfigPreview json.RawMessage  `json:"config_preview"`
	TimeoutSecs   int              `json:"timeout_secs"`
	Status        CredentialStatus `json:"status"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}


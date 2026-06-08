package domain

import (
	"encoding/json"
	"time"
)

type TargetType string

const (
	TargetHTTP   TargetType = "http"
	TargetFeishu TargetType = "feishu"
	TargetGRPC   TargetType = "grpc"
	TargetMQ     TargetType = "mq"
)

type Target struct {
	ID        string          `json:"id"`
	TenantID  string          `json:"tenant_id"`
	Name      string          `json:"name"`
	Type      TargetType      `json:"type"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type HTTPTargetConfig struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          json.RawMessage   `json:"body,omitempty"`
	ContentType   string            `json:"content_type,omitempty"`
	CredentialIDs []string          `json:"credential_ids,omitempty"`
}

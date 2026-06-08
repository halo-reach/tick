package domain

import "encoding/json"

type HookType string

const (
	HookTypeCredential HookType = "credential"
	HookTypeHTTP       HookType = "http"
	HookTypeFeishu     HookType = "feishu"
)

type HookWhen string

const (
	HookWhenSuccess HookWhen = "success"
	HookWhenFailure HookWhen = "failure"
	HookWhenAlways  HookWhen = "always"
)

type InjectConfig struct {
	Location string `json:"location"`
	Key      string `json:"key"`
	Prefix   string `json:"prefix,omitempty"`
}

type HookRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

type ExtractConfig struct {
	Path string `json:"path"`
	As   string `json:"as"`
}

type PreHook struct {
	Type         HookType       `json:"type"`
	TimeoutSecs  int            `json:"timeout_secs,omitempty"`
	CredentialID string         `json:"credential_id,omitempty"`
	Inject       *InjectConfig  `json:"inject,omitempty"`
	Request      *HookRequest   `json:"request,omitempty"`
	Extract      *ExtractConfig `json:"extract,omitempty"`
}

type PostHook struct {
	Type            HookType        `json:"type"`
	When            HookWhen        `json:"when"`
	TimeoutSecs     int             `json:"timeout_secs,omitempty"`
	Request         *HookRequest    `json:"request,omitempty"`
	ResponseExtract []ExtractConfig `json:"response_extract,omitempty"`
}

type HookResultEntry struct {
	Index      int               `json:"index"`
	Type       HookType          `json:"type"`
	When       HookWhen          `json:"when,omitempty"`
	Status     string            `json:"status"`
	DurationMs int               `json:"duration_ms"`
	Extracted  map[string]string `json:"extracted,omitempty"`
	ErrorMsg   string            `json:"error_msg,omitempty"`
}

type CredentialInjected struct {
	CredentialID string `json:"credential_id"`
	InjectKey    string `json:"inject_key"`
	Status       string `json:"status"`
}

type CredentialInjection struct {
	Location string `json:"location"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type HooksResult struct {
	PreHooks            []HookResultEntry    `json:"pre_hooks,omitempty"`
	PostHooks           []HookResultEntry    `json:"post_hooks,omitempty"`
	CredentialsInjected []CredentialInjected `json:"credentials_injected,omitempty"`
}

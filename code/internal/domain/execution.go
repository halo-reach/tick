package domain

import (
	"encoding/json"
	"time"
)

type ExecStatus string

const (
	ExecSuccess ExecStatus = "success"
	ExecFailed  ExecStatus = "failed"
	ExecTimeout ExecStatus = "timeout"
	ExecSkipped ExecStatus = "skipped"
)

type Execution struct {
	ID           int64      `json:"id"`
	TaskID       string     `json:"task_id"`
	TenantID     string     `json:"tenant_id"`
	TriggerTime  time.Time  `json:"trigger_time"`
	Attempt      int        `json:"attempt"`
	Status       ExecStatus `json:"status"`
	StatusCode   int        `json:"status_code,omitempty"`
	DurationMs   int        `json:"duration_ms"`
	RequestHeaders string     `json:"request_headers,omitempty"`
	RequestBody  string     `json:"request_body,omitempty"`
	ResponseBody string     `json:"response_body,omitempty"`
	ErrorMsg     string     `json:"error_msg,omitempty"`
	IsMakeup     bool       `json:"is_makeup"`
	IsManual     bool            `json:"is_manual"`
	TriggeredBy  string          `json:"triggered_by,omitempty"`
	HooksResult  json.RawMessage `json:"hooks_result,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

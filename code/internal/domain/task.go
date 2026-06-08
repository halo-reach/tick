package domain

import (
	"encoding/json"
	"time"
)

type ScheduleType string

const (
	ScheduleCron     ScheduleType = "cron"
	ScheduleInterval ScheduleType = "interval"
	ScheduleOnce     ScheduleType = "once"
)

type IntervalUnit string

const (
	UnitSeconds IntervalUnit = "s"
	UnitMinutes IntervalUnit = "m"
	UnitHours   IntervalUnit = "h"
	UnitDays    IntervalUnit = "d"
)

type MissedPolicy string

const (
	MissedSkip     MissedPolicy = "skip"
	MissedFireOnce MissedPolicy = "fire_once"
)

type ConcurrencyPolicy string

const (
	ConcurrencyAllow ConcurrencyPolicy = "allow"
	ConcurrencySkip  ConcurrencyPolicy = "skip"
	ConcurrencyQueue ConcurrencyPolicy = "queue"
)

type RetryBackoff string

const (
	BackoffExponential RetryBackoff = "exponential"
	BackoffFixed       RetryBackoff = "fixed"
	BackoffNone        RetryBackoff = "none"
)

type TaskStatus string

const (
	TaskActive  TaskStatus = "active"
	TaskPaused  TaskStatus = "paused"
	TaskDeleted TaskStatus = "deleted"
)

type Task struct {
	ID              string       `json:"id"`
	TenantID        string       `json:"tenant_id"`
	Name            string       `json:"name"`
	ScheduleType    ScheduleType `json:"schedule_type"`
	CronExpr        string       `json:"cron_expr,omitempty"`
	IntervalValue   int          `json:"interval_value,omitempty"`
	IntervalUnit    IntervalUnit `json:"interval_unit,omitempty"`
	OnceAt          *time.Time   `json:"once_at,omitempty"`
	TargetID        string       `json:"target_id"`
	TimeoutSecs     int          `json:"timeout_secs"`
	RetryCount      int          `json:"retry_count"`
	RetryBackoff    RetryBackoff `json:"retry_backoff"`
	ConcurrencyPolicy   ConcurrencyPolicy `json:"concurrency_policy"`
	MaxConcurrency      int               `json:"max_concurrency"`
	ExecutionRetentionDays int            `json:"execution_retention_days"`
	MissedPolicy    MissedPolicy    `json:"missed_policy"`
	PreHooks        json.RawMessage `json:"pre_hooks,omitempty"`
	PostHooks       json.RawMessage `json:"post_hooks,omitempty"`
	Status          TaskStatus   `json:"status"`
	NextTriggerAt   *time.Time   `json:"next_trigger_at,omitempty"`
	TotalExecutions int64        `json:"total_executions"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
	DeletedAt       *time.Time   `json:"deleted_at,omitempty"`
}

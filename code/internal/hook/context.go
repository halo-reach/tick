package hook

import (
	"fmt"
	"time"
)

type VariableContext struct {
	vars map[string]string
}

func NewVariableContext() *VariableContext {
	return &VariableContext{vars: make(map[string]string)}
}

func (vc *VariableContext) Set(key, value string) {
	vc.vars[key] = value
}

func (vc *VariableContext) Get(key string) (string, bool) {
	v, ok := vc.vars[key]
	return v, ok
}

func (vc *VariableContext) Merge(other *VariableContext) {
	if other == nil {
		return
	}
	for k, v := range other.vars {
		vc.vars[k] = v
	}
}

func (vc *VariableContext) All() map[string]string {
	out := make(map[string]string, len(vc.vars))
	for k, v := range vc.vars {
		out[k] = v
	}
	return out
}

func (vc *VariableContext) SetBuiltins(taskID, taskName, tenantID, triggerTime, executionID string) {
	now := time.Now()
	vc.vars["task_id"] = taskID
	vc.vars["task_name"] = taskName
	vc.vars["tenant_id"] = tenantID
	vc.vars["trigger_time"] = triggerTime
	vc.vars["execution_id"] = executionID
	vc.vars["current_date"] = now.Format("2006-01-02")
	vc.vars["current_time"] = now.Format("15:04:05")
	vc.vars["current_datetime"] = now.Format("2006-01-02 15:04:05")
	vc.vars["current_timestamp"] = fmt.Sprintf("%d", now.Unix())
}

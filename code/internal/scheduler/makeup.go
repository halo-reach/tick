package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/tickplatform/tick/internal/domain"
)

type MakeupExecutor struct {
	trigger TriggerFunc
}

func NewMakeupExecutor(trigger TriggerFunc) *MakeupExecutor {
	return &MakeupExecutor{trigger: trigger}
}

func (m *MakeupExecutor) ProcessMissed(ctx context.Context, tasks []*domain.Task, now time.Time) {
	for _, task := range tasks {
		if task.NextTriggerAt == nil || task.NextTriggerAt.After(now) {
			continue
		}
		if task.Status != domain.TaskActive {
			continue
		}
		switch task.MissedPolicy {
		case domain.MissedFireOnce:
			slog.Info("makeup fire_once", "task_id", task.ID)
			m.trigger(task, *task.NextTriggerAt)
		default:
			slog.Info("skip missed task", "task_id", task.ID)
		}
	}
}

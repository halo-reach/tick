package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/tickplatform/tick/internal/domain"
)

type TriggerFunc func(task *domain.Task, triggerTime time.Time)
type OnceCompleteFunc func(taskID, tenantID string)
type PersistNextTriggerFunc func(taskID string, next *time.Time)

type Scheduler struct {
	mu             sync.RWMutex
	tasks          map[string]*domain.Task
	trigger        TriggerFunc
	onceComplete   OnceCompleteFunc
	persistNext    PersistNextTriggerFunc
	parser         cron.Parser
	stopCh         chan struct{}
	stopped        bool
}

func New(trigger TriggerFunc) *Scheduler {
	return &Scheduler{
		tasks:   make(map[string]*domain.Task),
		trigger: trigger,
		parser:  defaultParser(),
		stopCh:  make(chan struct{}),
	}
}

func (s *Scheduler) SetOnceComplete(fn OnceCompleteFunc) {
	s.onceComplete = fn
}

func (s *Scheduler) SetPersistNextTrigger(fn PersistNextTriggerFunc) {
	s.persistNext = fn
}

func defaultParser() cron.Parser {
	return cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
}

func (s *Scheduler) AddTask(task *domain.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
}

func (s *Scheduler) RemoveTask(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tasks, id)
}

func (s *Scheduler) LoadTasks(tasks []*domain.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range tasks {
		s.tasks[t.ID] = t
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	slog.Info("scheduler started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopped {
		s.stopped = true
		close(s.stopCh)
	}
}

func (s *Scheduler) tick(now time.Time) {
	s.mu.RLock()
	var ready []*domain.Task
	for _, task := range s.tasks {
		if task.NextTriggerAt != nil && !task.NextTriggerAt.After(now) && task.Status != domain.TaskPaused {
			ready = append(ready, task)
		}
	}
	s.mu.RUnlock()

	for _, task := range ready {
		triggerTime := *task.NextTriggerAt
		if now.Sub(triggerTime) > 30*time.Second {
			s.advanceTo(task, now)
			continue
		}
		scatter := Scatter(task.ID, triggerTime.Unix())
		time.AfterFunc(scatter, func() {
			s.trigger(task, triggerTime)
		})
		s.advance(task, triggerTime)
	}
}

func (s *Scheduler) advanceTo(task *domain.Task, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch task.ScheduleType {
	case domain.ScheduleCron:
		sched, err := s.parser.Parse(task.CronExpr)
		if err != nil {
			return
		}
		next := sched.Next(now)
		task.NextTriggerAt = &next
		s.persistNextTrigger(task.ID, &next)
		slog.Info("advanceTo: skipped past trigger", "task_id", task.ID, "next", next.Format(time.RFC3339))
	case domain.ScheduleInterval:
		next := now.Add(intervalDuration(task.IntervalValue, task.IntervalUnit))
		task.NextTriggerAt = &next
		s.persistNextTrigger(task.ID, &next)
	case domain.ScheduleOnce:
		task.NextTriggerAt = nil
		delete(s.tasks, task.ID)
		s.persistNextTrigger(task.ID, nil)
	}
}

func (s *Scheduler) advance(task *domain.Task, triggerTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch task.ScheduleType {
	case domain.ScheduleCron:
		sched, err := s.parser.Parse(task.CronExpr)
		if err != nil {
			slog.Error("parse cron", "task_id", task.ID, "error", err)
			return
		}
		next := sched.Next(triggerTime.Add(time.Second))
		task.NextTriggerAt = &next
		s.persistNextTrigger(task.ID, &next)
	case domain.ScheduleInterval:
		next := triggerTime.Add(intervalDuration(task.IntervalValue, task.IntervalUnit))
		task.NextTriggerAt = &next
		s.persistNextTrigger(task.ID, &next)
	case domain.ScheduleOnce:
		task.NextTriggerAt = nil
		delete(s.tasks, task.ID)
		s.persistNextTrigger(task.ID, nil)
		if s.onceComplete != nil {
			go s.onceComplete(task.ID, task.TenantID)
		}
	}
}

type TaskLoader func(ctx context.Context) ([]*domain.Task, error)

func (s *Scheduler) persistNextTrigger(taskID string, next *time.Time) {
	if s.persistNext != nil {
		s.persistNext(taskID, next)
	}
}

func (s *Scheduler) StartSync(ctx context.Context, loader TaskLoader, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			tasks, err := loader(ctx)
			if err != nil {
				slog.Error("sync tasks failed", "error", err)
				continue
			}
			s.mu.Lock()
			s.tasks = make(map[string]*domain.Task, len(tasks))
			for _, t := range tasks {
				s.tasks[t.ID] = t
			}
			s.mu.Unlock()
		}
	}
}

func intervalDuration(value int, unit domain.IntervalUnit) time.Duration {
	switch unit {
	case domain.UnitSeconds:
		return time.Duration(value) * time.Second
	case domain.UnitMinutes:
		return time.Duration(value) * time.Minute
	case domain.UnitHours:
		return time.Duration(value) * time.Hour
	case domain.UnitDays:
		return time.Duration(value) * 24 * time.Hour
	default:
		return time.Duration(value) * time.Second
	}
}

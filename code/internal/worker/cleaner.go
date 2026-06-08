package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/tickplatform/tick/internal/repo"
)

type Cleaner struct {
	taskRepo *repo.TaskRepo
	execRepo *repo.ExecutionRepo
	stop     chan struct{}
}

func NewCleaner(taskRepo *repo.TaskRepo, execRepo *repo.ExecutionRepo) *Cleaner {
	return &Cleaner{
		taskRepo: taskRepo,
		execRepo: execRepo,
		stop:     make(chan struct{}),
	}
}

func (c *Cleaner) Start() {
	go c.run()
}

func (c *Cleaner) Stop() {
	close(c.stop)
}

func (c *Cleaner) run() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

func (c *Cleaner) cleanup() {
	ctx := context.Background()
	tasks, err := c.taskRepo.LoadActive(ctx)
	if err != nil {
		slog.Error("cleaner: load tasks", "error", err)
		return
	}
	for _, task := range tasks {
		if task.ExecutionRetentionDays <= 0 {
			continue
		}
		before := time.Now().AddDate(0, 0, -task.ExecutionRetentionDays)
		deleted, err := c.execRepo.DeleteExpired(ctx, task.ID, before)
		if err != nil {
			slog.Error("cleaner: delete expired", "task_id", task.ID, "error", err)
			continue
		}
		if deleted > 0 {
			slog.Info("cleaner: removed expired executions", "task_id", task.ID, "count", deleted)
		}
	}
}

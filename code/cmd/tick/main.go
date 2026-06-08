package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tickplatform/tick/internal/api"
	"github.com/tickplatform/tick/internal/auth"
	"github.com/tickplatform/tick/internal/cli"
	"github.com/tickplatform/tick/internal/config"
	"github.com/tickplatform/tick/internal/credential"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/hook"
	"github.com/tickplatform/tick/internal/repo"
	"github.com/tickplatform/tick/internal/scheduler"
	"github.com/tickplatform/tick/internal/worker"
)

// Build-time metadata injected via -ldflags.
var (
	Version       = "dev"
	Commit        = "unknown"
	BuildTime     = "unknown"
	ProdServerURL = "http://localhost:8080"
	SITServerURL  = "http://localhost:8080"
	BuiltForEnv   = "prod"
	SourcePath    = ""
)

func main() {
	if len(os.Args) > 1 && os.Args[1] != "serve" {
		cli.SetBuildMetadata(Version, Commit, BuildTime, ProdServerURL, SITServerURL, BuiltForEnv, SourcePath)
		root := cli.NewRootCmd()
		if err := root.Execute(); err != nil {
			os.Exit(1)
		}
		return
	}

	config.InitLogger()
	cfg := config.Load()
	auth.InitJWT(cfg.JWT.Secret)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := repo.NewPool(ctx, cfg.Database)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	tenantRepo := repo.NewTenantRepo(pool)
	keyRepo := repo.NewApiKeyRepo(pool)
	secretRepo := repo.NewSecretRepo(pool)
	targetRepo := repo.NewTargetRepo(pool)
	taskRepo := repo.NewTaskRepo(pool)
	execRepo := repo.NewExecutionRepo(pool)
	auditRepo := repo.NewAuditRepo(pool)
	credRepo := repo.NewCredentialRepo(pool)
	variableRepo := repo.NewVariableRepo(pool)
	userRepo := repo.NewUserRepo(pool)
	memberRepo := repo.NewMemberRepo(pool)
	invitationRepo := repo.NewInvitationRepo(pool)

	credKey := []byte(os.Getenv("TICK_CREDENTIAL_KEY"))
	if len(credKey) == 0 {
		credKey = []byte("default-dev-key-32bytes-long!!!!")
	}
	credStore := credential.NewStore(credRepo, credKey)

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	tokenCache := credential.NewTokenCache(rdb)
	credResolver := credential.NewResolver(credStore, tokenCache)

	guard := worker.NewConcurrencyGuard(rdb)

	enqueuer := worker.NewEnqueuer(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, guard, execRepo, taskRepo)
	defer enqueuer.Close()

	triggerFn := func(task *domain.Task, triggerTime time.Time) {
		enqueuer.Enqueue(task, triggerTime)
	}
	sched := scheduler.New(triggerFn)
	sched.SetOnceComplete(func(taskID, tenantID string) {
		_ = taskRepo.UpdateStatus(context.Background(), taskID, tenantID, domain.TaskPaused)
		_ = taskRepo.UpdateNextTrigger(context.Background(), taskID, nil)
	})
	sched.SetPersistNextTrigger(func(taskID string, next *time.Time) {
		_ = taskRepo.UpdateNextTrigger(context.Background(), taskID, next)
	})

	lock := scheduler.NewRedisLock(rdb, 60*time.Second)
	schedCtx, schedCancel := context.WithCancel(ctx)

	startScheduler := func() {
		tasks, err := taskRepo.LoadActive(schedCtx)
		if err != nil {
			slog.Error("load active tasks", "error", err)
			return
		}
		makeup := scheduler.NewMakeupExecutor(triggerFn)
		makeup.ProcessMissed(schedCtx, tasks, time.Now())
		sched.LoadTasks(tasks)
		go sched.Start(schedCtx)
		go sched.StartSync(schedCtx, func(ctx context.Context) ([]*domain.Task, error) {
			return taskRepo.LoadActive(ctx)
		}, 30*time.Second)
		go scheduler.RunRenewal(schedCtx, schedCancel, lock)
	}

	if acquired, _ := lock.Acquire(ctx); acquired {
		slog.Info("acquired scheduler lock")
		startScheduler()
	} else {
		slog.Info("scheduler lock held by another instance, running as worker only")
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if acquired, _ := lock.Acquire(ctx); acquired {
						slog.Info("acquired scheduler lock (failover)")
						startScheduler()
						return
					}
				}
			}
		}()
	}

	hookEngine := hook.NewEngine(credResolver)

	handler := worker.NewHandler(taskRepo, targetRepo, execRepo, secretRepo, variableRepo, rdb, guard, enqueuer, credResolver, hookEngine)
	srv := worker.NewServer(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err := srv.Start(handler); err != nil {
		slog.Error("start worker", "error", err)
		os.Exit(1)
	}
	defer srv.Stop()

	cleaner := worker.NewCleaner(taskRepo, execRepo)
	cleaner.Start()
	defer cleaner.Stop()

	router := api.NewRouter(tenantRepo, keyRepo, secretRepo, targetRepo, taskRepo, execRepo, auditRepo, variableRepo, sched, credRepo, credStore, credResolver, userRepo, memberRepo, invitationRepo)
	httpSrv := &http.Server{Addr: cfg.Server.Addr, Handler: router}

	go func() {
		slog.Info("server starting", "addr", cfg.Server.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	schedCancel()
	sched.Stop()
	_ = lock.Release(context.Background())
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	httpSrv.Shutdown(shutdownCtx)

	fmt.Println("bye")
}

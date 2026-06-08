package worker

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/tickplatform/tick/internal/domain"
)

type Server struct {
	srv    *asynq.Server
	queues map[string]int
}

func NewServer(redisAddr, redisPassword string, db int) *Server {
	queues := map[string]int{"default": 1}
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword, DB: db},
		asynq.Config{
			Concurrency: 50,
			Queues:      queues,
			RetryDelayFunc: func(n int, _ error, t *asynq.Task) time.Duration {
				var p TriggerPayload
				if err := json.Unmarshal(t.Payload(), &p); err == nil {
					switch p.RetryBackoff {
					case domain.BackoffFixed:
						return 10 * time.Second
					case domain.BackoffNone:
						return 0
					}
				}
				switch n {
				case 1:
					return 10 * time.Second
				case 2:
					return 30 * time.Second
				default:
					return 90 * time.Second
				}
			},
			Logger: &asynqLogger{},
		},
	)
	return &Server{srv: srv, queues: queues}
}

func (s *Server) RegisterQueue(name string) {
	s.queues[name] = 1
}

func (s *Server) Start(handler *Handler) error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeTrigger, handler.HandleTrigger)
	return s.srv.Start(mux)
}

func (s *Server) Stop() {
	s.srv.Stop()
}

type asynqLogger struct{}

func (l *asynqLogger) Debug(args ...any)                    { slog.Debug("asynq", "msg", args) }
func (l *asynqLogger) Info(args ...any)                     { slog.Info("asynq", "msg", args) }
func (l *asynqLogger) Warn(args ...any)                     { slog.Warn("asynq", "msg", args) }
func (l *asynqLogger) Error(args ...any)                    { slog.Error("asynq", "msg", args) }
func (l *asynqLogger) Fatal(args ...any)                    { slog.Error("asynq fatal", "msg", args) }

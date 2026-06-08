# Tick - Schedule Trigger Platform

Tick is a personal-level scheduled trigger platform designed for AI Agent scenarios. It provides reliable, second-level precision task scheduling with multi-tenant isolation.

**Core Principle**: Time wheel handles "when to trigger", Asynq handles "how to execute".

[中文](README.md) | English

---

## Features

- **Second-level Precision**: Custom time wheel with micro-scattering algorithm
- **Multi-tenant Isolation**: API Key authentication with complete data isolation
- **Multiple Trigger Types**: HTTP Webhook (more protocols in V2)
- **CLI-first**: Full-featured command-line interface for task management
- **Observable**: Execution history, retry with exponential backoff, timeout control
- **Credential Center**: AES-256 encrypted credential storage with automatic injection
- **Hook Engine**: Pre-Hook (credential fetching) and Post-Hook (notifications) with `{{variable}}` template syntax

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    CLI / API / Agent                    │
│                         │                               │
│                         ▼                               │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  │   Gateway    │────▶│   Service    │────▶│  PostgreSQL  │
│  │ Auth/Rate   │     │ Task CRUD    │     │ Tasks/Tenants│
│  └──────────────┘     │ Quota/Audit  │     └──────────────┘
│                       └──────┬───────┘
│                              │ notify
│                       ┌──────▼───────┐
│                       │  Scheduler   │ ◀── Startup: load from PG
│                       │ Time Wheel   │     Runtime: listen for changes
│                       └──────┬───────┘
│                              │ Enqueue
│                       ┌──────▼───────┐     ┌─────────┐
│                       │ Asynq Client │────▶│  Redis  │
│                       └──────────────┘     └────┬────┘
│                                                  │
│                       ┌──────────────┐     ┌────▼────┐
│                       │ Asynq Server  │◀────│  Redis  │
│                       │   (Workers)  │     └─────────┘
│                       └──────┬───────┘
│                              │
│                       ┌──────▼───────┐
│                       │   Target     │ ──▶ HTTP / Feishu / gRPC / MQ
│                       │   Handler   │
│                       └──────────────┘
```

**Scheduler Leader Election**: Uses Redis distributed lock (SETNX + TTL + renewal) for HA. Only one scheduler triggers tasks; other instances run as workers only.

**Micro-scattering**: `fnv32(task_id|trigger_time) % 1000` distributes tasks within 1-second window to avoid peak load.

**Missed Trigger Policy**:
- `fire_once` (default): Triggers once for any missed slots
- `fire_all`: Triggers for each missed slot sequentially
- `skip`: Does not trigger, waits for next natural trigger

## Quick Start

### Prerequisites

- Go 1.22+
- PostgreSQL 15+
- Redis 7.0+
- Node.js 18+ (for frontend development)

### Build & Run

```bash
# Clone and build
git clone https://github.com/tickplatform/tick.git
cd tick/code

# Create database
createdb tick
psql -d tick -f migrations/schema.sql

# Build CLI
make build

# Start server (default :8080)
go run cmd/tick/main.go server

# Or start with embedded frontend
go run cmd/tick/main.go server --http=:8080
```

### Frontend Development

```bash
cd code/web
npm install
npm run dev
# Access http://localhost:5173
```

### Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `TICK_SERVER_ADDR` | Server address | `:8080` |
| `TICK_DATABASE_URL` | PostgreSQL connection | `postgres://tick:tick@localhost:5432/tick` |
| `TICK_REDIS_ADDR` | Redis address | `localhost:6379` |
| `TICK_JWT_SECRET` | JWT signing secret | `change-me-in-production` |

## CLI Usage

```bash
# Login
tick auth login --api-key tk_live_xxx

# Create a cron task
tick task create \
  --name "Daily Report" \
  --cron "0 8 * * *" \
  --url "https://api.example.com/hook" \
  --method POST

# Create an interval task
tick task create \
  --name "Heartbeat" \
  --every 30s \
  --url "https://api.example.com/heartbeat"

# Create a one-time task
tick task create \
  --name "One-time Notification" \
  --at "2026-06-10T09:00:00+08:00" \
  --url "https://api.example.com/notify"

# Manage tasks
tick task list
tick task get <task-id>
tick task pause <task-id>
tick task resume <task-id>
tick task delete <task-id> --yes
tick task history <task-id> --limit 20

# Credential management
tick credential create --name "prod-api" --type bearer --config '{"token":"tk_xxx"}'
tick credential list

# Signature secrets
tick secret create
tick secret list
tick secret revoke <secret-id> --yes

# Check quota and status
tick quota
tick status
tick auth whoami
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Register new tenant |
| POST | `/api/v1/auth/login` | Login with credentials |
| GET | `/api/v1/tasks` | List tasks |
| POST | `/api/v1/tasks` | Create task |
| GET | `/api/v1/tasks/:id` | Get task details |
| PUT | `/api/v1/tasks/:id` | Update task |
| DELETE | `/api/v1/tasks/:id` | Delete task |
| POST | `/api/v1/tasks/:id/pause` | Pause task |
| POST | `/api/v1/tasks/:id/resume` | Resume task |
| GET | `/api/v1/tasks/:id/history` | Execution history |
| GET | `/api/v1/targets` | List targets |
| POST | `/api/v1/targets` | Create target |
| GET | `/api/v1/credentials` | List credentials |
| POST | `/api/v1/credentials` | Create credential |
| GET | `/api/v1/quota` | Quota usage |
| GET | `/api/v1/status` | Platform status |

## Tech Stack

| Layer | Technology | Notes |
|-------|------------|-------|
| Language | Go | Goroutines for high-concurrency scheduling |
| Web Framework | Gin | High-performance HTTP framework |
| Scheduler | Custom Time Wheel | Second-level precision + micro-scattering |
| Execution Engine | Asynq | Redis queue + Worker pool + retry + deduplication |
| Database | PostgreSQL | Tasks, executions, tenants, audit logs |
| Cache/Queue | Redis | Scheduling index + Asynq queue |
| Frontend | React + Vite + TailwindCSS | go:embed single binary |
| CLI Framework | Cobra | Command-line tool |

## Development

```bash
# Run tests
go test ./...

# Build for development (with source path marker)
make build-dev

# Build for SIT environment
make build-sit

# Build cross-platform binaries
make build-cross
```

## Roadmap

| Version | Features |
|---------|----------|
| V1 | CLI + API + second-level scheduling + HTTP Webhook + multi-tenant + Credential Center + Hook Engine |
| V2 | Multi-user roles + tenant login + Dashboard full CRUD + failure alerts + circuit breaker |
| V3 | SDK (Python/Go/JS) + Webhook callback verification + IP whitelist |

## License

MIT License - see [LICENSE](LICENSE) file for details.

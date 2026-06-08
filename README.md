# Tick - 定时触发平台

滴答之间，精准触发。Tick 是一个面向 AI Agent 场景的秒级定时触发平台，支持 Cron、固定间隔、一次性定时三种调度类型，提供 CLI-first 的交互体验。

[English](README-en.md) | 中文

---

## 简介

Tick 是一个面向系统和 AI Agent 用户的秒级定时触发平台。核心理念：**时间轮管"什么时候"触发，Asynq 管"怎么执行"**。

### 核心特性

- **秒级精度**：自建分层时间轮，支持 Cron 表达式（6 位，秒级精度）、固定间隔、一次性定时三种调度类型
- **微打散算法**：同秒触发的任务在 1 秒内均匀散布，避免瞬时峰值
- **多租户隔离**：API Key 认证，租户数据完全隔离，支持 per-tenant 队列优先级
- **CLI-first**：完整的命令行工具，Agent/脚本/终端直接使用
- **可观测**：执行记录完整保留，失败自动重试（指数退避），超时控制
- **凭证中心**：AES-256 加密存储认证凭证，支持 Bearer/Basic/OAuth2/Dynamic HMAC 等类型，自动注入 HTTP 请求
- **Hook 引擎**：支持 Pre-Hook（获取凭证/准备上下文）和 Post-Hook（发送通知），通过 `{{变量名}}` 模板语法传递上下文

### 架构概览

```
┌─────────────────────────────────────────────────────────┐
│                    CLI / API / Agent                    │
│                         │                               │
│                         ▼                               │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  │   Gateway    │────▶│   Service    │────▶│  PostgreSQL  │
│  │ 鉴权/限流   │     │ 任务 CRUD    │     │  任务/租户   │
│  └──────────────┘     │ 配额/审计    │     └──────────────┘
│                       └──────┬───────┘
│                              │ 变更通知
│                       ┌──────▼───────┐
│                       │   调度层     │ ◀── 启动：从 PG 加载
│                       │  时间轮      │     运行：监听变更
│                       └──────┬───────┘
│                              │ 到期 Enqueue
│                       ┌──────▼───────┐     ┌─────────┐
│                       │ Asynq Client │────▶│  Redis  │
│                       └──────────────┘     └────┬────┘
│                                                  │
│                       ┌──────────────┐     ┌────▼────┐
│                       │ Asynq Server │◀────│  Redis  │
│                       │  (Workers)  │     └─────────┘
│                       └──────┬───────┘
│                              │
│                       ┌──────▼───────┐
│                       │  Target      │ ──▶ HTTP / Feishu / gRPC / MQ
│                       │  Handler     │
│                       └──────────────┘
```

**调度器 Leader 选举**：通过 Redis 分布式锁（SETNX + TTL + 续约）实现高可用，只有一个调度器触发任务，其他实例仅作为 Worker 运行。

**微打散算法**：`fnv32(task_id|trigger_time) % 1000` 将任务均匀分布在 1 秒窗口内，单租户同秒任务数受 `quota_max_rps` 约束。

**重启补漏策略**：
- `fire_once`（默认）：错过的触发只补一次
- `fire_all`：逐个补齐所有错过的触发
- `skip`：不补，等下次自然触发

---

## 快速开始

### 前置依赖

- Go 1.22+
- PostgreSQL 15+
- Redis 7.0+
- Node.js 18+（前端开发用）

### 构建与运行

```bash
# 克隆项目
git clone https://github.com/tickplatform/tick.git
cd tick/code

# 创建数据库
createdb tick
psql -d tick -f migrations/schema.sql

# 构建 CLI
make build

# 启动服务（默认 :8080）
go run cmd/tick/main.go server

# 前端开发模式
cd code/web
npm install
npm run dev
# 访问 http://localhost:5173
```

### 配置

环境变量：

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `TICK_SERVER_ADDR` | 服务地址 | `:8080` |
| `TICK_DATABASE_URL` | PostgreSQL 连接 | `postgres://tick:tick@localhost:5432/tick` |
| `TICK_REDIS_ADDR` | Redis 地址 | `localhost:6379` |
| `TICK_JWT_SECRET` | JWT 密钥 | `change-me-in-production` |

---

## CLI 命令

### 登录与身份

```bash
# 登录
tick auth login --api-key tk_live_xxx

# 查看当前身份
tick auth whoami

# 查看配额
tick quota

# 查看平台状态
tick status
```

### 任务管理

```bash
# 创建 Cron 任务
tick task create \
  --name "日报" \
  --cron "0 8 * * *" \
  --url "https://api.example.com/hook" \
  --method POST

# 创建固定间隔任务
tick task create \
  --name "心跳检测" \
  --every 30s \
  --url "https://api.example.com/heartbeat"

# 创建一次性任务
tick task create \
  --name "延迟通知" \
  --at "2026-06-10T09:00:00+08:00" \
  --url "https://api.example.com/notify"

# 查看和管理任务
tick task list
tick task list --status active
tick task get <task-id>
tick task pause <task-id>
tick task resume <task-id>
tick task delete <task-id> --yes
tick task history <task-id> --limit 20
```

### 凭证管理

```bash
# 创建 Bearer Token 凭证
tick credential create --name "生产环境 Token" --type bearer --config '{"token":"tk_xxx"}'

# 列出凭证（脱敏展示）
tick credential list

# 删除凭证
tick credential delete <credential-id> --yes
```

### 签名密钥

```bash
# 创建签名密钥
tick secret create

# 列出签名密钥
tick secret list

# 撤销签名密钥
tick secret revoke <secret-id> --yes
```

### 触发目标

```bash
# 创建 HTTP 触发目标
tick target create \
  --name "webhook-a" \
  --type http \
  --url "https://example.com/hook"

# 列出目标
tick target list

# 删除目标
tick target delete <target-id> --yes
```

---

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 自助注册租户 |
| POST | `/api/v1/auth/login` | 账密登录 |
| GET | `/api/v1/tasks` | 列出任务 |
| POST | `/api/v1/tasks` | 创建任务 |
| GET | `/api/v1/tasks/:id` | 获取任务详情 |
| PUT | `/api/v1/tasks/:id` | 更新任务 |
| DELETE | `/api/v1/tasks/:id` | 删除任务 |
| POST | `/api/v1/tasks/:id/pause` | 暂停任务 |
| POST | `/api/v1/tasks/:id/resume` | 恢复任务 |
| GET | `/api/v1/tasks/:id/history` | 执行历史 |
| GET | `/api/v1/targets` | 列出触发目标 |
| POST | `/api/v1/targets` | 创建触发目标 |
| GET | `/api/v1/credentials` | 列出凭证 |
| POST | `/api/v1/credentials` | 创建凭证 |
| GET | `/api/v1/quota` | 配额使用情况 |
| GET | `/api/v1/status` | 平台状态 |

---

## 技术栈

| 层级 | 技术选型 | 说明 |
|------|----------|------|
| 语言 | Go | goroutine 天然适合高并发调度 |
| Web 框架 | Gin | 高性能 HTTP 框架 |
| 调度引擎 | 自建时间轮 | 秒级精度 + 微打散 |
| 执行引擎 | Asynq | Redis 队列 + Worker 池 + 重试 + 去重 |
| 数据库 | PostgreSQL | 任务 + 执行记录 + 租户 + 审计 |
| 缓存/队列 | Redis | 调度索引 + Asynq 队列 |
| 前端 | React + Vite + TailwindCSS | go:embed 嵌入单二进制 |
| CLI 框架 | Cobra | 命令行工具 |

---

## 开发

```bash
# 运行测试
go test ./...

# 开发构建（写入 .tick-source marker，支持 tick update --from-git）
make build-dev

# SIT 环境构建
make build-sit

# 交叉编译（darwin/linux/windows × amd64/arm64）
make build-cross
```

---

## 路线图

| 版本 | 特性 |
|------|------|
| V1 | CLI + API + 秒级调度 + HTTP Webhook + 多租户 + 凭证中心 + Hook 引擎 |
| V2 | 多用户角色 + 租户账密登录 + Dashboard 完整 CRUD + 连续失败告警 + 熔断机制 |
| V3 | SDK (Python/Go/JS) + Webhook 回调验证 + IP 白名单 |

---

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件。

# Tick 对象设计文档

状态：Draft
作者：Adam
最后更新：2026-05-15
版本：0.1

---

## 1. 设计目标

基于 PRD 中"时间轮管什么时候，Asynq 管怎么执行"的分层原则，定义 Tick 平台的核心对象模型。对象设计需要：

1. 与 asynq 的执行层无缝对接
2. 支持触发目标类型扩展（target 解耦）
3. 支持多租户隔离
4. 支持调度器重启后状态恢复

---

## 2. Asynq 能力评估

### 2.1 Asynq 提供的能力（直接复用）

| 能力 | Asynq 实现 | Tick 用法 |
|------|-----------|-----------|
| 任务队列 | Redis List + Sorted Set | 时间轮到期后 Enqueue 到 asynq |
| Worker 并发 | processor + sema（信号量控制并发数） | 配置 Concurrency 控制同时执行数 |
| 重试 | RetryDelayFunc + MaxRetry | 用 `MaxRetry(n)` 选项 |
| 超时 | Timeout/Deadline option | 用 `Timeout(d)` 选项 |
| 去重 | Unique option（基于 Type+Payload+Queue） | 用 `Unique(ttl)` 防重复入队 |
| 延迟执行 | ProcessAt/ProcessIn | 一次性任务可直接用此 |
| Dead Letter | Archive（exhausted retries） | 自动归档失败任务 |
| 优先级队列 | Weighted/Strict priority | 按租户分队列，设权重 |
| 任务聚合 | Group option + aggregator | 暂不需要 |
| 健康检查 | heartbeater + healthchecker | 复用做 Worker 存活检测 |
| 任务查询 | Inspector API | 用于 `tick task history` 实现 |

### 2.2 Asynq 不提供的能力（Tick 自建）

| 能力 | 为什么 Asynq 不够 | Tick 怎么做 |
|------|-------------------|-------------|
| 持久化调度配置 | Asynq Scheduler 是内存态，重启丢失 | PG 持久化 + 启动时加载 |
| 秒级精度 | Asynq Scheduler 基于 robfig/cron，最小粒度分钟 | 自建时间轮（秒级 tick） |
| 微打散 | Asynq 无此概念 | fnv32 散列到 1s 窗口内 |
| 多租户隔离 | Asynq 无租户概念 | PG 租户表 + 按租户分队列 |
| 触发目标管理 | Asynq 只关心 Task 执行 | targets 表 + type handler |
| 执行记录持久化 | Asynq 的 Retention 有限 | PG executions 表 |
| 补漏策略 | Asynq 不管错过的调度 | missed_policy 字段 |

### 2.3 PeriodicTaskManager 的适用性

Asynq 提供了 `PeriodicTaskManager`，它通过 `PeriodicTaskConfigProvider` 接口定期从外部数据源同步 cron 配置。**但 Tick 不应直接使用它**，原因：

1. 它的同步间隔默认 3 分钟，不满足秒级精度
2. 它无法处理 interval 和 once 类型调度
3. 它的去重基于 config hash，不是基于 trigger_time
4. 它没有微打散能力

**结论**：Tick 自建时间轮做调度决策，仅使用 `asynq.Client.Enqueue()` 做执行投递。

---

## 3. 核心对象模型

### 3.1 领域对象总览

```
┌─────────────────────────────────────────────────────┐
│                    Tick Domain                       │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Tenant ──┬──▶ Task ──▶ Target                     │
│           │                                         │
│           ├──▶ ApiKey                               │
│           │                                         │
│           └──▶ Quota                                │
│                                                     │
│  Task ──▶ Execution (1:N)                          │
│                                                     │
│  Tenant ──▶ AuditLog (1:N)                         │
│                                                     │
├─────────────────────────────────────────────────────┤
│                   Asynq Domain                      │
├─────────────────────────────────────────────────────┤
│  asynq.Task (投递载体)                              │
│  asynq.TaskInfo (执行状态)                          │
│  asynq.Queue (per-tenant queue)                    │
└─────────────────────────────────────────────────────┘
```

### 3.2 Tenant（租户）

```go
type Tenant struct {
    ID            string    // ten_xxxx
    Name          string
    Status        TenantStatus // active | suspended
    QuotaMaxTasks int       // 默认 100
    QuotaMaxRPS   int       // 默认 50
    CreatedAt     time.Time
}

type TenantStatus string

const (
    TenantActive    TenantStatus = "active"
    TenantSuspended TenantStatus = "suspended"
)
```

### 3.3 ApiKey（认证凭证）

```go
type ApiKey struct {
    ID         string    // key_xxxx
    TenantID   string
    KeyHash    string    // SHA-256(raw_key)
    KeyPrefix  string    // 前 8 位，用于审计日志显示
    Status     KeyStatus // active | revoked
    CreatedAt  time.Time
    RevokedAt  *time.Time
}
```

说明：一个租户可以有多个 ApiKey，支持 key 轮换。

### 3.4 Target（触发目标）

```go
type Target struct {
    ID        string          // tgt_xxxx
    TenantID  string
    Name      string
    Type      TargetType      // http | feishu | grpc | mq
    Config    json.RawMessage // 按 type 存储不同结构
    CreatedAt time.Time
    UpdatedAt time.Time
}

type TargetType string

const (
    TargetHTTP   TargetType = "http"
    TargetFeishu TargetType = "feishu"  // V2
    TargetGRPC   TargetType = "grpc"    // V2
    TargetMQ     TargetType = "mq"      // V2
)

// HTTP 类型的 Config 结构
type HTTPTargetConfig struct {
    URL     string            `json:"url"`
    Method  string            `json:"method"` // GET|POST|PUT|DELETE
    Headers map[string]string `json:"headers,omitempty"`
    Body    json.RawMessage   `json:"body,omitempty"`
}
```

### 3.5 Task（任务）

```go
type Task struct {
    ID              string        // t_xxxx
    TenantID        string
    Name            string
    ScheduleType    ScheduleType  // cron | interval | once
    CronExpr        string        // 6 位 cron（秒 分 时 日 月 周）
    IntervalValue   int
    IntervalUnit    IntervalUnit  // s | m | h | d
    OnceAt          *time.Time
    TargetID        string        // → Target.ID
    TimeoutSecs     int           // 默认 30
    RetryCount      int           // 默认 3
    MissedPolicy    MissedPolicy  // fire_once | fire_all | skip
    Status          TaskStatus    // active | paused | deleted
    NextTriggerAt   *time.Time
    TotalExecutions int64
    CreatedAt       time.Time
    UpdatedAt       time.Time
    DeletedAt       *time.Time
}

type ScheduleType string
const (
    ScheduleCron     ScheduleType = "cron"
    ScheduleInterval ScheduleType = "interval"
    ScheduleOnce     ScheduleType = "once"
)

type MissedPolicy string
const (
    MissedFireOnce MissedPolicy = "fire_once"
    MissedFireAll  MissedPolicy = "fire_all"
    MissedSkip     MissedPolicy = "skip"
)

type TaskStatus string
const (
    TaskActive  TaskStatus = "active"
    TaskPaused  TaskStatus = "paused"
    TaskDeleted TaskStatus = "deleted"
)
```

### 3.6 Execution（执行记录）

```go
type Execution struct {
    ID           int64     // BIGSERIAL
    TaskID       string
    TenantID     string
    TriggerTime  time.Time
    Attempt      int       // 第几次尝试 1/2/3
    Status       ExecStatus // success | failed | timeout
    StatusCode   int       // HTTP 响应码
    DurationMs   int
    RequestBody  string    // 截断至 4KB
    ResponseBody string    // 截断至 4KB
    ErrorMsg     string
    IsMakeup     bool      // 是否为补漏触发
    CreatedAt    time.Time
}

type ExecStatus string
const (
    ExecSuccess ExecStatus = "success"
    ExecFailed  ExecStatus = "failed"
    ExecTimeout ExecStatus = "timeout"
)
```

### 3.7 AuditLog（审计日志）

```go
type AuditLog struct {
    ID           int64
    TenantID     string
    Actor        string    // ApiKey 前 8 位
    Action       string    // create | update | delete | pause | resume
    ResourceType string    // task | target | tenant
    ResourceID   string
    Payload      json.RawMessage
    CreatedAt    time.Time
}
```

---

## 4. 对象与 Asynq 的映射关系

### 4.1 投递时的映射

当时间轮决定某个 Task 到期时，构造 asynq.Task 并投递：

```go
func (s *Scheduler) enqueueTask(task *Task, target *Target, triggerTime time.Time) error {
    // 构造 asynq payload
    payload := TriggerPayload{
        TaskID:      task.ID,
        TenantID:    task.TenantID,
        TargetID:    task.TargetID,
        TargetType:  target.Type,
        TargetConfig: target.Config,
        TriggerTime: triggerTime,
        Attempt:     1,
    }
    data, _ := json.Marshal(payload)

    // 构造 asynq task
    asynqTask := asynq.NewTask("tick:trigger", data)

    // 计算微打散偏移
    offset := scatterOffset(task.ID, triggerTime)

    // 投递选项
    opts := []asynq.Option{
        asynq.Queue(fmt.Sprintf("tenant:%s", task.TenantID)), // 租户隔离队列
        asynq.MaxRetry(task.RetryCount),
        asynq.Timeout(time.Duration(task.TimeoutSecs) * time.Second),
        asynq.Unique(60 * time.Second), // 60s 去重窗口
        asynq.ProcessIn(offset),         // 微打散延迟
        asynq.TaskID(fmt.Sprintf("%s|%d", task.ID, triggerTime.Unix())), // 唯一 ID
    }

    _, err := s.asynqClient.Enqueue(asynqTask, opts...)
    return err
}
```

### 4.2 执行时的映射

Worker 收到 `tick:trigger` 类型任务后：

```go
func (h *TriggerHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload TriggerPayload
    json.Unmarshal(t.Payload(), &payload)

    // 根据 target type 分发到不同 handler
    handler := h.handlers[payload.TargetType] // map[TargetType]TargetHandler
    result, err := handler.Execute(ctx, payload)

    // 写执行记录到 PG
    h.recordExecution(payload, result, err)

    return err // 返回 error 则 asynq 自动重试
}
```

### 4.3 租户队列隔离

每个租户对应一个 asynq 队列（`tenant:{tenant_id}`），Server 配置加权优先级：

```go
// 动态生成队列配置
queues := map[string]int{}
for _, tenant := range activeTenants {
    queues[fmt.Sprintf("tenant:%s", tenant.ID)] = tenant.Priority // 默认 1
}

srv := asynq.NewServer(redisOpt, asynq.Config{
    Concurrency: 100,
    Queues:      queues,
})
```

---

## 5. 关键接口定义

### 5.1 TargetHandler（触发目标处理器）

```go
type TargetHandler interface {
    Execute(ctx context.Context, payload TriggerPayload) (*ExecutionResult, error)
}

type ExecutionResult struct {
    StatusCode   int
    ResponseBody []byte // 截断至 4KB
    DurationMs   int
}

// V1 只需实现 HTTPTargetHandler
type HTTPTargetHandler struct {
    client *http.Client
    signer *HMACSigner
}
```

### 5.2 ScheduleProvider（调度配置提供者）

时间轮启动时从 PG 加载任务配置，运行期间监听变更：

```go
type ScheduleProvider interface {
    // 启动时全量加载（只加载需要补漏的）
    LoadPending(ctx context.Context) ([]*Task, error)
    // 运行时监听变更（新建/暂停/删除/恢复）
    WatchChanges(ctx context.Context) <-chan TaskEvent
}

type TaskEvent struct {
    Type EventType // created | paused | resumed | deleted | updated
    Task *Task
}
```

### 5.3 TimingWheel（时间轮）

```go
type TimingWheel interface {
    // 添加任务到时间轮
    Add(task *Task) error
    // 移除任务
    Remove(taskID string) error
    // 启动时间轮
    Start(ctx context.Context) error
    // 停止
    Stop()
}
```

---

## 6. 对象生命周期

### 6.1 Task 生命周期

```
                    ┌─── pause ───┐
                    ▼             │
    create ──▶ active ◀── resume ─┘
                 │
                 ├── delete ──▶ deleted (软删除)
                 │
                 └── once 触发完成 ──▶ deleted
```

### 6.2 Execution 生命周期（一次触发的完整过程）

```
时间轮到期
    │
    ▼
构造 TriggerPayload
    │
    ▼
asynq.Client.Enqueue() ──▶ Redis 队列
    │
    ▼
Worker 取出任务
    │
    ▼
TargetHandler.Execute()
    │
    ├── 成功 (2xx) ──▶ Execution{status: success}
    │
    ├── 失败 (4xx/5xx/网络错误)
    │       │
    │       ▼
    │   asynq 自动重试 (attempt 2/3)
    │       │
    │       ├── 重试成功 ──▶ Execution{status: success, attempt: N}
    │       │
    │       └── 全部失败 ──▶ Execution{status: failed}
    │                         ──▶ Archive (Dead Letter)
    │
    └── 超时 ──▶ Execution{status: timeout}
                   ──▶ asynq 重试
```

---

## 7. 数据流总览

```
┌──────────┐     ┌─────────┐     ┌──────────────┐
│ CLI/API  │────▶│ Service │────▶│  PostgreSQL  │
│          │     │  Layer  │     │  tasks       │
└──────────┘     └────┬────┘     │  targets     │
                      │          │  tenants     │
                      │          │  executions  │
                      │          └──────────────┘
                      │ notify
                      ▼
               ┌──────────────┐
               │  时间轮       │◀── 启动时从 PG 加载
               │  (Scheduler) │    运行时监听变更
               └──────┬───────┘
                      │ 到期：Enqueue
                      ▼
               ┌──────────────┐     ┌─────────┐
               │ Asynq Client │────▶│  Redis  │
               └──────────────┘     └────┬────┘
                                         │
                                         ▼
               ┌──────────────┐     ┌─────────┐
               │ Asynq Server │◀────│  Redis  │
               │  (Workers)   │     └─────────┘
               └──────┬───────┘
                      │
                      ▼
               ┌──────────────┐
               │TargetHandler │──▶ HTTP/Feishu/...
               └──────┬───────┘
                      │ 写结果
                      ▼
               ┌──────────────┐
               │  PostgreSQL  │
               │  executions  │
               └──────────────┘
```

---

## 8. Asynq 用法总结

| Tick 操作 | 对应 Asynq API |
|-----------|---------------|
| 任务到期投递 | `client.Enqueue(task, opts...)` |
| 一次性延迟任务 | `client.Enqueue(task, ProcessAt(t))` |
| 防重复投递 | `Unique(60s)` + `TaskID("taskID|timestamp")` |
| 超时控制 | `Timeout(d)` |
| 重试次数 | `MaxRetry(n)` |
| 租户隔离 | `Queue("tenant:xxx")` + Server queues config |
| Worker 并发 | `Config{Concurrency: N}` |
| 执行结果查询 | `Inspector.GetTaskInfo(queue, taskID)` |
| 队列暂停 | `Inspector.PauseQueue(queue)` |

### 不使用的 Asynq 能力

| 能力 | 为什么不用 |
|------|-----------|
| `Scheduler` | 内存态，不持久化，精度只到分钟 |
| `PeriodicTaskManager` | 同步间隔太长，不支持 interval/once 类型 |
| `Group` (聚合) | Tick 不需要任务聚合 |
| `Retention` | Tick 用 PG 自己存执行记录 |

---

## 9. 关键设计决策

### 9.1 为什么不直接用 Asynq Scheduler？

Asynq 的 `Scheduler` 看起来能直接做 cron 调度，但有三个致命问题：

1. **内存态**：所有 cron entry 存在进程内存，重启后需要重新 Register，没有持久化
2. **分钟级精度**：底层是 `robfig/cron`，最小粒度 1 分钟
3. **无打散**：所有同分钟的任务在同一毫秒触发，无法避免峰值

Tick 选择自建时间轮（秒级 tick + 微打散），仅用 Asynq 的 Client 做执行投递。

### 9.2 为什么按租户分队列？

- Asynq 的 `Queues` 配置支持加权优先级
- 单租户任务爆发不会饿死其他租户
- `Inspector.PauseQueue` 可以暂停单个租户的执行
- 未来可以为高优租户分配独立 Worker

### 9.3 TaskID 格式设计

```
asynq TaskID = "{tick_task_id}|{trigger_unix_timestamp}"
```

保证：
- 同一任务的不同触发时间不冲突（`ErrTaskIDConflict`）
- 同一任务同一时间不重复（幂等）
- 可从 TaskID 反解出 task_id 和触发时间

---

## 10. 数据库 DDL（完整）

```sql
-- 租户表
CREATE TABLE tenants (
    id              VARCHAR(32) PRIMARY KEY,
    name            VARCHAR(255),
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    quota_max_tasks INT NOT NULL DEFAULT 100,
    quota_max_rps   INT NOT NULL DEFAULT 50,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- API Key 表
CREATE TABLE api_keys (
    id          VARCHAR(32) PRIMARY KEY,
    tenant_id   VARCHAR(32) NOT NULL REFERENCES tenants(id),
    key_hash    VARCHAR(64) NOT NULL,  -- SHA-256
    key_prefix  VARCHAR(8) NOT NULL,   -- 前 8 位用于审计
    status      VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash) WHERE status = 'active';

-- 触发目标表
CREATE TABLE targets (
    id          VARCHAR(32) PRIMARY KEY,
    tenant_id   VARCHAR(32) NOT NULL REFERENCES tenants(id),
    name        VARCHAR(255),
    type        VARCHAR(32) NOT NULL,  -- http | feishu | grpc | mq
    config      JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_targets_tenant ON targets(tenant_id);

-- 任务表
CREATE TABLE tasks (
    id              VARCHAR(32) PRIMARY KEY,
    tenant_id       VARCHAR(32) NOT NULL REFERENCES tenants(id),
    name            VARCHAR(255) NOT NULL,
    schedule_type   VARCHAR(16) NOT NULL,  -- cron | interval | once
    cron_expr       VARCHAR(64),
    interval_value  INT,
    interval_unit   VARCHAR(4),  -- s | m | h | d
    once_at         TIMESTAMPTZ,
    target_id       VARCHAR(32) NOT NULL REFERENCES targets(id),
    timeout_secs    INT NOT NULL DEFAULT 30,
    retry_count     INT NOT NULL DEFAULT 3,
    missed_policy   VARCHAR(16) NOT NULL DEFAULT 'fire_once',
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    next_trigger_at TIMESTAMPTZ,
    total_executions BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_tasks_tenant ON tasks(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_next_trigger ON tasks(next_trigger_at) WHERE status = 'active';

-- 执行记录表
CREATE TABLE executions (
    id            BIGSERIAL PRIMARY KEY,
    task_id       VARCHAR(32) NOT NULL,
    tenant_id     VARCHAR(32) NOT NULL,
    trigger_time  TIMESTAMPTZ NOT NULL,
    attempt       INT NOT NULL DEFAULT 1,
    status        VARCHAR(16) NOT NULL,  -- success | failed | timeout
    status_code   INT,
    duration_ms   INT,
    request_body  TEXT,      -- 截断至 4KB
    response_body TEXT,      -- 截断至 4KB
    error_msg     TEXT,
    is_makeup     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_executions_task ON executions(task_id, created_at DESC);
CREATE INDEX idx_executions_tenant ON executions(tenant_id, created_at DESC);

-- 签名密钥表（per-tenant 多 secret，支持轮换）
CREATE TABLE signing_secrets (
    id          VARCHAR(32) PRIMARY KEY,
    tenant_id   VARCHAR(32) NOT NULL REFERENCES tenants(id),
    secret      VARCHAR(64) NOT NULL,   -- 32 字节 hex 编码
    status      VARCHAR(16) NOT NULL DEFAULT 'active', -- active | revoked
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);
CREATE INDEX idx_signing_secrets_tenant ON signing_secrets(tenant_id) WHERE status = 'active';

-- 审计日志表
CREATE TABLE audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    tenant_id     VARCHAR(32) NOT NULL,
    actor         VARCHAR(64) NOT NULL,   -- API Key 前 8 位
    action        VARCHAR(32) NOT NULL,
    resource_type VARCHAR(32) NOT NULL,
    resource_id   VARCHAR(32) NOT NULL,
    payload       JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_tenant ON audit_logs(tenant_id, created_at DESC);

-- 执行记录表分区（按月，保留 1 个月）
-- CREATE TABLE executions (...) PARTITION BY RANGE (created_at);
-- 每月创建新分区，drop 超过 1 个月的旧分区
```

---

## 11. 与 PRD 的差异说明

| 项目 | PRD 原设计 | 本文档调整 | 原因 |
|------|-----------|-----------|------|
| API Key 存储 | tenants 表内 api_key_hash 字段 | 独立 api_keys 表 | 支持一租户多 key，支持 key 轮换 |
| Target | task 表内 trigger_* 字段 | 独立 targets 表 | 类型扩展解耦 |
| Asynq Scheduler | 未明确使用方式 | 明确不使用，只用 Client.Enqueue | 精度和持久化不满足 |
| missed_policy | PRD 补充的新字段 | 保持一致 | — |
| is_makeup | PRD 未涉及 | executions 表新增 | 区分补漏触发和正常触发 |

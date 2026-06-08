# Tick 实施任务清单与校验标准

状态：Draft
作者：Adam
最后更新：2026-05-20
版本：0.1

---

## 1. 项目总览

### 1.1 核心定位

秒级定时触发平台。CLI-first，Agent 友好，多租户隔离。

### 1.2 架构分层

```
┌─────────────────────────────────────────────────┐
│            接入层 (CLI + REST API)               │
├─────────────────────────────────────────────────┤
│            业务逻辑层 (Service)                  │
│  · 租户管理  · 任务 CRUD  · 配额/审计          │
├─────────────────────────────────────────────────┤
│            调度层 (自建)                         │
│  · 时间轮(秒级 tick)  · 微打散  · 补漏策略     │
├─────────────────────────────────────────────────┤
│            执行层 (Asynq)                       │
│  · Redis 队列  · Worker 池  · 重试/超时/去重    │
├─────────────────────────────────────────────────┤
│            触发层 (TargetHandler)                │
│  · HTTP Webhook (V1)  · Feishu/gRPC/MQ (V2)   │
│  · 前端"触发类型"选择器预留扩展入口            │
├─────────────────────────────────────────────────┤
│            存储层                                │
│  · PostgreSQL (配置+记录)  · Redis (队列)       │
└─────────────────────────────────────────────────┘
```

### 1.3 技术栈

| 组件 | 选型 | 版本要求 |
|------|------|----------|
| 语言 | Go | ≥ 1.22 |
| Web 框架 | gin 或 chi | — |
| 执行引擎 | github.com/hibiken/asynq | latest |
| 数据库 | PostgreSQL | ≥ 15 |
| 缓存/队列 | Redis | ≥ 7.0 |
| CLI 框架 | cobra | latest |
| 配置 | viper | latest |
| 迁移工具 | golang-migrate 或 goose | — |
| 前端框架 | React + Vite + TailwindCSS | — |
| 前端嵌入 | go:embed | Go 1.16+ |

---

## 2. 模块划分

```
tick/
├── cmd/                    # CLI 入口
│   └── tick/
│       └── main.go
├── internal/
│   ├── api/               # HTTP API handlers
│   │   ├── router.go
│   │   ├── middleware/    # 鉴权、限流、审计
│   │   ├── task.go
│   │   ├── target.go
│   │   └── tenant.go
│   ├── cli/               # CLI 命令实现
│   │   ├── root.go
│   │   ├── task.go
│   │   ├── target.go
│   │   └── auth.go
│   ├── domain/            # 领域模型（struct + 枚举）
│   │   ├── task.go
│   │   ├── target.go
│   │   ├── tenant.go
│   │   ├── execution.go
│   │   └── audit.go
│   ├── repo/              # 数据库访问层
│   │   ├── task_repo.go
│   │   ├── target_repo.go
│   │   ├── tenant_repo.go
│   │   └── execution_repo.go
│   ├── scheduler/         # 时间轮调度器
│   │   ├── wheel.go       # 时间轮实现
│   │   ├── scatter.go     # 微打散算法
│   │   ├── provider.go    # PG 加载 + 变更监听
│   │   └── makeup.go      # 补漏策略
│   ├── worker/            # Asynq Worker
│   │   ├── handler.go     # TriggerHandler
│   │   ├── http.go        # HTTPTargetHandler
│   │   └── server.go      # Asynq Server 启动
│   ├── auth/              # 认证
│   │   ├── apikey.go      # SHA-256 验证
│   │   └── signer.go      # Webhook HMAC 签名
│   └── config/            # 应用配置
│       └── config.go
├── web/                   # 前端源码（React + Vite）
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx    # 平台概览
│   │   │   ├── TaskList.tsx     # 任务列表
│   │   │   ├── TaskDetail.tsx   # 任务详情 + 执行历史
│   │   │   └── Quota.tsx        # 配额使用
│   │   ├── components/
│   │   ├── api/                 # 后端 API 调用封装
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── tailwind.config.js
├── migrations/            # SQL 迁移文件
├── doc/                   # 文档
└── Makefile
```

---

## 3. 实施任务清单

### Phase 1: 基础设施（Week 1 前半）

| # | 任务 | 产出 | 校验标准 |
|---|------|------|----------|
| 1.1 | 项目脚手架搭建 | go mod init, 目录结构, Makefile | `go build ./...` 通过 |
| 1.2 | 数据库 DDL + 迁移 | migrations/ 下 6 张表的 up/down SQL | `migrate up` 成功，`migrate down` 可回滚，表结构与对象设计文档一致 |
| 1.3 | executions 分区表 DDL | 按月分区的 executions 表 + 旧分区自动清理脚本 | 分区表创建成功，插入数据落入正确分区，清理脚本可 drop 过期分区 |
| 1.4 | 配置管理 | config.go + config.yaml | 支持 PG/Redis/Server 三段配置，环境变量覆盖 |
| 1.6 | 日志规范落地 | slog 初始化 + 统一 logger | structured JSON log，字段含 ts/level/msg/tenant_id/task_id |
| 1.7 | 错误码规范落地 | errors 包 + 统一响应格式 | HTTP status + `{"error": {"code": "QUOTA_EXCEEDED", "message": "..."}}` 格式 |

### Phase 2: 核心域模型 + 数据层（Week 1 后半）

| # | 任务 | 产出 | 校验标准 |
|---|------|------|----------|
| 2.1 | domain 包 — 所有结构体和枚举 | domain/*.go | 编译通过，字段与对象设计文档 §3 一致 |
| 2.2 | repo 包 — Tenant CRUD | tenant_repo.go | 单元测试：创建/查询/更新状态 通过 |
| 2.3 | repo 包 — Target CRUD | target_repo.go | 单元测试：创建/查询/按 tenant 列表/删除 通过 |
| 2.4 | repo 包 — Task CRUD | task_repo.go | 单元测试：创建/查询/暂停/恢复/软删除/更新 next_trigger_at 通过 |
| 2.5 | repo 包 — Execution 写入+查询 | execution_repo.go | 单元测试：写入记录/按 task_id 分页查询/按 tenant_id 查询 通过 |
| 2.6 | repo 包 — AuditLog 写入 | audit_repo.go | 单元测试：写入审计日志 通过 |

### Phase 3: 认证与租户（Week 2 前半）

| # | 任务 | 产出 | 校验标准 |
|---|------|------|----------|
| 3.1 | API Key 生成 + SHA-256 存储 | auth/apikey.go | 生成格式 `tk_live_<32chars>`，存储 SHA-256 hash，验证时常量时间比较 |
| 3.2 | 自助注册租户 API | POST /api/v1/auth/register | 返回 tenant_id + api_key（明文仅此一次），重复注册拒绝 |
| 3.3 | 鉴权中间件 | middleware/auth.go | 无 key → 401；无效 key → 401；有效 key → ctx 注入 tenant_id |
| 3.4 | 限流中间件 | middleware/ratelimit.go | 按 tenant_id 限流，超限 → 429，Redis 令牌桶或滑动窗口 |
| 3.5 | 配额检查 | service 层 | 创建任务超配额 → 403 + 错误消息 |
| 3.6 | Webhook 签名模块 | auth/signer.go | 输入 (tenant_signing_secret, task_id, trigger_time) → 输出 X-Tick-Signature 头 |
| 3.7 | Signing Secret 管理 API | POST/GET/DELETE /api/v1/secrets | 创建返回明文（仅此一次）、列出、撤销；注册租户时自动生成首个 secret |
| 3.8 | API Key 管理 API | GET/POST/DELETE /api/v1/auth/keys | 列出（显示前缀+状态）、创建新 key、撤销旧 key |

### Phase 4: API 层（Week 2 后半）

| # | 任务 | 产出 | 校验标准 |
|---|------|------|----------|
| 4.1 | Task API — CRUD | POST/GET/PUT/DELETE /api/v1/tasks | curl 测试全部 2xx 成功；非本租户数据 404 |
| 4.2 | Task API — pause/resume | POST /tasks/:id/pause, /resume | 暂停后 status=paused，恢复后 status=active |
| 4.3 | Task API — history | GET /tasks/:id/history | 返回 executions 列表，支持 ?limit=N |
| 4.4 | Target API — CRUD | POST/GET/PUT/DELETE /api/v1/targets | curl 测试全部 2xx；删除被任务引用的 target → 400 |
| 4.5 | 语法糖 — task create 自动创建 target | POST /api/v1/tasks 带 url 字段 | 无 target_id 时自动创建 http target 并关联；支持透传 `headers`（map）、`body`（JSON/Form）、`content_type`（json\|form）字段写入 HTTPTargetConfig |
| 4.6 | Quota/Status API | GET /quota, GET /status | 返回当前配额使用量和平台健康状态 |
| 4.7 | JSON 输出支持 | 所有 API 统一 JSON 响应格式 | `{"data": ..., "error": null}` 格式一致 |
| 4.8 | 鉴权模板（前端快捷配置） | Dashboard 表单 | 前端提供 Bearer Token / Basic Auth / 自定义 Header 三种鉴权模板，选择后自动生成对应 Header 写入 target config |

### Phase 5: 调度引擎（Week 2-3）

| # | 任务 | 产出 | 校验标准 |
|---|------|------|----------|
| 5.1 | 时间轮核心 — Add/Remove/Tick | scheduler/wheel.go | 单元测试：添加任务后在正确 slot 触发；Remove 后不再触发 |
| 5.2 | Cron 表达式解析 → next_trigger_at | 使用 robfig/cron/v3 库（SecondOptional 解析器） | 测试 `0 0 8 * * *`（6位秒级）→ 正确计算下次触发 UTC 时间 |
| 5.3 | Interval 类型调度 | interval 计算逻辑 | 测试 `every 30s` → 每 30s 计算出下一次触发时间 |
| 5.4 | Once 类型调度 | once_at 处理 | 触发一次后 task status → deleted |
| 5.5 | 微打散算法 | scheduler/scatter.go | `fnv32("task_id|unix_ts") % 1000` 输出 0-999ms 偏移；同 input 同 output |
| 5.6 | Enqueue 投递 | asynq.Client.Enqueue | 集成测试：时间轮到期 → Redis 队列中出现对应任务 |
| 5.7 | 启动加载 (ScheduleProvider) | scheduler/provider.go | 启动时从 PG 加载 `WHERE status='active' AND next_trigger_at <= NOW()` |
| 5.8 | 运行时变更监听 | PG LISTEN/NOTIFY 或轮询 | 新建/暂停/删除任务 → 时间轮实时响应 |
| 5.9 | 补漏策略 | scheduler/makeup.go | 测试三种策略：fire_once 只补一次、fire_all 逐个补、skip 不补 |
| 5.10 | PG Advisory Lock | 启动时获取锁 | 第二个实例启动 → 获取锁失败 → 退出或等待 |

### Phase 6: 执行引擎（Week 3）

| # | 任务 | 产出 | 校验标准 |
|---|------|------|----------|
| 6.1 | Asynq Server 启动配置 | worker/server.go | 启动后从 per-tenant 队列消费；Concurrency 可配 |
| 6.2 | TriggerHandler — 任务分发 | worker/handler.go | 收到 `tick:trigger` → 按 TargetType 路由到对应 handler |
| 6.3 | HTTPTargetHandler — HTTP 执行 | worker/http.go | 发出 HTTP 请求，带 X-Tick-Task-ID / X-Tick-Timestamp / X-Tick-Signature 头；同时携带用户配置的自定义 Headers 和 Body（JSON/Form 编码） |
| 6.4 | 执行结果写入 | handler 内调 execution_repo | 成功/失败/超时 → executions 表写入正确记录 |
| 6.5 | 重试行为验证 | Asynq MaxRetry + RetryDelayFunc | 模拟目标返回 500 → 重试 3 次 → 全部失败 → Archive |
| 6.6 | 超时行为验证 | Asynq Timeout option | 模拟目标响应慢 → 超过 timeout_secs → 标记 timeout |
| 6.7 | 去重验证 | TaskID + Unique option | 同一 task 同一 trigger_time 重复 Enqueue → 只执行一次 |
| 6.8 | 租户队列隔离 | Queue("tenant:xxx") | 暂停一个租户队列 → 该租户任务停止执行，其他租户不受影响 |
| 6.9 | 动态租户队列注册 | 新租户注册时通知 Worker 添加队列 | 注册新租户 → 无需重启 → 新租户任务可被消费。方案：Asynq Server 使用 IsFailure + 通配队列，或 Server graceful restart |

### Phase 7: CLI（Week 3）

| # | 任务 | 产出 | 校验标准 |
|---|------|------|----------|
| 7.1 | CLI 骨架 + auth login/whoami | cobra 命令 | `tick auth login --token xxx` 存储到 ~/.tick/config；`tick auth whoami` 显示租户信息 |
| 7.2 | tick task create | cron/interval/once 三种 | 创建成功返回 task_id；参数校验错误有清晰提示 |
| 7.3 | tick task list/get/pause/resume/delete | 全部子命令 | 输出表格格式 + `--output json` 格式 |
| 7.4 | tick task history | 执行记录查询 | 显示最近 N 次触发时间/状态码/耗时 |
| 7.5 | tick target create/list/get/delete | 目标管理 | CRUD 正常工作 |
| 7.6 | tick quota / tick status | 配额与状态 | 显示当前使用量/平台状态 |
| 7.7 | --url 语法糖 | task create 自动创建 target | `tick task create --url xxx --cron "..."` 一步完成 |
| 7.8 | tick secret list/create/revoke | 签名密钥管理 | 创建返回明文、列出、撤销正常工作 |
| 7.9 | tick auth keys list/create/revoke | API Key 管理 | 列出显示前缀+状态、创建返回新 key、撤销正常工作 |

### Phase 8: 端到端测试 + 发布准备（Week 3-4）

| # | 任务 | 产出 | 校验标准 |
|---|------|------|----------|
| 8.1 | E2E 场景 1：Cron 任务 | 测试脚本 | 创建 `*/1 * * * *` 任务 → 1 分钟后触发 → history 有记录 |
| 8.2 | E2E 场景 2：一次性任务 | 测试脚本 | 创建 once at +30s → 30s 后触发 → 任务自动 deleted |
| 8.3 | E2E 场景 3：固定间隔 | 测试脚本 | 创建 every 10s → 连续 3 次触发 → 间隔误差 < 2s |
| 8.4 | E2E 场景 4：失败重试 | 测试脚本 | 目标返回 500 → 重试 3 次 → 全部 failed → history attempt=1/2/3 |
| 8.5 | E2E 场景 5：多租户隔离 | 测试脚本 | 租户 A 看不到租户 B 的任务；A 爆发不影响 B 的触发延迟 |
| 8.6 | 补漏测试 | 手动模拟 | 停调度器 2 分钟 → 重启 → fire_once 只补一次 |
| 8.8 | CLI 分发脚本 | install.sh (curl \| bash) | macOS/Linux amd64/arm64 下载正确二进制 |

### Phase 9: 前端 Dashboard（Week 4）

| # | 任务 | 产出 | 校验标准 |
|---|------|------|----------|
| 9.1 | 前端脚手架 | web/ 目录，React + Vite + TailwindCSS | `cd web && npm run dev` 启动成功，显示空白首页 |
| 9.2 | API 调用层封装 | web/src/api/ | 封装 tasks/targets/quota/status 的 fetch 调用，统一错误处理，请求带 API Key Header |
| 9.3 | 平台概览页 (Dashboard) | Dashboard.tsx | 显示：活跃任务数、今日触发次数、成功率、最近失败任务 |
| 9.4 | 任务列表页 | TaskList.tsx | 表格展示：名称、调度类型、状态、下次触发时间、最近执行结果；支持分页 |
| 9.5 | 任务详情页 + 执行历史 | TaskDetail.tsx | 上半部分任务配置信息，下半部分执行记录列表（触发时间、状态码、耗时、attempt） |
| 9.6 | 配额使用页 | Quota.tsx | 显示：已用任务数/总配额、当前 RPS/限额 |
| 9.7 | 前端路由 + 导航 | App.tsx, react-router | 侧边栏导航：概览/任务/配额，路由切换正常 |
| 9.8 | go:embed 静态文件嵌入 | internal/api/static.go | `go build` 后单二进制包含前端；访问 `/` 显示 Dashboard |
| 9.9 | 前端登录页（简易） | Login.tsx | 输入 API Key → 存 localStorage → 后续请求自动带上；无效 key 提示错误 |
| 9.10 | Makefile 集成构建 | Makefile | `make build` 先编译前端再编译 Go，产出单二进制 |
| 9.11 | 交互状态覆盖 | 所有页面 Loading/Empty/Error/Success 态 | 每页至少覆盖 4 种状态；断网时显示错误态而非白屏 |
| 9.12 | 自动刷新 | Dashboard/TaskList 30s 轮询 | 页面显示"上次更新: Xs前"；切换 tab 暂停轮询 |

#### Phase 9 设计规范

**导航结构**：左侧固定侧边栏（概览/任务/配额），顶部显示租户名称 + 登出按钮。

**状态设计要求**：

| 页面 | Loading | Empty | Error | Success |
|------|---------|-------|-------|---------|
| Dashboard | 4 骨架卡片 + 列表占位 | 指标显示 "0"，失败列表显示"暂无" | 红色横幅 + 重试按钮 | 4 指标卡 + 最近失败列表（可点击跳转） |
| TaskList | 表头 + 5 行骨架 | 居中插图 + "还没有任务" + CLI 示例 | 全页错误 + 重试 | 分页表格（名称/类型/状态/下次触发/最近结果） |
| TaskDetail | 左配置骨架 + 右历史骨架 | 配置正常显示，历史区"尚未执行过" | 404→"任务不存在"+返回链接 | 配置面板 + 执行历史表 |
| Quota | 2 进度条骨架 | 进度条 0% + "配额充足" | 错误提示 + 重试 | 2 进度条（>70%黄 >90%红） |
| Login | 按钮 spinner + input disabled | — | 红边框 + 内联错误消息 | 跳转 Dashboard |

**设计约束**：
- 最小宽度 1024px，低于此显示桌面提示
- 不实现 dark mode（V1）
- 无动画/过渡效果，保持简洁
- 30s 轮询刷新，非 WebSocket
- localStorage 存 API Key，无过期

**色彩规范**：success=green-600, error=red-600, warning=yellow-600, primary=blue-600, muted=gray-500

---

## 4. 校验标准汇总

### 4.1 功能校验

| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| 触发成功率 | ≥ 99.9%（排除目标服务错误） | 创建 100 个 every-10s 任务，运行 10 分钟，统计成功率 |
| 触发延迟 P99 | < 1s | 目标服务记录收到时间 vs trigger_time 的差值 |
| 同秒触发能力 | ≥ 1000 | 创建 1000 个同 cron 表达式任务，验证同秒全部投递 |
| 补漏正确性 | fire_once/fire_all/skip 各自行为正确 | 停机 N 分钟后重启，验证补漏数量 |
| 幂等性 | 同一 trigger_time 不重复执行 | 模拟重复 Enqueue，验证 executions 表无重复 |

### 4.2 安全校验

| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| 租户隔离 | 无法跨租户访问 | 用 A 的 key 请求 B 的资源 → 404 |
| API Key 存储 | 数据库中无明文 key | 查 api_keys 表，只有 SHA-256 hash |
| Webhook 签名 | 目标服务可验签 | 用 signing_secret 验算 X-Tick-Signature 一致 |
| 限流生效 | 超限请求被拒绝 | 短时间内发超过 quota_max_rps 的请求 → 429 |

### 4.3 可靠性校验

| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| 调度器重启恢复 | 重启后 active 任务全部恢复调度 | kill → restart → 验证 next_trigger_at 正确 |
| Worker 重启 | 正在执行的任务不丢失 | kill worker → 任务回到 Redis pending → 新 worker 重新执行 |
| PG 断连恢复 | 连接池自动重连 | 断开 PG 30s → 恢复 → 服务正常 |
| Redis 断连恢复 | Enqueue 暂存或报错，不 panic | 断开 Redis → Enqueue 返回错误 → 日志记录 |

### 4.4 性能基线

| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| API 响应 P99 | < 100ms | wrk 或 hey 压测 task list/create |
| 时间轮内存占用 | < 100MB（10 万任务） | pprof heap 观察 |
| Enqueue 吞吐 | ≥ 5000/s | benchmark 测试 asynq client enqueue |
| 单 Worker HTTP 出站 | ≥ 500 req/s | 目标服务为 mock（1ms 响应） |

---

## 5. 里程碑与交付节点

| 里程碑 | 日期目标 | 交付物 | Done 标志 |
|--------|----------|--------|-----------|
| M1: 能存能查 | Week 1 末 | PG 表就绪 + repo 层 + domain 模型 | 所有 repo 单元测试通过 |
| M2: 能认能调 | Week 2 末 | 认证 + API + 调度器投递 | curl 创建任务 → Redis 队列出现 |
| M3: 能触能记 | Week 3 中 | Worker 执行 + 结果写入 + CLI | `tick task create` → 触发 → `tick task history` 有记录 |
| M4: 能跑能测 | Week 3 末 | E2E 5 场景全过 | CI 绿灯 |
| M5: 能看能用 | Week 4 | Dashboard + CLI 分发 | 浏览器访问 `/` 看到任务列表，单二进制部署成功 |

---

## 6. 关键设计决策备忘

| 决策 | 选项 | 结论 | 原因 |
|------|------|------|------|
| 调度器 HA | 单实例 vs 热备 | V1 单实例 | k8s 重启 + missed_policy 覆盖 30s 窗口，内部产品不需要零宕机 |
| Signing secret | 全局 vs per-tenant | per-tenant | 多租户安全隔离原则，成本只是 tenants 表加一列 |
| 时间轮 vs ZSET 扫描 | 自建时间轮 vs Redis ZSET | 时间轮 | O(1) 触发、无轮询、不依赖分布式锁 |
| Asynq 使用范围 | 全用 vs 只用 Client | 只用 Client.Enqueue | Scheduler/PeriodicTaskManager 精度和持久化不满足需求 |
| 前端部署模式 | 同项目 go:embed vs 独立部署 | 同项目 go:embed 嵌入 | 单二进制部署、零跨域、版本天然一致，内部工具无需分离 |
| 前端技术栈 | React+Vite vs Vue | React + Vite + TailwindCSS | 轻量、构建快，Dashboard 只做只读展示 |
| Cron 位数 | 5 位 vs 6 位 | 6 位 cron（秒 分 时 日 月 周） | 秒级精度全场景覆盖，robfig/cron/v3 SecondOptional 解析器原生支持 |
| API Key 哈希 | bcrypt vs SHA-256 | SHA-256 | API Key 高熵随机串，不需要慢哈希，高频验证需要低延迟 |
| 租户队列 | 共享 vs per-tenant | per-tenant asynq queue | 防饿死 + 可单独暂停 + 加权优先级 |
| 时间轮租户隔离 | 隔离 vs 共享 | 共享 | 时间轮只是闹钟，隔离发生在下游队列层 |

---

## 7. 已确认事项

| # | 问题 | 结论 |
|---|------|------|
| 1 | signing_secret 存储位置 | **独立表 `signing_secrets`**，一个租户可有多个 secret，支持轮换 |
| 2 | Target 删除策略 | 有 **active 状态任务**引用时禁止删除（400）；关联任务为 paused/deleted 时允许删除，但返回**预警提示**告知用户哪些任务受影响 |
| 3 | 执行记录保留策略 | 保留 **1 个月**，PG 分区表 by created_at，定期 drop 旧分区 |
| 4 | interval 基准时间 | 上次触发时间（避免漂移累积） |
| 5 | CLI 配置文件路径 | `~/.tick/config.yaml` (token + server_url) |
| 6 | API 错误码规范 | HTTP status + `{"error": {"code": "QUOTA_EXCEEDED", "message": "..."}}` |
| 7 | 日志规范 | structured JSON log (slog)，字段: ts, level, msg, tenant_id, task_id |

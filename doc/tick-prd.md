# Tick 定时触发平台 PRD

状态：Draft
作者：Adam
最后更新：2026-06-03
版本：0.7

---

## 0. 环境信息

| 环境 | 域名 | 用途 |
|------|------|------|
| SIT（测试环境） | ticksit\.example\.com | 集成测试、联调验证 |
| PROD（正式环境） | tick\.example\.com | 生产部署 |

---

## 1. 问题陈述

### 1.1 核心驱动：AI Agent 时代的个人级定时调度

每个用户拥有多个 AI Agent，需要为个人任务定时调度（如定时给 Agent 发消息、定时调接口）。现有方案无法满足"个人级 + CLI-first + Agent 友好"的场景——要么太重（XXL-JOB），要么太简（系统 cron）。

### 1.2 附带收益：统一现有分散实现

公司内各业务系统的定时调度需求长期分散：

1. **能力不统一**：有的系统自建了 cron，有的没有，参差不齐
2. **不可观测**：任务执行状态无记录、失败无感知，静默故障频发
3. **重复建设**：每套系统各自实现定时逻辑，浪费工程资源

### 1.3 证据

- Agent 场景下用户无法便捷地设置定时触发（核心驱动）
- 内部至少 5+ 套系统各自实现定时逻辑，维护成本高
- 多次出现定时任务静默失败，业务方数小时后才发现

---

## 2. 目标与成功指标

| 目标 | 指标 | 目标值 | 度量窗口 |
|------|------|--------|----------|
| Agent 用户采用 | 活跃租户数（至少 1 个任务在运行） | ≥ 20 | 上线 30 天 |
| 触发可靠性 | 触发成功率 | ≥ 99.9% | 上线 30 天 |
| 触发时效 | P99 触发延迟 | < 1s | 上线 60 天 |
| 同秒并发能力 | 同秒触发任务数 | ≥ 1000 | 上线 60 天 |
| 统一调度入口 | 接入系统数 | ≥ 3 | 上线 90 天 |
| 替代重复建设 | 已有系统迁移数 | ≥ 3 | 上线 90 天 |

---

## 3. Non-Goals

明确说明本产品不做什么：

- :x: **不做任务编排**：不支持 A 完成后触发 B 的 DAG 工作流
- :x: **不做工作流引擎**：不支持条件分支、子任务依赖
- :x: **不管理执行环境**：不提供沙箱、不执行代码，只负责"到点调你的 URL"
- :x: **不做 Web IDE**：不提供在线编写任务逻辑的能力
- :x: **不做复杂数据管道**：不是 ETL 工具，不做数据转换
- :white_check_mark: **V2 Dashboard 升级为管理界面**：白色主题，支持任务完整 CRUD（创建/编辑/删除/暂停）、API Key 管理、租户账密登录。前端仍 go:embed 单二进制。

---

## 4. 产品名称与定位

### 4.1 名称：Tick

一句话释义：滴答之间，精准触发。

- 滴答声 = 钟表的精准节拍 = 秒级精度
- 短（4字母）、好打、CLI 友好
- 中英文均无歧义

### 4.2 定位

给系统和 Agent 用的秒级定时触发平台——到点就调，调了就知道，谁的任务谁管。

**交互范式：CLI-first**。Agent/脚本/终端直接使用，不需要打开浏览器。

---

## 5. 用户画像与故事

### 5.1 系统侧用户

**画像**：后端开发 / 运维工程师

**Story 1**：通过一行 CLI 命令创建定时任务，无需自建 cron。

- Given 我有一个 HTTP 接口需要每 5 分钟调用一次，When 我执行 `tick task create --cron "*/5 * * * *" --url "https://api/sync"`，Then 任务创建成功并返回任务 ID
- Given 任务已创建，When 到达调度时间，Then 系统调用指定 URL 并记录执行结果
- Given URL 调用失败，When 重试次数未达上限，Then 系统按指数退避自动重试

**Story 2**：快速定位任务执行问题。

- Given 我有一个运行中的任务，When 我执行 `tick task history <task-id>`，Then 显示最近 N 次执行的触发时间、状态码、耗时
- Given 任务连续失败 3 次，When 我查看历史，Then 能看到失败原因和错误信息

### 5.2 个人侧用户（Agent 运维者）

**画像**：AI Agent 使用者，需要在 Agent 工作流中设置定时触发

**Story 3**：让 Agent 在每天早上 8 点自动执行某个操作。

- Given 我有一个 Agent webhook，When 我执行 `tick task create --cron "0 8 * * *" --url "https://agent/webhook"`，Then 每天早上 8 点自动触发
- Given 我在 Agent 对话中说"每天8点提醒我站会"，When Agent 调用 CLI，Then 自动创建定时任务

**Story 4**：个人任务与他人隔离，配额清晰。

- Given 我用我的 API Key 认证，When 我执行 `tick task list`，Then 只看到我的任务
- Given 我的配额是 100 个任务，When 我尝试创建第 101 个，Then 被拒绝并提示配额不足

### 5.3 系统 Agent（程序化接入）

**画像**：自动化系统 / AI Agent 通过 API 程序化创建和管理任务

**Story 5**：通过 HTTP API 创建一次性定时任务。

- Given 我需要 30 分钟后触发一个回调，When 我调用 `POST /api/v1/tasks` 设置 at 时间，Then 任务创建成功
- Given 一次性任务触发完成，When 我查询任务状态，Then 状态为 completed 且不再触发

---

## 6. 架构

### 6.1 整体架构

```
CLI / API / Agent
       │
       ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Gateway    │────▶│  业务逻辑层   │────▶│  PostgreSQL  │
│  鉴权/限流   │     │ · 任务 CRUD   │     │ · 任务表      │
└──────────────┘     │ · 租户管理    │     │ · 执行记录表  │
                     │ · 配额/审计   │     │ · 租户表      │
                     └───────┬───────┘     │ · 配额表      │
                             │             └──────────────┘
                     ┌───────▼───────┐
                     │  调度层（自建） │
                     │ · 分层时间轮   │
                     │ · 微打散       │
                     │ · 租户配额调度 │
                     └───────┬───────┘
                             │ 到期：asynq.Client.Enqueue()
                             ▼
                     ┌──────────────┐     ┌──────────────┐
                     │  Asynq 层     │────▶│    Redis     │
                     │ · Redis 队列  │     └──────────────┘
                     │ · Worker 池   │
                     │ · 重试/超时   │
                     │ · Dead Letter │
                     └───────┬───────┘
                             │
                     ┌───────▼───────┐
                     │ HTTP Handler  │
                     │ · 调目标 URL  │
                     │ · 记录结果→PG │
                     └───────────────┘
```

### 6.2 核心设计原则

> 时间轮管"什么时候"，Asynq 管"怎么执行"。

- **调度层（自建）**：分层时间轮实现秒级精度 + 微打散避免同秒峰值 + 租户配额调度
- **执行层（Asynq）**：直接复用队列、Worker 池、重试逻辑、超时控制、Dead Letter，节省 ~1400 行代码
- **单调度器保证**：多容器部署时，通过 Redis 分布式锁实现 scheduler leader 选举，运行期间定期续约，锁丢失时自动退出

### 6.2.1 Scheduler Leader 选举机制

**问题**：PostgreSQL `pg_try_advisory_lock` 是会话级锁，在 pgxpool 连接池中可能失效——多个容器可能同时获取锁成功，导致同一时间多个 scheduler 触发任务。

**解决方案**：Redis 分布式锁（SETNX + TTL + 续约）

```
scheduler 启动 → Redis SETNX 获取锁（Key: tick:scheduler:leader，TTL: 60s）
                → 启动 tick 循环
                → 每 30 秒续约一次锁（EXPIRE）
                → 续约失败则停止 scheduler
```

**竞态处理**：
- 容器 A 获取锁成功 → 续约成功 → 持续调度
- 容器 B/C 获取锁失败 → 不启动 scheduler，只做 worker
- 容器 A 崩溃 → 锁 60 秒后自动过期 → 容器 B 获取锁成为新 scheduler

**优势**：
- Redis 锁专为分布式场景设计，不存在会话-连接池映射问题
- 锁自动过期，无需手动清理
- 崩溃节点自动释放锁， failover 自动完成

**协作模式**：
```
┌─────────────────────────────────────────────────────────┐
│   Scheduler (唯一)         Redis (asynq)     Workers   │
│   判断"何时触发"           任务队列         竞争执行    │
│         │                      ▲              │        │
│         └──── Enqueue ─────────┘              └────────┘
└─────────────────────────────────────────────────────────┘
```

scheduler 很轻量（每秒扫一遍 + Enqueue 到 Redis），真正消耗资源的是 worker 执行任务。

### 6.4 微打散算法

同秒触发的任务在 1 秒内均匀散布，避免瞬时峰值：

```
offset_ms = fnv32(fmt.Sprintf("%s|%d", task.ID, triggerTime.Unix())) % 1000
```

- 同一任务每次触发偏移量一致（可预测）
- 单租户同秒任务数受 `quota_max_rps` 约束

### 6.5 重启补漏策略

调度器重启后，对错过的触发按任务级 `missed_policy` 处理：

| 策略 | 行为 | 适用场景 |
|------|------|----------|
| fire_once（默认） | 错过的只补触发一次 | 大多数场景 |
| fire_all | 逐个补齐所有错过的触发 | 计费/对账 |
| skip | 不补，等下次自然触发 | 心跳检测 |

补漏触发请求头附加 `X-Tick-Makeup: true`，方便目标服务区分。重启加载只查 `WHERE status = 'active' AND next_trigger_at <= NOW()`。

### 6.6 调度时间持久化

调度器每次推进任务的 `next_trigger_at`（包括正常 advance 和跳过过期触发的 advanceTo）后，必须同步写回 DB。确保 StartSync 从 DB 重新加载时不会覆盖为旧值导致无限循环触发。

### 6.7 凭证解析与执行

执行 HTTP 请求前解析凭证占位符时，若凭证解析失败（如 token 服务超时），必须中断执行并返回错误触发 Asynq 重试，不得发送未携带认证的请求。

### 6.3 Agent 集成模式

**模式 A：Agent 直接调用 CLI**

```
用户 → Agent → tick task create ... → Tick 平台
```

**模式 B：Agent 调用 HTTP API**

```
用户 → Agent → POST /api/v1/tasks → Tick 平台
```

两种模式共用同一套后端，CLI 是 API 的薄封装。

---

## 7. 功能规格

### 7.1 CLI 命令设计

命名空间：`tick <resource> <action> [flags]`

#### 任务管理

```bash
# Cron 表达式调度
tick task create \
  --name "每日数据同步" \
  --cron "0 8 * * *" \
  --url "https://api.internal/sync" \
  --method POST \
  --headers '{"Authorization":"Bearer xxx"}' \
  --body '{"action":"full_sync"}' \
  --timeout 30 \
  --retry 3

# 一次性定时
tick task once \
  --name "延迟通知" \
  --at "2026-05-16T09:00:00+08:00" \
  --url "https://api.internal/notify"

# 固定间隔
tick task repeat \
  --name "心跳检测" \
  --every 30s \
  --url "https://api.internal/healthz"

# 查询与管理
tick task list                              # 列出我的任务
tick task get <task-id>                     # 查看任务详情
tick task pause <task-id>                   # 暂停
tick task resume <task-id>                  # 恢复
tick task delete <task-id>                  # 删除
tick task history <task-id> --limit 20     # 执行历史
tick task logs <task-id>                    # 实时日志流
```

#### 认证与租户

```bash
tick auth login --token <api-key>          # 设置认证
tick auth whoami                            # 当前身份
tick auth keys list                         # 列出 API Key（显示前缀+状态）
tick auth keys create                       # 创建新 API Key
tick auth keys revoke <key-id>              # 撤销 API Key
tick quota                                  # 查看配额使用
tick status                                 # 平台状态
```

#### 签名密钥管理

```bash
tick secret list                            # 列出签名密钥（用于 Webhook 验签）
tick secret create                          # 创建新签名密钥，返回明文（仅此一次）
tick secret revoke <secret-id>              # 撤销旧密钥
```

#### 触发目标管理

```bash
tick target create --type http --name "同步接口" \
  --url "https://api.internal/sync" --method POST \
  --headers '{"Authorization":"Bearer xxx"}'      # 创建目标，返回 target_id

tick target list                            # 列出我的触发目标
tick target get <target-id>                 # 查看目标详情
tick target delete <target-id>              # 删除目标
```

> 语法糖：`tick task create --url ...` 自动创建 http target 再关联，无需手动管理 target。

#### 输出格式

```bash
# 默认：人可读表格
tick task list

# Agent/AI 友好：JSON
tick task list --output json
```

### 7.2 API 设计

所有 CLI 命令与 API 1:1 对应，RESTful 风格。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/tasks` | 创建任务 |
| GET | `/api/v1/tasks` | 列出任务 |
| GET | `/api/v1/tasks/:id` | 获取任务详情 |
| PUT | `/api/v1/tasks/:id` | 更新任务 |
| DELETE | `/api/v1/tasks/:id` | 删除任务 |
| POST | `/api/v1/tasks/:id/pause` | 暂停任务 |
| POST | `/api/v1/tasks/:id/resume` | 恢复任务 |
| POST | `/api/v1/tasks/:id/trigger` | 立即执行（手动触发） |
| GET | `/api/v1/tasks/:id/history` | 执行历史 |
| POST | `/api/v1/targets` | 创建触发目标 |
| GET | `/api/v1/targets` | 列出触发目标 |
| GET | `/api/v1/targets/:id` | 获取目标详情 |
| PUT | `/api/v1/targets/:id` | 更新目标 |
| DELETE | `/api/v1/targets/:id` | 删除目标 |
| GET | `/api/v1/quota` | 配额查询 |
| GET | `/api/v1/status` | 平台状态 |
| POST | `/api/v1/auth/register` | 自助注册（用户名+密码），返回 tenant_id + 首个 API Key |
| POST | `/api/v1/auth/login` | 账密登录，返回 JWT token |
| GET | `/api/v1/auth/me` | 当前登录租户信息 |
| GET | `/api/v1/auth/keys` | 列出 API Key |
| POST | `/api/v1/auth/keys` | 创建新 API Key（附 name 说明） |
| DELETE | `/api/v1/auth/keys/:id` | 撤销 API Key |
| GET | `/api/v1/secrets` | 列出签名密钥 |
| POST | `/api/v1/secrets` | 创建签名密钥 |
| DELETE | `/api/v1/secrets/:id` | 撤销签名密钥 |

**认证**：API Key（Bearer Token），请求头 `Authorization: Bearer <api-key>`

**API Key 格式**：`tk_live_<32位随机字符串>`，注册时生成

### 7.3 调度能力

| 调度类型 | 说明 | 精度 |
|----------|------|------|
| Cron 表达式 | 6 位 cron（秒 分 时 日 月 周） | 秒级 |
| 固定间隔 | 每隔 N 秒/分/时 | 秒级 |
| 一次性定时 | 指定 ISO 8601 时间点 | 秒级 |

### 7.4 触发方式

**V1：HTTP Webhook**

支持 GET / POST / PUT / DELETE，自动带上签名验证头供目标服务校验来源。

**请求头**：

```
X-Tick-Task-ID: t_abc123
X-Tick-Timestamp: 1715731200
X-Tick-Signature: sha256=<签名>
Content-Type: application/json
```

**请求体（POST/PUT）**：

```json
{
  "task_id": "t_abc123",
  "task_name": "每日数据同步",
  "trigger_time": "2026-05-15T08:00:00+08:00",
  "attempt": 1,
  "payload": { ...用户自定义body }
}
```

### 7.5 可观测能力

| 能力 | V1 | V2 |
|------|:--:|:--:|
| 执行记录（最近 7 天） | ✅ | |
| 请求/响应/状态码/耗时 | ✅ | |
| 失败重试（3次，指数退避） | ✅ | |
| 超时控制（默认30s，可配） | ✅ | |
| CLI 查询命令 | ✅ | |
| 连续失败告警 | | ✅ |
| 熔断（目标连续失败暂停触发） | | ✅ |
| Prometheus 指标 | | ✅ |

### 7.6 多租户与用户管理

#### 7.6.1 租户能力

| 能力 | 说明 |
|------|------|
| 租户隔离 | 每个租户独立任务空间 |
| API Key 管理 | 一个租户可创建多个 API Key，每个 key 全权限，附带用途说明 |
| 配额管理 | 每租户最大任务数、每秒最大触发数 |
| 操作审计 | 记录任务创建/修改/删除操作 |

#### 7.6.2 用户与认证

| 能力 | 说明 |
|------|------|
| 用户独立于租户 | 用户是独立身份，通过成员关系加入租户 |
| 认证方式 | Dashboard：用户名/密码登录 → JWT（含 user_id + tenant_id）；CLI/API：API Key（Bearer Token，直接绑定租户） |
| 多租户归属 | 一个用户可加入多个租户，登录后选择/切换工作空间 |
| 自助注册 | 用户名 + 密码注册，注册后引导创建租户或加入已有租户 |
| 新用户引导 | 无租户用户登录后进入引导页：「创建租户」或「输入邀请码加入」 |
| 修改密码 | 顶部用户菜单入口触发**居中模态弹窗**（带遮罩、ESC 关闭、点遮罩关闭），含「当前密码 / 新密码」两个字段，新密码至少 8 个字符 |

#### 7.6.3 角色权限

| 角色 | 说明 |
|------|------|
| Owner | 创建租户的人自动成为 Owner，拥有全部权限（含成员管理、配额管理、删除租户） |
| Member | 通过邀请加入的用户，可使用租户内全部业务功能（任务/凭证/变量等），不可管理成员和租户配置 |

**权限矩阵：**

| 能力 | Owner | Member |
|------|-------|--------|
| 使用任务/凭证/变量等业务功能 | ✅ | ✅ |
| 邀请/移除成员 | ✅ | ❌ |
| 变更成员角色 | ✅ | ❌ |
| 管理租户配置/配额 | ✅ | ❌ |
| 删除租户 | ✅ | ❌ |

#### 7.6.4 邀请机制

| 能力 | 说明 |
|------|------|
| 邀请码 | Owner 生成邀请码（短码），被邀请人输入后加入租户 |
| 分享链接 | 邀请码可组装为链接，点击后自动填充邀请码 |
| 有效期 | 邀请码有过期时间 |
| 使用次数 | 可限制使用次数（0=无限制） |
| 默认角色 | 邀请时指定加入后的角色，默认 Member |

#### 7.6.5 直接添加成员

| 能力 | 说明 |
|------|------|
| 搜索用户 | Owner 可通过用户名模糊搜索已注册用户 |
| 指定角色 | 添加时可选择角色（Owner / Member） |
| 直接加入 | 无需被邀请方确认，添加后立即生效 |
| 去重校验 | 已是租户成员的用户不可重复添加 |

### 7.7 凭证中心（Credential Center）

统一管理各类鉴权信息，支持多种凭证类型，加密存储，按需获取和缓存。

#### 凭证类型

| 类型 | 说明 | 配置字段 |
|------|------|----------|
| bearer | Bearer Token | token |
| basic | 用户名密码 | username, password（自动 Base64 编码） |
| custom_header | 自定义 Header | headers (map) |
| hmac | HMAC 签名 | secret |
| dynamic | 动态获取 Token | token_request, token_extract |
| oauth2_cc | OAuth2 Client Credentials | token_url, client_id, client_secret, scope |

#### 凭证生命周期

- **active**：正常可用
- **disabled**：停用，可恢复
- **deleted**：删除，不可恢复

#### 动态凭证缓存

- dynamic / oauth2_cc 类型支持自动缓存
- 缓存过期前自动重新获取
- 使用 Redis 缓存 + singleflight 防止缓存击穿
- 重试策略：失败重试 2 次，1 秒指数退避

#### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/credentials` | 创建凭证 |
| GET | `/api/v1/credentials` | 列出凭证（掩码展示） |
| GET | `/api/v1/credentials/:id` | 获取凭证详情 |
| GET | `/api/v1/credentials/:id/config` | 获取解密配置（仅一次） |
| PUT | `/api/v1/credentials/:id` | 更新凭证 |
| PATCH | `/api/v1/credentials/:id/status` | 更新状态 |
| DELETE | `/api/v1/credentials/:id` | 删除凭证 |

#### 安全设计

- 敏感字段（token, password, secret）AES-256 加密存储
- config_preview 字段仅显示掩码（如 `sk-****xyz`）
- 凭证明文仅在创建时返回一次
- 凭证按 tenant_id 隔离

### 7.8 Hook 引擎（Hook Engine）

支持任务级前置 Hook（Pre-Hook）和后置 Hook（Post-Hook），实现变量上下文传递和执行编排。

#### Pre-Hook（前置 Hook）

在主请求发送前执行，用于：
- 从凭证中心获取 Token 并注入到主请求
- 调用外部接口获取动态参数
- 准备主请求需要的上下文数据

**类型**：
- `credential`：从凭证中心解析凭证并注入
- `http`：发送自定义 HTTP 请求

**配置示例**：
```json
{
  "type": "credential",
  "credential_id": "cred_xxx",
  "inject": {
    "location": "header",
    "key": "Authorization",
    "prefix": "Bearer "
  }
}
```

```json
{
  "type": "http",
  "timeout_secs": 10,
  "request": {
    "url": "https://api.internal/get-token",
    "method": "POST"
  },
  "extract": {
    "path": "data.token",
    "as": "dynamic_token"
  }
}
```

#### Post-Hook（后置 Hook）

在主请求完成后执行，用于：
- 发送通知（成功/失败/始终）
- 清理资源
- 链式调用

**类型**：
- `http`：发送自定义 HTTP 请求
- `feishu`：发送飞书通知

**触发条件**：
- `success`：仅主请求成功时触发
- `failure`：仅主请求失败时触发
- `always`：无论成功失败都触发

**配置示例**：
```json
{
  "type": "http",
  "when": "success",
  "request": {
    "url": "https://notify.internal/slack",
    "method": "POST",
    "body": {"text": "任务 {{task_id}} 执行成功，耗时 {{duration_ms}}ms"}
  }
}
```

#### 变量上下文

Hook 之间通过变量上下文传递数据，支持 `{{变量名}}` 模板语法：

**内置变量**：
- `task_id`：任务 ID
- `execution_status`：执行状态（success/failed/timeout）
- `status_code`：HTTP 响应码
- `duration_ms`：执行耗时
- `trigger_time`：触发时间

**用户定义变量**：
- 从 Pre-Hook 响应中提取
- JSONPath 语法：`{"path": "data.batch_id", "as": "batch_id"}`

**模板渲染**：
- 支持在 URL、Header、Body 中使用 `{{变量名}}`
- 变量未定义或提取失败视为错误，中止执行

#### 执行顺序

1. **Pre-Hook 依次执行** → 提取变量，注入凭证
2. **发送主请求**
3. **Post-Hook 依次执行** → 按触发条件判断是否执行

#### 执行结果

每个 Hook 的执行结果记录在 Execution 的 `hooks_result` 字段中：
```json
{
  "pre_hooks": [
    {"index": 0, "type": "credential", "status": "success", "duration_ms": 12}
  ],
  "post_hooks": [
    {"index": 0, "type": "http", "when": "success", "status": "success", "duration_ms": 45}
  ],
  "credentials_injected": [
    {"credential_id": "cred_xxx", "inject_key": "Authorization", "status": "success"}
  ]
}
```

### 7.9 CLI 安装 / 配置 / 使用 / 更新设计

把 CLI 作为"系统能力对开发者与 AI Agent 的出口"，明确拆为四个节点。每个节点都对应一个独立的交付物，避免配置项/URL/凭据混在同一个配置文件里。

#### 7.9.1 安装方式（Git 为主）

**主路径**：开发者通过 `git clone` + `make build` 自构建。Makefile 提供分层目标：

| 目标 | 用途 |
|------|------|
| `make build` | 标准发布构建（CI 用，注入 prod URL） |
| `make build-sit` | SIT 验证构建（URL 翻转为 SIT 域名） |
| `make build-dev` | 开发者本机构建（注入 `SourcePath` + 写 `.tick-source` marker） |
| `make build-cross` | 交叉编译 darwin/linux/windows × amd64/arm64 |
| `make install-dev` | dev 同学一键安装到 `/usr/local/bin/tick` |

**备用路径**：`install.sh` 从 GitHub Releases / 内部制品库下载预构建二进制（含 SHA256 校验）。

**关键约束**：
- URL、内网地址等"技术性信息"在 Makefile 阶段通过 `-ldflags` 烧进二进制，**用户看不到、也不应修改**
- 同一份源码可以出"连 SIT 的 tick"和"连 prod 的 tick"——二者是独立二进制
- SIT 二进制不入 install.sh / 制品库分发渠道，**仅供本机 dev/test 自 build**

#### 7.9.2 环境配置（Key 初始化，URL 硬编码）

**核心原则**：
- **URL / API 路径前缀 / 内网域名** — 编译期硬编码，**用户不应配置**
- **API Key** — 凭据，**用户必须初始化**
- **SIT / PROD 切换** — 通过 build 时选定（`BuiltForEnv` 编译变量），**不是运行时切换**

**用户配置文件** `~/.tick/config.yaml`（极简）：

```yaml
current_env: prod
api_keys:
  prod: tk_xxx
  # sit: tk_yyy    # 可选——dev 同学本地登录 SIT 时写入
output: json
```

**`tick auth login` 流程**：
- 默认登录当前二进制指向的环境（由 `BuiltForEnv` 决定），交互式提示输入 API Key
- `--api-key` flag 支持非交互场景（CI / 脚本）
- 服务器二进制（`BuiltForEnv=prod`）上 `--env sit` 直接报错："当前二进制不支持 SIT，请使用 make build-sit 重新构建"
- 配置文件权限强制 `0o600`，原子写入（写临时文件 + `os.Rename`）

**凭据解析优先级**（用于 cron / CI 等无交互场景）：
```
TICK_API_KEY 环境变量（不落盘，CI / cron 用）
> api_keys[current_env]
> 未配置 → 报错提示先 tick auth login
```

URL 直接来自二进制内嵌的常量，**不接受任何环境变量 / flag 覆盖**——这正是"URL 写死"的设计意图。

**SIT 的边界**：
- 默认 install 路径里没有 SIT 二进制
- 本机 dev 想测 SIT：`make build-sit` 自行构建 `bin/tick-sit`
- 服务器上不会有 SIT 二进制——避免误操作

**旧配置迁移**：启动时检测旧 `server_url` / `token` 字段，迁移到 `api_keys[default]`，丢弃 `server_url`（不再需要），仅迁移一次。

#### 7.9.3 使用方式（保留现有 + 完善 help）

**命令树**（保持现状，补充 `tick update`）：

```
tick --version                      # 版本信息
tick --help                         # 全局帮助

tick auth   login / whoami / keys / quota / status
tick task   list / get / create / pause / resume / delete / history
tick target list / get / create / update / delete
tick credential list / get / create / update / delete
tick secret   list / create / revoke
tick update                         # 新增——自更新
```

**`help` / `whoami` 增强**：
- 每个子命令必须有清晰的 `Short` 描述（一行说清用途）和 `Long` 描述（含至少一个示例）
- `tick whoami` 输出包含 `tenant / api_key / server / built` 四元组，一眼看出"连的是哪个环境、用的哪个二进制"
- 错误信息包含 HTTP 状态码 + 原因 + 修复建议（如 "API key 无效，请重新 tick auth login"）

**统一约定**：
- 全命令支持 `--output json` / `--output table`
- 危险操作（`delete` 等）加 `--yes` / `-y` 跳过二次确认
- 长 help (`--help`) 比 Short 详细，含至少一个示例

#### 7.9.4 更新迭代（`tick update`）

CLI 工具自我更新是常见诉求。**先识别当初是怎么装的**——三种安装路径对应三种更新机制。

**安装模式探测**（marker 文件）：
- `make build-dev` 时在二进制同目录写 `.tick-source`（一行，源码绝对路径）
- `tick update` 启动时读取该 marker：有 → git 模式；无 → release 模式
- `--from-git` / `--from-go` / `--from-release` flag 强制覆盖

**三种更新路径**：

| 模式 | 触发 | 行为 |
|------|------|------|
| **release**（默认） | 无 `.tick-source` | 查 GitHub `/repos/tickplatform/tick/releases/latest` → 下载 `tick-${OS}-${ARCH}` + `.sha256` → 校验 → 原子替换 |
| **from-git** | 有 `.tick-source` | `cd $(cat .tick-source) && git pull && make build` → 替换当前二进制 |
| **from-go** | `--from-go` flag | `go install github.com/tickplatform/tick/cmd/tick@<version>`（需本地有 Go 工具链） |

**命令树**：

```bash
tick update                       # 自动探测
tick update --check               # 仅检查，不动手
tick update --from-release        # 强制 release
tick update --from-git            # 强制 git
tick update --from-go             # 强制 go install
tick update v0.3.0                # 显式指定目标版本
tick update --force               # 跳过版本比较
```

**关键工程细节**：
- **原子替换**：`mv tick tick.old && mv tick.new tick && chmod +x tick && rm tick.old`；失败时回滚
- **权限**：`/usr/local/bin` 写不了 → 提示"请用 sudo tick update"并退出 1
- **SHA256 校验**：GitHub release asset 旁挂 `.sha256`；不匹配 abort
- **运行中进程**：Linux inode 引用，已运行的 `tick` 不受影响；新进程拿新二进制
- **CI 环境**：`/usr/local/bin` 不可写 → fail fast，提示"请重跑 install.sh"
- **首次升级**：老版本无 metadata → 走"查 latest → 下载 → 替换"路径，覆盖后自带 metadata

**库选型**：`github.com/rhysd/go-github-selfupdate`（纯 Go、跨平台、含 SHA256 校验、API 简单）。

#### 7.9.5 Build Metadata 注入

`code/cmd/tick/main.go`（或 `internal/cli/root.go`）新增包级变量，由 Makefile 通过 `-ldflags` 注入：

```go
var (
    Version       = "dev"
    Commit        = "unknown"
    BuildTime     = "unknown"
    SourcePath    = ""                      // dev build 时填源码绝对路径

    ProdServerURL = "https://tick\.example\.com"    // 编译期硬编码
    SITServerURL  = "https://ticksit\.example\.com"  // 编译期硬编码
    BuiltForEnv   = "prod"                       // "prod" | "sit"，build 决定
)
```

`tick --version` 输出格式：
```
tick version v0.2.0 (commit a1b2c3d, built 2026-06-01)
```

Makefile 注入示例：
```makefile
VERSION    := $(shell git describe --tags --always --dirty)
COMMIT     := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)

build:
    go build -ldflags "$(LDFLAGS) -X main.ProdServerURL=https://tick\.example\.com -X main.BuiltForEnv=prod" -o bin/tick ./cmd/tick

build-sit:
    go build -ldflags "$(LDFLAGS) -X main.ProdServerURL=https://ticksit\.example\.com -X main.BuiltForEnv=sit" -o bin/tick-sit ./cmd/tick
```

#### 7.9.6 Auth Middleware 与错误码映射（增量）

> 借鉴飞书 CLI 做法：把"鉴权注入"和"错误码到用户提示"两件事从每个子命令里抽出来，作为横切关注点统一处理。目的是让每个子命令的 `Run` 函数只关心"自己的业务参数"，不重复关心 key 解析、HTTP 错误格式。

##### 7.9.6.1 Auth Middleware（`RequireAuth` 横切）

**问题**：当前每个调用 `doRequest` 的子命令都依赖 `ResolveAPIKey`（在 `doRequest` 内调用），但 `tick whoami`、`tick auth login`、`tick auth logout`、`tick update` 等子命令其实有差异化的鉴权需求：
- `auth login` / `auth logout` / `update` / `help` / `version` 不应要求已登录
- 其余所有调用 `/api/v1/*` 的子命令都必须先有可用的 key

**方案**：在 cobra 根命令上挂 `PersistentPreRunE`（仅对鉴权敏感子命令生效；通过"opt-out 列表"明确豁免）。`RequireAuth` 中间件职责：
1. 解析 `TICK_API_KEY` 环境变量；命中即用（不落盘）
2. 否则读 `~/.tick/config.yaml` 中 `api_keys[BuiltForEnv]`
3. 两者都没有 → 输出"未登录: 请先执行 `tick auth login`"，退出码 `2`（与 cobra 解析失败的 `1` 区分）
4. 解析成功 → 注入到 `cmd.Context()`，下游 `doRequest` 从 ctx 读取（不再每次 `LoadConfig` + `ResolveAPIKey`）

**豁免清单**（不需登录即可执行）：
- `auth login` / `auth logout` / `auth whoami`（whoami 在没 key 时仍展示本地元数据，server 调用降级为可选）
- `update`（自更新不依赖 API Key）
- `help` / `--version`（cobra 内建）
- 全局 `--config` flag 处理前的早期错误

**安全约束**（与 FR-015 一致）：`TICK_API_KEY` 路径**绝不**写入 `~/.tick/config.yaml`，`ResolveAPIKey` 与 `RequireAuth` 都**不**经过 viper。

##### 7.9.6.2 HTTP 错误码 → 用户友好提示映射

**问题**：当前 `fixSuggestion(status)` 在 `auth.go` 内仅覆盖 401/403/404/409，输出格式裸 `fmt.Fprintf`。5xx 全部走"查看 help"过于简陋，429 限流未识别 `Retry-After`，网络错误与 HTTP 错误混在一起。

**方案**：抽出独立 `error_mapper.go`（与 `doRequest` 解耦），按状态码分类输出：

| 状态码 / 类别 | 提示文案 | 修复建议 |
|--------------|----------|----------|
| 400 Bad Request | `<server msg>` | 检查参数 |
| 401 Unauthorized | `<server msg>` | API key 无效或过期，请重新 `tick auth login` |
| 403 Forbidden | `<server msg>` | 权限不足，联系租户 Owner |
| 404 Not Found | `<server msg>` | 资源不存在（ID 是否正确？） |
| 409 Conflict | `<server msg>` | 资源冲突（重名 / 已删除 / 已存在） |
| 422 Unprocessable | `<server msg>` | 参数语义错误（cron 表达式、URL 格式） |
| 429 Too Many Requests | `<server msg>` | 请求过于频繁，Retry-After: <N> 秒（从响应头读取） |
| 500 Internal Error | `<server msg>` | 服务端内部错误，稍后重试或联系管理员 |
| 502 Bad Gateway | `<server msg>` | 上游网关错误 |
| 503 Unavailable | `<server msg>` | 服务暂时不可用（健康检查失败），稍后重试 |
| 504 Gateway Timeout | `<server msg>` | 上游超时 |
| 其他 4xx | `<server msg>` | 请求问题，参考 `tick --help` |
| 其他 5xx | `<server msg>` | 服务端问题，稍后重试 |
| 网络错误 (timeout/conn refused/DNS) | `<err msg>` | 检查网络 / 服务可达性 / 代理设置 |

**统一输出格式**（保持与 FR-021 一致）：
```
tick <command>: <HTTP status> <reason>: <server msg> (<fix suggestion>)
```

**退出码分级**（让脚本/CI 能区分错误类型）：
- 退出码 0：成功
- 退出码 1：通用业务错误（包含资源不存在、参数错误等）
- 退出码 2：未登录（鉴权失败）
- 退出码 3：网络/服务端不可用
- 退出码 4：限流（429，可指数退避重试）

**结构化 JSON 错误模式**：当 `--output json` 启用时，错误也以 JSON 输出（便于 Agent 解析）：
```json
{
  "error": true,
  "command": "tick task list",
  "status_code": 401,
  "reason": "Unauthorized",
  "server_message": "API key expired",
  "fix_suggestion": "请重新执行 tick auth login",
  "exit_code": 2
}
```

---

## 8. 数据模型

### 8.1 触发目标表（targets）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(32) | 主键，格式：`tgt_` + 随机字符 |
| tenant_id | VARCHAR(32) | 租户 ID，外键 |
| name | VARCHAR(255) | 目标名称（可选） |
| type | VARCHAR(32) | 触发类型：http / feishu / grpc / mq（V1 仅实现 http） |
| config | JSONB | 按 type 存储不同结构的配置 |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

**config 按 type 的结构**：

```json
// type = "http"
{"url": "https://api/sync", "method": "POST", "headers": {...}, "body": {...}}

// type = "feishu" (V2)
{"webhook_url": "https://open.feishu.cn/...", "msg_type": "text", "template": "..."}
```

### 8.2 任务表（tasks）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(32) | 主键，格式：`t_` + 22位随机字符 |
| tenant_id | VARCHAR(32) | 租户 ID，外键 |
| name | VARCHAR(255) | 任务名称 |
| schedule_type | ENUM | cron / interval / once |
| cron_expr | VARCHAR(64) | Cron 表达式（6 位，如 `0 0 8 * * *`） |
| interval_value | INT | 间隔值 |
| interval_unit | ENUM | s / m / h / d |
| once_at | TIMESTAMPTZ | 一次性任务触发时间 |
| target_id | VARCHAR(32) | 触发目标 ID，外键 → targets.id |
| timeout_secs | INT | 超时秒数，默认 30 |
| retry_count | INT | 最大重试次数，默认 3 |
| retry_backoff | ENUM | exponential / fixed / none，默认 exponential |
| concurrency_policy | ENUM | allow / skip / queue，默认 skip |
| max_concurrency | INT | 最大并发执行数，默认 1 |
| execution_retention_days | INT | 执行记录保留天数，默认 30，上限 90 |
| missed_policy | ENUM | fire_once / fire_all / skip，默认 fire_once |
| pre_hooks | JSONB | 前置 Hook 列表 |
| post_hooks | JSONB | 后置 Hook 列表 |
| status | ENUM | active / paused / deleted |
| next_trigger_at | TIMESTAMPTZ | 下次触发时间（索引） |
| total_executions | BIGINT | 累计执行次数 |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |
| deleted_at | TIMESTAMPTZ | 软删除时间 |

> **设计说明**：task 只关心"什么时候触发"，target 关心"怎么触发谁"。同一个 target 可被多个 task 复用。CLI 提供语法糖：`tick task create --url ...` 自动创建 http target 再关联，用户体验不变。

### 8.3 执行记录表（executions）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| task_id | VARCHAR(32) | 任务 ID，外键 |
| tenant_id | VARCHAR(32) | 租户 ID |
| trigger_time | TIMESTAMPTZ | 触发时间 |
| attempt | INT | 第几次尝试（1/2/3） |
| status | ENUM | success / failed / timeout |
| status_code | INT | HTTP 响应码 |
| duration_ms | INT | 执行耗时（毫秒） |
| request_headers | TEXT | 发出请求头（含注入的凭证） |
| request_body | TEXT | 发出请求体（截断至 4KB） |
| response_body | TEXT | 响应体（截断至 4KB） |
| error_msg | TEXT | 错误信息 |
| is_makeup | BOOL | 是否为补漏触发 |
| is_manual | BOOL | 是否手动触发 |
| triggered_by | VARCHAR(16) | 触发来源：scheduler / manual |
| hooks_result | JSONB | Hook 执行结果 |
| created_at | TIMESTAMPTZ | 创建时间 |

### 8.4 用户表（users）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(32) | 主键，格式：`usr_` + 随机字符 |
| username | VARCHAR(64) | 登录用户名，唯一 |
| password_hash | VARCHAR(128) | 密码哈希（bcrypt） |
| display_name | VARCHAR(255) | 显示名称 |
| email | VARCHAR(255) | 邮箱（可选，唯一） |
| status | ENUM | active / suspended |
| created_at | TIMESTAMPTZ | 创建时间 |

### 8.5 租户表（tenants）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(32) | 主键，格式：`ten_` + 随机字符 |
| name | VARCHAR(255) | 租户名称 |
| quota_max_tasks | INT | 最大任务数，默认 100 |
| quota_max_rps | INT | 每秒最大触发数，默认 50 |
| status | ENUM | active / suspended |
| created_at | TIMESTAMPTZ | 创建时间 |

### 8.6 租户成员表（tenant_members）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(32) | 主键，格式：`mbr_` + 随机字符 |
| tenant_id | VARCHAR(32) | 租户 ID，外键 → tenants.id |
| user_id | VARCHAR(32) | 用户 ID，外键 → users.id |
| role | VARCHAR(16) | 角色：owner / member |
| joined_at | TIMESTAMPTZ | 加入时间 |

> 联合唯一约束：(tenant_id, user_id)

### 8.7 邀请表（invitations）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(32) | 主键，格式：`inv_` + 随机字符 |
| tenant_id | VARCHAR(32) | 租户 ID，外键 → tenants.id |
| code | VARCHAR(64) | 邀请码，唯一 |
| created_by | VARCHAR(32) | 创建人用户 ID，外键 → users.id |
| role | VARCHAR(16) | 加入后角色，默认 member |
| max_uses | INT | 最大使用次数，0=无限制 |
| used_count | INT | 已使用次数 |
| expires_at | TIMESTAMPTZ | 过期时间 |
| created_at | TIMESTAMPTZ | 创建时间 |

### 8.8 API Key 表（api_keys）

> **V1→V2 变更**：原 tenants 表的 api_key_hash 字段移除，改为独立 api_keys 表实现一对多。注册时自动创建第一个 API Key。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(32) | 主键，格式：`key_` + 随机字符 |
| tenant_id | VARCHAR(32) | 租户 ID，外键 → tenants.id |
| name | VARCHAR(255) | Key 用途说明（如 "production"、"agent-bot"） |
| key_hash | VARCHAR(64) | API Key 哈希（SHA-256） |
| key_prefix | VARCHAR(12) | Key 前缀，用于列表展示（如 `tk_live_abc...`） |
| status | ENUM | active / revoked |
| created_at | TIMESTAMPTZ | 创建时间 |
| revoked_at | TIMESTAMPTZ | 撤销时间 |

### 8.9 审计日志表（audit_logs）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| tenant_id | VARCHAR(32) | 租户 ID |
| actor | VARCHAR(64) | 操作者（API Key 前8位） |
| action | VARCHAR(32) | 操作类型 |
| resource_type | VARCHAR(32) | 资源类型 |
| resource_id | VARCHAR(32) | 资源 ID |
| payload | JSONB | 操作详情 |
| created_at | TIMESTAMPTZ | 操作时间 |

### 8.10 凭证表（credentials）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | VARCHAR(32) | 主键，格式：`cred_` + 随机字符 |
| tenant_id | VARCHAR(32) | 租户 ID，外键 |
| name | VARCHAR(128) | 凭证名称 |
| code | VARCHAR(64) | 凭证代码（租户内唯一） |
| type | VARCHAR(32) | 凭证类型：bearer / basic / oauth2_cc / dynamic / hmac / custom_header |
| config | BYTEA | 加密后的配置（AES-256） |
| config_preview | JSONB | 配置预览（掩码展示） |
| timeout_secs | INT | 凭证获取超时秒数，默认 10 |
| status | VARCHAR(16) | 状态：active / disabled / deleted |
| created_at | TIMESTAMPTZ | 创建时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

**config 按 type 的结构**：

```json
// type = "bearer"
{"token": "sk_live_xxx"}

// type = "basic"
{"username": "user", "password": "pass"}

// type = "hmac"
{"secret": "hmac_secret_key"}

// type = "custom_header"
{"headers": {"X-API-Key": "xxx"}}

// type = "dynamic"
{"token_request": {"url": "...", "method": "POST", "body": {...}}, "token_extract": {"path": "data.token", "ttl_secs": 3600}}

// type = "oauth2_cc"
{"token_url": "https://auth.example.com/oauth/token", "client_id": "xxx", "client_secret": "xxx", "scope": "read write"}
```

---

## 9. 任务生命周期

### 9.1 状态机

```
created ──▶ active ──▶ paused ──▶ active ──▶ deleted
   │                ▲                │
   │                │                │
   └──── once: 触发完成后 ─────────────┘
```

- **active**：正常调度中
- **paused**：暂停，不触发，但保留任务
- **deleted**：软删除，不再调度，不出现在列表中
- **once**：一次性任务，触发后自动进入 deleted 状态

### 9.2 执行状态

| 状态 | 说明 |
|------|------|
| success | HTTP 响应码 2xx |
| failed | HTTP 响应码 4xx/5xx 或网络错误 |
| timeout | 超过 timeout_secs 无响应 |
| skipped | 被并发控制跳过（concurrency_policy=skip 时触发） |

### 9.3 重试策略

- 默认 3 次重试
- 退避策略：可配置（exponential / fixed / none），默认 exponential
  - exponential：10s → 30s → 90s
  - fixed：固定间隔 10s
  - none：立即重试
- 任意一次成功，执行记录标记为 success
- 3 次全部失败，执行记录标记为 failed
- 超时也正常重试，不做特殊区分

### 9.4 执行保护

平台侧单方面提供的执行保护能力，不要求被调用方改造。

#### 9.4.1 并发控制

任务级配置 `concurrency_policy`，控制同一任务并发执行时的行为：

| 策略 | 行为 | 适用场景 |
|------|------|----------|
| skip（默认） | 上一次未完成则跳过本轮，记录为 skipped | 大多数场景，安全优先 |
| queue | 排队等待，队列深度上限 5，超出则丢弃 | 每次触发都不能丢的场景 |
| allow | 不限制，允许并发执行 | 无状态接口，并发安全 |

- `max_concurrency`：最大并发执行数，默认 1
- 当在飞请求数 ≥ max_concurrency 时触发 concurrency_policy 逻辑
- queue 模式队列深度固定为 5，超出部分按 skip 处理
- 并发槽位通过 Redis INCR/DECR 管理，每次 Acquire 成功后刷新 TTL（兜底过期），确保进程崩溃后槽位自动释放，无需人工清理

#### 9.4.2 唯一请求 ID

每次 HTTP 触发增加 `X-Tick-Execution-ID` header（UUID），为被调用方做幂等提供便利。

#### 9.4.3 执行记录保留

- 任务级配置 `execution_retention_days`，默认 30 天，上限 90 天
- 平台定期清理过期记录

---

## 10. 时区处理

**策略**：所有时间统一存储 UTC，展示和输入支持本地时区。

| 场景 | 处理方式 |
|------|----------|
| Cron 表达式解析 | 按租户本地时区解析（默认 Asia/Shanghai），存储时转为 UTC |
| 一次性定时（at） | 输入为本地时间，存储为 UTC |
| API 输入/输出 | ISO 8601 格式，带时区信息（+08:00） |
| CLI 输出 | 本地时区（读取系统 TZ） |
| 内部调度 | 统一用 UTC，消除时区歧义 |

---

## 11. 幂等性设计

### 11.1 为什么需要幂等性

防止调度引擎 Bug、网络抖动、或 Asynq 任务重复消费导致同一个任务被触发多次。

### 11.2 设计

**方案：基于时间的去重窗口**

- **平台侧**：每次触发使用唯一 trigger_time 字符串，Asynq 开启 Unique 选项（同 ID + 同 trigger_time 不能重复入队）
- **目标侧**：通过签名验证来源合法性，防止伪造

**触发请求头**：

```
X-Tick-Task-ID: t_abc123
X-Tick-Timestamp: 1715731200
X-Tick-Signature: sha256(task_id + trigger_time + secret)
```

**目标服务验证**：

1. 从 `X-Tick-Task-ID` 取出 task_id
2. 从 `X-Tick-Timestamp` 取出触发时间（UTC秒）
3. 用同样算法计算期望签名，对比 `X-Tick-Signature`
4. 签名一致 + 时间在最近60s内 = 合法请求

---

## 12. 安全模型

| 能力 | 说明 |
|------|------|
| API Key 认证 | CLI/API 使用 Bearer Token，SHA-256 存储哈希，Redis 缓存 tenant 映射（TTL 5min） |
| 账密认证 | Dashboard 使用用户名/密码登录，密码 bcrypt 哈希存储，登录后发放 JWT（有效期 24h） |
| 传输加密 | 全站 HTTPS |
| 签名验证 | Webhook 带 X-Tick-Signature，目标可验签 |
| 限流 | 按租户维度限流（默认 100 req/min，可配） |
| 敏感数据 | 请求体不记录完整内容，最多 4KB |
| 凭证加密 | 凭证敏感字段 AES-256 加密存储，密钥通过环境变量注入 |
| IP 白名单 | V2 支持（任务级别配置允许的触发 IP 范围） |

---

## 13. 平台自身监控

谁监控调度引擎自己？

- **健康检查**：`GET /api/v1/status` 返回调度引擎状态
- **自监控**：定时任务检测时间轮 ticker 是否正常跳动，5s 内无跳动则告警
- **Redis 连接**：检测 Asynq 与 Redis 连接状态
- **PG 连接**：检测 PostgreSQL 连接池状态
- **告警**：调度引擎异常时通知管理员（飞书/邮件）

---

## 14. MVP 范围（V1）

#### 做

- [x] CLI 全部核心命令（task create/list/get/pause/resume/delete/history, auth, quota, status）
- [x] RESTful API（与 CLI 1:1 对应）
- [x] 三种调度类型：Cron + 固定间隔 + 一次性定时
- [x] 秒级调度精度
- [x] 同秒微打散 + Asynq Worker 池
- [x] HTTP Webhook 触发（GET/POST/PUT/DELETE）
- [x] Webhook 签名验证
- [x] 执行记录（最近 7 天）
- [x] 失败重试（3 次，指数退避）
- [x] 超时控制（可配）
- [x] 多租户（API Key + 配额）
- [x] 操作审计日志
- [x] CLI `--output json`（Agent/AI 友好）
- [x] CLI 一行安装（curl | bash）
- [x] 自助注册租户
- [x] 时区处理（UTC存储，本地展示）
- [x] 幂等性（Unique + 签名验证）
- [x] 只读 Dashboard（前端 go:embed 嵌入，任务列表/执行历史/概览）
- [x] Dashboard 认证（使用 API Key 登录，只显示当前租户数据，未登录返回登录页）
- [x] 凭证中心（Credential Center）：6 种凭证类型，AES-256 加密存储
- [x] Hook 引擎（Pre/Post Hook）：变量上下文传递，JSONPath 提取
- [x] Dashboard 凭证管理页面
- [x] Dashboard Hook 配置编辑器

#### 不做

- [ ] Agent 消息适配器（V2）
- [ ] 连续失败告警（V2）
- [ ] 熔断机制（V2）
- [ ] Prometheus 指标（V2）
- [ ] SDK（Python/Go/JS）（V2）
- [ ] IP 白名单（V2）
- [ ] DAG 工作流编排
- [ ] 执行环境/沙箱
- [ ] Web IDE

#### V2 当前迭代

- [ ] 多用户角色管理（用户独立注册、多租户归属、Owner/Member 权限、邀请码加入）
- [ ] 租户账密登录（用户名/密码注册、JWT session）
- [ ] 多 API Key 管理（创建/列表/吊销，带用途说明）
- [ ] Dashboard 白色主题
- [ ] Dashboard 任务完整 CRUD（创建/编辑/删除/暂停/恢复）
- [ ] Dashboard 任务手动触发（立即执行按钮，用于测试验证）
- [ ] Dashboard API Key 管理页面
- [ ] Dashboard bug 修复：task 详情页点击无响应
- [x] Dashboard 任务详情页优化：展示定时周期、合并卡片排版、分页标识、手动刷新按钮
- [ ] Dashboard UI 风格改造：从蓝色 SaaS 模板风格改为 Cursor 风格的极简克制设计
- [x] 凭证中心：统一管理鉴权凭证，绑定到 Target 后自动注入 HTTP 请求
- [ ] 变量模块：租户级全局变量，用于请求模板替换

---

## 7.7 凭证中心

### 7.7.1 定位

统一管理调用第三方 API 时的鉴权凭证。凭证绑定到 Target 后，每次执行时自动解析并注入到 HTTP 请求中（header/query/cookie），用户无需在每个任务中手动拼鉴权头。

### 7.7.2 凭证类型与默认注入规则

每种凭证类型内置默认的注入位置（location）、字段名（key）和前缀（prefix），用户创建时可覆盖：

| 类型 | 说明 | 默认 location | 默认 key | 默认 prefix |
|------|------|--------------|----------|-------------|
| bearer | 静态 Token | header | Authorization | `Bearer ` |
| basic | 用户名/密码 | header | Authorization | `Basic ` |
| oauth2_cc | OAuth2 Client Credentials | header | Authorization | `Bearer ` |
| dynamic | 动态获取 Token（调接口+提取） | header | Authorization | `Bearer ` |
| hmac | HMAC 签名 | header | X-Signature | (空) |
| custom_header | 自定义多个 Header | header | (N/A) | (N/A) |

- bearer：存储 token，注入时拼 prefix + token
- basic：存储 username + password，注入时 base64 编码后拼 prefix
- oauth2_cc：存储 token_url + client_id + client_secret + scope，运行时自动换取 access_token 并缓存
- dynamic：存储请求配置 + 提取路径，运行时调接口获取 token 并缓存
- hmac：存储 secret，注入签名值
- custom_header：存储 headers map，直接注入所有 k/v 对

### 7.7.3 凭证与 Target 绑定

- Target 的 config 中包含 `credential_ids` 数组，引用一个或多个凭证
- 执行时按数组顺序逐个解析注入
- 注入规则（location/key/prefix）由凭证自身配置决定，Target 无需关心
- 凭证被 Target 引用时不可删除（返回 409 Conflict）

#### 7.7.3.1 任务表单凭证绑定 UI

- 任务**创建**和**编辑**页面均须提供凭证选择控件，二者能力一致（多选下拉、加载/卸载、删除已选项）
- 凭证选择区域位于 HTTP 请求 URL 行下方，独立于"自定义 Headers / 请求体"等可折叠面板，默认可见
- 控件需阻止点击事件冒泡至 document 级 click 监听，避免下拉框被立即关闭
- 提交时将选中凭证 ID 列表写入 `credential_ids`，创建/编辑接口契约一致

### 7.7.4 凭证 Config 存储

凭证的完整配置（鉴权数据 + 注入规则）统一存储在 `config` 字段中，AES-256-GCM 加密。列表接口返回脱敏的 `config_preview`。

### 7.7.5 CLI 命令

```bash
tick credential create --type bearer --name "生产环境 Token" \
  --config '{"token":"eyJ..."}'                        # inject_* 不填则用默认值

tick credential list                                   # 列出凭证（config 脱敏）
tick credential get <id>                               # 查看详情
tick credential delete <id>                            # 删除（有引用时拒绝）
```

### 7.7.6 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/credentials` | 创建凭证 |
| GET | `/api/v1/credentials` | 列表（config 脱敏） |
| GET | `/api/v1/credentials/:id` | 获取详情 |
| GET | `/api/v1/credentials/:id/config` | 获取解密配置（敏感操作） |
| PUT | `/api/v1/credentials/:id` | 更新凭证 |
| PATCH | `/api/v1/credentials/:id/status` | 启用/禁用 |
| DELETE | `/api/v1/credentials/:id` | 删除（有引用时 409） |

---

## 7.8 变量模块

### 7.8.1 定位

租户级全局键值对，用于 Target 请求配置中的模板替换。适合存储环境相关的非敏感配置（如 API base URL、环境标识等）。敏感信息应使用凭证模块。

### 7.8.2 模板语法

沿用 `{{key}}` 语法。Target 的 URL、Headers、Body 中均可使用 `{{变量名}}` 引用全局变量，执行时自动替换。

示例：
- 变量：`api_base` = `https://api.production.internal`
- Target URL：`{{api_base}}/v1/sync`
- 执行时替换为：`https://api.production.internal/v1/sync`

### 7.8.3 存储

明文存储，不加密。变量不用于存储敏感信息。

### 7.8.4 内置变量

系统自动注入以下内置变量（不可覆盖）：

| 变量名 | 值 |
|--------|------|
| task_id | 当前任务 ID |
| task_name | 当前任务名称 |
| tenant_id | 当前租户 ID |
| trigger_time | 触发时间（RFC3339） |
| execution_id | 执行 ID |
| current_date | 当前日期（2006-01-02） |
| current_time | 当前时间（15:04:05） |
| current_datetime | 当前日期时间（2006-01-02 15:04:05） |
| current_timestamp | 当前 Unix 时间戳（秒） |

### 7.8.5 CLI 命令

```bash
tick variable set api_base "https://api.production.internal"  # 创建/更新变量
tick variable list                                             # 列出所有变量
tick variable get api_base                                     # 获取变量值
tick variable delete api_base                                  # 删除变量
```

### 7.8.6 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/variables` | 创建变量 |
| GET | `/api/v1/variables` | 列表 |
| PUT | `/api/v1/variables/:id` | 更新 |
| DELETE | `/api/v1/variables/:id` | 删除 |

---

## 15. 路线图

### Now — V1 MVP

| 项目 | 成功指标 | ETA |
|------|----------|-----|
| 调度引擎（时间轮 + 微打散） | 秒级精度，同秒 1000+ | Week 1-2 |
| Asynq 集成（Worker + 重试） | 重试按指数退避，Dead Letter 生效 | Week 1-2 |
| CLI + API | 全部核心命令可用 | Week 1-2 |
| 多租户 + 认证 | API Key 隔离，配额生效 | Week 2 |
| 执行记录 + 幂等性 | 执行记录可查，签名验证可用 | Week 2-3 |
| 端到端测试 | 5 个场景全部通过 | Week 3 |

### Next — V2

| 项目 | 假设 | 信心 |
|------|------|------|
| Agent 消息适配器 | Agent 用户需要比 webhook 更便捷的触发方式 | High |
| 告警通知 | 连续失败需要主动通知 | High |
| 熔断机制 | 目标不可用时需要自动暂停 | Medium |

### Later — V3

| 项目 | 战略假设 | 推进信号 |
|------|----------|----------|
| SDK（Python/Go/JS） | 多语言接入需求增长 | ≥ 3 个系统用 HTTP SDK 封装 |
| Webhook 回调验证 | 安全性需求提升 | 出现伪造回调问题 |
| IP 白名单 | 高安全场景需求 | 出现合规要求 |

---

## 16. 发布计划

| 阶段 | 日期 | 受众 | 通过标准 |
|------|------|------|----------|
| Internal alpha | Week 3 | 团队内部 | 核心流程完整，无 P0 Bug |
| Dogfood beta | Week 4 | 5 个内部系统 | <1% 错误率，P99 < 2s |
| Internal GA | Week 5 | 全公司 | P99 < 1s，同秒 1000+ 稳定 |

**回滚标准**：触发成功率 < 99% 或 P99 延迟 > 5s，暂停新任务创建并通知管理员。

---

## 16.1 前端设计规范

> 详见 [Dashboard UI 设计规范](./dashboard-design.md)（独立文档）

---

## 17. 技术选型

| 层 | 选型 | 理由 |
|----|------|------|
| 语言 | Go | goroutine 天然适合高并发调度；CLI 分发简单（单二进制） |
| 调度引擎 | 自建分层时间轮 | 秒级精度 + 微打散，核心逻辑 ~400 行 |
| 执行引擎 | Asynq | 队列 + Worker 池 + 重试 + 超时 + 去重 + Dead Letter，直接复用 |
| 持久化 | PostgreSQL | 任务 + 执行记录 + 租户 + 审计，一套搞定 |
| 缓存/队列 | Redis | 调度索引 + Asynq 队列 |
| API | RESTful + JSON | 简单直接，SDK 好写 |

---

## 18. 参考项目

| 项目 | Stars | 参考什么 |
|------|-------|----------|
| Asynq | 9k+ | 执行层核心：Redis 数据结构、Worker 模型、重试策略 |
| Dkron | 4k+ | 平台化思路、CLI 设计、分布式调度架构 |
| XXL-JOB | 27k+ | 权限模型、阻塞处理策略、告警设计（不参考架构） |
| Temporal | 12k+ | CLI 设计、Namespace 隔离（不参考编排） |

---

## 19. 已知风险与缓解

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| 同秒高并发下 Redis 压力 | Medium | High | 微打散 + Pipeline 批量入队 |
| 时间轮重启后状态恢复 | Low | High | PG 持久化 + 启动时重加载 |
| 目标服务不可用导致重试风暴 | Medium | Medium | 指数退避 + 熔断（V2） |
| 单租户占用过多 Worker | Medium | Medium | 租户配额 + 优先级队列 |
| API Key 泄露 | Low | High | 哈希存储 + 支持 Key 轮换 |

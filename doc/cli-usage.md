# Tick CLI 使用指南

Tick CLI 把定时触发平台能力对开发者与 AI 开放，按"四节点"组织：**安装 → 配置 → 使用 → 更新**。

---

## 1. 安装

### 1.1 从 Git 源码构建（推荐 dev 同学）

```bash
git clone https://github.com/tickplatform/tick.git
cd tick/code
make build-dev         # 产出 bin/tick，同时写 .tick-source marker
sudo mv bin/tick /usr/local/bin/
tick --version
```

`make build-dev` 会注入 `SourcePath`，使 `tick update` 自动走 from-git 模式。

### 1.2 跨平台产物

```bash
make build-cross       # 产出 5 个平台二进制到 bin/
```

`darwin/amd64`、`darwin/arm64`、`linux/amd64`、`linux/arm64`、`windows/amd64`，全部 `CGO_ENABLED=0`。

### 1.3 install.sh（标准 release 安装）

```bash
curl -fsSL https://raw.githubusercontent.com/tickplatform/tick/main/code/install.sh | sh
```

支持环境变量：

- `TICK_VERSION`：指定版本（默认 `latest`）
- `TICK_INSTALL_URL`：自定义下载源（默认 GitHub Releases）
- `TICK_INSTALL_DIR`：安装目录（默认 `/usr/local/bin`）

脚本会下载 `.sha256` 旁路文件做完整性校验，校验失败时**不覆盖**旧二进制。

### 1.4 SIT 独立二进制

```bash
make build-sit         # 产出 bin/tick-sit（BuiltForEnv=sit）
```

`bin/tick`（prod）和 `bin/tick-sit`（SIT）可并存于同一台机器，配置文件按 env 维度隔离。

---

## 2. 配置（登录）

### 2.1 首次登录

```bash
tick auth login --api-key tk_xxx
```

URL 来自编译期常量，**无需手动输入**。提示示例：

```
当前二进制连的是 https://tick\.example\.com (prod)，请输入 API key: tk_xxx
登录成功 (env=prod)
```

配置文件写到 `~/.tick/config.yaml`：

```yaml
api_keys:
  prod: tk_xxx
output: json   # 可选
```

权限自动设为 `0o600`。

### 2.2 切换环境

不是运行时切换，而是切换二进制：

```bash
# 登录 SIT（前提是当前是 tick-sit 二进制）
./bin/tick-sit auth login --api-key tk_sit_yyy
# 此时 ~/.tick/config.yaml 同时有 prod 和 sit 两组 key
```

### 2.3 临时覆盖 key

```bash
TICK_API_KEY=tk_ci ./bin/tick task list
```

环境变量**优先级最高**，但**不会写回**配置文件。

### 2.4 登出

```bash
tick auth logout        # 交互式确认
tick auth logout --yes  # 跳过确认
```

只清除当前 `BuiltForEnv` 对应槽位，其他 env 的 key 保留。

### 2.5 旧配置自动迁移

如果你的 `~/.tick/config.yaml` 来自旧版本（含 `server_url` / `token` / `current_env` 字段）：

- `server_url` → 丢弃（URL 现已硬编码进二进制）
- `token` → 自动迁移到 `api_keys.default`
- `current_env` → 丢弃（env 现由 `BuiltForEnv` 编译期决定）
- `output` → 保留

迁移在首次加载时自动执行（一次性）。迁移后原字段消失，新结构生效。

---

## 3. 日常命令

### 3.1 身份与连接信息

```bash
tick auth whoami
# tenant:     acme-corp
# user:       ops-team
# api_key:    tk_live_...f9d508
# server:     https://tick\.example\.com (prod)
# built:      v0.2.0 (commit a1b2c3d, 2026-06-01)

tick auth whoami --output json
```

### 3.2 任务（task）

```bash
# 列出
tick task list
tick task list --status active
tick task list --output json

# 创建
tick task create --name "日报" --cron "0 8 * * *" --url https://example.com/hook
tick task create --name "心跳" --every 30s --url https://example.com/heartbeat --method GET
tick task create --name "一次性" --at 2026-06-02T20:00:00Z --url https://example.com/hook

# 详情 / 启停 / 删除
tick task get t_abc123
tick task pause t_abc123
tick task resume t_abc123
tick task delete t_abc123 --yes

# 历史
tick task history t_abc123 --limit 50
```

### 3.3 目标（target）

```bash
tick target list
tick target create --name "webhook-a" --type http --url https://example.com/hook
tick target delete tgt_xxx --yes
```

### 3.4 凭证（credential）

```bash
tick credential list
tick credential create --name "prod-api" --type bearer --config '{"token":"tk_xxx"}'
tick credential delete cred_xxx --yes
```

### 3.5 签名密钥（secret）

```bash
tick secret list
tick secret create
tick secret revoke sec_xxx --yes
```

### 3.6 通用

```bash
--output json|table    # 全局输出格式
--yes / -y             # 跳过危险操作确认
--config <path>        # 指定配置文件（覆盖默认 ~/.tick/config.yaml）
```

---

## 4. 自更新

### 4.1 检查

```bash
tick update --check
# 当前版本: v0.2.0
# 最新版本: v0.3.0
# 模式:     release
# 需要更新: 是
```

### 4.2 自动模式升级

```bash
tick update
```

自动探测安装方式：

- 二进制同目录存在 `.tick-source` → **from-git 模式**（dev build）
- 不存在 → **release 模式**（标准 install）

### 4.3 显式指定模式

```bash
tick update --from-release   # 强制从 GitHub Releases 下载
tick update --from-git       # 强制从本地 git 仓库（重建）
tick update --from-go        # 强制通过 go install
```

### 4.4 指定版本

```bash
tick update v0.3.0
```

### 4.5 失败回滚

升级过程中任一步失败（网络错、SHA256 不匹配、git pull 冲突），**原二进制保留**。可继续使用旧版本，重新执行 `tick update` 重试。

### 4.6 权限问题

`/usr/local/bin/tick` 不可写时：

```
需要 sudo 权限，请用 sudo env HOME=$HOME tick update
```

按提示执行即可。注意用 `env HOME=$HOME` 避免 `$HOME` 被替换为 `/root` 导致读不到 `~/.tick/config.yaml`。

---

## 5. 故障排查

| 现象 | 原因 | 解决 |
|------|------|------|
| `tick task list: 401 Unauthorized` | API key 无效或过期 | 重新 `tick auth login` |
| `配置文件损坏（YAML 解析失败）` | 手动改坏了 yaml | `rm ~/.tick/config.yaml` 后重新 `tick auth login` |
| `当前二进制不支持 SIT` | 用 prod 二进制加了 `--env sit` | 用 `make build-sit` 重新构建 |
| `tick update` 卡在 release 模式 | `.tick-source` 被误删 | 用 `tick update --from-release` 显式指定 |
| GitHub 不可达（境内） | 网络问题 | 用 dev build + `tick update --from-git` |

---

## 6. 注意事项

- **prod 服务器只能装 prod 二进制**：SIT 二进制是本地 dev 工具，不入制品库分发。
- **配置文件权限始终 `0o600`**：写盘时强制，CI 中显式断言（SC-006）。
- **不要在配置文件中放 URL**：URL 是编译期常量，运行时不可覆盖（设计意图）。
- **升级后 `tick --version` 不变** 说明：原二进制被破坏，请重新 `make build-dev` 或 install.sh。

# 配置说明

> 🌐 **Language / 语言**: [English](../enUS/CONFIGURATION.md) | [中文](CONFIGURATION.md) | [Français](../frFR/CONFIGURATION.md) | [Italiano](../itIT/CONFIGURATION.md) | [日本語](../jaJP/CONFIGURATION.md) | [Deutsch](../deDE/CONFIGURATION.md) | [한국어](../koKR/CONFIGURATION.md)

本文档详细说明 Warden 的配置选项，包括运行模式、配置文件格式、环境变量等。配置来源优先级：**命令行参数 > 环境变量 > 配置文件 (YAML) > 默认值**。

## 配置项总览

以下为当前代码实际支持的配置项与默认值（参见 `internal/config/config.go`、`internal/define/define.go`、`internal/cmd/cmd.go`）：

| 分类 | 配置项 | YAML 路径 | 环境变量 | 默认值 |
|------|--------|-----------|----------|--------|
| 服务 | 端口 | `server.port` | `PORT` | `8081` |
| 服务 | 读超时 | `server.read_timeout` | — | `5s` |
| 服务 | 写超时 | `server.write_timeout` | — | `5s` |
| 服务 | 关闭超时 | `server.shutdown_timeout` | — | `5s` |
| 服务 | 空闲超时 | `server.idle_timeout` | — | `120s` |
| 服务 | 最大请求头 | `server.max_header_bytes` | — | `1048576` (1MB) |
| Redis | 地址 | `redis.addr` | `REDIS` | `localhost:6379` |
| Redis | 密码 | `redis.password` | `REDIS_PASSWORD` | 空 |
| Redis | 密码文件 | `redis.password_file` | `REDIS_PASSWORD_FILE` | 空 |
| Redis | DB 索引 | `redis.db` | — | `0` |
| Redis | 是否启用 | — | `REDIS_ENABLED` | `true`（仅 `ONLY_LOCAL` 且未设 REDIS 时为 `false`） |
| 缓存 | TTL | `cache.ttl` | — | 无（仅 `update_interval` 有默认） |
| 缓存 | 更新间隔 | `cache.update_interval` | — | `5s` |
| 限流 | 速率 | `rate_limit.rate` | — | `60`（次/分钟） |
| 限流 | 窗口 | `rate_limit.window` | — | `1m` |
| HTTP 客户端 | 超时 | `http.timeout` | `HTTP_TIMEOUT` | `5s` |
| HTTP 客户端 | 最大空闲连接 | `http.max_idle_conns` | `HTTP_MAX_IDLE_CONNS` | `100` |
| HTTP 客户端 | 跳过 TLS 验证 | `http.insecure_tls` | `HTTP_INSECURE_TLS` | `false` |
| HTTP 客户端 | 最大重试 | `http.max_retries` | — | `3` |
| HTTP 客户端 | 重试延迟 | `http.retry_delay` | — | `1s` |
| 远程 | URL | `remote.url` | `CONFIG` | `http://localhost:8080/data.json` |
| 远程 | 认证 Key | `remote.key` | `KEY` | 空 |
| 远程 | 模式 | `remote.mode` / `app.mode` | `MODE` | `DEFAULT` |
| 远程 | 解密启用 | `remote.decrypt_enabled` | `REMOTE_DECRYPT_ENABLED` | `false` |
| 远程 | RSA 私钥文件 | `remote.rsa_private_key_file` | `REMOTE_RSA_PRIVATE_KEY_FILE` | 空（推荐用文件） |
| 远程 | RSA 私钥 PEM（内联） | — | `REMOTE_RSA_PRIVATE_KEY` | 空（当未设置文件时可用，多行 PEM 需保留换行） |
| 任务 | 间隔 | `task.interval` | `INTERVAL` | `5`（秒） |
| 应用 | 模式 | `app.mode` | `MODE` | `DEFAULT` |
| 应用 | API Key | `app.api_key` | `API_KEY` | 空 |
| 应用 | 本地数据文件 | `app.data_file` | `DATA_FILE` | `./data.json` |
| 应用 | 本地数据目录 | `app.data_dir` | `DATA_DIR` | 空（与 data_file 可同时使用，合并目录下所有 \*.json） |
| 应用 | 响应字段白名单 | `app.response_fields` | `RESPONSE_FIELDS` | 空（空=返回全部字段；逗号分隔如 `phone,mail,user_id,status,name`） |
| 追踪 | 启用 | `tracing.enabled` | `OTLP_ENABLED` | `false` |
| 追踪 | OTLP 端点 | `tracing.endpoint` | `OTLP_ENDPOINT` | 空 |
| 其他 | 配置文件路径 | — | `CONFIG_FILE` | 空（用于仅通过环境变量指定 YAML 时加载 tracing 等） |
| 其他 | 信任代理 IP | — | `TRUSTED_PROXY_IPS` | 空 |
| 其他 | 健康检查 IP 白名单 | — | `HEALTH_CHECK_IP_WHITELIST` | 空 |
| 其他 | 全局 IP 白名单 | — | `IP_WHITELIST` | 空 |
| 其他 | 日志级别 | — | `LOG_LEVEL` | `info` |
| 服务间鉴权 | HMAC 密钥（JSON） | — | `WARDEN_HMAC_KEYS` | 空 |
| 服务间鉴权 | HMAC 时间戳容差（秒） | — | `WARDEN_HMAC_TIMESTAMP_TOLERANCE` | `60`（仅当配置了 HMAC 时） |
| 服务间鉴权 | TLS 服务端证书路径 | — | `WARDEN_TLS_CERT` | 空 |
| 服务间鉴权 | TLS 服务端私钥路径 | — | `WARDEN_TLS_KEY` | 空 |
| 服务间鉴权 | 客户端 CA 路径（mTLS） | — | `WARDEN_TLS_CA` | 空 |
| 服务间鉴权 | 是否要求客户端证书 | — | `WARDEN_TLS_REQUIRE_CLIENT_CERT` | `false` |

说明：
- 使用 `--config-file` 时，主配置（端口、Redis、远程、任务等）从 YAML 加载，环境变量仍可覆盖。**追踪（tracing）**：主程序在初始化追踪时仅读取 `CONFIG_FILE` 环境变量或 `OTLP_ENABLED`/`OTLP_ENDPOINT`，不会自动使用 `--config-file` 指向的 YAML 中的 `tracing` 段。若希望从 YAML 启用追踪，请同时设置 `CONFIG_FILE` 指向同一配置文件，或直接使用环境变量 `OTLP_ENABLED` + `OTLP_ENDPOINT`。
- 未使用 `--config-file` 时，可通过 `CONFIG_FILE` 指定 YAML 路径，用于从文件读取 `tracing` 等（主配置仍来自命令行/环境变量）。
- 任务间隔：使用配置文件时，`task.interval` 来自 YAML 与默认值，环境变量 `INTERVAL` 不参与覆盖；仅在不使用配置文件时 `INTERVAL` 生效。

## 配置校验规则

启动时 `cmd.ValidateConfig` 会校验以下项（参见 `internal/cmd/validate.go`），不通过则退出：

| 项 | 规则 |
|----|------|
| 端口 | 合法端口格式（cli-kit `ValidatePortString`） |
| Redis | 非空时为 `host:port` 格式（cli-kit `ValidateHostPort`） |
| 远程 URL | 非默认且非空时须为合法 URL（SSRF 校验） |
| 任务间隔 | `TaskInterval >= 1`（秒） |
| 模式 | 须为枚举之一：`DEFAULT`、`REMOTE_FIRST`、`ONLY_REMOTE`、`ONLY_LOCAL`、`LOCAL_FIRST`、`REMOTE_FIRST_ALLOW_REMOTE_FAILED`、`LOCAL_FIRST_ALLOW_REMOTE_FAILED` |
| 生产 + TLS | `app.mode` 为 `production`/`prod` 时禁止 `http.insecure_tls: true`（config 层校验） |
| DATA_DIR | 非空时须存在且为目录 |
| 远程解密 | `REMOTE_DECRYPT_ENABLED=true` 时须配置 `REMOTE_RSA_PRIVATE_KEY_FILE`（且文件存在可读）或 `REMOTE_RSA_PRIVATE_KEY` 之一 |

## 运行模式 (MODE)

系统支持 6 种数据合并模式，根据 `MODE` 参数选择：

| 模式 | 说明 | 使用场景 |
|------|------|----------|
| `DEFAULT` / `REMOTE_FIRST` | 远程优先，远程数据不存在时使用本地数据补充 | 默认模式，适合大多数场景 |
| `ONLY_REMOTE` | 仅使用远程数据源 | 完全依赖远程配置 |
| `ONLY_LOCAL` | 仅使用本地配置文件，**默认禁用 Redis**（如果显式设置了 `REDIS` 地址或 `REDIS_ENABLED=true`，则会启用 Redis） | 离线环境或测试环境 |
| `LOCAL_FIRST` | 本地优先，本地数据不存在时使用远程数据补充 | 本地配置为主，远程为辅 |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | 远程优先，允许远程失败时回退到本地 | 高可用场景 |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | 本地优先，允许远程失败时回退到本地 | 混合模式 |

### `REDIS_ENABLED` 与 `MODE=ONLY_LOCAL` 交互规则

当前实现中，`ONLY_LOCAL` 不等同于“强制关闭 Redis”，而是“默认关闭 Redis”：

1. 当 `MODE=ONLY_LOCAL` 且未显式设置 `REDIS_ENABLED` 与 `REDIS` 地址时，`REDIS_ENABLED` 默认视为 `false`。
2. 当 `MODE=ONLY_LOCAL` 但显式设置了 `REDIS_ENABLED=true`，则会启用 Redis。
3. 当 `MODE=ONLY_LOCAL` 且未设置 `REDIS_ENABLED`，但显式设置了 `REDIS=<host:port>`，则会启用 Redis。

实践建议：

- **离线/轻量测试**：`MODE=ONLY_LOCAL` + `REDIS_ENABLED=false`
- **本地数据但希望缓存更稳**：`MODE=ONLY_LOCAL` + `REDIS_ENABLED=true` + `REDIS=...`
- **生产环境**：建议启用 Redis，并显式设置 `REDIS_ENABLED=true`，避免依赖隐式推导

### 配置方式

可以通过以下方式设置运行模式：

**命令行参数**:
```bash
go run . --mode DEFAULT
```

**环境变量**:
```bash
export MODE=DEFAULT
```

**配置文件**:
```yaml
remote:
  mode: "DEFAULT"
# 或
app:
  mode: "DEFAULT"
```

## 配置文件格式

### 本地用户数据文件路径（DATA_FILE）

本地用户数据文件路径可通过以下方式配置，默认值为 `./data.json`：

**命令行参数**：
```bash
go run . --data-file /path/to/data.json
```

**环境变量**：
```bash
export DATA_FILE=/path/to/data.json
```

**配置文件**（YAML）：
```yaml
app:
  data_file: "/path/to/data.json"
```

### 本地用户数据文件格式

本地用户数据文件（默认 `data.json`）格式（可参考 `data.example.json`）：

**最小格式**（仅必需字段）：
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**完整格式**（包含所有可选字段）：
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write", "admin"],
        "role": "admin"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com",
        "status": "active",
        "scope": ["read"],
        "role": "user"
    }
]
```

**字段说明**：
- `phone`（与 mail 二选一或同时提供）：用户手机号；支持仅邮箱用户（phone 可为空）
- `mail`（与 phone 二选一或同时提供）：用户邮箱地址；支持仅邮箱用户（mail 可为空）
- `user_id`（可选）：用户唯一标识符，如果未提供则基于 phone 或 mail 自动生成
- `status`（可选）：用户状态，默认为 "active"
- `scope`（可选）：用户权限范围数组，默认为空数组
- `role`（可选）：用户角色，默认为空字符串
- `name`（可选）：用户显示名称，会出现在 API 响应及 `/v1/lookup` 中

### 本地用户数据目录（DATA_DIR）

除单文件 `DATA_FILE` 外，可配置 `DATA_DIR` 指定一个目录，程序会将该目录下所有 `*.json` 文件（按文件名排序）与单文件一起参与加载与合并。

**与 DATA_FILE 同时存在时的行为**：
- 数据源顺序由运行模式（MODE）决定：REMOTE_FIRST 时为先远程（若配置）、再目录内文件（按文件名排序）、再单文件；LOCAL_FIRST 时为先目录内文件与单文件、再远程。
- 同一用户（按 phone/mail 去重）在多文件中出现时，按上述优先级合并，高优先级覆盖低优先级。

**配置方式**：
- 环境变量：`export DATA_DIR=/path/to/data/dir`
- YAML：`app.data_dir: "/path/to/data/dir"`

### API 响应字段白名单（RESPONSE_FIELDS）

未配置时，`GET /` 与 `GET /user` 返回用户对象的全部字段。配置后仅返回白名单中的字段（适用于减少泄露或统一出口格式）。

- 环境变量：`export RESPONSE_FIELDS=phone,mail,user_id,status,scope,role,name`
- YAML：`app.response_fields: ["phone","mail","user_id","status","scope","role","name"]`
- Stargate 所需字段建议包含：`user_id`、`mail`/`phone`、`status`、`scope`、`role`。

### 远程响应 RSA 解密

当远程 API 返回加密 body 时，可启用 RSA 解密后再解析 JSON。

- **配置**：`REMOTE_DECRYPT_ENABLED=true`，并设置 `REMOTE_RSA_PRIVATE_KEY_FILE=/path/to/private.pem`（推荐）或 `REMOTE_RSA_PRIVATE_KEY`（内联 PEM，适用于密钥管理服务注入）。
- **约定**：远程响应需设置 `Content-Type: application/x-warden-encrypted`，body 为 Base64 编码的混合密文：前 256 字节为 RSA(2048) 加密的 AES 密钥+IV，其后为 AES-CTR 密文；解密后为 JSON 数组格式，与本地 `data.json` 结构一致。
- 启用解密时，远程拉取与解密在 Warden 内完成，再与本地文件源按 MODE 合并。

### 应用配置文件 (`config.yaml`)

支持使用 YAML 格式的配置文件，通过 `--config-file` 参数指定：

```yaml
server:
  port: "8081"
  read_timeout: 5s
  write_timeout: 5s
  shutdown_timeout: 5s
  max_header_bytes: 1048576  # 1MB
  idle_timeout: 120s

redis:
  addr: "localhost:6379"
  password: ""  # 建议使用环境变量 REDIS_PASSWORD 或 REDIS_PASSWORD_FILE
  password_file: ""  # 密码文件路径（优先级高于 password）
  db: 0

cache:
  ttl: 3600s
  update_interval: 5s

rate_limit:
  rate: 60  # 每分钟请求数
  window: 1m

http:
  timeout: 5s
  max_idle_conns: 100
  insecure_tls: false  # 仅用于开发环境
  max_retries: 3
  retry_delay: 1s

remote:
  url: "http://localhost:8080/data.json"
  key: ""
  mode: "DEFAULT"
  decrypt_enabled: false       # 远程响应 RSA 解密
  rsa_private_key_file: ""    # PEM 私钥文件路径

task:
  interval: 5s

app:
  mode: "DEFAULT"  # 可选值: DEFAULT, production, prod
  api_key: ""      # 建议使用环境变量 API_KEY
  data_file: "./data.json"
  data_dir: ""     # 可选：用户数据目录，合并该目录下所有 *.json
  response_fields: []  # 可选：API 响应字段白名单，空则返回全部

tracing:
  enabled: false
  endpoint: ""     # 例如 "http://localhost:4318"
```

**配置优先级**：命令行参数 > 环境变量 > 配置文件 > 默认值

参考示例文件：[config.example.yaml](../../config.example.yaml)

## 命令行参数

```bash
go run . \
  --port 8081 \                    # Web 服务端口 (默认: 8081)
  --redis localhost:6379 \         # Redis 地址 (默认: localhost:6379)
  --redis-password "password" \    # Redis 密码（可选，建议使用环境变量）
  --redis-enabled=true \           # 启用/禁用 Redis（默认: true）
  --config http://example.com/api \ # 远程配置 URL
  --key "Bearer token" \           # 远程配置认证头
  --interval 5 \                   # 定时任务间隔（秒，默认: 5）
  --mode DEFAULT \                 # 运行模式（见上方说明）
  --http-timeout 5 \               # HTTP 请求超时时间（秒，默认: 5）
  --http-max-idle-conns 100 \     # HTTP 最大空闲连接数 (默认: 100)
  --http-insecure-tls \           # 跳过 TLS 证书验证（仅用于开发环境）
  --api-key "your-secret-api-key" \ # API Key 用于认证（可选，建议使用环境变量）
  --data-file ./data.json \        # 本地用户数据文件路径（默认: ./data.json）
  --config-file config.yaml        # 配置文件路径（支持 YAML 格式）
```

**注意**：
- 配置文件支持：可以使用 `--config-file` 参数指定 YAML 格式的配置文件
- Redis 密码安全：建议使用环境变量 `REDIS_PASSWORD` 或 `REDIS_PASSWORD_FILE` 而不是命令行参数
- TLS 证书验证：`--http-insecure-tls` 仅用于开发环境，生产环境不应使用

## 环境变量

支持通过环境变量配置，优先级低于命令行参数：

```bash
export PORT=8081
export REDIS=localhost:6379
export REDIS_PASSWORD="password"        # Redis 密码（可选）
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Redis 密码文件路径（可选；整体优先级：REDIS_PASSWORD > REDIS_PASSWORD_FILE > 配置文件）
export REDIS_ENABLED=true               # 启用/禁用 Redis（可选，默认: true，支持 true/false/1/0）
                                        # 注意: 在 ONLY_LOCAL 模式下，默认值为 false
                                        #       但如果显式设置了 REDIS 地址，则会自动启用 Redis
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5                      # 定时任务间隔（秒）；仅在不使用配置文件时生效（使用 YAML 时 task.interval 来自文件，INTERVAL 不参与覆盖）
export MODE=DEFAULT
export DATA_FILE=./data.json          # 本地用户数据文件路径
export DATA_DIR=                      # 可选：用户数据目录，合并该目录下所有 *.json（可与 DATA_FILE 同时使用）
export RESPONSE_FIELDS=               # 可选：API 响应字段白名单，逗号分隔，如 phone,mail,user_id,status,name；空=全部
export REMOTE_DECRYPT_ENABLED=false   # 可选：是否对远程响应做 RSA 解密
export REMOTE_RSA_PRIVATE_KEY_FILE=   # 可选：RSA 私钥 PEM 文件路径（与 REMOTE_RSA_PRIVATE_KEY 二选一，文件优先）
export REMOTE_RSA_PRIVATE_KEY=        # 可选：RSA 私钥 PEM 内联（多行需保留换行；未设置 FILE 时使用）
export HTTP_TIMEOUT=5                  # HTTP 请求超时（支持整数秒或 duration，如 30s、1m30s）
export HTTP_MAX_IDLE_CONNS=100         # HTTP 最大空闲连接数
export HTTP_INSECURE_TLS=false         # 是否跳过 TLS 证书验证（true/false 或 1/0）
export API_KEY="your-secret-api-key"   # API Key 用于认证（强烈建议设置）
export CONFIG_FILE=config.yaml         # 配置文件路径（可选；用于在不使用 --config-file 时仍从 YAML 加载 tracing 等）
export OTLP_ENABLED=false              # 是否启用 OpenTelemetry（true/false 或 1/0）
export OTLP_ENDPOINT=http://localhost:4318  # OTLP 端点，OTLP_ENABLED 为 true 时必填
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # 信任的代理 IP 列表（逗号分隔）
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # 健康检查端点 IP 白名单（可选）
export IP_WHITELIST="192.168.1.0/24"  # 全局 IP 白名单（可选）
export LOG_LEVEL="info"                # 日志级别（可选，默认: info，可选值: trace, debug, info, warn, error, fatal, panic）
export WARDEN_HMAC_KEYS='{"key-id-1":"secret-1"}'  # 服务间 HMAC 鉴权密钥（JSON，可选）
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60         # HMAC 时间戳容差（秒），默认 60
export WARDEN_TLS_CERT=/path/to/warden.crt        # 服务端 TLS 证书（可选，与 KEY 同时设置则启用 TLS）
export WARDEN_TLS_KEY=/path/to/warden.key         # 服务端 TLS 私钥
export WARDEN_TLS_CA=/path/to/ca.crt              # 客户端 CA（mTLS 时验证客户端证书）
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true        # 是否强制要求客户端证书（mTLS）
```

**环境变量优先级**：
- Redis 密码：`REDIS_PASSWORD` > `REDIS_PASSWORD_FILE` > 命令行参数 `--redis-password`

**安全配置说明**：
- `API_KEY`: 用于保护敏感端点（`/`、`/log/level`），强烈建议在生产环境设置
- `TRUSTED_PROXY_IPS`: 配置信任的反向代理 IP，用于正确获取客户端真实 IP
- `HEALTH_CHECK_IP_WHITELIST`: 限制健康检查端点的访问 IP（可选，支持 CIDR 网段）
- `IP_WHITELIST`: 全局 IP 白名单（可选，支持 CIDR 网段）

## 远程配置 API 要求

远程配置 API 应返回相同格式的 JSON 数组，支持可选的 Authorization 头认证。

API 响应格式应与 `data.json` 文件格式一致：

```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write"],
        "role": "admin"
    }
]
```

如果配置了 `KEY` 环境变量或 `--key` 参数，请求时会自动添加 `Authorization` 头：

```http
Authorization: Bearer your-token-here
```

## 与 Stargate 集成（调用方配置）

Warden 支持 **mTLS、HMAC、API Key** 三种服务间鉴权（优先级 mTLS > HMAC > API Key）。若由 Stargate 调用 Warden，请在 **Stargate 侧**配置 Warden 服务地址与认证方式：

**Stargate 配置示例**（环境变量）：
```bash
# Warden 服务地址（启用 TLS 时使用 https）
export STARGATE_WARDEN_BASE_URL=http://warden:8081

# 认证方式一：API Key（与 Warden API_KEY 一致）
# 认证方式二：HMAC（与 Warden WARDEN_HMAC_KEYS 中某 key 的 secret 一致，请求时带 X-Signature、X-Timestamp、X-Key-Id）
# 认证方式三：mTLS（Stargate 配置客户端证书，Warden 配置 WARDEN_TLS_*）
```

## 详细配置说明

更多关于参数解析机制、优先级规则和使用示例的详细信息，请参考：

- [参数解析设计文档](CONFIG_PARSING.md) - 详细的参数解析机制说明
- [架构设计文档](ARCHITECTURE.md) - 了解整体架构和配置影响
- [安全文档](SECURITY.md) - 了解服务间鉴权详细说明

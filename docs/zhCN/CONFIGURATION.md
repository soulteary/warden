# 配置说明

> 🌐 **Language / 语言**: [English](../enUS/CONFIGURATION.md) | [中文](CONFIGURATION.md)

本文档详细说明 Warden 的配置选项，包括运行模式、配置文件格式、环境变量等。

## 运行模式 (MODE)

系统支持 6 种数据合并模式，根据 `MODE` 参数选择：

| 模式 | 说明 | 使用场景 |
|------|------|----------|
| `DEFAULT` / `REMOTE_FIRST` | 远程优先，远程数据不存在时使用本地数据补充 | 默认模式，适合大多数场景 |
| `ONLY_REMOTE` | 仅使用远程数据源 | 完全依赖远程配置 |
| `ONLY_LOCAL` | 仅使用本地配置文件，**默认禁用 Redis**（可通过 `REDIS_ENABLED=true` 显式启用） | 离线环境或测试环境 |
| `LOCAL_FIRST` | 本地优先，本地数据不存在时使用远程数据补充 | 本地配置为主，远程为辅 |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | 远程优先，允许远程失败时回退到本地 | 高可用场景 |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | 本地优先，允许远程失败时回退到本地 | 混合模式 |

### 配置方式

可以通过以下方式设置运行模式：

**命令行参数**:
```bash
go run main.go --mode DEFAULT
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

### 本地用户数据文件 (`data.json`)

本地用户数据文件 `data.json` 格式（可参考 `data.example.json`）：

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
- `phone`（必需）：用户手机号
- `mail`（必需）：用户邮箱地址
- `user_id`（可选）：用户唯一标识符，如果未提供则基于 phone 或 mail 自动生成
- `status`（可选）：用户状态，默认为 "active"
- `scope`（可选）：用户权限范围数组，默认为空数组
- `role`（可选）：用户角色，默认为空字符串

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

task:
  interval: 5s

app:
  mode: "DEFAULT"  # 可选值: DEFAULT, production, prod
```

**配置优先级**：命令行参数 > 环境变量 > 配置文件 > 默认值

参考示例文件：[config.example.yaml](../config.example.yaml)

## 命令行参数

```bash
go run main.go \
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
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Redis 密码文件路径（可选，优先级高于 REDIS_PASSWORD）
export REDIS_ENABLED=true               # 启用/禁用 Redis（可选，默认: true，支持 true/false/1/0）
                                        # 注意: 在 ONLY_LOCAL 模式下，默认值为 false（除非显式设置）
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5
export MODE=DEFAULT
export HTTP_TIMEOUT=5                  # HTTP 请求超时时间（秒）
export HTTP_MAX_IDLE_CONNS=100         # HTTP 最大空闲连接数
export HTTP_INSECURE_TLS=false         # 是否跳过 TLS 证书验证（true/false 或 1/0）
export API_KEY="your-secret-api-key"   # API Key 用于认证（强烈建议设置）
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # 信任的代理 IP 列表（逗号分隔）
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # 健康检查端点 IP 白名单（可选）
export IP_WHITELIST="192.168.1.0/24"  # 全局 IP 白名单（可选）
export LOG_LEVEL="info"                # 日志级别（可选，默认: info，可选值: trace, debug, info, warn, error, fatal, panic）
```

**环境变量优先级**：
- Redis 密码：`REDIS_PASSWORD_FILE` > `REDIS_PASSWORD` > 命令行参数 `--redis-password`

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

## 详细配置说明

更多关于参数解析机制、优先级规则和使用示例的详细信息，请参考：

- [参数解析设计文档](CONFIG_PARSING.md) - 详细的参数解析机制说明
- [架构设计文档](ARCHITECTURE.md) - 了解整体架构和配置影响

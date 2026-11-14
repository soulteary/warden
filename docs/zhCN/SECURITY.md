# 安全文档

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

本文档说明 Warden 的安全特性、安全配置和最佳实践。

## 已实现的安全功能

1. **API 认证**: 支持 API Key 认证，保护敏感端点
2. **SSRF 防护**: 严格验证远程配置 URL，防止服务器端请求伪造攻击
3. **输入验证**: 严格验证所有输入参数，防止注入攻击
4. **速率限制**: 基于 IP 的速率限制，防止 DDoS 攻击
5. **TLS 验证**: 生产环境强制启用 TLS 证书验证
6. **错误处理**: 生产环境隐藏详细错误信息，防止信息泄露
7. **安全响应头**: 自动添加安全相关的 HTTP 响应头
8. **IP 白名单**: 支持为健康检查端点配置 IP 白名单
9. **配置文件验证**: 防止路径遍历攻击
10. **JSON 大小限制**: 限制 JSON 响应体大小，防止内存耗尽攻击
11. **用户查询参数长度限制**: `phone`/`mail`/`user_id` 单参数不超过 512 字节，防止 DoS 与日志/缓存膨胀
12. **审计记录**: 每次用户查询（`/user`、`/v1/user`、`/v1/lookup`）以及每次修改日志级别的请求，都会以一行带 `"audit": true` 的 JSON 写入应用日志。记录包含结果、按 `TRUSTED_PROXY_IPS` 解析出的客户端地址、请求 ID；修改日志级别时还包含已认证的调用方以及修改前后的级别。手机号、邮箱和 user_id 均已脱敏。Warden 没有独立的审计存储：保留期限与访问控制由日志管道决定

## 安全最佳实践

### 1. 生产环境配置

**必须配置项**：
- **必须**设置 `ENVIRONMENT=production` 启用生产安全策略。
- **必须**至少配置一种服务鉴权方式：`API_KEY`、HMAC v2 或 mTLS。
- **必须**将 `TRUSTED_PROXY_IPS` 配置为反向代理地址；Warden 前面没有反向代理时配置为 `none`（忽略 `X-Forwarded-For` 与 `X-Real-IP`，使用 TCP 对端地址）。该变量为空时，私有或回环地址的对端可以通过 `X-Forwarded-For` 自行指定客户端地址，而认证失败的限流按客户端地址计数，因此这类对端每伪造一个地址就获得新的配额，来自内网的凭证猜测不受限流约束；此时 Warden 会在启动时输出警告。同时设置了 `IP_WHITELIST` 而该变量为空时，生产环境拒绝启动
- **必须**使用 `HEALTH_CHECK_IP_WHITELIST` 限制健康检查访问（或通过网络/反向代理限制 `/health`、`/healthcheck`）
- **必须**限制 `/metrics`。`ENVIRONMENT=production` 下默认即要求认证，请保持该默认值（或在反向代理/网络层限制该路径），**不要**在生产环境设置 `WARDEN_METRICS_REQUIRE_AUTH=false`。

**配置示例**:
```bash
export API_KEY="your-strong-api-key-here"
export ENVIRONMENT=production
export WARDEN_METRICS_REQUIRE_AUTH=true
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"
```

### 2. 敏感信息管理

**推荐做法**:
- ✅ 使用环境变量存储密码和密钥
- ✅ 使用密码文件（`REDIS_PASSWORD_FILE`）存储 Redis 密码
- ✅ 在配置文件中使用占位符或注释说明
- ✅ 确保配置文件权限设置正确（如 `chmod 600`）

**不推荐做法**:
- ❌ 在配置文件中硬编码密码
- ❌ 通过命令行参数传递密码（会出现在进程列表中）
- ❌ 将包含敏感信息的配置文件提交到版本控制

**示例**:
```yaml
# config.yaml
redis:
  addr: "localhost:6379"
  # password: ""  # 使用环境变量 REDIS_PASSWORD 或 REDIS_PASSWORD_FILE

app:
  # api_key: ""  # 使用环境变量 API_KEY
```

### 3. 网络安全

**必须配置**:
- 生产环境必须使用 HTTPS
- 配置防火墙规则限制访问
- 定期更新依赖项以修复已知漏洞

**推荐配置**:
- 使用反向代理（如 Nginx）处理 SSL/TLS
- 配置 `TRUSTED_PROXY_IPS` 以正确获取客户端真实 IP
- 使用强密码和 API 密钥
- 禁用 `HTTP_INSECURE_TLS`（生产环境必须为 `false`）

### 4. 监控和审计

**推荐做法**:
- 监控安全事件日志
- 定期审查访问日志
- 使用 CI/CD 中的安全扫描工具
- 设置告警机制

**日志级别管理**:
- 生产环境建议使用 `info` 或 `warn` 级别
- 每次修改日志级别的请求（无论成功或被拒绝）都会作为审计记录（`"audit": true`）写入应用日志，包含调用方、客户端地址以及修改前后的级别
- 通过 `/log/level` API 可以动态调整日志级别（需要配置的服务鉴权）

## API 安全

### API Key 认证

部分 API 端点需要配置的服务鉴权：

**需要认证的端点**:
- `GET /` - 获取用户列表
- `GET /user` - 查询单个用户
- `GET /log/level` - 获取日志级别
- `POST /log/level` - 设置日志级别

这些端点不会在没有凭证的情况下开放。未配置 `API_KEY`、`WARDEN_HMAC_KEYS` 或 mTLS 时，生产环境拒绝启动；其他环境可以启动，但这些端点对所有请求返回 401，并在启动时输出一条警告说明原因。

**不需要认证的端点**（生产环境必须通过其他手段保护）:
- `GET /health` - 健康检查（**必须**配置 `HEALTH_CHECK_IP_WHITELIST` 或网络隔离）
- `GET /healthcheck` - 健康检查（同上）
- `GET /metrics` - Prometheus 指标（**必须**配置 API Key 使拉取时带 Key，或反向代理/网络限制，不对外网暴露）

**认证方式**:
1. **X-API-Key 请求头**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer 头**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### 速率限制

默认情况下，API 请求受到速率限制保护：

- **限制**: 每分钟 60 次请求
- **窗口**: 1 分钟
- **超出限制**: 返回 `429 Too Many Requests`

可以通过配置文件调整：

```yaml
rate_limit:
  rate: 60  # 每分钟请求数
  window: 1m
```

### IP 白名单

支持两种 IP 白名单配置：

1. **全局 IP 白名单** (`IP_WHITELIST`):
   - 限制所有端点的访问
   - 支持 CIDR 网段格式

2. **健康检查 IP 白名单** (`HEALTH_CHECK_IP_WHITELIST`):
   - 仅限制 `/health` 和 `/healthcheck` 端点
   - 支持 CIDR 网段格式

**配置示例**:
```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## 数据安全

### 远程配置 API 安全

- 远程配置 API 应使用认证机制（Authorization 头）
- 建议使用 HTTPS 协议
- 验证远程 API 的 TLS 证书（生产环境必须）

### Redis 安全

- **Redis 实例属于信任边界，不得与其他服务共用。** 除 `ONLY_LOCAL` 外的所有模式，启动时都会在读取任何数据源之前先采用 Redis 中缓存的规则集。默认的 `REDIS_SNAPSHOT_MODE=legacy` 下，该缓存位于固定键 `warden:users:cache`（以及 `warden:users:cache:version`），没有命名空间、不带认证，且 `redis.db` 不生效，始终使用 0 号库。任何能写入该键的一方——同一实例上的其他服务、数据集不同的另一套 Warden——都能决定允许名单的内容，直到下一次刷新成功为止；数据源持续失败时则一直有效。该键还会以明文保存完整用户列表一小时
- 当 Redis 库中存在非 Warden 写入的键时，Warden 会在启动时输出警告；生产环境下 Redis 未设置密码时同样会输出警告。前一项检查只抽样读取部分键，没有警告并不能证明实例是专用的
- Redis 应配置密码保护
- 使用 `REDIS_PASSWORD` 或 `REDIS_PASSWORD_FILE` 环境变量
- 限制 Redis 的网络访问（仅允许应用服务器访问）
- 定期更新 Redis 以修复已知漏洞

### 数据文件安全

- 确保 `data.json` 文件权限设置正确
- 不要将敏感数据提交到版本控制
- 定期备份数据文件

## 安全响应头

Warden 自动添加以下安全相关的 HTTP 响应头：

- `X-Content-Type-Options: nosniff` - 防止 MIME 类型嗅探
- `X-Frame-Options: DENY` - 防止点击劫持
- `X-XSS-Protection: 1; mode=block` - XSS 保护

## 错误处理

### 生产模式

在生产模式下（`ENVIRONMENT=production`）：

- 隐藏详细的错误信息，防止信息泄露
- 返回通用的错误消息
- 详细的错误信息仅记录在日志中

### 开发模式

在开发模式下：

- 显示详细的错误信息，便于调试
- 包含堆栈跟踪信息

## 安全审计

发布加固与验证说明请参考[发布安全指南](../RELEASE_SECURITY.md)。

## 漏洞报告

如果发现安全漏洞，请通过以下方式报告：

1. 创建私有安全 Issue（如果支持）
2. 发送邮件到项目维护者
3. 不要公开披露漏洞，直到修复完成

<a name="服务间鉴权"></a>

## 服务间鉴权（可选）

Warden 支持 mTLS、HMAC 和 API Key，按下述实际请求分支鉴权。独立部署也必须满足生产鉴权要求；只有开发或测试环境可以使用文档允许的未鉴权策略。

### mTLS（已实现/推荐）

使用双向 TLS 证书进行身份验证，提供更高的安全性。

**配置方式**：

1. **生成证书**：
   ```bash
   # 生成 CA 证书
   openssl genrsa -out ca.key 2048
   openssl req -new -x509 -days 365 -key ca.key -out ca.crt
   
   # 生成 Warden 服务端证书
   openssl genrsa -out warden.key 2048
   openssl req -new -key warden.key -out warden.csr
   openssl x509 -req -days 365 -in warden.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out warden.crt
   
   # 生成 Stargate 客户端证书
   openssl genrsa -out stargate.key 2048
   openssl req -new -key stargate.key -out stargate.csr
   openssl x509 -req -days 365 -in stargate.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out stargate.crt
   ```

2. **Warden 配置**（环境变量）：
   ```bash
   export WARDEN_TLS_CERT=/path/to/warden.crt
   export WARDEN_TLS_KEY=/path/to/warden.key
   export WARDEN_TLS_CA=/path/to/ca.crt
   export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
   ```

3. **Stargate 配置**：
   - 配置客户端证书路径
   - 配置 CA 证书路径以验证 Warden 服务端证书

### HMAC 签名（已实现）

使用 HMAC-SHA256 签名验证请求，更易于部署。

**签名算法**：
```text
canonical_v2 = METHOD + "\n" + ESCAPED_PATH_AND_QUERY + "\n" + KEY_ID + "\n" +
               TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(BODY)
signature = HEX(HMAC_SHA256(secret, canonical_v2))
```

**请求头**：
- `X-Signature`: HMAC 签名值
- `X-Timestamp`: Unix 时间戳（秒）
- `X-Key-Id`: 密钥 ID（纳入签名，用于安全轮换）
- `X-Nonce`: 唯一的 128 bit 十六进制随机数
- `X-Signature-Version`: `v2`

**Warden 配置**（环境变量）：
```bash
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60  # 时间戳容差（秒），默认 60
export WARDEN_HMAC_ALLOW_V1=false          # 默认值；仅在旧调用方迁移期间临时设为 true
```

生产环境要求每个 HMAC secret 至少包含 32 个原始字节。请使用密码学安全随机源生成，
并存储到密钥管理服务中；不要复用上面的示例值。

**Go SDK 示例**：
```go
client, err := warden.NewClient(warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET")))
```

**验证规则**：
- Warden 会验证时间戳是否在容差范围内（默认 ±60 秒）
- Warden 会验证包含 Key ID 在内的全部 canonical 字段，并拒绝重复 nonce
- 如果签名验证失败，返回 `401 Unauthorized`

### 配置优先级

1. 启用 `WARDEN_SERVICE_POLICIES` 时，只接受已验证的 HMAC v2 key ID 及其允许的 action；API Key 和 mTLS 不能绕过该模式。
2. 未启用该模式时，请求携带符合服务端 TLS 策略的客户端证书，才可跳过请求头鉴权。仅配置服务端证书不等于每个请求都已鉴权。
3. 否则，完整的 HMAC 请求头与已配置密钥进入签名校验；签名失败直接返回 401，不回退 API Key。旧 v1 还需显式开启兼容开关。
4. 没有进入上述完整 HMAC 分支的请求交给 API Key 中间件（`X-API-Key` 或 Bearer）。配置 HMAC 本身不会关闭另外配置的 API Key 访问。

### 安全建议

1. **生产环境**：强烈建议使用 mTLS 进行服务间鉴权
2. **密钥管理**：使用密钥管理服务（如 HashiCorp Vault）存储密钥和证书
3. **密钥轮换**：定期轮换 HMAC 密钥和 TLS 证书
4. **网络隔离**：在可能的情况下，使用网络策略限制只有 Stargate 可以访问 Warden

## 相关文档

- [配置文档](CONFIGURATION.md) - 了解安全相关的配置选项
- [部署文档](DEPLOYMENT.md) - 了解生产环境部署建议
- [API 文档](API.md) - 了解 API 安全特性
- [架构文档](ARCHITECTURE.md) - 了解服务集成架构

## Diagnostic compatibility and historical logs

Upgrade requirements, diagnostic field semantics and historical log handling are
covered in the [v1.5.0 release notes](../releases/v1.5.0.md). Upgrading does not
rewrite previously stored logs. Masked identifiers and resource digests remain
correlatable; access and retention controls still apply.

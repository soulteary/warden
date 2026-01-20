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

## 安全最佳实践

### 1. 生产环境配置

**必须配置项**:
- 必须设置 `API_KEY` 环境变量
- 设置 `MODE=production` 启用生产模式
- 配置 `TRUSTED_PROXY_IPS` 以正确获取客户端 IP
- 使用 `HEALTH_CHECK_IP_WHITELIST` 限制健康检查访问

**配置示例**:
```bash
export API_KEY="your-strong-api-key-here"
export MODE=production
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
- 所有日志级别修改操作都会被记录到安全审计日志中
- 通过 `/log/level` API 可以动态调整日志级别（需要 API Key 认证）

## API 安全

### API Key 认证

部分 API 端点需要 API Key 认证：

**需要认证的端点**:
- `GET /` - 获取用户列表
- `GET /user` - 查询单个用户
- `GET /log/level` - 获取日志级别
- `POST /log/level` - 设置日志级别

**不需要认证的端点**:
- `GET /health` - 健康检查（可通过 IP 白名单限制）
- `GET /healthcheck` - 健康检查（可通过 IP 白名单限制）
- `GET /metrics` - Prometheus 指标

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

在生产模式下（`MODE=production` 或 `MODE=prod`）：

- 隐藏详细的错误信息，防止信息泄露
- 返回通用的错误消息
- 详细的错误信息仅记录在日志中

### 开发模式

在开发模式下：

- 显示详细的错误信息，便于调试
- 包含堆栈跟踪信息

## 安全审计

详细的安全审计报告请参考 [SECURITY_AUDIT.md](../SECURITY_AUDIT.md)（如果存在）。

## 漏洞报告

如果发现安全漏洞，请通过以下方式报告：

1. 创建私有安全 Issue（如果支持）
2. 发送邮件到项目维护者
3. 不要公开披露漏洞，直到修复完成

## 相关文档

- [配置文档](CONFIGURATION.md) - 了解安全相关的配置选项
- [部署文档](DEPLOYMENT.md) - 了解生产环境部署建议
- [API 文档](API.md) - 了解 API 安全特性

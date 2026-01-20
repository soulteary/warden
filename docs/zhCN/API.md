# API 文档

> 🌐 **Language / 语言**: [English](../enUS/API.md) | [中文](API.md)

本文档详细说明 Warden 提供的所有 API 端点。

## OpenAPI 文档

项目提供了完整的 OpenAPI 3.0 规范文档，位于 `openapi.yaml` 文件中。

你可以使用以下工具查看和测试 API：

1. **Swagger UI**: 使用 [Swagger Editor](https://editor.swagger.io/) 打开 `openapi.yaml` 文件
2. **Postman**: 导入 `openapi.yaml` 文件到 Postman
3. **Redoc**: 使用 Redoc 生成美观的 API 文档页面

## 认证

部分 API 端点需要 API Key 认证。可以通过以下两种方式提供认证信息：

1. **X-API-Key 请求头**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer 头**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

API Key 可以通过环境变量 `API_KEY` 或命令行参数 `--api-key` 配置。

## API 端点

### 获取用户列表

获取所有用户或分页用户列表。

**请求**
```http
GET /
X-API-Key: your-secret-api-key

GET /?page=1&page_size=100
X-API-Key: your-secret-api-key
```

**查询参数**:
- `page` (可选): 页码，从 1 开始，默认为 1
- `page_size` (可选): 每页数量，默认为所有数据（不分页）

**注意**: 此端点需要 API Key 认证。

**响应（无分页）**
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com"
    }
]
```

**响应（有分页）**
```json
{
    "data": [
        {
            "phone": "13800138000",
            "mail": "admin@example.com"
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 100,
        "total": 200,
        "total_pages": 2
    }
}
```

**状态码**: `200 OK`

**Content-Type**: `application/json`

### 查询单个用户

根据手机号、邮箱或用户 ID 查询单个用户信息。

**请求**
```http
GET /user?phone=13800138000
X-API-Key: your-secret-api-key

GET /user?mail=admin@example.com
X-API-Key: your-secret-api-key

GET /user?user_id=user-123
X-API-Key: your-secret-api-key
```

**查询参数**（必须提供且只能提供一个）:
- `phone`: 用户手机号
- `mail`: 用户邮箱地址
- `user_id`: 用户唯一标识符

**注意**: 
- 此端点需要 API Key 认证
- 只能提供一个查询参数（`phone`、`mail` 或 `user_id` 之一）

**响应（用户存在）**
```json
{
    "phone": "13800138000",
    "mail": "admin@example.com",
    "user_id": "user-123",
    "status": "active",
    "scope": ["read", "write"],
    "role": "admin"
}
```

**响应（用户不存在）**
- **状态码**: `404 Not Found`
- **响应体**: `User not found`

**错误响应（缺少参数）**
- **状态码**: `400 Bad Request`
- **响应体**: `Bad Request: missing identifier (phone, mail, or user_id)`

**错误响应（多个参数）**
- **状态码**: `400 Bad Request`
- **响应体**: `Bad Request: only one identifier allowed (phone, mail, or user_id)`

### 健康检查

检查服务健康状态，包括 Redis 连接状态、数据加载状态等。

**请求**
```http
GET /health
GET /healthcheck
```

**注意**: 此端点不需要认证，但可以通过 `HEALTH_CHECK_IP_WHITELIST` 环境变量限制访问 IP。

**响应**
```json
{
    "status": "ok",
    "details": {
        "redis": "ok",
        "data_loaded": true,
        "user_count": 100
    },
    "mode": "DEFAULT"
}
```

**状态码**: `200 OK`

**响应字段说明**:
- `status`: 服务状态，`"ok"` 表示正常
- `details.redis`: Redis 连接状态，可能的值：
  - `"ok"`: Redis 正常
  - `"unavailable"`: Redis 连接失败（fallback 模式）或 Redis 客户端为 nil
  - `"disabled"`: Redis 被显式禁用
- `details.data_loaded`: 数据是否已加载
- `details.user_count`: 当前用户数量
- `mode`: 当前运行模式

### 日志级别管理

动态获取和设置日志级别。

#### 获取当前日志级别

**请求**
```http
GET /log/level
X-API-Key: your-secret-api-key
```

**响应**
```json
{
    "level": "info"
}
```

**注意**: 此端点需要 API Key 认证。

#### 设置日志级别

**请求**
```http
POST /log/level
Content-Type: application/json
X-API-Key: your-secret-api-key

{
    "level": "debug"
}
```

**请求体**:
```json
{
    "level": "debug"
}
```

**支持的日志级别**: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

**响应**
```json
{
    "level": "debug",
    "message": "Log level updated successfully"
}
```

**注意**: 
- 此端点需要 API Key 认证
- 所有日志级别修改操作都会被记录到安全审计日志中

### Prometheus 指标

获取 Prometheus 格式的监控指标数据。

**请求**
```http
GET /metrics
```

**响应**: Prometheus 格式的指标数据

**注意**: 此端点不需要认证。

**示例响应**:
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/",status="200"} 1234

# HELP http_request_duration_seconds HTTP request duration in seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/",le="0.005"} 1000
http_request_duration_seconds_bucket{method="GET",path="/",le="0.01"} 1200
...
```

## 错误响应

所有 API 端点都可能返回以下错误响应：

### 401 Unauthorized

当 API Key 认证失败时返回：

```json
{
    "error": "Unauthorized",
    "message": "Invalid or missing API key"
}
```

### 429 Too Many Requests

当请求超过速率限制时返回：

```json
{
    "error": "Too Many Requests",
    "message": "Rate limit exceeded"
}
```

### 500 Internal Server Error

当服务器内部错误时返回：

```json
{
    "error": "Internal Server Error",
    "message": "An internal error occurred"
}
```

在生产模式下，详细的错误信息会被隐藏以防止信息泄露。

## 速率限制

默认情况下，API 请求受到速率限制保护：

- **限制**: 每分钟 60 次请求
- **窗口**: 1 分钟
- **超出限制**: 返回 `429 Too Many Requests`

速率限制可以通过配置文件调整：

```yaml
rate_limit:
  rate: 60  # 每分钟请求数
  window: 1m
```

## IP 白名单

可以通过以下环境变量配置 IP 白名单：

- `IP_WHITELIST`: 全局 IP 白名单（限制所有端点的访问）
- `HEALTH_CHECK_IP_WHITELIST`: 健康检查端点 IP 白名单（仅限制 `/health` 和 `/healthcheck`）

支持 CIDR 网段格式，多个 IP 或网段用逗号分隔：

```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## 响应压缩

所有 API 响应都支持自动压缩（gzip），客户端可以通过 `Accept-Encoding: gzip` 请求头启用压缩。

## 相关文档

- [OpenAPI 规范](../openapi.yaml) - 完整的 OpenAPI 3.0 规范
- [配置文档](CONFIGURATION.md) - 了解如何配置 API Key 和其他选项
- [安全文档](SECURITY.md) - 了解安全特性和最佳实践

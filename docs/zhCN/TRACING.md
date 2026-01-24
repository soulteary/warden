# Warden OpenTelemetry Tracing

Warden 服务支持 OpenTelemetry 分布式追踪，用于监控和调试服务间的调用链路。

## 功能特性

- **自动 HTTP 请求追踪**：自动为所有 HTTP 请求创建 span
- **用户查询追踪**：为 `/user` 端点添加详细的追踪信息
- **上下文传播**：支持 W3C Trace Context 标准，与 Stargate 和 Herald 服务无缝集成
- **可配置**：通过环境变量或配置文件启用/禁用

## 配置

### 环境变量

```bash
# 启用 OpenTelemetry 追踪
OTLP_ENABLED=true

# OTLP 端点（例如：Jaeger、Tempo、OpenTelemetry Collector）
OTLP_ENDPOINT=http://localhost:4318
```

### 配置文件（YAML）

```yaml
tracing:
  enabled: true
  endpoint: "http://localhost:4318"
```

## 核心 Span

### HTTP 请求 Span

所有 HTTP 请求都会自动创建 span，包含以下属性：
- `http.method`: HTTP 方法
- `http.url`: 请求 URL
- `http.status_code`: 响应状态码
- `http.user_agent`: 用户代理
- `http.remote_addr`: 客户端地址

### 用户查询 Span (`warden.get_user`)

`/user` 端点的查询会创建专门的 span，包含：
- `warden.query.phone`: 查询的手机号（脱敏）
- `warden.query.mail`: 查询的邮箱（脱敏）
- `warden.query.user_id`: 查询的用户 ID
- `warden.user.found`: 是否找到用户
- `warden.user.id`: 找到的用户 ID

## 使用示例

### 启动 Warden 并启用追踪

```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://localhost:4318
./warden
```

### 在代码中使用追踪

```go
import "github.com/soulteary/warden/internal/tracing"

// 创建子 span
ctx, span := tracing.StartSpan(ctx, "warden.custom_operation")
defer span.End()

// 设置属性
span.SetAttributes(attribute.String("key", "value"))

// 记录错误
if err != nil {
    tracing.RecordError(span, err)
}
```

## 与 Stargate 和 Herald 集成

Warden 的追踪会自动与 Stargate 和 Herald 服务的追踪上下文集成：

1. **Stargate** 调用 Warden 时，会通过 HTTP 头传递 trace context
2. **Warden** 自动提取并继续追踪链路
3. 所有三个服务的 span 会在同一个 trace 中显示

## 支持的追踪后端

- **Jaeger**: `OTLP_ENDPOINT=http://localhost:4318`
- **Tempo**: `OTLP_ENDPOINT=http://localhost:4318`
- **OpenTelemetry Collector**: `OTLP_ENDPOINT=http://localhost:4318`
- **其他 OTLP 兼容后端**

## 性能考虑

- 追踪默认使用批处理导出，对性能影响最小
- 可以通过采样率控制追踪数据量
- 生产环境建议使用采样策略（当前为全采样，适合开发环境）

## 故障排查

### 追踪未启用

检查环境变量：
```bash
echo $OTLP_ENABLED
echo $OTLP_ENDPOINT
```

### 追踪数据未到达后端

1. 检查 OTLP 端点是否可访问
2. 检查网络连接
3. 查看 Warden 日志中的错误信息

### Span 缺失

确保在请求处理中使用 `r.Context()` 传递上下文，而不是创建新的 context。

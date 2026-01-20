# 部署文档

> 🌐 **Language / 语言**: [English](DEPLOYMENT.en.md) | [中文](DEPLOYMENT.md)

本文档说明如何部署 Warden 服务，包括 Docker 部署、本地部署等。

## 前置要求

- Go 1.25+ (参考 [go.mod](../go.mod))
- Redis (用于分布式锁和缓存)
- Docker (可选，用于容器化部署)

## Docker 部署

> 🚀 **快速部署**: 查看 [示例目录](../example/README.md) / [Examples Directory](../example/README.en.md) 获取完整的 Docker Compose 配置示例：
> - [简单示例](../example/basic/docker-compose.yml) / [Simple Example](../example/basic/docker-compose.yml) - 基础 Docker Compose 配置
> - [复杂示例](../example/advanced/docker-compose.yml) / [Advanced Example](../example/advanced/docker-compose.yml) - 包含 Mock API 的完整配置

### 使用 Docker Compose

1. **准备环境变量文件**
   
   如果项目根目录存在 `.env.example` 文件，可以复制它：
   ```bash
   cp .env.example .env
   ```
   
   如果不存在 `.env.example` 文件，可以手动创建 `.env` 文件，参考以下内容：
   ```env
   # 服务器配置
   PORT=8081
   
   # Redis 配置
   REDIS=warden-redis:6379
   # Redis 密码（可选，建议使用环境变量而不是配置文件）
   # REDIS_PASSWORD=your-redis-password
   # 或使用密码文件（更安全）
   # REDIS_PASSWORD_FILE=/path/to/redis-password.txt
   
   # 远程数据 API
   CONFIG=http://example.com/api/data.json
   # 远程配置 API 认证密钥
   KEY=Bearer your-token-here
   
   # 任务配置
   INTERVAL=5
   
   # 应用模式
   MODE=DEFAULT
   
   # HTTP 客户端配置（可选）
   # HTTP_TIMEOUT=5
   # HTTP_MAX_IDLE_CONNS=100
   # HTTP_INSECURE_TLS=false
   
   # API Key（用于 API 认证，生产环境必须设置）
   API_KEY=your-api-key-here
   
   # 健康检查 IP 白名单（可选，逗号分隔）
   # HEALTH_CHECK_IP_WHITELIST=127.0.0.1,::1,10.0.0.0/8
   
   # 信任的代理 IP 列表（可选，逗号分隔，用于反向代理环境）
   # TRUSTED_PROXY_IPS=127.0.0.1,10.0.0.1
   
   # 日志级别（可选）
   # LOG_LEVEL=info
   ```
   
   > ⚠️ **安全提示**: `.env` 文件包含敏感信息，不要提交到版本控制系统。`.env` 文件已被 `.gitignore` 忽略。请使用上述内容作为模板创建 `.env` 文件。

2. **启动服务**
```bash
docker-compose up -d
```

### 手动构建镜像

```bash
docker build -f docker/Dockerfile -t warden-release .
```

### 运行容器

```bash
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api \
  -e KEY="Bearer token" \
  warden-release
```

## 本地部署

### 1. 克隆项目

```bash
git clone <repository-url>
cd warden
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置本地数据文件

创建 `data.json` 文件（可参考 `data.example.json`）：
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**注意**：`data.json` 支持以下字段：
- `phone`（必需）：用户手机号
- `mail`（必需）：用户邮箱地址
- `user_id`（可选）：用户唯一标识符，如果未提供则自动生成
- `status`（可选）：用户状态，如 "active"、"inactive"、"suspended"，默认为 "active"
- `scope`（可选）：用户权限范围数组，如 `["read", "write"]`
- `role`（可选）：用户角色，如 "admin"、"user"

完整示例请参考 `data.example.json` 文件。

### 4. 运行服务

```bash
go run main.go
```

## 生产环境部署建议

### 1. 使用反向代理

建议在生产环境使用 Nginx 或 Traefik 等反向代理：

**Nginx 配置示例**:
```nginx
upstream warden {
    server localhost:8081;
}

server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://warden;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. 使用 HTTPS

生产环境必须使用 HTTPS。可以通过以下方式实现：

- 使用 Let's Encrypt 免费证书
- 使用反向代理（如 Nginx）处理 SSL/TLS
- 配置 `TRUSTED_PROXY_IPS` 环境变量以正确获取客户端真实 IP

### 3. 配置监控

- 使用 Prometheus 收集指标（通过 `/metrics` 端点）
- 配置健康检查（通过 `/health` 端点）
- 设置日志收集和分析

### 4. 高可用部署

- 部署多个实例，使用负载均衡器分发请求
- 使用共享的 Redis 实例确保数据一致性
- 配置自动重启和故障转移

### 5. 资源限制

在 Docker Compose 或 Kubernetes 中配置资源限制：

```yaml
services:
  warden:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Kubernetes 部署

### 基本部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: warden
spec:
  replicas: 3
  selector:
    matchLabels:
      app: warden
  template:
    metadata:
      labels:
        app: warden
    spec:
      containers:
      - name: warden
        image: warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "redis-service:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: warden-service
spec:
  selector:
    app: warden
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8081
  type: LoadBalancer
```

## 性能优化

### 1. Redis 配置

- 使用 Redis 持久化（RDB 或 AOF）
- 配置合适的 Redis 内存限制
- 使用 Redis 集群（如果需要）

### 2. 应用配置

- 调整 `HTTP_MAX_IDLE_CONNS` 以优化连接池
- 配置合适的 `INTERVAL` 以平衡实时性和性能
- 使用合适的运行模式（`MODE`）

### 3. 监控和调优

基于 wrk 压力测试结果（30秒测试，16线程，100连接）：

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
平均延迟:       21.30ms
最大延迟:       226.09ms
```

根据实际负载调整配置参数。

## 相关文档

- [配置文档](CONFIGURATION.md) - 了解详细的配置选项
- [安全文档](SECURITY.md) - 了解安全配置和最佳实践
- [架构设计文档](ARCHITECTURE.md) - 了解系统架构

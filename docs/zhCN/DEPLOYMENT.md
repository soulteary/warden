# 部署文档

> 🌐 **Language / 语言**: [English](../enUS/DEPLOYMENT.md) | [中文](DEPLOYMENT.md) | [Français](../frFR/DEPLOYMENT.md) | [Italiano](../itIT/DEPLOYMENT.md) | [日本語](../jaJP/DEPLOYMENT.md) | [Deutsch](../deDE/DEPLOYMENT.md) | [한국어](../koKR/DEPLOYMENT.md)

本文档说明如何部署 Warden 服务，包括 Docker 部署、本地部署等。

## 前置要求

- Go 1.27+（参考 [go.mod](../../go.mod)）
- Redis (用于分布式锁和缓存)
- Docker (可选，用于容器化部署)

## Docker 部署

> 🚀 **快速部署**: 查看 [示例目录](../../example/README.md) / [Examples Directory](../../example/README.md) 获取完整的 Docker Compose 配置示例：
> - [简单示例](../../example/basic/docker-compose.yml) / [Simple Example](../../example/basic/docker-compose.yml) - 基础 Docker Compose 配置
> - [复杂示例](../../example/advanced/docker-compose.yml) / [Advanced Example](../../example/advanced/docker-compose.yml) - 包含 Mock API 的完整配置

### 使用预构建镜像（推荐）

Warden 提供了预构建的 Docker 镜像，可以直接从 GitHub Container Registry (GHCR) 拉取使用，无需手动构建：

```bash
# 拉取最新版本的镜像
docker pull ghcr.io/soulteary/warden:latest

# 运行容器
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api/data.json \
  -e KEY="Bearer your-token-here" \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **提示**: 使用预构建镜像可以快速开始，无需本地构建环境。镜像会自动更新，确保使用最新版本。

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
   MERGE_MODE=DEFAULT
   
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
- `status`（可选）：用户状态，如 "active"、"inactive"、"suspended"；省略时默认为 "inactive"
- `scope`（可选）：用户权限范围数组，如 `["read", "write"]`
- `role`（可选）：用户角色，如 "admin"、"user"

完整示例请参考 `data.example.json` 文件。

### 4. 运行服务

```bash
go run .
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
- 使用合适的数据合并模式（`MERGE_MODE`）

### 3. 监控和调优

基于 wrk 压力测试结果（30秒测试，16线程，100连接）：

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
平均延迟:       21.30ms
最大延迟:       226.09ms
```

根据实际负载调整配置参数。

## 可选集成部署（与 Stargate/Herald）

Warden 可以独立部署使用，也可以选择性地与 Stargate 和 Herald 集成部署。以下是可选的集成部署配置示例。

**注意**：以下集成部署方案是可选的，Warden 完全可以独立部署和使用。

### Docker Compose 集成示例

完整的 Stargate + Warden + Herald 集成部署配置：

```yaml
version: '3.8'

services:
  # Warden 服务
  warden:
    image: ghcr.io/soulteary/warden:latest
    container_name: warden
    ports:
      - "8081:8081"
    networks:
      - auth-network
    environment:
      - PORT=8081
      - REDIS=warden-redis:6379
      - API_KEY=${WARDEN_API_KEY}
      - MERGE_MODE=DEFAULT
      # 服务间鉴权配置（HMAC 示例）
      - WARDEN_HMAC_KEYS=${WARDEN_HMAC_KEYS}
      - WARDEN_HMAC_TIMESTAMP_TOLERANCE=60
    volumes:
      - ./warden-data.json:/app/data.json:ro
    healthcheck:
      test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
      interval: 10s
      timeout: 1s
      retries: 3
    depends_on:
      - warden-redis

  # Warden Redis
  warden-redis:
    image: redis:7.4-alpine
    container_name: warden-redis
    networks:
      - auth-network
    volumes:
      - warden-redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 1s
      retries: 3

  # Stargate 服务（示例配置）
  stargate:
    image: ghcr.io/soulteary/stargate:latest
    container_name: stargate
    ports:
      - "8080:8080"
    networks:
      - auth-network
    environment:
      - STARGATE_WARDEN_BASE_URL=http://warden:8081
      - STARGATE_WARDEN_AUTH_TYPE=hmac
      - STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
      - STARGATE_WARDEN_HMAC_SECRET=${WARDEN_HMAC_SECRET}
      - STARGATE_HERALD_BASE_URL=http://herald:8082
    depends_on:
      - warden
      - herald

  # Herald 服务（示例配置）
  herald:
    image: ghcr.io/soulteary/herald:latest
    container_name: herald
    ports:
      - "8082:8082"
    networks:
      - auth-network
    environment:
      - HERALD_REDIS_URL=redis://herald-redis:6379
    depends_on:
      - herald-redis

  # Herald Redis
  herald-redis:
    image: redis:7.4-alpine
    container_name: herald-redis
    networks:
      - auth-network
    volumes:
      - herald-redis-data:/data

networks:
  auth-network:
    driver: bridge

volumes:
  warden-redis-data:
  herald-redis-data:
```

### 环境变量配置

创建 `.env` 文件：

```bash
# Warden API Key
WARDEN_API_KEY=your-warden-api-key-here

# Warden HMAC 密钥（JSON 格式）
WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'

# Stargate 使用的 HMAC 密钥（与 WARDEN_HMAC_KEYS 中的密钥对应）
WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef
```

### 网络配置

所有服务应在同一 Docker 网络中，以便相互通信：

- **Warden**：监听 `8081` 端口，供 Stargate 调用
- **Stargate**：监听 `8080` 端口，作为 Traefik forwardAuth 服务
- **Herald**：监听 `8082` 端口，供 Stargate 调用

### 服务依赖

- **Stargate** 依赖 **Warden** 和 **Herald**
- **Warden** 依赖 **warden-redis**（可选，如果启用 Redis）
- **Herald** 依赖 **herald-redis**

### 健康检查

所有服务都应配置健康检查，确保服务正常运行：

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
  interval: 10s
  timeout: 1s
  retries: 3
```

### 生产环境建议

1. **使用独立的 Redis 实例**：Warden 和 Herald 应使用独立的 Redis 实例，避免数据冲突
2. **配置服务间鉴权**：生产环境必须配置 mTLS 或 HMAC 签名
3. **使用密钥管理服务**：使用 HashiCorp Vault 或类似服务管理密钥和证书
4. **网络隔离**：使用 Docker 网络策略限制服务间访问
5. **监控和日志**：配置统一的监控和日志收集系统

### Kubernetes 集成部署

在 Kubernetes 中部署时，建议：

1. **使用 Service**：为每个服务创建 Kubernetes Service
2. **使用 ConfigMap 和 Secret**：存储配置和密钥
3. **使用 NetworkPolicy**：限制服务间网络访问
4. **使用 Ingress**：配置 Traefik Ingress 路由到 Stargate

示例 Kubernetes 配置：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: warden
spec:
  selector:
    app: warden
  ports:
    - port: 8081
      targetPort: 8081
---
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
        image: ghcr.io/soulteary/warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "warden-redis:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        - name: WARDEN_HMAC_KEYS
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: hmac-keys
```

## 相关文档

- [配置文档](CONFIGURATION.md) - 了解详细的配置选项
- [安全文档](SECURITY.md) - 了解安全配置和最佳实践
- [架构设计文档](ARCHITECTURE.md) - 了解系统架构
- [API 文档](API.md) - 了解 API 接口和集成示例

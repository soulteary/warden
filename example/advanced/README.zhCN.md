# 复杂示例 - 完整功能演示

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md)

这是 Warden 的完整功能示例，展示了所有核心特性，包括：
- 本地数据文件
- 远程 API 数据源
- Redis 缓存和分布式锁
- 定时任务自动同步
- 多种数据合并策略
- Docker Compose 完整部署

## 📋 前置要求

- Docker 和 Docker Compose
- 或 Go 1.27+ 和 Redis

## 🏗️ 架构说明

本示例包含以下组件：

```
┌─────────────────┐
│   Warden API    │  ← 主服务（端口 8081）
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼───┐  ┌──▼──────┐
│ Redis │  │ Mock    │  ← 模拟远程 API（端口 8080）
│ Cache │  │ API     │
└───────┘  └─────────┘
```

## 🚀 快速开始

### 方式一：使用 Docker Compose（推荐）

1. **准备环境**

```bash
cd example/advanced
cp .env.example .env
# 编辑 .env 文件，设置你的配置
```

2. **启动所有服务**

```bash
# 从 GHCR 拉取最新镜像（可选，docker-compose 会自动拉取）
docker-compose pull

# 启动所有服务
docker-compose up -d
```

这将启动：
- Warden 主服务（端口 8081）- 默认使用 `ghcr.io/soulteary/warden:latest`
- Redis 缓存服务（端口 6379）
- Mock 远程 API 服务（端口 8080）

**注意**：示例使用 GitHub Container Registry (GHCR) 提供的预构建镜像。你可以通过在 `.env` 文件中设置 `WARDEN_IMAGE` 和 `WARDEN_IMAGE_TAG` 来自定义镜像。

3. **查看服务状态**

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f warden
docker-compose logs -f mock-api
```

4. **测试服务**

```bash
# 健康检查
curl http://localhost:8081/health

# 获取用户列表（需要 API Key）
curl -H "X-API-Key: your-secret-api-key" http://localhost:8081/

# 查看 Prometheus 指标
curl http://localhost:8081/metrics
```

### 方式二：本地运行

1. **启动 Redis**

```bash
docker run -d --name redis -p 6379:6379 redis:7.4-alpine
```

2. **启动 Mock API 服务**

```bash
cd example/advanced
go run mock-api/main.go
```

Mock API 将在 `http://localhost:8080/api/users` 提供服务。

3. **运行 Warden**

```bash
# 在项目根目录
go run . \
  --port 8081 \
  --redis localhost:6379 \
  --config http://localhost:8080/api/users \
  --key "Bearer mock-token" \
  --mode DEFAULT \
  --interval 10
```

## 📝 配置说明

### 数据合并策略

本示例演示了 `DEFAULT`（远程优先）模式：

- ✅ 优先从远程 API 获取数据
- ✅ 远程数据不存在时，使用本地数据补充
- ✅ 定时任务每 10 秒自动同步一次

### 环境变量配置

编辑 `.env` 文件：

```env
# 服务端口
PORT=8081

# Docker 镜像配置（可选）
# WARDEN_IMAGE=ghcr.io/soulteary/warden
# WARDEN_IMAGE_TAG=latest

# Redis 配置
REDIS=warden-redis:6379
REDIS_PASSWORD=

# 远程 API 配置
CONFIG=http://mock-api:8080/api/users
KEY=Bearer mock-token

# 任务配置
INTERVAL=10

# 运行模式
MERGE_MODE=DEFAULT

# API 认证
API_KEY=your-secret-api-key-here

# HTTP 客户端配置
HTTP_TIMEOUT=5
HTTP_MAX_IDLE_CONNS=100
```

### 数据文件

**本地数据文件** (`data.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**远程 API 数据** (由 Mock API 提供):
```json
[
    {
        "phone": "13900139000",
        "mail": "remote@example.com"
    },
    {
        "phone": "15000150000",
        "mail": "user@example.com"
    }
]
```

**合并结果** (远程优先):
```json
[
    {
        "phone": "13900139000",
        "mail": "remote@example.com"
    },
    {
        "phone": "15000150000",
        "mail": "user@example.com"
    },
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

## 🔍 功能演示

### 1. 数据同步流程

观察定时任务如何自动同步数据：

```bash
# 查看 Warden 日志
docker-compose logs -f warden

# 你会看到类似输出：
# INFO 从远程 API 加载数据 ✓ count=2
# INFO 后台更新数据 📦 count=3 duration=0.123
```

### 2. 修改远程数据

修改 Mock API 的数据文件，观察自动同步：

```bash
# 编辑 Mock API 数据
vim mock-api/data.json

# 等待 10 秒（定时任务间隔），数据会自动更新
```

### 3. 测试不同合并模式

修改 `.env` 中的 `MERGE_MODE` 参数测试不同模式（`MODE` 已弃用）：

- `DEFAULT`：保留历史的远程优先、容错行为
- `REMOTE_FIRST`：严格远程优先；远程刷新失败时保留最后一次成功快照
- `REMOTE_FIRST_ALLOW_REMOTE_FAILED`：远程优先，允许降级回退到本地
- `LOCAL_FIRST`: 本地优先
- `ONLY_REMOTE`: 仅远程
- `ONLY_LOCAL`: 仅本地

```bash
# 修改配置后重启服务
docker-compose restart warden
```

### 4. 查看监控指标

```bash
# Prometheus 指标
curl http://localhost:8081/metrics | grep warden

# 健康检查（非生产环境包含详情）
curl http://localhost:8081/health | jq
```

### 5. 测试 API 功能

```bash
# 获取所有用户
curl -H "X-API-Key: your-secret-api-key" http://localhost:8081/

# 分页查询
curl -H "X-API-Key: your-secret-api-key" \
  "http://localhost:8081/?page=1&page_size=10"

# 动态调整日志级别
curl -X POST -H "X-API-Key: your-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{"level":"debug"}' \
  http://localhost:8081/log/level
```

## 🧪 测试场景

### 场景 1: 远程 API 故障

1. 停止 Mock API 服务：
```bash
docker-compose stop mock-api
```

2. 观察 Warden 自动回退到本地数据：
```bash
docker-compose logs -f warden
# 应该看到：从本地文件加载数据
```

3. 恢复 Mock API：
```bash
docker-compose start mock-api
```

4. 观察自动恢复：
```bash
# 等待定时任务执行，数据会从远程恢复
```

### 场景 2: Redis 故障

1. 停止 Redis：
```bash
docker-compose stop warden-redis
```

2. 观察服务行为：
```bash
# Warden 会继续运行，但无法使用 Redis 缓存
# 定时任务的分布式锁会失效（多实例场景）
```

### 场景 3: 数据冲突测试

1. 修改本地和远程数据，使其有重叠：
   - 本地：`phone: 13800138000`
   - 远程：`phone: 13800138000` (不同邮箱)

2. 观察合并结果（取决于选择的模式）

## 📊 性能测试

使用 `wrk` 进行压力测试：

```bash
# 安装 wrk
# macOS: brew install wrk
# Linux: apt-get install wrk

# 运行测试
wrk -t4 -c100 -d30s \
  -H "X-API-Key: your-secret-api-key" \
  http://localhost:8081/
```

预期结果：
- 请求速率：5000+ req/s
- 平均延迟：< 25ms

## 🛠️ 故障排查

### 问题 1: 无法连接到远程 API

**症状**: 日志显示 "远程 API 加载失败"

**解决方案**:
1. 检查 Mock API 是否运行：`docker-compose ps`
2. 检查网络连接：`curl http://localhost:8080/api/users`
3. 检查认证头：确保 `KEY` 配置正确

### 问题 2: Redis 连接失败

**症状**: 启动时显示 "Redis 连接失败"

**解决方案**:
1. 检查 Redis 是否运行：`docker-compose ps warden-redis`
2. 检查 Redis 密码配置
3. 检查网络连接：`redis-cli -h localhost -p 6379 ping`

### 问题 3: 数据未更新

**症状**: 修改数据后，API 返回旧数据

**解决方案**:
1. 检查定时任务间隔配置（`INTERVAL`）
2. 查看日志确认定时任务是否执行
3. 手动触发：重启服务或等待下一个定时任务周期

## 📚 下一步

- 阅读 [完整文档](../../README.md) 了解所有功能
- 查看 [API 文档](../../openapi.yaml) 了解 API 详情
- 参考 [简单示例](../basic/README.md) 了解基础用法
- 查看 [配置示例](../../config.example.yaml) 了解所有配置选项

## 🔗 相关资源

- [Warden 主文档](../../README.md)
- [Docker Compose 文档](https://docs.docker.com/compose/)
- [Redis 文档](https://redis.io/docs/)

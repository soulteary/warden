# 简单示例 - 快速开始

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md)

这是 Warden 的最简单使用示例，仅使用本地数据文件，适合快速测试和开发环境。

## 📋 前置要求

- Go 1.27+ 或 Docker
- Redis（用于缓存和分布式锁，即使只使用本地文件也需要）

## 🚀 快速开始

### 方式一：使用 Go 运行

1. **准备数据文件**

创建 `data.json` 文件：

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

2. **运行 Warden**（在 ONLY_LOCAL 模式下 Redis 是可选的）

```bash
# 在项目根目录执行（默认禁用 Redis）
go run . \
  --port 8081 \
  --mode ONLY_LOCAL

# 或者设置 Redis 地址（会自动启用 Redis，无需额外设置 --redis-enabled）
go run . \
  --port 8081 \
  --redis localhost:6379 \
  --mode ONLY_LOCAL
```

**注意**: 如果需要使用 Redis，请先启动它：
```bash
# 使用 Docker 启动 Redis（最简单）
docker run -d --name redis -p 6379:6379 redis:7.4-alpine

# 或使用本地 Redis
redis-server
```

4. **测试服务**

```bash
# 获取用户列表（需要设置 API Key）
curl -H "X-API-Key: your-api-key" http://localhost:8081/

# 健康检查（不需要 API Key）
curl http://localhost:8081/health
```

### 方式二：使用 Docker Compose

1. **准备数据文件**

将示例数据文件复制到当前目录：

```bash
cp ../../data.example.json ./data.json
```

2. **创建环境变量文件 `.env`**

```env
PORT=8081
REDIS=warden-redis:6379
MERGE_MODE=ONLY_LOCAL
API_KEY=your-secret-api-key-here

# 可选：Docker 镜像配置
# WARDEN_IMAGE=ghcr.io/soulteary/warden
# WARDEN_IMAGE_TAG=latest
```

3. **启动服务**

```bash
# 拉取最新镜像（可选，docker-compose 会自动拉取）
docker-compose pull

# 启动服务
docker-compose up -d
```

4. **测试服务**

```bash
# 获取用户列表
curl -H "X-API-Key: your-secret-api-key-here" http://localhost:8081/

# 健康检查
curl http://localhost:8081/health
```

## 📝 配置说明

### 运行模式

本示例使用 `ONLY_LOCAL` 模式，表示：
- ✅ 仅从本地 `data.json` 文件读取数据
- ❌ 不使用远程 API
- ⚠️  **Redis 默认禁用**（如果显式设置了 `REDIS` 地址，则会自动启用 Redis）
- ✅ 如果启用 Redis，数据会缓存在 Redis 中以提高性能

### 数据文件格式

`data.json` 文件必须是 JSON 数组格式，每个元素包含：
- `phone`: 手机号（字符串）
- `mail`: 邮箱地址（字符串）

示例：
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

## 🔍 验证服务

### 1. 检查服务状态

```bash
curl http://localhost:8081/health
```

预期响应：
```json
{
    "status": "ok",
    "service": "warden"
}
```

开发/测试环境还会包含各项检查详情；生产环境隐藏详情，只返回汇总状态和服务名。

### 2. 获取用户列表

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8081/
```

预期响应：
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

### 3. 分页查询

```bash
curl -H "X-API-Key: your-api-key" "http://localhost:8081/?page=1&page_size=1"
```

预期响应：
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
        "page_size": 1,
        "total": 2,
        "total_pages": 2
    }
}
```

## 🛠️ 常见问题

### Q: 为什么需要 Redis？

A: Warden 使用 Redis 进行：
- 数据缓存（提高性能）
- 分布式锁（防止定时任务重复执行）
- 多实例数据同步

即使只使用本地文件，Redis 也是必需的。

### Q: 如何修改数据？

A: 修改 `data.json` 文件后，服务会在下次定时任务执行时自动加载（默认每 5 秒）。你也可以重启服务立即生效。

### Q: 如何设置 API Key？

A: 通过环境变量设置：
```bash
export API_KEY=your-secret-api-key-here
go run . --port 8081 --redis localhost:6379 --mode ONLY_LOCAL
```

## 📚 下一步

- 查看 [复杂示例](../advanced/README.md) 了解如何使用远程 API
- 阅读 [完整文档](../../README.md) 了解更多功能
- 查看 [API 文档](../../openapi.yaml) 了解所有 API 端点

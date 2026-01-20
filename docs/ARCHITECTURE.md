# 架构设计文档

> 🌐 **Language / 语言**: [English](ARCHITECTURE.en.md) | [中文](ARCHITECTURE.md)

本文档详细说明 Warden 的系统架构、核心组件和数据流程。

## 系统架构图

```mermaid
graph TB
    subgraph "客户端层"
        Client[HTTP 客户端]
    end

    subgraph "Warden 服务"
        subgraph "HTTP 层"
            Router[路由处理器]
            Middleware[中间件层]
            RateLimit[速率限制]
            Compress[压缩中间件]
            Metrics[指标收集]
        end

        subgraph "业务层"
            UserCache[内存缓存<br/>SafeUserCache]
            RedisCache[Redis 缓存<br/>RedisUserCache]
            Parser[数据解析器]
            Scheduler[定时调度器<br/>gocron]
        end

        subgraph "基础设施层"
            Logger[日志系统<br/>zerolog]
            Prometheus[Prometheus 指标]
            RedisLock[分布式锁<br/>Redis Lock]
        end
    end

    subgraph "数据源"
        LocalFile[本地数据文件<br/>data.json]
        RemoteAPI[远程数据 API]
    end

    subgraph "外部服务"
        Redis[(Redis 服务器)]
    end

    Client -->|HTTP 请求| Router
    Router --> Middleware
    Middleware --> RateLimit
    Middleware --> Compress
    Middleware --> Metrics
    Router --> UserCache
    UserCache -->|读取| RedisCache
    RedisCache --> Redis
    Scheduler -->|定时触发| Parser
    Parser -->|读取| LocalFile
    Parser -->|请求| RemoteAPI
    Parser -->|更新| UserCache
    Parser -->|更新| RedisCache
    Scheduler -->|获取锁| RedisLock
    RedisLock --> Redis
    Router --> Logger
    Metrics --> Prometheus
```

## 核心组件

1. **HTTP 服务器**: 提供 JSON API 接口返回用户列表
   - 支持分页查询
   - 压缩响应数据
   - 速率限制保护
   - 请求指标收集

2. **数据解析器**: 支持从本地文件和远程 API 解析用户数据
   - 本地文件解析（JSON 格式）
   - 远程 API 调用（支持认证）
   - 多种数据合并策略

3. **定时调度器**: 使用 gocron 定期更新用户数据
   - 可配置的更新间隔
   - 基于 Redis 的分布式锁
   - 防止重复执行

4. **缓存系统**: 多级缓存架构
   - 内存缓存（SafeUserCache）：快速响应
   - Redis 缓存（RedisUserCache）：持久化存储
   - 智能缓存更新策略

5. **日志系统**: 基于 zerolog 的结构化日志记录
   - 结构化日志输出
   - 可动态调整日志级别
   - 访问日志和错误日志

6. **监控系统**: Prometheus 指标收集
   - HTTP 请求指标
   - 缓存命中率
   - 后台任务执行情况

## 数据流程

### 启动时数据加载流程

```mermaid
sequenceDiagram
    participant App as 应用程序
    participant Redis as Redis 缓存
    participant Remote as 远程 API
    participant Local as 本地文件
    participant Memory as 内存缓存

    App->>Redis: 1. 尝试从 Redis 加载
    alt Redis 有数据
        Redis-->>App: 返回缓存数据
        App->>Memory: 加载到内存
    else Redis 无数据
        App->>Remote: 2. 尝试从远程 API 加载
        alt 远程 API 成功
            Remote-->>App: 返回用户数据
            App->>Memory: 加载到内存
            App->>Redis: 更新 Redis 缓存
        else 远程 API 失败
            App->>Local: 3. 从本地文件加载
            Local-->>App: 返回用户数据
            App->>Memory: 加载到内存
            App->>Redis: 更新 Redis 缓存
        end
    end
```

### 定时任务更新流程

```mermaid
sequenceDiagram
    participant Scheduler as 定时调度器
    participant Lock as 分布式锁
    participant Parser as 数据解析器
    participant Remote as 远程 API
    participant Local as 本地文件
    participant Memory as 内存缓存
    participant Redis as Redis 缓存

    Scheduler->>Lock: 1. 尝试获取分布式锁
    alt 获取锁成功
        Lock-->>Scheduler: 锁获取成功
        Scheduler->>Parser: 2. 触发数据更新
        Parser->>Remote: 请求远程 API
        alt 远程 API 成功
            Remote-->>Parser: 返回数据
        else 远程 API 失败
            Parser->>Local: 回退到本地文件
            Local-->>Parser: 返回数据
        end
        Parser->>Parser: 3. 应用合并策略
        Parser->>Parser: 4. 计算数据哈希
        alt 数据有变化
            Parser->>Memory: 5. 更新内存缓存
            Parser->>Redis: 6. 更新 Redis 缓存
            Redis-->>Parser: 更新成功
        else 数据无变化
            Parser->>Parser: 跳过更新
        end
        Scheduler->>Lock: 7. 释放锁
    else 获取锁失败
        Lock-->>Scheduler: 其他实例正在执行
        Scheduler->>Scheduler: 跳过本次执行
    end
```

### 请求处理流程

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant RateLimit as 速率限制
    participant Compress as 压缩中间件
    participant Router as 路由处理器
    participant Cache as 内存缓存
    participant Metrics as 指标收集

    Client->>RateLimit: 1. HTTP 请求
    alt 超过速率限制
        RateLimit-->>Client: 429 Too Many Requests
    else 通过速率限制
        RateLimit->>Compress: 2. 转发请求
        Compress->>Router: 3. 处理请求
        Router->>Cache: 4. 读取缓存数据
        Cache-->>Router: 返回用户数据
        Router->>Router: 5. 应用分页（如需要）
        Router->>Metrics: 6. 记录指标
        Router->>Compress: 7. 返回响应
        Compress->>Compress: 8. 压缩响应
        Compress->>Client: 9. 返回 JSON 响应
    end
```

## 数据合并策略

系统支持 6 种数据合并模式，根据 `MODE` 参数选择：

| 模式 | 说明 | 使用场景 |
|------|------|----------|
| `DEFAULT` / `REMOTE_FIRST` | 远程优先，远程数据不存在时使用本地数据补充 | 默认模式，适合大多数场景 |
| `ONLY_REMOTE` | 仅使用远程数据源 | 完全依赖远程配置 |
| `ONLY_LOCAL` | 仅使用本地配置文件 | 离线环境或测试环境 |
| `LOCAL_FIRST` | 本地优先，本地数据不存在时使用远程数据补充 | 本地配置为主，远程为辅 |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | 远程优先，允许远程失败时回退到本地 | 高可用场景 |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | 本地优先，允许远程失败时回退到本地 | 混合模式 |

详细说明请参考 [配置文档](CONFIGURATION.md)。

## Redis Fallback 和可选支持架构

### Redis 启用状态架构图

```mermaid
graph TB
    App[App 初始化] --> CheckRedis{Redis 启用?}
    CheckRedis -->|是| TryConnect[尝试连接 Redis]
    CheckRedis -->|否| MemoryOnly[仅内存模式]
    TryConnect --> ConnectSuccess{连接成功?}
    ConnectSuccess -->|是| RedisMode[Redis + 内存模式]
    ConnectSuccess -->|否| Fallback[Fallback 到内存模式]
    
    RedisMode --> RedisCache[RedisUserCache]
    RedisMode --> DistLock[Redis 分布式锁]
    Fallback --> MemoryCache[SafeUserCache]
    Fallback --> LocalLock[本地锁]
    MemoryOnly --> MemoryCache
    MemoryOnly --> LocalLock
    
    RedisCache --> DataLoad[数据加载]
    MemoryCache --> DataLoad
    DistLock --> Scheduler[定时任务调度器]
    LocalLock --> Scheduler
```

### 设计说明

#### 1. Redis 启用状态

应用支持三种 Redis 状态：

- **启用且可用** (`redis-enabled=true` 且连接成功)
  - 使用 Redis 缓存和分布式锁
  - 数据加载优先级：Redis 缓存 > 远程 API > 本地文件

- **启用但不可用** (`redis-enabled=true` 但连接失败)
  - 自动降级到内存模式（fallback）
  - 使用本地锁替代分布式锁
  - 数据加载优先级：远程 API > 本地文件

- **禁用** (`redis-enabled=false`)
  - 跳过 Redis 初始化
  - 使用内存缓存和本地锁
  - 数据加载优先级：远程 API > 本地文件

#### 2. 锁实现

- **Redis 分布式锁** (`cache.Locker`)
  - 适用于多实例部署
  - 基于 Redis SETNX 实现
  - 支持自动过期，防止死锁

- **本地锁** (`cache.LocalLocker`)
  - 适用于单机部署
  - 基于 `sync.Mutex` 实现
  - 进程退出时自动释放

#### 3. 数据加载策略

数据加载采用多级降级策略：

1. **Redis 缓存**（如果 Redis 可用）
2. **远程 API**（如果配置了远程地址）
3. **本地文件**（`data.json`）

#### 4. 健康检查状态

健康检查端点 (`/health`) 返回 Redis 状态：

- `"ok"`: Redis 正常
- `"unavailable"`: Redis 连接失败（fallback 模式）或 Redis 客户端为 nil
- `"disabled"`: Redis 被显式禁用

### 配置参数

### 命令行参数

```bash
--redis-enabled=true|false  # 启用/禁用 Redis（默认: true）
```

### 环境变量

```bash
REDIS_ENABLED=true|false|1|0  # 启用/禁用 Redis（默认: true）
```

### 优先级

命令行参数 > 环境变量 > 配置文件 > 默认值

### 使用示例

### 禁用 Redis

```bash
# 命令行
go run main.go --redis-enabled=false

# 环境变量
export REDIS_ENABLED=false
go run main.go
```

### 启用 Redis（默认）

```bash
go run main.go --redis localhost:6379
```

### Redis 连接失败时自动 fallback

```bash
# Redis 不可用，但应用仍能启动
go run main.go --redis invalid-host:6379
# 会记录警告，但继续使用内存缓存
```

### 注意事项

1. **性能影响**：内存模式下，多实例部署时数据不同步，适合单机部署
2. **数据持久化**：禁用 Redis 后，数据仅存在内存中，重启后丢失
3. **分布式锁**：本地锁仅适用于单机部署，多实例时无法防止重复执行
4. **日志记录**：Redis 不可用时应记录清晰的警告日志，便于运维排查

## 相关文档

- [配置文档](CONFIGURATION.md) - 了解详细的配置选项
- [部署文档](DEPLOYMENT.md) - 了解部署架构
- [开发文档](DEVELOPMENT.md) - 了解开发相关架构

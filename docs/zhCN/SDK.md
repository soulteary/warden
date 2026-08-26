# SDK 使用文档

> 🌐 **Language / 语言**: [English](../enUS/SDK.md) | [中文](SDK.md) | [Français](../frFR/SDK.md) | [Italiano](../itIT/SDK.md) | [日本語](../jaJP/SDK.md) | [Deutsch](../deDE/SDK.md) | [한국어](../koKR/SDK.md)

Warden 提供了 Go SDK，方便其他项目集成使用。SDK 提供了简洁的 API 接口，支持缓存、认证等功能。

## 功能特性

- 🚀 **简单易用**: 提供简洁的 API 接口
- ⚡ **高性能**: 内置缓存支持（GetUsers），直接查询（GetUserByIdentifier）减少 API 调用
- 🔒 **安全**: 支持 API Key 认证，错误处理不泄露敏感信息
- 📦 **灵活**: 可配置的超时时间、缓存 TTL 等
- 🔌 **可扩展**: 支持自定义日志实现
- 🎯 **智能回退**: CheckUserInList 支持 phone 未找到时自动回退到 mail

## 安装

```bash
go get github.com/soulteary/warden/pkg/warden
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // 创建客户端选项
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)
    
    // 创建客户端
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    // 获取用户列表
    ctx := context.Background()
    users, err := client.GetUsers(ctx)
    if err != nil {
        panic(err)
    }
    
    // 检查用户是否在列表中（可以只提供 phone 或 mail，或同时提供）
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list and active")
    }
    
    // 也可以只使用 phone 或 mail
    existsByPhone := client.CheckUserInList(ctx, "13800138000", "")
    existsByMail := client.CheckUserInList(ctx, "", "user@example.com")
    
    // 获取用户详细信息
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
}
```

### 使用自定义日志

SDK 支持自定义日志实现。例如，使用 logrus:

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    logger := logrus.StandardLogger()
    
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithLogger(warden.NewLogrusAdapter(logger))
    
    client, err := warden.NewClient(opts)
    // ...
}
```

### 分页查询

```go
// 获取分页用户列表
resp, err := client.GetUsersPaginated(ctx, 1, 10) // 第1页，每页10条
if err != nil {
    panic(err)
}

fmt.Printf("Total users: %d\n", resp.Pagination.Total)
fmt.Printf("Total pages: %d\n", resp.Pagination.TotalPages)
for _, user := range resp.Data {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
}
```

### 获取单个用户信息

```go
// 通过手机号获取用户信息
user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
        println("User not found")
    } else {
        panic(err)
    }
} else {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
    if user.IsActive() {
        println("User is active")
    }
}

// 通过邮箱获取用户信息
user, err = client.GetUserByIdentifier(ctx, "", "user@example.com", "")

// 通过用户ID获取用户信息
user, err = client.GetUserByIdentifier(ctx, "", "", "user123")
```

### 清除缓存

```go
// 手动清除客户端缓存
client.ClearCache()

// 或使用别名
client.InvalidateCache()
```

### 自定义 HTTP Transport

```go
import "net/http"

// 创建自定义 transport
customTransport := &http.Transport{
    MaxIdleConns: 100,
    IdleConnTimeout: 90 * time.Second,
}

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithTransport(customTransport)

client, err := warden.NewClient(opts)
```

### 重试配置

```go
// 配置重试选项
retryOpts := warden.DefaultRetryOptions()
retryOpts.MaxRetries = 3
retryOpts.RetryDelay = 100 * time.Millisecond
retryOpts.MaxRetryDelay = 5 * time.Second
retryOpts.BackoffMultiplier = 2.0

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithRetry(retryOpts)

client, err := warden.NewClient(opts)
```

### 事件驱动缓存失效

```go
// 创建缓存失效事件通道
invalidationCh := make(chan struct{}, 1)

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithCacheInvalidationChannel(invalidationCh)

client, err := warden.NewClient(opts)
if err != nil {
    panic(err)
}
defer client.Close() // 重要：关闭以停止后台监听器

// 稍后，从外部事件触发缓存失效
invalidationCh <- struct{}{}

// 当接收到信号时，缓存会自动清除
```

## API 参考

### Options

`Options` 结构体用于配置客户端：

- `BaseURL`: Warden 服务地址（必需）
- `APIKey`: API Key（可选）
- `Timeout`: HTTP 请求超时时间（默认 10 秒）
- `CacheTTL`: 缓存 TTL（默认 5 分钟）
- `Logger`: 日志接口（可选，默认使用 NoOpLogger）
- `Transport`: 自定义 HTTP transport（可选）
- `Retry`: 重试配置（可选，默认不重试）
- `CacheInvalidationChannel`: 事件驱动缓存失效通道（可选）

### Client 方法

#### `NewClient(opts *Options) (*Client, error)`

创建新的 Warden 客户端。

#### `GetUsers(ctx context.Context) ([]AllowListUser, error)`

获取所有用户列表。如果缓存有效，会直接返回缓存的数据。

#### `GetUsersPaginated(ctx context.Context, page, pageSize int) (*PaginatedResponse, error)`

获取分页用户列表。

- `page`: 页码（从 1 开始）
- `pageSize`: 每页大小

返回 `PaginatedResponse`，包含：
- `Data`: 用户列表
- `Pagination`: 分页信息（页码、每页大小、总数、总页数）

**注意：** 此方法不使用缓存，每次调用都会从 API 获取最新数据。

#### `GetUserByIdentifier(ctx context.Context, phone, mail, userID string) (*AllowListUser, error)`

根据标识符获取单个用户信息。

- `phone`: 用户手机号（可选，但必须提供 phone、mail 或 userID 中的一个）
- `mail`: 用户邮箱（可选）
- `userID`: 用户唯一标识符（可选）

**重要：** 必须且只能提供 `phone`、`mail` 或 `userID` 中的一个标识符。

返回 `*AllowListUser` 和错误。如果用户不存在，返回 `ErrCodeNotFound` 错误。

**注意：** 此方法不使用缓存，每次调用都会从 API 获取最新数据。

#### `CheckUserInList(ctx context.Context, phone, mail string) bool`

检查用户是否在允许列表中。

- `phone`: 用户手机号（可选）
- `mail`: 用户邮箱（可选）

如果用户存在（通过手机号或邮箱匹配），返回 `true`；否则返回 `false`。

**行为说明：**
- 如果同时提供 `phone` 和 `mail`，优先使用 `phone` 进行查找
- 如果 `phone` 查找失败（返回 `NotFound` 错误），且 `mail` 不为空，会自动回退到使用 `mail` 进行查找
- 如果 `phone` 查找成功但用户状态不活跃，不会回退到 `mail`（因为已经找到了用户）
- 如果 `phone` 查找失败且错误不是 `NotFound`（如网络错误），不会回退到 `mail`
- 输入会自动规范化：`phone` 会去除首尾空格，`mail` 会去除首尾空格并转换为小写
- 此方法使用 `GetUserByIdentifier` 进行查找，性能优于遍历用户列表
- 只有状态为 "active" 的用户才会返回 `true`

#### `ClearCache()`

清除客户端内部缓存。

#### `InvalidateCache()`

`ClearCache()` 的别名，用于与事件驱动失效保持一致。

#### `Close()`

停止后台 goroutine（如缓存失效监听器）并释放资源。
当客户端不再需要时应调用此方法。

## 类型定义

### AllowListUser

```go
type AllowListUser struct {
    Phone  string   `json:"phone"`   // 用户手机号
    Mail   string   `json:"mail"`    // 用户邮箱地址
    UserID string   `json:"user_id"` // 用户唯一标识符（可选，未提供时自动生成）
    Status string   `json:"status"`  // 用户状态（如 "active", "inactive", "suspended"）
    Scope  []string `json:"scope"`   // 用户权限范围（可选）
    Role   string   `json:"role"`    // 用户角色（可选）
}
```

**方法：**
- `IsActive() bool`: 检查用户状态是否为 "active"
- `IsValid() bool`: 检查用户状态是否为有效状态（当前仅支持 "active"）

### PaginatedResponse

```go
type PaginatedResponse struct {
    Data       []AllowListUser `json:"data"`
    Pagination PaginationInfo  `json:"pagination"`
}

type PaginationInfo struct {
    Page       int `json:"page"`        // 当前页码（从 1 开始）
    PageSize   int `json:"page_size"`   // 每页大小
    Total      int `json:"total"`        // 总记录数
    TotalPages int `json:"total_pages"` // 总页数
}
```

## 错误处理

SDK 使用自定义错误类型，包含错误代码和详细信息：

```go
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok {
        switch sdkErr.Code {
        case warden.ErrCodeUnauthorized:
            // 处理认证错误
        case warden.ErrCodeRequestFailed:
            // 处理请求失败
        case warden.ErrCodeNotFound:
            // 处理未找到错误
        case warden.ErrCodeServerError:
            // 处理服务器错误
        // ...
        }
    }
}
```

### 错误代码

- `ErrCodeInvalidConfig`: 配置无效
- `ErrCodeRequestFailed`: 请求失败
- `ErrCodeInvalidResponse`: 响应格式无效
- `ErrCodeUnauthorized`: 未授权
- `ErrCodeNotFound`: 未找到
- `ErrCodeServerError`: 服务器错误

## 最佳实践

1. **复用客户端**: 创建一次客户端，在整个应用生命周期中复用
2. **合理设置缓存 TTL**: 根据数据更新频率设置合适的缓存时间
3. **使用 Context**: 传递 context 以支持取消和超时控制
4. **错误处理**: 始终检查并处理错误
5. **日志记录**: 在生产环境中使用合适的日志实现
6. **关闭客户端**: 当客户端不再需要时调用 `Close()` 以停止后台 goroutine
7. **配置重试**: 在生产环境中启用重试以处理临时故障
8. **自定义 Transport**: 在高级场景中使用自定义 transport（TLS、代理、连接池等）

## 设计说明

### 设计原则

1. **简单易用**：提供简洁的 API 接口
2. **高性能**：内置缓存支持，减少 API 调用
3. **线程安全**：所有方法都是并发安全的
4. **灵活配置**：支持自定义超时、缓存、日志等

### 架构设计

#### 核心组件

1. **Client**：HTTP 客户端封装
2. **Cache**：线程安全的内存缓存
3. **Options**：配置选项（使用 Builder 模式）
4. **Logger**：日志接口（支持不同日志库）

#### 并发安全

- `http.Client` 是并发安全的
- `Cache` 使用 `sync.RWMutex` 保证线程安全
- `Client` 的所有字段在创建后都是只读的
- 所有方法都是线程安全的，可以在多个 goroutine 中并发调用

#### 缓存策略

1. **GetUsers()**：使用缓存
   - 首先检查缓存
   - 如果缓存有效，直接返回
   - 如果缓存无效或不存在，从 API 获取并更新缓存

2. **GetUsersPaginated()**：不使用缓存
   - 原因：不同的分页参数会产生不同的结果
   - 如果实现分页缓存，需要按分页参数缓存，复杂度较高
   - 当前设计：每次都从 API 获取，保证数据准确性

3. **GetUserByIdentifier()**：不使用缓存
   - 原因：需要获取最新的单个用户信息，保证数据实时性
   - 每次调用都会从 API 获取，避免缓存导致的数据不一致

4. **CheckUserInList()**：不使用缓存
   - 使用 `GetUserByIdentifier()` 直接查询单个用户
   - 每次调用都会发起 API 请求，保证数据实时性
   - 支持智能回退：当 phone 查找失败（NotFound）且 mail 不为空时，自动回退到 mail 查找
   - 性能优化：直接查询单个用户，比遍历整个用户列表更高效

#### CheckUserInList 实现策略

`CheckUserInList()` 方法采用以下策略：

1. **输入规范化**：自动去除 phone 和 mail 的首尾空格，并将 mail 转换为小写
2. **优先级策略**：如果同时提供 phone 和 mail，优先使用 phone 查找
3. **智能回退**：
   - 当 phone 查找返回 `NotFound` 错误时，如果 mail 不为空，自动回退到 mail 查找
   - 当 phone 查找成功但用户状态不活跃时，不回退到 mail（因为已找到用户）
   - 当 phone 查找遇到其他错误（如网络错误）时，不回退到 mail
4. **状态验证**：只有状态为 "active" 的用户才会返回 `true`
5. **性能优化**：使用 `GetUserByIdentifier()` 直接查询，避免获取整个用户列表

### RetryOptions

`RetryOptions` 结构体配置重试行为：

- `MaxRetries`: 最大重试次数（默认 0，不重试）
- `RetryDelay`: 重试之间的初始延迟（默认 100ms）
- `MaxRetryDelay`: 重试之间的最大延迟（默认 5s）
- `BackoffMultiplier`: 指数退避乘数（默认 2.0）
- `RetryableStatusCodes`: 触发重试的 HTTP 状态码（默认：5xx）

**注意：** 网络错误总是可重试的。客户端错误（4xx）从不重试。

### 已知限制

1. **分页缓存**：`GetUsersPaginated()` 不使用缓存
   - 这是有意的设计，保证数据准确性
   - 如需分页缓存，可以实现更复杂的缓存策略

2. **单用户查询缓存**：`GetUserByIdentifier()` 和 `CheckUserInList()` 不使用缓存
   - 这是有意的设计，保证数据实时性
   - 如需缓存，可以实现基于用户标识符的缓存策略

### 未来改进方向

1. 支持请求/响应中间件
2. 支持指标收集（metrics）
3. 支持连接池配置
4. 支持熔断器模式

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // 创建客户端
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)

    client, err := warden.NewClient(opts)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // 获取所有用户
    users, err := client.GetUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total users: %d\n", len(users))

    // 通过手机号获取单个用户
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            fmt.Println("User not found")
        } else {
            log.Fatal(err)
        }
    } else {
        fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
    }

    // 分页查询
    result, err := client.GetUsersPaginated(ctx, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page 1: %d users\n", len(result.Data))

    // 检查用户
    exists := client.CheckUserInList(ctx, "13800138000", "admin@example.com")
    fmt.Printf("User exists and active: %v\n", exists)

    // 清除缓存
    client.ClearCache()
    fmt.Println("Cache cleared")
}
```

## 相关文档

- [API 文档](API.md) - 了解 API 端点详情
- [配置文档](CONFIGURATION.md) - 了解服务器配置选项

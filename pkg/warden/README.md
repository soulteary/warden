# Warden SDK

Warden SDK 是一个用于与 Warden API 交互的 Go 客户端库。它提供了简单易用的接口来获取用户列表、检查用户是否在允许列表中，并支持缓存以提高性能。

## 功能特性

- 🚀 **简单易用**: 提供简洁的 API 接口
- ⚡ **高性能**: 内置缓存支持，减少 API 调用
- 🔒 **安全**: 支持 API Key 认证
- 📦 **灵活**: 可配置的超时时间、缓存 TTL 等
- 🔌 **可扩展**: 支持自定义日志实现

## 安装

```bash
go get soulteary.com/soulteary/warden/pkg/warden
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "time"
    
    "soulteary.com/soulteary/warden/pkg/warden"
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
    
    // 检查用户是否在列表中
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list")
    }
}
```

### 使用自定义日志

SDK 支持自定义日志实现。例如，使用 logrus:

```go
import (
    "github.com/sirupsen/logrus"
    "soulteary.com/soulteary/warden/pkg/warden"
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
    fmt.Printf("Phone: %s, Mail: %s\n", user.Phone, user.Mail)
}
```

### 清除缓存

```go
// 清除客户端缓存
client.ClearCache()
```

## API 参考

### Options

`Options` 结构体用于配置客户端：

- `BaseURL`: Warden 服务地址（必需）
- `APIKey`: API Key（可选）
- `Timeout`: HTTP 请求超时时间（默认 10 秒）
- `CacheTTL`: 缓存 TTL（默认 5 分钟）
- `Logger`: 日志接口（可选，默认使用 NoOpLogger）

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

#### `CheckUserInList(ctx context.Context, phone, mail string) bool`

检查用户是否在允许列表中。

- `phone`: 用户手机号（可选）
- `mail`: 用户邮箱（可选）

如果用户存在（通过手机号或邮箱匹配），返回 `true`；否则返回 `false`。

#### `ClearCache()`

清除客户端内部缓存。

## 类型定义

### AllowListUser

```go
type AllowListUser struct {
    Phone string `json:"phone"` // 用户手机号
    Mail  string `json:"mail"`  // 用户邮箱地址
}
```

### PaginatedResponse

```go
type PaginatedResponse struct {
    Data       []AllowListUser `json:"data"`
    Pagination PaginationInfo  `json:"pagination"`
}

type PaginationInfo struct {
    Page       int `json:"page"`        // 当前页码（从 1 开始）
    PageSize   int `json:"page_size"`    // 每页大小
    Total      int `json:"total"`        // 总记录数
    TotalPages int `json:"total_pages"`  // 总页数
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

## 示例

完整示例请参考 [example](../example) 目录。

## 许可证

MIT License
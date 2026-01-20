# SDK 使用文档

> 🌐 **Language / 语言**: [English](SDK.en.md) | [中文](SDK.md)

Warden 提供了 Go SDK，方便其他项目集成使用。SDK 提供了简洁的 API 接口，支持缓存、认证等功能。

## 安装 SDK

```bash
go get github.com/soulteary/warden/pkg/warden
```

## 快速开始

```go
package main

import (
    "context"
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

    // 检查用户是否在列表中
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list")
    }
}
```

## 主要功能

### 获取用户列表

```go
// 获取所有用户（支持缓存）
users, err := client.GetUsers(ctx)
if err != nil {
    // 处理错误
}

// 遍历用户
for _, user := range users {
    fmt.Printf("Phone: %s, Mail: %s\n", user.Phone, user.Mail)
}
```

### 分页查询

```go
// 获取分页用户列表
page := 1
pageSize := 100
result, err := client.GetUsersPaginated(ctx, page, pageSize)
if err != nil {
    // 处理错误
}

fmt.Printf("Total: %d, Page: %d/%d\n", 
    result.Pagination.Total, 
    result.Pagination.Page, 
    result.Pagination.TotalPages)

for _, user := range result.Data {
    fmt.Printf("Phone: %s, Mail: %s\n", user.Phone, user.Mail)
}
```

### 用户检查

```go
// 检查用户是否在允许列表中
exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
if exists {
    println("User is in the allow list")
} else {
    println("User is not in the allow list")
}
```

### 缓存管理

```go
// 清除客户端缓存
client.ClearCache()
```

## 客户端选项

### 基本配置

```go
opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithAPIKey("your-api-key").
    WithTimeout(10 * time.Second)
```

### 缓存配置

```go
opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithCacheTTL(5 * time.Minute)  // 设置缓存 TTL
```

### 自定义 HTTP 客户端

```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns: 100,
    },
}

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithHTTPClient(httpClient)
```

## 使用自定义日志

SDK 支持自定义日志实现。例如，使用 logrus:

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/soulteary/warden/pkg/warden"
)

logger := logrus.StandardLogger()
opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithLogger(warden.NewLogrusAdapter(logger))
```

## 错误处理

SDK 返回的错误实现了 `error` 接口，可以检查错误类型：

```go
users, err := client.GetUsers(ctx)
if err != nil {
    // 检查是否是网络错误
    if netErr, ok := err.(net.Error); ok {
        fmt.Printf("Network error: %v\n", netErr)
    }
    
    // 检查是否是 HTTP 错误
    if httpErr, ok := err.(*warden.HTTPError); ok {
        fmt.Printf("HTTP error: %d %s\n", httpErr.StatusCode, httpErr.Message)
    }
    
    return err
}
```

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

    // 分页查询
    result, err := client.GetUsersPaginated(ctx, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page 1: %d users\n", len(result.Data))

    // 检查用户
    exists := client.CheckUserInList(ctx, "13800138000", "admin@example.com")
    fmt.Printf("User exists: %v\n", exists)

    // 清除缓存
    client.ClearCache()
    fmt.Println("Cache cleared")
}
```

## 详细文档

更多使用说明和 API 参考，请查看 [SDK 文档](../pkg/warden/README.md)（如果存在）。

## 相关文档

- [API 文档](API.md) - 了解 API 端点详情
- [配置文档](CONFIGURATION.md) - 了解服务器配置选项

# SDK Usage Documentation

> 🌐 **Language / 语言**: [English](../enUS/SDK.md) | [中文](../zhCN/SDK.md) | [Français](../frFR/SDK.md) | [Italiano](../itIT/SDK.md) | [日本語](../jaJP/SDK.md) | [Deutsch](SDK.md) | [한국어](../koKR/SDK.md)

Warden provides a Go SDK for easy integration into other projects. The SDK provides a clean API interface with support for caching, authentication, and more.

## Install SDK

```bash
go get github.com/soulteary/warden/pkg/warden
```

## Quick Start

```go
package main

import (
    "context"
    "time"

    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Create client options
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)

    // Create client
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }

    // Get user list
    ctx := context.Background()
    users, err := client.GetUsers(ctx)
    if err != nil {
        panic(err)
    }

    // Check if user is in the list
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list")
    }
}
```

## Main Features

### Get User List

```go
// Get all users (with caching support)
users, err := client.GetUsers(ctx)
if err != nil {
    // Handle error
}

// Iterate users
for _, user := range users {
    fmt.Printf("Phone: %s, Mail: %s\n", user.Phone, user.Mail)
}
```

### Paginated Query

```go
// Get paginated user list
page := 1
pageSize := 100
result, err := client.GetUsersPaginated(ctx, page, pageSize)
if err != nil {
    // Handle error
}

fmt.Printf("Total: %d, Page: %d/%d\n", 
    result.Pagination.Total, 
    result.Pagination.Page, 
    result.Pagination.TotalPages)

for _, user := range result.Data {
    fmt.Printf("Phone: %s, Mail: %s\n", user.Phone, user.Mail)
}
```

### User Check

```go
// Check if user is in the allow list
exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
if exists {
    println("User is in the allow list")
} else {
    println("User is not in the allow list")
}
```

### Clear Cache

```go
// Clear client cache manually
client.ClearCache()

// Or use the alias
client.InvalidateCache()
```

### Custom HTTP Transport

```go
import "net/http"

// Create custom transport
customTransport := &http.Transport{
    MaxIdleConns: 100,
    IdleConnTimeout: 90 * time.Second,
}

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithTransport(customTransport)

client, err := warden.NewClient(opts)
```

### Retry Configuration

```go
// Configure retry options
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

### Event-Driven Cache Invalidation

```go
// Create channel for cache invalidation events
invalidationCh := make(chan struct{}, 1)

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithCacheInvalidationChannel(invalidationCh)

client, err := warden.NewClient(opts)
if err != nil {
    panic(err)
}
defer client.Close() // Important: close to stop background listener

// Later, trigger cache invalidation from external event
invalidationCh <- struct{}{}

// Cache will be automatically cleared when signal is received
```

## Using Custom Logger

The SDK supports custom logger implementations. For example, using logrus:

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

## Error Handling

Errors returned by the SDK implement the `error` interface, and you can check error types:

```go
users, err := client.GetUsers(ctx)
if err != nil {
    // Check if it's a network error
    if netErr, ok := err.(net.Error); ok {
        fmt.Printf("Network error: %v\n", netErr)
    }
    
    // Check if it's an HTTP error
    if httpErr, ok := err.(*warden.HTTPError); ok {
        fmt.Printf("HTTP error: %d %s\n", httpErr.StatusCode, httpErr.Message)
    }
    
    return err
}
```

## Complete Example

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
    // Create client
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

    // Get all users
    users, err := client.GetUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total users: %d\n", len(users))

    // Paginated query
    result, err := client.GetUsersPaginated(ctx, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page 1: %d users\n", len(result.Data))

    // Check user
    exists := client.CheckUserInList(ctx, "13800138000", "admin@example.com")
    fmt.Printf("User exists: %v\n", exists)

    // Clear cache
    client.ClearCache()
    fmt.Println("Cache cleared")
    
    // Close client to stop background goroutines
    client.Close()
}
```

## Detailed Documentation

For source code and design documentation, please refer to:

- **[SDK Source Code](../../pkg/warden/)** - SDK source code directory
- **[SDK Design Documentation](../../pkg/warden/DESIGN.md)** - Design principles and implementation details

## Related Documentation

- [API Documentation](API.md) - Learn about API endpoint details
- [Configuration Documentation](CONFIGURATION.md) - Learn about server configuration options

# SDK Usage Documentation

> 🌐 **Language / 语言**: [English](SDK.md) | [中文](../zhCN/SDK.md) | [Français](../frFR/SDK.md) | [Italiano](../itIT/SDK.md) | [日本語](../jaJP/SDK.md) | [Deutsch](../deDE/SDK.md) | [한국어](../koKR/SDK.md)

Warden provides a Go SDK for easy integration into other projects. The SDK provides a clean API interface with support for caching, authentication, and more.

## Features

- 🚀 **Simple and Easy**: Provides clean API interfaces
- ⚡ **High Performance**: Built-in cache support (GetUsers), direct queries (GetUserByIdentifier) reduce API calls
- 🔒 **Secure**: Supports API Key authentication, error handling doesn't leak sensitive information
- 📦 **Flexible**: Configurable timeout, cache TTL, etc.
- 🔌 **Extensible**: Supports custom logger implementations
- 🎯 **Smart Fallback**: CheckUserInList supports automatic fallback to mail when phone is not found

## Install SDK

```bash
go get github.com/soulteary/warden/pkg/warden
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
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
    
    // Check if user is in the list (can provide phone, mail, or both)
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list and active")
    }
    
    // Can also use only phone or mail
    existsByPhone := client.CheckUserInList(ctx, "13800138000", "")
    existsByMail := client.CheckUserInList(ctx, "", "user@example.com")
    
    // Get user details
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
}
```

### Using Custom Logger

The SDK supports custom logger implementations. For example, using logrus:

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

### Paginated Query

```go
// Get paginated user list
resp, err := client.GetUsersPaginated(ctx, 1, 10) // Page 1, 10 items per page
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

### Get Single User Information

```go
// Get user information by phone
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

// Get user information by email
user, err = client.GetUserByIdentifier(ctx, "", "user@example.com", "")

// Get user information by user ID
user, err = client.GetUserByIdentifier(ctx, "", "", "user123")
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

## API Reference

### Options

The `Options` struct is used to configure the client:

- `BaseURL`: Warden service address (required)
- `APIKey`: API Key (optional)
- `Timeout`: HTTP request timeout (default 10 seconds)
- `CacheTTL`: Cache TTL (default 5 minutes)
- `Logger`: Logger interface (optional, defaults to NoOpLogger)
- `Transport`: Custom HTTP transport (optional)
- `Retry`: Retry configuration (optional, defaults to no retry)
- `CacheInvalidationChannel`: Channel for event-driven cache invalidation (optional)

### Client Methods

#### `NewClient(opts *Options) (*Client, error)`

Creates a new Warden client.

#### `GetUsers(ctx context.Context) ([]AllowListUser, error)`

Gets all user list. If cache is valid, returns cached data directly.

#### `GetUsersPaginated(ctx context.Context, page, pageSize int) (*PaginatedResponse, error)`

Gets paginated user list.

- `page`: Page number (starts from 1)
- `pageSize`: Page size

Returns `PaginatedResponse`, containing:
- `Data`: User list
- `Pagination`: Pagination information (page number, page size, total, total pages)

**Note:** This method does not use cache, each call fetches the latest data from the API.

#### `GetUserByIdentifier(ctx context.Context, phone, mail, userID string) (*AllowListUser, error)`

Gets a single user information by identifier.

- `phone`: User phone number (optional, but must provide one of phone, mail, or userID)
- `mail`: User email (optional)
- `userID`: User unique identifier (optional)

**Important:** Must provide exactly one identifier among `phone`, `mail`, or `userID`.

Returns `*AllowListUser` and error. If user does not exist, returns `ErrCodeNotFound` error.

**Note:** This method does not use cache, each call fetches the latest data from the API.

#### `CheckUserInList(ctx context.Context, phone, mail string) bool`

Checks if a user is in the allow list.

- `phone`: User phone number (optional)
- `mail`: User email (optional)

Returns `true` if the user exists (matched by phone or email), `false` otherwise.

**Behavior:**
- If both `phone` and `mail` are provided, `phone` takes priority
- If `phone` lookup fails (returns `NotFound` error) and `mail` is not empty, automatically falls back to `mail` lookup
- If `phone` lookup succeeds but user status is not active, does not fall back to `mail` (user already found)
- If `phone` lookup fails and error is not `NotFound` (e.g., network error), does not fall back to `mail`
- Input is automatically normalized: `phone` is trimmed, `mail` is trimmed and converted to lowercase
- This method uses `GetUserByIdentifier` for lookup, more efficient than iterating through user list
- Only users with status "active" will return `true`

#### `ClearCache()`

Clears the internal client cache.

#### `InvalidateCache()`

Alias for `ClearCache()` for consistency with event-driven invalidation.

#### `Close()`

Stops background goroutines (e.g., cache invalidation listener) and releases resources.
Should be called when the client is no longer needed.

## Type Definitions

### AllowListUser

```go
type AllowListUser struct {
    Phone  string   `json:"phone"`   // User phone number
    Mail   string   `json:"mail"`    // User email address
    UserID string   `json:"user_id"` // User unique identifier (optional, auto-generated if not provided)
    Status string   `json:"status"`  // User status (e.g., "active", "inactive", "suspended")
    Scope  []string `json:"scope"`   // User permission scope (optional)
    Role   string   `json:"role"`    // User role (optional)
}
```

**Methods:**
- `IsActive() bool`: Checks if user status is "active"
- `IsValid() bool`: Checks if user status is valid (currently only supports "active")

### PaginatedResponse

```go
type PaginatedResponse struct {
    Data       []AllowListUser `json:"data"`
    Pagination PaginationInfo  `json:"pagination"`
}

type PaginationInfo struct {
    Page       int `json:"page"`        // Current page number (starts from 1)
    PageSize   int `json:"page_size"`   // Page size
    Total      int `json:"total"`       // Total number of records
    TotalPages int `json:"total_pages"` // Total number of pages
}
```

## Error Handling

The SDK uses custom error types with error codes and detailed information:

```go
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok {
        switch sdkErr.Code {
        case warden.ErrCodeUnauthorized:
            // Handle authentication error
        case warden.ErrCodeRequestFailed:
            // Handle request failure
        case warden.ErrCodeNotFound:
            // Handle not found error
        case warden.ErrCodeServerError:
            // Handle server error
        // ...
        }
    }
}
```

### Error Codes

- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeRequestFailed`: Request failed
- `ErrCodeInvalidResponse`: Invalid response format
- `ErrCodeUnauthorized`: Unauthorized
- `ErrCodeNotFound`: Not found
- `ErrCodeServerError`: Server error

## Best Practices

1. **Reuse Client**: Create the client once and reuse it throughout the application lifecycle
2. **Set Cache TTL Appropriately**: Set appropriate cache time based on data update frequency
3. **Use Context**: Pass context to support cancellation and timeout control
4. **Error Handling**: Always check and handle errors
5. **Logging**: Use appropriate logger implementation in production environments
6. **Close Client**: Call `Close()` when the client is no longer needed to stop background goroutines
7. **Configure Retry**: Enable retry for production environments to handle transient failures
8. **Custom Transport**: Use custom transport for advanced scenarios (TLS, proxy, connection pooling, etc.)

## Design Documentation

### Design Principles

1. **Simple and Easy**: Provides clean API interfaces
2. **High Performance**: Built-in cache support, reduces API calls
3. **Thread Safe**: All methods are concurrency-safe
4. **Flexible Configuration**: Supports custom timeout, cache, logger, etc.

### Architecture Design

#### Core Components

1. **Client**: HTTP client wrapper
2. **Cache**: Thread-safe in-memory cache
3. **Options**: Configuration options (using Builder pattern)
4. **Logger**: Logger interface (supports different logging libraries)

#### Concurrency Safety

- `http.Client` is concurrency-safe
- `Cache` uses `sync.RWMutex` to ensure thread safety
- All fields of `Client` are read-only after creation
- All methods are thread-safe and can be called concurrently in multiple goroutines

#### Cache Strategy

1. **GetUsers()**: Uses cache
   - First checks cache
   - If cache is valid, returns directly
   - If cache is invalid or doesn't exist, fetches from API and updates cache

2. **GetUsersPaginated()**: Does not use cache
   - Reason: Different pagination parameters produce different results
   - If pagination cache is implemented, caching by pagination parameters is complex
   - Current design: Fetches from API each time to ensure data accuracy

3. **GetUserByIdentifier()**: Does not use cache
   - Reason: Needs to fetch the latest single user information to ensure data real-time
   - Each call fetches from API to avoid data inconsistency caused by cache

4. **CheckUserInList()**: Does not use cache
   - Uses `GetUserByIdentifier()` to directly query a single user
   - Each call makes an API request to ensure data real-time
   - Supports smart fallback: When phone lookup fails (NotFound) and mail is not empty, automatically falls back to mail lookup
   - Performance optimization: Direct query of a single user is more efficient than iterating through the entire user list

#### CheckUserInList Implementation Strategy

The `CheckUserInList()` method uses the following strategy:

1. **Input Normalization**: Automatically trims leading and trailing spaces from phone and mail, and converts mail to lowercase
2. **Priority Strategy**: If both phone and mail are provided, phone takes priority
3. **Smart Fallback**:
   - When phone lookup returns `NotFound` error, if mail is not empty, automatically falls back to mail lookup
   - When phone lookup succeeds but user status is not active, does not fall back to mail (user already found)
   - When phone lookup encounters other errors (e.g., network error), does not fall back to mail
4. **Status Validation**: Only users with status "active" will return `true`
5. **Performance Optimization**: Uses `GetUserByIdentifier()` for direct query, avoiding fetching the entire user list

### RetryOptions

The `RetryOptions` struct configures retry behavior:

- `MaxRetries`: Maximum number of retries (default 0, no retry)
- `RetryDelay`: Initial delay between retries (default 100ms)
- `MaxRetryDelay`: Maximum delay between retries (default 5s)
- `BackoffMultiplier`: Multiplier for exponential backoff (default 2.0)
- `RetryableStatusCodes`: HTTP status codes that trigger retry (default: 5xx)

**Note:** Network errors are always retryable. Client errors (4xx) are never retried.

### Known Limitations

1. **Pagination Cache**: `GetUsersPaginated()` does not use cache
   - This is an intentional design to ensure data accuracy
   - If pagination cache is needed, more complex caching strategies can be implemented

2. **Single User Query Cache**: `GetUserByIdentifier()` and `CheckUserInList()` do not use cache
   - This is an intentional design to ensure data real-time
   - If caching is needed, cache strategies based on user identifiers can be implemented

### Future Improvements

1. Support request/response middleware
2. Support metrics collection
3. Support connection pooling configuration
4. Support circuit breaker pattern

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

    // Get single user by phone
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

    // Paginated query
    result, err := client.GetUsersPaginated(ctx, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page 1: %d users\n", len(result.Data))

    // Check user
    exists := client.CheckUserInList(ctx, "13800138000", "admin@example.com")
    fmt.Printf("User exists and active: %v\n", exists)

    // Clear cache
    client.ClearCache()
    fmt.Println("Cache cleared")
}
```

## Related Documentation

- [API Documentation](API.md) - Learn about API endpoint details
- [Configuration Documentation](CONFIGURATION.md) - Learn about server configuration options

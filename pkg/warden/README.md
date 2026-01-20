# Warden SDK

> 📖 **Multi-language Documentation**: For documentation in other languages, please refer to [docs directory](../../docs/)

Warden SDK is a Go client library for interacting with the Warden API. It provides simple and easy-to-use interfaces for fetching user lists, querying individual user information, checking if users are in the allow list, and supports caching for improved performance.

## Features

- 🚀 **Simple and Easy**: Provides clean API interfaces
- ⚡ **High Performance**: Built-in cache support (GetUsers), direct queries (GetUserByIdentifier) reduce API calls
- 🔒 **Secure**: Supports API Key authentication, error handling doesn't leak sensitive information
- 📦 **Flexible**: Configurable timeout, cache TTL, etc.
- 🔌 **Extensible**: Supports custom logger implementations
- 🎯 **Smart Fallback**: CheckUserInList supports automatic fallback to mail when phone is not found

## Installation

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
// Clear client cache
client.ClearCache()
```

## API Reference

### Options

The `Options` struct is used to configure the client:

- `BaseURL`: Warden service address (required)
- `APIKey`: API Key (optional)
- `Timeout`: HTTP request timeout (default 10 seconds)
- `CacheTTL`: Cache TTL (default 5 minutes)
- `Logger`: Logger interface (optional, defaults to NoOpLogger)

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

## Examples

For complete examples, please refer to the [example](../example) directory.

## License

MIT License

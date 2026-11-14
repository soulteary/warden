# Warden SDK

> 📖 **Multi-language Documentation**: For documentation in other languages, please refer to [docs directory](../../docs/)

Warden SDK is a Go client library for interacting with the Warden API. It provides simple and easy-to-use interfaces for fetching user lists, querying individual user information, checking if users are in the allow list, and supports caching for improved performance.

## Features

- 🚀 **Simple and Easy**: Provides clean API interfaces
- ⚡ **High Performance**: Built-in cache support (GetUsers), direct queries (GetUserByIdentifier) reduce API calls
- 🔒 **Secure**: Supports API keys, HMAC and TLS client certificates; internal diagnostic errors are redacted (returned errors retain details; see SDK diagnostic logging)
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
- The status comes from the `/user` response. A server whose `RESPONSE_FIELDS` leaves `status` out returns none, and every user is then reported as `false`; the client logs a warning that names the cause

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

## RetryOptions

The `RetryOptions` struct configures retry behavior:

- `MaxRetries`: Maximum number of retries (default 0, no retry)
- `RetryDelay`: Initial delay between retries (default 100ms)
- `MaxRetryDelay`: Maximum delay between retries (default 5s)
- `BackoffMultiplier`: Multiplier for exponential backoff (default 2.0)
- `RetryableStatusCodes`: HTTP status codes that trigger retry (default: 500, 502, 503, 504)

**Note:** Retries require `MaxRetries > 0` and remain bounded by the request context. Transport errors are eligible for retry, except redirect-policy errors (`ErrRedirect`). Client errors (4xx) are never retried, even if listed in `RetryableStatusCodes`.

## Best Practices

1. **Reuse Client**: Create the client once and reuse it throughout the application lifecycle
2. **Set Cache TTL Appropriately**: Set appropriate cache time based on data update frequency
3. **Use Context**: Pass context to support cancellation and timeout control
4. **Error Handling**: Always check and handle errors
5. **Logging**: Use appropriate logger implementation in production environments
6. **Close Client**: Call `Close()` when the client is no longer needed to stop background goroutines
7. **Configure Retry**: Enable retry for production environments to handle transient failures
8. **Custom Transport**: Use custom transport for advanced scenarios (TLS, proxy, connection pooling, etc.)

## Examples

For complete examples, please refer to the [example](../../example) directory.

## License

[Apache License 2.0](../../LICENSE)

### List cache invalidation during a request

`ClearCache`, `InvalidateCache` and invalidation-channel events fence list requests
that started before the invalidation. Such a response is neither cached nor
returned by `GetUsers`; the call returns `REQUEST_FAILED` wrapping
`ErrCacheInvalidated` (detectable with `errors.Is`). The caller may start a new
request. There is no automatic additional fetch. Cache TTL, optional source
freshness and the normal request retry policy are unchanged.

### Concurrent list responses

Within one client and invalidation epoch, a later-started request that successfully
commits prevents earlier requests from publishing or returning their responses.
Those calls return `REQUEST_FAILED` wrapping `ErrCacheSuperseded`. Starting a later
request, or receiving an invalid or expired response, does not suppress an earlier
valid response. Empty lists participate in the same ordering.

This is local request ordering, not a global source-version guarantee. Snapshot
hashes and Redis revision tokens are not ordered, and sequential requests routed
to different replicas may still observe different accepted datasets. No extra
fetch, retry or change to default freshness settings is introduced.

### Source errors from an enforcing server

With `WithSourceFreshness` enabled, a metadata-v1 server can classify a source
refusal on HTTP 503. `errors.Is(err, ErrSourceStale)` identifies expiry;
`ErrSourceUnknown` identifies unavailable age evidence, and `ErrSourceFuture`
identifies future source time. These errors use `SOURCE_FRESHNESS` and retain the
original server error in their chain. Plain or unrecognized 503 responses keep
`SERVER_ERROR`. Default clients keep their existing mapping and cache behavior.
The existing retry policy runs before classification, including configured 503
retries. See the configuration guide for paired server and SDK enablement.

## Cache and source freshness contracts

- [Cache isolation and hit behavior](../../docs/enUS/SDK_CACHE_ISOLATION.md)
- [Optional source freshness and source-error classification](../../docs/enUS/SDK_SOURCE_FRESHNESS.md)


### Redirect policy

The SDK rejects HTTP redirects with `ErrRedirect`. This applies to all SDK calls
and all credentials, including API keys, HMAC signing and client certificates. A
redirect policy error is not retried. Set `BaseURL` to the final service URL;
redirect response bodies are not returned or cached.

### Credential selection

When HMAC signing is configured, the SDK sends only HMAC authentication headers.
A simultaneously configured API key is omitted from both `X-API-Key` and
`Authorization`. API-key-only clients continue to send both headers. This avoids
exposing an additional reusable credential on signed requests; it does not change
server authentication policies or TLS client certificate configuration.

### SDK diagnostic logging

The SDK uses `NoOpLogger` by default. With a custom logger, internal request
failure messages report SDK error categories and HTTP status codes without raw
transport error text, response bodies or retry URLs. This prevents those error
paths from exposing lookup identifiers or upstream response data in SDK logs.
Returned errors retain their existing details and wrapping for caller
compatibility; applications must apply their own logging policy to those errors.
The `Logger` interface and adapter do not sanitize messages supplied by callers.

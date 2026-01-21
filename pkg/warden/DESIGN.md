# Warden SDK Design Documentation

> 📖 **Multi-language Documentation**: For documentation in other languages, please refer to [docs directory](../../docs/)

This document describes the design principles, architecture, and implementation details of the Warden SDK.

## Design Principles

1. **Simple and Easy**: Provides clean API interfaces
2. **High Performance**: Built-in cache support, reduces API calls
3. **Thread Safe**: All methods are concurrency-safe
4. **Flexible Configuration**: Supports custom timeout, cache, logger, etc.

## Architecture Design

### Core Components

1. **Client**: HTTP client wrapper
2. **Cache**: Thread-safe in-memory cache
3. **Options**: Configuration options (using Builder pattern)
4. **Logger**: Logger interface (supports different logging libraries)

### Concurrency Safety

- `http.Client` is concurrency-safe
- `Cache` uses `sync.RWMutex` to ensure thread safety
- All fields of `Client` are read-only after creation
- All methods are thread-safe and can be called concurrently in multiple goroutines

### Cache Strategy

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

### Error Handling

- Uses custom `Error` type with error codes and detailed information
- Supports error wrapping (`Unwrap()` method)
- All errors implement the `error` interface
- `CheckUserInList()` method does not expose detailed information when encountering errors to avoid leaking whether a user exists (security consideration)

### CheckUserInList Implementation Strategy

The `CheckUserInList()` method uses the following strategy:

1. **Input Normalization**: Automatically trims leading and trailing spaces from phone and mail, and converts mail to lowercase
2. **Priority Strategy**: If both phone and mail are provided, phone takes priority
3. **Smart Fallback**:
   - When phone lookup returns `NotFound` error, if mail is not empty, automatically falls back to mail lookup
   - When phone lookup succeeds but user status is not active, does not fall back to mail (user already found)
   - When phone lookup encounters other errors (e.g., network error), does not fall back to mail
4. **Status Validation**: Only users with status "active" will return `true`
5. **Performance Optimization**: Uses `GetUserByIdentifier()` for direct query, avoiding fetching the entire user list

### Configuration Validation

- `Validate()` method normalizes `BaseURL` (adds protocol, removes trailing slash)
- Validates timeout and cache TTL
- If Logger is not provided, uses `NoOpLogger`

### Retry Mechanism

The SDK supports configurable retry logic for handling transient failures:

1. **Retryable Errors**: Network errors and server errors (5xx) are retryable by default
2. **Non-Retryable Errors**: Client errors (4xx) like 401 Unauthorized and 404 Not Found are never retried
3. **Exponential Backoff**: Uses exponential backoff with configurable multiplier and max delay
4. **Configurable**: Retry behavior can be customized via `RetryOptions`

### Custom HTTP Transport

The SDK supports custom `http.Transport` configuration:
- Allows custom TLS settings, proxy configuration, connection pooling, etc.
- Set via `Options.WithTransport()`
- If not provided, uses default HTTP client transport

### Cache Invalidation

The SDK supports multiple cache invalidation strategies:

1. **TTL Expiration**: Automatic expiration based on configured TTL (default behavior)
2. **Manual Invalidation**: `ClearCache()` or `InvalidateCache()` methods
3. **Event-Driven Invalidation**: Listen to external events via channel
   - Configure via `Options.WithCacheInvalidationChannel()`
   - Automatically clears cache when signal is received
   - Runs in background goroutine, call `Close()` to stop listener

## Known Limitations

1. **Pagination Cache**: `GetUsersPaginated()` does not use cache
   - This is an intentional design to ensure data accuracy
   - If pagination cache is needed, more complex caching strategies can be implemented

2. **Single User Query Cache**: `GetUserByIdentifier()` and `CheckUserInList()` do not use cache
   - This is an intentional design to ensure data real-time
   - If caching is needed, cache strategies based on user identifiers can be implemented

## Usage Recommendations

1. **Reuse Client**: Create the Client once and reuse it throughout the application lifecycle
2. **Set Cache TTL Appropriately**: Set appropriate cache time based on data update frequency
3. **Use Context**: Pass context to support cancellation and timeout control
4. **Error Handling**: Always check and handle errors
5. **Logging**: Use appropriate logger implementation in production environments
6. **Close Client**: Call `Close()` when the client is no longer needed to stop background goroutines (e.g., cache invalidation listener)
7. **Configure Retry**: Enable retry for production environments to handle transient failures
8. **Custom Transport**: Use custom transport for advanced scenarios (TLS, proxy, etc.)

## Future Improvements

1. Support request/response middleware
2. Support metrics collection
3. Support connection pooling configuration
4. Support circuit breaker pattern

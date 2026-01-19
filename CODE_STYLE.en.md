# Code Style Guide

> 🌐 **Language / 语言**: [English](CODE_STYLE.en.md) | [中文](CODE_STYLE.md)

This document defines the code style and best practices for the Warden project. All contributors should follow these standards.

## 📋 Table of Contents

- [Go Code Standards](#go-code-standards)
- [Naming Conventions](#naming-conventions)
- [Code Organization](#code-organization)
- [Comment Standards](#comment-standards)
- [Error Handling](#error-handling)
- [Testing Standards](#testing-standards)
- [Performance Optimization](#performance-optimization)
- [Security Standards](#security-standards)

## 🔧 Go Code Standards

### Basic Standards

1. **Follow Go Official Standards**
   - Refer to [Effective Go](https://go.dev/doc/effective_go)
   - Refer to [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

2. **Use `gofmt` for Formatting**
   ```bash
   go fmt ./...
   ```

3. **Use `golint` for Checking**
   ```bash
   golint ./...
   ```

4. **Use `go vet` for Checking**
   ```bash
   go vet ./...
   ```

### Code Format

- Use 1 Tab for indentation (not spaces)
- Line length: Try to keep within 100 characters, can exceed if necessary
- Use `gofmt` for automatic formatting, don't manually adjust format

## 📝 Naming Conventions

### Package Names

- Use lowercase letters, short and meaningful
- Avoid underscores or mixed case
- Package name should be the last element of the import path

```go
// ✅ Correct
package cache
package router
package parser

// ❌ Wrong
package Cache
package user_cache
package UserCache
```

### Variables and Functions

- **Exported (public)**: Use PascalCase
- **Unexported (private)**: Use camelCase
- **Constants**: Use ALL_CAPS (UPPER_SNAKE_CASE)

```go
// ✅ Correct
var UserCache *cache.SafeUserCache
var redisClient *redis.Client
const DEFAULT_TIMEOUT = 5 * time.Second
const MAX_RETRIES = 3
const DEFAULT_RATE_LIMIT = 60

// ❌ Wrong
var user_cache *cache.SafeUserCache
var RedisClient *redis.Client
const DefaultTimeout = 5 * time.Second  // Constants should use ALL_CAPS
```

### Interface Names

- Interface names should be verbs or verb phrases
- If an interface has only one method, the interface name should be the method name + "er"

```go
// ✅ Correct
type Reader interface {
    Read([]byte) (int, error)
}

type UserCache interface {
    Get() []define.AllowListUser
    Set(users []define.AllowListUser)
}

// ❌ Wrong
type IReader interface {
    Read([]byte) (int, error)
}
```

### Error Variables

- Error variables should start with `Err`
- Error types should end with `Error`

```go
// ✅ Correct
var ErrNotFound = errors.New("not found")
var ErrInvalidInput = errors.New("invalid input")

type ValidationError struct {
    Field string
    Message string
}

// ❌ Wrong
var NotFound = errors.New("not found")
var InvalidInputError = errors.New("invalid input")
```

## 📁 Code Organization

### File Structure

```
internal/
├── cache/          # Cache related
│   ├── cache.go
│   ├── cache_test.go
│   └── redis_cache.go
├── router/         # Route handling
│   ├── router.go
│   ├── json.go
│   └── health.go
└── ...
```

### Import Order

Organize imports in the following order:

1. Standard library
2. Third-party libraries
3. Project internal packages

```go
import (
    // Standard library
    "context"
    "fmt"
    "net/http"
    "time"
    
    // Third-party libraries
    "github.com/redis/go-redis/v9"
    "github.com/rs/zerolog"
    
    // Project internal packages
    "github.com/soulteary/warden/internal/cache"
    "github.com/soulteary/warden/internal/define"
)
```

### Function Length

- Single functions should not exceed 50 lines
- If a function is too long, consider splitting into multiple smaller functions
- Complex logic should be extracted as independent functions

### File Length

- Single files should not exceed 500 lines
- If a file is too long, consider splitting into multiple files

## 💬 Comment Standards

### Package Comments

Each package should have a package comment describing the package's purpose and usage.

```go
// Package cache provides caching functionality for user data.
// Supports both in-memory cache and Redis cache implementations.
package cache
```

### Exported Function Comments

All exported functions, types, and variables should have comments.

```go
// NewSafeUserCache creates a new thread-safe user cache instance.
// The returned cache instance supports concurrent read and write operations.
func NewSafeUserCache() *SafeUserCache {
    // ...
}
```

### Function Comment Format

```go
// FunctionName briefly describes the function's purpose.
//
// Detailed description (if needed).
//
// Parameters:
//   - param1: Description of parameter 1
//   - param2: Description of parameter 2
//
// Returns:
//   - Description of return value 1
//   - Description of return value 2
//
// Example:
//   result := FunctionName(param1, param2)
func FunctionName(param1 Type1, param2 Type2) (ReturnType1, ReturnType2) {
    // ...
}
```

### Inline Comments

- Explain "why" rather than "what"
- Avoid obvious comments
- Complex logic must have comments

```go
// ✅ Correct
// Use hash value to quickly detect data changes, avoiding performance overhead of full comparison
if oldHash != newHash {
    // ...
}

// ❌ Wrong
// Compare hash values
if oldHash != newHash {
    // ...
}
```

## ⚠️ Error Handling

### Error Checking

- Always check errors, don't ignore them
- Use meaningful error messages
- Use `fmt.Errorf` to wrap errors, adding context

```go
// ✅ Correct
if err != nil {
    return fmt.Errorf("failed to load config file: %w", err)
}

// ❌ Wrong
if err != nil {
    return err
}
```

### Error Returns

- Errors should be the last return value
- If a function may fail, it should return an error

```go
// ✅ Correct
func LoadConfig(path string) (*Config, error) {
    // ...
}

// ❌ Wrong
func LoadConfig(path string) (error, *Config) {
    // ...
}
```

### Custom Errors

For errors that need additional information, use custom error types.

```go
type ConfigError struct {
    Path    string
    Message string
    Err     error
}

func (e *ConfigError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("config error [%s]: %s: %v", e.Path, e.Message, e.Err)
    }
    return fmt.Sprintf("config error [%s]: %s", e.Path, e.Message)
}

func (e *ConfigError) Unwrap() error {
    return e.Err
}
```

## 🧪 Testing Standards

### Test Files

- Test files end with `_test.go`
- Test functions start with `Test`
- Benchmark tests start with `Benchmark`

```go
// cache_test.go
package cache

import "testing"

func TestNewSafeUserCache(t *testing.T) {
    // ...
}

func BenchmarkGet(b *testing.B) {
    // ...
}
```

### Test Naming

Test function names should describe the test scenario.

```go
// ✅ Correct
func TestSafeUserCache_Get_EmptyCache(t *testing.T)
func TestSafeUserCache_Set_ConcurrentAccess(t *testing.T)

// ❌ Wrong
func TestCache1(t *testing.T)
func TestCache2(t *testing.T)
```

### Table-Driven Tests

For multiple test cases, use table-driven tests.

```go
func TestParsePaginationParams(t *testing.T) {
    tests := []struct {
        name        string
        pageStr     string
        sizeStr     string
        wantPage    int
        wantSize    int
        wantErr     bool
    }{
        {
            name:     "valid params",
            pageStr:  "1",
            sizeStr:  "100",
            wantPage: 1,
            wantSize: 100,
            wantErr:  false,
        },
        {
            name:     "invalid page",
            pageStr:  "abc",
            sizeStr:  "100",
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

### Test Coverage

- New features must include tests
- Target coverage: 80% or higher
- Critical paths must have 100% coverage

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## ⚡ Performance Optimization

### Avoid Premature Optimization

- Ensure code correctness first, then consider optimization
- Use profiling tools to identify bottlenecks
- Optimization should be data-driven

### Common Optimization Techniques

1. **Use Object Pools**
   ```go
   var bufferPool = sync.Pool{
       New: func() interface{} {
           return &bytes.Buffer{}
       },
   }
   ```

2. **Avoid Unnecessary Memory Allocation**
   ```go
   // ✅ Correct: Pre-allocate capacity
   result := make([]User, 0, len(users))
   
   // ❌ Wrong: Dynamic expansion
   result := []User{}
   ```

3. **Use Concurrent-Safe Data Structures**
   ```go
   // ✅ Correct
   var mu sync.RWMutex
   
   // ❌ Wrong: Use global variables without locks
   var globalData []User
   ```

## 🔒 Security Standards

### Input Validation

- All user input must be validated
- Limit parameter length and range
- Prevent injection attacks

```go
// ✅ Correct
const MAX_PARAM_LENGTH = 20
if len(input) > MAX_PARAM_LENGTH {
    return fmt.Errorf("parameter length exceeds limit")
}

// Validate number range
if page < 1 || page > maxPage {
    return fmt.Errorf("page number out of range")
}
```

### Sensitive Information

- Don't log sensitive information (passwords, tokens, etc.)
- Use environment variables to store sensitive configuration
- Don't hardcode keys in code

```go
// ✅ Correct
password := os.Getenv("REDIS_PASSWORD")

// ❌ Wrong
password := "hardcoded_password"
```

### Error Messages

- Don't expose detailed error information in production
- Use generic error messages returned to users
- Detailed errors are only recorded in logs

```go
// ✅ Correct
if isProduction {
    http.Error(w, "Internal server error", http.StatusInternalServerError)
    log.Error().Err(err).Msg("detailed error information")
} else {
    http.Error(w, err.Error(), http.StatusInternalServerError)
}
```

## 📚 Reference Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Best Practices](https://golang.org/doc/effective_go.html)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

## 🔍 Code Checking Tools

### Recommended Tools

1. **gofmt**: Code formatting
   ```bash
   gofmt -w .
   ```

2. **golint**: Code style checking
   ```bash
   golint ./...
   ```

3. **go vet**: Static analysis
   ```bash
   go vet ./...
   ```

4. **golangci-lint**: Comprehensive lint tool
   ```bash
   golangci-lint run
   ```

### Editor Configuration

#### VS Code

Install Go extension, configure `.vscode/settings.json`:

```json
{
    "go.formatTool": "gofmt",
    "go.lintTool": "golangci-lint",
    "go.lintOnSave": true,
    "go.formatOnSave": true
}
```

#### GoLand

- Enable `gofmt` formatting
- Enable `go vet` checking
- Configure code style templates

---

Following these standards helps maintain consistency and maintainability of the codebase. If you have questions, please refer to existing code in the project or contact the maintainers.


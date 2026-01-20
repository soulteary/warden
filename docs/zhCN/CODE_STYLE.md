# 代码风格指南

> 🌐 **Language / 语言**: [English](../enUS/CODE_STYLE.md) | [中文](CODE_STYLE.md)

本文档定义了 Warden 项目的代码风格和最佳实践。所有贡献者都应遵循这些规范。

## 📋 目录

- [Go 代码规范](#go-代码规范)
- [命名规范](#命名规范)
- [代码组织](#代码组织)
- [注释规范](#注释规范)
- [错误处理](#错误处理)
- [测试规范](#测试规范)
- [性能优化](#性能优化)
- [安全规范](#安全规范)

## 🔧 Go 代码规范

### 基本规范

1. **遵循 Go 官方规范**
   - 参考 [Effective Go](https://go.dev/doc/effective_go)
   - 参考 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

2. **使用 `gofmt` 格式化**
   ```bash
   go fmt ./...
   ```

3. **使用 `golint` 检查**
   ```bash
   golint ./...
   ```

4. **使用 `go vet` 检查**
   ```bash
   go vet ./...
   ```

### 代码格式

- 使用 1 个 Tab 进行缩进（不是空格）
- 行长度：尽量保持在 100 字符以内，必要时可以超过
- 使用 `gofmt` 自动格式化，不要手动调整格式

## 📝 命名规范

### 包名

- 使用小写字母，简短且有意义
- 避免使用下划线或混合大小写
- 包名应该是导入路径的最后一个元素

```go
// ✅ 正确
package cache
package router
package parser

// ❌ 错误
package Cache
package user_cache
package UserCache
```

### 变量和函数

- **导出（公共）**: 使用 PascalCase
- **未导出（私有）**: 使用 camelCase
- **常量**: 使用 ALL_CAPS (UPPER_SNAKE_CASE)

```go
// ✅ 正确
var UserCache *cache.SafeUserCache
var redisClient *redis.Client
const DEFAULT_TIMEOUT = 5 * time.Second
const MAX_RETRIES = 3
const DEFAULT_RATE_LIMIT = 60

// ❌ 错误
var user_cache *cache.SafeUserCache
var RedisClient *redis.Client
const DefaultTimeout = 5 * time.Second  // 常量应使用 ALL_CAPS
```

### 接口名

- 接口名应该是动词或动词短语
- 如果接口只有一个方法，接口名应该是方法名 + "er"

```go
// ✅ 正确
type Reader interface {
    Read([]byte) (int, error)
}

type UserCache interface {
    Get() []define.AllowListUser
    Set(users []define.AllowListUser)
}

// ❌ 错误
type IReader interface {
    Read([]byte) (int, error)
}
```

### 错误变量

- 错误变量应该以 `Err` 开头
- 错误类型应该以 `Error` 结尾

```go
// ✅ 正确
var ErrNotFound = errors.New("not found")
var ErrInvalidInput = errors.New("invalid input")

type ValidationError struct {
    Field string
    Message string
}

// ❌ 错误
var NotFound = errors.New("not found")
var InvalidInputError = errors.New("invalid input")
```

## 📁 代码组织

### 文件结构

```
internal/
├── cache/          # 缓存相关
│   ├── cache.go
│   ├── cache_test.go
│   └── redis_cache.go
├── router/         # 路由处理
│   ├── router.go
│   ├── json.go
│   └── health.go
└── ...
```

### 导入顺序

按照以下顺序组织导入：

1. 标准库
2. 第三方库
3. 项目内部包

```go
import (
    // 标准库
    "context"
    "fmt"
    "net/http"
    "time"
    
    // 第三方库
    "github.com/redis/go-redis/v9"
    "github.com/rs/zerolog"
    
    // 项目内部包
    "github.com/soulteary/warden/internal/cache"
    "github.com/soulteary/warden/internal/define"
)
```

### 函数长度

- 单个函数尽量不超过 50 行
- 如果函数过长，考虑拆分为多个小函数
- 复杂逻辑应该提取为独立函数

### 文件长度

- 单个文件尽量不超过 500 行
- 如果文件过长，考虑拆分为多个文件

## 💬 注释规范

### 包注释

每个包都应该有一个包注释，介绍包的目的和用法。

```go
// Package cache 提供了用户数据的缓存功能。
// 支持内存缓存和 Redis 缓存两种实现。
package cache
```

### 导出函数注释

所有导出的函数、类型、变量都应该有注释。

```go
// NewSafeUserCache 创建一个新的线程安全的用户缓存实例。
// 返回的缓存实例支持并发读写操作。
func NewSafeUserCache() *SafeUserCache {
    // ...
}
```

### 函数注释格式

```go
// FunctionName 简要描述函数的功能。
//
// 详细描述（如果需要）。
//
// 参数:
//   - param1: 参数1的描述
//   - param2: 参数2的描述
//
// 返回:
//   - 返回值1的描述
//   - 返回值2的描述
//
// 示例:
//   result := FunctionName(param1, param2)
func FunctionName(param1 Type1, param2 Type2) (ReturnType1, ReturnType2) {
    // ...
}
```

### 内联注释

- 解释"为什么"而不是"是什么"
- 避免显而易见的注释
- 复杂逻辑必须添加注释

```go
// ✅ 正确
// 使用哈希值快速检测数据变化，避免全量比较的性能开销
if oldHash != newHash {
    // ...
}

// ❌ 错误
// 比较哈希值
if oldHash != newHash {
    // ...
}
```

## ⚠️ 错误处理

### 错误检查

- 总是检查错误，不要忽略
- 使用有意义的错误消息
- 使用 `fmt.Errorf` 包装错误，添加上下文

```go
// ✅ 正确
if err != nil {
    return fmt.Errorf("加载配置文件失败: %w", err)
}

// ❌ 错误
if err != nil {
    return err
}
```

### 错误返回

- 错误应该是最后一个返回值
- 如果函数可能失败，应该返回错误

```go
// ✅ 正确
func LoadConfig(path string) (*Config, error) {
    // ...
}

// ❌ 错误
func LoadConfig(path string) (error, *Config) {
    // ...
}
```

### 自定义错误

对于需要额外信息的错误，使用自定义错误类型。

```go
type ConfigError struct {
    Path    string
    Message string
    Err     error
}

func (e *ConfigError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("配置错误 [%s]: %s: %v", e.Path, e.Message, e.Err)
    }
    return fmt.Sprintf("配置错误 [%s]: %s", e.Path, e.Message)
}

func (e *ConfigError) Unwrap() error {
    return e.Err
}
```

## 🧪 测试规范

### 测试文件

- 测试文件以 `_test.go` 结尾
- 测试函数以 `Test` 开头
- 基准测试以 `Benchmark` 开头

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

### 测试命名

测试函数名应该描述测试的场景。

```go
// ✅ 正确
func TestSafeUserCache_Get_EmptyCache(t *testing.T)
func TestSafeUserCache_Set_ConcurrentAccess(t *testing.T)

// ❌ 错误
func TestCache1(t *testing.T)
func TestCache2(t *testing.T)
```

### 表驱动测试

对于多个测试用例，使用表驱动测试。

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

### 测试覆盖率

- 新功能必须包含测试
- 目标覆盖率：80% 以上
- 关键路径必须 100% 覆盖

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## ⚡ 性能优化

### 避免过早优化

- 先确保代码正确，再考虑优化
- 使用性能分析工具找出瓶颈
- 优化要有数据支撑

### 常见优化技巧

1. **使用对象池**
   ```go
   var bufferPool = sync.Pool{
       New: func() interface{} {
           return &bytes.Buffer{}
       },
   }
   ```

2. **避免不必要的内存分配**
   ```go
   // ✅ 正确：预分配容量
   result := make([]User, 0, len(users))
   
   // ❌ 错误：动态扩容
   result := []User{}
   ```

3. **使用并发安全的数据结构**
   ```go
   // ✅ 正确
   var mu sync.RWMutex
   
   // ❌ 错误：使用全局变量不加锁
   var globalData []User
   ```

## 🔒 安全规范

### 输入验证

- 所有用户输入必须验证
- 限制参数长度和范围
- 防止注入攻击

```go
// ✅ 正确
const MAX_PARAM_LENGTH = 20
if len(input) > MAX_PARAM_LENGTH {
    return fmt.Errorf("参数长度超过限制")
}

// 验证数字范围
if page < 1 || page > maxPage {
    return fmt.Errorf("页码超出范围")
}
```

### 敏感信息

- 不要在日志中记录敏感信息（密码、token 等）
- 使用环境变量存储敏感配置
- 不要在代码中硬编码密钥

```go
// ✅ 正确
password := os.Getenv("REDIS_PASSWORD")

// ❌ 错误
password := "hardcoded_password"
```

### 错误信息

- 生产环境不要暴露详细的错误信息
- 使用通用错误消息返回给用户
- 详细错误只记录在日志中

```go
// ✅ 正确
if isProduction {
    http.Error(w, "Internal server error", http.StatusInternalServerError)
    log.Error().Err(err).Msg("详细错误信息")
} else {
    http.Error(w, err.Error(), http.StatusInternalServerError)
}
```

## 📚 参考资源

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Best Practices](https://golang.org/doc/effective_go.html)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

## 🔍 代码检查工具

### 推荐工具

1. **gofmt**: 代码格式化
   ```bash
   gofmt -w .
   ```

2. **golint**: 代码风格检查
   ```bash
   golint ./...
   ```

3. **go vet**: 静态分析
   ```bash
   go vet ./...
   ```

4. **golangci-lint**: 综合 lint 工具
   ```bash
   golangci-lint run
   ```

### 编辑器配置

#### VS Code

安装 Go 扩展，配置 `.vscode/settings.json`:

```json
{
    "go.formatTool": "gofmt",
    "go.lintTool": "golangci-lint",
    "go.lintOnSave": true,
    "go.formatOnSave": true
}
```

#### GoLand

- 启用 `gofmt` 格式化
- 启用 `go vet` 检查
- 配置代码风格模板

---

遵循这些规范有助于保持代码库的一致性和可维护性。如有疑问，请参考项目中的现有代码或联系维护者。


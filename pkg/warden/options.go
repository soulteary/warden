package warden

import (
	"strings"
	"time"
)

// Options contains configuration options for the Warden SDK client.
//
//nolint:govet // fieldalignment: 字段顺序已优化，但为了保持 API 兼容性，不进一步调整
type Options struct {
	BaseURL  string        // Warden 服务地址（必需）
	APIKey   string        // API Key（可选）
	Timeout  time.Duration // HTTP 请求超时（默认 10s）
	CacheTTL time.Duration // 缓存 TTL（默认 5 分钟）
	Logger   Logger        // 日志接口（可选，默认使用 NoOpLogger）
}

// DefaultOptions returns default options with sensible defaults.
func DefaultOptions() *Options {
	return &Options{
		Timeout:  10 * time.Second,
		CacheTTL: 5 * time.Minute,
		Logger:   &NoOpLogger{},
	}
}

// Validate validates the options and returns an error if invalid.
func (o *Options) Validate() error {
	if o.BaseURL == "" {
		return NewError(ErrCodeInvalidConfig, "BaseURL is required", nil)
	}

	// Normalize BaseURL
	o.BaseURL = strings.TrimSuffix(o.BaseURL, "/")

	// Add protocol prefix if missing
	if !strings.HasPrefix(o.BaseURL, "http://") && !strings.HasPrefix(o.BaseURL, "https://") {
		o.BaseURL = "http://" + o.BaseURL
	}

	// Validate timeout
	if o.Timeout <= 0 {
		return NewError(ErrCodeInvalidConfig, "Timeout must be greater than 0", nil)
	}

	// Validate cache TTL
	if o.CacheTTL < 0 {
		return NewError(ErrCodeInvalidConfig, "CacheTTL must be non-negative", nil)
	}

	// Set default logger if not provided
	if o.Logger == nil {
		o.Logger = &NoOpLogger{}
	}

	return nil
}

// WithBaseURL sets the base URL for the Warden service.
func (o *Options) WithBaseURL(url string) *Options {
	o.BaseURL = url
	return o
}

// WithAPIKey sets the API key for authentication.
func (o *Options) WithAPIKey(key string) *Options {
	o.APIKey = key
	return o
}

// WithTimeout sets the HTTP request timeout.
func (o *Options) WithTimeout(timeout time.Duration) *Options {
	o.Timeout = timeout
	return o
}

// WithCacheTTL sets the cache TTL.
func (o *Options) WithCacheTTL(ttl time.Duration) *Options {
	o.CacheTTL = ttl
	return o
}

// WithLogger sets the logger implementation.
func (o *Options) WithLogger(logger Logger) *Options {
	o.Logger = logger
	return o
}

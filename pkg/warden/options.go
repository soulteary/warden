package warden

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"
)

// RetryOptions contains configuration for request retry behavior.
//
//nolint:govet // fieldalignment: field order optimized for API compatibility
type RetryOptions struct {
	RetryableStatusCodes []int         // HTTP status codes that should trigger retry (default: 5xx)
	MaxRetries           int           // Maximum number of retries (default 0, no retry)
	RetryDelay           time.Duration // Initial delay between retries (default 100ms)
	MaxRetryDelay        time.Duration // Maximum delay between retries (default 5s)
	BackoffMultiplier    float64       // Multiplier for exponential backoff (default 2.0)
}

// DefaultRetryOptions returns default retry options with sensible defaults.
func DefaultRetryOptions() *RetryOptions {
	return &RetryOptions{
		MaxRetries:        0, // No retry by default
		RetryDelay:        100 * time.Millisecond,
		MaxRetryDelay:     5 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableStatusCodes: []int{
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	}
}

// Options contains configuration options for the Warden SDK client.
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Options struct {
	BaseURL                  string          // Warden service address (required)
	APIKey                   string          // API Key (optional)
	Timeout                  time.Duration   // HTTP request timeout (default 10s)
	CacheTTL                 time.Duration   // Cache TTL (default 5 minutes)
	Logger                   Logger          // Logger interface (optional, defaults to NoOpLogger)
	Transport                *http.Transport // Custom HTTP transport (optional)
	Retry                    *RetryOptions   // Retry configuration (optional)
	CacheInvalidationChannel <-chan struct{} // Channel for event-driven cache invalidation (optional)

	// HMACKeyID and HMACSecret enable request signing (HMAC v2). When both are set,
	// every request is signed. The secret is never logged.
	HMACKeyID  string
	HMACSecret string
	// Clock overrides the time source used for signature timestamps (testing).
	Clock Clock
	// TLSConfig, when set, is applied to the HTTP transport (e.g. client certs, root
	// CAs). It takes precedence over Transport's own TLSClientConfig.
	TLSConfig *tls.Config
	// MaxResponseBytes bounds the response body read to defend against unbounded
	// payloads. Zero means use DefaultMaxResponseBytes; a negative value disables it.
	MaxResponseBytes int64
}

// DefaultMaxResponseBytes is the default cap on response body size (8 MiB).
const DefaultMaxResponseBytes int64 = 8 << 20

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

	// Validate retry options if provided
	if o.Retry != nil {
		if o.Retry.MaxRetries < 0 {
			return NewError(ErrCodeInvalidConfig, "Retry.MaxRetries must be non-negative", nil)
		}
		if o.Retry.RetryDelay < 0 {
			return NewError(ErrCodeInvalidConfig, "Retry.RetryDelay must be non-negative", nil)
		}
		if o.Retry.MaxRetryDelay < 0 {
			return NewError(ErrCodeInvalidConfig, "Retry.MaxRetryDelay must be non-negative", nil)
		}
		if o.Retry.BackoffMultiplier <= 0 {
			return NewError(ErrCodeInvalidConfig, "Retry.BackoffMultiplier must be greater than 0", nil)
		}
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

// WithTransport sets a custom HTTP transport.
func (o *Options) WithTransport(transport *http.Transport) *Options {
	o.Transport = transport
	return o
}

// WithRetry sets the retry configuration.
func (o *Options) WithRetry(retry *RetryOptions) *Options {
	o.Retry = retry
	return o
}

// WithCacheInvalidationChannel sets a channel for event-driven cache invalidation.
// When a signal is received on this channel, the cache will be automatically cleared.
func (o *Options) WithCacheInvalidationChannel(ch <-chan struct{}) *Options {
	o.CacheInvalidationChannel = ch
	return o
}

// WithHMAC enables HMAC v2 request signing with the given key id and secret.
// The secret is never logged. Passing empty values disables signing.
func (o *Options) WithHMAC(keyID, secret string) *Options {
	o.HMACKeyID = keyID
	o.HMACSecret = secret
	return o
}

// WithClock overrides the time source used for signature timestamps. Primarily
// useful for deterministic tests.
func (o *Options) WithClock(clock Clock) *Options {
	o.Clock = clock
	return o
}

// WithTLSConfig sets a custom *tls.Config (e.g. client certificates, root CAs)
// that will be applied to the HTTP transport.
func (o *Options) WithTLSConfig(cfg *tls.Config) *Options {
	o.TLSConfig = cfg
	return o
}

// WithTLSClientCert configures mutual TLS by presenting the given client
// certificate. It preserves any existing TLSConfig fields.
func (o *Options) WithTLSClientCert(cert tls.Certificate) *Options {
	if o.TLSConfig == nil {
		o.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12} //nolint:gosec // MinVersion set explicitly
	}
	o.TLSConfig.Certificates = append(o.TLSConfig.Certificates, cert)
	return o
}

// WithMaxResponseBytes bounds how many bytes of a response body will be read.
// Zero uses DefaultMaxResponseBytes; a negative value disables the limit.
func (o *Options) WithMaxResponseBytes(n int64) *Options {
	o.MaxResponseBytes = n
	return o
}

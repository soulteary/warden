package warden

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RetryOptions contains configuration for request retry behavior.
//
//nolint:govet // fieldalignment: field order optimized for API compatibility
type RetryOptions struct {
	RetryableStatusCodes []int         // HTTP status codes eligible for retry, excluding 4xx (default: 500, 502, 503, 504)
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
	// HMACAudience, when set, binds every signature to the Warden instance
	// with that name (signature version v3). The server must list the same
	// name in WARDEN_HMAC_AUDIENCES. Empty keeps the v2 form.
	HMACAudience string
	// Clock overrides the time source used for signature timestamps and enabled source freshness (testing).
	Clock Clock
	// RequireHTTPS refuses a plain-HTTP BaseURL when an API key would be sent
	// to a host other than the local machine. Without it such a configuration
	// is accepted and reported once through Logger at warn level.
	RequireHTTPS bool
	// TLSConfig, when set, is applied to the HTTP transport (e.g. client certs, root
	// CAs). It takes precedence over Transport's own TLSClientConfig.
	TLSConfig *tls.Config
	// MaxResponseBytes bounds the response body read to defend against unbounded
	// payloads. Zero means use DefaultMaxResponseBytes; a negative value disables it.
	MaxResponseBytes int64
	// SourceMaxAge enables negotiated source freshness when positive; zero keeps compatibility.
	SourceMaxAge time.Duration
	// SourceClockSkew is subtracted from the source deadline and bounds future timestamps.
	SourceClockSkew time.Duration
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

	if err := o.validateHMAC(); err != nil {
		return err
	}

	// Normalize BaseURL
	o.BaseURL = strings.TrimSuffix(o.BaseURL, "/")

	if err := o.validateBaseURL(); err != nil {
		return err
	}

	// Validate timeout
	if o.Timeout <= 0 {
		return NewError(ErrCodeInvalidConfig, "Timeout must be greater than 0", nil)
	}

	if o.SourceMaxAge < 0 || o.SourceClockSkew < 0 || (o.SourceMaxAge > 0 && o.SourceClockSkew >= o.SourceMaxAge) {
		return NewError(ErrCodeInvalidConfig, "source age and clock allowance must be non-negative, with allowance below an enabled age limit", nil)
	}

	// Validate cache TTL
	if o.CacheTTL < 0 {
		return NewError(ErrCodeInvalidConfig, "CacheTTL must be non-negative", nil)
	}

	// Set default logger if not provided
	if o.Logger == nil {
		o.Logger = &NoOpLogger{}
	}

	return o.validateRetry()
}

// WithBaseURL sets the base URL for the Warden service.
func (o *Options) WithBaseURL(baseURL string) *Options {
	o.BaseURL = baseURL
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

// WithRequireHTTPS makes NewClient refuse a configuration that would send the
// API key over plain HTTP to a host other than the local machine.
func (o *Options) WithRequireHTTPS(require bool) *Options {
	o.RequireHTTPS = require
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
// The secret is never logged. Passing two empty values disables signing; a partial
// key-id/secret pair is rejected by Validate.
func (o *Options) WithHMAC(keyID, secret string) *Options {
	o.HMACKeyID = keyID
	o.HMACSecret = secret
	return o
}

// WithHMACAudience binds every HMAC signature to the Warden instance with the
// given name, so a signed request is not valid at another instance that shares
// the key. The name is compared without regard to case and must be listed in
// that server's WARDEN_HMAC_AUDIENCES; a server that does not list it answers
// 401. It requires WithHMAC. An empty name keeps the unbound v2 form.
func (o *Options) WithHMACAudience(audience string) *Options {
	o.HMACAudience = audience
	return o
}

// WithClock overrides the time source used for signature timestamps and enabled source freshness. Primarily
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
func (o *Options) WithTLSClientCert(cert tls.Certificate) *Options { //nolint:gocritic // public API: value receiver keeps a stable, ergonomic signature
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

// WithSourceFreshness enables source-bounded reads and caching. Zero maxAge
// disables the constraint; clockSkew is a conservative allowance, not extra age.
func (o *Options) WithSourceFreshness(maxAge, clockSkew time.Duration) *Options {
	o.SourceMaxAge = maxAge
	o.SourceClockSkew = clockSkew
	return o
}

func (o *Options) validateHMAC() error {
	// HMAC signing is enabled only as a complete key-id/secret pair. Reject a
	// partial pair instead of silently constructing an unsigned client.
	hmacKeyIDSet := strings.TrimSpace(o.HMACKeyID) != ""
	hmacSecretSet := strings.TrimSpace(o.HMACSecret) != ""
	switch {
	case !hmacKeyIDSet && !hmacSecretSet:
		// Treat a semantically blank pair as disabled. newHMACSigner intentionally
		// preserves secret bytes, so leaving whitespace here would create a signer
		// that overrides an otherwise valid API key with unusable HMAC headers.
		o.HMACKeyID = ""
		o.HMACSecret = ""
	case hmacKeyIDSet != hmacSecretSet:
		return NewError(ErrCodeInvalidConfig, "HMACKeyID and HMACSecret must be configured together", nil)
	default:
		o.HMACKeyID = strings.TrimSpace(o.HMACKeyID)
	}

	// The audience is signed and must match the server's list byte for byte,
	// so it is normalized the way the server normalizes its own.
	o.HMACAudience = strings.ToLower(strings.TrimSpace(o.HMACAudience))
	if o.HMACAudience == "" {
		return nil
	}
	if o.HMACSecret == "" {
		return NewError(ErrCodeInvalidConfig, "HMACAudience requires HMACKeyID and HMACSecret", nil)
	}
	if len(o.HMACAudience) > 255 || strings.ContainsFunc(o.HMACAudience, func(r rune) bool { return r <= ' ' || r > '~' || r == ',' }) {
		return NewError(ErrCodeInvalidConfig, "HMACAudience must be 1 to 255 printable ASCII characters without spaces or commas", nil)
	}
	return nil
}

func (o *Options) validateRetry() error {
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

// validateBaseURL requires an absolute http or https URL. A value without a
// scheme used to be given "http://", which chose cleartext without the caller
// asking for it.
//
// With RequireHTTPS, a plain-HTTP URL to a host other than the local machine
// is refused when an API key is configured: that key is sent in a request
// header and would cross the network in clear. HMAC secrets and client
// certificates are never transmitted, so they are not affected.
func (o *Options) validateBaseURL() error {
	parsed, err := url.Parse(o.BaseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return NewError(ErrCodeInvalidConfig, "BaseURL must be an absolute URL that starts with http:// or https://", nil)
	}
	if o.RequireHTTPS && o.sendsAPIKeyInClear(parsed) {
		return NewError(ErrCodeInvalidConfig, "BaseURL must use https when an API key is configured and RequireHTTPS is set", nil)
	}
	return nil
}

// sendsAPIKeyInClear reports whether requests would carry the API key over
// plain HTTP to a host other than the local machine.
func (o *Options) sendsAPIKeyInClear(base *url.URL) bool {
	if base.Scheme != "http" || o.APIKey == "" || o.HMACSecret != "" {
		return false
	}
	host := base.Hostname()
	if strings.EqualFold(host, "localhost") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return false
	}
	return true
}

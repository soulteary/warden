// Package parser provides data parsing functionality.
// Supports parsing user data from local files and remote APIs, and provides multiple data merging strategies.
package parser

import (
	// Standard library
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
)

// httpClient global HTTP client, uses connection pool to reuse connections
var httpClient = &http.Client{
	Timeout: define.DEFAULT_TIMEOUT * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        define.DEFAULT_MAX_IDLE_CONNS,
		MaxIdleConnsPerHost: define.DEFAULT_MAX_IDLE_CONNS_PER_HOST,
		IdleConnTimeout:     define.DEFAULT_IDLE_CONN_TIMEOUT,
		DisableKeepAlives:   false, // Explicitly set, enable connection reuse
	},
}

// InitHTTPClient initializes HTTP client (using configuration)
func InitHTTPClient(timeout, maxIdleConns int, insecureTLS bool) {
	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: define.DEFAULT_MAX_IDLE_CONNS_PER_HOST,
		IdleConnTimeout:     define.DEFAULT_IDLE_CONN_TIMEOUT,
		DisableKeepAlives:   false,
	}

	// Configure TLS
	if insecureTLS {
		// #nosec G402 -- Only for development environment, allows skipping TLS verification
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // Only for development environment
		}
	}

	httpClient = &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: transport,
	}
}

// doRequestWithRetry executes HTTP request with retry mechanism
//
// This function implements exponential backoff retry strategy with the following features:
// - Context cancellation: checks if context is cancelled before each retry
// - Automatic retry: network errors and 5xx server errors are automatically retried
// - Incremental delay: delay time increases with each retry (retryDelay * attempt)
//
// Parameters:
//   - ctx: context for request cancellation and timeout control
//   - req: HTTP request object
//   - maxRetries: maximum retry count (excluding initial request)
//   - retryDelay: base retry delay time, actual delay increases with retry count
//
// Returns:
//   - *http.Response: returns response object on success, caller is responsible for closing response body
//   - error: returns error on failure, includes retry count and last error information
//
// Side effects:
//   - Records debug and warning logs
//   - For 5xx errors, closes response body before retry
func doRequestWithRetry(ctx context.Context, req *http.Request, maxRetries int, retryDelay time.Duration) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
		default:
		}

		if attempt > 0 {
			// Wait before retry, but check context
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
			case <-time.After(retryDelay * time.Duration(attempt)):
			}
			log.Debug().
				Int("attempt", attempt).
				Str("url", req.URL.String()).
				Msg("Retrying HTTP request")
		}

		// Add context to request
		reqWithCtx := req.WithContext(ctx)
		res, err := httpClient.Do(reqWithCtx)
		if err == nil {
			// Check status code, 5xx errors also retry
			if res.StatusCode >= 500 && res.StatusCode < 600 && attempt < maxRetries {
				if closeErr := res.Body.Close(); closeErr != nil {
					log.Warn().Err(closeErr).Msg("Failed to close response body")
				}
				lastErr = fmt.Errorf("server error: HTTP %d", res.StatusCode)
				continue
			}
			return res, nil
		}

		lastErr = err
		// Retry on network errors, other errors (like timeout) also retry
		if attempt < maxRetries {
			log.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Int("max_retries", maxRetries).
				Str("url", req.URL.String()).
				Msg("HTTP request failed, will retry")
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// buildRemoteRequest builds remote request
func buildRemoteRequest(ctx context.Context, url, authorizationHeader string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERR_REQ_INIT_FAILED, err)).
			Str("url", url).
			Msg(define.ERR_REQ_INIT_FAILED)
		return nil, fmt.Errorf("%s: %w", define.ERR_REQ_INIT_FAILED, err)
	}

	req.Header = http.Header{
		"Content-Type":  {"application/json"},
		"Cache-Control": {"max-age=0"},
	}
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	return req, nil
}

// parseRemoteResponse parses remote response
func parseRemoteResponse(res *http.Response, url string) ([]define.AllowListUser, error) {
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Warn().Err(err).Str("url", url).Msg("Failed to close response body")
		}
	}()

	// Check HTTP status code
	if res.StatusCode != http.StatusOK {
		log.Warn().
			Int("status_code", res.StatusCode).
			Str("url", url).
			Msgf("%s: HTTP status %d", define.ERR_GET_CONFIG_FAILED, res.StatusCode)
		return nil, fmt.Errorf("%s: HTTP status %d", define.ERR_GET_CONFIG_FAILED, res.StatusCode)
	}

	// Limit response body size to prevent memory exhaustion attacks
	body, err := io.ReadAll(io.LimitReader(res.Body, define.MAX_JSON_SIZE))
	if err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERR_READ_CONFIG_FAILED, err)).
			Str("url", url).
			Msg(define.ERR_READ_CONFIG_FAILED)
		return nil, fmt.Errorf("%s: %w", define.ERR_READ_CONFIG_FAILED, err)
	}

	var data []define.AllowListUser
	if err := json.Unmarshal(body, &data); err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERR_PARSE_CONFIG_FAILED, err)).
			Str("url", url).
			Msg(define.ERR_PARSE_CONFIG_FAILED)
		return nil, fmt.Errorf("%s: %w", define.ERR_PARSE_CONFIG_FAILED, err)
	}

	// Normalize all user data (set default values, generate user_id)
	for i := range data {
		data[i].Normalize()
	}

	return data, nil
}

// FromRemoteConfig gets user list from remote configuration (supports context)
//
// This function retrieves JSON-format user configuration data from remote URL with the following features:
// - Context control: supports timeout and cancellation operations
// - Automatic retry: uses doRequestWithRetry to implement automatic retry mechanism
// - Authentication support: optional Authorization request header
//
// Parameters:
//   - ctx: context for request cancellation and timeout control
//   - url: remote configuration URL address
//   - authorizationHeader: optional Authorization request header value, not added if empty
//
// Returns:
//   - []define.AllowListUser: returns parsed user list on success
//   - error: returns error on failure, possible reasons include: request initialization failure, network error, HTTP status code error, JSON parsing failure
//
// Side effects:
//   - Records error and warning logs
//   - Sets request headers (Content-Type, Cache-Control, Authorization)
func FromRemoteConfig(ctx context.Context, url, authorizationHeader string) ([]define.AllowListUser, error) {
	req, err := buildRemoteRequest(ctx, url, authorizationHeader)
	if err != nil {
		return nil, err
	}

	res, err := doRequestWithRetry(ctx, req, define.HTTP_RETRY_MAX_RETRIES, define.HTTP_RETRY_DELAY)
	if err != nil {
		log.Error().
			Err(fmt.Errorf("%s: %w", define.ERR_GET_CONFIG_FAILED, err)).
			Str("url", url).
			Msg(define.ERR_GET_CONFIG_FAILED)
		return nil, fmt.Errorf("%s: %w", define.ERR_GET_CONFIG_FAILED, err)
	}

	return parseRemoteResponse(res, url)
}

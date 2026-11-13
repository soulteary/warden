package warden

import (
	"errors"
	"fmt"
	"time"
)

// ErrRedirect indicates a rejected service redirect. SDK credentials are bound
// to the configured service and are never forwarded to a redirect target.
var ErrRedirect = errors.New("warden: redirects are disabled")

// ErrResponseTooLarge is returned when a response exceeds MaxResponseBytes.
var ErrResponseTooLarge = errors.New("warden: response exceeds size limit")

// ErrCacheInvalidated indicates that a list response crossed a cache invalidation.
// The response is neither returned nor cached; callers may issue a new request.
var ErrCacheInvalidated = errors.New("warden: cache invalidated during request")

// ErrCacheSuperseded indicates a later-started list request already committed.
// The older response is neither returned nor cached.
var ErrCacheSuperseded = errors.New("warden: list response superseded by a newer request")

// Error represents an error that occurred in the SDK.
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Error struct {
	Code    string // Error code
	Message string // Error message
	Err     error  // Original error (if any)

	// RetryAfter is how long the server asked the caller to wait before the
	// next request. It is set on ErrCodeRateLimited errors when the response
	// carried a Retry-After header, and zero otherwise.
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Predefined error codes
const (
	ErrCodeInvalidConfig   = "INVALID_CONFIG"
	ErrCodeRequestFailed   = "REQUEST_FAILED"
	ErrCodeInvalidResponse = "INVALID_RESPONSE"
	ErrCodeUnauthorized    = "UNAUTHORIZED"
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeServerError     = "SERVER_ERROR"
	ErrCodeSourceFreshness = "SOURCE_FRESHNESS"
	// ErrCodeRateLimited is returned for HTTP 429. The SDK does not retry it;
	// Error.RetryAfter says when the server will accept another request.
	ErrCodeRateLimited = "RATE_LIMITED"
)

// NewError creates a new SDK error.
func NewError(code, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

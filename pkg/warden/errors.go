package warden

import "fmt"

// Error represents an error that occurred in the SDK.
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type Error struct {
	Code    string // Error code
	Message string // Error message
	Err     error  // Original error (if any)
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
)

// NewError creates a new SDK error.
func NewError(code, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

package warden

import "fmt"

// Error represents an error that occurred in the SDK.
//
//nolint:govet // fieldalignment: 字段顺序已优化，但为了保持 API 兼容性，不进一步调整
type Error struct {
	Code    string // 错误代码
	Message string // 错误消息
	Err     error  // 原始错误（如果有）
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

// 预定义的错误代码
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

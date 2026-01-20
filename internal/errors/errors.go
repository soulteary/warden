// Package errors provides unified error handling functionality.
// Defines application error types and predefined error variables, supports error wrapping and context information.
//
//nolint:revive // Package name conflicts with standard library, but this is an internal package, acceptable
package errors

import (
	// Standard library
	"fmt"
	"net/http"

	// Internal packages
	"github.com/soulteary/warden/internal/i18n"
)

// AppError application error type, provides unified error handling
//
//nolint:govet // fieldalignment: field order has been optimized, but not further adjusted to maintain API compatibility
type AppError struct {
	Code    string // Error code (16 bytes)
	Message string // Error message (16 bytes)
	Err     error  // Underlying error (16 bytes interface)
}

// Error implements error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap implements error wrapping interface, supports errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithError wraps underlying error
func (e *AppError) WithError(err error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Err:     err,
	}
}

// WithMessage adds custom message
func (e *AppError) WithMessage(msg string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: msg,
		Err:     e.Err,
	}
}

// WithLanguage gets localized error message based on request context
// If request is nil or language cannot be obtained, returns original message
func (e *AppError) WithLanguage(r *http.Request) *AppError {
	if r == nil {
		return e
	}

	// Get i18n key based on error code
	key := getI18nKey(e.Code)
	if key == "" {
		return e
	}

	// Get localized message
	localizedMsg := i18n.T(r, key)
	if localizedMsg == key {
		// If translation does not exist, use original message
		return e
	}

	return &AppError{
		Code:    e.Code,
		Message: localizedMsg,
		Err:     e.Err,
	}
}

// getI18nKey gets i18n key based on error code
func getI18nKey(code string) string {
	switch code {
	case "REDIS_CONN_ERR":
		return "error.redis_connection_failed"
	case "REDIS_OP_ERR":
		return "error.redis_operation_failed"
	case "REDIS_LOCK_ERR":
		return "error.redis_lock_failed"
	case "CONFIG_LOAD_ERR":
		return "error.config_load_failed"
	case "CONFIG_VALIDATION_ERR":
		return "error.config_validation_failed"
	case "CONFIG_PARSE_ERR":
		return "error.config_parse_failed"
	case "APP_INIT_ERR":
		return "error.app_init_failed"
	case "HTTP_REQ_ERR":
		return "error.http_request_failed"
	case "HTTP_RESP_ERR":
		return "error.http_response_failed"
	case "DATA_LOAD_ERR":
		return "error.data_load_failed"
	case "DATA_PARSE_ERR":
		return "error.data_parse_failed"
	case "CACHE_OP_ERR":
		return "error.cache_operation_failed"
	case "INVALID_PARAM_ERR":
		return "error.invalid_parameter"
	case "TASK_EXEC_ERR":
		return "error.task_execution_failed"
	default:
		return ""
	}
}

// Predefined error types
var (
	// Redis related errors
	ErrRedisConnection = &AppError{
		Code:    "REDIS_CONN_ERR",
		Message: "Redis 连接失败",
	}
	ErrRedisOperation = &AppError{
		Code:    "REDIS_OP_ERR",
		Message: "Redis 操作失败",
	}
	ErrRedisLock = &AppError{
		Code:    "REDIS_LOCK_ERR",
		Message: "Redis 分布式锁操作失败",
	}

	// Configuration related errors
	ErrConfigLoad = &AppError{
		Code:    "CONFIG_LOAD_ERR",
		Message: "配置加载失败",
	}
	ErrConfigValidation = &AppError{
		Code:    "CONFIG_VALIDATION_ERR",
		Message: "配置验证失败",
	}
	ErrConfigParse = &AppError{
		Code:    "CONFIG_PARSE_ERR",
		Message: "配置解析失败",
	}

	// Application initialization errors
	ErrAppInit = &AppError{
		Code:    "APP_INIT_ERR",
		Message: "应用初始化失败",
	}

	// HTTP related errors
	ErrHTTPRequest = &AppError{
		Code:    "HTTP_REQ_ERR",
		Message: "HTTP 请求失败",
	}
	ErrHTTPResponse = &AppError{
		Code:    "HTTP_RESP_ERR",
		Message: "HTTP 响应处理失败",
	}

	// Data related errors
	ErrDataLoad = &AppError{
		Code:    "DATA_LOAD_ERR",
		Message: "数据加载失败",
	}
	ErrDataParse = &AppError{
		Code:    "DATA_PARSE_ERR",
		Message: "数据解析失败",
	}

	// Cache related errors
	ErrCacheOperation = &AppError{
		Code:    "CACHE_OP_ERR",
		Message: "缓存操作失败",
	}

	// Parameter validation errors
	ErrInvalidParameter = &AppError{
		Code:    "INVALID_PARAM_ERR",
		Message: "无效的参数",
	}

	// Task execution errors
	ErrTaskExecution = &AppError{
		Code:    "TASK_EXEC_ERR",
		Message: "任务执行失败",
	}
)

// Wrap wraps error, provides context information
func Wrap(err error, code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Wrapf wraps error using formatted string
func Wrapf(err error, code, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

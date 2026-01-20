// Package errors 提供了统一的错误处理功能。
// 定义了应用错误类型和预定义的错误变量，支持错误包装和上下文信息。
//
//nolint:revive // 包名与标准库冲突，但这是项目内部包，可以接受
package errors

import (
	// 标准库
	"fmt"
	"net/http"

	// 项目内部包
	"github.com/soulteary/warden/internal/i18n"
)

// AppError 应用错误类型，提供统一的错误处理
//
//nolint:govet // fieldalignment: 字段顺序已优化，但为了保持 API 兼容性，不进一步调整
type AppError struct {
	Code    string // 错误码 (16 bytes)
	Message string // 错误消息 (16 bytes)
	Err     error  // 底层错误 (16 bytes interface)
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 实现错误包装接口，支持 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithError 包装底层错误
func (e *AppError) WithError(err error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Err:     err,
	}
}

// WithMessage 添加自定义消息
func (e *AppError) WithMessage(msg string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: msg,
		Err:     e.Err,
	}
}

// WithLanguage 根据请求上下文获取本地化的错误消息
// 如果请求为 nil 或无法获取语言，则返回原始消息
func (e *AppError) WithLanguage(r *http.Request) *AppError {
	if r == nil {
		return e
	}

	// 根据错误码获取 i18n 键
	key := getI18nKey(e.Code)
	if key == "" {
		return e
	}

	// 获取本地化消息
	localizedMsg := i18n.T(r, key)
	if localizedMsg == key {
		// 如果翻译不存在，使用原始消息
		return e
	}

	return &AppError{
		Code:    e.Code,
		Message: localizedMsg,
		Err:     e.Err,
	}
}

// getI18nKey 根据错误码获取 i18n 键
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

// 预定义的错误类型
var (
	// Redis 相关错误
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

	// 配置相关错误
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

	// 应用初始化错误
	ErrAppInit = &AppError{
		Code:    "APP_INIT_ERR",
		Message: "应用初始化失败",
	}

	// HTTP 相关错误
	ErrHTTPRequest = &AppError{
		Code:    "HTTP_REQ_ERR",
		Message: "HTTP 请求失败",
	}
	ErrHTTPResponse = &AppError{
		Code:    "HTTP_RESP_ERR",
		Message: "HTTP 响应处理失败",
	}

	// 数据相关错误
	ErrDataLoad = &AppError{
		Code:    "DATA_LOAD_ERR",
		Message: "数据加载失败",
	}
	ErrDataParse = &AppError{
		Code:    "DATA_PARSE_ERR",
		Message: "数据解析失败",
	}

	// 缓存相关错误
	ErrCacheOperation = &AppError{
		Code:    "CACHE_OP_ERR",
		Message: "缓存操作失败",
	}

	// 参数验证错误
	ErrInvalidParameter = &AppError{
		Code:    "INVALID_PARAM_ERR",
		Message: "无效的参数",
	}

	// 任务执行错误
	ErrTaskExecution = &AppError{
		Code:    "TASK_EXEC_ERR",
		Message: "任务执行失败",
	}
)

// Wrap 包装错误，提供上下文信息
func Wrap(err error, code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Wrapf 使用格式化字符串包装错误
func Wrapf(err error, code, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

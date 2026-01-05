// Package errors 提供了统一的错误处理功能。
// 定义了应用错误类型和预定义的错误变量，支持错误包装和上下文信息。
package errors

import (
	// 标准库
	"fmt"
)

// AppError 应用错误类型，提供统一的错误处理
type AppError struct {
	Code    string // 错误码 (16 bytes)
	Message string // 错误消息 (16 bytes)
	Err     error  // 底层错误 (16 bytes interface)
	// 注意：字段顺序已优化以减少内存对齐填充
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

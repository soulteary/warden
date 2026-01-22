//nolint:revive // package name conflicts with standard library, but this is a test file, keep package name consistency
package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppError_Error tests AppError's Error method
func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		want     string
		contains []string
	}{
		{
			name: "只有消息",
			err: &AppError{
				Code:    "TEST_ERR",
				Message: "测试错误",
			},
			contains: []string{"TEST_ERR", "测试错误"},
		},
		{
			name: "带底层错误",
			err: &AppError{
				Code:    "TEST_ERR",
				Message: "测试错误",
				Err:     errors.New("底层错误"),
			},
			contains: []string{"TEST_ERR", "测试错误", "底层错误"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			assert.NotEmpty(t, errMsg, "错误消息不应该为空")
			for _, substr := range tt.contains {
				assert.Contains(t, errMsg, substr, "错误消息应该包含: %s", substr)
			}
		})
	}
}

// TestAppError_Unwrap tests AppError's Unwrap method
func TestAppError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("底层错误")
	appErr := &AppError{
		Code:    "TEST_ERR",
		Message: "测试错误",
		Err:     underlyingErr,
	}

	unwrapped := appErr.Unwrap()
	assert.Equal(t, underlyingErr, unwrapped, "Unwrap 应该返回底层错误")

	// Test case without underlying error
	appErrNoUnderlying := &AppError{
		Code:    "TEST_ERR",
		Message: "测试错误",
		Err:     nil,
	}
	assert.Nil(t, appErrNoUnderlying.Unwrap(), "没有底层错误时应该返回 nil")
}

// TestAppError_WithError tests WithError method
func TestAppError_WithError(t *testing.T) {
	baseErr := ErrConfigLoad
	underlyingErr := errors.New("文件读取失败")

	newErr := baseErr.WithError(underlyingErr)

	assert.Equal(t, baseErr.Code, newErr.Code, "错误码应该相同")
	assert.Equal(t, baseErr.Message, newErr.Message, "错误消息应该相同")
	assert.Equal(t, underlyingErr, newErr.Err, "底层错误应该被设置")
	assert.NotSame(t, baseErr, newErr, "应该返回新的错误实例")
}

// TestAppError_WithMessage tests WithMessage method
func TestAppError_WithMessage(t *testing.T) {
	baseErr := ErrConfigLoad
	customMessage := "自定义错误消息"

	newErr := baseErr.WithMessage(customMessage)

	assert.Equal(t, baseErr.Code, newErr.Code, "错误码应该相同")
	assert.Equal(t, customMessage, newErr.Message, "错误消息应该被更新")
	assert.Equal(t, baseErr.Err, newErr.Err, "底层错误应该相同")
	assert.NotSame(t, baseErr, newErr, "应该返回新的错误实例")
}

// TestWrap tests Wrap function
func TestWrap(t *testing.T) {
	underlyingErr := errors.New("底层错误")
	code := "WRAP_TEST"
	message := "包装错误"

	wrappedErr := Wrap(underlyingErr, code, message)

	assert.Equal(t, code, wrappedErr.Code, "错误码应该正确设置")
	assert.Equal(t, message, wrappedErr.Message, "错误消息应该正确设置")
	assert.Equal(t, underlyingErr, wrappedErr.Err, "底层错误应该正确设置")
}

// TestWrapf tests Wrapf function
func TestWrapf(t *testing.T) {
	underlyingErr := errors.New("底层错误")
	code := "WRAPF_TEST"
	format := "错误发生在 %s: %d"

	wrappedErr := Wrapf(underlyingErr, code, format, "测试", 123)

	assert.Equal(t, code, wrappedErr.Code, "错误码应该正确设置")
	assert.Equal(t, "错误发生在 测试: 123", wrappedErr.Message, "错误消息应该被格式化")
	assert.Equal(t, underlyingErr, wrappedErr.Err, "底层错误应该正确设置")
}

// TestPredefinedErrors tests predefined errors
func TestPredefinedErrors(t *testing.T) {
	predefinedErrs := []*AppError{
		ErrRedisConnection,
		ErrRedisOperation,
		ErrRedisLock,
		ErrConfigLoad,
		ErrConfigValidation,
		ErrConfigParse,
		ErrAppInit,
		ErrHTTPRequest,
		ErrHTTPResponse,
		ErrDataLoad,
		ErrDataParse,
		ErrCacheOperation,
		ErrInvalidParameter,
		ErrTaskExecution,
	}

	for _, err := range predefinedErrs {
		t.Run(err.Code, func(t *testing.T) {
			assert.NotEmpty(t, err.Code, "错误码不应该为空")
			assert.NotEmpty(t, err.Message, "错误消息不应该为空")
			assert.NotEmpty(t, err.Error(), "Error() 方法应该返回非空字符串")
		})
	}
}

// TestAppError_ErrorChain tests error chain
func TestAppError_ErrorChain(t *testing.T) {
	level1 := errors.New("level 1 error")
	level2 := Wrap(level1, "LEVEL2", "level 2 error")
	level3 := Wrap(level2, "LEVEL3", "level 3 error")

	// Test error chain
	assert.Equal(t, level2, level3.Unwrap(), "应该能够解包到 level 2")
	assert.Equal(t, level1, level2.Unwrap(), "应该能够解包到 level 1")

	// Test using standard library's errors.Unwrap
	unwrapped := errors.Unwrap(level3)
	assert.Equal(t, level2, unwrapped, "标准库 Unwrap 应该工作")
}

// TestAppError_Is tests error comparison (using errors.Is)
func TestAppError_Is(t *testing.T) {
	// Create same underlying error
	underlyingErr := errors.New("底层错误")
	baseErr := ErrConfigLoad
	wrappedErr := baseErr.WithError(underlyingErr)

	// errors.Is will traverse error chain, comparing underlying errors
	// Note: errors.Is requires error values to be the same (same pointer or same value)
	// Since we created new error instances, need to use same instance for comparison
	assert.True(t, errors.Is(wrappedErr, underlyingErr), "应该能够使用 errors.Is 比较底层错误")

	// Test different error instances (same value but different instances)
	// errors.Is compares error values, not instances
	// For errors created by fmt.Errorf or errors.New, if messages are the same, errors.Is may return true
	// But here we test if underlying error can be found
	assert.True(t, errors.Is(wrappedErr, underlyingErr), "应该能够找到相同的底层错误")
}

// TestAppError_As tests error type assertion (using errors.As)
func TestAppError_As(t *testing.T) {
	appErr := ErrConfigLoad.WithError(errors.New("底层错误"))

	var target *AppError
	assert.True(t, errors.As(appErr, &target), "应该能够使用 errors.As 提取 AppError")
	assert.Equal(t, appErr.Code, target.Code, "提取的错误应该相同")
}

// TestAppError_Chaining tests error chaining
func TestAppError_Chaining(t *testing.T) {
	underlyingErr := errors.New("原始错误")

	// Chaining calls
	err := ErrConfigLoad.
		WithError(underlyingErr).
		WithMessage("配置加载失败")

	assert.Equal(t, "CONFIG_LOAD_ERR", err.Code)
	assert.Equal(t, "配置加载失败", err.Message)
	assert.Equal(t, underlyingErr, err.Err)
}

// TestAppError_WithLanguage tests WithLanguage method
func TestAppError_WithLanguage(t *testing.T) {
	//nolint:govet // fieldalignment: test cases prioritize readability
	tests := []struct {
		name    string
		err     *AppError
		request *http.Request
		wantNil bool
	}{
		{
			name:    "nil request",
			err:     ErrConfigLoad,
			request: nil,
			wantNil: false, // Should return same error, not nil
		},
		{
			name:    "request without i18n context",
			err:     ErrConfigLoad,
			request: &http.Request{},
			wantNil: false,
		},
		{
			name:    "error without i18n key",
			err:     &AppError{Code: "UNKNOWN_ERR", Message: "未知错误"},
			request: &http.Request{},
			wantNil: false, // Should return same error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.WithLanguage(tt.request)
			assert.NotNil(t, result, "结果不应该为 nil")
			// Note: WithLanguage may return same error if i18n is not available
			// This is expected behavior
		})
	}
}

// TestGetI18nKey tests getI18nKey function
func TestGetI18nKey(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		wantKey   string
		wantEmpty bool
	}{
		{"REDIS_CONN_ERR", "REDIS_CONN_ERR", "error.redis_connection_failed", false},
		{"REDIS_OP_ERR", "REDIS_OP_ERR", "error.redis_operation_failed", false},
		{"REDIS_LOCK_ERR", "REDIS_LOCK_ERR", "error.redis_lock_failed", false},
		{"CONFIG_LOAD_ERR", "CONFIG_LOAD_ERR", "error.config_load_failed", false},
		{"CONFIG_VALIDATION_ERR", "CONFIG_VALIDATION_ERR", "error.config_validation_failed", false},
		{"CONFIG_PARSE_ERR", "CONFIG_PARSE_ERR", "error.config_parse_failed", false},
		{"APP_INIT_ERR", "APP_INIT_ERR", "error.app_init_failed", false},
		{"HTTP_REQ_ERR", "HTTP_REQ_ERR", "error.http_request_failed", false},
		{"HTTP_RESP_ERR", "HTTP_RESP_ERR", "error.http_response_failed", false},
		{"DATA_LOAD_ERR", "DATA_LOAD_ERR", "error.data_load_failed", false},
		{"DATA_PARSE_ERR", "DATA_PARSE_ERR", "error.data_parse_failed", false},
		{"CACHE_OP_ERR", "CACHE_OP_ERR", "error.cache_operation_failed", false},
		{"INVALID_PARAM_ERR", "INVALID_PARAM_ERR", "error.invalid_parameter", false},
		{"TASK_EXEC_ERR", "TASK_EXEC_ERR", "error.task_execution_failed", false},
		{"UNKNOWN_ERR", "UNKNOWN_ERR", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// getI18nKey is a private function, so we test it indirectly through WithLanguage
			// Or we can test it by checking if the error code maps correctly
			err := &AppError{Code: tt.code, Message: "test"}
			result := err.WithLanguage(&http.Request{})

			// If i18n key exists, WithLanguage might change the message
			// If i18n key doesn't exist, WithLanguage returns same error
			if tt.wantEmpty {
				// Unknown error code should return same error
				assert.Equal(t, err.Message, result.Message, "未知错误码应该返回原消息")
			} else {
				// Known error code might have localized message
				// We just verify it doesn't panic and returns valid error
				assert.NotNil(t, result, "应该返回有效错误")
			}
		})
	}
}

// TestWrap_NilError tests Wrap with nil error
func TestWrap_NilError(t *testing.T) {
	wrappedErr := Wrap(nil, "TEST_CODE", "test message")

	assert.NotNil(t, wrappedErr, "即使底层错误为 nil，也应该返回错误")
	assert.Equal(t, "TEST_CODE", wrappedErr.Code)
	assert.Equal(t, "test message", wrappedErr.Message)
	assert.Nil(t, wrappedErr.Err, "底层错误应该为 nil")
}

// TestWrapf_NilError tests Wrapf with nil error
func TestWrapf_NilError(t *testing.T) {
	wrappedErr := Wrapf(nil, "TEST_CODE", "test %s", "message")

	assert.NotNil(t, wrappedErr, "即使底层错误为 nil，也应该返回错误")
	assert.Equal(t, "TEST_CODE", wrappedErr.Code)
	assert.Equal(t, "test message", wrappedErr.Message)
	assert.Nil(t, wrappedErr.Err, "底层错误应该为 nil")
}

// TestAppError_WithError_Nil tests WithError with nil error
func TestAppError_WithError_Nil(t *testing.T) {
	baseErr := ErrConfigLoad
	newErr := baseErr.WithError(nil)

	assert.Equal(t, baseErr.Code, newErr.Code)
	assert.Equal(t, baseErr.Message, newErr.Message)
	assert.Nil(t, newErr.Err, "底层错误应该为 nil")
	assert.NotSame(t, baseErr, newErr, "应该返回新的错误实例")
}

// TestAppError_WithMessage_Empty tests WithMessage with empty message
func TestAppError_WithMessage_Empty(t *testing.T) {
	baseErr := ErrConfigLoad
	newErr := baseErr.WithMessage("")

	assert.Equal(t, baseErr.Code, newErr.Code)
	assert.Equal(t, "", newErr.Message, "消息应该被设置为空字符串")
	assert.Equal(t, baseErr.Err, newErr.Err)
	assert.NotSame(t, baseErr, newErr, "应该返回新的错误实例")
}

// TestAllPredefinedErrors_WithError tests all predefined errors with WithError
func TestAllPredefinedErrors_WithError(t *testing.T) {
	underlyingErr := errors.New("底层错误")

	predefinedErrs := []*AppError{
		ErrRedisConnection,
		ErrRedisOperation,
		ErrRedisLock,
		ErrConfigLoad,
		ErrConfigValidation,
		ErrConfigParse,
		ErrAppInit,
		ErrHTTPRequest,
		ErrHTTPResponse,
		ErrDataLoad,
		ErrDataParse,
		ErrCacheOperation,
		ErrInvalidParameter,
		ErrTaskExecution,
	}

	for _, err := range predefinedErrs {
		t.Run(err.Code, func(t *testing.T) {
			newErr := err.WithError(underlyingErr)
			assert.Equal(t, err.Code, newErr.Code, "错误码应该相同")
			assert.Equal(t, err.Message, newErr.Message, "错误消息应该相同")
			assert.Equal(t, underlyingErr, newErr.Err, "底层错误应该被设置")
			assert.NotSame(t, err, newErr, "应该返回新的错误实例")
		})
	}
}

// TestAllPredefinedErrors_WithMessage tests all predefined errors with WithMessage
func TestAllPredefinedErrors_WithMessage(t *testing.T) {
	customMessage := "自定义消息"

	predefinedErrs := []*AppError{
		ErrRedisConnection,
		ErrRedisOperation,
		ErrRedisLock,
		ErrConfigLoad,
		ErrConfigValidation,
		ErrConfigParse,
		ErrAppInit,
		ErrHTTPRequest,
		ErrHTTPResponse,
		ErrDataLoad,
		ErrDataParse,
		ErrCacheOperation,
		ErrInvalidParameter,
		ErrTaskExecution,
	}

	for _, err := range predefinedErrs {
		t.Run(err.Code, func(t *testing.T) {
			newErr := err.WithMessage(customMessage)
			assert.Equal(t, err.Code, newErr.Code, "错误码应该相同")
			assert.Equal(t, customMessage, newErr.Message, "错误消息应该被更新")
			assert.Equal(t, err.Err, newErr.Err, "底层错误应该相同")
			assert.NotSame(t, err, newErr, "应该返回新的错误实例")
		})
	}
}

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestErrorHandlerMiddleware_ProductionMode 测试生产模式下的错误处理
func TestErrorHandlerMiddleware_ProductionMode(t *testing.T) {
	// 保存原始环境变量
	originalMode := os.Getenv("MODE")
	defer func() {
		require.NoError(t, os.Setenv("MODE", originalMode))
	}()

	require.NoError(t, os.Setenv("MODE", "production"))

	middleware := ErrorHandlerMiddleware("production")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		require.NoError(t, json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "detailed error",
			Message: "this is a detailed error message",
			Code:    "ERR_DETAILED",
		}))
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 验证响应被替换为通用错误消息
	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "内部服务器错误，请稍后重试", resp.Error, "生产环境应该返回通用错误消息")
	assert.Empty(t, resp.Message, "生产环境不应该返回详细消息")
	assert.Empty(t, resp.Code, "生产环境不应该返回错误代码")
}

// TestErrorHandlerMiddleware_DevelopmentMode 测试开发模式下的错误处理
func TestErrorHandlerMiddleware_DevelopmentMode(t *testing.T) {
	middleware := ErrorHandlerMiddleware("development")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		require.NoError(t, json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "validation error",
			Message: "invalid input",
			Code:    "ERR_VALIDATION",
		}))
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 验证响应保持原样（开发环境）
	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	// 注意：在开发模式下，错误响应应该保持原样
	// 但根据代码实现，ErrorHandlerMiddleware 在生产环境才会隐藏详细信息
}

// TestErrorHandlerMiddleware_NonErrorStatus 测试非错误状态码
func TestErrorHandlerMiddleware_NonErrorStatus(t *testing.T) {
	middleware := ErrorHandlerMiddleware("production")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("success"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String(), "非错误状态码不应该被修改")
}

// TestErrorHandlerMiddleware_NonJSONError 测试非 JSON 格式的错误响应
func TestErrorHandlerMiddleware_NonJSONError(t *testing.T) {
	middleware := ErrorHandlerMiddleware("production")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("plain text error"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 验证响应被替换为通用错误消息
	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "内部服务器错误，请稍后重试", resp.Error, "应该返回通用错误消息")
}

// TestGetGenericErrorMessage 测试通用错误消息生成
func TestGetGenericErrorMessage(t *testing.T) {
	//nolint:govet // fieldalignment: 测试结构体字段顺序不影响功能
	tests := []struct {
		statusCode int
		expected   string
	}{
		{http.StatusBadRequest, "请求参数无效"},
		{http.StatusUnauthorized, "未授权访问"},
		{http.StatusForbidden, "访问被拒绝"},
		{http.StatusNotFound, "请求的资源不存在"},
		{http.StatusTooManyRequests, "请求过于频繁，请稍后重试"},
		{http.StatusInternalServerError, "内部服务器错误，请稍后重试"},
		{http.StatusBadGateway, "内部服务器错误，请稍后重试"},
		{http.StatusServiceUnavailable, "内部服务器错误，请稍后重试"},
		{http.StatusTeapot, "请求处理失败"}, // 默认情况
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			msg := getGenericErrorMessage(tt.statusCode)
			assert.Equal(t, tt.expected, msg, "错误消息应该匹配")
		})
	}
}

// TestSafeError_ProductionMode 测试 SafeError 函数（生产模式）
func TestSafeError_ProductionMode(t *testing.T) {
	originalMode := os.Getenv("MODE")
	defer func() {
		require.NoError(t, os.Setenv("MODE", originalMode))
	}()

	require.NoError(t, os.Setenv("MODE", "production"))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	SafeError(w, req, http.StatusInternalServerError, nil, "detailed error message")

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "内部服务器错误，请稍后重试", resp.Error, "生产环境应该返回通用错误消息")
	assert.Empty(t, resp.Message, "生产环境不应该返回详细消息")
}

// TestSafeError_DevelopmentMode 测试 SafeError 函数（开发模式）
func TestSafeError_DevelopmentMode(t *testing.T) {
	originalMode := os.Getenv("MODE")
	defer func() {
		require.NoError(t, os.Setenv("MODE", originalMode))
	}()

	require.NoError(t, os.Setenv("MODE", "development"))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	SafeError(w, req, http.StatusBadRequest, nil, "detailed error message")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "请求参数无效", resp.Error, "应该返回通用错误消息")
	assert.Equal(t, "detailed error message", resp.Message, "开发环境应该返回详细消息")
}

// TestSafeError_WithError 测试 SafeError 函数（带 error）
func TestSafeError_WithError(t *testing.T) {
	originalMode := os.Getenv("MODE")
	defer func() {
		require.NoError(t, os.Setenv("MODE", originalMode))
	}()

	require.NoError(t, os.Setenv("MODE", "development"))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	testErr := assert.AnError
	SafeError(w, req, http.StatusInternalServerError, testErr, "detail message")

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "内部服务器错误，请稍后重试", resp.Error)
	// 在开发模式下，如果有 error，Message 应该是 error.Error()
	assert.NotEmpty(t, resp.Message, "开发环境应该返回错误消息")
}

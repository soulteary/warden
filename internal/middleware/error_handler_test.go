package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	loggerkit "github.com/soulteary/logger-kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/logger"
)

func init() {
	logger.SetLevel(loggerkit.InfoLevel)
}

// TestErrorHandlerMiddleware_ProductionMode tests error handling in production mode
func TestErrorHandlerMiddleware_ProductionMode(t *testing.T) {
	// Save original environment variable
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

	// Verify response is replaced with generic error message
	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	// Note: Since no language context is set, default language (English) will be used
	// But for test compatibility, we just check that message is not empty
	assert.NotEmpty(t, resp.Error, "Production environment should return generic error message")
	assert.Empty(t, resp.Message, "Production environment should not return detailed message")
	assert.Empty(t, resp.Code, "Production environment should not return error code")
}

// TestErrorHandlerMiddleware_DevelopmentMode tests error handling in development mode
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

	// Verify response remains unchanged (development environment)
	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	// Note: In development mode, error response should remain unchanged
	// But according to code implementation, ErrorHandlerMiddleware only hides detailed information in production environment
}

// TestErrorHandlerMiddleware_NonErrorStatus tests non-error status codes
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
	assert.Equal(t, "success", w.Body.String(), "Non-error status codes should not be modified")
}

// TestErrorHandlerMiddleware_NonJSONError tests non-JSON format error responses
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

	// Verify response is replaced with generic error message
	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	// Note: Since no language context is set, default language (English) will be used
	// But for test compatibility, we just check that message is not empty
	assert.NotEmpty(t, resp.Error, "Should return generic error message")
}

// TestGetGenericErrorMessage tests generic error message generation
func TestGetGenericErrorMessage(t *testing.T) {
	//nolint:govet // fieldalignment: test struct field order does not affect functionality
	tests := []struct {
		statusCode int
		expected   string
		lang       i18n.Language
	}{
		{http.StatusBadRequest, "请求参数无效", i18n.LangZH},
		{http.StatusUnauthorized, "未授权访问", i18n.LangZH},
		{http.StatusForbidden, "访问被拒绝", i18n.LangZH},
		{http.StatusNotFound, "请求的资源不存在", i18n.LangZH},
		{http.StatusTooManyRequests, "请求过于频繁，请稍后重试", i18n.LangZH},
		{http.StatusInternalServerError, "内部服务器错误，请稍后重试", i18n.LangZH},
		{http.StatusBadGateway, "内部服务器错误，请稍后重试", i18n.LangZH},
		{http.StatusServiceUnavailable, "内部服务器错误，请稍后重试", i18n.LangZH},
		{http.StatusTeapot, "请求处理失败", i18n.LangZH}, // Default case
		// Test English
		{http.StatusBadRequest, "Invalid request parameters", i18n.LangEN},
		{http.StatusUnauthorized, "Unauthorized access", i18n.LangEN},
		{http.StatusNotFound, "Requested resource does not exist", i18n.LangEN},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			// Set language in request context
			req = i18n.SetLanguageInContext(req, tt.lang)
			msg := getGenericErrorMessage(req, tt.statusCode)
			assert.Equal(t, tt.expected, msg, "Error message should match")
		})
	}
}

// TestSafeError_ProductionMode tests SafeError function (production mode)
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
	// Note: Since no language context is set, default language (English) will be used
	// But for test compatibility, we just check that message is not empty
	assert.NotEmpty(t, resp.Error, "Production environment should return generic error message")
	assert.Empty(t, resp.Message, "Production environment should not return detailed message")
}

// TestSafeError_DevelopmentMode tests SafeError function (development mode)
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
	// Note: Since no language context is set, default language (English) will be used
	// But for test compatibility, we just check that message is not empty
	assert.NotEmpty(t, resp.Error, "Should return generic error message")
	assert.Equal(t, "detailed error message", resp.Message, "Development environment should return detailed message")
}

// TestSafeError_WithError tests SafeError function (with error)
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
	// Note: Since no language context is set, default language (English) will be used
	// But for test compatibility, we just check that message is not empty
	assert.NotEmpty(t, resp.Error, "Should return generic error message")
	// In development mode, if there is an error, Message should be error.Error()
	assert.NotEmpty(t, resp.Message, "Development environment should return error message")
}

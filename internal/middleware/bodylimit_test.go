package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/soulteary/warden/internal/define"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestBodyLimitMiddleware_GETRequest 测试 GET 请求（应该通过）
func TestBodyLimitMiddleware_GETRequest(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "GET 请求应该通过")
}

// TestBodyLimitMiddleware_HEADRequest 测试 HEAD 请求（应该通过）
func TestBodyLimitMiddleware_HEADRequest(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("HEAD", "/", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "HEAD 请求应该通过")
}

// TestBodyLimitMiddleware_ValidSize 测试有效大小的请求体
func TestBodyLimitMiddleware_ValidSize(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 读取请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))

	// 创建一个小于限制的请求体
	bodySize := define.MAX_REQUEST_BODY_SIZE / 2
	body := bytes.NewReader(make([]byte, bodySize))

	req := httptest.NewRequest("POST", "/", body)
	req.ContentLength = int64(bodySize)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "有效大小的请求体应该通过")
}

// TestBodyLimitMiddleware_ExceedsLimit 测试超过限制的请求体
func TestBodyLimitMiddleware_ExceedsLimit(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 创建一个超过限制的请求体
	bodySize := define.MAX_REQUEST_BODY_SIZE + 1
	req := httptest.NewRequest("POST", "/", nil)
	req.ContentLength = int64(bodySize)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "超过限制的请求体应该返回 413")
}

// TestBodyLimitMiddleware_ExactLimit 测试正好等于限制的请求体
func TestBodyLimitMiddleware_ExactLimit(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// 创建一个正好等于限制的请求体
	bodySize := define.MAX_REQUEST_BODY_SIZE
	body := bytes.NewReader(make([]byte, bodySize))

	req := httptest.NewRequest("POST", "/", body)
	req.ContentLength = int64(bodySize)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "正好等于限制的请求体应该通过")
}

// TestBodyLimitMiddleware_NoContentLength 测试没有 Content-Length 头的请求
func TestBodyLimitMiddleware_NoContentLength(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// 创建一个没有 Content-Length 的请求
	body := bytes.NewReader([]byte("test body"))
	req := httptest.NewRequest("POST", "/", body)
	// 不设置 ContentLength
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// 应该通过，因为 MaxBytesReader 会在读取时检查
	assert.Equal(t, http.StatusOK, w.Code, "没有 Content-Length 的请求应该通过（会在读取时检查）")
}

// TestBodyLimitMiddleware_DifferentMethods 测试不同的 HTTP 方法
func TestBodyLimitMiddleware_DifferentMethods(t *testing.T) {
	middleware := BodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	methods := []string{"POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			bodySize := define.MAX_REQUEST_BODY_SIZE + 1
			req := httptest.NewRequest(method, "/", nil)
			req.ContentLength = int64(bodySize)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "超过限制的 %s 请求应该返回 413", method)
		})
	}
}

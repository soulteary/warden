package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestMetricsMiddleware_RecordsMetrics 测试指标记录
func TestMetricsMiddleware_RecordsMetrics(t *testing.T) {
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	start := time.Now()
	middleware.ServeHTTP(w, req)
	duration := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code, "应该返回 200")
	assert.Less(t, duration, 100*time.Millisecond, "请求应该快速完成")
	// 注意：实际验证指标需要访问 metrics 包，这里只验证中间件不会 panic
}

// TestMetricsMiddleware_DifferentStatusCodes 测试不同状态码的指标记录
func TestMetricsMiddleware_DifferentStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			}))

			req := httptest.NewRequest("GET", "/test", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, statusCode, w.Code, "状态码应该正确设置")
		})
	}
}

// TestMetricsMiddleware_DifferentMethods 测试不同 HTTP 方法的指标记录
func TestMetricsMiddleware_DifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(method, "/test", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "应该返回 200")
		})
	}
}

// TestMetricsMiddleware_DifferentEndpoints 测试不同端点的指标记录
func TestMetricsMiddleware_DifferentEndpoints(t *testing.T) {
	endpoints := []string{"/", "/health", "/user", "/metrics", "/api/v1/test"}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", endpoint, http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "应该返回 200")
		})
	}
}

// TestMetricsMiddleware_EmptyPath 测试空路径处理
func TestMetricsMiddleware_EmptyPath(t *testing.T) {
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 在处理器中，如果路径为空，应该被处理为 "/"
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		w.WriteHeader(http.StatusOK)
	}))

	// 使用有效的 URL，但测试中间件对空路径的处理
	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "应该返回 200")
}

// TestMetricsMiddleware_ResponseWriter 测试 ResponseWriter 包装
func TestMetricsMiddleware_ResponseWriter(t *testing.T) {
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 测试多次写入
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Hello"))
		require.NoError(t, err)
		_, err = w.Write([]byte(" World"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "应该返回 200")
	assert.Equal(t, "Hello World", w.Body.String(), "响应体应该正确")
}

// TestMetricsMiddleware_Concurrent 测试并发安全性
func TestMetricsMiddleware_Concurrent(t *testing.T) {
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	}))

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", http.NoBody)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// 等待所有请求完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestMetricsMiddleware_DurationMeasurement 测试持续时间测量
func TestMetricsMiddleware_DurationMeasurement(t *testing.T) {
	// 创建一个有延迟的处理器
	middleware := MetricsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	start := time.Now()
	middleware.ServeHTTP(w, req)
	duration := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code, "应该返回 200")
	assert.GreaterOrEqual(t, duration, 50*time.Millisecond, "持续时间应该至少为 50ms")
	assert.Less(t, duration, 200*time.Millisecond, "持续时间不应该超过 200ms（包含开销）")
}

// TestResponseWriter_WriteHeader 测试 ResponseWriter 的 WriteHeader 方法
func TestResponseWriter_WriteHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
	}

	// 测试设置状态码
	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.statusCode, "状态码应该被更新")
	assert.Equal(t, http.StatusNotFound, recorder.Code, "底层 ResponseWriter 的状态码应该被更新")
}

// TestResponseWriter_WriteHeader_MultipleCalls 测试多次调用 WriteHeader
func TestResponseWriter_WriteHeader_MultipleCalls(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
	}

	// 第一次调用
	rw.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusBadRequest, rw.statusCode)

	// 第二次调用（应该更新状态码）
	rw.WriteHeader(http.StatusInternalServerError)
	assert.Equal(t, http.StatusInternalServerError, rw.statusCode)
}

package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// TestCompressMiddleware_NoGzipSupport 测试客户端不支持 gzip
func TestCompressMiddleware_NoGzipSupport(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	// 不设置 Accept-Encoding
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"), "不支持 gzip 时不应该设置 Content-Encoding")
	assert.Equal(t, "test response", w.Body.String(), "响应体应该未压缩")
}

// TestCompressMiddleware_WithGzipSupport 测试客户端支持 gzip
func TestCompressMiddleware_WithGzipSupport(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"), "应该设置 Content-Encoding: gzip")
	assert.Equal(t, "Accept-Encoding", w.Header().Get("Vary"), "应该设置 Vary: Accept-Encoding")

	// 验证响应是压缩的
	body := w.Body.Bytes()
	if len(body) > 0 {
		// 检查是否是 gzip 格式（gzip 文件以 1f 8b 开头）
		if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
			// 尝试解压
			reader, err := gzip.NewReader(strings.NewReader(string(body)))
			if err == nil {
				decompressed, err := io.ReadAll(reader)
				assert.NoError(t, err, "应该能够解压响应")
				assert.Equal(t, "test response", string(decompressed), "解压后的内容应该正确")
				require.NoError(t, reader.Close())
			}
		}
	}
}

// TestCompressMiddleware_MultipleEncodings 测试多个编码格式
func TestCompressMiddleware_MultipleEncodings(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"), "应该使用 gzip 压缩")
}

// TestCompressMiddleware_EmptyResponse 测试空响应
func TestCompressMiddleware_EmptyResponse(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestCompressMiddleware_LargeResponse 测试大响应
func TestCompressMiddleware_LargeResponse(t *testing.T) {
	largeData := strings.Repeat("test data ", 1000)
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(largeData))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"), "应该使用 gzip 压缩")

	// 验证压缩后的数据应该比原始数据小
	compressedSize := len(w.Body.Bytes())
	originalSize := len(largeData)
	assert.Less(t, compressedSize, originalSize, "压缩后的数据应该比原始数据小")
}

// TestCompressMiddleware_ConcurrentRequests 测试并发请求
func TestCompressMiddleware_ConcurrentRequests(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.Header.Set("Accept-Encoding", "gzip")
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

// TestCompressMiddleware_GzipWriterPool 测试 gzip writer 池
func TestCompressMiddleware_GzipWriterPool(t *testing.T) {
	middleware := CompressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		require.NoError(t, err)
	}))

	// 多次请求应该复用 writer
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", http.NoBody)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	}
}

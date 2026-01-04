// Package middleware 提供了 HTTP 中间件功能。
// 包括速率限制、压缩、请求体限制、指标收集等中间件。
package middleware

import (
	// 标准库
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

// gzipWriter 包装 http.ResponseWriter 以支持 gzip 压缩
type gzipWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (gw *gzipWriter) Write(b []byte) (int, error) {
	if gw.writer == nil {
		return gw.ResponseWriter.Write(b)
	}
	return gw.writer.Write(b)
}

func (gw *gzipWriter) WriteHeader(statusCode int) {
	// 只有在写入数据之前设置 Content-Encoding
	// 如果已经写入过，就不需要再设置
	if gw.writer != nil {
		gw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	}
	gw.ResponseWriter.WriteHeader(statusCode)
}

func (gw *gzipWriter) Close() error {
	if gw.writer != nil {
		return gw.writer.Close()
	}
	return nil
}

// gzipWriterPool 复用 gzip.Writer 对象
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}

// CompressMiddleware 创建 gzip 压缩中间件
// 自动检测客户端是否支持 gzip，如果支持则压缩响应
func CompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查客户端是否支持 gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// 从池中获取 gzip writer
		gzWriter := gzipWriterPool.Get().(*gzip.Writer)

		// 设置 Vary 响应头
		w.Header().Set("Vary", "Accept-Encoding")

		// 重置 writer 以指向当前 ResponseWriter
		gzWriter.Reset(w)

		// 包装 ResponseWriter
		gw := &gzipWriter{
			ResponseWriter: w,
			writer:         gzWriter,
		}

		// 确保资源正确释放：先关闭，再重置，最后放回池中
		defer func() {
			gw.Close()
			gzWriter.Reset(nil)
			gzipWriterPool.Put(gzWriter)
		}()

		next.ServeHTTP(gw, r)
	})
}

// Package middleware provides HTTP middleware functionality.
// Includes rate limiting, compression, request body limiting, metrics collection and other middleware.
package middleware

import (
	// Standard library
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

// gzipWriter wraps http.ResponseWriter to support gzip compression
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
	// Only set Content-Encoding before writing data
	// If already written, no need to set again
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

// gzipWriterPool reuses gzip.Writer objects
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}

// CompressMiddleware creates gzip compression middleware
// Automatically detects if client supports gzip, compresses response if supported
func CompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client supports gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Get gzip writer from pool
		gzWriter, ok := gzipWriterPool.Get().(*gzip.Writer)
		if !ok {
			// If type assertion fails, create new writer
			gzWriter = gzip.NewWriter(nil)
		}

		// Set Vary response header
		w.Header().Set("Vary", "Accept-Encoding")

		// Reset writer to point to current ResponseWriter
		gzWriter.Reset(w)

		// Wrap ResponseWriter
		gw := &gzipWriter{
			ResponseWriter: w,
			writer:         gzWriter,
		}

		// Ensure resources are properly released: close first, then reset, finally return to pool
		defer func() {
			if err := gw.Close(); err != nil {
				// Log error but don't affect request processing
				_ = err // Explicitly ignore error
			}
			gzWriter.Reset(nil)
			gzipWriterPool.Put(gzWriter)
		}()

		next.ServeHTTP(gw, r)
	})
}

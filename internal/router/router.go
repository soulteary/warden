// Package router provides HTTP routing functionality.
// Includes request logging, JSON responses, health checks and other route handlers.
package router

import (
	// Standard library
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	// Third-party libraries
	loggerkit "github.com/soulteary/logger-kit"

	// Internal packages
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

// ProcessWithLogger injects logger-kit logger and request ID into request context.
// Does not perform access logging; that is done by the outer AccessLogMiddleware.
// Uses X-Request-ID header (read or generated) to match logger-kit middleware behavior.
func ProcessWithLogger(handler func(http.ResponseWriter, *http.Request)) http.Handler {
	lk := logger.GetLoggerKit()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = generateRequestID()
			r.Header.Set("X-Request-ID", rid)
		}
		ctx := r.Context()
		ctx = loggerkit.ContextWithRequestID(ctx, rid)
		ctx = loggerkit.ContextWithLogger(ctx, lk)
		r = r.WithContext(ctx)
		handler(w, r)
	})
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// AccessLogMiddleware creates access log middleware using logger-kit
//
// This middleware can be used at the outermost layer to ensure all requests (including authentication failures) are logged.
// Returns a middleware function that can wrap any http.Handler.
//
// Returns:
//   - func(http.Handler) http.Handler: HTTP middleware function
func AccessLogMiddleware() func(http.Handler) http.Handler {
	lkLog := logger.GetLoggerKit()
	return loggerkit.Middleware(loggerkit.MiddlewareConfig{
		Logger:           lkLog,
		SkipPaths:        define.SkipPathsHealthAndMetrics,
		IncludeRequestID: true,
		IncludeLatency:   true,
	})
}

// Package router provides HTTP routing functionality.
// Includes request logging, JSON responses, health checks and other route handlers.
package router

import (
	// Standard library
	"net/http"

	// Third-party libraries
	"github.com/justinas/alice"
	"github.com/rs/zerolog/hlog"
	loggerkit "github.com/soulteary/logger-kit"

	// Internal packages
	"github.com/soulteary/warden/internal/logger"
)

// ProcessWithLogger adds logging middleware to HTTP handlers
//
// This function uses alice middleware chain to add the following features to handlers:
// - Remote address logging: records client IP address
// - User agent logging: records client User-Agent
// - Referer logging: records HTTP Referer header
// - Request ID generation: generates unique ID for each request (read from Request-Id header or auto-generated)
//
// Note: Access logs are handled uniformly by the outer AccessLogMiddleware to avoid duplicate logging.
//
// Parameters:
//   - handler: HTTP request handler function
//
// Returns:
//   - http.Handler: wrapped HTTP handler with logging functionality
func ProcessWithLogger(handler func(http.ResponseWriter, *http.Request)) http.Handler {
	logInstance := logger.GetLogger()
	c := alice.New()
	c = c.Append(hlog.NewHandler(logInstance))

	// Add field handlers to ensure these fields are available in access logs
	c = c.Append(hlog.RemoteAddrHandler("ip"))
	c = c.Append(hlog.UserAgentHandler("user_agent"))
	c = c.Append(hlog.RefererHandler("referer"))
	c = c.Append(hlog.RequestIDHandler("req_id", "Request-Id"))

	// Note: Access log handler has been moved to outer AccessLogMiddleware to avoid duplicate logging

	return c.Then(http.HandlerFunc(handler))
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
		SkipPaths:        []string{"/healthz", "/metrics"},
		IncludeRequestID: true,
		IncludeLatency:   true,
	})
}

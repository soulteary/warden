// Package logger provides logging functionality.
// This package now delegates to logger-kit for logging functionality,
// while keeping the utility functions for backward compatibility.
package logger

import (
	// Standard library
	"net/http"
	"net/url"
	"strings"

	// Third-party libraries
	"github.com/rs/zerolog"
	loggerkit "github.com/soulteary/logger-kit"
	secure "github.com/soulteary/secure-kit"
)

// log is the global logger-kit instance
var log *loggerkit.Logger

// zerologInstance is kept in sync with log.Zerolog() so ZerologPtr() can return
// *zerolog.Logger without requiring logger-kit to expose ZerologPtr().
var zerologInstance zerolog.Logger
var zerologPtr *zerolog.Logger

func init() {
	// Initialize logger using logger-kit
	log = loggerkit.New(loggerkit.Config{
		Level:          loggerkit.ParseLevelFromEnv("LOG_LEVEL", loggerkit.InfoLevel),
		Format:         loggerkit.FormatJSON,
		ServiceName:    "warden",
		ServiceVersion: "0.7.0",
	})
	zerologInstance = log.Zerolog()
	zerologPtr = &zerologInstance
}

// GetLogger returns zerolog.Logger for backward compatibility with code that
// requires zerolog (e.g. middleware-kit Config). Prefer GetLoggerKit or
// ZerologPtr for new code.
func GetLogger() zerolog.Logger {
	return log.Zerolog()
}

// GetLoggerKit gets the logger-kit Logger instance.
func GetLoggerKit() *loggerkit.Logger {
	return log
}

// ZerologPtr returns a pointer to the underlying zerolog.Logger for use when
// filling middleware-kit Config.Logger (e.g. RateLimit, APIKey, BodyLimit).
// Implemented via log.Zerolog() so warden works with published logger-kit
// that may not expose ZerologPtr().
func ZerologPtr() *zerolog.Logger {
	zerologInstance = log.Zerolog()
	return zerologPtr
}

// FromRequest returns the request-scoped logger from the request context,
// or the default logger if not set. Use this instead of zerolog/hlog in
// handlers and middleware.
func FromRequest(r *http.Request) *loggerkit.Logger {
	return loggerkit.LoggerFromRequest(r)
}

// SetLevel sets log level at runtime (logger-kit only).
func SetLevel(level loggerkit.Level) {
	log.SetLevel(level)
}

// SanitizeString sanitizes sensitive information
// Performs partial sanitization on strings that may contain sensitive information
// Uses secure-kit's MaskString for consistent masking behavior
func SanitizeString(s string) string {
	return secure.MaskString(s, 2)
}

// SanitizeHeader sanitizes HTTP header information
func SanitizeHeader(header string) string {
	// Sanitize sensitive headers like Authorization
	lowerHeader := strings.ToLower(header)
	if strings.Contains(lowerHeader, "authorization") ||
		strings.Contains(lowerHeader, "token") ||
		strings.Contains(lowerHeader, "api-key") {
		return SanitizeString(header)
	}
	return header
}

// SanitizePhone sanitizes phone number
func SanitizePhone(phone string) string {
	return secure.MaskString(phone, 2)
}

// SanitizeEmail sanitizes email address
func SanitizeEmail(email string) string {
	return secure.MaskString(email, 2)
}

// SanitizeURL sanitizes URL by masking sensitive query parameters (phone, mail, email)
// Parameter names are matched case-insensitively
func SanitizeURL(u *url.URL) string {
	if u == nil {
		return ""
	}

	// Create a copy to avoid modifying the original
	sanitized := *u
	query := u.Query()

	// Sanitize sensitive query parameters (case-insensitive matching)
	sensitiveParams := []string{"phone", "mail", "email"}
	for key, values := range query {
		keyLower := strings.ToLower(key)
		for _, param := range sensitiveParams {
			if keyLower == param {
				sanitizedValues := make([]string, len(values))
				for i, v := range values {
					sanitizedValues[i] = secure.MaskString(v, 2)
				}
				query[key] = sanitizedValues
				break
			}
		}
	}

	sanitized.RawQuery = query.Encode()
	return sanitized.String()
}

// SanitizeURLString sanitizes URL string by parsing and masking sensitive query parameters
func SanitizeURLString(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		// If parsing fails, return original string (better than empty)
		return urlStr
	}

	return SanitizeURL(u)
}

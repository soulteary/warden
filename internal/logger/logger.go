// Package logger provides logging functionality.
// This package now delegates to logger-kit for logging functionality,
// while keeping the utility functions for backward compatibility.
package logger

import (
	// Standard library
	"net/url"
	"strings"

	// Third-party libraries
	"github.com/rs/zerolog"
	loggerkit "github.com/soulteary/logger-kit"
	secure "github.com/soulteary/secure-kit"
)

// log is the global logger-kit instance
var log *loggerkit.Logger

func init() {
	// Initialize logger using logger-kit
	log = loggerkit.New(loggerkit.Config{
		Level:          loggerkit.ParseLevelFromEnv("LOG_LEVEL", loggerkit.InfoLevel),
		Format:         loggerkit.FormatJSON,
		ServiceName:    "warden",
		ServiceVersion: "0.7.0",
	})
}

// GetLogger gets zerolog.Logger instance for backward compatibility
func GetLogger() zerolog.Logger {
	return log.Zerolog()
}

// GetLoggerKit gets the logger-kit Logger instance
func GetLoggerKit() *loggerkit.Logger {
	return log
}

// SetLevel sets log level (for runtime adjustment)
func SetLevel(level zerolog.Level) {
	// Set zerolog global level for backward compatibility
	zerolog.SetGlobalLevel(level)

	// Convert zerolog.Level to loggerkit.Level
	var lkLevel loggerkit.Level
	switch level {
	case zerolog.DebugLevel:
		lkLevel = loggerkit.DebugLevel
	case zerolog.InfoLevel:
		lkLevel = loggerkit.InfoLevel
	case zerolog.WarnLevel:
		lkLevel = loggerkit.WarnLevel
	case zerolog.ErrorLevel:
		lkLevel = loggerkit.ErrorLevel
	case zerolog.FatalLevel:
		lkLevel = loggerkit.FatalLevel
	case zerolog.TraceLevel:
		lkLevel = loggerkit.TraceLevel
	default:
		lkLevel = loggerkit.InfoLevel
	}
	log.SetLevel(lkLevel)
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

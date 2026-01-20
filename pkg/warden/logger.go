package warden

import (
	"net/url"
	"strings"
)

// sanitizeString sanitizes sensitive information
// Performs partial sanitization on strings that may contain sensitive information
func sanitizeString(s string) string {
	if s == "" {
		return s
	}

	// If string is short, only show first and last characters
	if len(s) <= 4 {
		return "***"
	}

	// Show first 2 characters and last 2 characters, replace middle with *
	prefix := s[:2]
	suffix := s[len(s)-2:]
	masked := strings.Repeat("*", len(s)-4)
	return prefix + masked + suffix
}

// sanitizePhone sanitizes phone number
func sanitizePhone(phone string) string {
	return sanitizeString(phone)
}

// sanitizeEmail sanitizes email address
func sanitizeEmail(email string) string {
	return sanitizeString(email)
}

// sanitizeURL sanitizes URL by masking sensitive query parameters (phone, mail, email)
// Parameter names are matched case-insensitively
func sanitizeURL(u *url.URL) string {
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
					sanitizedValues[i] = sanitizeString(v)
				}
				query[key] = sanitizedValues
				break
			}
		}
	}

	sanitized.RawQuery = query.Encode()
	return sanitized.String()
}

// sanitizeURLString sanitizes URL string by parsing and masking sensitive query parameters
func sanitizeURLString(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		// If parsing fails, return original string (better than empty)
		return urlStr
	}

	return sanitizeURL(u)
}

// Logger defines the interface for logging operations.
// This allows the SDK to work with different logging libraries
// (e.g., zerolog, logrus, standard log).
type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
}

// NoOpLogger is a no-op implementation of Logger that discards all log messages.
// This is used as the default logger when no logger is provided.
type NoOpLogger struct{}

// Debug discards debug messages.
func (n *NoOpLogger) Debug(msg string) {}

// Debugf discards formatted debug messages.
func (n *NoOpLogger) Debugf(format string, args ...interface{}) {}

// Info discards info messages.
func (n *NoOpLogger) Info(msg string) {}

// Infof discards formatted info messages.
func (n *NoOpLogger) Infof(format string, args ...interface{}) {}

// Warn discards warning messages.
func (n *NoOpLogger) Warn(msg string) {}

// Warnf discards formatted warning messages.
func (n *NoOpLogger) Warnf(format string, args ...interface{}) {}

// Error discards error messages.
func (n *NoOpLogger) Error(msg string) {}

// Errorf discards formatted error messages.
func (n *NoOpLogger) Errorf(format string, args ...interface{}) {}

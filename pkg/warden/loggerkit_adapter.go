package warden

import (
	loggerkit "github.com/soulteary/logger-kit"
)

// LoggerKitAdapter adapts logger-kit Logger to the SDK Logger interface.
type LoggerKitAdapter struct {
	logger *loggerkit.Logger
}

// NewLoggerKitAdapter creates a new LoggerKitAdapter.
func NewLoggerKitAdapter(l *loggerkit.Logger) *LoggerKitAdapter {
	if l == nil {
		l = loggerkit.Default()
	}
	return &LoggerKitAdapter{logger: l}
}

// Debug logs a debug message using logger-kit.
func (l *LoggerKitAdapter) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message using logger-kit.
func (l *LoggerKitAdapter) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

// Info logs an info message using logger-kit.
func (l *LoggerKitAdapter) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof logs a formatted info message using logger-kit.
func (l *LoggerKitAdapter) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

// Warn logs a warning message using logger-kit.
func (l *LoggerKitAdapter) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message using logger-kit.
func (l *LoggerKitAdapter) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

// Error logs an error message using logger-kit.
func (l *LoggerKitAdapter) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Errorf logs a formatted error message using logger-kit.
func (l *LoggerKitAdapter) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

// Ensure LoggerKitAdapter implements Logger at compile time.
var _ Logger = (*LoggerKitAdapter)(nil)

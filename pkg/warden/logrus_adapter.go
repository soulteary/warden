package warden

import (
	"github.com/sirupsen/logrus"
)

// LogrusAdapter adapts logrus.Logger to the SDK Logger interface.
type LogrusAdapter struct {
	logger *logrus.Logger
}

// NewLogrusAdapter creates a new LogrusAdapter.
func NewLogrusAdapter(logger *logrus.Logger) *LogrusAdapter {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &LogrusAdapter{logger: logger}
}

// Debug logs a debug message using logrus.
func (l *LogrusAdapter) Debug(msg string) {
	l.logger.Debug(msg)
}

// Debugf logs a formatted debug message using logrus.
func (l *LogrusAdapter) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Info logs an info message using logrus.
func (l *LogrusAdapter) Info(msg string) {
	l.logger.Info(msg)
}

// Infof logs a formatted info message using logrus.
func (l *LogrusAdapter) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Warn logs a warning message using logrus.
func (l *LogrusAdapter) Warn(msg string) {
	l.logger.Warn(msg)
}

// Warnf logs a formatted warning message using logrus.
func (l *LogrusAdapter) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

// Error logs an error message using logrus.
func (l *LogrusAdapter) Error(msg string) {
	l.logger.Error(msg)
}

// Errorf logs a formatted error message using logrus.
func (l *LogrusAdapter) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

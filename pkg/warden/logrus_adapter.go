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

func (l *LogrusAdapter) Debug(msg string) {
	l.logger.Debug(msg)
}

func (l *LogrusAdapter) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *LogrusAdapter) Info(msg string) {
	l.logger.Info(msg)
}

func (l *LogrusAdapter) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *LogrusAdapter) Warn(msg string) {
	l.logger.Warn(msg)
}

func (l *LogrusAdapter) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *LogrusAdapter) Error(msg string) {
	l.logger.Error(msg)
}

func (l *LogrusAdapter) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

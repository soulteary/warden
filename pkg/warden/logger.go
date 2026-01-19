package warden

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

func (n *NoOpLogger) Debug(msg string)                          {}
func (n *NoOpLogger) Debugf(format string, args ...interface{}) {}
func (n *NoOpLogger) Info(msg string)                           {}
func (n *NoOpLogger) Infof(format string, args ...interface{})  {}
func (n *NoOpLogger) Warn(msg string)                           {}
func (n *NoOpLogger) Warnf(format string, args ...interface{})  {}
func (n *NoOpLogger) Error(msg string)                          {}
func (n *NoOpLogger) Errorf(format string, args ...interface{}) {}

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

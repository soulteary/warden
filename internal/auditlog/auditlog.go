// Package auditlog provides audit logging functionality for Warden service.
// It wraps the audit-kit library to provide convenient logging methods for
// user operations, access control events, and configuration changes.
package auditlog

import (
	"context"
	"sync"

	audit "github.com/soulteary/audit-kit/v2"
	loggerkit "github.com/soulteary/logger-kit/v3"
)

var (
	mu         sync.RWMutex
	logger     *audit.Logger
	loggerInit sync.Once
)

// Init initializes the audit logger with the given storage and config.
//
// With a nil storage the records are written to the structured application log
// (see logStorage), so an audit trail exists without any further setup. A
// storage passed here receives the records through the asynchronous writer.
func Init(storage audit.Storage, cfg *audit.Config) {
	loggerInit.Do(func() {
		if cfg == nil {
			cfg = audit.DefaultConfig()
		}

		var l *audit.Logger
		if storage == nil {
			// Written synchronously: a log line needs no queue, and a queue
			// could drop records under load.
			l = audit.NewLogger(logStorage{}, cfg)
		} else {
			l = audit.NewLoggerWithWriter(storage, cfg)
		}

		mu.Lock()
		logger = l
		mu.Unlock()
	})
}

// GetLogger returns the audit logger instance, initializing it with the
// application-log sink when Init has not been called.
func GetLogger() *audit.Logger {
	Init(nil, nil)
	return current()
}

func current() *audit.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return logger
}

// Stop stops the audit logger
func Stop() error {
	if l := current(); l != nil {
		return l.Stop()
	}
	return nil
}

// LogUserQuery records a user query event
func LogUserQuery(ctx context.Context, userID, identifier, identifierType, ip string, success bool, reason string) {
	l := GetLogger()
	if l == nil {
		return
	}

	result := audit.ResultSuccess
	if !success {
		result = audit.ResultFailure
	}

	l.LogAccess(ctx, audit.EventCustom, userID, "user_query", result,
		audit.WithRecordIP(ip),
		audit.WithRecordRequestID(loggerkit.RequestIDFromContext(ctx)),
		audit.WithRecordReason(reason),
		audit.WithRecordMetadata("identifier", identifier),
		audit.WithRecordMetadata("identifier_type", identifierType),
	)
}

// LogUserCreate records a user creation event
func LogUserCreate(ctx context.Context, userID, ip string) {
	l := GetLogger()
	if l == nil {
		return
	}

	l.LogAuth(ctx, audit.EventUserCreated, userID, audit.ResultSuccess,
		audit.WithRecordIP(ip),
	)
}

// LogUserUpdate records a user update event
func LogUserUpdate(ctx context.Context, userID, ip string) {
	l := GetLogger()
	if l == nil {
		return
	}

	l.LogAuth(ctx, audit.EventUserUpdated, userID, audit.ResultSuccess,
		audit.WithRecordIP(ip),
	)
}

// LogUserDelete records a user deletion event
func LogUserDelete(ctx context.Context, userID, ip string) {
	l := GetLogger()
	if l == nil {
		return
	}

	l.LogAuth(ctx, audit.EventUserDeleted, userID, audit.ResultSuccess,
		audit.WithRecordIP(ip),
	)
}

// LogConfigChange records a configuration change event (like log level)
func LogConfigChange(ctx context.Context, configKey, oldValue, newValue, ip, userAgent string) {
	l := GetLogger()
	if l == nil {
		return
	}

	record := audit.NewRecord(audit.EventCustom, audit.ResultSuccess).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithResource("config:"+configKey).
		WithMetadata("old_value", oldValue).
		WithMetadata("new_value", newValue).
		WithMetadata("config_key", configKey)

	l.Log(ctx, record)
}

// LogLevelChange records a request to change the log level through the
// log-level endpoint. principal is the authenticated caller (an HMAC key ID, a
// client certificate fingerprint or the API key constant); it is not a secret.
// A refused or invalid request is recorded as a failure with unchanged levels.
func LogLevelChange(ctx context.Context, principal, oldLevel, newLevel, ip, userAgent string, success bool) {
	result := audit.ResultSuccess
	if !success {
		result = audit.ResultFailure
	}

	record := audit.NewRecord(audit.EventCustom, result).
		WithIP(ip).
		WithUserAgent(userAgent).
		WithResource("config:log_level").
		WithMetadata("config_key", "log_level").
		WithMetadata("old_value", oldLevel).
		WithMetadata("new_value", newLevel).
		WithMetadata("principal", principal)
	record.RequestID = loggerkit.RequestIDFromContext(ctx)

	GetLogger().Log(ctx, record)
}

// LogAccessDenied records an access denied event
func LogAccessDenied(ctx context.Context, userID, resource, ip, reason string) {
	l := GetLogger()
	if l == nil {
		return
	}

	l.LogAccess(ctx, audit.EventAccessDenied, userID, resource, audit.ResultFailure,
		audit.WithRecordIP(ip),
		audit.WithRecordReason(reason),
	)
}

// LogAccessGranted records an access granted event
func LogAccessGranted(ctx context.Context, userID, resource, ip string) {
	l := GetLogger()
	if l == nil {
		return
	}

	l.LogAccess(ctx, audit.EventAccessGranted, userID, resource, audit.ResultSuccess,
		audit.WithRecordIP(ip),
	)
}

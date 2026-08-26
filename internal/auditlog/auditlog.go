// Package auditlog provides audit logging functionality for Warden service.
// It wraps the audit-kit library to provide convenient logging methods for
// user operations, access control events, and configuration changes.
package auditlog

import (
	"context"
	"sync"

	audit "github.com/soulteary/audit-kit"
)

var (
	logger     *audit.Logger
	loggerInit sync.Once
)

// Init initializes the audit logger with the given storage and config
func Init(storage audit.Storage, cfg *audit.Config) {
	loggerInit.Do(func() {
		if cfg == nil {
			cfg = audit.DefaultConfig()
		}

		if storage == nil {
			// Use no-op storage if none provided
			storage = audit.NewNoopStorage()
		}

		logger = audit.NewLoggerWithWriter(storage, cfg)
	})
}

// GetLogger returns the audit logger instance
func GetLogger() *audit.Logger {
	if logger == nil {
		// Initialize with no-op storage if not initialized
		Init(nil, nil)
	}
	return logger
}

// Stop stops the audit logger
func Stop() error {
	if logger != nil {
		return logger.Stop()
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

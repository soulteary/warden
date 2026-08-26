package auditlog

import (
	"context"
	"sync"
	"testing"

	audit "github.com/soulteary/audit-kit"
	"github.com/stretchr/testify/assert"
)

func TestAuditLogFunctions(t *testing.T) {
	// Initialize with no-op storage for testing
	storage := audit.NewNoopStorage()
	cfg := audit.DefaultConfig()
	cfg.Enabled = true

	// Reset logger for testing
	logger = nil
	loggerInit = sync.Once{}

	Init(storage, cfg)

	l := GetLogger()
	assert.NotNil(t, l)

	ctx := context.Background()

	// Test all logging functions (should not panic)
	t.Run("LogUserQuery Success", func(t *testing.T) {
		LogUserQuery(ctx, "user1", "test@example.com", "email", "127.0.0.1", true, "")
	})

	t.Run("LogUserQuery Failure", func(t *testing.T) {
		LogUserQuery(ctx, "", "unknown@example.com", "email", "127.0.0.1", false, "user_not_found")
	})

	t.Run("LogUserCreate", func(t *testing.T) {
		LogUserCreate(ctx, "user1", "127.0.0.1")
	})

	t.Run("LogUserUpdate", func(t *testing.T) {
		LogUserUpdate(ctx, "user1", "127.0.0.1")
	})

	t.Run("LogUserDelete", func(t *testing.T) {
		LogUserDelete(ctx, "user1", "127.0.0.1")
	})

	t.Run("LogConfigChange", func(t *testing.T) {
		LogConfigChange(ctx, "log_level", "info", "debug", "127.0.0.1", "curl/7.64.1")
	})

	t.Run("LogAccessDenied", func(t *testing.T) {
		LogAccessDenied(ctx, "user1", "/admin", "127.0.0.1", "unauthorized")
	})

	t.Run("LogAccessGranted", func(t *testing.T) {
		LogAccessGranted(ctx, "user1", "/api/users", "127.0.0.1")
	})

	// Test Stop
	err := Stop()
	assert.NoError(t, err)
}

func TestGetLoggerWithoutInit(t *testing.T) {
	// Reset logger
	logger = nil
	loggerInit = sync.Once{}

	// GetLogger should auto-initialize with no-op storage
	l := GetLogger()
	assert.NotNil(t, l)
}

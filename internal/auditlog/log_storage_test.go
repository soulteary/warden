package auditlog

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"

	loggerkit "github.com/soulteary/logger-kit/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	applog "github.com/soulteary/warden/internal/logger"
)

// With no storage configured, records go to the structured log rather than
// being discarded.
func TestDefaultSinkWritesRecordsToTheLog(t *testing.T) {
	logger = nil
	loggerInit = sync.Once{}
	var out bytes.Buffer
	t.Cleanup(UseLogger(loggerkit.New(loggerkit.Config{
		Level: loggerkit.InfoLevel, Format: loggerkit.FormatJSON, ServiceName: "warden", Output: &out,
	})))

	ctx := loggerkit.ContextWithRequestID(context.Background(), "req-1")
	LogUserQuery(ctx, "0123456789abcdef", "13***00", "phone", "203.0.113.7", true, "")

	var line map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &line), out.String())
	assert.Equal(t, true, line["audit"])
	assert.Equal(t, "user_query", line["resource"])
	assert.Equal(t, "success", line["result"])
	assert.Equal(t, "203.0.113.7", line["ip"])
	assert.Equal(t, "req-1", line["request_id"])
	assert.Equal(t, "0123************", line["user_id"], "the user ID is masked in the log")
	metadata, ok := line["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "13***00", metadata["identifier"])
	assert.Equal(t, "phone", metadata["identifier_type"])
	assert.NotContains(t, out.String(), "0123456789abcdef")

	out.Reset()
	LogUserQuery(ctx, "0123456789abcdef", "0123456789abcdef", "user_id", "203.0.113.7", true, "")
	assert.NotContains(t, out.String(), "0123456789abcdef", "a query by user ID does not log the ID in full")
	assert.Contains(t, out.String(), `"identifier":"0123************"`)

	out.Reset()
	LogLevelChange(ctx, "hmac:gateway", "info", "debug", "203.0.113.7", "curl", false)
	require.NoError(t, json.Unmarshal(out.Bytes(), &line), out.String())
	assert.Equal(t, "failure", line["result"])
	assert.Equal(t, "config:log_level", line["resource"])

	records, err := logStorage{}.Query(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, records)
	require.NoError(t, logStorage{}.Write(ctx, nil))
	require.NoError(t, Stop())
}

// A log level above info must not switch the audit trail off.
func TestDefaultSinkWritesRecordsAtAnyLogLevel(t *testing.T) {
	for _, level := range []loggerkit.Level{loggerkit.WarnLevel, loggerkit.ErrorLevel, loggerkit.DebugLevel} {
		logger = nil
		loggerInit = sync.Once{}
		var out bytes.Buffer
		restore := UseLogger(loggerkit.New(loggerkit.Config{
			Level: level, Format: loggerkit.FormatJSON, ServiceName: "warden", Output: &out,
		}))

		LogUserQuery(context.Background(), "0123456789abcdef", "13***00", "phone", "203.0.113.7", true, "")
		restore()

		var line map[string]any
		require.NoError(t, json.Unmarshal(out.Bytes(), &line), "level %s: %q", level, out.String())
		assert.Equal(t, true, line["audit"])
		assert.Equal(t, "info", line["level"])
		assert.Equal(t, "warden", line["service"])
	}
}

// Without an override the sink is the application logger, and Stop before
// any use has nothing to stop.
func TestDefaultSinkUsesTheApplicationLogger(t *testing.T) {
	logger = nil
	loggerInit = sync.Once{}
	require.NoError(t, Stop())

	require.Same(t, applog.GetLoggerKit(), sink())
	other := loggerkit.New(loggerkit.Config{Level: loggerkit.InfoLevel, Format: loggerkit.FormatJSON, Output: &bytes.Buffer{}})
	restore := UseLogger(other)
	require.Same(t, other, sink())
	restore()
	require.Same(t, applog.GetLoggerKit(), sink())
}

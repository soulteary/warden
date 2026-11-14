package auditlog

import (
	"context"
	"sync/atomic"

	"github.com/rs/zerolog"
	audit "github.com/soulteary/audit-kit/v2"
	loggerkit "github.com/soulteary/logger-kit/v3"

	"github.com/soulteary/warden/internal/identity"
	applog "github.com/soulteary/warden/internal/logger"
)

// logStorage writes audit records to the structured application log. It is the
// default sink, so lookups and configuration changes leave a record even when
// no dedicated audit storage is configured.
//
// Each record is one Info line carrying "audit": true, which lets a log
// pipeline route audit records separately from ordinary application messages.
// The line is written whatever the level of the application log is: an audit
// trail that LOG_LEVEL=warn or a request to /log/level can switch off does not
// record the request that switched it off.
// The user ID is masked like everywhere else in the application log; the
// identifier in the metadata is already masked by the caller.
type logStorage struct{}

// sinkLogger overrides the logger the default sink writes to. Nil selects the
// application logger.
var sinkLogger atomic.Pointer[loggerkit.Logger]

// UseLogger makes the default sink write to l instead of the application
// logger and returns a function that restores the previous destination.
func UseLogger(l *loggerkit.Logger) (restore func()) {
	previous := sinkLogger.Swap(l)
	return func() { sinkLogger.Store(previous) }
}

func sink() *loggerkit.Logger {
	if l := sinkLogger.Load(); l != nil {
		return l
	}
	return applog.GetLoggerKit()
}

// Write implements audit.Storage.
func (logStorage) Write(_ context.Context, record *audit.Record) error {
	if record == nil {
		return nil
	}
	// A copy of the sink's logger with its own level, so the record does not
	// depend on the level the application log is filtered at.
	log := sink().Zerolog().Level(zerolog.InfoLevel)
	event := log.Info().
		Bool("audit", true).
		Str("event_type", string(record.EventType)).
		Str("result", string(record.Result)).
		Int64("timestamp", record.Timestamp)
	for _, field := range []struct{ key, value string }{
		{"user_id", identity.MaskUserID(record.UserID)},
		{"resource", record.Resource},
		{"reason", record.Reason},
		{"ip", record.IP},
		{"user_agent", record.UserAgent},
		{"request_id", record.RequestID},
		{"trace_id", record.TraceID},
	} {
		if field.value != "" {
			event = event.Str(field.key, field.value)
		}
	}
	if len(record.Metadata) > 0 {
		event = event.Interface("metadata", maskedMetadata(record.Metadata))
	}
	event.Msg("audit")
	return nil
}

// maskedMetadata returns the metadata as it is written to the log. A query by
// user ID carries that ID as its identifier; a derived user ID is an unsalted
// digest of the phone or mail, so it is masked here like the record's user ID.
// Storages passed to Init receive the record unchanged.
func maskedMetadata(metadata map[string]interface{}) map[string]interface{} {
	identifier, ok := metadata["identifier"].(string)
	if !ok || metadata["identifier_type"] != "user_id" {
		return metadata
	}
	masked := make(map[string]interface{}, len(metadata))
	for key, value := range metadata {
		masked[key] = value
	}
	masked["identifier"] = identity.MaskUserID(identifier)
	return masked
}

// Query implements audit.Storage. The application log cannot be queried from
// inside the process.
func (logStorage) Query(context.Context, *audit.QueryFilter) ([]*audit.Record, error) {
	return []*audit.Record{}, nil
}

// Close implements audit.Storage.
func (logStorage) Close() error { return nil }

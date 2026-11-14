package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	loggerkit "github.com/soulteary/logger-kit/v3"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/auditlog"
)

func TestAuditLogLevelRecordsChangeRequests(t *testing.T) {
	var out bytes.Buffer
	t.Cleanup(auditlog.UseLogger(loggerkit.New(loggerkit.Config{Level: loggerkit.InfoLevel, Format: loggerkit.FormatJSON, Output: &out})))
	controlled := loggerkit.New(loggerkit.Config{Level: loggerkit.InfoLevel, Format: loggerkit.FormatJSON, Output: &bytes.Buffer{}})
	handler := AuditLogLevel(controlled, loggerkit.LevelHandler(loggerkit.LevelHandlerConfig{Logger: controlled}))

	serve := func(method, body string) int {
		req := httptest.NewRequest(method, "/log/level", strings.NewReader(body))
		req.RemoteAddr = "198.51.100.7:4711"
		req = WithPrincipal(req, "hmac:gateway")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}
	require.Equal(t, http.StatusOK, serve(http.MethodGet, ""))
	require.Empty(t, out.String(), "reading the level is not recorded")

	require.Equal(t, http.StatusOK, serve(http.MethodPut, `{"level":"debug"}`))
	require.NotEqual(t, http.StatusOK, serve(http.MethodPost, `{"level":"nonsense"}`))

	var records []map[string]any
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		var record map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &record))
		records = append(records, record)
	}
	require.Len(t, records, 2)
	require.Equal(t, "success", records[0]["result"])
	require.Equal(t, "198.51.100.7", records[0]["ip"])
	metadata, ok := records[0]["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "info", metadata["old_value"])
	require.Equal(t, "debug", metadata["new_value"])
	require.Equal(t, "hmac:gateway", metadata["principal"])
	require.Equal(t, "failure", records[1]["result"])
}

// The audit sink is normally the logger whose level is being changed. Raising
// that level above info must not drop the record of the request that raised
// it, nor the records that follow.
func TestAuditLogLevelRecordsARaiseAboveInfo(t *testing.T) {
	var out bytes.Buffer
	shared := loggerkit.New(loggerkit.Config{Level: loggerkit.InfoLevel, Format: loggerkit.FormatJSON, Output: &out})
	t.Cleanup(auditlog.UseLogger(shared))
	handler := AuditLogLevel(shared, loggerkit.LevelHandler(loggerkit.LevelHandlerConfig{Logger: shared}))

	post := func(body string) int {
		req := httptest.NewRequest(http.MethodPost, "/log/level", strings.NewReader(body))
		req.RemoteAddr = "198.51.100.7:4711"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, WithPrincipal(req, "api-key"))
		return rec.Code
	}
	require.Equal(t, http.StatusOK, post(`{"level":"error"}`))
	require.Equal(t, loggerkit.ErrorLevel, shared.GetLevel())
	require.NotEqual(t, http.StatusOK, post(`{"level":"nonsense"}`))

	var records []map[string]any
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		var record map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &record))
		if record["audit"] == true {
			records = append(records, record)
		}
	}
	require.Len(t, records, 2, "the change to error and the refused request after it")
	require.Equal(t, "info", records[0]["level"], "the shape of the line is unchanged")
	metadata, ok := records[0]["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "info", metadata["old_value"])
	require.Equal(t, "error", metadata["new_value"])
	require.Equal(t, "failure", records[1]["result"])
}

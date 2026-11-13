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

package main

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
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/logger"
)

// captureAuditTrail directs the default audit sink to a buffer and returns a
// function that decodes the audit records written so far.
func captureAuditTrail(t *testing.T) func() []map[string]any {
	t.Helper()
	var out bytes.Buffer
	restore := auditlog.UseLogger(loggerkit.New(loggerkit.Config{
		Level: loggerkit.InfoLevel, Format: loggerkit.FormatJSON, ServiceName: "warden", Output: &out,
	}))
	t.Cleanup(restore)
	return func() []map[string]any {
		var records []map[string]any
		scanner := bufio.NewScanner(bytes.NewReader(out.Bytes()))
		for scanner.Scan() {
			var line map[string]any
			require.NoError(t, json.Unmarshal(scanner.Bytes(), &line), scanner.Text())
			if line["audit"] == true {
				records = append(records, line)
			}
		}
		return records
	}
}

// A user query must leave an audit record in the application log, carrying the
// client address resolved through the trusted proxy and no raw identifier.
func TestUserQueryIsWrittenToTheAuditTrail(t *testing.T) {
	app := rateLimitedApp(t, 100, 100)
	t.Setenv("TRUSTED_PROXY_IPS", "10.0.0.1")
	records := captureAuditTrail(t)
	mux := registerRoutes(app)

	for _, path := range []string{"/user?mail=member@example.com", "/v1/lookup?identifier=member@example.com", "/user?mail=absent@example.com"} {
		req := apiKeyRequest(path, "key")
		req.Header.Set("X-Forwarded-For", "203.0.113.7")
		serve(mux, req, "10.0.0.1:4711")
	}

	got := records()
	require.Len(t, got, 3, "one audit record per query")
	for i, record := range got {
		require.Equal(t, "user_query", record["resource"])
		require.Equal(t, "203.0.113.7", record["ip"], "the address behind the trusted proxy, not the proxy")
		require.NotEmpty(t, record["request_id"])
		encoded, err := json.Marshal(record)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), "member@example.com")
		require.NotContains(t, string(encoded), "absent@example.com")
		wantResult := "success"
		if i == 2 {
			wantResult = "failure"
			require.Equal(t, "user_not_found", record["reason"])
		}
		require.Equal(t, wantResult, record["result"])
	}
}

// Without a trusted proxy the forwarded header is the caller's own claim and
// must not reach the audit record.
func TestAuditTrailIgnoresForwardedHeadersFromUntrustedPeers(t *testing.T) {
	app := rateLimitedApp(t, 100, 100)
	t.Setenv("TRUSTED_PROXY_IPS", "none")
	records := captureAuditTrail(t)
	mux := registerRoutes(app)

	req := apiKeyRequest("/user?user_id=member", "key")
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	serve(mux, req, "10.0.0.9:4711")

	got := records()
	require.Len(t, got, 1)
	require.Equal(t, "10.0.0.9", got[0]["ip"])
}

// Every attempt to change the log level is recorded with the caller, the
// address and the levels before and after.
func TestLogLevelChangeIsWrittenToTheAuditTrail(t *testing.T) {
	app := rateLimitedApp(t, 100, 100)
	records := captureAuditTrail(t)
	mux := registerRoutes(app)
	lk := logger.GetLoggerKit()
	previous := lk.GetLevel()
	t.Cleanup(func() { lk.SetLevel(previous) })
	lk.SetLevel(loggerkit.InfoLevel)

	post := func(body string) int {
		req := httptest.NewRequest(http.MethodPost, define.PATH_LOG_LEVEL, strings.NewReader(body))
		req.Header.Set("X-API-Key", "key")
		req.Header.Set("User-Agent", "ops-cli")
		return serve(mux, req, "198.51.100.40:1234").Code
	}
	require.Equal(t, http.StatusOK, post(`{"level":"warn"}`))
	require.NotEqual(t, http.StatusOK, post(`{"level":"no-such-level"}`))
	// Reading the level is not a change.
	require.Equal(t, http.StatusOK, serve(mux, apiKeyRequest(define.PATH_LOG_LEVEL, "key"), "198.51.100.40:1234").Code)

	got := records()
	require.Len(t, got, 2)
	require.Equal(t, "config:log_level", got[0]["resource"])
	require.Equal(t, "success", got[0]["result"])
	require.Equal(t, "198.51.100.40", got[0]["ip"])
	require.Equal(t, "ops-cli", got[0]["user_agent"])
	metadata, ok := got[0]["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "info", metadata["old_value"])
	require.Equal(t, "warn", metadata["new_value"])
	require.Equal(t, "api-key", metadata["principal"])

	require.Equal(t, "failure", got[1]["result"])
	rejected, ok := got[1]["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "warn", rejected["old_value"])
	require.Equal(t, "warn", rejected["new_value"])
}

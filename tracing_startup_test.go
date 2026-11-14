package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	tracing "github.com/soulteary/tracing-kit/v2"
	"github.com/soulteary/tracing-kit/v2/tracingtest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/logger"
)

// writeTracingConfig writes a configuration file with the given tracing block.
func writeTracingConfig(t *testing.T, tracingBlock string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("tracing:\n"+tracingBlock), 0o600))
	return path
}

// Tracing is started from the configuration file when it enables tracing with
// an endpoint, otherwise from OTLP_ENABLED and OTLP_ENDPOINT, and not at all
// when neither names an endpoint.
func TestInitializeTracingSelectsItsConfiguration(t *testing.T) {
	// The collector address is never contacted: no span is exported here.
	const endpoint = "http://127.0.0.1:4318"
	for _, tc := range []struct {
		name        string
		configFile  func(t *testing.T) string
		otlpEnabled string
		otlpEnv     string
		started     bool
	}{
		{name: "nothing configured"},
		{name: "environment enables tracing", otlpEnabled: "true", otlpEnv: endpoint, started: true},
		{name: "environment enables tracing without an endpoint", otlpEnabled: "true"},
		{name: "environment endpoint without the switch", otlpEnv: endpoint},
		{
			name: "configuration file enables tracing",
			configFile: func(t *testing.T) string {
				return writeTracingConfig(t, "  enabled: true\n  endpoint: \""+endpoint+"\"\n")
			},
			started: true,
		},
		{
			name:        "configuration file without an endpoint falls back to the environment",
			configFile:  func(t *testing.T) string { return writeTracingConfig(t, "  enabled: true\n") },
			otlpEnabled: "true", otlpEnv: endpoint,
			started: true,
		},
		{
			name: "configuration file disables tracing and the environment is silent",
			configFile: func(t *testing.T) string {
				return writeTracingConfig(t, "  enabled: false\n  endpoint: \""+endpoint+"\"\n")
			},
		},
		{
			name:        "unreadable configuration file falls back to the environment",
			configFile:  func(t *testing.T) string { return filepath.Join(t.TempDir(), "absent.yaml") },
			otlpEnabled: "true", otlpEnv: endpoint,
			started: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OTLP_ENABLED", tc.otlpEnabled)
			t.Setenv("OTLP_ENDPOINT", tc.otlpEnv)
			t.Cleanup(tracing.Uninstall)
			cfg := &cmd.Config{}
			if tc.configFile != nil {
				cfg.ConfigFile = tc.configFile(t)
			}

			provider := initializeTracing(cfg, logger.GetLoggerKit())

			require.Equal(t, tc.started, provider != nil)
			require.Equal(t, tc.started, tracing.IsEnabled())
			if provider != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				require.NoError(t, provider.Shutdown(ctx))
			}
		})
	}
}

// A tracer that cannot be built is reported and leaves the service untraced
// instead of stopping startup.
func TestInitializeTracingSurvivesAFailedSetup(t *testing.T) {
	t.Cleanup(tracing.Uninstall)
	// The resource attributes are read while the tracer is built; a pair
	// without "=" makes that fail.
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "not-a-key-value-pair")
	const endpoint = "http://127.0.0.1:4318"
	for _, viaFile := range []bool{false, true} {
		cfg := &cmd.Config{}
		if viaFile {
			t.Setenv("OTLP_ENABLED", "")
			cfg.ConfigFile = writeTracingConfig(t, "  enabled: true\n  endpoint: \""+endpoint+"\"\n")
		} else {
			t.Setenv("OTLP_ENABLED", "true")
			t.Setenv("OTLP_ENDPOINT", endpoint)
		}
		require.Nil(t, initializeTracing(cfg, logger.GetLoggerKit()), "via file: %v", viaFile)
		require.False(t, tracing.IsEnabled())
	}
}

// With tracing installed the routes record one server span per request, named
// after the path and carrying the response status.
func TestRoutesRecordSpansWhenTracingIsEnabled(t *testing.T) {
	_, exporter := tracingtest.Setup(t)

	mux := registerRoutes(responseMetadataApp(t))
	for _, tc := range []struct {
		path, apiKey string
		status       int
	}{
		{"/user?user_id=member", "key", http.StatusOK},
		{"/user?user_id=member", "", http.StatusUnauthorized},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, http.NoBody)
		if tc.apiKey != "" {
			req.Header.Set("X-API-Key", tc.apiKey)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, tc.status, rec.Code)
	}

	recorded := exporter.GetSpans()
	served := recorded[:0:0]
	for _, span := range recorded {
		if span.SpanKind == trace.SpanKindServer {
			served = append(served, span)
		}
	}
	require.Len(t, served, 2)
	for i, status := range []int{http.StatusOK, http.StatusUnauthorized} {
		require.Equal(t, "/user", served[i].Name)
		var code int64
		for _, attr := range served[i].Attributes {
			if attr.Key == "http.status_code" {
				code = attr.Value.AsInt64()
			}
		}
		require.Equal(t, int64(status), code)
	}
	recordedBefore := len(recorded)

	// Without tracing no span is recorded.
	tracing.Uninstall()
	mux = registerRoutes(responseMetadataApp(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/user?user_id=member", http.NoBody)
	req.Header.Set("X-API-Key", "key")
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, exporter.GetSpans(), recordedBefore)
}

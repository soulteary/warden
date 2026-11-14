package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	loggerkit "github.com/soulteary/logger-kit/v3"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/loader"
	"github.com/soulteary/warden/internal/prommetrics"
)

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	metric := &dto.Metric{}
	require.NoError(t, c.Write(metric))
	return metric.GetCounter().GetValue()
}

// logLines decodes the JSON log lines written to out.
func logLines(t *testing.T, out *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, raw := range bytes.Split(bytes.TrimSpace(out.Bytes()), []byte("\n")) {
		if len(raw) == 0 {
			continue
		}
		var line map[string]any
		require.NoError(t, json.Unmarshal(raw, &line), "%s", raw)
		lines = append(lines, line)
	}
	return lines
}

// A panic inside a refresh is contained: it is logged and counted, the
// refresh lock is released, and the next refresh runs.
func TestBackgroundTaskContainsAPanic(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "data.json")
	require.NoError(t, os.WriteFile(dataFile, []byte(`[{"phone":"13800138000","user_id":"member","status":"active"}]`), 0o600))
	rules, err := loader.NewRulesLoader(&cmd.Config{Mode: "ONLY_LOCAL"}, "ONLY_LOCAL")
	require.NoError(t, err)

	var out bytes.Buffer
	app := &App{
		log:         loggerkit.New(loggerkit.Config{Level: loggerkit.InfoLevel, Format: loggerkit.FormatJSON, Output: &out}),
		appMode:     "ONLY_LOCAL",
		rulesLoader: rules,
		snapshots:   newSnapshotStore(),
		// No cache: applying the loaded rules dereferences it.
	}
	errorsBefore := counterValue(t, prommetrics.BackgroundTaskErrors)

	require.NotPanics(t, func() { app.backgroundTask(dataFile, "") })

	require.InDelta(t, 1, counterValue(t, prommetrics.BackgroundTaskErrors)-errorsBefore, 0)
	var panicLogged bool
	for _, line := range logLines(t, &out) {
		if line["level"] == "error" && line["panic"] != nil {
			panicLogged = true
		}
	}
	require.True(t, panicLogged, "%s", out.String())
	require.True(t, app.refreshMu.TryLock(), "the refresh lock must be released after a panic")
	app.refreshMu.Unlock()

	// With the cache in place the same task now applies the rules.
	app.userCache = cache.NewSafeUserCache()
	app.backgroundTask(dataFile, "")
	require.Equal(t, 1, app.userCache.Len())
	require.InDelta(t, 1, counterValue(t, prommetrics.BackgroundTaskErrors)-errorsBefore, 0)
}

// The startup identity report names the migration risks in the loaded data
// without rejecting it.
func TestIdentityDiagnosticsReportRisks(t *testing.T) {
	report := func(t *testing.T, users []define.AllowListUser) []map[string]any {
		t.Helper()
		var out bytes.Buffer
		app := &App{
			log:       loggerkit.New(loggerkit.Config{Level: loggerkit.InfoLevel, Format: loggerkit.FormatJSON, Output: &out}),
			userCache: cache.NewSafeUserCache(),
		}
		app.userCache.Set(users)
		app.reportIdentityDiagnostics()
		require.Len(t, app.userCache.Get(), len(users), "the report must not change the cache")
		return logLines(t, &out)
	}
	ok := i18n.TWithLang(i18n.LangZH, "log.identity_diagnostic_ok")
	risk := i18n.TWithLang(i18n.LangZH, "log.identity_diagnostic_risk")

	t.Run("nothing loaded", func(t *testing.T) {
		require.Empty(t, report(t, nil))
	})

	t.Run("clean set", func(t *testing.T) {
		lines := report(t, []define.AllowListUser{{Phone: "13800138000", UserID: "first"}, {Mail: "second@example.com", UserID: "second"}})
		require.Len(t, lines, 1)
		require.Equal(t, ok, lines[0]["message"])
		require.EqualValues(t, 0, lines[0]["conflicts"])
		require.EqualValues(t, 0, lines[0]["missing_user_id"])
	})

	t.Run("user_id equal to another user's phone", func(t *testing.T) {
		lines := report(t, []define.AllowListUser{{Phone: "13800138000", UserID: "first"}, {Mail: "second@example.com", UserID: "13800138000"}})
		require.Len(t, lines, 2)
		require.Equal(t, "warn", lines[0]["level"])
		require.EqualValues(t, 1, lines[0]["records"])
		require.Contains(t, lines[0]["message"], "/v1/lookup")
		// Per-field uniqueness holds, so the summary line still reports no conflict.
		require.Equal(t, ok, lines[1]["message"])
	})

	t.Run("records without user_id when an explicit one is required", func(t *testing.T) {
		define.SetRequireExplicitUserID(true)
		t.Cleanup(func() { define.SetRequireExplicitUserID(false) })
		lines := report(t, []define.AllowListUser{{Phone: "13800138000"}, {Mail: "second@example.com", UserID: "second"}})
		require.Len(t, lines, 1)
		require.Equal(t, risk, lines[0]["message"])
		require.EqualValues(t, 1, lines[0]["missing_user_id"])
		require.Equal(t, true, lines[0]["require_explicit_user_id"])
	})
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	loggerkit "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/warden/internal/loader"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceAttemptLoggingIsDiagnosticAndRedacted(t *testing.T) {
	file := filepath.Join(t.TempDir(), "private-rules.json")
	require.NoError(t, os.WriteFile(file, []byte(`[{"mail":"private@example.com","user_id":"private-member"}]`), 0o600))
	rules, err := loader.NewRulesLoader(nil, loader.ModeOnlyLocal)
	require.NoError(t, err)
	result := rules.LoadWithResult(context.Background(), file, "", "", "")
	require.NoError(t, result.Err)
	result.Err = errors.New("private-error-with-auth")
	for _, level := range []loggerkit.Level{loggerkit.InfoLevel, loggerkit.DebugLevel} {
		var buf bytes.Buffer
		app := &App{log: loggerkit.New(loggerkit.Config{Level: level, Format: loggerkit.FormatJSON, Output: &buf})}
		app.recordSourceAttempts(&result)
		if level == loggerkit.InfoLevel {
			require.Empty(t, buf.String())
			continue
		}
		var event map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &event))
		require.Equal(t, "source attempt completed", event["message"])
		require.Equal(t, "local_1", event["source_id"])
		require.Equal(t, "file", event["transport"])
		require.Equal(t, true, event["selected_input"])
		require.Equal(t, false, event["load_succeeded"])
		require.EqualValues(t, 1, event["winning_rows"])
		for _, secret := range []string{file, "private@example.com", "private-member", "private-error-with-auth"} {
			require.NotContains(t, buf.String(), secret)
		}
	}
}

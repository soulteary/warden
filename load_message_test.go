package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	loggerkit "github.com/soulteary/logger-kit/v3"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/i18n"
	"github.com/soulteary/warden/internal/loader"
)

// The first load reports where the data came from in the source field. Its
// message must not name a source: it said "loaded from remote API" for a
// service that has no remote source and read a local file.
func TestInitialLoadMessageDoesNotNameASource(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "data.json")
	require.NoError(t, os.WriteFile(dataFile, []byte(`[{"phone":"13800138000","mail":"admin@example.com","status":"active"}]`), 0o600))
	rules, err := loader.NewRulesLoader(&cmd.Config{Mode: "DEFAULT"}, "DEFAULT")
	require.NoError(t, err)

	var out bytes.Buffer
	app := &App{
		log:         loggerkit.New(loggerkit.Config{Level: loggerkit.InfoLevel, Format: loggerkit.FormatJSON, Output: &out}),
		appMode:     "DEFAULT",
		rulesLoader: rules,
		userCache:   cache.NewSafeUserCache(),
		snapshots:   newSnapshotStore(),
	}
	require.NoError(t, app.loadInitialData(dataFile, ""))
	require.Equal(t, 1, app.userCache.Len())

	var loaded map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(out.Bytes()), []byte("\n")) {
		var entry map[string]any
		require.NoError(t, json.Unmarshal(line, &entry), string(line))
		if entry["source"] == "local" {
			loaded = entry
		}
	}
	require.NotNil(t, loaded, out.String())
	require.EqualValues(t, 1, loaded["count"])
	require.Equal(t, i18n.TWithLang(i18n.LangZH, "log.loaded_from_sources"), loaded["message"])
	require.NotContains(t, loaded["message"], "远程")

	for _, lang := range []i18n.Language{i18n.LangEN, i18n.LangZH, i18n.LangFR, i18n.LangIT, i18n.LangJA, i18n.LangDE, i18n.LangKO} {
		message := i18n.TWithLang(lang, "log.loaded_from_sources")
		require.NotEqual(t, "log.loaded_from_sources", message, "%s has no translation", lang)
		require.Equal(t, "log.loaded_from_remote_api", i18n.TWithLang(lang, "log.loaded_from_remote_api"), "the replaced message is gone from %s", lang)
	}
}

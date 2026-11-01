package loader

import (
	"context"
	"encoding/json"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestSourceAttemptsCaptureActualFailuresAndSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret-file.json")
	require.NoError(t, os.WriteFile(path, []byte(`[{"mail":"member@example.com","user_id":"member"}]`), 0o600))
	for _, mode := range []string{"DEFAULT", ModeRemoteFirst, ModeRemoteFirstAllowRemoteFail} {
		for _, decrypt := range []bool{false, true} {
			t.Run(mode+map[bool]string{true: "/decrypt", false: "/plain"}[decrypt], func(t *testing.T) {
				var calls atomic.Int64
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					_, err := w.Write([]byte(`invalid JSON secret-payload`))
					require.NoError(t, err)
				}))
				defer server.Close()
				cfg := &cmd.Config{HTTPTimeout: 1}
				if decrypt {
					cfg.RemoteDecryptEnabled = true
					cfg.RemoteRSAPrivateKey = mustGenKeyLoader(t)
					cfg.RemoteEncryptionRequired = true
				}
				loader, err := NewRulesLoader(cfg, mode)
				require.NoError(t, err)
				result := loader.LoadWithResult(context.Background(), path, "", server.URL, "secret-auth")
				require.EqualValues(t, 1, calls.Load(), "diagnostics must not add a source read")
				require.NotEmpty(t, result.Attempts)
				first := result.Attempts[0]
				require.Equal(t, SourceRemote, first.Kind)
				require.Equal(t, "failure", first.Outcome)
				require.False(t, first.SelectedInput)
				require.Zero(t, first.WinningRows)
				if mode == ModeRemoteFirst {
					require.Error(t, result.Err)
					require.Len(t, result.Attempts, 1)
				} else {
					require.NoError(t, result.Err)
					require.Len(t, result.Attempts, 2)
					local := result.Attempts[1]
					require.Equal(t, SourceLocal, local.Kind)
					require.Equal(t, "success", local.Outcome)
					require.True(t, local.SelectedInput)
					require.Equal(t, 1, local.WinningRows)
					// Preserve the plaintext parser's partial-source behavior, including its old label.
					if mode == "DEFAULT" && !decrypt {
						require.Equal(t, SourceMerged, result.Source)
						require.False(t, result.Degraded)
					} else {
						require.True(t, result.Degraded)
					}
				}
				for i := range result.Attempts {
					a := &result.Attempts[i]
					require.Equal(t, i+1, a.Attempt)
					require.False(t, a.StartedAt.IsZero())
					require.False(t, a.FinishedAt.Before(a.StartedAt))
				}
				encoded, err := json.Marshal(result.Attempts)
				require.NoError(t, err)
				for _, secret := range []string{path, server.URL, "secret-auth", "secret-payload", "member@example.com"} {
					require.NotContains(t, string(encoded), secret)
				}
			})
		}
	}
}

func TestSourceAttemptsFallbackSkipsUnreadSourcesAndDistinguishesAbsence(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.json")
	second := filepath.Join(dir, "b.json")
	require.NoError(t, os.WriteFile(first, []byte(`[]`), 0o600))
	require.NoError(t, os.WriteFile(second, []byte(`invalid`), 0o600))
	loader, err := NewRulesLoader(nil, ModeOnlyLocal)
	require.NoError(t, err)
	result := loader.LoadWithResult(context.Background(), "", dir, "", "")
	require.NoError(t, result.Err)
	require.Len(t, result.Attempts, 1)
	require.Equal(t, "success", result.Attempts[0].Outcome)
	require.True(t, result.Attempts[0].SelectedInput)
	require.Zero(t, result.Attempts[0].WinningRows)
	missing := loader.LoadWithResult(context.Background(), filepath.Join(t.TempDir(), "absent.json"), "", "", "")
	require.NoError(t, missing.Err)
	require.Len(t, missing.Attempts, 1)
	require.Equal(t, "absent", missing.Attempts[0].Outcome)
	require.True(t, missing.Attempts[0].SelectedInput)
	noSources := loader.LoadWithResult(context.Background(), "", "", "", "")
	require.Error(t, noSources.Err)
	require.Empty(t, noSources.Attempts)
}

func TestSourceAttemptsShadowingAndIgnoredEmptyInput(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`[{"mail":"same@example.com","user_id":"same"}]`)
	for _, name := range []string{"a.json", "b.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), body, 0o600))
	}
	loader, err := NewRulesLoader(nil, "DEFAULT")
	require.NoError(t, err)
	result := loader.LoadWithResult(context.Background(), "", dir, "", "")
	require.NoError(t, result.Err)
	require.Len(t, result.Attempts, 2)
	require.True(t, result.Attempts[0].SelectedInput)
	require.Zero(t, result.Attempts[0].WinningRows)
	require.True(t, result.Attempts[1].SelectedInput)
	require.Equal(t, 1, result.Attempts[1].WinningRows)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, writeErr := w.Write(body)
		require.NoError(t, writeErr)
	}))
	defer server.Close()
	file := filepath.Join(t.TempDir(), "empty.json")
	require.NoError(t, os.WriteFile(file, []byte(`[]`), 0o600))
	remoteFirst, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 1}, ModeRemoteFirst)
	require.NoError(t, err)
	resolved := remoteFirst.LoadWithResult(context.Background(), file, "", server.URL, "")
	require.NoError(t, resolved.Err)
	require.Len(t, resolved.Attempts, 2)
	require.True(t, resolved.Attempts[0].SelectedInput)
	require.False(t, resolved.Attempts[1].SelectedInput)
	require.Equal(t, "success", resolved.Attempts[1].Outcome)
}

func TestSourceAttemptsDoNotChangeLoadedTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.json")
	require.NoError(t, os.WriteFile(path, []byte(`[]`), 0o600))
	loader, err := NewRulesLoader(nil, ModeOnlyLocal)
	require.NoError(t, err)
	before := time.Now()
	result := loader.LoadWithResult(context.Background(), path, "", "", "")
	require.NoError(t, result.Err)
	require.False(t, result.LoadedAt.Before(before))
	require.False(t, result.LoadedAt.After(result.Attempts[0].StartedAt))
}

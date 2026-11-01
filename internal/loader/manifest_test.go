package loader

import (
	"context"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestIdentitySurvivesFallbackDiscoveryChanges(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.json"), []byte("bad"), 0o600))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "0.json"), []byte(`[{"mail":"new@example.com","user_id":"new"}]`), 0o600))
		_, err := w.Write([]byte("bad"))
		require.NoError(t, err)
	}))
	defer server.Close()
	l, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 1, SourceSetID: "tenant-a"}, "DEFAULT")
	require.NoError(t, err)
	result := l.LoadWithResult(context.Background(), "", dir, server.URL, "")
	require.NoError(t, result.Err)
	require.Len(t, result.Attempts, 4)
	require.Len(t, result.Manifests, 2)
	require.Equal(t, result.Attempts[1].SourceOrdinal, result.Attempts[2].SourceOrdinal)
	require.NotEqual(t, result.Attempts[1].SourceID, result.Attempts[2].SourceID)
	require.Equal(t, result.Attempts[1].SourceID, result.Attempts[3].SourceID)
	require.NotEqual(t, result.Manifests[0].Revision, result.Manifests[1].Revision)
	require.Equal(t, "tenant-a", result.SourceSetID)
}
func TestManifestIncludesUnreadSourcesWithoutAttemptingThem(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.json", "b.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(`[]`), 0o600))
	}
	l, err := NewRulesLoader(nil, ModeOnlyLocal)
	require.NoError(t, err)
	result := l.LoadWithResult(context.Background(), "", dir, "", "")
	require.NoError(t, result.Err)
	require.Len(t, result.Attempts, 1)
	require.Len(t, result.Manifests, 1)
	require.Len(t, result.Manifests[0].Sources, 2)
	require.Equal(t, "success", result.Manifests[0].Discovery)
	again := l.LoadWithResult(context.Background(), "", dir, "", "")
	require.Equal(t, result.Manifests[0].Revision, again.Manifests[0].Revision)
	copied := CloneManifests(result.Manifests)
	copied[0].Sources[0].ID = "changed"
	require.NotEqual(t, copied[0].Sources[0].ID, result.Manifests[0].Sources[0].ID)
	bad := l.LoadWithResult(context.Background(), "", filepath.Join(dir, "["), "", "")
	require.Error(t, bad.Err)
	require.Equal(t, "failed", bad.Manifests[0].Discovery)
}
func TestSourceIdentityScopesAndCanonicalPaths(t *testing.T) {
	file := filepath.Join(t.TempDir(), "rules.json")
	require.Equal(t, sourceIdentity("set", SourceLocal, file), sourceIdentity("set", SourceLocal, filepath.Join(filepath.Dir(file), ".", "rules.json")))
	require.NotEqual(t, sourceIdentity("a", SourceRemote, "https://example.com"), sourceIdentity("b", SourceRemote, "https://example.com"))
	require.NotEqual(t, sourceIdentity("a", SourceRemote, "https://example.com/a"), sourceIdentity("a", SourceRemote, "https://example.com/b"))
}

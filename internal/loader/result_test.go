package loader

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
)

// mustGenKeyLoader generates an RSA-2048 private key and returns it as PKCS1 PEM.
func mustGenKeyLoader(t *testing.T) string {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	return string(pem.EncodeToMemory(block))
}

// writeLocal writes a local users file and returns its path.
func writeLocal(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "users.json")
	body := `[{"phone":"13800138000","mail":"a@example.com"}]`
	require.NoError(t, os.WriteFile(p, []byte(body), 0o600))
	return p
}

// unreachableURL returns a URL that immediately fails to connect.
func unreachableURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	u := srv.URL
	srv.Close() // close so connections are refused
	return u
}

func TestAllowsRemoteFailure(t *testing.T) {
	cases := map[string]bool{
		ModeOnlyLocal:                  true,
		ModeOnlyRemote:                 false,
		ModeRemoteFirst:                false,
		ModeRemoteFirstAllowRemoteFail: true,
		ModeLocalFirst:                 true,
		ModeLocalFirstAllowRemoteFail:  true,
	}
	for mode, want := range cases {
		t.Run(mode, func(t *testing.T) {
			assert.Equal(t, want, allowsRemoteFailure(mode))
		})
	}
}

func TestLoadWithResult_LocalSuccess(t *testing.T) {
	dir := t.TempDir()
	path := writeLocal(t, dir)

	r, err := NewRulesLoader(nil, ModeOnlyLocal)
	require.NoError(t, err)

	res := r.LoadWithResult(context.Background(), path, "", "", "")
	require.NoError(t, res.Err)
	require.Len(t, res.Users, 1)
	assert.Equal(t, SourceLocal, res.Source)
	assert.False(t, res.Degraded)
	assert.NotEmpty(t, res.Version)
	assert.False(t, res.LoadedAt.IsZero())
}

func TestLoadWithResult_RemoteFailure_Modes(t *testing.T) {
	dir := t.TempDir()
	path := writeLocal(t, dir)
	badURL := unreachableURL(t)

	cases := []struct {
		mode         string
		wantErr      bool
		wantDegraded bool
	}{
		{ModeLocalFirst, false, false},                 // parser-kit merges local first; remote failure tolerated
		{ModeLocalFirstAllowRemoteFail, false, false},  // local first, remote failure tolerated
		{ModeRemoteFirstAllowRemoteFail, false, false}, // parser-kit falls back to local within merge
		{ModeRemoteFirst, false, false},                // parser-kit merge still yields local; strict handled in decrypt path
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			cfg := &cmd.Config{HTTPTimeout: 1}
			r, err := NewRulesLoader(cfg, tc.mode)
			require.NoError(t, err)
			res := r.LoadWithResult(context.Background(), path, "", badURL, "")
			if tc.wantErr {
				require.Error(t, res.Err)
				return
			}
			// Either merged/local succeeded (parser-kit fallback) or our explicit
			// local fallback kicked in; in all tolerant modes we must have users.
			require.NoError(t, res.Err)
			require.NotEmpty(t, res.Users)
		})
	}
}

func TestLoadWithResult_OnlyRemote_StrictFailure(t *testing.T) {
	badURL := unreachableURL(t)
	cfg := &cmd.Config{HTTPTimeout: 1}
	r, err := NewRulesLoader(cfg, ModeOnlyRemote)
	require.NoError(t, err)

	res := r.LoadWithResult(context.Background(), "", "", badURL, "")
	require.Error(t, res.Err, "ONLY_REMOTE must not silently fall back")
	assert.Equal(t, SourceNone, res.Source)
}

func TestLoadWithResult_DecryptPath_StrictModeSurfacesError(t *testing.T) {
	// Server returns plaintext; decrypt is enabled + required, so it must fail.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(`[{"phone":"1"}]`)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer srv.Close()

	priv := mustGenKeyLoader(t)
	cfg := &cmd.Config{
		HTTPTimeout:              2,
		RemoteDecryptEnabled:     true,
		RemoteRSAPrivateKey:      priv,
		RemoteEncryptionRequired: true,
		RemoteEncryptionFormat:   "v2",
	}
	r, err := NewRulesLoader(cfg, ModeRemoteFirst)
	require.NoError(t, err)

	res := r.LoadWithResult(context.Background(), "", "", srv.URL, "")
	require.Error(t, res.Err, "strict mode must surface decrypt failure")
	assert.Equal(t, SourceNone, res.Source)
}

func TestLoadWithResult_DecryptPath_TolerantFallsBackDegraded(t *testing.T) {
	// Server returns plaintext (decrypt fails) but a valid local file exists.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(`not-an-envelope`)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := writeLocal(t, dir)

	priv := mustGenKeyLoader(t)
	cfg := &cmd.Config{
		HTTPTimeout:              2,
		RemoteDecryptEnabled:     true,
		RemoteRSAPrivateKey:      priv,
		RemoteEncryptionRequired: true,
		RemoteEncryptionFormat:   "v2",
	}
	r, err := NewRulesLoader(cfg, ModeLocalFirstAllowRemoteFail)
	require.NoError(t, err)

	res := r.LoadWithResult(context.Background(), path, "", srv.URL, "")
	require.NoError(t, res.Err)
	assert.Equal(t, SourceLocal, res.Source)
	assert.True(t, res.Degraded)
	assert.Equal(t, "remote_failed", res.DegradedReason)
	require.Len(t, res.Users, 1)
}

func TestLoadWithResult_ConcurrentReads(t *testing.T) {
	dir := t.TempDir()
	path := writeLocal(t, dir)
	r, err := NewRulesLoader(nil, ModeOnlyLocal)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := r.LoadWithResult(context.Background(), path, "", "", "")
			assert.NoError(t, res.Err)
		}()
	}
	wg.Wait()
	_ = define.AllowListUser{}
}

package loader

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/remote"
	"github.com/stretchr/testify/require"
)

func TestTransportFallbackContract(t *testing.T) {
	keyPEM := mustGenKeyLoader(t)
	block, _ := pem.Decode([]byte(keyPEM))
	require.NotNil(t, block)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
	localJSON := `[{"mail":"local@example.com","user_id":"local","status":"active","role":"reader","scope":["read"]}]`
	remoteJSON := `[{"mail":"remote@example.com","user_id":"remote","status":"inactive","role":"admin","scope":["write"]}]`
	modes := []string{define.DEFAULT_MODE, ModeRemoteFirst, ModeOnlyRemote, ModeOnlyLocal, ModeLocalFirst, ModeRemoteFirstAllowRemoteFail, ModeLocalFirstAllowRemoteFail}
	// Outcome columns follow modes above. Degraded is observable independently of
	// which users are returned. "empty" is a successful empty list, not an error.
	for _, tc := range []struct {
		name          string
		localBody     string
		remoteBody    string
		plainWant     []string
		encryptedWant []string
		remoteStatus  int
	}{
		{
			name: "both_empty", localBody: "[]", remoteBody: "[]", remoteStatus: http.StatusOK,
			plainWant:     []string{"empty", "empty", "empty", "empty", "empty", "empty", "empty"},
			encryptedWant: []string{"empty", "empty", "empty", "empty", "empty", "empty", "empty"},
		},
		{
			name: "remote_empty", localBody: localJSON, remoteBody: "[]", remoteStatus: http.StatusOK,
			plainWant:     []string{"local", "local", "empty", "local", "local", "local", "local"},
			encryptedWant: []string{"local", "local", "empty", "local", "local", "local", "local"},
		},
		{
			name: "local_empty", localBody: "[]", remoteBody: remoteJSON, remoteStatus: http.StatusOK,
			plainWant:     []string{"remote", "remote", "remote", "empty", "remote", "remote", "remote"},
			encryptedWant: []string{"remote", "remote", "remote", "empty", "remote", "remote", "remote"},
		},
		{
			name: "remote_http_error", localBody: localJSON, remoteBody: remoteJSON, remoteStatus: http.StatusNotFound,
			plainWant:     []string{"local", "error", "error", "local", "local", "local_degraded", "local"},
			encryptedWant: []string{"local_degraded", "error", "error", "local", "local_degraded", "local_degraded", "local_degraded"},
		},
		{
			name: "remote_invalid_json", localBody: localJSON, remoteBody: "[", remoteStatus: http.StatusOK,
			plainWant:     []string{"local", "error", "error", "local", "local", "local_degraded", "local"},
			encryptedWant: []string{"local_degraded", "error", "error", "local", "local_degraded", "local_degraded", "local_degraded"},
		},
		{
			name: "remote_error_local_empty", localBody: "[]", remoteBody: remoteJSON, remoteStatus: http.StatusNotFound,
			plainWant:     []string{"error", "error", "error", "empty", "error", "error", "error"},
			encryptedWant: []string{"error", "error", "error", "empty", "error", "error", "error"},
		},
		{
			name: "local_invalid_json", localBody: "[", remoteBody: remoteJSON, remoteStatus: http.StatusOK,
			plainWant:     []string{"remote", "remote", "remote", "error", "remote", "remote", "remote"},
			encryptedWant: []string{"remote", "remote", "remote", "error", "remote", "remote", "remote"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Len(t, tc.plainWant, len(modes))
			require.Len(t, tc.encryptedWant, len(modes))
			localFile := filepath.Join(t.TempDir(), "users.json")
			require.NoError(t, os.WriteFile(localFile, []byte(tc.localBody), 0o600))
			encrypted, err := remote.Encrypt([]byte(tc.remoteBody), &key.PublicKey, "fallback-contract")
			require.NoError(t, err)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := []byte(tc.remoteBody)
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/encrypted" {
					body = encrypted
					w.Header().Set("Content-Type", remote.EncryptedContentType)
				}
				w.WriteHeader(tc.remoteStatus)
				if _, err := w.Write(body); err != nil {
					t.Errorf("write remote fixture: %v", err)
				}
			}))
			t.Cleanup(srv.Close)
			for i, mode := range modes {
				t.Run(mode, func(t *testing.T) {
					for _, transport := range []struct {
						name string
						want string
					}{
						{"plain", tc.plainWant[i]},
						{"encrypted", tc.encryptedWant[i]},
					} {
						t.Run(transport.name, func(t *testing.T) {
							cfg := &cmd.Config{HTTPTimeout: 1}
							if transport.name == "encrypted" {
								cfg.RemoteDecryptEnabled = true
								cfg.RemoteRSAPrivateKey = keyPEM
								cfg.RemoteEncryptionRequired = true
								cfg.RemoteEncryptionFormat = "v2"
							}
							loader, err := newTestRulesLoader(cfg, mode)
							require.NoError(t, err)
							url := srv.URL + "/" + transport.name
							// App passes an empty remote URL in ONLY_LOCAL mode.
							if mode == ModeOnlyLocal {
								url = ""
							}
							ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
							defer cancel()
							result := loader.LoadWithResult(ctx, localFile, "", url, "")
							if transport.want == "error" {
								require.Error(t, result.Err)
								require.Empty(t, result.Users)
								require.Empty(t, result.Version)
								return
							}
							require.NoError(t, result.Err)
							degraded := strings.HasSuffix(transport.want, "_degraded")
							require.Equal(t, degraded, result.Degraded)
							if degraded {
								require.Equal(t, "remote_failed", result.DegradedReason)
							} else {
								require.Empty(t, result.DegradedReason)
							}
							want := []define.AllowListUser{}
							switch strings.TrimSuffix(transport.want, "_degraded") {
							case "local":
								want = append(want, define.AllowListUser{Mail: "local@example.com", UserID: "local", Status: "active", Role: "reader", Scope: []string{"read"}})
							case "remote":
								want = append(want, define.AllowListUser{Mail: "remote@example.com", UserID: "remote", Status: "inactive", Role: "admin", Scope: []string{"write"}})
							case "empty":
							default:
								t.Fatalf("unknown outcome %q", transport.want)
							}
							require.Equal(t, want, result.Users)
							require.Equal(t, cache.HashUserList(result.Users), result.Version)
						})
					}
				})
			}
		})
	}
}

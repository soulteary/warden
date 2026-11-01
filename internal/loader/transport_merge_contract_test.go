package loader

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/remote"
	"github.com/stretchr/testify/require"
)

// TestTransportMergeContract records compatibility differences, rather than
// requiring plaintext and encrypted loading to produce the same grants.
func TestTransportMergeContract(t *testing.T) {
	keyPEM := mustGenKeyLoader(t)
	block, _ := pem.Decode([]byte(keyPEM))
	require.NotNil(t, block)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	for _, identifier := range []string{"phone", "mail"} {
		t.Run(identifier, func(t *testing.T) {
			localShared := define.AllowListUser{Mail: "shared@example.com", UserID: "shared", Status: "active", Role: "reader", Scope: []string{"read"}, Name: "Local member", DingtalkUserID: "local-member"}
			remoteShared := define.AllowListUser{Mail: "shared@example.com", UserID: "shared", Status: "inactive", Role: "admin", Scope: []string{"read", "write"}, Name: "Remote member", DingtalkUserID: "remote-member"}
			if identifier == "phone" {
				localShared.Phone = "13800138000"
				remoteShared.Phone = "13800138000"
			}
			localOnly := define.AllowListUser{Mail: "local@example.com", UserID: "local-only", Status: "active", Role: "reader", Scope: []string{}}
			remoteOnly := define.AllowListUser{Mail: "remote@example.com", UserID: "remote-only", Status: "active", Role: "reader", Scope: []string{}}
			localBody, err := json.Marshal([]define.AllowListUser{localShared, localOnly})
			require.NoError(t, err)
			remoteBody, err := json.Marshal([]define.AllowListUser{remoteShared, remoteOnly})
			require.NoError(t, err)
			encryptedBody, err := remote.Encrypt(remoteBody, &key.PublicKey, "merge-contract")
			require.NoError(t, err)
			localFile := filepath.Join(t.TempDir(), "users.json")
			require.NoError(t, os.WriteFile(localFile, localBody, 0o600))
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := remoteBody
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/encrypted" {
					body = encryptedBody
					w.Header().Set("Content-Type", remote.EncryptedContentType)
				}
				if _, err := w.Write(body); err != nil {
					t.Errorf("write remote users: %v", err)
				}
			}))
			t.Cleanup(srv.Close)

			for _, tc := range []struct {
				mode            string
				plainWinner     string
				encryptedWinner string
			}{
				{define.DEFAULT_MODE, "local", "remote"},
				{ModeRemoteFirst, "remote", "remote"},
				{ModeOnlyRemote, "remote", "remote"},
				{ModeOnlyLocal, "local", "local"},
				{ModeLocalFirst, "remote", "local"},
				{ModeRemoteFirstAllowRemoteFail, "remote", "remote"},
				{ModeLocalFirstAllowRemoteFail, "remote", "local"},
			} {
				t.Run(tc.mode, func(t *testing.T) {
					results := make(map[string]map[string]define.AllowListUser)
					for _, transport := range []struct {
						name   string
						winner string
					}{
						{"plain", tc.plainWinner},
						{"encrypted", tc.encryptedWinner},
					} {
						t.Run(transport.name, func(t *testing.T) {
							cfg := &cmd.Config{HTTPTimeout: 1}
							if transport.name == "encrypted" {
								cfg.RemoteDecryptEnabled = true
								cfg.RemoteRSAPrivateKey = keyPEM
								cfg.RemoteEncryptionRequired = true
								cfg.RemoteEncryptionFormat = "v2"
							}
							loader, err := NewRulesLoader(cfg, tc.mode)
							require.NoError(t, err)
							url := srv.URL + "/" + transport.name
							// Match App's ONLY_LOCAL invocation: it supplies no remote URL.
							if tc.mode == ModeOnlyLocal {
								url = ""
							}
							ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
							defer cancel()
							result := loader.LoadWithResult(ctx, localFile, "", url, "")
							require.NoError(t, result.Err)
							require.False(t, result.Degraded)
							require.Equal(t, cache.HashUserList(result.Users), result.Version)
							got := make(map[string]define.AllowListUser)
							for _, user := range result.Users {
								require.NotContains(t, got, user.UserID)
								got[user.UserID] = user
							}
							shared := localShared
							if transport.winner == "remote" {
								shared = remoteShared
							}
							want := map[string]define.AllowListUser{"shared": shared}
							if tc.mode != ModeOnlyRemote {
								want["local-only"] = localOnly
							}
							if tc.mode != ModeOnlyLocal {
								want["remote-only"] = remoteOnly
							}
							require.Equal(t, want, got)
							results[transport.name] = got
						})
					}
					require.Len(t, results, 2)
					if tc.plainWinner == tc.encryptedWinner {
						require.Equal(t, results["plain"], results["encrypted"])
					} else {
						require.NotEqual(t, results["plain"]["shared"], results["encrypted"]["shared"])
					}
				})
			}
		})
	}
}

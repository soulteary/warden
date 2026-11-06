package loader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/stretchr/testify/require"
)

func TestBaselinePlaintextMergeModeContract(t *testing.T) {
	local := filepath.Join(t.TempDir(), "users.json")
	require.NoError(t, os.WriteFile(local, []byte(`[{"phone":"13800138000","mail":"local@example.com","user_id":"shared"},{"phone":"13800138001","mail":"local-only@example.com","user_id":"local-only"}]`), 0o600))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`[{"phone":"13800138000","mail":"remote@example.com","user_id":"shared"},{"phone":"13800138002","mail":"remote-only@example.com","user_id":"remote-only"}]`))
		if err != nil {
			t.Errorf("write remote response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	for _, tc := range []struct {
		mode       string
		winner     string
		localOnly  bool
		remoteOnly bool
	}{
		{"DEFAULT", "local@example.com", true, true},
		{ModeRemoteFirst, "remote@example.com", true, true},
		{ModeOnlyRemote, "remote@example.com", false, true},
		{ModeOnlyLocal, "local@example.com", true, false},
		{ModeLocalFirst, "remote@example.com", true, true},
		{ModeRemoteFirstAllowRemoteFail, "remote@example.com", true, true},
		{ModeLocalFirstAllowRemoteFail, "remote@example.com", true, true},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			loader, err := newTestRulesLoader(&cmd.Config{HTTPTimeout: 1}, tc.mode)
			require.NoError(t, err)
			result := loader.LoadWithResult(context.Background(), local, "", srv.URL, "")
			require.NoError(t, result.Err)
			byID := map[string]string{}
			for _, user := range result.Users {
				byID[user.UserID] = user.Mail
			}
			want := map[string]string{"shared": tc.winner}
			if tc.localOnly {
				want["local-only"] = "local-only@example.com"
			}
			if tc.remoteOnly {
				want["remote-only"] = "remote-only@example.com"
			}
			require.Equal(t, want, byID)
		})
	}
}

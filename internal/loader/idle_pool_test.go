package loader

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/remote"
)

// HTTP_MAX_IDLE_CONNS reaches the client that fetches the remote source.
func TestConfiguredIdlePoolReachesTheRemoteClient(t *testing.T) {
	var got []remote.ClientOptions
	original := newRestrictedClient
	t.Cleanup(func() { newRestrictedClient = original })
	newRestrictedClient = func(opts remote.ClientOptions) (*http.Client, error) {
		got = append(got, opts)
		return &http.Client{Transport: &countingTransport{}}, nil
	}

	r, err := NewRulesLoader(&cmd.Config{HTTPTimeout: 5, HTTPMaxIdleConns: 7}, "ONLY_REMOTE")
	require.NoError(t, err)
	res := r.LoadWithResult(context.Background(), "", "", "https://rules.example.com/data.json", "")
	require.NoError(t, res.Err)
	require.Len(t, got, 1)
	require.Equal(t, 7, got[0].MaxIdleConns)

	require.Zero(t, BuildRemoteOptions(nil).MaxIdleConns, "no configuration selects the default")
}

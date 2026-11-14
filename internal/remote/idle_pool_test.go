package remote

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientIdlePoolFollowsOptions(t *testing.T) {
	t.Setenv("REMOTE_PRIVATE_ORIGINS", "")
	for _, tc := range []struct{ configured, pool, perHost int }{
		{configured: 0, pool: 20, perHost: 2},
		{configured: -3, pool: 20, perHost: 2},
		{configured: 7, pool: 7, perHost: 2},
		{configured: 1, pool: 1, perHost: 1},
		{configured: 100, pool: 100, perHost: 2},
	} {
		client, err := NewHTTPClientWithOptions(ClientOptions{MaxIdleConns: tc.configured})
		require.NoError(t, err)
		transport, ok := client.Transport.(*originTransport)
		require.True(t, ok)
		require.Equal(t, tc.pool, transport.inner.MaxIdleConns, "configured %d", tc.configured)
		require.Equal(t, tc.perHost, transport.inner.MaxIdleConnsPerHost, "configured %d", tc.configured)
	}

	// The two-argument constructor keeps the default pool.
	client, err := NewHTTPClient(0, false)
	require.NoError(t, err)
	transport, ok := client.Transport.(*originTransport)
	require.True(t, ok)
	require.Equal(t, 20, transport.inner.MaxIdleConns)
}

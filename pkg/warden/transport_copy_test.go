package warden

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// A transport handed to NewClient may be shared with other clients; applying
// this client's TLS configuration must not change it.
func TestNewClientDoesNotModifyTheCallersTransport(t *testing.T) {
	shared := &http.Transport{MaxIdleConns: 7}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13, ServerName: "warden.internal"}

	opts := DefaultOptions().WithBaseURL("https://warden.internal").WithTransport(shared)
	opts.TLSConfig = tlsConfig
	client, err := NewClient(opts)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	// Cloning lets the standard library fill in protocol defaults on the
	// source transport; what must not arrive there is this client's settings.
	require.NotSame(t, tlsConfig, shared.TLSClientConfig, "the caller's transport keeps its own TLS settings")
	if shared.TLSClientConfig != nil {
		require.Empty(t, shared.TLSClientConfig.ServerName)
		require.Zero(t, shared.TLSClientConfig.MinVersion)
	}
	require.Equal(t, 7, shared.MaxIdleConns)
}

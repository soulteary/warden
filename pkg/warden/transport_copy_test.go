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

// clientTransport returns the transport the client sends requests through.
func clientTransport(t *testing.T, client *Client) *http.Transport {
	t.Helper()
	transport, ok := client.httpClient.GetHTTPClient().Transport.(*http.Transport)
	require.True(t, ok)
	return transport
}

// A client that is given a TLS configuration and no transport must keep the
// standard defaults. It used to get a zero-value transport: no proxy from the
// environment and no dial, TLS handshake or idle connection timeout.
func TestTLSConfigKeepsTheDefaultTransportSettings(t *testing.T) {
	cert := tls.Certificate{Certificate: [][]byte{{0x30}}}
	for name, opts := range map[string]*Options{
		"WithTLSConfig":     DefaultOptions().WithBaseURL("https://warden.internal").WithTLSConfig(&tls.Config{MinVersion: tls.VersionTLS13, ServerName: "warden.internal"}),
		"WithTLSClientCert": DefaultOptions().WithBaseURL("https://warden.internal").WithTLSClientCert(cert),
	} {
		client, err := NewClient(opts)
		require.NoError(t, err, name)
		t.Cleanup(client.Close)

		transport := clientTransport(t, client)
		require.Same(t, opts.TLSConfig, transport.TLSClientConfig, name)
		require.NotNil(t, transport.Proxy, "%s: HTTP_PROXY and HTTPS_PROXY apply", name)
		require.NotNil(t, transport.DialContext, "%s: connections are dialled with a timeout", name)
		require.Positive(t, transport.TLSHandshakeTimeout, name)
		require.Positive(t, transport.IdleConnTimeout, name)
		require.True(t, transport.ForceAttemptHTTP2, name)
		require.NotSame(t, http.DefaultTransport, transport, "%s: the shared default transport is not modified", name)
	}
	defaults, ok := http.DefaultTransport.(*http.Transport)
	require.True(t, ok)
	// Cloning lets the standard library fill in protocol defaults there; this
	// client's settings must not arrive.
	if defaults.TLSClientConfig != nil {
		require.Empty(t, defaults.TLSClientConfig.ServerName)
		require.Zero(t, defaults.TLSClientConfig.MinVersion)
		require.Empty(t, defaults.TLSClientConfig.Certificates)
	}
}

// An application may replace http.DefaultTransport with a wrapper. The client
// then builds the same defaults itself.
func TestTLSConfigDefaultsSurviveAReplacedDefaultTransport(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, http.ErrSkipAltProtocol })

	client, err := NewClient(DefaultOptions().WithBaseURL("https://warden.internal").WithTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
	require.NoError(t, err)
	t.Cleanup(client.Close)

	transport := clientTransport(t, client)
	require.NotNil(t, transport.Proxy)
	require.NotNil(t, transport.DialContext)
	require.Positive(t, transport.TLSHandshakeTimeout)
	require.Positive(t, transport.IdleConnTimeout)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

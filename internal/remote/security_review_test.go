package remote

import (
	"context"
	"crypto/x509"
	"github.com/stretchr/testify/require"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"
)

func TestRemoteFetchRejectsLoopbackBeforeRequest(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1); w.WriteHeader(http.StatusOK) }))
	defer target.Close()
	_, err := FetchDecryptedWithOptions(context.Background(), &FetchOptions{URL: target.URL, Timeout: time.Second})
	require.ErrorContains(t, err, "prohibited address")
	require.Zero(t, hits.Load())
}
func TestRemoteFetchDoesNotFollowRedirect(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1); w.WriteHeader(http.StatusOK) }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer origin.Close()
	client, err := NewHTTPClient(time.Second, false)
	require.NoError(t, err)
	// Route the permitted public address to the fixture, retaining the real
	// redirect policy and address validation before the dialer's connection.
	transport, ok := client.Transport.(*originTransport)
	require.True(t, ok)
	transport.inner.DialContext = restrictedDial(staticResolver{ips: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}, fixtureDialer{address: origin.Listener.Addr().String()})
	defer client.CloseIdleConnections()
	_, err = FetchDecryptedWithOptions(context.Background(), &FetchOptions{URL: "http://rules.example", HTTPClient: client})
	require.ErrorIs(t, err, errRedirect)
	require.Zero(t, hits.Load())
}

type staticResolver struct {
	err error
	ips []netip.Addr
}

func (r staticResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return r.ips, r.err
}

type fixtureDialer struct{ address string }

func (d fixtureDialer) DialContext(ctx context.Context, network, _ string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, d.address)
}

type recordingDialer struct{ addresses []string }

func (d *recordingDialer) DialContext(_ context.Context, _, addr string) (net.Conn, error) {
	d.addresses = append(d.addresses, addr)
	return nil, context.Canceled
}

func TestRemoteDialValidatesEveryResolvedAddress(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "::1", "::ffff:127.0.0.1", "10.0.0.1", "169.254.169.254", "fe80::1", "0.0.0.0", "100.100.100.200", "168.63.129.16", "64:ff9b::a9fe:a9fe", "2002:7f00:1::"} {
		t.Run(raw, func(t *testing.T) {
			dialer := &recordingDialer{}
			_, err := restrictedDial(staticResolver{ips: []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr(raw)}}, dialer)(context.Background(), "tcp", "rules.example:443")
			require.ErrorContains(t, err, "prohibited address")
			require.Empty(t, dialer.addresses)
		})
	}
	dialer := &recordingDialer{}
	_, err := restrictedDial(staticResolver{ips: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}, dialer)(context.Background(), "tcp", "rules.example:443")
	require.Error(t, err)
	require.Equal(t, []string{"8.8.8.8:443"}, dialer.addresses)
}
func TestRemotePrivateOriginDoesNotPermitLocalOrMetadata(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "::1", "169.254.169.254", "fe80::1", "168.63.129.16"} {
		require.False(t, allowedAddress(netip.MustParseAddr(raw), true), raw)
	}
	require.True(t, allowedAddress(netip.MustParseAddr("10.0.0.1"), true))
	t.Setenv("REMOTE_PRIVATE_ORIGINS", "https://rules.internal:8443")
	require.True(t, PrivateOriginAllowed("https://rules.internal:8443/data.json"))
	require.False(t, PrivateOriginAllowed("http://rules.internal:8443/data.json"))
	require.False(t, PrivateOriginAllowed("https://rules.internal:443/data.json"))
	require.False(t, PrivateOriginAllowed("https://other.internal:8443/data.json"))
	client, err := NewHTTPClient(time.Second, false)
	require.NoError(t, err)
	transport, ok := client.Transport.(*originTransport)
	require.True(t, ok)
	require.Nil(t, transport.inner.Proxy)
}

func TestRemoteClientConstructionErrors(t *testing.T) {
	for _, raw := range []string{"file:///tmp/rules", "https://user:password@rules.example", "https://rules.example/a", "https://*.example", "https://rules.example:0", "https://rules.example:65536", "https://rules.example?x=1", "https://rules.example#fragment", "http://[fe80::1%25eth0]"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("REMOTE_PRIVATE_ORIGINS", raw)
			_, err := NewHTTPClient(time.Second, false)
			require.Error(t, err)
			require.False(t, PrivateOriginAllowed("https://rules.example"))
		})
	}
	for _, raw := range []string{"file:///tmp/rules", "https://user:password@rules.example", "http://[fe80::1%25eth0]"} {
		client, err := NewHTTPClient(time.Second, false)
		require.NoError(t, err)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, raw, http.NoBody)
		require.NoError(t, err)
		_, err = client.Do(req)
		require.Error(t, err)
		client.CloseIdleConnections()
	}
}
func TestLegacyFetchWrappersUseRestrictedDefault(t *testing.T) {
	_, err := FetchDecrypted(context.Background(), "http://127.0.0.1:1", "", false, "", "", time.Second, false)
	require.ErrorContains(t, err, "prohibited address")
	_, err = FetchDecryptedUsers(context.Background(), "http://127.0.0.1:1", "", false, "", "", time.Second, false)
	require.ErrorContains(t, err, "prohibited address")
}
func TestRemoteDialFailureBoundaries(t *testing.T) {
	dialer := &recordingDialer{}
	_, err := restrictedDial(staticResolver{err: context.DeadlineExceeded}, dialer)(context.Background(), "tcp", "rules.example:443")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	_, err = restrictedDial(staticResolver{}, dialer)(context.Background(), "tcp", "rules.example:443")
	require.ErrorContains(t, err, "no addresses")
	_, err = restrictedDial(staticResolver{}, dialer)(context.Background(), "tcp", "no-port")
	require.Error(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = restrictedDial(staticResolver{ips: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}, dialer)(ctx, "tcp", "rules.example:443")
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, allowedAddress(netip.Addr{}, false))
	require.False(t, allowedAddress(netip.MustParseAddr("fe80::1%eth0"), false))
	require.False(t, AddressAllowed(nil, false))
}

func TestRemoteTLSVerifiesOriginalHostname(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "example.com", r.TLS.ServerName)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client, err := NewHTTPClient(time.Second, false)
	require.NoError(t, err)
	transport, ok := client.Transport.(*originTransport)
	require.True(t, ok)
	transport.inner.TLSClientConfig.RootCAs = roots
	transport.inner.DialContext = restrictedDial(staticResolver{ips: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}, fixtureDialer{address: server.Listener.Addr().String()})
	defer client.CloseIdleConnections()
	response, err := client.Get("https://example.com/rules.json")
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	_, err = client.Get("https://invalid.example/rules.json")
	require.Error(t, err, "a checked IP must not replace the TLS hostname")
}

package remote

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The restricted client cancels a context made by WithAddressPolicyStop when
// the policy refuses the address, and only then.
func TestAddressPolicyStopCancelsOnRefusalOnly(t *testing.T) {
	client, err := NewHTTPClient(5*time.Second, false)
	require.NoError(t, err)
	defer client.CloseIdleConnections()
	transport, ok := client.Transport.(*originTransport)
	require.True(t, ok)
	do := func(ctx context.Context) error {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, "http://rules.example/data.json", http.NoBody)
		require.NoError(t, reqErr)
		resp, doErr := client.Do(req)
		if doErr == nil {
			require.NoError(t, resp.Body.Close())
		}
		return doErr
	}

	// A refused address: the context is cancelled with the refusal as cause.
	transport.inner.DialContext = restrictedDial(staticResolver{ips: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}, &recordingDialer{})
	ctx, release := WithAddressPolicyStop(context.Background())
	require.False(t, AddressPolicyStopped(ctx))
	require.ErrorIs(t, do(ctx), ErrAddressPolicy)
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	require.True(t, AddressPolicyStopped(ctx))
	release()
	require.True(t, AddressPolicyStopped(ctx), "release must not replace the recorded cause")

	// A permitted address whose connection fails: the context stays live, so
	// a caller may retry.
	dialErr := errors.New("connection refused by fixture")
	transport.inner.DialContext = restrictedDial(staticResolver{ips: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}, failingDialer{err: dialErr})
	ctx, release = WithAddressPolicyStop(context.Background())
	require.ErrorIs(t, do(ctx), dialErr)
	require.NoError(t, ctx.Err())
	require.False(t, AddressPolicyStopped(ctx))
	release()
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	require.False(t, AddressPolicyStopped(ctx))

	// Without WithAddressPolicyStop the refusal is only returned.
	transport.inner.DialContext = restrictedDial(staticResolver{ips: []netip.Addr{netip.MustParseAddr("127.0.0.1")}}, &recordingDialer{})
	plain, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.ErrorIs(t, do(plain), ErrAddressPolicy)
	require.NoError(t, plain.Err())
	require.False(t, AddressPolicyStopped(plain))
}

type failingDialer struct{ err error }

func (d failingDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return nil, d.err
}

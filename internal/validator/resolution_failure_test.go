package validator

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

type failedResolver struct{}

func (failedResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return nil, errors.New("DNS unavailable")
}
func TestRemoteURLResolutionFailures(t *testing.T) {
	require.ErrorContains(t, validateRemoteURL("https://example.com", failedResolver{}), "failed to resolve")
	require.ErrorContains(t, validateRemoteURL("https://example.com", staticIPResolver{}), "no addresses")
	require.NoError(t, validateRemoteURL("https://example.com:8443", staticIPResolver{addrs: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}}))
	require.Equal(t, "[2001:4860:4860::8888]", resolvedURLHost("2001:4860:4860::8888", ""))
}

package remote

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

type policyOnlyTransport struct{ called bool }

func (t *policyOnlyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	t.called = true
	return nil, errRedirect
}

func TestRequiredHTTPSCannotUsePlaintextOrInsecureTLS(t *testing.T) {
	transport := &policyOnlyTransport{}
	client := &http.Client{Transport: transport}
	_, err := FetchDecryptedWithOptions(context.Background(), &FetchOptions{URL: "http://rules.example/data.json", RequireHTTPS: true, HTTPClient: client})
	require.ErrorContains(t, err, "requires HTTPS")
	require.False(t, transport.called)
	_, err = FetchDecryptedWithOptions(context.Background(), &FetchOptions{URL: "https://rules.example/data.json", RequireHTTPS: true, InsecureTLS: true, HTTPClient: client})
	require.ErrorContains(t, err, "verified TLS")
	require.False(t, transport.called)
	for _, origin := range []string{"http://[invalid", "file:///tmp/rules", "https://user:password@rules.example"} {
		require.Error(t, ValidateSourceURL(origin, false))
	}
	require.NoError(t, ValidateSourceURL("http://rules.example/data.json", false))
	require.NoError(t, ValidateSourceURL("https://rules.example/data.json", true))
}

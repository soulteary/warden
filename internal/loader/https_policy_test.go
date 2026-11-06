package loader

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/stretchr/testify/require"
)

type sourcePolicyTransport func(*http.Request) (*http.Response, error)

func (f sourcePolicyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProductionSourcesRequireHTTPSBeforeFetching(t *testing.T) {
	t.Setenv("REMOTE_PRIVATE_ORIGINS", "http://rules.internal:8080")
	for _, encrypted := range []bool{false, true} {
		for _, origin := range []string{"http://93.184.216.34/rules.json", "http://rules.internal:8080/rules.json"} {
			t.Run(origin+"/"+map[bool]string{false: "plain", true: "encrypted"}[encrypted], func(t *testing.T) {
				cfg := &cmd.Config{Environment: "production", HTTPTimeout: 1, RemoteDecryptEnabled: encrypted, RemoteRSAPrivateKey: "unused-before-transport", RemoteEncryptionFormat: "v2"}
				calls := 0
				client := &http.Client{Transport: sourcePolicyTransport(func(*http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`))}, nil
				})}
				rules, err := NewRulesLoaderWithHTTPClient(cfg, ModeOnlyRemote, client)
				require.NoError(t, err)
				result := rules.LoadWithResult(context.Background(), "", "", origin, "")
				require.ErrorContains(t, result.Err, "requires HTTPS")
				require.Zero(t, calls)
			})
		}
	}
}

func TestProductionSourcePolicyUsesVerifiedTLS(t *testing.T) {
	cfg := &cmd.Config{Environment: "production", HTTPTimeout: 1, HTTPInsecureTLS: true}
	opts := BuildRemoteOptions(cfg)
	require.True(t, opts.RequireHTTPS)
	fetcher := newRemoteFetcher("https://rules.example/data.json", "", opts)
	_, err := fetcher.Fetch(context.Background(), 1024)
	require.ErrorContains(t, err, "verified TLS")
	for _, environment := range []string{"development", "test", ""} {
		require.False(t, BuildRemoteOptions(&cmd.Config{Environment: environment}).RequireHTTPS)
	}
}

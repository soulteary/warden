package warden

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSDKRejectsRedirectBeforeForwardingCredentials(t *testing.T) {
	for _, code := range []int{301, 302, 303, 307, 308} {
		for _, credential := range []string{"api-key", "hmac", "combined"} {
			t.Run(http.StatusText(code)+"/"+credential, func(t *testing.T) {
				var targetRequests, originRequests atomic.Int32
				target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { targetRequests.Add(1); w.WriteHeader(http.StatusOK) }))
				defer target.Close()
				destination := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)
				origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					originRequests.Add(1)
					if credential == "api-key" {
						require.Equal(t, "directory-secret", r.Header.Get("X-API-Key"))
					} else {
						require.NotEmpty(t, r.Header.Get("X-Signature"))
						require.Empty(t, r.Header.Get("X-API-Key"))
						require.Empty(t, r.Header.Get("Authorization"))
					}
					http.Redirect(w, r, destination, code)
				}))
				defer origin.Close()
				retry := DefaultRetryOptions()
				retry.MaxRetries = 2
				opts := DefaultOptions().WithBaseURL(origin.URL).WithRetry(retry)
				if credential == "api-key" {
					opts = opts.WithAPIKey("directory-secret")
				} else {
					opts = opts.WithHMAC("gateway", "signing-secret")
					if credential == "combined" {
						opts = opts.WithAPIKey("directory-secret")
					}
				}
				client, err := NewClient(opts)
				require.NoError(t, err)
				defer client.Close()
				user, err := client.GetUserByIdentifier(context.Background(), "", "user@example.com", "")
				require.ErrorIs(t, err, ErrRedirect)
				require.Nil(t, user)
				require.EqualValues(t, 1, originRequests.Load(), "a policy rejection is not retried")
				require.Zero(t, targetRequests.Load(), "redirect target must never receive credentials")
			})
		}
	}
}

func TestSDKRejectsSameOriginAndTLSRedirects(t *testing.T) {
	var targetRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			http.Redirect(w, r, "/redirected", http.StatusFound)
			return
		}
		targetRequests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithAPIKey("directory-secret"))
	require.NoError(t, err)
	defer client.Close()
	_, err = client.GetUserByIdentifier(context.Background(), "", "user@example.com", "")
	require.ErrorIs(t, err, ErrRedirect)
	require.Zero(t, targetRequests.Load())
	// The SDK policy does not modify a shared/global HTTP client.
	response, err := http.Get(server.URL + "/user")
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.EqualValues(t, 1, targetRequests.Load())

	var targetHandshakes, originRequests atomic.Int32
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { targetRequests.Add(1); w.WriteHeader(http.StatusOK) }))
	target.TLS = &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAnyClientCert, GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) { targetHandshakes.Add(1); return nil, nil }}
	target.StartTLS()
	defer target.Close()
	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originRequests.Add(1)
		require.NotEmpty(t, r.TLS.PeerCertificates)
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	origin.TLS = &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAnyClientCert}
	origin.StartTLS()
	defer origin.Close()
	roots := x509.NewCertPool()
	roots.AddCert(origin.Certificate())
	secureClient, err := NewClient(DefaultOptions().WithBaseURL(origin.URL).WithTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, Certificates: origin.TLS.Certificates}))
	require.NoError(t, err)
	defer secureClient.Close()
	_, err = secureClient.GetUsers(context.Background())
	require.ErrorIs(t, err, ErrRedirect)
	require.EqualValues(t, 1, originRequests.Load())
	require.Zero(t, targetHandshakes.Load(), "redirect must not cause a client-certificate handshake")
}

func TestSDKRedirectRejectionDoesNotReturnOrCacheResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/another")
		w.WriteHeader(302)
		_, err := io.WriteString(w, `[{"user_id":"untrusted","status":"active"}]`)
		require.NoError(t, err)
	}))
	defer server.Close()
	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	require.NoError(t, err)
	defer client.Close()
	users, err := client.GetUsers(context.Background())
	require.True(t, errors.Is(err, ErrRedirect))
	require.Nil(t, users)
	require.Nil(t, client.cache.Get())
}

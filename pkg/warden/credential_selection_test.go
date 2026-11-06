package warden

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSDKSelectsRequestCredentials(t *testing.T) {
	for _, mode := range []string{"api-key", "hmac", "combined"} {
		for _, operation := range []string{"list", "paginated-list", "lookup", "membership"} {
			t.Run(mode+"/"+operation, func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if mode == "api-key" {
						require.Equal(t, "directory-secret", r.Header.Get("X-API-Key"))
						require.Equal(t, "Bearer directory-secret", r.Header.Get("Authorization"))
						require.Empty(t, r.Header.Get("X-Signature"))
					} else {
						require.NotEmpty(t, r.Header.Get("X-Signature"))
						require.Equal(t, "gateway", r.Header.Get("X-Key-Id"))
						require.Empty(t, r.Header.Get("X-API-Key"))
						require.Empty(t, r.Header.Get("Authorization"))
					}
					body := `{"user_id":"user","status":"active"}`
					if operation == "list" {
						body = `[{"user_id":"user","status":"active"}]`
					}
					if operation == "paginated-list" {
						body = `{"data":[],"pagination":{"page":1,"page_size":10,"total":0}}`
					}
					_, err := io.WriteString(w, body)
					require.NoError(t, err)
				}))
				defer server.Close()
				opts := DefaultOptions().WithBaseURL(server.URL)
				if mode != "hmac" {
					opts = opts.WithAPIKey("directory-secret")
				}
				if mode != "api-key" {
					opts = opts.WithHMAC("gateway", "signing-secret")
				}
				client, err := NewClient(opts)
				require.NoError(t, err)
				defer client.Close()
				switch operation {
				case "list":
					_, err = client.GetUsers(context.Background())
				case "paginated-list":
					_, err = client.GetUsersPaginated(context.Background(), 1, 10)
				case "lookup":
					_, err = client.GetUserByIdentifier(context.Background(), "", "user@example.com", "")
				case "membership":
					require.True(t, client.CheckUserInList(context.Background(), "", "user@example.com"))
				}
				require.NoError(t, err)
				require.EqualValues(t, 1, requests.Load())
			})
		}
	}
}

func TestSDKHMACCredentialSelectionPreservesMTLS(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotEmpty(t, r.TLS.PeerCertificates)
		require.NotEmpty(t, r.Header.Get("X-Signature"))
		require.Empty(t, r.Header.Get("X-API-Key"))
		require.Empty(t, r.Header.Get("Authorization"))
		_, err := io.WriteString(w, `[]`)
		require.NoError(t, err)
	}))
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAnyClientCert}
	server.StartTLS()
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	opts := DefaultOptions().WithBaseURL(server.URL).WithAPIKey("directory-secret").WithHMAC("gateway", "signing-secret").WithTLSConfig(&tls.Config{
		MinVersion: tls.VersionTLS12, RootCAs: roots, Certificates: server.TLS.Certificates,
	})
	client, err := NewClient(opts)
	require.NoError(t, err)
	defer client.Close()
	_, err = client.GetUsers(context.Background())
	require.NoError(t, err)
}

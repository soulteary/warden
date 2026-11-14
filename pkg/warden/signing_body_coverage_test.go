package warden

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingSigningReader struct{ err error }

func (r failingSigningReader) Read([]byte) (int, error) { return 0, r.err }
func TestSigningBodyHashPreservesReplayableAndStreamingBodies(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		req, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("signed-payload"))
		require.NoError(t, err)
		if streaming {
			req.GetBody = nil
		}
		sum := sha256.Sum256([]byte("signed-payload"))
		want := hex.EncodeToString(sum[:])
		for i := 0; i < 2; i++ {
			hash, hashErr := bodyHashHex(req)
			require.NoError(t, hashErr)
			require.Equal(t, want, hash)
		}
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Equal(t, "signed-payload", string(body))
		require.NoError(t, req.Body.Close())
		replay, err := req.GetBody()
		require.NoError(t, err)
		copyBody, err := io.ReadAll(replay)
		require.NoError(t, err)
		require.NoError(t, replay.Close())
		require.Equal(t, body, copyBody)
	}
}
func TestSigningBodyFailuresPropagate(t *testing.T) {
	sentinel := errors.New("body unavailable")
	for _, getBody := range []bool{false, true} {
		req := httptest.NewRequest(http.MethodPost, "https://example.com", io.NopCloser(failingSigningReader{sentinel}))
		if getBody {
			req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(failingSigningReader{sentinel}), nil }
		}
		_, err := bodyHashHex(req)
		require.ErrorIs(t, err, sentinel)
	}
	req := httptest.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("body"))
	req.GetBody = func() (io.ReadCloser, error) { return nil, sentinel }
	_, err := bodyHashHex(req)
	require.ErrorIs(t, err, sentinel)
	signer := newHMACSigner("key", "secret", "", nil)
	require.ErrorIs(t, signer.sign(req), sentinel)
}
func TestTLSClientCertificateOptionsPreserveTrustConfiguration(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, err := w.Write([]byte(`[]`)); require.NoError(t, err) }))
	defer server.Close()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	configuration := &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}
	cert := server.TLS.Certificates[0]
	opts := DefaultOptions().WithBaseURL(server.URL).WithTLSConfig(configuration).WithTLSClientCert(cert)
	require.Same(t, roots, opts.TLSConfig.RootCAs)
	require.EqualValues(t, tls.VersionTLS13, opts.TLSConfig.MinVersion)
	require.Len(t, opts.TLSConfig.Certificates, 1)
	client, err := NewClient(opts)
	require.NoError(t, err)
	defer client.Close()
	users, err := client.GetUsers(context.Background())
	require.NoError(t, err)
	require.Empty(t, users)
	defaults := DefaultOptions().WithTLSClientCert(cert)
	require.EqualValues(t, tls.VersionTLS12, defaults.TLSConfig.MinVersion)
}

package middleware

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type hmacReadFailure struct{}

func (hmacReadFailure) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func TestHMACBodyFailureAndPolicyBoundaries(t *testing.T) {
	cfg := HMACConfig{}.normalized()
	require.Equal(t, 60, cfg.TimestampToleranceSec)
	require.Equal(t, 300, (HMACConfig{TimestampToleranceSec: 1000}).normalized().TimestampToleranceSec)
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.Body = nil
	empty, ok := readBodyHash(req)
	require.True(t, ok)
	require.NotEmpty(t, empty)
	req.Body = io.NopCloser(hmacReadFailure{})
	_, ok = readBodyHash(req)
	require.False(t, ok, "an unreadable body must fail verification")
	require.Equal(t, "/", escapedPathAndQuery(&http.Request{URL: &url.URL{}}))
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set(headerVersion, "v99")
	require.False(t, verifyHMAC(HMACConfig{Keys: map[string]string{"key": "secret"}}, req, "invalid", ts, "key"))
}
func TestServiceAuthClientCertificateBypassesAPIKey(t *testing.T) {
	middleware := ServiceAuthChain(HMACConfig{}, func(http.Handler) http.Handler { t.Fatal("API key middleware called"); return nil })
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{}}}
	w := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })).ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)
}

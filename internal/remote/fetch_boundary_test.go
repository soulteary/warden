package remote

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type fetchReadFailure struct{}

func (fetchReadFailure) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func TestRemoteFetchTransportAndParseFailures(t *testing.T) {
	//nolint:staticcheck // The wrapper must propagate request construction errors.
	_, _, err := fetchBody(nil, &FetchOptions{URL: "https://example.com"})
	require.Error(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = fetchBody(ctx, &FetchOptions{URL: "http://127.0.0.1:1"})
	require.Error(t, err)
	_, err = readLimitedBody(fetchReadFailure{}, 10)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	_, err = parseUsers([]byte("invalid"))
	require.Error(t, err)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "token", r.Header.Get("Authorization"))
		_, writeErr := io.WriteString(w, "invalid")
		require.NoError(t, writeErr)
	}))
	defer server.Close()
	_, err = FetchDecryptedUsersWithOptions(context.Background(), &FetchOptions{URL: server.URL, InsecureTLS: true, AuthHeader: "token"})
	require.Error(t, err)
	_, err = FetchDecryptedUsersWithOptions(ctx, &FetchOptions{URL: server.URL})
	require.Error(t, err)
	_, err = loadRSAPrivateKey("", "")
	require.Error(t, err)
}
func TestEnvelopeRejectsInvalidKeyAndNonceLengths(t *testing.T) {
	private := mustGenKey(t, 2048)
	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &private.PublicKey, []byte("short"), []byte(oaepLabel))
	require.NoError(t, err)
	_, err = unwrapKey(&envelopeV2{EncKey: base64.RawURLEncoding.EncodeToString(encrypted)}, private)
	require.ErrorIs(t, err, ErrIntegrityCheckFailed)
	_, err = aeadOpen([]byte("short"), nil, nil, nil)
	require.ErrorIs(t, err, ErrIntegrityCheckFailed)
	_, err = aeadOpen(make([]byte, 32), []byte("short"), nil, nil)
	require.ErrorIs(t, err, ErrIntegrityCheckFailed)
	_, err = decryptHybridLegacy(nil, nil)
	require.Error(t, err)
	encrypted, err = rsa.EncryptOAEP(sha256.New(), rand.Reader, &private.PublicKey, []byte("short"), nil)
	require.NoError(t, err)
	_, err = decryptHybridLegacy([]byte(base64.StdEncoding.EncodeToString(encrypted)), private)
	require.ErrorContains(t, err, "too short")
}

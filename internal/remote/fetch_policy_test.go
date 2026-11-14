package remote

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/define"
)

// serveBody answers every request with body.
func serveBody(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(body); err != nil {
			t.Logf("write fixture response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// The decryption path selects its format, fails on a key it cannot load, and
// never returns the body of a failed decryption.
func TestFetchDecryptFormatSelectionAndKeyFailures(t *testing.T) {
	ctx := context.Background()
	private := mustGenKey(t, 2048)
	plain := []byte(`[{"phone":"13800138000","user_id":"member"}]`)
	envelope, err := Encrypt(plain, &private.PublicKey, "key-1")
	require.NoError(t, err)
	server := serveBody(t, envelope)
	fetch := func(mutate func(*FetchOptions)) ([]byte, error) {
		opts := &FetchOptions{URL: server.URL, DecryptEnabled: true, RSAKeyPEM: privToPEM(t, private), Timeout: 5 * time.Second}
		mutate(opts)
		return fetchTestWithOptions(ctx, opts)
	}

	// No format is the automatic selection: a v2 envelope is decrypted.
	body, err := fetch(func(*FetchOptions) {})
	require.NoError(t, err)
	require.Equal(t, plain, body)

	// A format the code does not know is refused, not treated as automatic.
	body, err = fetch(func(o *FetchOptions) { o.Format = "v9" })
	require.Nil(t, body)
	require.ErrorContains(t, err, `unknown format "v9"`)

	// A key file that cannot be read fails the fetch; the path takes
	// precedence over the inline key.
	body, err = fetch(func(o *FetchOptions) { o.RSAKeyPath = filepath.Join(t.TempDir(), "absent.pem") })
	require.Nil(t, body)
	require.ErrorContains(t, err, "remote decrypt: load key")
	require.ErrorIs(t, err, os.ErrNotExist)

	// The right format with another key: the envelope fails its check and
	// the ciphertext is not returned.
	other := mustGenKey(t, 2048)
	body, err = fetch(func(o *FetchOptions) { o.Format = FormatV2; o.RSAKeyPEM = privToPEM(t, other) })
	require.Nil(t, body)
	require.ErrorIs(t, err, ErrIntegrityCheckFailed)
}

// The users entry point returns the parsed list, and both size limits apply:
// the response before it is read further, the plaintext before it is parsed.
func TestFetchUsersAndSizeLimits(t *testing.T) {
	ctx := context.Background()
	server := serveBody(t, []byte(`[{"phone":"13800138000","user_id":"member"},{"mail":"second@example.com"}]`))
	users, err := FetchDecryptedUsersWithOptions(ctx, &FetchOptions{URL: server.URL, HTTPClient: testHTTPClient(5*time.Second, false)})
	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Equal(t, "member", users[0].UserID)
	require.Equal(t, "second@example.com", users[1].Mail)

	oversized := serveBody(t, bytes.Repeat([]byte(" "), define.MAX_ENVELOPE_JSON_SIZE+1))
	_, err = FetchDecryptedWithOptions(ctx, &FetchOptions{URL: oversized.URL, HTTPClient: testHTTPClient(30*time.Second, false)})
	require.ErrorIs(t, err, ErrResponseTooLarge)

	_, err = parseUsers(bytes.Repeat([]byte(" "), define.MAX_JSON_SIZE+1))
	require.ErrorIs(t, err, ErrPlaintextTooLarge)
}

// REMOTE_PRIVATE_ORIGINS is validated as a whole: one entry that is not an
// exact origin invalidates the list, for startup validation, for the question
// whether a URL is covered, and for building the client.
func TestPrivateOriginsListIsAllOrNothing(t *testing.T) {
	t.Setenv("REMOTE_PRIVATE_ORIGINS", "https://rules.internal:8443, http://10.0.0.5")
	require.NoError(t, ValidatePrivateOrigins())
	require.True(t, PrivateOriginAllowed("http://10.0.0.5/data.json"))
	require.True(t, PrivateOriginAllowed("https://RULES.internal.:8443/data.json"), "host case and a trailing dot do not matter")
	require.False(t, PrivateOriginAllowed("http://10.0.0.5:8080/data.json"), "the port is part of the origin")
	require.False(t, PrivateOriginAllowed("http://10.0.0.5/%zz"), "a URL that does not parse is not covered")

	for _, list := range []string{
		"https://rules.internal:8443, http://[::1",
		"https://rules.internal:8443, https://rules.internal/path",
	} {
		t.Setenv("REMOTE_PRIVATE_ORIGINS", list)
		require.Error(t, ValidatePrivateOrigins(), list)
		require.False(t, PrivateOriginAllowed("https://rules.internal:8443/data.json"), list)
		// A fetch that has to build its own client fails before any request.
		_, err := FetchDecryptedWithOptions(context.Background(), &FetchOptions{URL: "https://rules.internal:8443/data.json", Timeout: time.Second})
		require.ErrorContains(t, err, "REMOTE_PRIVATE_ORIGINS", list)
	}
}

package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// runNonce returns a nonce prefix that no earlier run of the test in this
// process has used. Signed requests without a Redis client are checked against
// the process-wide replay guard in internal/middleware, which remembers a
// key and nonce for longer than one run of this package takes, so a nonce
// fixed in the source is refused as a replay when the tests run again in the
// same process (go test -count=N). A test that replays a nonce on purpose
// calls this once and reuses the value.
func runNonce(t *testing.T) string {
	t.Helper()
	var raw [8]byte
	_, err := rand.Read(raw[:])
	require.NoError(t, err)
	return t.Name() + "-" + hex.EncodeToString(raw[:]) + "-"
}

func policySignedRequest(path, nonce string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	body := sha256.Sum256(nil)
	canonical := "GET\n" + req.URL.RequestURI() + "\ngateway\n" + ts + "\n" + nonce + "\n" + hex.EncodeToString(body[:])
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(canonical))
	req.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Key-Id", "gateway")
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature-Version", "v2")
	return req
}

func TestServicePolicyRoutes(t *testing.T) {
	t.Setenv("WARDEN_SERVICE_POLICIES", `{"gateway":{"actions":["directory:lookup"]}}`)
	app := responseMetadataApp(t)
	app.hmacKeys = map[string]string{"gateway": "secret"}
	mux := registerRoutes(app)
	nonce := runNonce(t)
	for i, tc := range []struct {
		path   string
		status int
	}{
		{"/user?user_id=member", 200}, {"/v1/user?user_id=member", 200}, {"/v1/lookup?identifier=member", 200},
		{"/", 403}, {"/data.json", 403}, {"/v1/users", 403}, {"/log/level", 403},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, policySignedRequest(tc.path, nonce+strconv.Itoa(i)))
		require.Equal(t, tc.status, rec.Code, tc.path)
	}
	for _, raw := range []string{"{", `{"unknown":{"actions":["directory:list"]}}`} {
		t.Setenv("WARDEN_SERVICE_POLICIES", raw)
		mux = registerRoutes(app)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, policySignedRequest("/user?user_id=member", nonce+raw))
		require.Equal(t, 403, rec.Code)
	}
}

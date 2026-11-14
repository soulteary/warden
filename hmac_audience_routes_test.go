package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
)

func audienceSignedRequest(path, version, audience, nonce string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	body := sha256.Sum256(nil)
	canonical := "GET\n" + req.URL.RequestURI() + "\ngateway\n" + ts + "\n" + nonce + "\n" + hex.EncodeToString(body[:])
	if version == "v3" {
		canonical = "v3\n" + audience + "\n" + canonical
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(canonical))
	req.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Key-Id", "gateway")
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature-Version", version)
	return req
}

// The audience settings reach the data routes from the environment.
func TestHMACAudienceSettingsOnRoutes(t *testing.T) {
	for _, tc := range []struct {
		name            string
		audiences       string
		required        string
		v2, v3, v3Other int
	}{
		{name: "default", v2: 200, v3: 401, v3Other: 401},
		{name: "audiences", audiences: "Warden-A.internal:8081, warden-a", v2: 200, v3: 200, v3Other: 401},
		{name: "required", audiences: "warden-a.internal:8081", required: "true", v2: 401, v3: 200, v3Other: 401},
		// Direct construction with settings startup validation rejects: the
		// stricter reading applies.
		{name: "invalid boolean", audiences: "warden-a.internal:8081", required: "sometimes", v2: 401, v3: 200, v3Other: 401},
		{name: "invalid audiences", audiences: "warden a", v2: 200, v3: 401, v3Other: 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("WARDEN_HMAC_AUDIENCES", tc.audiences)
			t.Setenv("WARDEN_HMAC_REQUIRE_AUDIENCE", tc.required)
			app := responseMetadataApp(t)
			app.hmacKeys = map[string]string{"gateway": "secret"}
			app.initHMACAudience()
			mux := registerRoutes(app)
			for i, request := range []struct {
				version, audience string
				status            int
			}{
				{"v2", "", tc.v2},
				{"v3", "warden-a.internal:8081", tc.v3},
				{"v3", "warden-b.internal:8081", tc.v3Other},
			} {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, audienceSignedRequest("/user?user_id=member", request.version, request.audience, tc.name+strconv.Itoa(i)))
				require.Equal(t, request.status, rec.Code, "%s %s", request.version, request.audience)
			}
			// The API key is a separate credential and is not affected.
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/user?user_id=member", http.NoBody)
			req.Header.Set("X-API-Key", "key")
			mux.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

// The values are normalized by the parser startup validation uses.
func TestInitHMACAudienceNormalizesTheSettings(t *testing.T) {
	t.Setenv("WARDEN_HMAC_AUDIENCES", "Warden-A.internal:8081")
	t.Setenv("WARDEN_HMAC_REQUIRE_AUDIENCE", "true")
	app := &App{}
	app.initHMACAudience()
	require.Equal(t, []string{"warden-a.internal:8081"}, app.hmacAudiences)
	require.True(t, app.hmacRequireAudience)
	audiences, err := cmd.ParseHMACAudiences("Warden-A.internal:8081")
	require.NoError(t, err)
	require.Equal(t, app.hmacAudiences, audiences)
}

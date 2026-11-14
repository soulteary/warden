package warden

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/middleware"
)

// audienceServer is a Warden instance as far as authentication goes: the
// server's HMAC middleware with the given audience settings.
func audienceServer(t *testing.T, audiences []string, requireAudience bool, sawVersion *string) *httptest.Server {
	t.Helper()
	cfg := middleware.HMACConfig{
		Keys:            map[string]string{"gateway": "shared-secret"},
		ReplayGuard:     middleware.NewMemoryReplayGuard(0),
		Audiences:       audiences,
		RequireAudience: requireAudience,
	}
	handler := middleware.HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerHMACSignature) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if sawVersion != nil {
			*sawVersion = r.Header.Get(headerHMACVersion)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"phone":"13800138000","status":"active"}]`)) //nolint:errcheck // test server
	}))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func requireUnauthorized(t *testing.T, err error) {
	t.Helper()
	var sdkErr *Error
	require.True(t, errors.As(err, &sdkErr), "%v", err)
	require.Equal(t, ErrCodeUnauthorized, sdkErr.Code)
}

// A client with an audience signs v3, and the server verifies it only where
// the audience is listed.
func TestClientSignsForAnAudience(t *testing.T) {
	var version string
	a := audienceServer(t, []string{"warden-a.internal:8081"}, true, &version)
	b := audienceServer(t, []string{"warden-b.internal:8081"}, true, nil)
	ctx := context.Background()

	// The option value is normalized like the server's list.
	opts := func(url string) *Options {
		return DefaultOptions().WithBaseURL(url).WithHMAC("gateway", "shared-secret").WithHMACAudience(" Warden-A.internal:8081 ")
	}
	toA, err := NewClient(opts(a.URL))
	require.NoError(t, err)
	users, err := toA.GetUsers(ctx)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "v3", version)

	// The same client configuration pointed at instance B is refused.
	toB, err := NewClient(opts(b.URL))
	require.NoError(t, err)
	_, err = toB.GetUsers(ctx)
	requireUnauthorized(t, err)
}

// A client without the option keeps signing v2: accepted by a server with
// default settings, with or without audiences, refused where v3 is required.
func TestClientWithoutAudienceKeepsSigningV2(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name      string
		audiences []string
		require   bool
		accepted  bool
	}{
		{name: "default server", accepted: true},
		{name: "audiences configured, not required", audiences: []string{"warden-a"}, accepted: true},
		{name: "audience required", audiences: []string{"warden-a"}, require: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var version string
			server := audienceServer(t, tc.audiences, tc.require, &version)
			client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithHMAC("gateway", "shared-secret"))
			require.NoError(t, err)
			_, err = client.GetUsers(ctx)
			if !tc.accepted {
				requireUnauthorized(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "v2", version)
		})
	}
}

func TestSignerV3CanonicalForm(t *testing.T) {
	require.Equal(t,
		"v3\nwarden-a\nGET\n/user?phone=1\nkey-1\n1700000000\nnonce\nhash",
		canonicalStringV3("warden-a", "GET", "/user?phone=1", "key-1", "1700000000", "nonce", "hash"))

	bound := newHMACSigner("key-1", "secret", "warden-a", fixedClock{t: time.Unix(1700000000, 0)})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/user?phone=1", http.NoBody)
	require.NoError(t, bound.sign(req))
	require.Equal(t, "v3", req.Header.Get(headerHMACVersion))
	// The audience is signed, not sent.
	for name, values := range req.Header {
		for _, value := range values {
			require.NotContains(t, value, "warden-a", name)
		}
	}
}

func TestHMACAudienceOptionValidation(t *testing.T) {
	base := func() *Options { return DefaultOptions().WithBaseURL("http://warden:8081") }

	opts := base().WithHMAC("gateway", "secret").WithHMACAudience("  Warden.Internal:8081 ")
	require.NoError(t, opts.Validate())
	require.Equal(t, "warden.internal:8081", opts.HMACAudience)

	blank := base().WithHMAC("gateway", "secret").WithHMACAudience("   ")
	require.NoError(t, blank.Validate())
	require.Empty(t, blank.HMACAudience)

	for name, invalid := range map[string]*Options{
		"without HMAC": base().WithHMACAudience("warden"),
		"space":        base().WithHMAC("gateway", "secret").WithHMACAudience("warden a"),
		"comma":        base().WithHMAC("gateway", "secret").WithHMACAudience("warden-a,warden-b"),
		"non-ASCII":    base().WithHMAC("gateway", "secret").WithHMACAudience("wärden"),
		"too long":     base().WithHMAC("gateway", "secret").WithHMACAudience(string(make([]byte, 256))),
	} {
		err := invalid.Validate()
		var sdkErr *Error
		require.True(t, errors.As(err, &sdkErr), name)
		require.Equal(t, ErrCodeInvalidConfig, sdkErr.Code, name)
	}
}

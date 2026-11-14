package middleware

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

	"github.com/soulteary/warden/internal/serviceauth"
)

const (
	v3Key    = "gateway"
	v3Secret = "shared-between-instances"
)

// newV3Request builds a GET request signed in the v3 form for audience.
func newV3Request(target, audience, nonce string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, http.NoBody)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	body := sha256.Sum256(nil)
	canonical := "v3\n" + audience + "\nGET\n" + escapedPathAndQuery(req) + "\n" + v3Key + "\n" + ts + "\n" + nonce + "\n" + hex.EncodeToString(body[:])
	mac := hmac.New(sha256.New, []byte(v3Secret))
	mac.Write([]byte(canonical))
	req.Header.Set(headerSignature, hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set(headerTimestamp, ts)
	req.Header.Set(headerKeyID, v3Key)
	req.Header.Set(headerNonce, nonce)
	req.Header.Set(headerVersion, signatureV3)
	return req
}

// instance is one Warden instance: its own replay guard, as two deployments
// that do not share Redis have, and the shared key.
func instance(audiences ...string) http.Handler {
	cfg := HMACConfig{
		Keys:        map[string]string{v3Key: v3Secret},
		ReplayGuard: NewMemoryReplayGuard(0),
	}
	cfg.Audiences = audiences
	return HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
}

// serve sends a copy of req, as a party that captured it would.
func serve(h http.Handler, req *http.Request) int {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req.Clone(req.Context()))
	return rec.Code
}

// A v2 request captured on its way to one instance is accepted by another
// instance that shares the key: nothing in the signed text names the receiver.
// With the default settings this stays as it is, for existing clients.
func TestHMACUnboundSignatureIsValidAtEveryInstanceByDefault(t *testing.T) {
	a, b := instance(), instance()
	req := newV2Request(http.MethodGet, "/user?phone=13800138000", "", v3Secret, v3Key, time.Now().Unix(), "captured-v2")
	require.Equal(t, http.StatusOK, serve(a, req))
	require.Equal(t, http.StatusOK, serve(b, req), "the default keeps accepting v2")

	// Configuring audiences alone does not reject v2 either.
	c := instance("warden-c")
	require.Equal(t, http.StatusOK, serve(c, newV2Request(http.MethodGet, "/user", "", v3Secret, v3Key, time.Now().Unix(), "v2-with-audiences")))
}

// A v3 request is valid only at an instance that lists the signed audience.
func TestHMACv3BindsTheRequestToItsAudience(t *testing.T) {
	a := instance("warden-a.internal:8081")
	b := instance("warden-b.internal:8081")

	req := newV3Request("/user?phone=13800138000", "warden-a.internal:8081", "captured-v3")
	require.Equal(t, http.StatusOK, serve(a, req))
	require.Equal(t, http.StatusUnauthorized, serve(b, req), "replayed at the instance it was not signed for")

	// The audience is not read from the request: naming instance A in the
	// Host header does not make B accept the request.
	forged := newV3Request("/user?phone=13800138000", "warden-a.internal:8081", "forged-host")
	forged.Host = "warden-a.internal:8081"
	require.Equal(t, http.StatusUnauthorized, serve(b, forged))

	// Nor does the Host header a reverse proxy rewrote stop A from accepting it.
	proxied := newV3Request("/user?phone=13800138000", "warden-a.internal:8081", "proxied")
	proxied.Host = "10.0.0.7:8081"
	require.Equal(t, http.StatusOK, serve(a, proxied))

	// An instance may have several names.
	both := instance("warden-b.internal:8081", "warden-a.internal:8081")
	require.Equal(t, http.StatusOK, serve(both, newV3Request("/user", "warden-a.internal:8081", "second-name")))
}

func TestHMACv3Rejections(t *testing.T) {
	a := instance("warden-a")

	// The nonce is single use, as in v2.
	req := newV3Request("/user", "warden-a", "once")
	require.Equal(t, http.StatusOK, serve(a, req))
	require.Equal(t, http.StatusUnauthorized, serve(a, req))

	// A nonce is required.
	missingNonce := newV3Request("/user", "warden-a", "")
	missingNonce.Header.Del(headerNonce)
	require.Equal(t, http.StatusUnauthorized, serve(a, missingNonce))

	// The audience is signed, not only compared: a v2 signature relabelled as
	// v3 does not verify, and neither does a v3 signature relabelled as v2.
	relabelled := newV2Request(http.MethodGet, "/user", "", v3Secret, v3Key, time.Now().Unix(), "relabel-v2")
	relabelled.Header.Set(headerVersion, signatureV3)
	require.Equal(t, http.StatusUnauthorized, serve(a, relabelled))
	downgraded := newV3Request("/user", "warden-a", "relabel-v3")
	downgraded.Header.Set(headerVersion, signatureV2)
	require.Equal(t, http.StatusUnauthorized, serve(a, downgraded))
	downgraded.Header.Del(headerVersion)
	require.Equal(t, http.StatusUnauthorized, serve(a, downgraded))

	// An instance without a configured audience cannot check the binding and
	// refuses v3, whatever audience the client signed, including none.
	unnamed := instance()
	require.Equal(t, http.StatusUnauthorized, serve(unnamed, newV3Request("/user", "warden-a", "no-audience-configured")))
	require.Equal(t, http.StatusUnauthorized, serve(unnamed, newV3Request("/user", "", "empty-audience")))
}

// RequireAudience refuses every form that is not bound to an instance.
func TestHMACRequireAudienceRejectsUnboundForms(t *testing.T) {
	allowV1 := true
	cfg := HMACConfig{
		Keys:            map[string]string{v3Key: v3Secret},
		ReplayGuard:     NewMemoryReplayGuard(0),
		Audiences:       []string{"warden-a"},
		RequireAudience: true,
		AllowV1:         &allowV1,
	}
	strict := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	now := time.Now().Unix()

	require.Equal(t, http.StatusOK, serve(strict, newV3Request("/user", "warden-a", "bound")))
	require.Equal(t, http.StatusUnauthorized, serve(strict, newV2Request(http.MethodGet, "/user", "", v3Secret, v3Key, now, "unbound-v2")))

	unversioned := newV2Request(http.MethodGet, "/user", "", v3Secret, v3Key, now, "unbound-unversioned")
	unversioned.Header.Del(headerVersion)
	require.Equal(t, http.StatusUnauthorized, serve(strict, unversioned))

	v1 := httptest.NewRequest(http.MethodGet, "/user", http.NoBody)
	v1.Header.Set(headerSignature, computeHMAC(http.MethodGet, "/user", "", v3Secret, now))
	v1.Header.Set(headerTimestamp, strconv.FormatInt(now, 10))
	v1.Header.Set(headerKeyID, v3Key)
	require.Equal(t, http.StatusUnauthorized, serve(strict, v1))

	// The same v1 and v2 requests pass when the requirement is off.
	cfg.RequireAudience = false
	cfg.ReplayGuard = NewMemoryReplayGuard(0)
	lenient := HMACAuth(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	require.Equal(t, http.StatusOK, serve(lenient, v1))
	require.Equal(t, http.StatusOK, serve(lenient, unversioned))
}

// Service policies accept v3 as they accept v2, and still authorize by key.
func TestServicePolicyAcceptsAudienceBoundSignature(t *testing.T) {
	cfg := HMACConfig{
		Keys:            map[string]string{v3Key: v3Secret},
		ReplayGuard:     NewMemoryReplayGuard(0),
		Audiences:       []string{"warden-a"},
		ServicePolicies: map[string]serviceauth.Policy{v3Key: {Actions: []string{"directory:lookup"}}},
	}
	var principal string
	chain := ServiceAuthChain(cfg, func(http.Handler) http.Handler {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("API key fallback must not be reached") })
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal = PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	require.Equal(t, http.StatusOK, serve(chain, newV3Request("/user?user_id=member", "warden-a", "policy-lookup")))
	require.Equal(t, principalHMACPrefix+v3Key, principal)
	require.Equal(t, http.StatusForbidden, serve(chain, newV3Request("/", "warden-a", "policy-list")))
	require.Equal(t, http.StatusUnauthorized, serve(chain, newV3Request("/user?user_id=member", "warden-b", "policy-other-audience")))
}

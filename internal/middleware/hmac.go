// Package middleware provides HMAC signature verification for service-to-service auth.
package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/soulteary/warden/internal/logger"
)

const (
	headerSignature = "X-Signature"
	headerTimestamp = "X-Timestamp"
	headerKeyID     = "X-Key-Id"
	headerNonce     = "X-Nonce"
	headerVersion   = "X-Signature-Version"

	// signatureV1 is the legacy canonical form: METHOD + PATH(+QUERY) + TS + BODY_HASH.
	signatureV1 = "v1"
	// signatureV2 is the current canonical form:
	//   METHOD\nPATH_AND_QUERY\nTIMESTAMP\nNONCE\nSHA256_HEX(body)
	signatureV2 = "v2"

	// defaultMaxTimestampToleranceSec is the hard upper bound on the configurable
	// timestamp tolerance. It bounds the replay window even if an operator sets an
	// unreasonably large tolerance.
	defaultMaxTimestampToleranceSec = 300
)

// ReplayObserver is called when a v2 request is rejected as a replay. It lets the
// caller record a metric without this package importing prommetrics (avoids an
// import cycle and keeps the middleware dependency-light).
type ReplayObserver func()

// HMACConfig holds HMAC verification settings.
//
//nolint:govet // fieldalignment: readability over a few bytes for a config struct
type HMACConfig struct {
	// Keys maps key_id -> secret for verification.
	Keys map[string]string
	// TimestampToleranceSec is the allowed clock skew in seconds (default 60). It is
	// clamped to MaxTimestampToleranceSec.
	TimestampToleranceSec int
	// MaxTimestampToleranceSec is the hard upper bound on the tolerance (default 300).
	MaxTimestampToleranceSec int
	// AllowV1 keeps the legacy v1 canonical form acceptable during migration. Nil
	// defaults to true so existing signers keep working; point to false to require v2.
	AllowV1 *bool
	// ReplayGuard rejects reused v2 nonces within the timestamp window. When nil a
	// process-local in-memory guard is used. See ReplayGuard docs for the multi-replica
	// limitation.
	ReplayGuard ReplayGuard
	// OnReplayRejected, when set, is called each time a v2 request is rejected as a replay.
	OnReplayRejected ReplayObserver
	// OnV1Used, when set, is called when a request is accepted via the legacy v1 form
	// (used to emit a deprecation metric).
	OnV1Used ReplayObserver
}

// normalized returns a copy of cfg with defaults and bounds applied.
func (cfg HMACConfig) normalized() HMACConfig {
	if cfg.Keys == nil {
		cfg.Keys = make(map[string]string)
	}
	if cfg.AllowV1 == nil {
		allowV1 := true
		cfg.AllowV1 = &allowV1
	}
	if cfg.MaxTimestampToleranceSec <= 0 {
		cfg.MaxTimestampToleranceSec = defaultMaxTimestampToleranceSec
	}
	if cfg.TimestampToleranceSec <= 0 {
		cfg.TimestampToleranceSec = 60
	}
	if cfg.TimestampToleranceSec > cfg.MaxTimestampToleranceSec {
		cfg.TimestampToleranceSec = cfg.MaxTimestampToleranceSec
	}
	if cfg.ReplayGuard == nil {
		cfg.ReplayGuard = defaultReplayGuard
	}
	return cfg
}

// defaultReplayGuard is a shared single-node guard used when a config does not
// supply one. It bounds memory to a generous cap.
var defaultReplayGuard = NewMemoryReplayGuard(1 << 20)

// HMACAuth returns a middleware that verifies an HMAC-SHA256 request signature.
// It accepts both the legacy v1 canonical form and the authenticated v2 form
// (with per-request nonce + replay rejection). If Keys is empty or the required
// headers are absent, the middleware passes through without requiring HMAC
// (unchanged historical behavior — the API-Key layer enforces auth).
func HMACAuth(cfg HMACConfig) func(http.Handler) http.Handler {
	c := cfg.normalized()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sig := r.Header.Get(headerSignature)
			tsStr := r.Header.Get(headerTimestamp)
			keyID := r.Header.Get(headerKeyID)
			if sig == "" || tsStr == "" || keyID == "" {
				next.ServeHTTP(w, r)
				return
			}
			if verifyHMAC(c, r, sig, tsStr, keyID) {
				next.ServeHTTP(w, r)
				return
			}
			writeUnauthorized(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// ServiceAuthChain returns a middleware that implements: mTLS (client cert) > HMAC > API Key.
// If the request has a verified TLS client certificate, the inner handler is called directly.
// If HMAC headers are present and valid (v1 or v2), the inner handler is called (API Key
// skipped). Otherwise the request is passed to apiKeyMiddleware.
func ServiceAuthChain(hmacCfg HMACConfig, apiKeyMiddleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	c := hmacCfg.normalized()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				next.ServeHTTP(w, r)
				return
			}
			sig := r.Header.Get(headerSignature)
			tsStr := r.Header.Get(headerTimestamp)
			keyID := r.Header.Get(headerKeyID)
			if sig != "" && tsStr != "" && keyID != "" && len(c.Keys) > 0 {
				if verifyHMAC(c, r, sig, tsStr, keyID) {
					next.ServeHTTP(w, r)
					return
				}
				writeUnauthorized(w, r)
				return
			}
			apiKeyMiddleware(next).ServeHTTP(w, r)
		})
	}
}

// verifyHMAC reads the body, verifies the signature (v2 preferred, v1 accepted when
// allowed) and returns true if valid. It always restores r.Body for downstream
// handlers. Secrets are never logged.
func verifyHMAC(cfg HMACConfig, r *http.Request, sig, tsStr, keyID string) bool {
	cfg = cfg.normalized()
	secret, ok := cfg.Keys[keyID]
	if !ok {
		logger.FromRequest(r).Debug().Str("key_id", keyID).Msg("hmac: unknown key_id")
		return false
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		logger.FromRequest(r).Debug().Err(err).Msg("hmac: invalid timestamp")
		return false
	}
	now := time.Now().Unix()
	tol := int64(cfg.TimestampToleranceSec)
	if ts < now-tol || ts > now+tol {
		logger.FromRequest(r).Debug().Int64("ts", ts).Int64("now", now).Msg("hmac: timestamp out of range")
		return false
	}

	bodyHash := readBodyHash(r)

	version := r.Header.Get(headerVersion)
	nonce := r.Header.Get(headerNonce)

	// Decide which canonical form(s) to try:
	//   - explicit "v2"      => v2 only
	//   - explicit "v1"      => v1 only (when AllowV1)
	//   - no version header  => try v2 when a nonce is present, then v1 (when AllowV1)
	// This maximizes migration compatibility without weakening either form.
	switch version {
	case signatureV2:
		return cfg.verifyV2(r, secret, sig, tsStr, nonce, bodyHash)
	case signatureV1, "":
		if version == "" && nonce != "" {
			if cfg.verifyV2(r, secret, sig, tsStr, nonce, bodyHash) {
				return true
			}
			// Fall through to v1 (when allowed): a legacy signer might coincidentally
			// send a nonce header; do not silently reject during migration.
		}
		if *cfg.AllowV1 && verifyV1(r, secret, sig, ts, bodyHash) {
			if cfg.OnV1Used != nil {
				cfg.OnV1Used()
			}
			return true
		}
		return false
	default:
		logger.FromRequest(r).Debug().Str("version", version).Msg("hmac: unknown signature version")
		return false
	}
}

// verifyV2 checks the v2 canonical form and enforces nonce replay rejection.
func (cfg HMACConfig) verifyV2(r *http.Request, secret, sig, tsStr, nonce, bodyHash string) bool {
	if nonce == "" {
		logger.FromRequest(r).Debug().Msg("hmac: v2 missing nonce")
		return false
	}
	canonical := r.Method + "\n" + escapedPathAndQuery(r) + "\n" + tsStr + "\n" + nonce + "\n" + bodyHash
	if !equalHMAC(secret, canonical, sig) {
		logger.FromRequest(r).Debug().Msg("hmac: v2 signature mismatch")
		return false
	}
	// Signature is valid; now enforce single-use of the nonce within the window. The
	// replay key binds the key_id + nonce so different keys cannot collide.
	replayKey := r.Header.Get(headerKeyID) + ":" + nonce
	ttl := time.Duration(cfg.TimestampToleranceSec)*time.Second*2 + time.Second
	seenBefore, err := cfg.ReplayGuard.SeenBefore(replayKey, ttl)
	if err != nil {
		logger.FromRequest(r).Debug().Err(err).Msg("hmac: replay guard unavailable; request rejected")
		return false
	}
	if seenBefore {
		logger.FromRequest(r).Debug().Msg("hmac: v2 nonce replay rejected")
		if cfg.OnReplayRejected != nil {
			cfg.OnReplayRejected()
		}
		return false
	}
	return true
}

// verifyV1 checks the legacy canonical form: METHOD + PATH(+QUERY) + TS + BODY_HASH.
func verifyV1(r *http.Request, secret, sig string, ts int64, bodyHash string) bool {
	message := r.Method + pathAndQuery(r) + strconv.FormatInt(ts, 10) + bodyHash
	return equalHMAC(secret, message, sig)
}

// equalHMAC computes HMAC-SHA256(secret, message) and constant-time compares it to
// the hex-encoded expected signature.
func equalHMAC(secret, message, sig string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

// readBodyHash reads and restores r.Body, returning the hex SHA-256 of the body.
// For an empty body it returns the hash of the empty string, matching the SDK.
func readBodyHash(r *http.Request) string {
	if r.Body == nil {
		h := sha256.Sum256(nil)
		return hex.EncodeToString(h[:])
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r.Body); err != nil {
		logger.FromRequest(r).Warn().Err(err).Msg("hmac: failed to read body")
		// Restore an empty body and return the empty hash; signature will simply not match.
		r.Body = io.NopCloser(bytes.NewReader(nil))
		h := sha256.Sum256(nil)
		return hex.EncodeToString(h[:])
	}
	data := buf.Bytes()
	r.Body = io.NopCloser(bytes.NewReader(data))
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// pathAndQuery returns the request path (r.URL.Path) plus raw query when present.
// The legacy v1 canonical form uses this exact form for byte-identical compatibility
// with existing signers and the historical middleware.
func pathAndQuery(r *http.Request) string {
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	return path
}

// escapedPathAndQuery returns EscapedPath (defaulting to "/") plus raw query. This
// matches the SDK's v2 signer, which uses url.EscapedPath so reserved characters are
// canonicalized identically on both sides.
func escapedPathAndQuery(r *http.Request) string {
	p := r.URL.EscapedPath()
	if p == "" {
		p = "/"
	}
	if r.URL.RawQuery != "" {
		return p + "?" + r.URL.RawQuery
	}
	return p
}

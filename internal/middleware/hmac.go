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
)

// HMACConfig holds HMAC verification settings.
type HMACConfig struct {
	// Keys maps key_id -> secret for verification
	Keys map[string]string
	// TimestampToleranceSec is the allowed clock skew in seconds (default 60)
	TimestampToleranceSec int
}

// HMACAuth returns a middleware that verifies HMAC-SHA256 request signature.
// Signature = HMAC_SHA256(secret, method + path + timestamp + body_hash).
// If Keys is empty or nil, the middleware passes through without requiring HMAC.
func HMACAuth(cfg HMACConfig) func(http.Handler) http.Handler {
	if cfg.Keys == nil {
		cfg.Keys = make(map[string]string)
	}
	if cfg.TimestampToleranceSec <= 0 {
		cfg.TimestampToleranceSec = 60
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sig := r.Header.Get(headerSignature)
			tsStr := r.Header.Get(headerTimestamp)
			keyID := r.Header.Get(headerKeyID)
			if sig == "" || tsStr == "" || keyID == "" {
				next.ServeHTTP(w, r)
				return
			}
			secret, ok := cfg.Keys[keyID]
			if !ok {
				logger.FromRequest(r).Debug().Str("key_id", keyID).Msg("hmac: unknown key_id")
				writeUnauthorized(w, r)
				return
			}
			ts, err := strconv.ParseInt(tsStr, 10, 64)
			if err != nil {
				logger.FromRequest(r).Debug().Err(err).Msg("hmac: invalid timestamp")
				writeUnauthorized(w, r)
				return
			}
			now := time.Now().Unix()
			tol := int64(cfg.TimestampToleranceSec)
			if ts < now-tol || ts > now+tol {
				logger.FromRequest(r).Debug().Int64("ts", ts).Int64("now", now).Msg("hmac: timestamp out of range")
				writeUnauthorized(w, r)
				return
			}
			bodyHash := ""
			if r.Body != nil {
				var buf bytes.Buffer
				if _, err := io.Copy(&buf, r.Body); err != nil {
					logger.FromRequest(r).Warn().Err(err).Msg("hmac: failed to read body")
					writeUnauthorized(w, r)
					return
				}
				r.Body = io.NopCloser(&buf)
				h := sha256.Sum256(buf.Bytes())
				bodyHash = hex.EncodeToString(h[:])
			}
			// method + path (+ query) + timestamp + body_hash (per SECURITY.md)
			path := r.URL.Path
			if r.URL.RawQuery != "" {
				path += "?" + r.URL.RawQuery
			}
			message := r.Method + path + strconv.FormatInt(ts, 10) + bodyHash
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(message))
			expected := hex.EncodeToString(mac.Sum(nil))
			if !hmac.Equal([]byte(sig), []byte(expected)) {
				logger.FromRequest(r).Debug().Msg("hmac: signature mismatch")
				writeUnauthorized(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// ServiceAuthChain returns a middleware that implements: mTLS (client cert) > HMAC > API Key.
// If the request has a verified TLS client certificate, the inner handler is called directly.
// If HMAC headers are present and valid, the inner handler is called (API Key skipped).
// Otherwise the request is passed to apiKeyMiddleware (e.g. API Key check).
func ServiceAuthChain(hmacCfg HMACConfig, apiKeyMiddleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	if hmacCfg.Keys == nil {
		hmacCfg.Keys = make(map[string]string)
	}
	if hmacCfg.TimestampToleranceSec <= 0 {
		hmacCfg.TimestampToleranceSec = 60
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				next.ServeHTTP(w, r)
				return
			}
			sig := r.Header.Get(headerSignature)
			tsStr := r.Header.Get(headerTimestamp)
			keyID := r.Header.Get(headerKeyID)
			if sig != "" && tsStr != "" && keyID != "" && len(hmacCfg.Keys) > 0 {
				if verifyHMAC(hmacCfg, r, sig, tsStr, keyID) {
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

// verifyHMAC reads body, verifies signature and returns true if valid. Restores r.Body.
func verifyHMAC(cfg HMACConfig, r *http.Request, sig, tsStr, keyID string) bool {
	secret, ok := cfg.Keys[keyID]
	if !ok {
		return false
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	tol := int64(cfg.TimestampToleranceSec)
	if ts < now-tol || ts > now+tol {
		return false
	}
	bodyHash := ""
	if r.Body != nil {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r.Body); err != nil {
			return false
		}
		r.Body = io.NopCloser(&buf)
		h := sha256.Sum256(buf.Bytes())
		bodyHash = hex.EncodeToString(h[:])
	}
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	message := r.Method + path + strconv.FormatInt(ts, 10) + bodyHash
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

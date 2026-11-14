package warden

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"time"
)

// HMAC v2 signing headers. The server verifies these using the same canonical form.
const (
	headerHMACSignature = "X-Signature"
	headerHMACTimestamp = "X-Timestamp"
	headerHMACKeyID     = "X-Key-Id"
	headerHMACNonce     = "X-Nonce"
	headerHMACVersion   = "X-Signature-Version"

	// signatureVersionV2 is the canonical version emitted by this SDK.
	signatureVersionV2 = "v2"
	// signatureVersionV3 is emitted when an audience is configured. It is the
	// v2 form with the audience added to the signed text.
	signatureVersionV3 = "v3"
)

// Clock abstracts time for deterministic testing of signature timestamps.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// hmacSigner signs outgoing requests using the HMAC v2 canonical form:
//
//	METHOD\nPATH_AND_QUERY\nKEY_ID\nTIMESTAMP\nNONCE\nSHA256_HEX(body)
//
// The signature is HMAC-SHA256(secret, canonical) hex-encoded. A fresh random
// nonce is generated per request to allow the server to reject replays.
//
//nolint:govet // fieldalignment: field order chosen for readability
type hmacSigner struct {
	keyID  string
	secret []byte
	clock  Clock
	// audience, when set, selects the v3 form: the name of the Warden instance
	// the request is meant for is signed, so the request is not valid at
	// another instance that shares the key.
	audience string
}

// newHMACSigner builds a signer; returns nil when signing is not configured.
func newHMACSigner(keyID, secret, audience string, clock Clock) *hmacSigner {
	if keyID == "" || secret == "" {
		return nil
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &hmacSigner{keyID: keyID, secret: []byte(secret), clock: clock, audience: audience}
}

// newNonce returns a 128-bit hex nonce from crypto/rand.
func newNonce() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// bodyHashHex returns the hex SHA-256 of the request body without consuming it.
// For a nil body the hash of the empty string is used, matching the server.
func bodyHashHex(req *http.Request) (string, error) {
	if req.Body == nil || req.Body == http.NoBody {
		sum := sha256.Sum256(nil)
		return hex.EncodeToString(sum[:]), nil
	}
	// GetBody (set by net/http for in-memory bodies) lets us read without
	// destroying the request body used by the transport.
	if req.GetBody != nil {
		rc, err := req.GetBody()
		if err != nil {
			return "", err
		}
		defer func() { _ = rc.Close() }() //nolint:errcheck // read-only body copy; close error is irrelevant
		data, err := io.ReadAll(rc)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:]), nil
	}
	// Fallback: read and restore.
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	_ = req.Body.Close() //nolint:errcheck // body fully read; close error is irrelevant
	sum := sha256.Sum256(data)
	// Restore a re-readable body.
	req.Body = io.NopCloser(bytes.NewReader(data))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil }
	return hex.EncodeToString(sum[:]), nil
}

// canonicalString builds the v2 canonical string. Exposed (unexported) for tests.
func canonicalString(method, pathAndQuery, keyID, timestamp, nonce, bodyHash string) string {
	return method + "\n" + pathAndQuery + "\n" + keyID + "\n" + timestamp + "\n" + nonce + "\n" + bodyHash
}

// canonicalStringV3 builds the v3 canonical string: a version line and the
// audience, followed by the v2 canonical string.
func canonicalStringV3(audience, method, pathAndQuery, keyID, timestamp, nonce, bodyHash string) string {
	return signatureVersionV3 + "\n" + audience + "\n" + canonicalString(method, pathAndQuery, keyID, timestamp, nonce, bodyHash)
}

// pathAndQuery returns the request path including a raw query string when present.
func pathAndQuery(req *http.Request) string {
	p := req.URL.EscapedPath()
	if p == "" {
		p = "/"
	}
	if req.URL.RawQuery != "" {
		return p + "?" + req.URL.RawQuery
	}
	return p
}

// sign computes and attaches the HMAC headers to req: v3 when an audience is
// configured, v2 otherwise. It never logs the secret.
func (s *hmacSigner) sign(req *http.Request) error {
	nonce, err := newNonce()
	if err != nil {
		return err
	}
	ts := strconv.FormatInt(s.clock.Now().Unix(), 10)
	bh, err := bodyHashHex(req)
	if err != nil {
		return err
	}
	version := signatureVersionV2
	canonical := canonicalString(req.Method, pathAndQuery(req), s.keyID, ts, nonce, bh)
	if s.audience != "" {
		version = signatureVersionV3
		canonical = canonicalStringV3(s.audience, req.Method, pathAndQuery(req), s.keyID, ts, nonce, bh)
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(canonical))
	sig := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set(headerHMACVersion, version)
	req.Header.Set(headerHMACKeyID, s.keyID)
	req.Header.Set(headerHMACTimestamp, ts)
	req.Header.Set(headerHMACNonce, nonce)
	req.Header.Set(headerHMACSignature, sig)
	return nil
}

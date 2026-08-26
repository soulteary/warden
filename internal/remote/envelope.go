package remote

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/soulteary/warden/internal/define"
)

// Envelope v2 constants. The envelope is an authenticated, versioned JSON container.
//
// Layout (JSON object, RawURL base64 for binary fields):
//
//	{
//	  "version":     "warden-remote-envelope-v2",
//	  "key_id":      "<opaque key identifier>",
//	  "key_alg":     "RSA-OAEP-256",
//	  "content_alg": "A256GCM",
//	  "enc_key":     "<base64url RSA-OAEP-SHA256(content_key)>",
//	  "nonce":       "<base64url GCM nonce>",
//	  "ciphertext":  "<base64url GCM seal output, ciphertext||tag>"
//	}
//
// Integrity is provided by AES-256-GCM. The AAD binds the metadata
// (version|key_id|key_alg|content_alg) so an attacker cannot swap algorithms
// or key ids without invalidating the tag.
const (
	// EnvelopeVersionV2 is the only supported envelope version for the v2 format.
	EnvelopeVersionV2 = "warden-remote-envelope-v2"
	// KeyAlgRSAOAEP256 identifies RSA-OAEP with SHA-256 for key wrapping.
	KeyAlgRSAOAEP256 = "RSA-OAEP-256"
	// ContentAlgA256GCM identifies AES-256-GCM for content encryption.
	ContentAlgA256GCM = "A256GCM"
	// oaepLabel is the RSA-OAEP label bound to the v2 envelope. It provides domain
	// separation so wrapped keys from other contexts cannot be reused here.
	oaepLabel = "warden-remote-envelope-v2"
	// contentKeySize is the AES-256 key size in bytes.
	contentKeySize = 32
	// maxEnvelopeBytes caps the raw envelope JSON size before parsing to bound memory.
	maxEnvelopeBytes = define.MAX_ENVELOPE_JSON_SIZE
)

// Sentinel errors for the envelope layer. Callers use errors.Is to branch on them.
// None of these errors carry key material, plaintext, or the full envelope.
var (
	// ErrEncryptionRequired is returned when encryption is required by policy but the
	// response was not a valid v2 envelope (e.g. plaintext downgrade attempt).
	ErrEncryptionRequired = errors.New("remote: encryption required but response is not an encrypted envelope")
	// ErrUnsupportedEnvelopeVersion is returned for an unknown envelope version.
	ErrUnsupportedEnvelopeVersion = errors.New("remote: unsupported envelope version")
	// ErrUnsupportedAlgorithm is returned for an unknown key_alg or content_alg.
	ErrUnsupportedAlgorithm = errors.New("remote: unsupported envelope algorithm")
	// ErrIntegrityCheckFailed is returned when authenticated decryption fails
	// (tampered ciphertext, nonce, tag, metadata, or wrong key).
	ErrIntegrityCheckFailed = errors.New("remote: envelope integrity check failed")
	// ErrEnvelopeMalformed is returned for structurally invalid envelopes
	// (bad JSON, bad base64, oversized input, trailing data, missing fields).
	ErrEnvelopeMalformed = errors.New("remote: malformed envelope")
	// ErrPlaintextTooLarge is returned when the decrypted plaintext exceeds MAX_JSON_SIZE.
	ErrPlaintextTooLarge = errors.New("remote: decrypted plaintext exceeds size limit")
	// ErrResponseTooLarge is returned when the HTTP response exceeds the encrypted
	// envelope limit instead of silently returning a truncated prefix.
	ErrResponseTooLarge = errors.New("remote: response exceeds size limit")
)

// envelopeV2 is the wire representation of a v2 envelope.
type envelopeV2 struct {
	Version    string `json:"version"`
	KeyID      string `json:"key_id"`
	KeyAlg     string `json:"key_alg"`
	ContentAlg string `json:"content_alg"`
	EncKey     string `json:"enc_key"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// looksLikeEnvelopeV2 reports whether body appears to be a JSON object (heuristic
// used only to distinguish an envelope from a plaintext JSON array, never to decide
// whether decryption is allowed).
func looksLikeEnvelopeV2(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

// parseEnvelope decodes and validates the envelope structure without decrypting.
// It enforces a size limit, rejects unknown fields and trailing data, and validates
// version/algorithm identifiers. It returns typed sentinel errors on failure.
func parseEnvelope(body []byte) (*envelopeV2, error) {
	if len(body) > maxEnvelopeBytes {
		return nil, fmt.Errorf("%w: input too large", ErrEnvelopeMalformed)
	}
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(body)))
	dec.DisallowUnknownFields()
	var env envelopeV2
	if err := dec.Decode(&env); err != nil {
		return nil, fmt.Errorf("%w: json decode failed", ErrEnvelopeMalformed)
	}
	// Reject any trailing JSON tokens after the object.
	if dec.More() {
		return nil, fmt.Errorf("%w: trailing data after envelope", ErrEnvelopeMalformed)
	}
	if env.Version != EnvelopeVersionV2 {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedEnvelopeVersion, env.Version)
	}
	if env.KeyAlg != KeyAlgRSAOAEP256 {
		return nil, fmt.Errorf("%w: key_alg %q", ErrUnsupportedAlgorithm, env.KeyAlg)
	}
	if env.ContentAlg != ContentAlgA256GCM {
		return nil, fmt.Errorf("%w: content_alg %q", ErrUnsupportedAlgorithm, env.ContentAlg)
	}
	if env.EncKey == "" || env.Nonce == "" || env.Ciphertext == "" {
		return nil, fmt.Errorf("%w: missing required field", ErrEnvelopeMalformed)
	}
	return &env, nil
}

// envelopeAAD builds the deterministic additional authenticated data. It does not
// depend on JSON/map iteration order and binds the security-relevant metadata.
func envelopeAAD(env *envelopeV2) []byte {
	// Fixed field order and separator; none of these fields are secret.
	return []byte(env.Version + "|" + env.KeyID + "|" + env.KeyAlg + "|" + env.ContentAlg)
}

// unwrapKey decrypts the wrapped content key using RSA-OAEP-SHA256 with the fixed label.
func unwrapKey(env *envelopeV2, priv *rsa.PrivateKey) ([]byte, error) {
	if priv == nil {
		return nil, fmt.Errorf("%w: nil private key", ErrIntegrityCheckFailed)
	}
	encKey, err := base64.RawURLEncoding.DecodeString(env.EncKey)
	if err != nil {
		return nil, fmt.Errorf("%w: enc_key base64", ErrEnvelopeMalformed)
	}
	key, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKey, []byte(oaepLabel))
	if err != nil {
		// Do not leak RSA error internals; treat as integrity failure.
		return nil, fmt.Errorf("%w: key unwrap", ErrIntegrityCheckFailed)
	}
	if len(key) != contentKeySize {
		return nil, fmt.Errorf("%w: content key size", ErrIntegrityCheckFailed)
	}
	return key, nil
}

// aeadOpen performs AES-256-GCM authenticated decryption. Any failure (wrong key,
// tampered ciphertext/nonce/tag/metadata) returns ErrIntegrityCheckFailed.
func aeadOpen(key, nonce, ciphertext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: cipher", ErrIntegrityCheckFailed)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: gcm", ErrIntegrityCheckFailed)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("%w: nonce size", ErrIntegrityCheckFailed)
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("%w", ErrIntegrityCheckFailed)
	}
	return plain, nil
}

// DecryptEnvelopeV2 parses and authenticates a v2 envelope and returns the plaintext.
// It enforces the decrypted plaintext size limit. Errors are typed sentinels and never
// contain key material, plaintext, or the raw envelope.
func DecryptEnvelopeV2(body []byte, priv *rsa.PrivateKey) ([]byte, error) {
	env, err := parseEnvelope(body)
	if err != nil {
		return nil, err
	}
	key, err := unwrapKey(env, priv)
	if err != nil {
		return nil, err
	}
	// Zero the content key when done to reduce residency of secret material.
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()
	nonce, err := base64.RawURLEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, fmt.Errorf("%w: nonce base64", ErrEnvelopeMalformed)
	}
	ct, err := base64.RawURLEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: ciphertext base64", ErrEnvelopeMalformed)
	}
	plain, err := aeadOpen(key, nonce, ct, envelopeAAD(env))
	if err != nil {
		return nil, err
	}
	if int64(len(plain)) > int64(define.MAX_JSON_SIZE) {
		return nil, ErrPlaintextTooLarge
	}
	return plain, nil
}

// Encrypt builds a v2 envelope for plaintext using the recipient public key and keyID.
// A fresh random 32-byte content key and GCM nonce are generated per call, so encrypting
// the same plaintext twice yields different ciphertext and nonce. Intended for producers
// and round-trip tests.
func Encrypt(plaintext []byte, pub *rsa.PublicKey, keyID string) ([]byte, error) {
	if pub == nil {
		return nil, errors.New("remote: nil public key")
	}
	key := make([]byte, contentKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("remote: content key generation: %w", err)
	}
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("remote: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("remote: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("remote: nonce generation: %w", err)
	}
	env := &envelopeV2{
		Version:    EnvelopeVersionV2,
		KeyID:      keyID,
		KeyAlg:     KeyAlgRSAOAEP256,
		ContentAlg: ContentAlgA256GCM,
	}
	ct := gcm.Seal(nil, nonce, plaintext, envelopeAAD(env))
	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, key, []byte(oaepLabel))
	if err != nil {
		return nil, fmt.Errorf("remote: key wrap: %w", err)
	}
	env.EncKey = base64.RawURLEncoding.EncodeToString(encKey)
	env.Nonce = base64.RawURLEncoding.EncodeToString(nonce)
	env.Ciphertext = base64.RawURLEncoding.EncodeToString(ct)
	out, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("remote: marshal envelope: %w", err)
	}
	return out, nil
}

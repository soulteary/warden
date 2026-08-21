package remote

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// Legacy hybrid cipher constants (v1). Retained only for the explicit legacy
// migration path. The v1 format uses AES-CTR without authentication and is
// therefore vulnerable to tampering; new deployments must use envelope v2.
const (
	// legacyRSAKeySize2048 is the RSA-2048 ciphertext block size in bytes for v1.
	// v1 assumed a fixed 2048-bit key; v2 derives the size from the key.
	legacyRSAKeySize2048 = 256
	// legacyAESKeySize is the AES key size (bytes) wrapped by RSA in v1.
	legacyAESKeySize = 32
	// legacyIVSize is the AES-CTR IV size (bytes) in v1.
	legacyIVSize = 16
)

// decryptHybridLegacy decrypts the deprecated v1 hybrid format:
// base64( RSA-OAEP_SHA256(aes_key_32 + iv_16) || aes_ctr_ciphertext ).
//
// Deprecated: v1 uses unauthenticated AES-CTR and provides no integrity protection.
// Use DecryptEnvelopeV2 instead. This function exists only for the explicit legacy
// migration path (REMOTE_ENCRYPTION_FORMAT=legacy or auto).
func decryptHybridLegacy(body []byte, priv *rsa.PrivateKey) ([]byte, error) {
	if priv == nil {
		return nil, fmt.Errorf("legacy decrypt: nil private key")
	}
	raw, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(body)))
	if err != nil {
		return nil, fmt.Errorf("legacy decrypt: base64 decode: %w", err)
	}
	// The wrapped key+IV lives inside the fixed-size RSA ciphertext block, so the
	// minimum valid body is exactly one RSA block (the AES-CTR ciphertext that
	// follows may be any length, including empty for empty plaintext).
	if len(raw) < legacyRSAKeySize2048 {
		return nil, fmt.Errorf("legacy decrypt: body too short for hybrid cipher")
	}
	encKeyBlock := raw[:legacyRSAKeySize2048]
	plainKeyIV, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKeyBlock, nil)
	if err != nil {
		return nil, fmt.Errorf("legacy decrypt: rsa decrypt failed")
	}
	if len(plainKeyIV) < legacyAESKeySize+legacyIVSize {
		return nil, fmt.Errorf("legacy decrypt: decrypted key block too short")
	}
	aesKey := plainKeyIV[:legacyAESKeySize]
	iv := plainKeyIV[legacyAESKeySize : legacyAESKeySize+legacyIVSize]
	ciphertext := raw[legacyRSAKeySize2048:]
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("legacy decrypt: aes cipher")
	}
	stream := cipher.NewCTR(block, iv)
	plain := make([]byte, len(ciphertext))
	stream.XORKeyStream(plain, ciphertext)
	return plain, nil
}

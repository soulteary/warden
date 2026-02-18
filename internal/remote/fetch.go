// Package remote provides remote config fetch with optional RSA decryption.
package remote

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/soulteary/warden/internal/define"
)

const (
	// EncryptedContentType is the Content-Type value when response body is RSA+AES encrypted.
	EncryptedContentType = "application/x-warden-encrypted"
	// RSAKeySize2048 ciphertext block size in bytes.
	RSAKeySize2048 = 256
	// AESKeySize + IVSize = 32 + 16 = 48 bytes encrypted by RSA.
	AESKeySize = 32
	// IVSize is the AES CTR IV size in bytes.
	IVSize = 16
)

// FetchDecrypted fetches url with optional auth header. If decryptEnabled and rsaKey (file path or PEM) are set,
// and response Content-Type is EncryptedContentType (or body looks like base64), decrypts with RSA private key.
// Expected encrypted format: base64( RSA-OAEP_SHA256(aes_key_32 + iv_16) || aes_ctr_ciphertext ).
// Returns decrypted or raw body and error.
// rsaKeyPath and rsaKeyPEM: use file when rsaKeyPath is non-empty, else use rsaKeyPEM (inline PEM).
func FetchDecrypted(ctx context.Context, url, authHeader string, decryptEnabled bool, rsaKeyPath, rsaKeyPEM string, timeout time.Duration, insecureTLS bool) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("remote fetch: %w", err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	client := &http.Client{Timeout: timeout}
	if insecureTLS {
		client.Transport = &http.Transport{
			// InsecureSkipVerify is intentional when HTTP_INSECURE_TLS is set (e.g. dev/self-signed).
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
		}
	}
	resp, err := client.Do(req) // #nosec G704 -- URL from config, caller is responsible for allowlist
	if err != nil {
		return nil, fmt.Errorf("remote fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // #nosec G104 -- ignore close in defer to avoid masking main error
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote fetch: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, define.MAX_JSON_SIZE))
	if err != nil {
		return nil, fmt.Errorf("remote fetch: read %w", err)
	}
	if !decryptEnabled || (rsaKeyPath == "" && rsaKeyPEM == "") {
		return body, nil
	}
	ct := strings.TrimSpace(strings.ToLower(resp.Header.Get("Content-Type")))
	if ct != EncryptedContentType && !strings.HasPrefix(ct, EncryptedContentType+";") {
		return body, nil
	}
	privKey, err := loadRSAPrivateKey(rsaKeyPath, rsaKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("remote decrypt: load key %w", err)
	}
	dec, err := decryptHybrid(body, privKey)
	if err != nil {
		return nil, fmt.Errorf("remote decrypt: %w", err)
	}
	return dec, nil
}

// loadRSAPrivateKey loads RSA private key from file path (if keyPath != "") or from inline PEM (keyPEM).
// File path takes precedence when both are set.
func loadRSAPrivateKey(keyPath, keyPEM string) (*rsa.PrivateKey, error) {
	var data []byte
	switch {
	case keyPath != "":
		keyPath = filepath.Clean(keyPath)
		var err error
		data, err = os.ReadFile(keyPath) // #nosec G304 path is from config and validated by caller
		if err != nil {
			return nil, err
		}
	case keyPEM != "":
		data = []byte(strings.TrimSpace(keyPEM))
	default:
		return nil, fmt.Errorf("no RSA private key: set REMOTE_RSA_PRIVATE_KEY_FILE or REMOTE_RSA_PRIVATE_KEY")
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block in key source")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		key2, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, err
		}
		var ok bool
		key, ok = key2.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return key, nil
	}
	return key, nil
}

// decryptHybrid expects body = base64( RSA-OAEP_SHA256(aes_key_32 + iv_16) || aes_ctr_ciphertext ).
func decryptHybrid(body []byte, priv *rsa.PrivateKey) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(body)))
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	if len(raw) < RSAKeySize2048+AESKeySize+IVSize {
		return nil, fmt.Errorf("body too short for hybrid cipher")
	}
	encKeyBlock := raw[:RSAKeySize2048]
	plainKeyIV, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKeyBlock, nil)
	if err != nil {
		return nil, fmt.Errorf("rsa decrypt: %w", err)
	}
	if len(plainKeyIV) < AESKeySize+IVSize {
		return nil, fmt.Errorf("decrypted key block too short")
	}
	aesKey := plainKeyIV[:AESKeySize]
	iv := plainKeyIV[AESKeySize : AESKeySize+IVSize]
	ciphertext := raw[RSAKeySize2048:]
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, iv)
	plain := make([]byte, len(ciphertext))
	stream.XORKeyStream(plain, ciphertext)
	return plain, nil
}

// FetchDecryptedUsers fetches remote URL and returns parsed []AllowListUser.
// If decrypt is enabled and response is encrypted, decrypts then parses JSON.
// rsaKeyPath and rsaKeyPEM: use file when rsaKeyPath is non-empty, else use rsaKeyPEM.
func FetchDecryptedUsers(ctx context.Context, url, authHeader string, decryptEnabled bool, rsaKeyPath, rsaKeyPEM string, timeout time.Duration, insecureTLS bool) ([]define.AllowListUser, error) {
	body, err := FetchDecrypted(ctx, url, authHeader, decryptEnabled, rsaKeyPath, rsaKeyPEM, timeout, insecureTLS)
	if err != nil {
		return nil, err
	}
	var users []define.AllowListUser
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("remote json parse: %w", err)
	}
	return users, nil
}

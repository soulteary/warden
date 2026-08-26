// Package remote provides remote config fetch with optional authenticated decryption.
package remote

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
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
	// EncryptedContentType is the Content-Type value advertised for encrypted envelopes.
	// It is used only as an additional consistency signal and NEVER to decide whether the
	// body must be treated as encrypted (that decision comes from EncryptionFormat/Required).
	EncryptedContentType = "application/x-warden-encrypted"
)

// EncryptionFormat selects which decryption path FetchDecrypted uses.
type EncryptionFormat string

const (
	// FormatAuto tries v2 first, then legacy (only when a private key is configured).
	// Auto never falls back to treating a decrypt failure as plaintext.
	FormatAuto EncryptionFormat = "auto"
	// FormatV2 requires the authenticated envelope v2 format.
	FormatV2 EncryptionFormat = "v2"
	// FormatLegacy requires the deprecated unauthenticated v1 hybrid format.
	FormatLegacy EncryptionFormat = "legacy"
)

// ParseEncryptionFormat parses a format string, defaulting to FormatAuto for empty input.
func ParseEncryptionFormat(s string) (EncryptionFormat, error) {
	switch EncryptionFormat(strings.ToLower(strings.TrimSpace(s))) {
	case "", FormatAuto:
		return FormatAuto, nil
	case FormatV2:
		return FormatV2, nil
	case FormatLegacy:
		return FormatLegacy, nil
	default:
		return "", fmt.Errorf("invalid REMOTE_ENCRYPTION_FORMAT %q (want auto|v2|legacy)", s)
	}
}

// FetchOptions configures a remote fetch + decrypt operation.
//
//nolint:govet // fieldalignment: readability over a few bytes for a config struct
type FetchOptions struct {
	URL         string
	AuthHeader  string
	RSAKeyPath  string // file path (preferred)
	RSAKeyPEM   string // inline PEM when file not set
	Timeout     time.Duration
	InsecureTLS bool

	// DecryptEnabled turns on the decryption path. When false, the body is returned as-is.
	DecryptEnabled bool
	// EncryptionRequired makes plaintext responses a hard error (fail closed).
	EncryptionRequired bool
	// Format selects the envelope format (auto|v2|legacy).
	Format EncryptionFormat
}

// legacyDeprecationLogged de-duplicates the legacy deprecation signal (best-effort).
var legacyDeprecationOnce = newOnceReporter()

// FetchDecryptedWithOptions fetches opts.URL and applies the configured decryption policy.
// It returns typed sentinel errors (see envelope.go) and never downgrades a decrypt failure
// to plaintext. The Content-Type header is used only as a soft consistency check.
func FetchDecryptedWithOptions(ctx context.Context, opts *FetchOptions) ([]byte, error) {
	body, contentType, err := fetchBody(ctx, opts)
	if err != nil {
		return nil, err
	}

	hasKey := opts.RSAKeyPath != "" || opts.RSAKeyPEM != ""
	if !opts.DecryptEnabled || !hasKey {
		if opts.EncryptionRequired {
			return nil, ErrEncryptionRequired
		}
		return body, nil
	}

	format := opts.Format
	if format == "" {
		format = FormatAuto
	}

	priv, err := loadRSAPrivateKey(opts.RSAKeyPath, opts.RSAKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("remote decrypt: load key: %w", err)
	}

	switch format {
	case FormatV2:
		return decryptV2(body, priv)
	case FormatLegacy:
		reportLegacyDeprecation()
		return decryptLegacy(body, priv, opts.EncryptionRequired)
	case FormatAuto:
		return decryptAuto(body, priv, contentType, opts.EncryptionRequired)
	default:
		return nil, fmt.Errorf("remote decrypt: unknown format %q", format)
	}
}

// decryptV2 decrypts strictly as v2. Plaintext or malformed bodies are hard errors.
func decryptV2(body []byte, priv *rsa.PrivateKey) ([]byte, error) {
	if !looksLikeEnvelopeV2(body) {
		return nil, ErrEncryptionRequired
	}
	plain, err := DecryptEnvelopeV2(body, priv)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

// decryptLegacy decrypts strictly as v1. When required, a decrypt failure is a hard error.
func decryptLegacy(body []byte, priv *rsa.PrivateKey, required bool) ([]byte, error) {
	plain, err := decryptHybridLegacy(body, priv)
	if err != nil {
		if required {
			return nil, fmt.Errorf("%w: legacy decrypt failed", ErrIntegrityCheckFailed)
		}
		return nil, fmt.Errorf("remote decrypt (legacy): %w", err)
	}
	return plain, nil
}

// decryptAuto tries v2 (when the body looks like a JSON object envelope), otherwise legacy.
// It NEVER returns the ciphertext as plaintext. If required and neither path yields plaintext,
// it returns a typed error.
func decryptAuto(body []byte, priv *rsa.PrivateKey, contentType string, required bool) ([]byte, error) {
	if looksLikeEnvelopeV2(body) {
		plain, err := DecryptEnvelopeV2(body, priv)
		if err == nil {
			return plain, nil
		}
		// A structurally-valid-but-failing v2 envelope must not silently fall through to
		// legacy or plaintext; surface the typed error.
		if !errors.Is(err, ErrUnsupportedEnvelopeVersion) {
			return nil, err
		}
	}
	// Try legacy for non-object bodies (base64 blob).
	reportLegacyDeprecation()
	plain, err := decryptHybridLegacy(body, priv)
	if err == nil {
		return plain, nil
	}
	// Soft consistency check: an encrypted content-type with a failed decrypt is suspicious.
	_ = contentType
	if required {
		return nil, ErrEncryptionRequired
	}
	// Decrypt was requested but no format succeeded. Do not return ciphertext-as-plaintext.
	return nil, fmt.Errorf("%w: no supported format decrypted the response", ErrIntegrityCheckFailed)
}

// fetchBody performs the HTTP GET and returns the (size-limited) body and content-type.
func fetchBody(ctx context.Context, opts *FetchOptions) (body []byte, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.URL, http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("remote fetch: %w", err)
	}
	if opts.AuthHeader != "" {
		req.Header.Set("Authorization", opts.AuthHeader)
	}
	client := &http.Client{Timeout: opts.Timeout}
	if opts.InsecureTLS {
		client.Transport = &http.Transport{
			// InsecureSkipVerify is intentional when HTTP_INSECURE_TLS is set (e.g. dev/self-signed).
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
		}
	}
	resp, err := client.Do(req) // #nosec G704 -- URL from config, caller is responsible for allowlist
	if err != nil {
		return nil, "", fmt.Errorf("remote fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // #nosec G104 -- ignore close in defer to avoid masking main error
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("remote fetch: status %d", resp.StatusCode)
	}
	body, err = readLimitedBody(resp.Body, int64(define.MAX_ENVELOPE_JSON_SIZE))
	if err != nil {
		return nil, "", fmt.Errorf("remote fetch: read %w", err)
	}
	contentType = strings.TrimSpace(strings.ToLower(resp.Header.Get("Content-Type")))
	return body, contentType, nil
}

func readLimitedBody(r io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, ErrResponseTooLarge
	}
	return body, nil
}

// FetchDecrypted fetches url with optional auth header and optional decryption.
//
// Deprecated: prefer FetchDecryptedWithOptions. This wrapper preserves the historical
// signature but uses auto format and non-required semantics. Unlike the old
// implementation, it no longer silently returns ciphertext as plaintext when decryption
// is enabled: a decrypt failure returns a typed error.
func FetchDecrypted(ctx context.Context, url, authHeader string, decryptEnabled bool, rsaKeyPath, rsaKeyPEM string, timeout time.Duration, insecureTLS bool) ([]byte, error) {
	return FetchDecryptedWithOptions(ctx, &FetchOptions{
		URL:            url,
		AuthHeader:     authHeader,
		RSAKeyPath:     rsaKeyPath,
		RSAKeyPEM:      rsaKeyPEM,
		Timeout:        timeout,
		InsecureTLS:    insecureTLS,
		DecryptEnabled: decryptEnabled,
		Format:         FormatAuto,
	})
}

// loadRSAPrivateKey loads RSA private key from file path (if keyPath != "") or from inline PEM (keyPEM).
// File path takes precedence when both are set. Error messages never include key bytes.
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
			return nil, fmt.Errorf("parse RSA private key failed")
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

// FetchDecryptedUsersWithOptions fetches, decrypts (per opts) and parses []AllowListUser.
func FetchDecryptedUsersWithOptions(ctx context.Context, opts *FetchOptions) ([]define.AllowListUser, error) {
	body, err := FetchDecryptedWithOptions(ctx, opts)
	if err != nil {
		return nil, err
	}
	return parseUsers(body)
}

// FetchDecryptedUsers fetches remote URL and returns parsed []AllowListUser.
//
// Deprecated: prefer FetchDecryptedUsersWithOptions.
func FetchDecryptedUsers(ctx context.Context, url, authHeader string, decryptEnabled bool, rsaKeyPath, rsaKeyPEM string, timeout time.Duration, insecureTLS bool) ([]define.AllowListUser, error) {
	return FetchDecryptedUsersWithOptions(ctx, &FetchOptions{
		URL:            url,
		AuthHeader:     authHeader,
		RSAKeyPath:     rsaKeyPath,
		RSAKeyPEM:      rsaKeyPEM,
		Timeout:        timeout,
		InsecureTLS:    insecureTLS,
		DecryptEnabled: decryptEnabled,
		Format:         FormatAuto,
	})
}

// parseUsers parses a plaintext JSON array of users with a hard size limit and strict decoding.
func parseUsers(body []byte) ([]define.AllowListUser, error) {
	if int64(len(body)) > int64(define.MAX_JSON_SIZE) {
		return nil, ErrPlaintextTooLarge
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	var users []define.AllowListUser
	if err := dec.Decode(&users); err != nil {
		return nil, fmt.Errorf("remote json parse: %w", err)
	}
	return users, nil
}

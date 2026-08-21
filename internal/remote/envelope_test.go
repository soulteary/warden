package remote

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustGenKey(t *testing.T, bits int) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)
	return k
}

// privToPEM encodes an RSA private key as PKCS1 PEM (test helper).
func privToPEM(t *testing.T, priv *rsa.PrivateKey) string {
	t.Helper()
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	return string(pem.EncodeToMemory(block))
}

// makeLegacyBody builds a v1 (legacy) hybrid-encrypted body for the given plaintext.
func makeLegacyBody(t *testing.T, priv *rsa.PrivateKey, plaintext []byte) []byte {
	t.Helper()
	aesKey := make([]byte, legacyAESKeySize)
	iv := make([]byte, legacyIVSize)
	_, err := rand.Read(aesKey)
	require.NoError(t, err)
	_, err = rand.Read(iv)
	require.NoError(t, err)
	keyIV := append(append([]byte{}, aesKey...), iv...)
	encKeyBlock, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &priv.PublicKey, keyIV, nil)
	require.NoError(t, err)
	block, err := aes.NewCipher(aesKey)
	require.NoError(t, err)
	stream := cipher.NewCTR(block, iv)
	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, plaintext)
	raw := append(append([]byte{}, encKeyBlock...), ciphertext...)
	return []byte(base64.StdEncoding.EncodeToString(raw))
}

func TestEnvelopeV2_RoundTrip(t *testing.T) {
	priv := mustGenKey(t, 2048)
	plaintext := []byte(`[{"phone":"13800138000","mail":"a@example.com"}]`)

	env, err := Encrypt(plaintext, &priv.PublicKey, "key-1")
	require.NoError(t, err)

	got, err := DecryptEnvelopeV2(env, priv)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestEnvelopeV2_NonDeterministicCiphertext(t *testing.T) {
	priv := mustGenKey(t, 2048)
	plaintext := []byte(`{"same":"payload"}`)

	e1, err := Encrypt(plaintext, &priv.PublicKey, "k")
	require.NoError(t, err)
	e2, err := Encrypt(plaintext, &priv.PublicKey, "k")
	require.NoError(t, err)

	var env1, env2 envelopeV2
	require.NoError(t, json.Unmarshal(e1, &env1))
	require.NoError(t, json.Unmarshal(e2, &env2))
	assert.NotEqual(t, env1.Nonce, env2.Nonce, "nonce must differ per encryption")
	assert.NotEqual(t, env1.Ciphertext, env2.Ciphertext, "ciphertext must differ per encryption")
	assert.NotEqual(t, env1.EncKey, env2.EncKey, "wrapped key must differ per encryption")
}

func TestEnvelopeV2_VariableRSAKeySizes(t *testing.T) {
	for _, bits := range []int{2048, 3072, 4096} {
		bits := bits
		t.Run(bitsName(bits), func(t *testing.T) {
			priv := mustGenKey(t, bits)
			plaintext := []byte(`[{"user_id":"u1"}]`)
			env, err := Encrypt(plaintext, &priv.PublicKey, "k")
			require.NoError(t, err)
			got, err := DecryptEnvelopeV2(env, priv)
			require.NoError(t, err)
			assert.Equal(t, plaintext, got)
		})
	}
}

func bitsName(bits int) string {
	switch bits {
	case 2048:
		return "rsa2048"
	case 3072:
		return "rsa3072"
	case 4096:
		return "rsa4096"
	default:
		return "rsa"
	}
}

func TestEnvelopeV2_TamperDetection(t *testing.T) {
	priv := mustGenKey(t, 2048)
	plaintext := []byte(`{"a":1}`)
	base, err := Encrypt(plaintext, &priv.PublicKey, "k")
	require.NoError(t, err)

	tamper := func(mutate func(e *envelopeV2)) []byte {
		var e envelopeV2
		require.NoError(t, json.Unmarshal(base, &e))
		mutate(&e)
		b, err := json.Marshal(&e)
		require.NoError(t, err)
		return b
	}

	flipB64 := func(s string) string {
		raw, err := base64.RawURLEncoding.DecodeString(s)
		require.NoError(t, err)
		require.NotEmpty(t, raw)
		raw[0] ^= 0xFF
		return base64.RawURLEncoding.EncodeToString(raw)
	}

	cases := map[string]func(e *envelopeV2){
		"ciphertext": func(e *envelopeV2) { e.Ciphertext = flipB64(e.Ciphertext) },
		"nonce":      func(e *envelopeV2) { e.Nonce = flipB64(e.Nonce) },
		"key_id_aad": func(e *envelopeV2) { e.KeyID = e.KeyID + "-x" },
		"enc_key":    func(e *envelopeV2) { e.EncKey = flipB64(e.EncKey) },
	}
	for name, mut := range cases {
		mut := mut
		t.Run(name, func(t *testing.T) {
			corrupted := tamper(mut)
			_, err := DecryptEnvelopeV2(corrupted, priv)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrIntegrityCheckFailed) || errors.Is(err, ErrEnvelopeMalformed),
				"want integrity/malformed error, got %v", err)
		})
	}
}

func TestEnvelopeV2_WrongKey(t *testing.T) {
	priv := mustGenKey(t, 2048)
	other := mustGenKey(t, 2048)
	env, err := Encrypt([]byte(`{"a":1}`), &priv.PublicKey, "k")
	require.NoError(t, err)
	_, err = DecryptEnvelopeV2(env, other)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrIntegrityCheckFailed))
}

func TestEnvelopeV2_MalformedInputs(t *testing.T) {
	priv := mustGenKey(t, 2048)
	cases := map[string][]byte{
		"not json":        []byte("not json"),
		"empty":           []byte(""),
		"array":           []byte(`[]`),
		"bad enc_key b64": []byte(`{"version":"warden-remote-envelope-v2","key_id":"k","key_alg":"RSA-OAEP-256","content_alg":"A256GCM","enc_key":"!!!","nonce":"AA","ciphertext":"AA"}`),
		"unknown field":   []byte(`{"version":"warden-remote-envelope-v2","key_id":"k","key_alg":"RSA-OAEP-256","content_alg":"A256GCM","enc_key":"AA","nonce":"AA","ciphertext":"AA","extra":1}`),
		"missing field":   []byte(`{"version":"warden-remote-envelope-v2","key_alg":"RSA-OAEP-256","content_alg":"A256GCM","enc_key":"AA","nonce":"AA"}`),
	}
	for name, body := range cases {
		body := body
		t.Run(name, func(t *testing.T) {
			_, err := DecryptEnvelopeV2(body, priv)
			require.Error(t, err)
		})
	}
}

func TestEnvelopeV2_UnknownVersionAndAlg(t *testing.T) {
	priv := mustGenKey(t, 2048)
	base, err := Encrypt([]byte(`{"a":1}`), &priv.PublicKey, "k")
	require.NoError(t, err)
	mutate := func(f func(e *envelopeV2)) []byte {
		var e envelopeV2
		require.NoError(t, json.Unmarshal(base, &e))
		f(&e)
		b, _ := json.Marshal(&e)
		return b
	}

	_, err = DecryptEnvelopeV2(mutate(func(e *envelopeV2) { e.Version = "v9" }), priv)
	assert.True(t, errors.Is(err, ErrUnsupportedEnvelopeVersion))

	_, err = DecryptEnvelopeV2(mutate(func(e *envelopeV2) { e.KeyAlg = "RSA-OAEP-1" }), priv)
	assert.True(t, errors.Is(err, ErrUnsupportedAlgorithm))

	_, err = DecryptEnvelopeV2(mutate(func(e *envelopeV2) { e.ContentAlg = "A128GCM" }), priv)
	assert.True(t, errors.Is(err, ErrUnsupportedAlgorithm))
}

func TestEnvelopeV2_TrailingData(t *testing.T) {
	priv := mustGenKey(t, 2048)
	env, err := Encrypt([]byte(`{"a":1}`), &priv.PublicKey, "k")
	require.NoError(t, err)
	withTrailing := append(append([]byte{}, bytes.TrimSpace(env)...), []byte(`{"x":1}`)...)
	_, err = DecryptEnvelopeV2(withTrailing, priv)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEnvelopeMalformed))
}

func TestEnvelopeV2_Oversized(t *testing.T) {
	priv := mustGenKey(t, 2048)
	big := bytes.Repeat([]byte("A"), maxEnvelopeBytes+1)
	_, err := DecryptEnvelopeV2(big, priv)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEnvelopeMalformed))
}

func TestEncrypt_NilPublicKey(t *testing.T) {
	_, err := Encrypt([]byte("x"), nil, "k")
	require.Error(t, err)
}

func TestDecryptEnvelopeV2_NilPrivateKey(t *testing.T) {
	priv := mustGenKey(t, 2048)
	env, err := Encrypt([]byte(`{"a":1}`), &priv.PublicKey, "k")
	require.NoError(t, err)
	// Should not panic on nil key.
	assert.NotPanics(t, func() {
		_, _ = DecryptEnvelopeV2(env, nil)
	})
}

func TestFetchDecryptedWithOptions_V2(t *testing.T) {
	priv := mustGenKey(t, 2048)
	plaintext := []byte(`[{"phone":"13800138000"}]`)
	env, err := Encrypt(plaintext, &priv.PublicKey, "k")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", EncryptedContentType)
		_, _ = w.Write(env)
	}))
	defer srv.Close()

	pemKey := privToPEM(t, priv)
	got, err := FetchDecryptedWithOptions(context.Background(), FetchOptions{
		URL:            srv.URL,
		RSAKeyPEM:      pemKey,
		Timeout:        testTimeout,
		DecryptEnabled: true,
		Format:         FormatV2,
	})
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestFetchDecryptedWithOptions_RequiredRejectsPlaintext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"phone":"1"}]`))
	}))
	defer srv.Close()

	priv := mustGenKey(t, 2048)
	_, err := FetchDecryptedWithOptions(context.Background(), FetchOptions{
		URL:                srv.URL,
		RSAKeyPEM:          privToPEM(t, priv),
		Timeout:            testTimeout,
		DecryptEnabled:     true,
		EncryptionRequired: true,
		Format:             FormatV2,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionRequired))
}

func TestFetchDecryptedWithOptions_RequiredNoDecrypt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"phone":"1"}]`))
	}))
	defer srv.Close()

	// DecryptEnabled=false but EncryptionRequired=true must fail closed.
	_, err := FetchDecryptedWithOptions(context.Background(), FetchOptions{
		URL:                srv.URL,
		Timeout:            testTimeout,
		DecryptEnabled:     false,
		EncryptionRequired: true,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEncryptionRequired))
}

func TestFetchDecryptedWithOptions_WrongContentTypeNoDowngrade(t *testing.T) {
	priv := mustGenKey(t, 2048)
	plaintext := []byte(`[{"phone":"1"}]`)
	env, err := Encrypt(plaintext, &priv.PublicKey, "k")
	require.NoError(t, err)

	// Server sends a wrong (plaintext) content-type but an actual v2 envelope body.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write(env)
	}))
	defer srv.Close()

	got, err := FetchDecryptedWithOptions(context.Background(), FetchOptions{
		URL:            srv.URL,
		RSAKeyPEM:      privToPEM(t, priv),
		Timeout:        testTimeout,
		DecryptEnabled: true,
		Format:         FormatAuto,
	})
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestFetchDecryptedWithOptions_LegacyExplicitAndDeprecation(t *testing.T) {
	ResetLegacyDeprecationForTest()
	priv := mustGenKey(t, 2048)
	legacyBody := makeLegacyBody(t, priv, []byte(`[{"phone":"1"}]`))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(legacyBody)
	}))
	defer srv.Close()

	got, err := FetchDecryptedWithOptions(context.Background(), FetchOptions{
		URL:            srv.URL,
		RSAKeyPEM:      privToPEM(t, priv),
		Timeout:        testTimeout,
		DecryptEnabled: true,
		Format:         FormatLegacy,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `[{"phone":"1"}]`, string(got))
	// Deprecation reporter should have fired once.
	assert.False(t, legacyDeprecationOnce.fire(), "deprecation should already have fired")
}

func TestParseEncryptionFormat(t *testing.T) {
	cases := map[string]struct {
		want EncryptionFormat
		err  bool
	}{
		"":       {FormatAuto, false},
		"auto":   {FormatAuto, false},
		"AUTO":   {FormatAuto, false},
		"v2":     {FormatV2, false},
		"legacy": {FormatLegacy, false},
		"bogus":  {"", true},
	}
	for in, tc := range cases {
		got, err := ParseEncryptionFormat(in)
		if tc.err {
			assert.Error(t, err, "input %q", in)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}

func TestErrorMessagesDoNotLeakSecrets(t *testing.T) {
	priv := mustGenKey(t, 2048)
	secret := "SUPER-SECRET-PLAINTEXT-13800138000"
	env, err := Encrypt([]byte(secret), &priv.PublicKey, "k")
	require.NoError(t, err)
	// Tamper to force integrity failure.
	var e envelopeV2
	require.NoError(t, json.Unmarshal(env, &e))
	raw, _ := base64.RawURLEncoding.DecodeString(e.Ciphertext)
	raw[0] ^= 0xFF
	e.Ciphertext = base64.RawURLEncoding.EncodeToString(raw)
	b, _ := json.Marshal(&e)
	_, err = DecryptEnvelopeV2(b, priv)
	require.Error(t, err)
	assert.False(t, strings.Contains(err.Error(), secret), "error must not contain plaintext")
	assert.False(t, strings.Contains(err.Error(), e.EncKey), "error must not contain wrapped key")
}

package remote

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTimeout = 5 * time.Second

func TestLoadRSAPrivateKey_FromFile(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	pemBytes := pem.EncodeToMemory(block)

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.pem")
	err = os.WriteFile(keyPath, pemBytes, 0o600)
	require.NoError(t, err)

	loaded, err := loadRSAPrivateKey(keyPath, "")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, priv.N, loaded.N)
}

func TestLoadRSAPrivateKey_FromPEM(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	pemBytes := pem.EncodeToMemory(block)

	loaded, err := loadRSAPrivateKey("", string(pemBytes))
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, priv.N, loaded.N)
}

func TestLoadRSAPrivateKey_FileTakesPrecedence(t *testing.T) {
	priv1, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	priv2, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	block1 := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv1)}
	block2 := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv2)}

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(block1), 0o600))

	loaded, err := loadRSAPrivateKey(keyPath, string(pem.EncodeToMemory(block2)))
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, priv1.N, loaded.N)
}

func TestLoadRSAPrivateKey_Empty(t *testing.T) {
	_, err := loadRSAPrivateKey("", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no RSA private key")
}

func TestLoadRSAPrivateKey_FileNotFound(t *testing.T) {
	_, err := loadRSAPrivateKey("/nonexistent/key.pem", "")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestDecryptHybrid_RoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	plaintext := []byte(`[{"phone":"13800138000","mail":"a@example.com"}]`)
	aesKey := make([]byte, AESKeySize)
	iv := make([]byte, IVSize)
	_, err = rand.Read(aesKey)
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
	body := []byte(base64.StdEncoding.EncodeToString(raw))

	dec, err := decryptHybrid(body, priv)
	require.NoError(t, err)
	assert.Equal(t, plaintext, dec)
}

func TestDecryptHybrid_InvalidBase64(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	_, err = decryptHybrid([]byte("not-base64!!!"), priv)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base64")
}

func TestDecryptHybrid_BodyTooShort(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	short := base64.StdEncoding.EncodeToString(make([]byte, 100))
	_, err = decryptHybrid([]byte(short), priv)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestFetchDecrypted_PlainResponse(t *testing.T) {
	payload := []byte(`[{"phone":"138","mail":"x@y.com"}]`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(payload)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.Background()
	body, err := FetchDecrypted(ctx, srv.URL, "", false, "", "", testTimeout, false)
	require.NoError(t, err)
	assert.Equal(t, payload, body)
}

func TestFetchDecrypted_StatusCodeNotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ctx := context.Background()
	_, err := FetchDecrypted(ctx, srv.URL, "", false, "", "", testTimeout, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

func TestFetchDecryptedUsers_PlainJSON(t *testing.T) {
	payload := []byte(`[{"phone":"13800138000","mail":"a@example.com"}]`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write(payload)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.Background()
	users, err := FetchDecryptedUsers(ctx, srv.URL, "", false, "", "", testTimeout, false)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "13800138000", users[0].Phone)
	assert.Equal(t, "a@example.com", users[0].Mail)
}

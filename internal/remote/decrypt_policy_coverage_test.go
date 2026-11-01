package remote

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAutoDecryptPreservesFormatAndFailureBoundaries(t *testing.T) {
	private := mustGenKey(t, 2048)
	plain := []byte(`[{"user_id":"member"}]`)
	v2, err := Encrypt(plain, &private.PublicKey, "policy-key")
	require.NoError(t, err)
	legacy := makeLegacyBody(t, private, plain)
	for _, body := range [][]byte{v2, legacy} {
		decoded, decodeErr := decryptAuto(body, private, EncryptedContentType, true)
		require.NoError(t, decodeErr)
		require.Equal(t, plain, decoded)
	}
	for _, required := range []bool{false, true} {
		decoded, decodeErr := decryptAuto([]byte(`[]`), private, EncryptedContentType, required)
		require.Nil(t, decoded)
		if required {
			require.ErrorIs(t, decodeErr, ErrEncryptionRequired)
		} else {
			require.ErrorIs(t, decodeErr, ErrIntegrityCheckFailed)
		}
		decoded, decodeErr = decryptLegacy([]byte("invalid base64"), private, required)
		require.Nil(t, decoded)
		require.Error(t, decodeErr)
		if required {
			require.ErrorIs(t, decodeErr, ErrIntegrityCheckFailed)
		}
	}
	var envelope envelopeV2
	require.NoError(t, json.Unmarshal(v2, &envelope))
	envelope.KeyAlg = "unsupported"
	unsupported, err := json.Marshal(envelope)
	require.NoError(t, err)
	_, err = decryptAuto(unsupported, private, EncryptedContentType, false)
	require.ErrorIs(t, err, ErrUnsupportedAlgorithm)
	envelope.KeyAlg = KeyAlgRSAOAEP256
	envelope.Version = "v99"
	future, err := json.Marshal(envelope)
	require.NoError(t, err)
	_, err = decryptAuto(future, private, EncryptedContentType, true)
	require.ErrorIs(t, err, ErrEncryptionRequired)
}
func TestPrivateKeyParsingRejectsNonRSAAndDoesNotExposeKeyMaterial(t *testing.T) {
	private := mustGenKey(t, 2048)
	der, err := x509.MarshalPKCS8PrivateKey(private)
	require.NoError(t, err)
	raw := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	loaded, err := loadRSAPrivateKey("", string(raw))
	require.NoError(t, err)
	require.Equal(t, private.N, loaded.N)
	_, other, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	otherDER, err := x509.MarshalPKCS8PrivateKey(other)
	require.NoError(t, err)
	for _, input := range []string{"not PEM", string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("secret-invalid-key")})), string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: otherDER}))} {
		key, parseErr := loadRSAPrivateKey("", input)
		require.Error(t, parseErr)
		require.Nil(t, key)
		require.NotContains(t, parseErr.Error(), input)
	}
}

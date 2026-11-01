package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/stretchr/testify/require"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTLSConfigCertificateAndClientAuthentication(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, public, private)
	require.NoError(t, err)
	key, err := x509.MarshalPKCS8PrivateKey(private)
	require.NoError(t, err)
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	badCA := filepath.Join(dir, "bad-ca.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0o600))
	require.NoError(t, os.WriteFile(badCA, []byte("not a certificate"), 0o600))
	for _, required := range []bool{false, true} {
		cfg, buildErr := buildTLSConfig(certFile, keyFile, certFile, required)
		require.NoError(t, buildErr)
		require.EqualValues(t, tls.VersionTLS12, cfg.MinVersion)
		require.Len(t, cfg.Certificates, 1)
		require.NotNil(t, cfg.ClientCAs)
		expected := tls.VerifyClientCertIfGiven
		if required {
			expected = tls.RequireAndVerifyClientCert
		}
		require.Equal(t, expected, cfg.ClientAuth)
	}
	cfg, err := buildTLSConfig(certFile, keyFile, "", false)
	require.NoError(t, err)
	require.Nil(t, cfg.ClientCAs)
	require.Equal(t, tls.NoClientCert, cfg.ClientAuth)
	srv := startServer("0", nil, certFile, keyFile, certFile, true)
	require.Equal(t, tls.RequireAndVerifyClientCert, srv.TLSConfig.ClientAuth)
	_, err = buildTLSConfig("missing", keyFile, "", false)
	require.ErrorContains(t, err, "load TLS cert/key")
	_, err = buildTLSConfig(certFile, keyFile, filepath.Join(dir, "missing"), true)
	require.ErrorContains(t, err, "read client CA")
	_, err = buildTLSConfig(certFile, keyFile, badCA, true)
	require.ErrorContains(t, err, "parse client CA")
}

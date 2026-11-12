package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
)

func TestRedisClientConfigDefaultsToPlaintextDefaultUser(t *testing.T) {
	redisCfg, err := redisClientConfig(&cmd.Config{Redis: "localhost:6379", RedisPassword: "secret"})
	require.NoError(t, err)
	require.Equal(t, "localhost:6379", redisCfg.Addr)
	require.Equal(t, "secret", redisCfg.Password)
	require.Empty(t, redisCfg.Username)
	require.Nil(t, redisCfg.TLSConfig)
}

func TestRedisClientConfigUsesACLUsernameAndTLS(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true,
		KeyUsage:    x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, public, private)
	require.NoError(t, err)
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))

	mr, err := miniredis.RunTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: private}},
	})
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	mr.RequireUserAuth("warden", "secret")

	cfg := &cmd.Config{
		Redis: mr.Addr(), RedisUsername: "warden", RedisPassword: "secret",
		RedisTLSEnabled: true, RedisTLSCAFile: caFile,
	}
	redisCfg, err := redisClientConfig(cfg)
	require.NoError(t, err)
	require.Equal(t, "warden", redisCfg.Username)
	require.NotNil(t, redisCfg.TLSConfig)
	require.False(t, redisCfg.TLSConfig.InsecureSkipVerify)

	client, err := newRecoverableRedisClient(&redisCfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Ping(context.Background()).Err())

	// Without the CA the server certificate is not trusted, and without TLS the
	// handshake never happens: neither configuration may reach the server.
	for _, broken := range []*cmd.Config{
		{Redis: mr.Addr(), RedisUsername: "warden", RedisPassword: "secret", RedisTLSEnabled: true},
		{Redis: mr.Addr(), RedisUsername: "warden", RedisPassword: "secret"},
	} {
		brokenCfg, cfgErr := redisClientConfig(broken)
		require.NoError(t, cfgErr)
		brokenCfg = brokenCfg.WithMaxRetries(-1).WithDialTimeout(time.Second).WithReadTimeout(time.Second)
		brokenClient, clientErr := newRecoverableRedisClient(&brokenCfg)
		require.Error(t, clientErr)
		if brokenClient != nil {
			require.NoError(t, brokenClient.Close())
		}
	}

	_, err = redisClientConfig(&cmd.Config{Redis: mr.Addr(), RedisTLSEnabled: true, RedisTLSCAFile: caFile + ".missing"})
	require.Error(t, err)
}

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationCollectsInvalidSourceAndSnapshotContracts(t *testing.T) {
	cfg := &Config{Port: "8081", TaskInterval: 5, Mode: "ONLY_LOCAL", Environment: "development", SourceSetID: "invalid identity", RedisEnabled: true, RedisSnapshotMode: "v2"}
	err := ValidateConfig(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SOURCE_SET_ID")
	require.Contains(t, err.Error(), "REDIS_SNAPSHOT_NAMESPACE")
}
func TestDataDirectoryAndPrivateKeyFileValidation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(file, []byte("key"), 0o600))
	for _, tc := range []struct {
		path string
		want string
	}{{filepath.Join(dir, "missing"), "does not exist"}, {file, "not a directory"}, {dir, ""}} {
		cfg := &Config{Port: "8081", TaskInterval: 5, Mode: "ONLY_LOCAL", Environment: "development", DataDir: tc.path}
		err := ValidateConfig(cfg)
		if tc.want == "" {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, tc.want)
		}
	}
	cfg := &Config{Port: "8081", TaskInterval: 5, Mode: "ONLY_LOCAL", Environment: "development", RemoteDecryptEnabled: true, RemoteRSAPrivateKeyFile: dir}
	require.ErrorContains(t, ValidateConfig(cfg), "is a directory")
	cfg.RemoteRSAPrivateKeyFile = file
	require.NoError(t, ValidateConfig(cfg))
	cfg.TLSRequireClientCert = true
	require.ErrorContains(t, ValidateConfig(cfg), "requires WARDEN_TLS_CERT")
}

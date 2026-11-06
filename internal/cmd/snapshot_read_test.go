package cmd

import (
	"github.com/soulteary/warden/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSnapshotReadPolicy(t *testing.T) {
	for _, raw := range []string{"", "observe", " OBSERVE ", "enforce", " ENFORCE "} {
		policy, err := ParseSnapshotReadPolicy(raw)
		require.NoError(t, err)
		require.Contains(t, []string{"observe", "enforce"}, policy)
	}
	_, err := ParseSnapshotReadPolicy("strict")
	require.ErrorContains(t, err, "SNAPSHOT_READ_POLICY")
	t.Setenv("SNAPSHOT_READ_POLICY", "enforce")
	withCleanFlags(t)
	for _, parsed := range []*Config{getArgsFromFlags(), convertToConfig(&config.CmdConfigData{})} {
		require.Equal(t, "enforce", parsed.SnapshotReadPolicy)
	}
	cfg := &Config{}
	processSnapshotReadPolicyFromEnv(cfg)
	require.Equal(t, "enforce", cfg.SnapshotReadPolicy)
	cfg.SnapshotReadPolicy = "invalid"
	require.ErrorContains(t, ValidateConfig(cfg), "SNAPSHOT_READ_POLICY")
}

func TestSnapshotReadPolicyDefaultsByEnvironment(t *testing.T) {
	for _, environment := range []string{"production", "prod", " PRODUCTION "} {
		policy, err := ParseSnapshotReadPolicy("", environment)
		require.NoError(t, err)
		require.Equal(t, "enforce", policy)
		policy, err = ParseSnapshotReadPolicy("observe", environment)
		require.NoError(t, err)
		require.Equal(t, "observe", policy)
	}
	for _, environment := range []string{"development", "test", ""} {
		policy, err := ParseSnapshotReadPolicy("", environment)
		require.NoError(t, err)
		require.Equal(t, "observe", policy)
	}
}

package cmd

import (
	"github.com/soulteary/warden/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSourceSetIDConfig(t *testing.T) {
	for _, value := range []string{"", "tenant-a", "dataset:prod_1"} {
		require.NoError(t, validateSourceSetID(value))
	}
	require.Error(t, validateSourceSetID("secret value"))
	t.Setenv("SOURCE_SET_ID", " tenant-a ")
	withCleanFlags(t)
	for _, cfg := range []*Config{getArgsFromFlags(), convertToConfig(&config.CmdConfigData{})} {
		require.Equal(t, "tenant-a", cfg.SourceSetID)
	}
}

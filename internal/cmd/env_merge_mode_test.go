package cmd

import (
	"os"
	"testing"

	"github.com/soulteary/cli-kit/testutil"
	"github.com/soulteary/warden/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearEnvModeVars clears every variable that participates in environment/merge-mode
// resolution so each case starts from a known-clean state.
func clearEnvModeVars(t *testing.T, envMgr *testutil.EnvManager) {
	t.Helper()
	for _, key := range []string{
		"PORT", "REDIS", "CONFIG", "KEY", "INTERVAL",
		"MODE", "MERGE_MODE", "ENVIRONMENT",
	} {
		if err := envMgr.Unset(key); err != nil {
			t.Logf("unset %s: %v", key, err)
		}
	}
}

// TestResolveEnvironmentAndMergeMode exercises the precedence and legacy-migration
// rules that split the historical overloaded MODE into Environment + MergeMode.
func TestResolveEnvironmentAndMergeMode(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	cases := []struct {
		name       string
		env        map[string]string
		wantMode   string
		wantEnv    string
		wantLegacy bool
	}{
		{
			name:       "defaults when nothing set",
			env:        map[string]string{},
			wantMode:   "DEFAULT",
			wantEnv:    "development",
			wantLegacy: false,
		},
		{
			name:       "legacy MODE used as merge mode only",
			env:        map[string]string{"MODE": "REMOTE_FIRST"},
			wantMode:   "REMOTE_FIRST",
			wantEnv:    "development",
			wantLegacy: false,
		},
		{
			name:       "legacy MODE=production migrates to environment, merge falls back to default",
			env:        map[string]string{"MODE": "production"},
			wantMode:   "DEFAULT",
			wantEnv:    "production",
			wantLegacy: true,
		},
		{
			name:       "legacy MODE=prod alias migrates",
			env:        map[string]string{"MODE": "prod"},
			wantMode:   "DEFAULT",
			wantEnv:    "production",
			wantLegacy: true,
		},
		{
			name:       "explicit ENVIRONMENT wins over legacy prod MODE",
			env:        map[string]string{"MODE": "production", "ENVIRONMENT": "development"},
			wantMode:   "DEFAULT",
			wantEnv:    "development",
			wantLegacy: false,
		},
		{
			name:       "MERGE_MODE overrides legacy MODE for merge",
			env:        map[string]string{"MODE": "REMOTE_FIRST", "MERGE_MODE": "ONLY_LOCAL"},
			wantMode:   "ONLY_LOCAL",
			wantEnv:    "development",
			wantLegacy: false,
		},
		{
			name:       "explicit ENVIRONMENT and MERGE_MODE are independent",
			env:        map[string]string{"ENVIRONMENT": "production", "MERGE_MODE": "LOCAL_FIRST"},
			wantMode:   "LOCAL_FIRST",
			wantEnv:    "production",
			wantLegacy: false,
		},
		{
			name:       "MERGE_MODE set while legacy MODE is prod: merge honored, env migrated",
			env:        map[string]string{"MODE": "production", "MERGE_MODE": "ONLY_REMOTE"},
			wantMode:   "ONLY_REMOTE",
			wantEnv:    "production",
			wantLegacy: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			envMgr := testutil.NewEnvManager()
			defer envMgr.Cleanup()
			clearEnvModeVars(t, envMgr)
			require.NoError(t, envMgr.SetMultiple(tc.env))

			os.Args = []string{"test"}
			cfg := GetArgs()

			assert.Equal(t, tc.wantMode, cfg.Mode, "merge mode")
			assert.Equal(t, tc.wantEnv, cfg.Environment, "environment")
			assert.Equal(t, tc.wantLegacy, cfg.LegacyModeUsedForEnv, "legacy-mode-used-for-env deprecation signal")

			// The resolved values must be recognized by the dedicated types.
			mode, okMode := config.ParseMergeMode(cfg.Mode)
			assert.True(t, okMode, "resolved merge mode should be valid: %q", cfg.Mode)
			assert.True(t, mode.Validate())
			env, okEnv := config.ParseEnvironment(cfg.Environment)
			assert.True(t, okEnv, "resolved environment should be valid: %q", cfg.Environment)
			assert.True(t, env.Validate())
		})
	}
}

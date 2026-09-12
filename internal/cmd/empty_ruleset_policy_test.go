package cmd

import (
	"flag"
	"os"
	"testing"

	"github.com/soulteary/warden/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessDataPolicyFromEnv(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want string
	}{
		{"未设置时保持为空并由默认策略接管", "", ""},
		{"读取一致性优先", "consistency-first", "consistency-first"},
		{"读取可用性优先", "availability-first", "availability-first"},
		{"大小写归一", "Availability-First", "availability-first"},
		{"去除空白", "  availability-first  ", "availability-first"},
		{"非法值原样保留以便校验报错", "nonsense", "nonsense"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EMPTY_RULESET_POLICY", tc.env)
			cfg := &Config{}
			processDataPolicyFromEnv(cfg)
			assert.Equal(t, tc.want, cfg.EmptyRulesetPolicy)
		})
	}
}

// TestValidateConfig_EmptyRulesetPolicy checks the startup gate. An unrecognized value has
// to fail startup: an operator who set availability-first and silently got consistency-first
// would still be exposed to the exact failure mode they opted out of.
func TestValidateConfig_EmptyRulesetPolicy(t *testing.T) {
	base := func(policy string) *Config {
		return &Config{
			Port:               "8081",
			TaskInterval:       5,
			Mode:               "ONLY_LOCAL",
			Environment:        "development",
			EmptyRulesetPolicy: policy,
		}
	}

	t.Run("留空视为默认策略", func(t *testing.T) {
		assert.NoError(t, ValidateConfig(base("")))
	})
	t.Run("一致性优先合法", func(t *testing.T) {
		assert.NoError(t, ValidateConfig(base("consistency-first")))
	})
	t.Run("可用性优先合法", func(t *testing.T) {
		assert.NoError(t, ValidateConfig(base("availability-first")))
	})
	t.Run("非法值必须导致启动失败", func(t *testing.T) {
		err := ValidateConfig(base("keep-last-known-good"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "keep-last-known-good", "错误信息应回显无效取值")
	})
}

// TestEmptyRulesetPolicy_ReachesConfigFromEnv walks the whole no-config-file startup path
// (getArgsFromFlags) instead of calling the processor directly, so a value that is read but
// never copied into the final Config is caught.
func TestEmptyRulesetPolicy_ReachesConfigFromEnv(t *testing.T) {
	t.Setenv("EMPTY_RULESET_POLICY", "availability-first")
	withCleanFlags(t)

	cfg := getArgsFromFlags()
	require.NotNil(t, cfg)
	assert.Equal(t, "availability-first", cfg.EmptyRulesetPolicy)

	policy, ok := config.ParseEmptyRulesetPolicy(cfg.EmptyRulesetPolicy)
	require.True(t, ok)
	assert.True(t, policy.KeepsLastKnownGood(), "环境变量必须真正切换到可用性优先")
}

// TestEmptyRulesetPolicy_ReachesConfigFromYAMLPath covers the other startup path: when a
// config file is used, the value travels through config.CmdConfigData and convertToConfig.
// A field added to one path but not the other is exactly how an env var becomes inert for
// half the deployments.
func TestEmptyRulesetPolicy_ReachesConfigFromYAMLPath(t *testing.T) {
	t.Setenv("EMPTY_RULESET_POLICY", "availability-first")

	yamlCfg := &config.Config{}
	cmdData := yamlCfg.ToCmdConfig()
	require.NotNil(t, cmdData)
	assert.Equal(t, "availability-first", cmdData.EmptyRulesetPolicy,
		"YAML 路径必须同样读取 EMPTY_RULESET_POLICY")

	converted := convertToConfig(cmdData)
	require.NotNil(t, converted)
	assert.Equal(t, "availability-first", converted.EmptyRulesetPolicy,
		"convertToConfig 必须把策略透传到最终配置")
}

// withCleanFlags isolates a test from the flag package's global state and from the test
// binary's own command-line arguments, which getArgsFromFlags would otherwise try to parse.
func withCleanFlags(t *testing.T) {
	t.Helper()
	origArgs := os.Args
	origFlags := flag.CommandLine
	os.Args = []string{"warden"}
	flag.CommandLine = flag.NewFlagSet("warden", flag.ContinueOnError)
	t.Cleanup(func() {
		os.Args = origArgs
		flag.CommandLine = origFlags
	})
}

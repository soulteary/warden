package define

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBool(t *testing.T) {
	for _, raw := range []string{"true", "TRUE", "True", "1", "yes", "YES", "on", "On", "t", "T", " true "} {
		value, ok := ParseBool(raw)
		assert.True(t, ok, raw)
		assert.True(t, value, raw)
	}
	for _, raw := range []string{"false", "FALSE", "False", "0", "no", "No", "off", "OFF", "f", "F", "\tfalse\n"} {
		value, ok := ParseBool(raw)
		assert.True(t, ok, raw)
		assert.False(t, value, raw)
	}
	for _, raw := range []string{"", " ", "2", "enabled", "y", "n", "tru", "yes please", "-1"} {
		value, ok := ParseBool(raw)
		assert.False(t, ok, raw)
		assert.False(t, value, raw)
	}
}

func TestEnvBool(t *testing.T) {
	const name = "WARDEN_TEST_BOOLEAN"
	t.Setenv(name, "")
	assert.True(t, EnvBool(name, true))
	assert.False(t, EnvBool(name, false))
	t.Setenv(name, "on")
	assert.True(t, EnvBool(name, false))
	t.Setenv(name, "off")
	assert.False(t, EnvBool(name, true))
	t.Setenv(name, "maybe")
	assert.False(t, EnvBool(name, true), "an unrecognized value never reads as enabled")
}

func TestValidateBooleanEnv(t *testing.T) {
	for _, name := range BooleanEnvVars {
		t.Setenv(name, "")
	}
	require.Empty(t, ValidateBooleanEnv())

	for _, name := range BooleanEnvVars {
		t.Setenv(name, "yes")
	}
	require.Empty(t, ValidateBooleanEnv())

	t.Setenv("REDIS_ENABLED", "enable")
	t.Setenv("OTLP_ENABLED", "2")
	errs := ValidateBooleanEnv()
	require.Len(t, errs, 2)
	assert.Contains(t, errs[0], `Invalid REDIS_ENABLED "enable"`)
	assert.Contains(t, errs[0], "yes/no")
	assert.Contains(t, errs[1], `Invalid OTLP_ENABLED "2"`)
}

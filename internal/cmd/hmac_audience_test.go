package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHMACAudiences(t *testing.T) {
	audiences, err := ParseHMACAudiences("")
	require.NoError(t, err)
	require.Empty(t, audiences)

	// Entries are trimmed and lower-cased; empty entries and repeats are dropped.
	audiences, err = ParseHMACAudiences(" Warden.Internal:8081 , ,prod-eu,warden.internal:8081,")
	require.NoError(t, err)
	require.Equal(t, []string{"warden.internal:8081", "prod-eu"}, audiences)

	for _, raw := range []string{"warden a", "warden\ta", "wärden", "a\x7f", strings.Repeat("a", 256)} {
		_, err = ParseHMACAudiences(raw)
		require.Error(t, err, "%q", raw)
	}
	_, err = ParseHMACAudiences(strings.Repeat("a", 255))
	require.NoError(t, err)

	sixteen := "a0,a1,a2,a3,a4,a5,a6,a7,a8,a9,a10,a11,a12,a13,a14,a15"
	audiences, err = ParseHMACAudiences(sixteen)
	require.NoError(t, err)
	require.Len(t, audiences, 16)
	_, err = ParseHMACAudiences(sixteen + ",a16")
	require.ErrorContains(t, err, "at most 16")
}

func TestParseHMACRequireAudience(t *testing.T) {
	for raw, want := range map[string]bool{"": false, " ": false, "false": false, "0": false, "true": true, " ON ": true} {
		got, err := ParseHMACRequireAudience(raw)
		require.NoError(t, err, "%q", raw)
		require.Equal(t, want, got, "%q", raw)
	}
	got, err := ParseHMACRequireAudience("enabled")
	require.Error(t, err)
	require.False(t, got)
}

// Startup refuses audience settings that cannot work as written.
func TestValidateConfigHMACAudienceSettings(t *testing.T) {
	for _, tc := range []struct {
		name      string
		audiences string
		require   string
		wantErr   string
	}{
		{name: "unset"},
		{name: "audiences only", audiences: "warden.internal:8081"},
		{name: "required with audiences", audiences: "warden.internal:8081", require: "true"},
		{name: "not required and no audiences", require: "false"},
		{name: "required without audiences", require: "true", wantErr: "WARDEN_HMAC_REQUIRE_AUDIENCE requires WARDEN_HMAC_AUDIENCES"},
		{name: "invalid audience", audiences: "warden a", wantErr: "Invalid WARDEN_HMAC_AUDIENCES"},
		{name: "invalid boolean", audiences: "warden", require: "sometimes", wantErr: `Invalid WARDEN_HMAC_REQUIRE_AUDIENCE "sometimes"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("WARDEN_HMAC_AUDIENCES", tc.audiences)
			t.Setenv("WARDEN_HMAC_REQUIRE_AUDIENCE", tc.require)
			errs := validateHMACConfig(&Config{})
			errs = append(errs, validateBooleanEnv()...)
			if tc.wantErr == "" {
				require.Empty(t, errs)
				return
			}
			require.Len(t, errs, 1)
			require.Contains(t, errs[0], tc.wantErr)
		})
	}
}

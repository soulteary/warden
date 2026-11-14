package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrustedProxyWarningNamesTheLimiterInProduction(t *testing.T) {
	for _, environment := range []string{"", "development", "test"} {
		warning := trustedProxyWarning(environment)
		require.True(t, strings.HasPrefix(warning, "TRUSTED_PROXY_IPS is empty"))
		require.Contains(t, warning, `"none"`)
		require.NotContains(t, warning, "credential guessing")
	}
	for _, environment := range []string{"production", "prod"} {
		warning := trustedProxyWarning(environment)
		require.True(t, strings.HasPrefix(warning, "TRUSTED_PROXY_IPS is empty"))
		require.Contains(t, warning, `"none"`)
		require.Contains(t, warning, "credential guessing from inside the network unthrottled")
		require.Contains(t, warning, "X-Forwarded-For")
	}
}

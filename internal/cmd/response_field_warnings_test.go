package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseFieldWarnings(t *testing.T) {
	require.Empty(t, ResponseFieldWarnings(&Config{}), "no allowlist returns every field")
	require.Empty(t, ResponseFieldWarnings(&Config{ResponseFields: []string{"phone", "status"}}))

	warnings := ResponseFieldWarnings(&Config{ResponseFields: []string{"phone", "mail", "name"}})
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "RESPONSE_FIELDS does not include status")
	require.Contains(t, warnings[0], "/v1/lookup")
}

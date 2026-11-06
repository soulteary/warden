package serviceauth

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParsePolicies(t *testing.T) {
	keys := map[string]string{"gateway": "secret", "admin": "other"}
	for _, raw := range []string{"", "  "} {
		p, err := Parse(raw, keys)
		require.NoError(t, err)
		require.Nil(t, p)
	}
	for _, raw := range []string{`{`, `[]`, `null`, `{}`, `{"gateway":{"actions":["directory:lookup"],"unknown":true}}`, `{"gateway":{"actions":["directory:lookup"]}} true`, `{"missing":{"actions":["directory:list"]}}`, `{" gateway":{"actions":["directory:list"]}}`, `{"gateway":{}}`, `{"gateway":{"actions":["*"]}}`, `{"gateway":{"actions":["directory:list","directory:list"]}}`} {
		t.Run(raw, func(t *testing.T) { _, err := Parse(raw, keys); require.Error(t, err) })
	}
	_, duplicateErr := Parse(`{"gateway":{"actions":["directory:lookup"]}}`, map[string]string{"gateway": "same", "unmapped": "same"})
	require.Error(t, duplicateErr)
	policies, err := Parse(`{"gateway":{"actions":["directory:lookup"]},"admin":{"actions":["directory:list","admin:loglevel","metrics:read"]}}`, keys)
	require.NoError(t, err)
	require.True(t, Allows(policies, "gateway", "directory:lookup"))
	require.False(t, Allows(policies, "gateway", "directory:list"))
	require.False(t, Allows(policies, "missing", "directory:lookup"))
	require.False(t, Allows(policies, "admin", ""))
}

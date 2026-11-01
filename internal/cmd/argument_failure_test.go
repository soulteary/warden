package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArgumentParseErrorsAreReportedAndLegacyWrapperPanics(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })
	t.Setenv("CONFIG_FILE", "")
	os.Args = []string{"warden", "-unknown-option"}
	_, err := GetArgsWithError()
	require.Error(t, err)
	require.Panics(t, func() { GetArgs() })
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("server:\n  port: 'invalid'\n"), 0o600))
	os.Args = []string{"warden", "-config-file", path}
	_, err = GetArgsWithError()
	require.Error(t, err)
}

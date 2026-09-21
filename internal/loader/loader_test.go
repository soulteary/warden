package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	parserkit "github.com/soulteary/parser-kit/v3"
	"github.com/soulteary/parser-kit/v3/remotesource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
)

// fileSourcePath returns the path of the file fetcher behind a source, failing
// the test when the source is not a file source.
func fileSourcePath(t *testing.T, src parserkit.Source) string {
	t.Helper()
	f, ok := unwrapFetcher(src.Fetcher).(*parserkit.FileFetcher)
	require.True(t, ok, "expected a file source, got %T", src.Fetcher)
	return f.Path()
}

// remoteSourceFetcher returns the remote fetcher behind a source, failing the
// test when the source is not a remote source.
func remoteSourceFetcher(t *testing.T, src parserkit.Source) *remotesource.Fetcher {
	t.Helper()
	f, ok := unwrapFetcher(src.Fetcher).(*remotesource.Fetcher)
	require.True(t, ok, "expected a remote source, got %T", src.Fetcher)
	return f
}

// unwrapFetcher peels the nonEmptyFetcher wrapper BuildSources puts on every
// source, so a test can assert on the parser-kit fetcher underneath.
func unwrapFetcher(f parserkit.Fetcher) parserkit.Fetcher {
	if w, ok := f.(interface{ Unwrap() parserkit.Fetcher }); ok {
		return w.Unwrap()
	}
	return f
}

func TestBuildLoadOptions(t *testing.T) {
	t.Run("nil_config_uses_defaults", func(t *testing.T) {
		opts := BuildLoadOptions(nil, "development")
		require.NotNil(t, opts)
		assert.Equal(t, int64(define.MAX_JSON_SIZE), opts.MaxBytes)
		assert.True(t, opts.AllowEmptyData)
	})

	t.Run("ONLY_REMOTE_strategy", func(t *testing.T) {
		opts := BuildLoadOptions(nil, "ONLY_REMOTE")
		require.NotNil(t, opts)
		assert.Equal(t, "fallback", string(opts.LoadStrategy))
	})

	t.Run("ONLY_LOCAL_strategy", func(t *testing.T) {
		opts := BuildLoadOptions(nil, "ONLY_LOCAL")
		require.NotNil(t, opts)
		assert.Equal(t, parserkit.LoadStrategyFallback, opts.LoadStrategy)
	})

	t.Run("default_mode_merge_strategy", func(t *testing.T) {
		opts := BuildLoadOptions(nil, "REMOTE_FIRST")
		require.NotNil(t, opts)
		assert.Equal(t, parserkit.LoadStrategyMerge, opts.LoadStrategy)
		assert.NotNil(t, opts.KeyFunc)
	})
}

// TestBuildRemoteOptions pins the timeout/TLS values that parser-kit v1 read
// from LoadOptions and v3 reads from the remote source.
func TestBuildRemoteOptions(t *testing.T) {
	t.Run("nil_config_uses_default_timeout", func(t *testing.T) {
		opts := BuildRemoteOptions(nil)
		assert.Equal(t, time.Duration(define.DEFAULT_TIMEOUT)*time.Second, opts.Timeout)
		assert.False(t, opts.InsecureSkipVerify)
	})

	t.Run("with_config", func(t *testing.T) {
		opts := BuildRemoteOptions(&cmd.Config{HTTPTimeout: 15, HTTPInsecureTLS: true})
		assert.Equal(t, 15*time.Second, opts.Timeout)
		assert.True(t, opts.InsecureSkipVerify)
	})

	t.Run("zero_timeout_is_preserved", func(t *testing.T) {
		// v1 put cfg.HTTPTimeout straight into LoadOptions, so a zero meant
		// "bounded only by the caller's context". WithTimeout(0) means the same.
		opts := BuildRemoteOptions(&cmd.Config{})
		assert.Equal(t, time.Duration(0), opts.Timeout)
	})
}

// TestBuildSourcesRemoteSettings checks that the retry policy v1 kept on
// LoadOptions is now on the remote source, with the same values.
func TestBuildSourcesRemoteSettings(t *testing.T) {
	sources := BuildSources("", "", "http://api/data", "Bearer x", "ONLY_REMOTE", BuildRemoteOptions(nil))
	require.Len(t, sources, 1)
	f := remoteSourceFetcher(t, sources[0])
	assert.Equal(t, "http://api/data", f.URL())

	retry := f.Retry()
	assert.Equal(t, define.HTTP_RETRY_MAX_RETRIES, retry.MaxRetries)
	assert.Equal(t, define.HTTP_RETRY_DELAY, retry.RetryDelay)
	// Left unset in warden, so parser-kit fills in its 30s default -- the same
	// ceiling parser-kit v1 applied.
	assert.Equal(t, 30*time.Second, retry.MaxRetryDelay)
}

func TestBuildSources(t *testing.T) {
	remoteOpts := BuildRemoteOptions(nil)

	t.Run("ONLY_LOCAL", func(t *testing.T) {
		sources := BuildSources("/data.json", "", "", "", "ONLY_LOCAL", remoteOpts)
		require.Len(t, sources, 1)
		assert.Equal(t, "/data.json", fileSourcePath(t, sources[0]))
		assert.Equal(t, 0, sources[0].Priority)
	})

	t.Run("ONLY_REMOTE_empty_url", func(t *testing.T) {
		sources := BuildSources("", "", "", "", "ONLY_REMOTE", remoteOpts)
		assert.Empty(t, sources)
	})

	t.Run("ONLY_REMOTE_with_url", func(t *testing.T) {
		sources := BuildSources("", "", "http://api/data", "Bearer x", "ONLY_REMOTE", remoteOpts)
		require.Len(t, sources, 1)
		assert.Equal(t, "http://api/data", remoteSourceFetcher(t, sources[0]).URL())
		assert.Equal(t, 0, sources[0].Priority)
	})

	t.Run("REMOTE_FIRST_with_remote", func(t *testing.T) {
		sources := BuildSources("/local.json", "", "http://remote", "key", "REMOTE_FIRST", remoteOpts)
		require.Len(t, sources, 2)
		assert.Equal(t, "http://remote", remoteSourceFetcher(t, sources[0]).URL())
		assert.Equal(t, "/local.json", fileSourcePath(t, sources[1]))
		assert.Equal(t, 0, sources[0].Priority)
		assert.Equal(t, 1, sources[1].Priority)
	})

	t.Run("REMOTE_FIRST_ALLOW_REMOTE_FAILED_keeps_remote_first", func(t *testing.T) {
		sources := BuildSources("/local.json", "", "http://remote", "", "REMOTE_FIRST_ALLOW_REMOTE_FAILED", remoteOpts)
		require.Len(t, sources, 2)
		assert.Equal(t, "http://remote", remoteSourceFetcher(t, sources[0]).URL())
		assert.Equal(t, "/local.json", fileSourcePath(t, sources[1]))
	})

	t.Run("LOCAL_FIRST_swaps_priority", func(t *testing.T) {
		sources := BuildSources("/local.json", "", "http://remote", "", "LOCAL_FIRST", remoteOpts)
		require.Len(t, sources, 2)
		assert.Equal(t, "/local.json", fileSourcePath(t, sources[0]))
		assert.Equal(t, "http://remote", remoteSourceFetcher(t, sources[1]).URL())
		assert.Equal(t, 0, sources[0].Priority)
		assert.Equal(t, 1, sources[1].Priority)
	})

	t.Run("LOCAL_FIRST_ALLOW_REMOTE_FAILED_swaps_priority", func(t *testing.T) {
		sources := BuildSources("/local.json", "", "http://remote", "", "LOCAL_FIRST_ALLOW_REMOTE_FAILED", remoteOpts)
		require.Len(t, sources, 2)
		assert.Equal(t, "/local.json", fileSourcePath(t, sources[0]))
		assert.Equal(t, "http://remote", remoteSourceFetcher(t, sources[1]).URL())
	})

	t.Run("default_no_remote_url", func(t *testing.T) {
		sources := BuildSources("/local.json", "", "", "", "development", remoteOpts)
		require.Len(t, sources, 1)
		assert.Equal(t, "/local.json", fileSourcePath(t, sources[0]))
	})

	t.Run("ONLY_LOCAL_with_dataDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.json"), []byte(`[{"phone":"1","mail":"a@x.com"}]`), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.json"), []byte(`[{"phone":"2","mail":"b@x.com"}]`), 0o600))
		sources := BuildSources("", tmpDir, "", "", "ONLY_LOCAL", remoteOpts)
		require.Len(t, sources, 2)
		assert.Equal(t, filepath.Join(tmpDir, "a.json"), fileSourcePath(t, sources[0]))
		assert.Equal(t, filepath.Join(tmpDir, "b.json"), fileSourcePath(t, sources[1]))
	})

	t.Run("REMOTE_FIRST_with_dataDir_and_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "extra.json"), []byte(`[]`), 0o600))
		sources := BuildSources("/main.json", tmpDir, "http://api/data", "Bearer x", "REMOTE_FIRST", remoteOpts)
		require.Len(t, sources, 3)
		assert.Equal(t, "http://api/data", remoteSourceFetcher(t, sources[0]).URL())
		assert.Equal(t, filepath.Join(tmpDir, "extra.json"), fileSourcePath(t, sources[1]))
		assert.Equal(t, "/main.json", fileSourcePath(t, sources[2]))
		assert.Equal(t, []int{0, 1, 2}, []int{sources[0].Priority, sources[1].Priority, sources[2].Priority})
	})

	t.Run("LOCAL_FIRST_with_dataDir_and_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "extra.json"), []byte(`[]`), 0o600))
		sources := BuildSources("/main.json", tmpDir, "http://api/data", "Bearer x", "LOCAL_FIRST", remoteOpts)
		require.Len(t, sources, 3)
		assert.Equal(t, filepath.Join(tmpDir, "extra.json"), fileSourcePath(t, sources[0]))
		assert.Equal(t, "/main.json", fileSourcePath(t, sources[1]))
		assert.Equal(t, "http://api/data", remoteSourceFetcher(t, sources[2]).URL())
		assert.Equal(t, []int{0, 1, 2}, []int{sources[0].Priority, sources[1].Priority, sources[2].Priority})
	})
}

func TestNewRulesLoader(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := &cmd.Config{HTTPTimeout: 5, HTTPInsecureTLS: false}
		r, err := NewRulesLoader(cfg, "development")
		require.NoError(t, err)
		require.NotNil(t, r)
	})

	t.Run("ONLY_LOCAL", func(t *testing.T) {
		r, err := NewRulesLoader(nil, "ONLY_LOCAL")
		require.NoError(t, err)
		require.NotNil(t, r)
	})

	t.Run("invalid encryption format", func(t *testing.T) {
		cfg := &cmd.Config{RemoteEncryptionFormat: "typo"}
		r, err := NewRulesLoader(cfg, "ONLY_REMOTE")
		require.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestRulesLoader_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "users.json")
	err := os.WriteFile(path, []byte(`[{"phone":"13800138000","mail":"a@example.com"}]`), 0o600)
	require.NoError(t, err)

	r, err := NewRulesLoader(nil, "ONLY_LOCAL")
	require.NoError(t, err)

	ctx := context.Background()
	users, err := r.FromFile(ctx, path)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.NotEmpty(t, users[0].Phone)
	assert.NotEmpty(t, users[0].Mail)
}

func TestRulesLoader_FromFile_NotFound(t *testing.T) {
	r, err := NewRulesLoader(nil, "ONLY_LOCAL")
	require.NoError(t, err)

	ctx := context.Background()
	users, err := r.FromFile(ctx, "/nonexistent/file.json")
	// parser-kit may return error or empty list; we only verify no panic
	if err != nil {
		assert.Error(t, err)
		assert.Empty(t, users)
	} else {
		assert.Empty(t, users)
	}
}

func TestRulesLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "users.json")
	err := os.WriteFile(path, []byte(`[{"phone":"13800138000","mail":"a@example.com"}]`), 0o600)
	require.NoError(t, err)

	r, err := NewRulesLoader(nil, "ONLY_LOCAL")
	require.NoError(t, err)

	ctx := context.Background()
	users, err := r.Load(ctx, path, "", "", "")
	require.NoError(t, err)
	require.Len(t, users, 1)
}

func TestRulesLoader_Load_NoSources(t *testing.T) {
	r, err := NewRulesLoader(nil, "ONLY_REMOTE")
	require.NoError(t, err)

	ctx := context.Background()
	_, err = r.Load(ctx, "", "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no sources")
}

func TestAllowListUserKey(t *testing.T) {
	t.Run("phone_priority", func(t *testing.T) {
		u := define.AllowListUser{Phone: " 13800138000 ", Mail: "a@example.com"}
		k, ok := allowListUserKey(u)
		assert.True(t, ok)
		assert.Equal(t, "13800138000", k)
	})
	t.Run("mail_fallback", func(t *testing.T) {
		u := define.AllowListUser{Phone: "", Mail: "  A@Example.COM  "}
		k, ok := allowListUserKey(u)
		assert.True(t, ok)
		assert.Equal(t, "a@example.com", k)
	})
	t.Run("both_empty", func(t *testing.T) {
		u := define.AllowListUser{}
		_, ok := allowListUserKey(u)
		assert.False(t, ok)
	})
}

func TestMergeByMode(t *testing.T) {
	remote := []define.AllowListUser{
		{Phone: "13800138000", Mail: "r@example.com", UserID: "remote1"},
		{Phone: "13900139000", Mail: "r2@example.com", UserID: "remote2"},
	}
	local := []define.AllowListUser{
		{Phone: "13800138000", Mail: "l@example.com", UserID: "local1"},
		{Phone: "13700137000", Mail: "l2@example.com", UserID: "local2"},
	}

	t.Run("REMOTE_FIRST", func(t *testing.T) {
		out := mergeByMode(remote, local, "REMOTE_FIRST")
		require.Len(t, out, 3)
		byPhone := make(map[string]define.AllowListUser)
		for _, u := range out {
			k, _ := allowListUserKey(u)
			byPhone[k] = u
		}
		assert.Equal(t, "r@example.com", byPhone["13800138000"].Mail, "remote wins on same key")
		assert.Equal(t, "r2@example.com", byPhone["13900139000"].Mail)
		assert.Equal(t, "l2@example.com", byPhone["13700137000"].Mail)
	})

	t.Run("LOCAL_FIRST", func(t *testing.T) {
		out := mergeByMode(remote, local, "LOCAL_FIRST")
		require.Len(t, out, 3)
		byPhone := make(map[string]define.AllowListUser)
		for _, u := range out {
			k, _ := allowListUserKey(u)
			byPhone[k] = u
		}
		assert.Equal(t, "l@example.com", byPhone["13800138000"].Mail, "local wins on same key")
		assert.Equal(t, "r2@example.com", byPhone["13900139000"].Mail)
		assert.Equal(t, "l2@example.com", byPhone["13700137000"].Mail)
	})

	t.Run("LOCAL_FIRST_ALLOW_REMOTE_FAILED", func(t *testing.T) {
		out := mergeByMode(remote, local, "LOCAL_FIRST_ALLOW_REMOTE_FAILED")
		require.Len(t, out, 3)
		byPhone := make(map[string]define.AllowListUser)
		for _, u := range out {
			k, _ := allowListUserKey(u)
			byPhone[k] = u
		}
		assert.Equal(t, "l@example.com", byPhone["13800138000"].Mail)
	})
}

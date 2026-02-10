package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soulteary/parser-kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
)

func TestBuildLoadOptions(t *testing.T) {
	t.Run("nil_config_uses_defaults", func(t *testing.T) {
		opts := BuildLoadOptions(nil, "development")
		require.NotNil(t, opts)
		assert.Equal(t, int64(define.MAX_JSON_SIZE), opts.MaxFileSize)
		assert.Equal(t, define.HTTP_RETRY_MAX_RETRIES, opts.MaxRetries)
		assert.Equal(t, define.HTTP_RETRY_DELAY, opts.RetryDelay)
		assert.True(t, opts.AllowEmptyData)
		assert.False(t, opts.AllowEmptyFile)
	})

	t.Run("with_config", func(t *testing.T) {
		cfg := &cmd.Config{HTTPTimeout: 15, HTTPInsecureTLS: true}
		opts := BuildLoadOptions(cfg, "development")
		require.NotNil(t, opts)
		assert.Equal(t, 15*time.Second, opts.HTTPTimeout)
		assert.True(t, opts.InsecureSkipVerify)
	})

	t.Run("ONLY_LOCAL_allows_empty_file", func(t *testing.T) {
		opts := BuildLoadOptions(nil, "ONLY_LOCAL")
		require.NotNil(t, opts)
		assert.True(t, opts.AllowEmptyFile)
	})

	t.Run("ONLY_REMOTE_strategy", func(t *testing.T) {
		opts := BuildLoadOptions(nil, "ONLY_REMOTE")
		require.NotNil(t, opts)
		assert.Equal(t, "fallback", string(opts.LoadStrategy))
	})

	t.Run("default_mode_merge_strategy", func(t *testing.T) {
		opts := BuildLoadOptions(nil, "REMOTE_FIRST")
		require.NotNil(t, opts)
		assert.Equal(t, parserkit.LoadStrategyMerge, opts.LoadStrategy)
		assert.NotNil(t, opts.KeyFunc)
	})
}

func TestBuildSources(t *testing.T) {
	t.Run("ONLY_LOCAL", func(t *testing.T) {
		sources := BuildSources("/data.json", "", "", "", "ONLY_LOCAL")
		require.Len(t, sources, 1)
		assert.Equal(t, parserkit.SourceTypeFile, sources[0].Type)
		assert.Equal(t, "/data.json", sources[0].Config.FilePath)
	})

	t.Run("ONLY_REMOTE_empty_url", func(t *testing.T) {
		sources := BuildSources("", "", "", "", "ONLY_REMOTE")
		assert.Empty(t, sources)
	})

	t.Run("ONLY_REMOTE_with_url", func(t *testing.T) {
		sources := BuildSources("", "", "http://api/data", "Bearer x", "ONLY_REMOTE")
		require.Len(t, sources, 1)
		assert.Equal(t, parserkit.SourceTypeRemote, sources[0].Type)
		assert.Equal(t, "http://api/data", sources[0].Config.RemoteURL)
		assert.Equal(t, "Bearer x", sources[0].Config.AuthorizationHeader)
	})

	t.Run("REMOTE_FIRST_with_remote", func(t *testing.T) {
		sources := BuildSources("/local.json", "", "http://remote", "key", "REMOTE_FIRST")
		require.Len(t, sources, 2)
		assert.Equal(t, parserkit.SourceTypeRemote, sources[0].Type)
		assert.Equal(t, parserkit.SourceTypeFile, sources[1].Type)
		assert.Equal(t, "/local.json", sources[1].Config.FilePath)
	})

	t.Run("LOCAL_FIRST_swaps_priority", func(t *testing.T) {
		sources := BuildSources("/local.json", "", "http://remote", "", "LOCAL_FIRST")
		require.Len(t, sources, 2)
		assert.Equal(t, parserkit.SourceTypeFile, sources[0].Type)
		assert.Equal(t, parserkit.SourceTypeRemote, sources[1].Type)
		assert.Equal(t, 0, sources[0].Priority)
		assert.Equal(t, 1, sources[1].Priority)
	})

	t.Run("default_no_remote_url", func(t *testing.T) {
		sources := BuildSources("/local.json", "", "", "", "development")
		require.Len(t, sources, 1)
		assert.Equal(t, parserkit.SourceTypeFile, sources[0].Type)
	})

	t.Run("ONLY_LOCAL_with_dataDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.json"), []byte(`[{"phone":"1","mail":"a@x.com"}]`), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.json"), []byte(`[{"phone":"2","mail":"b@x.com"}]`), 0o600))
		sources := BuildSources("", tmpDir, "", "", "ONLY_LOCAL")
		require.Len(t, sources, 2)
		assert.Equal(t, parserkit.SourceTypeFile, sources[0].Type)
		assert.Equal(t, parserkit.SourceTypeFile, sources[1].Type)
		assert.Contains(t, sources[0].Config.FilePath, ".json")
		assert.Contains(t, sources[1].Config.FilePath, ".json")
	})

	t.Run("REMOTE_FIRST_with_dataDir_and_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "extra.json"), []byte(`[]`), 0o600))
		sources := BuildSources("/main.json", tmpDir, "http://api/data", "Bearer x", "REMOTE_FIRST")
		require.GreaterOrEqual(t, len(sources), 2)
		assert.Equal(t, parserkit.SourceTypeRemote, sources[0].Type)
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

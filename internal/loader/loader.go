// Package loader provides rules loading via parser-kit (multi-source fallback/merge).
package loader

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	parserkit "github.com/soulteary/parser-kit"
	"github.com/soulteary/warden/internal/cache"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
	"github.com/soulteary/warden/internal/remote"
)

// normalizeAllowListUser normalizes each user in place (defaults, user_id) and returns the slice.
func normalizeAllowListUser(users []define.AllowListUser) []define.AllowListUser {
	for i := range users {
		users[i].Normalize()
	}
	return users
}

// allowListUserKey returns the dedup key for merge strategy; use Phone, fallback to Mail.
// Signature must match parser-kit KeyFunc[T](T)(string,bool), so value receiver is required.
//
//nolint:gocritic // hugeParam: cannot use *T, parser-kit KeyFunc is func(T)(string,bool)
func allowListUserKey(u define.AllowListUser) (string, bool) {
	k := strings.TrimSpace(u.Phone)
	if k == "" {
		k = strings.TrimSpace(strings.ToLower(u.Mail))
	}
	return k, k != ""
}

// BuildLoadOptions builds parser-kit LoadOptions from warden config and app mode.
func BuildLoadOptions(cfg *cmd.Config, appMode string) *parserkit.LoadOptions {
	mode := strings.ToUpper(strings.TrimSpace(appMode))
	opts := parserkit.DefaultLoadOptions()
	opts.MaxFileSize = define.MAX_JSON_SIZE
	opts.MaxRetries = define.HTTP_RETRY_MAX_RETRIES
	opts.RetryDelay = define.HTTP_RETRY_DELAY
	if cfg != nil {
		opts.HTTPTimeout = time.Duration(cfg.HTTPTimeout) * time.Second
		opts.InsecureSkipVerify = cfg.HTTPInsecureTLS
	} else {
		opts.HTTPTimeout = time.Duration(define.DEFAULT_TIMEOUT) * time.Second
	}
	opts.AllowEmptyFile = (mode == "ONLY_LOCAL")
	opts.AllowEmptyData = true // allow continuing to next source when one returns empty

	switch mode {
	case "ONLY_LOCAL", "ONLY_REMOTE":
		opts.LoadStrategy = parserkit.LoadStrategyFallback
	default:
		opts.LoadStrategy = parserkit.LoadStrategyMerge
		opts.KeyFunc = allowListUserKey
	}
	return opts
}

// listJSONFiles returns sorted *.json paths under dir (non-recursive).
func listJSONFiles(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

// BuildSources builds parser-kit sources for the given mode (priority order).
// When dataDir is non-empty, all *.json files in that directory are added as file sources (sorted by name).
func BuildSources(rulesFile, dataDir, configURL, auth, appMode string) []parserkit.Source {
	mode := strings.ToUpper(strings.TrimSpace(appMode))
	var sources []parserkit.Source

	addDirFiles := func(priority int) int {
		if dataDir == "" {
			return priority
		}
		files, err := listJSONFiles(dataDir)
		if err != nil || len(files) == 0 {
			return priority
		}
		for i, p := range files {
			sources = append(sources, parserkit.Source{
				Type:     parserkit.SourceTypeFile,
				Priority: priority + i,
				Config:   parserkit.SourceConfig{FilePath: p},
			})
		}
		return priority + len(files)
	}

	switch mode {
	case "ONLY_LOCAL":
		pri := 0
		pri = addDirFiles(pri)
		if rulesFile != "" {
			sources = append(sources, parserkit.Source{
				Type:     parserkit.SourceTypeFile,
				Priority: pri,
				Config:   parserkit.SourceConfig{FilePath: rulesFile},
			})
		}
		if len(sources) == 0 && rulesFile != "" {
			sources = []parserkit.Source{{
				Type:     parserkit.SourceTypeFile,
				Priority: 0,
				Config:   parserkit.SourceConfig{FilePath: rulesFile},
			}}
		}
	case "ONLY_REMOTE":
		if configURL != "" {
			sources = []parserkit.Source{{
				Type:     parserkit.SourceTypeRemote,
				Priority: 0,
				Config: parserkit.SourceConfig{
					RemoteURL:           configURL,
					AuthorizationHeader: auth,
				},
			}}
		}
	default:
		pri := 0
		if configURL != "" {
			sources = append(sources, parserkit.Source{
				Type:     parserkit.SourceTypeRemote,
				Priority: pri,
				Config: parserkit.SourceConfig{
					RemoteURL:           configURL,
					AuthorizationHeader: auth,
				},
			})
			pri++
		}
		pri = addDirFiles(pri)
		if rulesFile != "" {
			sources = append(sources, parserkit.Source{
				Type:     parserkit.SourceTypeFile,
				Priority: pri,
				Config:   parserkit.SourceConfig{FilePath: rulesFile},
			})
		}
		if mode == "LOCAL_FIRST" || mode == "LOCAL_FIRST_ALLOW_REMOTE_FAILED" {
			// swap: local (dir + file) first, then remote
			nLocal := 0
			for _, s := range sources {
				if s.Type == parserkit.SourceTypeFile {
					nLocal++
				}
			}
			if nLocal > 0 && len(sources) > nLocal {
				local := make([]parserkit.Source, 0, nLocal)
				remoteSources := make([]parserkit.Source, 0, len(sources)-nLocal)
				for _, s := range sources {
					if s.Type == parserkit.SourceTypeFile {
						local = append(local, s)
					} else {
						remoteSources = append(remoteSources, s)
					}
				}
				for i := range local {
					local[i].Priority = i
				}
				for i := range remoteSources {
					remoteSources[i].Priority = nLocal + i
				}
				local = append(local, remoteSources...)
				sources = local
			}
		}
	}
	return sources
}

// RulesLoader wraps parser-kit DataLoader and exposes FromFile/Load by (rulesFile, configURL, auth).
//
//nolint:govet // fieldalignment: keep field order for readability; optional size win would reorder bools/pointer/strings
type RulesLoader struct {
	remoteDecrypt          bool
	httpInsecureTLS        bool
	dl                     parserkit.DataLoader[define.AllowListUser]
	httpTimeout            time.Duration
	appMode                string
	remoteRSAPrivateKey    string // file path (preferred)
	remoteRSAPrivateKeyPEM string // inline PEM when file not set
	remoteEncRequired      bool   // fail closed on plaintext when true
	remoteEncFormat        remote.EncryptionFormat
}

// NewRulesLoader creates a RulesLoader using cfg and appMode.
func NewRulesLoader(cfg *cmd.Config, appMode string) (*RulesLoader, error) {
	opts := BuildLoadOptions(cfg, appMode)
	dl, err := parserkit.NewLoaderWithNormalize(opts, normalizeAllowListUser)
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(define.DEFAULT_TIMEOUT) * time.Second
	decrypt := false
	keyPath := ""
	keyPEM := ""
	encRequired := false
	encFormat := remote.FormatAuto
	if cfg != nil {
		if cfg.HTTPTimeout > 0 {
			timeout = time.Duration(cfg.HTTPTimeout) * time.Second
		}
		decrypt = cfg.RemoteDecryptEnabled && (cfg.RemoteRSAPrivateKeyFile != "" || cfg.RemoteRSAPrivateKey != "")
		keyPath = cfg.RemoteRSAPrivateKeyFile
		keyPEM = cfg.RemoteRSAPrivateKey
		encRequired = cfg.RemoteEncryptionRequired
		f, err := remote.ParseEncryptionFormat(cfg.RemoteEncryptionFormat)
		if err != nil {
			return nil, err
		}
		encFormat = f
	}
	return &RulesLoader{
		dl:                     dl,
		appMode:                appMode,
		remoteDecrypt:          decrypt,
		remoteRSAPrivateKey:    keyPath,
		remoteRSAPrivateKeyPEM: keyPEM,
		remoteEncRequired:      encRequired,
		remoteEncFormat:        encFormat,
		httpTimeout:            timeout,
		httpInsecureTLS:        cfg != nil && cfg.HTTPInsecureTLS,
	}, nil
}

// FromFile loads rules from a local file.
func (r *RulesLoader) FromFile(ctx context.Context, path string) ([]define.AllowListUser, error) {
	return r.dl.FromFile(ctx, path)
}

// Load loads rules from sources built from (rulesFile, dataDir, configURL, auth) and r.appMode.
// It preserves the historical signature and error semantics by delegating to LoadWithResult.
func (r *RulesLoader) Load(ctx context.Context, rulesFile, dataDir, configURL, auth string) ([]define.AllowListUser, error) {
	res := r.LoadWithResult(ctx, rulesFile, dataDir, configURL, auth)
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Users, nil
}

// LoadWithResult loads rules and returns a structured LoadResult that separates the
// read source from the mode decision. Remote failures no longer short-circuit before
// the mode policy is applied: modes that tolerate remote failure fall back to the
// local rule set and mark the result Degraded, while strict modes surface the root
// cause. This function never returns partial/unvalidated data with a nil Err.
func (r *RulesLoader) LoadWithResult(ctx context.Context, rulesFile, dataDir, configURL, auth string) LoadResult {
	mode := normalizeMode(r.appMode)
	now := time.Now()

	// Decryption path: remote fetch is performed explicitly so we can apply the
	// mode fallback policy uniformly across network/decrypt/integrity/JSON errors.
	if r.remoteDecrypt && configURL != "" && (r.remoteRSAPrivateKey != "" || r.remoteRSAPrivateKeyPEM != "") {
		return r.loadDecryptPath(ctx, rulesFile, dataDir, configURL, auth, mode, now)
	}

	// Non-decrypt path: parser-kit resolves the configured sources.
	sources := BuildSources(rulesFile, dataDir, configURL, auth, r.appMode)
	if len(sources) == 0 {
		return LoadResult{Source: SourceNone, LoadedAt: now, Err: fmt.Errorf("no sources for mode %s", r.appMode)}
	}
	users, err := r.dl.Load(ctx, sources...)
	if err != nil {
		// parser-kit already honors per-mode fallback across the source list; a hard
		// error here means all eligible sources failed. Attempt a local-only fallback
		// for modes that tolerate remote failure so an unreachable remote does not take
		// the service down when valid local rules exist.
		if configURL != "" && allowsRemoteFailure(mode) && mode != ModeOnlyRemote {
			if localUsers, lerr := r.loadLocalOnly(ctx, rulesFile, dataDir); lerr == nil && len(localUsers) > 0 {
				return LoadResult{
					Users:          localUsers,
					Source:         SourceLocal,
					Version:        cache.HashUserList(localUsers),
					LoadedAt:       now,
					Degraded:       true,
					DegradedReason: "remote_failed",
				}
			}
		}
		return LoadResult{Source: SourceNone, LoadedAt: now, Err: err}
	}
	return LoadResult{
		Users:    users,
		Source:   sourceForMode(mode, configURL, rulesFile, dataDir),
		Version:  cache.HashUserList(users),
		LoadedAt: now,
	}
}

// loadDecryptPath implements the encrypted remote path with uniform mode fallback.
func (r *RulesLoader) loadDecryptPath(ctx context.Context, rulesFile, dataDir, configURL, auth, mode string, now time.Time) LoadResult {
	remoteUsers, rerr := remote.FetchDecryptedUsersWithOptions(ctx, &remote.FetchOptions{
		URL:                configURL,
		AuthHeader:         auth,
		RSAKeyPath:         r.remoteRSAPrivateKey,
		RSAKeyPEM:          r.remoteRSAPrivateKeyPEM,
		Timeout:            r.httpTimeout,
		InsecureTLS:        r.httpInsecureTLS,
		DecryptEnabled:     true,
		EncryptionRequired: r.remoteEncRequired,
		Format:             r.remoteEncFormat,
	})
	if rerr != nil {
		// Uniform policy: strict modes surface the root cause; tolerant modes fall back
		// to validated local rules and mark degraded. Never return ciphertext/partial data.
		if allowsRemoteFailure(mode) {
			localUsers, lerr := r.loadLocalOnly(ctx, rulesFile, dataDir)
			if lerr == nil && len(localUsers) > 0 {
				return LoadResult{
					Users:          localUsers,
					Source:         SourceLocal,
					Version:        cache.HashUserList(localUsers),
					LoadedAt:       now,
					Degraded:       true,
					DegradedReason: "remote_failed",
				}
			}
			return LoadResult{Source: SourceNone, LoadedAt: now, Err: errors.Join(
				fmt.Errorf("remote decrypt fetch: %w", rerr), lerr)}
		}
		return LoadResult{Source: SourceNone, LoadedAt: now, Err: fmt.Errorf("remote decrypt fetch: %w", rerr)}
	}
	remoteUsers = normalizeAllowListUser(remoteUsers)

	fileSources := BuildSources(rulesFile, dataDir, "", "", r.appMode)
	if len(fileSources) == 0 {
		return LoadResult{Users: remoteUsers, Source: SourceRemote, Version: cache.HashUserList(remoteUsers), LoadedAt: now}
	}
	fileUsers, ferr := r.dl.Load(ctx, fileSources...)
	if ferr != nil {
		// Local read failed but remote succeeded: use remote (not degraded).
		return LoadResult{Users: remoteUsers, Source: SourceRemote, Version: cache.HashUserList(remoteUsers), LoadedAt: now}
	}
	merged := mergeByMode(remoteUsers, fileUsers, mode)
	return LoadResult{Users: merged, Source: SourceMerged, Version: cache.HashUserList(merged), LoadedAt: now}
}

// loadLocalOnly loads rules from local sources only (no remote), used as a fallback.
func (r *RulesLoader) loadLocalOnly(ctx context.Context, rulesFile, dataDir string) ([]define.AllowListUser, error) {
	fileSources := BuildSources(rulesFile, dataDir, "", "", r.appMode)
	if len(fileSources) == 0 {
		return nil, fmt.Errorf("no local sources available for fallback")
	}
	return r.dl.Load(ctx, fileSources...)
}

// sourceForMode reports the likely source label when parser-kit resolved sources
// without a hard error. It is a best-effort classification for observability only.
func sourceForMode(mode, configURL, rulesFile, dataDir string) Source {
	switch mode {
	case ModeOnlyLocal:
		return SourceLocal
	case ModeOnlyRemote:
		return SourceRemote
	default:
		hasLocal := rulesFile != "" || dataDir != ""
		if configURL != "" && hasLocal {
			return SourceMerged
		}
		if configURL != "" {
			return SourceRemote
		}
		return SourceLocal
	}
}

// mergeByMode merges remoteUsers and fileUsers by mode (REMOTE_FIRST = remote wins, LOCAL_FIRST = file wins).
func mergeByMode(remoteUsers, fileUsers []define.AllowListUser, mode string) []define.AllowListUser {
	keyToUser := make(map[string]define.AllowListUser)
	if mode == "LOCAL_FIRST" || mode == "LOCAL_FIRST_ALLOW_REMOTE_FAILED" {
		for i := range remoteUsers {
			k, ok := allowListUserKey(remoteUsers[i])
			if ok {
				keyToUser[k] = remoteUsers[i]
			}
		}
		for i := range fileUsers {
			k, ok := allowListUserKey(fileUsers[i])
			if ok {
				keyToUser[k] = fileUsers[i]
			}
		}
	} else {
		for i := range fileUsers {
			k, ok := allowListUserKey(fileUsers[i])
			if ok {
				keyToUser[k] = fileUsers[i]
			}
		}
		for i := range remoteUsers {
			k, ok := allowListUserKey(remoteUsers[i])
			if ok {
				keyToUser[k] = remoteUsers[i]
			}
		}
	}
	out := make([]define.AllowListUser, 0, len(keyToUser))
	for k := range keyToUser {
		out = append(out, keyToUser[k])
	}
	// Deterministic ordering: primarily by user_id, then canonical key, then mail.
	// This guarantees identical output regardless of map iteration order.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UserID != out[j].UserID {
			return out[i].UserID < out[j].UserID
		}
		ki, _ := allowListUserKey(out[i])
		kj, _ := allowListUserKey(out[j])
		if ki != kj {
			return ki < kj
		}
		return strings.ToLower(out[i].Mail) < strings.ToLower(out[j].Mail)
	})
	return out
}

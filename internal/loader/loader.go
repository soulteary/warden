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

	otelprop "github.com/soulteary/http-kit/v2/otelprop"
	parserkit "github.com/soulteary/parser-kit/v3"
	"github.com/soulteary/parser-kit/v3/remotesource"
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
//
// parser-kit v3 keeps only the decoding concerns here. Timeouts, retries and
// TLS belong to the source that has them, so they moved to RemoteOptions, and
// the former AllowEmptyFile is now AllowMissing() on the file source.
func BuildLoadOptions(_ *cmd.Config, appMode string) *parserkit.LoadOptions[define.AllowListUser] {
	mode := strings.ToUpper(strings.TrimSpace(appMode))
	opts := parserkit.DefaultLoadOptions[define.AllowListUser]()
	opts.MaxBytes = define.MAX_JSON_SIZE
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

// RemoteOptions carries the per-source remote settings that parser-kit v1 kept
// on LoadOptions. Retries are fixed by define, so only the two configurable
// values live here.
type RemoteOptions struct {
	// Timeout bounds one remote request, retries included. Zero leaves the
	// request bounded only by the caller's context, which is what a zero
	// LoadOptions.HTTPTimeout did in v1.
	Timeout time.Duration

	// InsecureSkipVerify disables TLS verification for remote sources.
	InsecureSkipVerify bool
}

// BuildRemoteOptions builds the remote source settings from warden config.
//
// The values are read exactly as BuildLoadOptions read them before the
// upgrade: cfg.HTTPTimeout seconds when a config is present -- including zero,
// which means "no bound of our own" -- and define.DEFAULT_TIMEOUT when it is
// nil. RulesLoader.httpTimeout is deliberately not reused: it substitutes
// DEFAULT_TIMEOUT for a non-positive cfg.HTTPTimeout, which is right for the
// encrypted-remote path it serves but would change the timeout parser-kit
// applies.
func BuildRemoteOptions(cfg *cmd.Config) RemoteOptions {
	if cfg == nil {
		return RemoteOptions{Timeout: time.Duration(define.DEFAULT_TIMEOUT) * time.Second}
	}
	return RemoteOptions{
		Timeout:            time.Duration(cfg.HTTPTimeout) * time.Second,
		InsecureSkipVerify: cfg.HTTPInsecureTLS,
	}
}

// nonEmptyFetcher fails a source whose content is present but zero bytes long.
//
// parser-kit v1 decoded every payload with json.Unmarshal, so an empty file or
// an empty response body failed that source with "unexpected end of JSON
// input" and the loader fell through to the next one. v3 decodes no bytes as
// an empty list instead, which AllowEmptyData -- which warden sets -- would
// then accept as a legitimate result and hand to EMPTY_RULESET_POLICY, wiping
// the rule set. This restores the v1 reading.
type nonEmptyFetcher struct {
	inner parserkit.Fetcher
}

// Fetch delegates and rejects a present-but-empty payload.
func (f nonEmptyFetcher) Fetch(ctx context.Context, maxBytes int64) ([]byte, error) {
	raw, err := f.inner.Fetch(ctx, maxBytes)
	if err != nil {
		return nil, err
	}
	// A nil slice means the source is absent -- AllowMissing() on a file that
	// is not there -- which v1 also reported as an empty result, so it passes
	// through. A non-nil empty slice is a source that exists and holds
	// nothing; io.ReadAll never returns nil for a zero-byte read, so this
	// separates the two reliably. An explicit "[]" is two bytes and is
	// unaffected.
	if raw != nil && len(raw) == 0 {
		return nil, errors.New("source returned no content")
	}
	return raw, nil
}

// Unwrap returns the wrapped fetcher, for tests that assert on the source.
func (f nonEmptyFetcher) Unwrap() parserkit.Fetcher { return f.inner }

// errFetcher reports a construction failure when the source is read.
//
// remotesource.New validates the URL up front, while v1 only discovered a bad
// URL at fetch time and failed that one source. Deferring the error keeps
// "one source fails, the others still load" intact.
type errFetcher struct {
	err error
}

// Fetch always returns the construction error.
func (f errFetcher) Fetch(_ context.Context, _ int64) ([]byte, error) { return nil, f.err }

// newFileFetcher builds the file source. allowMissing mirrors the v1
// AllowEmptyFile option, which warden set in ONLY_LOCAL mode.
func newFileFetcher(path string, allowMissing bool) parserkit.Fetcher {
	f := parserkit.File(path)
	if allowMissing {
		f = f.AllowMissing()
	}
	return nonEmptyFetcher{inner: f}
}

// newRemoteFetcher builds the remote source with the settings v1 kept on
// LoadOptions, plus the trace propagation it injected implicitly.
func newRemoteFetcher(rawURL, auth string, remoteOpts RemoteOptions) parserkit.Fetcher {
	opts := make([]remotesource.Option, 0, 5)
	if auth != "" {
		opts = append(opts, remotesource.WithAuthorization(auth))
	}
	opts = append(opts,
		remotesource.WithTimeout(remoteOpts.Timeout),
		// MaxRetryDelay is left unset so it takes the 30s default, which is
		// what v1 used.
		remotesource.WithRetry(remotesource.RetryPolicy{
			MaxRetries: define.HTTP_RETRY_MAX_RETRIES,
			RetryDelay: define.HTTP_RETRY_DELAY,
		}),
	)
	if remoteOpts.InsecureSkipVerify {
		opts = append(opts, remotesource.WithInsecureSkipVerify())
	}
	// v1 resolved the global OpenTelemetry propagator on every remote fetch;
	// v3 makes it opt-in. otelprop.Global() resolves just as late, so it picks
	// up the propagator tracing-kit installs and is a no-op until it does.
	opts = append(opts, remotesource.WithPropagator(otelprop.Global()))

	f, err := remotesource.New(rawURL, opts...)
	if err != nil {
		return errFetcher{err: err}
	}
	return nonEmptyFetcher{inner: f}
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
//
// parser-kit v3 dropped Source.Type, so local and remote fetchers are collected
// separately and the priorities are assigned per mode afterwards. The resulting
// order and priority numbers are identical to those the Type-based swap
// produced.
func BuildSources(rulesFile, dataDir, configURL, auth, appMode string, remoteOpts RemoteOptions) []parserkit.Source {
	mode := strings.ToUpper(strings.TrimSpace(appMode))
	// ONLY_LOCAL is the mode that tolerated a missing rules file in v1, via
	// LoadOptions.AllowEmptyFile.
	allowMissing := mode == "ONLY_LOCAL"

	localFiles := make([]parserkit.Fetcher, 0, 2)
	if dataDir != "" {
		if files, err := listJSONFiles(dataDir); err == nil {
			for _, path := range files {
				localFiles = append(localFiles, newFileFetcher(path, allowMissing))
			}
		}
	}
	if rulesFile != "" {
		localFiles = append(localFiles, newFileFetcher(rulesFile, allowMissing))
	}

	var remotes []parserkit.Fetcher
	if configURL != "" {
		remotes = append(remotes, newRemoteFetcher(configURL, auth, remoteOpts))
	}

	// Which family is tried first, per mode. Everything else keeps the order
	// the fetchers were collected in.
	var first, second []parserkit.Fetcher
	switch mode {
	case "ONLY_LOCAL":
		first = localFiles
	case "ONLY_REMOTE":
		first = remotes
	case "LOCAL_FIRST", "LOCAL_FIRST_ALLOW_REMOTE_FAILED":
		first, second = localFiles, remotes
	default:
		first, second = remotes, localFiles
	}

	sources := make([]parserkit.Source, 0, len(first)+len(second))
	for _, f := range first {
		sources = append(sources, parserkit.At(len(sources), f))
	}
	for _, f := range second {
		sources = append(sources, parserkit.At(len(sources), f))
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
	remoteOpts             RemoteOptions
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
		remoteOpts:             BuildRemoteOptions(cfg),
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
//
// The fetcher is built exactly as BuildSources builds a file source, so a
// missing file is tolerated in ONLY_LOCAL mode and an empty file fails, as
// parser-kit v1's FromFile did through AllowEmptyFile and its JSON decode.
func (r *RulesLoader) FromFile(ctx context.Context, path string) ([]define.AllowListUser, error) {
	allowMissing := normalizeMode(r.appMode) == ModeOnlyLocal
	return r.dl.LoadOne(ctx, newFileFetcher(path, allowMissing))
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

	// parser-kit merge mode intentionally tolerates individual source failures,
	// which would hide an unreachable remote behind a successful local load.
	// Resolve remote-first modes explicitly so strict REMOTE_FIRST surfaces the
	// failure and the tolerant variant records a degraded local fallback.
	if configURL != "" && (mode == ModeRemoteFirst || mode == ModeRemoteFirstAllowRemoteFail) {
		return r.loadPlainRemoteFirst(ctx, rulesFile, dataDir, configURL, auth, mode, now)
	}

	// Non-decrypt path: parser-kit resolves the configured sources.
	sources := BuildSources(rulesFile, dataDir, configURL, auth, r.appMode, r.remoteOpts)
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

func (r *RulesLoader) loadPlainRemoteFirst(ctx context.Context, rulesFile, dataDir, configURL, auth, mode string, now time.Time) LoadResult {
	remoteSources := BuildSources("", "", configURL, auth, ModeOnlyRemote, r.remoteOpts)
	remoteUsers, remoteErr := r.dl.Load(ctx, remoteSources...)
	if remoteErr != nil {
		if !allowsRemoteFailure(mode) {
			return LoadResult{
				Source:   SourceNone,
				LoadedAt: now,
				Err:      fmt.Errorf("remote load: %w", remoteErr),
			}
		}

		localUsers, localErr := r.loadLocalOnly(ctx, rulesFile, dataDir)
		if localErr != nil || len(localUsers) == 0 {
			return LoadResult{
				Source:   SourceNone,
				LoadedAt: now,
				Err:      errors.Join(fmt.Errorf("remote load: %w", remoteErr), localErr),
			}
		}
		return LoadResult{
			Users:          localUsers,
			Source:         SourceLocal,
			Version:        cache.HashUserList(localUsers),
			LoadedAt:       now,
			Degraded:       true,
			DegradedReason: "remote_failed",
		}
	}

	localUsers, localErr := r.loadLocalOnly(ctx, rulesFile, dataDir)
	if localErr != nil || len(localUsers) == 0 {
		return LoadResult{
			Users:    remoteUsers,
			Source:   SourceRemote,
			Version:  cache.HashUserList(remoteUsers),
			LoadedAt: now,
		}
	}

	merged := mergeByMode(remoteUsers, localUsers, mode)
	return LoadResult{
		Users:    merged,
		Source:   SourceMerged,
		Version:  cache.HashUserList(merged),
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

	fileSources := BuildSources(rulesFile, dataDir, "", "", r.appMode, r.remoteOpts)
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
	fileSources := BuildSources(rulesFile, dataDir, "", "", r.appMode, r.remoteOpts)
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

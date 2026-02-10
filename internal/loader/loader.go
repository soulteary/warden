// Package loader provides rules loading via parser-kit (multi-source fallback/merge).
package loader

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/soulteary/parser-kit"
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
	if cfg != nil {
		if cfg.HTTPTimeout > 0 {
			timeout = time.Duration(cfg.HTTPTimeout) * time.Second
		}
		decrypt = cfg.RemoteDecryptEnabled && (cfg.RemoteRSAPrivateKeyFile != "" || cfg.RemoteRSAPrivateKey != "")
		keyPath = cfg.RemoteRSAPrivateKeyFile
		keyPEM = cfg.RemoteRSAPrivateKey
	}
	return &RulesLoader{
		dl:                     dl,
		appMode:                appMode,
		remoteDecrypt:          decrypt,
		remoteRSAPrivateKey:    keyPath,
		remoteRSAPrivateKeyPEM: keyPEM,
		httpTimeout:            timeout,
		httpInsecureTLS:        cfg != nil && cfg.HTTPInsecureTLS,
	}, nil
}

// FromFile loads rules from a local file.
func (r *RulesLoader) FromFile(ctx context.Context, path string) ([]define.AllowListUser, error) {
	return r.dl.FromFile(ctx, path)
}

// Load loads rules from sources built from (rulesFile, dataDir, configURL, auth) and r.appMode.
// When remote decrypt is enabled, fetches remote with RSA decryption then merges with file sources.
func (r *RulesLoader) Load(ctx context.Context, rulesFile, dataDir, configURL, auth string) ([]define.AllowListUser, error) {
	mode := strings.ToUpper(strings.TrimSpace(r.appMode))
	if r.remoteDecrypt && configURL != "" && (r.remoteRSAPrivateKey != "" || r.remoteRSAPrivateKeyPEM != "") {
		remoteUsers, err := remote.FetchDecryptedUsers(ctx, configURL, auth, true, r.remoteRSAPrivateKey, r.remoteRSAPrivateKeyPEM, r.httpTimeout, r.httpInsecureTLS)
		if err != nil {
			return nil, fmt.Errorf("remote decrypt fetch: %w", err)
		}
		remoteUsers = normalizeAllowListUser(remoteUsers)
		fileSources := BuildSources(rulesFile, dataDir, "", "", r.appMode)
		if len(fileSources) == 0 {
			return remoteUsers, nil
		}
		fileUsers, err := r.dl.Load(ctx, fileSources...)
		if err != nil {
			return remoteUsers, nil
		}
		return mergeByMode(remoteUsers, fileUsers, mode), nil
	}
	sources := BuildSources(rulesFile, dataDir, configURL, auth, r.appMode)
	if len(sources) == 0 {
		return nil, fmt.Errorf("no sources for mode %s", r.appMode)
	}
	return r.dl.Load(ctx, sources...)
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
	return out
}

// Package loader provides rules loading via parser-kit (multi-source fallback/merge).
package loader

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/soulteary/parser-kit"
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/define"
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

// BuildSources builds parser-kit sources for the given mode (priority order).
func BuildSources(rulesFile, configURL, auth, appMode string) []parserkit.Source {
	mode := strings.ToUpper(strings.TrimSpace(appMode))
	var sources []parserkit.Source

	switch mode {
	case "ONLY_LOCAL":
		sources = []parserkit.Source{{
			Type:     parserkit.SourceTypeFile,
			Priority: 0,
			Config:   parserkit.SourceConfig{FilePath: rulesFile},
		}}
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
		// REMOTE_FIRST / LOCAL_FIRST / *_ALLOW_REMOTE_FAILED
		if configURL != "" {
			sources = append(sources, parserkit.Source{
				Type:     parserkit.SourceTypeRemote,
				Priority: 0,
				Config: parserkit.SourceConfig{
					RemoteURL:           configURL,
					AuthorizationHeader: auth,
				},
			})
		}
		sources = append(sources, parserkit.Source{
			Type:     parserkit.SourceTypeFile,
			Priority: 1,
			Config:   parserkit.SourceConfig{FilePath: rulesFile},
		})
		if mode == "LOCAL_FIRST" || mode == "LOCAL_FIRST_ALLOW_REMOTE_FAILED" {
			// swap so file is priority 0
			if len(sources) == 2 {
				sources[0], sources[1] = sources[1], sources[0]
				sources[0].Priority = 0
				sources[1].Priority = 1
			}
		}
	}
	return sources
}

// RulesLoader wraps parser-kit DataLoader and exposes FromFile/Load by (rulesFile, configURL, auth).
type RulesLoader struct {
	dl      parserkit.DataLoader[define.AllowListUser]
	appMode string
}

// NewRulesLoader creates a RulesLoader using cfg and appMode.
func NewRulesLoader(cfg *cmd.Config, appMode string) (*RulesLoader, error) {
	opts := BuildLoadOptions(cfg, appMode)
	dl, err := parserkit.NewLoaderWithNormalize(opts, normalizeAllowListUser)
	if err != nil {
		return nil, err
	}
	return &RulesLoader{dl: dl, appMode: appMode}, nil
}

// FromFile loads rules from a local file.
func (r *RulesLoader) FromFile(ctx context.Context, path string) ([]define.AllowListUser, error) {
	return r.dl.FromFile(ctx, path)
}

// Load loads rules from sources built from (rulesFile, configURL, auth) and r.appMode.
func (r *RulesLoader) Load(ctx context.Context, rulesFile, configURL, auth string) ([]define.AllowListUser, error) {
	sources := BuildSources(rulesFile, configURL, auth, r.appMode)
	if len(sources) == 0 {
		return nil, fmt.Errorf("no sources for mode %s", r.appMode)
	}
	return r.dl.Load(ctx, sources...)
}

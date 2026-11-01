package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	parserkit "github.com/soulteary/parser-kit/v3"
	"github.com/soulteary/warden/internal/cmd"
	"path/filepath"
	"time"
)

// SourceDescriptor identifies a configured resource without retaining its address.
type SourceDescriptor struct {
	ID      string
	Ordinal string
	Kind    Source
}

// SourceManifest records one actual source-building pass, including discovery.
// Repeated fallback discovery may produce a different revision in the same load.
type SourceManifest struct {
	ObservedAt  time.Time
	Revision    string
	SourceSetID string
	Discovery   string
	Sources     []SourceDescriptor
	Sequence    int
}

func sourceIdentity(set string, kind Source, resource string) string {
	if kind == SourceLocal {
		if absolute, err := filepath.Abs(resource); err == nil {
			resource = absolute
		}
		resource = filepath.Clean(resource)
	}
	body, err := json.Marshal([]string{"source-v1", set, string(kind), resource})
	if err != nil {
		panic("source identity contains unsupported fields")
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}
func (e *sourceEvidence) manifest(sources []SourceDescriptor, discovery string) SourceManifest {
	body, err := json.Marshal(struct {
		Set       string
		Discovery string
		Sources   []SourceDescriptor
	}{e.sourceSetID, discovery, sources})
	if err != nil {
		panic("source identity contains unsupported fields")
	}
	digest := sha256.Sum256(body)
	m := SourceManifest{ObservedAt: time.Now().UTC(), Sources: append([]SourceDescriptor{}, sources...), Revision: hex.EncodeToString(digest[:]), SourceSetID: e.sourceSetID, Discovery: discovery, Sequence: len(e.manifests) + 1}
	e.manifests = append(e.manifests, m)
	return m
}

type identifiedFetcher struct {
	inner      parserkit.Fetcher
	descriptor SourceDescriptor
	revision   string
}

func (f *identifiedFetcher) Fetch(ctx context.Context, maxBytes int64) ([]byte, error) {
	return f.inner.Fetch(ctx, maxBytes)
}
func (r *RulesLoader) buildObservedSources(ctx context.Context, file, dir, url, auth, mode string) []parserkit.Source {
	discovery := "not_requested"
	sources := buildSources(file, dir, url, auth, mode, r.remoteOpts, &discovery)
	e := evidenceFrom(ctx)
	if e == nil {
		return sources
	}
	descriptors := make([]SourceDescriptor, len(sources))
	counts := map[Source]int{}
	for i, source := range sources {
		kind := fetcherKind(source.Fetcher)
		counts[kind]++
		resource := url
		if kind == SourceLocal {
			if wrapper, ok := source.Fetcher.(nonEmptyFetcher); ok {
				if file, ok := wrapper.inner.(*parserkit.FileFetcher); ok {
					resource = file.Path()
				}
			}
		}
		descriptors[i] = SourceDescriptor{ID: sourceIdentity(e.sourceSetID, kind, resource), Ordinal: fmt.Sprintf("%s_%d", kind, counts[kind]), Kind: kind}
	}
	manifest := e.manifest(descriptors, discovery)
	for i := range sources {
		sources[i].Fetcher = &identifiedFetcher{inner: sources[i].Fetcher, descriptor: descriptors[i], revision: manifest.Revision}
	}
	return sources
}

// CloneManifests detaches the nested source lists for immutable consumers.
func CloneManifests(input []SourceManifest) []SourceManifest {
	if input == nil {
		return nil
	}
	out := append([]SourceManifest{}, input...)
	for i := range out {
		out[i].Sources = append([]SourceDescriptor(nil), input[i].Sources...)
	}
	return out
}

func sourceSetIDFromConfig(cfg *cmd.Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.SourceSetID
}

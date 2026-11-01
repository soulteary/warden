package loader

import (
	"context"
	"fmt"
	parserkit "github.com/soulteary/parser-kit/v3"
	"github.com/soulteary/warden/internal/define"
	"time"
)

// SourceAttempt describes an actual source invocation, not a configured source.
// IDs are resource digests; SourceOrdinal retains the per-family position.
// HTTP retries remain one invocation.
// SelectedInput refers to loader resolution, not subsequent cache validation or
// publication. WinningRows counts rows surviving the loader's existing merges.
// An empty or fully shadowed input can be selected with zero winning rows.
type SourceAttempt struct {
	StartedAt        time.Time
	FinishedAt       time.Time
	SourceID         string
	SourceOrdinal    string
	ManifestRevision string
	Kind             Source
	Transport        string
	Outcome          string
	Reason           string
	Rows             int
	WinningRows      int
	Attempt          int
	SelectedInput    bool
}

type evidenceKey struct{}
type sourceEvidence struct {
	selected    *sourceBatch
	sourceSetID string
	attempts    []SourceAttempt
	manifests   []SourceManifest
}
type sourceBatch struct {
	evidence      *sourceEvidence
	owners        map[string]int
	inputs        map[int]bool
	pending       int
	current       int
	fallbackOwner int
}

func evidenceFrom(ctx context.Context) *sourceEvidence {
	if evidence, ok := ctx.Value(evidenceKey{}).(*sourceEvidence); ok {
		return evidence
	}
	return nil
}
func newSourceBatch(e *sourceEvidence) *sourceBatch {
	return &sourceBatch{evidence: e, owners: map[string]int{}, inputs: map[int]bool{}, pending: -1, current: -1, fallbackOwner: -1}
}
func (e *sourceEvidence) begin(id string, kind Source, transport string) int {
	index := len(e.attempts)
	e.attempts = append(e.attempts, SourceAttempt{Attempt: index + 1, SourceID: id, Kind: kind, Transport: transport, StartedAt: time.Now().UTC()})
	return index
}
func (b *sourceBatch) finishPending() {
	if b.pending >= 0 {
		a := &b.evidence.attempts[b.pending]
		a.Outcome = "failure"
		a.Reason = "decode_failed"
		a.FinishedAt = time.Now().UTC()
		b.pending = -1
	}
}

type observedFetcher struct {
	inner    parserkit.Fetcher
	batch    *sourceBatch
	id       string
	kind     Source
	ordinal  string
	revision string
}

func (f *observedFetcher) Fetch(ctx context.Context, maxBytes int64) ([]byte, error) {
	f.batch.finishPending()
	transport := "plain"
	if f.kind == SourceLocal {
		transport = "file"
	}
	i := f.batch.evidence.begin(f.id, f.kind, transport)
	f.batch.evidence.attempts[i].SourceOrdinal = f.ordinal
	f.batch.evidence.attempts[i].ManifestRevision = f.revision
	f.batch.current = i
	raw, err := f.inner.Fetch(ctx, maxBytes)
	a := &f.batch.evidence.attempts[i]
	if err != nil {
		a.Outcome = "failure"
		a.Reason = "fetch_failed"
		a.FinishedAt = time.Now().UTC()
	} else {
		f.batch.pending = i
		if raw == nil {
			a.Outcome = "absent"
		}
	}
	return raw, err
}
func fetcherKind(fetcher parserkit.Fetcher) Source {
	if identified, ok := fetcher.(*identifiedFetcher); ok {
		return identified.descriptor.Kind
	}
	if wrapper, ok := fetcher.(nonEmptyFetcher); ok {
		if _, local := wrapper.inner.(*parserkit.FileFetcher); local {
			return SourceLocal
		}
	}
	return SourceRemote
}

// Observe the existing parser's normalization and key callbacks: no extra fetch,
// decode, normalization or replacement merge implementation is needed.
func (r *RulesLoader) loadObserved(ctx context.Context, sources ...parserkit.Source) ([]define.AllowListUser, error) {
	e := evidenceFrom(ctx)
	if e == nil {
		return r.dl.Load(ctx, sources...)
	}
	b := newSourceBatch(e)
	opts := *r.loadOptions
	originalKey := opts.KeyFunc
	if originalKey != nil {
		opts.KeyFunc = func(user define.AllowListUser) (string, bool) {
			key, include := originalKey(user)
			if include {
				b.owners[key] = b.current
			}
			return key, include
		}
	}
	dl, err := parserkit.NewLoaderWithNormalize(&opts, func(users []define.AllowListUser) []define.AllowListUser {
		users = normalizeAllowListUser(users)
		a := &e.attempts[b.current]
		if a.Outcome != "absent" {
			a.Outcome = "success"
		}
		a.Rows = len(users)
		a.FinishedAt = time.Now().UTC()
		b.pending = -1
		b.inputs[b.current] = true
		if opts.LoadStrategy == parserkit.LoadStrategyFallback {
			b.fallbackOwner = b.current
		}
		return users
	})
	if err != nil {
		return nil, err
	}
	wrapped := make([]parserkit.Source, len(sources))
	counts := map[Source]int{}
	for i, source := range sources {
		kind := fetcherKind(source.Fetcher)
		counts[kind]++
		ordinal := fmt.Sprintf("%s_%d", kind, counts[kind])
		id := ordinal
		revision := ""
		if identified, ok := source.Fetcher.(*identifiedFetcher); ok {
			id = identified.descriptor.ID
			ordinal = identified.descriptor.Ordinal
			revision = identified.revision
		}
		wrapped[i] = parserkit.At(source.Priority, &observedFetcher{inner: source.Fetcher, batch: b, kind: kind, id: id, ordinal: ordinal, revision: revision})
	}
	users, err := dl.Load(ctx, wrapped...)
	b.finishPending()
	e.selected = nil
	if err == nil {
		e.selected = b
	}
	return users, err
}
func selectedBatch(ctx context.Context) *sourceBatch {
	if e := evidenceFrom(ctx); e != nil {
		return e.selected
	}
	return nil
}
func selectBatch(ctx context.Context, b *sourceBatch) {
	if e := evidenceFrom(ctx); e != nil {
		e.selected = b
	}
}

func mergeObserved(ctx context.Context, remoteUsers, fileUsers []define.AllowListUser, mode string, remoteBatch, fileBatch *sourceBatch) []define.AllowListUser {
	users := mergeByMode(remoteUsers, fileUsers, mode)
	if e := evidenceFrom(ctx); e != nil {
		merged := newSourceBatch(e)
		// Fallback batches have one owner; materialize per-row ownership only
		// when their results participate in a subsequent merge.
		for _, group := range []struct {
			batch *sourceBatch
			users []define.AllowListUser
		}{{remoteBatch, remoteUsers}, {fileBatch, fileUsers}} {
			if group.batch != nil && group.batch.fallbackOwner >= 0 {
				for _, user := range group.users {
					if key, ok := allowListUserKey(user); ok {
						group.batch.owners[key] = group.batch.fallbackOwner
					}
				}
			}
		}
		first, second := fileBatch, remoteBatch
		if mode == ModeLocalFirst || mode == ModeLocalFirstAllowRemoteFail {
			first, second = remoteBatch, fileBatch
		}
		for _, b := range []*sourceBatch{first, second} {
			if b != nil {
				for id := range b.inputs {
					merged.inputs[id] = true
				}
				for key, id := range b.owners {
					merged.owners[key] = id
				}
			}
		}
		e.selected = merged
	}
	return users
}
func (e *sourceEvidence) finish(result *LoadResult) {
	if result.Err == nil && e.selected != nil {
		for id := range e.selected.inputs {
			e.attempts[id].SelectedInput = true
		}
		if e.selected.fallbackOwner >= 0 {
			e.attempts[e.selected.fallbackOwner].WinningRows = len(result.Users)
		} else {
			for i := range result.Users {
				key, ok := allowListUserKey(result.Users[i])
				id, owned := e.selected.owners[key]
				if !ok || !owned {
					id = e.selected.fallbackOwner
				}
				if id >= 0 {
					e.attempts[id].WinningRows++
				}
			}
		}
	}
	result.Attempts = append([]SourceAttempt(nil), e.attempts...)
	result.Manifests = CloneManifests(e.manifests)
	result.SourceSetID = e.sourceSetID
}

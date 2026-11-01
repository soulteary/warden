# Scope ownership and retained memory

`PrepareRules` detaches the input outer array and each non-nil Scope slice before
identity validation. Its candidate input and effective indexes use separate Scope
buffers. The private immutable ReadView rows share Scope only with those indexes.
Public cache reads, pages, iteration callbacks and indexed lookups return detached
Scope slices. Frozen snapshot input provenance is detached as well; its immutable
ReadView remains pinned to the original generation.

The SDK separately detaches Scope on cache writes and reads. A cache hit captures
an immutable view under a lock and clones after unlocking. Callers still have to
synchronize mutation of their own inputs while an API is reading them.

## Identity validation boundary

`ValidateAndIndexUsers` does not mutate caller records and returns a detached outer
array. Its historical nested Scope ownership remains shallow: the identity API
alone is not a deep-copy API. It does not modify existing Scope contents.

`ValidateAndIndexOwnedUsers` consumes an exclusively owned outer array. It
normalizes records before validation and can reorder that array after validation
succeeds. Errors retain original record positions after normalization. Callers
remain responsible for nested slice ownership. The cache's Prepare boundary makes
the deep copies required before using this owned path.

Owned results have a non-nil empty result and visible capacity equal to length.
For a non-empty result, clipping capacity does not release the original backing
allocation. A 16-record input with capacity 65,536 still pins that allocation;
append restrictions are not a memory-release guarantee. Prepare clones to exact
length, so its input does not carry this oversized-array overhead. No automatic
copy threshold or new Scope sharing is introduced.

## Validation and measurement

Tests cover overlapping input scopes, empty scopes with spare capacity, nil
scopes, rejected candidates, repeated publication and retained views. The identity
permutation and frozen legacy hash tests preserve order and both versions.

Run on Linux, separately from other builds:

```sh
GOMAXPROCS=1 go test ./internal/identity -run '^$' -bench '^BenchmarkOwnedBackingRetention$' -benchmem -benchtime=1x -count=3
```

`live-B/result` is a post-GC heap diagnostic with the result kept alive. It is not
a test assertion, peak memory or RSS. See [the staged benchmark guide](CANDIDATE_BENCHMARKS.md)
for candidate, freeze, complete local-file refresh and real Redis measurements.

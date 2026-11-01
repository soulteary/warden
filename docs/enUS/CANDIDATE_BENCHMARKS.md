# Candidate allocation and retained-view benchmarks

Run these benchmarks on Linux, separately from race tests and other builds:

```sh
GOMAXPROCS=1 go test ./internal/cache -run '^$' -bench '^BenchmarkCandidateStages/rows=10000/' -benchmem -benchtime=1s -count=5
GOMAXPROCS=1 go test ./internal/cache -run '^$' -bench '^BenchmarkReadViewRetention/' -benchmem -benchtime=1x -count=3
```

Stage fixtures contain unique identifiers, valid mixed-case email addresses and
non-empty Scope slices. Setup verifies the effective row count. CloneInput,
IdentityPublic and IndexesAndRows isolate their named operations; Prepare measures
the complete candidate path. Stage setup and ownership differ, so do not add their
results as an exact breakdown. IdentityPublic continues to measure the public
copy-preserving API even if preparation later uses a private outer slice.

Retention creates distinct generations and keeps their ReadViews alive across a
forced GC. It reports the live heap delta per pinned set and per generation.
Only view references are pinned, not PreparedRules. The measured heap includes the
rows, indexes, strings and scopes reachable from those views. Allocation B/op
measures generation construction; live-B measures retained data after GC. Neither
is peak memory, RSS or a stable pass/fail threshold. Small or negative heap deltas
can occur from runtime noise; repeat the measurement in isolation.

Compare baseline and changed commits using the same Go version, architecture,
fixtures and CPU configuration. Keep raw samples and report allocation and live
heap results separately. A reduction in temporary copying need not reduce the
memory retained by an old view. Retained-view setup is intentionally expensive;
use the bounded one-iteration command above for diagnostics.

## Owned validation ordering

```sh
GOMAXPROCS=1 go test ./internal/identity -run '^$' -bench '^BenchmarkOwnedValidationOrder/' -benchmem -benchtime=1s -count=5
```

The sorted, reverse and shuffled fixtures use 10,000 unique valid identifiers.
Public and Owned both reset a preallocated outer scratch slice inside the timed
loop; that reset copy is included in timing but allocates no additional list.
Public measures the copy-preserving entry point. Owned consumes the private outer
slice. Neither benchmark includes the caller-side Scope detachment performed by
PrepareRules; use the full Prepare stage to assess the complete candidate path.

## Refresh, frozen provenance and Redis stages

The root-package benchmarks exercise production functions without changing the
application's defaults. Run on Linux, separately from race checks and other builds:

```sh
GOMAXPROCS=1 go test . -run '^$' -bench '^BenchmarkRefreshStages$' -benchmem -benchtime=1s -count=3
GOMAXPROCS=1 go test . -run '^$' -bench '^BenchmarkSnapshotFreezeRetention$' -benchmem -benchtime=1x -count=3
WARDEN_REDIS_INTEGRATION_ADDR=localhost:6379 GOMAXPROCS=1 go test . -run '^$' -bench '^BenchmarkRedisSnapshotStages$' -benchmem -benchtime=1s -count=3
```

Use an explicitly selected disposable Redis 8 server. If the variable is absent,
Redis benchmarks skip; an unreachable or incompatible configured server fails.
Each fixture uses its own namespace and deletes only its snapshot and lease keys.
No database or script-cache flush is performed. These are warm steady-state
measurements, not cold-script or Redis Cluster benchmarks.

`BenchmarkRefreshStages` covers 0/1/1,000/10,000 valid unique identities and source
Scope shapes nil, empty, two short strings and one long string plus two short
strings. Normalization can make nil/empty inputs equivalent. Fixture generation,
JSON encoding and file writes are excluded. `Load` includes actual file reading,
parsing, merging, hashing and source diagnostics. `Prepare` uses loaded data;
`Freeze` copies snapshot provenance and Scope while retaining its immutable view.
These stages are separate measurements, not additive slices of one operation.

`RefreshChanged` and `RefreshUnchanged` call the actual `backgroundTask`, including
load, policy checks, publication, snapshot freezing, diagnostics and metrics.
An app-local Info logger writes to io.Discard, retaining log encoding but
excluding terminal I/O; no global logger settings are changed.
Changed refresh alternates two real payloads with distinct names/versions; empty
input has no changed variant. File writes are excluded, but reads are measured.
Redis is disabled in the local variant. The Redis variant includes writer election,
publication and release. Both changed and unchanged accepted source loads
republish V2; these are not TTL-only renewal benchmarks. Final counts and both
versions are checked;
Redis results must match the accepted snapshot. This local-file fixture does not
measure HTTP/decryption, overlapping refreshes or failure/rejection workloads.

Redis `PublishLease` includes acquire, Publish (Scope isolation, digest validation,
JSON encoding and script execution), and release. `ReadDecodeValidate` includes
transfer, strict JSON decoding and envelope validation. `Restore` calls the actual
versioned restore path, including Read, Prepare and snapshot publication; it repeats
bootstrap-style restore with a nil expected snapshot and does not model follow
eligibility or process startup. The source success time and both versions must
survive. Setup publication and client construction are outside timed loops.

`BenchmarkSnapshotFreezeRetention` pins 1 or 16 frozen provenance copies of one
1,000-row view, including separately copied Scope buffers and source evidence.
It reports post-GC `live-B/pinned-set`, not independently built index generations,
peak heap or RSS. Use the existing `BenchmarkReadViewRetention` for distinct old
index generations. Neither residency diagnostic is an exact-memory test gate.
Report B/op and allocs/op alongside timings; small samples or variable timings
alone do not establish a production throughput improvement.

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

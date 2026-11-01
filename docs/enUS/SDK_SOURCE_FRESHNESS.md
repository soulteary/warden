# Optional source-bounded SDK reads

The default remains compatible: no metadata request, no source-age requirement, and the existing local CacheTTL and status handling.

```go
opts := warden.DefaultOptions().WithBaseURL("https://warden.example").
    WithSourceFreshness(2*time.Minute, 5*time.Second)
client, err := warden.NewClient(opts)
```

A positive maximum source age enables the constraint. The second duration is a conservative allowance for relative clock disagreement: it is subtracted from the deadline, not added as extra stale time. Both durations must be non-negative; an enabled maximum must exceed the allowance. Zero maximum disables the constraint. WithClock also controls source/cache time when enabled; custom clocks must be concurrency-safe. Keep clocks synchronized.

Deploy the server's optional snapshot-response metadata support before enabling SDK constraints. Each data request negotiates `Warden-Snapshot-Metadata: v1` and requires a version digest, accepted source-success time and valid-until time. Missing, duplicate or malformed evidence from an older server/V1 recovery is rejected rather than assigned receipt time. Timestamps beyond the future-clock allowance are rejected.

The effective deadline is `min(server valid-until, source-success + client maximum age) - allowance`. List caching expires at the earlier of that deadline and local CacheTTL. Receipt, retries, same-content responses and body decoding cannot extend the original source deadline. A response that expires while reading/decoding is rejected and not cached. Such a rejection does not erase another concurrent request's fresh cache generation. Zero local TTL still disables reuse.

List, paginated and single-user reads enforce the same evidence check. A fresh 404 remains NOT_FOUND; an unknown/stale 404 returns SOURCE_FRESHNESS so absence is not asserted from untrusted-age data. Existing 401/503 handling remains unchanged. No expired-cache fallback is added. CheckUserInList still returns false on errors; use data-read methods to inspect Error.Code or errors.Is with ErrSourceUnknown, ErrSourceStale and ErrSourceFuture.

This bounds accepted resolved-set age, not the upstream content's creation time or guaranteed remote success. The headers rely on a trusted service/transport; a digest is not a body signature. Local fallback evidence retains its existing meaning. Downstream session lifetimes need their own revocation budget. Strict SDK caching does not enable server-side strict reads or change JSON.

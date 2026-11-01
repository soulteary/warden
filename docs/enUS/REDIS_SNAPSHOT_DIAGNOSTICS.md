# Redis snapshot migration diagnostics

`warden_redis_snapshot_operations_total{mode,operation,result}` records V2/shadow decisions separately from the existing cache hit/miss and refresh-error metrics. Legacy paths do not emit V2 operations. Labels are fixed allowlists; user records, keys, namespaces, versions, URLs and raw errors are never label values.

Operations: `read`, `publish`, `renew`, `lease_acquire`, `lease_release`.

Read outcomes distinguish `restored`, `missing`, `unavailable`, `schema_rejected`, `dataset_rejected`, `policy_rejected`, `timestamp_rejected`, `too_large`, `invalid_envelope`, `contract_rejected`, `effective_rejected` and `io_error`. `restored` means the application accepted the effective set, rather than merely decoding JSON. Rejected recovery logs contain a fixed result code, without payloads.

Publication distinguishes `published`, `no_lease`, `unknown_source_time`, `legacy_write_error` (shadow V1 failure) and bounded failure codes. A nil error after a skipped publication is not a successful V2 write. Renewal uses `renewed`, `missing`, `revision_conflict`, `lease_lost`, `too_large`, `invalid_envelope` or `io_error`. Missing/conflicting values are not recreated. Lease acquisition reports `acquired`, `busy`, `unavailable` or an error; release reports successful command completion or an error. A release success does not prove this process still owned the lease, since owner-safe release may do nothing.

In existing versioned health metadata, a known `input_version`, `effective_version`, `effective_count` and a known `redis_revision` supplement the compatibility `version`. V1 recovery has no original input evidence, so it omits `input_version`. Production detail hiding remains in force. Local metadata and a successful historical publication do not prove the Redis key still exists now; use operation outcomes and Redis monitoring together.

This changes diagnostics only: cache selection, source age, TTL, lease safety, default legacy behavior, data response bodies and HTTP statuses remain unchanged. Detailed storage errors retain broad `errors.Is` compatibility for invalid/contract/conflict errors. Snapshot schema and stored JSON fields are unchanged.

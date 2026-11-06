# Redis snapshot migration

`REDIS_SNAPSHOT_MODE` is an environment-only setting, including when YAML is used.

- `legacy` (default): existing array keys, bootstrap, TTL and health-age behavior. No V2 reads or writes. HTTP responses keep their existing fields.
- `shadow`: legacy reads and writes remain authoritative. After a successful legacy write, a leased writer also publishes V2 when source time is known. V2 errors do not undo the legacy write. Shadow never bootstraps from V2.
- `v2`: read and write only V2. Missing, malformed, oversized or incompatible V2 snapshots fall through to existing source loading. No automatic V1 fallback or conversion.

Shadow and V2 require Redis enabled and `REDIS_SNAPSHOT_NAMESPACE`: 1–64 letters, digits, `:`, `_`, `-`, starting with a letter or digit. Choose a namespace per logical dataset; do not share it between unrelated deployments. A policy fingerprint covers the merge mode, identity policy, empty-set policy and encryption path. A mismatch rejects recovery. Changing policy requires a source refresh; use a new namespace when running differing policies concurrently.

## Times and age

- `source_succeeded_at`: completion time of the accepted resolved source validation, including a successful identical-content check. An accepted local fallback is evidence for the resolved local set, not proof that remote succeeded. Rejected loads and last-known-good retention do not advance it.
- `published_at`: successful V2 publication time. Republishing newly validated identical data assigns a new revision. TTL renewal does not change this time.
- `restored_at`: this process's adoption time, never stored back as source evidence.

Shadow/V2 source snapshots and V2 recoveries expose known timestamps and `origin` in snapshot health metadata. Unknown timestamps are omitted. V1 restore has unknown source age, so shadow does not manufacture a V2 timestamp from it.

In V2, `loaded_at`, `age_seconds`, freshness health and the snapshot-age gauge use `source_succeeded_at`. Restarting and renewing TTL cannot rejuvenate the data. A stale V2 baseline can serve queries only under `observe`. Since v1.6.0, production defaults to `SNAPSHOT_READ_POLICY=enforce`: expired or unknown source evidence returns 503. Development/test defaults to `observe`. Storage format selection is independent of read policy; changing to V2 does not itself change an explicitly configured policy. Original input count/version survive recovery, even when validation removed all records; genuine empty revocations remain distinct.

## Rollout and rollback

1. Decide the read policy before upgrading. Production defaults to `enforce`; a legacy restore has unknown source age and cannot serve data until a source load succeeds. To stage availability during migration, explicitly set `SNAPSHOT_READ_POLICY=observe`, accepting stale reads, and switch to `enforce` after source evidence is established. The storage default remains legacy. Run and validate on Linux; use containers on other hosts.
2. Select a dataset namespace and deploy shadow mode with identical loading policies. Verify source refreshes succeed and V2 is populated. V1-only recovery or failed loads cannot establish the first V2 baseline.
3. Stop/drain legacy and shadow writers before enabling V2 replicas. Confirm the V2 baseline is recent and policy-compatible; require a successful source refresh if it is not. A slow writer has a one-minute lease; publication after expiry fails rather than extending ownership. Every replica still loads its own sources and may serve its local successful result.
4. Observe freshness health, failure counters and Redis errors. Do not delete V1 keys during the rollback window. V2 keys use `warden:{namespace}:snapshot:v2` and a separate owner-token lease; there is no implicit coordination with older writers.
5. To roll back, drain V2 writers and return to legacy (or shadow). V1 may be expired or stale because V2 does not update it. Refresh sources before relying on V1 again. Legacy diagnostic age resumes its recovery-time semantics; it cannot preserve source evidence that V1 never stored. On v1.6.0 with `enforce`, returning to legacy does not make unknown-age data readable: require a successful source load or deliberately select `observe`.

Changing storage mode does not change merge precedence, record validation, API authorization, scheduling or the empty-set default. Read enforcement is a separate setting with the environment-dependent defaults above. Neither timestamp is an upstream-signed time: Redis access and synchronized host clocks remain trusted. See [the storage protocol](REDIS_SNAPSHOT_PROTOCOL.md).

# Versioned Redis snapshot protocol

This storage API is additive. The existing user-array cache and application
selection are unchanged until an explicit migration mode is enabled.

Schema v2 stores effective users, their content digest, the input version/count,
original source family, source-success time, publication time and a publication
revision. Source success means an accepted source load, not upstream content age.
Recovery time is local to each process and is never persisted as source evidence.

Each dataset needs an explicit namespace and policy fingerprint. The envelope and
its lease share a Redis Cluster hash tag. Reads atomically distinguish a missing
key from a stored empty list. Unknown schema/policy, null users, invalid digests,
invalid timestamps and oversized payloads are rejected; errors do not include the
payload. A five-second future-clock allowance bounds accepted clock skew.

Publishing requires a Redis owner-token lease acquired before the source load.
The write atomically checks ownership; expired writers cannot publish or delete a
successor's lease. Leases are bounded, not automatically extended: slow work skips
shared publication after expiry and retries on a later refresh. Redis failures do
not fall back to process-local ownership.

Renewal compares the stored publication revision and changes only TTL. It never
rewrites users, source-success time or publication time. A replica holding an old
revision cannot renew or restore a newer publication, even when user content is
identical. A replica that kept its last known good set after a rejected refresh
publishes it only when Redis holds no readable publication with the same or a
later source-success time. Clock synchronization and trusted Redis writers remain deployment
requirements; this format is not a signed upstream revision protocol.

Protocol tests use miniredis with real Lua execution, including lease expiry,
replacement owners, stale renewal, empty lists, malformed envelopes and restart
recovery. Application migration and legacy compatibility are separate changes.

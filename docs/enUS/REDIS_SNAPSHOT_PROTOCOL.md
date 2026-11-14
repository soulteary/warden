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
payload. A five-second future-clock allowance bounds accepted clock skew. The
read decision under `SNAPSHOT_READ_POLICY=enforce` applies the same allowance, so
a publication the store accepts is not refused as future-dated by a replica that
restores it; a source time further ahead is rejected by both.

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
later source-success time. When the renewal finds no snapshot key at all, or a
value that is over the size limit or that Redis cannot decode as a JSON object
or array, the lease holder publishes the snapshot it holds again under the same
rule, with its original source-success time and a new revision; such a value
is not a publication any replica could restore. When the renewal finds a value
that decodes and carries a different revision, another writer may have
published since and the stored value is left untouched. Clock synchronization and trusted Redis writers remain deployment
requirements; this format is not a signed upstream revision protocol.

The renewal reads the stored revision from the beginning of the value. Every
envelope begins with `schema_version`, `dataset`, `policy` and `revision`, in
that order and without whitespace, followed by `users`. The script compares
that beginning with the one expected for its own dataset and policy, so it
does not decode the user list inside Redis, and nothing inside a user record
can take part in the comparison. A value with any other beginning is decoded
in full and classified as described above. The order of these leading members
is part of the stored format: a writer that changed it would still be renewed
correctly, but every renewal would decode the whole value again.

Protocol tests use miniredis with real Lua execution, including lease expiry,
replacement owners, stale renewal, empty lists, malformed envelopes and restart
recovery. Application migration and legacy compatibility are separate changes.

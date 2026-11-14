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
value that is not a publication of this dataset and policy, the lease holder
publishes the snapshot it holds again under the same rule, with its original
source-success time and a new revision; such a value is not a publication any
replica of this dataset and policy could restore. When the renewal finds a
publication of this dataset and policy that carries a different revision,
another writer may have published since and the stored value is left
untouched. Clock synchronization and trusted Redis writers remain deployment
requirements; this format is not a signed upstream revision protocol.

The renewal script decides what the stored value is before it compares
revisions, and returns one of these results:

| Result | Stored value | Renewal outcome | Lease holder |
| --- | --- | --- | --- |
| 0 | not examined: the lease is no longer held | `lease_lost` | writes nothing |
| 1 | a publication of this dataset and policy with the held revision | `renewed` | TTL extended |
| 2 | a publication of this dataset and policy with another revision, or a value that states a later `schema_version` | `revision_conflict` | leaves the value untouched |
| 3 | no key | `missing` | publishes again |
| 4 | longer than `MAX_ENVELOPE_JSON_SIZE` | `too_large` | publishes again |
| 5 | not an envelope | `invalid_envelope` | publishes again |
| 6 | an envelope of another dataset | `dataset_rejected` | publishes again |
| 7 | an envelope of another policy | `policy_rejected` | publishes again |

A value is a publication of this dataset and policy when either of these holds:

- it begins with the bytes every envelope of this dataset and policy begins
  with (described below), followed by 32 characters from `0-9a-f` and the start
  of the `users` member; or
- Redis decodes it to a JSON object whose `schema_version` is the number 2,
  whose `dataset` and `policy` are this store's, and whose `revision` is a
  string of 32 characters from `0-9a-f`.

A value that states a later schema version is one that begins with
`{"schema_version":` followed by an integer greater than 2 and `,` or `}`, or that Redis
decodes to an object whose `schema_version` is a number greater than 2. This
release cannot tell whether such a value is a publication, and a newer release
may be writing it under the same key during an upgrade, so it is treated like
another writer's publication and left untouched, whatever its other members
are. The first form is recognised without decoding and does not require the
rest of the value to be valid JSON.

Everything else is not an envelope (result 5): a value Redis cannot decode as
a JSON object or array, an array, `{}`, an object without `schema_version` or
with one that is lower than 2 or not a number, an object whose `dataset` or
`policy` is missing or not a string, and an object whose `revision` is
missing, not a string, or not 32 characters from `0-9a-f`. An object with
schema version 2 and a well-formed `dataset` and `policy` that differ from
this store's is reported as result 6 or 7 before its revision is examined.

Replicas that share a namespace but not a policy each find the other's
publication to be of another policy and replace it on their own cycle. Run
differing policies under different namespaces, as
[Redis snapshot migration](REDIS_SNAPSHOT_MIGRATION.md) requires.

These rules apply to renewal only. A replica that holds no revision, and any
replica after a successful source load, publishes without examining the stored
value's schema version, as before.

The renewal reads the stored revision from the beginning of the value. Every
envelope begins with `schema_version`, `dataset`, `policy` and `revision`, in
that order and without whitespace, followed by `users`. The script compares
that beginning with the one expected for its own dataset and policy, so it
does not decode the user list inside Redis, and nothing inside a user record
can take part in the comparison. A value with any other beginning, other than
one that states a later schema version, is decoded in full and classified as
described above. The order of these leading members
is part of the stored format: a writer that changed it would still be renewed
correctly, but every renewal would decode the whole value again.

Protocol tests use miniredis with real Lua execution, including lease expiry,
replacement owners, stale renewal, empty lists, malformed envelopes and restart
recovery. Application migration and legacy compatibility are separate changes.

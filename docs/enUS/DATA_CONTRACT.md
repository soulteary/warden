# Current data contract

This document records existing behavior before changes to snapshot storage or read
policy. It introduces no configuration, endpoint, or runtime change.

## Merging and removal

`ONLY_REMOTE` and `ONLY_LOCAL` select one source family. The other five modes
retain source-only records when both sources load successfully; `REMOTE_FIRST`
is not a replacement of the local list by the remote list. Use `ONLY_REMOTE`
when the remote list must be the sole authority.

The current **plaintext** loading path resolves records sharing a phone/mail key
as follows (observed with parser-kit v3.0.2):

| Mode | Record retained for a shared key |
| --- | --- |
| `DEFAULT` | Local |
| `REMOTE_FIRST` | Remote |
| `ONLY_REMOTE` | Remote |
| `ONLY_LOCAL` | Local |
| `LOCAL_FIRST` | Remote |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | Remote |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | Remote |

`DEFAULT` and both local-first modes differ from their documented precedence.
This baseline records their existing result rather than changing authorization
behavior. It does not generalize that result to encrypted loading, which uses a
separate merge path. Any correction to precedence needs a separate behavior-change
review and migration note.

`TestBaselinePlaintextMergeModeContract` in
`internal/loader/baseline_contract_test.go` checks this matrix and source-only
records. Existing `TestLoadWithResult_RemoteFailure_Modes` and strict-mode tests
cover refresh errors.

## Reads and health

`SNAPSHOT_MAX_AGE` affects health reporting, not data reads. In `ONLY_REMOTE` and
`REMOTE_FIRST`, an old snapshot can make health return HTTP 503 while authenticated
list and user queries still return cached data. Clients must not interpret a
successful query as proof that its source was recently checked.

A successfully loaded empty list is valid data: list queries return `[]`, missing
users return 404, and fresh empty snapshots can be healthy. This differs from a
process that has never loaded a known-good set. `TestBaselineSnapshotReadContract`
in `baseline_contract_test.go` checks all health and data-route aliases for stale
non-empty and fresh empty snapshots.

## Validation and shared-cache recovery

Identity conflicts reject a whole refresh and keep the previous valid set.
Per-record format validation can drop records. If non-empty input is entirely
rejected, `EMPTY_RULESET_POLICY` selects whether to apply the empty set or retain
the previous set. An actual empty source remains valid under either policy.
`empty_ruleset_policy_test.go` covers these cases, including Redis publication
and recovery. See [Configuration](CONFIGURATION.md#empty-ruleset-policy).

Redis stores the effective users without the original source timestamp. Restoring
them assigns the current replica's adoption time to `LoadedAt`; this is not the
time the upstream list was last checked. Background refresh failure preserves the
previous snapshot timestamp. Shared-cache TTL is not a source freshness guarantee.

## Caller caches and sessions

The Go SDK can cache list responses (`GetUsers`); its default options use a
five-minute TTL. A cached list can be returned without contacting Warden. User
status must still be checked by the caller. Warden does not invalidate already
issued gateway sessions: their cache and revalidation rules determine how quickly
a removal reaches existing sessions. There is no end-to-end revocation deadline
enforced by Warden alone.

## Loading failure matrix

[Empty-list and failure loading contract](LOAD_FAILURE_CONTRACT.md) records
plaintext and encrypted outcomes, including their degraded-state differences.

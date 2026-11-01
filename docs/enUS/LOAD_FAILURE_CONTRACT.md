# Empty-list and failure loading contract

This records the existing `RulesLoader.LoadWithResult` result for identical local
and remote data delivered as HTTP plaintext or a genuine encrypted v2 envelope.
It does not change merge modes, cache publication, or health reporting.

## Successful empty data

The tests use a single local file and one remote API. Their successful outcomes
agree between transports for these fixtures:

- Both sources return `[]`: success with no users in all seven modes.
- Remote returns `[]`, local has a user: `ONLY_REMOTE` returns an empty set;
  all other modes return the local user. In merge modes, a remote empty array
  does not erase a local grant.
- Local returns `[]`, remote has a user: `ONLY_LOCAL` returns an empty set;
  all other modes return the remote user.
- Local JSON is malformed, remote has a user: `ONLY_LOCAL` fails;
  all other modes return the remote user without a degraded flag.

`ONLY_LOCAL` is invoked with an empty remote URL, matching the application. Its
row does not claim that the raw loader ignores an explicitly supplied remote URL.

## Remote failure with valid, non-empty local data

The matrix below applies both to an HTTP 404 remote response and to malformed
remote user JSON. In the encrypted JSON case, a valid authenticated v2 envelope
contains malformed user JSON: the error is in decoding users, not in authentication.

| Mode | Plaintext result | Encrypted result |
| --- | --- | --- |
| `DEFAULT` | Local user, not degraded | Local user, degraded |
| `REMOTE_FIRST` | Error | Error |
| `ONLY_REMOTE` | Error | Error |
| `ONLY_LOCAL` | Local user, not degraded | Local user, not degraded |
| `LOCAL_FIRST` | Local user, not degraded | Local user, degraded |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | Local user, degraded | Local user, degraded |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | Local user, not degraded | Local user, degraded |

Every degraded result has reason `remote_failed`. These are observable differences
in fallback diagnostics, even when the returned grants agree. The tests preserve
both outcomes rather than silently changing the plaintext path to match decryption.

## Remote failure with an empty local list

For HTTP 404 with local `[]`, both transports fail in every mode except
`ONLY_LOCAL`, which loads its empty local set successfully without a remote URL.
An empty fallback is not accepted as a successful remote refresh. A failed result
contains no users or version and must not be consumed as a new empty rule set.
The application's handling of the previous snapshot is covered separately.

## Test scope

`TestTransportFallbackContract` in
`internal/loader/transport_fallback_contract_test.go` checks seven scenarios across
seven modes and both transports. It asserts complete returned user records, errors,
degraded flags and reasons, and successful content versions. It uses real HTTP
responses and `remote.Encrypt`; no merge function is mocked. It does not require
transport-specific errors, source labels, timestamps, or result ordering to match.

The scope is one file and one remote API, v2 encryption, HTTP 404, and malformed
user JSON. Directory ordering, zero-byte payloads, timeouts, tampered envelopes,
legacy encryption, and cache-layer record rejection are not generalized from this
matrix; those have separate tests or need their own contract review.

See [Current data contract](DATA_CONTRACT.md) for snapshot and caller-cache limits.

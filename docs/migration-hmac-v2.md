# Migration: HMAC v1 to v2

Warden 1.0 uses HMAC v2 as the default request-signing contract. Version 2
binds every security-sensitive selector, including the Key ID, and adds a
single-use nonce so a captured request cannot be replayed inside the timestamp
window.

## Canonical request

Join these UTF-8 fields with a single line feed (`\n`) and no trailing line
feed:

```text
METHOD
ESCAPED_PATH_AND_QUERY
KEY_ID
TIMESTAMP
NONCE
SHA256_HEX(BODY)
```

Compute the lowercase hexadecimal signature as:

```text
HEX(HMAC_SHA256(secret, canonical_v2))
```

The request must carry:

| Header | Value |
|---|---|
| `X-Signature-Version` | `v2` |
| `X-Key-Id` | The Key ID used in the canonical request |
| `X-Timestamp` | Unix time in seconds |
| `X-Nonce` | A fresh 128-bit hexadecimal nonce for every attempt |
| `X-Signature` | The computed lowercase hexadecimal signature |

Retries must be signed again with a new nonce. Reusing the same nonce is a
replay and returns `401 Unauthorized`.

## Rollout

1. Upgrade Warden and callers together so both use the Key-ID-bound v2
   canonical request. The bundled Go SDK emits the correct headers.
2. If an older v1 caller must remain temporarily, set
   `WARDEN_HMAC_ALLOW_V1=true`. This is an explicit compatibility exception;
   the default is `false`.
3. Watch `deprecation_total{feature="hmac_v1"}` until it remains at zero.
4. Remove `WARDEN_HMAC_ALLOW_V1` (or set it to `false`). Invalid boolean values
   fail configuration validation.

During key rotation, each Key ID may select a different secret. Because Key ID
is signed in v2, changing only `X-Key-Id` invalidates the signature even if two
rotation entries temporarily share the same secret.

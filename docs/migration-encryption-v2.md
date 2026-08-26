# Migration: Remote Encryption Legacy → Envelope v2

Warden's remote data source can be encrypted. Older deployments used an
unauthenticated AES-CTR scheme ("legacy"). Warden now ships an authenticated,
versioned JSON envelope ("v2", AES-256-GCM + RSA-OAEP-SHA256) that provides
integrity as well as confidentiality and removes the silent plaintext downgrade
that the legacy path allowed.

This guide describes a safe, two-phase migration with no downtime and an easy
rollback.

## Why migrate

- **Integrity.** Legacy AES-CTR has no authentication tag; a tampered or
  truncated body could be accepted. v2 uses AES-256-GCM, so any modification
  fails decryption with a typed error.
- **No silent downgrade.** The legacy path could return ciphertext as plaintext
  when the `Content-Type` did not match. v2 refuses to fall back to plaintext.
- **Key-size agility.** v2 derives the RSA key size dynamically instead of a
  hard-coded 2048-bit assumption.

## Configuration reference

| Environment variable | Values | Meaning |
|----------------------|--------|---------|
| `REMOTE_DECRYPT_ENABLED` | `true` / `false` | Enable remote-response decryption at all. |
| `REMOTE_ENCRYPTION_FORMAT` | `auto` / `v2` / `legacy` | Which envelope(s) to accept. `auto` accepts both during migration; `v2` accepts only v2; `legacy` accepts only the old format. Default: `auto`. |
| `REMOTE_ENCRYPTION_REQUIRED` | `true` / `false` | When `true`, a plaintext (undecryptable) body is rejected — "fail closed". Recommended `true` in production. |
| `REMOTE_RSA_PRIVATE_KEY_FILE` | path | RSA private key PEM file used to unwrap the content key. |
| `REMOTE_RSA_PRIVATE_KEY` | inline PEM | Inline RSA private key (used when `_FILE` is unset). Prefer a secret manager: `<set-via-secret-manager>`. |

> Never commit real keys. Use a placeholder such as `<set-via-secret-manager>`
> in examples and inject the real value at runtime.

## Two-phase migration

The producer (whatever generates the remote `data.json`) and the consumer
(Warden) must not change format at the same instant. Use `auto` as the bridge.

### Phase 1 — Accept both, keep producing legacy

1. Deploy Warden with:
   - `REMOTE_DECRYPT_ENABLED=true`
   - `REMOTE_ENCRYPTION_FORMAT=auto`
   - `REMOTE_ENCRYPTION_REQUIRED=true` (recommended)
2. Confirm the service is healthy and serving data.
3. Warden emits a deprecation signal whenever it decrypts a legacy body:
   - Metric: `deprecation_total{feature="encryption_legacy"}`
   - A deduplicated warning log: *"remote: legacy unauthenticated encryption
     format in use; migrate to envelope v2"*
   Use the metric to confirm the remaining volume of legacy traffic.

### Phase 2 — Switch the producer to v2, then pin Warden to v2

1. Update the producer to emit the v2 envelope.
2. Watch `deprecation_total{feature="encryption_legacy"}` fall to zero and stay
   there for a full refresh cycle.
3. Pin Warden to v2 only:
   - `REMOTE_ENCRYPTION_FORMAT=v2`
4. Redeploy. Legacy bodies are now rejected.

## Rollback

- If v2 output has a problem, set the producer back to legacy and, if Warden was
  already pinned, set `REMOTE_ENCRYPTION_FORMAT=auto` (or `legacy`) again.
- Because Phase 1 accepts both formats, rolling *back* to Phase 1 is always safe.

## Verification checklist

- [ ] `REMOTE_ENCRYPTION_REQUIRED=true` in production (no plaintext downgrade).
- [ ] `deprecation_total{feature="encryption_legacy"}` is `0` before pinning `v2`.
- [ ] Health endpoint reports the `snapshot` check as healthy (not degraded) —
      a degraded snapshot means Warden is serving a last-known-good copy after a
      refresh failure; investigate before pinning.
- [ ] A deliberately tampered v2 body is rejected (integrity works).

## Related signals

- `warden_snapshot_age_seconds` — age of the currently served snapshot.
- `warden_refresh_failures_total{reason}` — refresh failures by reason code.
- `warden_remote_fallback_total{mode,reason}` — fallbacks to local/last-known-good.

See also: [Configuration migration](migration-config.md).

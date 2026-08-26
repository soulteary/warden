# Migration: Configuration (`MODE` → `ENVIRONMENT` + `MERGE_MODE`)

Historically a single `MODE` variable controlled two unrelated things:

1. **How data sources are merged** (local vs remote precedence, fallback), and
2. **Whether the deployment is "production"** (which toggles security-hardening
   defaults such as hiding error detail).

Overloading one variable for both made "production" ambiguous and coupled
security posture to a data-merge setting. Warden now separates them into two
independent variables. `MODE` is still accepted as a **deprecated alias for the
merge mode only** — it no longer decides production hardening.

## New variables

| Variable | Values | Purpose | Default |
|----------|--------|---------|---------|
| `ENVIRONMENT` | `development` / `test` / `production` | Deployment environment. Drives security hardening and the production auth requirement. | `development` |
| `MERGE_MODE` | `DEFAULT`, `REMOTE_FIRST`, `ONLY_REMOTE`, `ONLY_LOCAL`, `LOCAL_FIRST`, `REMOTE_FIRST_ALLOW_REMOTE_FAILED`, `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | How local and remote data are combined and how remote failures fall back. | `DEFAULT` |

### Precedence

For each concept, the highest-priority source that is set wins:

```
CLI flag  >  new env var  >  legacy MODE  >  config file  >  built-in default
```

If both a new variable and legacy `MODE` are set and they conflict, the **new
variable wins** and Warden emits a deduplicated deprecation warning plus the
metric `deprecation_total{feature="mode_legacy_env"}`.

## Migration steps

1. **Inventory** current `MODE` usage. If `MODE` was one of the merge values
   (e.g. `REMOTE_FIRST`), that is a *merge mode*. If it was `production`/`prod`,
   that was being used as an *environment*.
2. **Split** the value:
   - Set `MERGE_MODE=<your merge value>` (default `DEFAULT`).
   - Set `ENVIRONMENT=production` for production deployments (or
     `development` / `test`).
3. **Remove** `MODE` once both new variables are in place. Watch
   `deprecation_total{feature="mode_legacy_env"}` drop to zero to confirm no
   component still relies on the legacy alias.

### Example

Before:

```env
MODE=production
```

After:

```env
ENVIRONMENT=production
MERGE_MODE=DEFAULT
```

## Production auth requirement (new, fail-closed)

When `ENVIRONMENT=production`, Warden now **refuses to start** unless at least
one service-authentication mechanism is configured. This prevents accidentally
shipping an unauthenticated production service.

Configure **at least one** of:

- **API key**: `API_KEY=<set-via-secret-manager>`
- **HMAC**: `WARDEN_HMAC_KEYS='{"<key-id>":"<set-via-secret-manager>"}'`
- **mTLS**: `WARDEN_TLS_CA=<path>` **and** `WARDEN_TLS_REQUIRE_CLIENT_CERT=true`

Non-production environments (`development`, `test`) remain permissive.

### Metrics / health exposure

`/metrics` is exposed anonymously by default and publishes only
low-cardinality, non-sensitive series. To require authentication for `/metrics`,
set:

```env
WARDEN_METRICS_REQUIRE_AUTH=true
```

Full user-rules endpoints always require authentication regardless of this
setting.

## Related configuration hardened in this release

### Remote encryption (fail-closed)

| Variable | Values | Purpose |
|----------|--------|---------|
| `REMOTE_ENCRYPTION_REQUIRED` | `true` / `false` | Reject plaintext remote bodies. Recommended `true` in production. |
| `REMOTE_ENCRYPTION_FORMAT` | `auto` / `v2` / `legacy` | Accepted envelope format(s). See [encryption migration](migration-encryption-v2.md). |

### Identity integrity

| Variable | Values | Purpose |
|----------|--------|---------|
| `REQUIRE_EXPLICIT_USER_ID` | `true` / `false` | When `true`, records without an explicit `user_id` are rejected instead of silently deriving one. |
| `USER_ID_STRATEGY` | `legacy` / `sha256-128` | ID derivation when no explicit `user_id` is present. `legacy` keeps the historical 16-char digest (emits a migration warning); `sha256-128` uses a domain-separated SHA-256 128-bit derivation. Contact/identifier changes never silently change an ID. |

## HMAC signing tolerance

| Variable | Values | Purpose |
|----------|--------|---------|
| `WARDEN_HMAC_KEYS` | JSON `{"key-id":"secret"}` | HMAC keys (use `<set-via-secret-manager>` for the secret). |
| `WARDEN_HMAC_TIMESTAMP_TOLERANCE` | seconds | Allowed clock skew for signed requests (bounded upper limit enforced). Default `60`. |

Warden accepts both HMAC v1 and the authenticated v2 signature (per-request
nonce + replay rejection) during migration. v1 usage is surfaced via
`deprecation_total{feature="hmac_v1"}`; replays rejected by v2 increment
`hmac_replay_rejected_total`. The in-memory replay guard is single-node only —
in a multi-replica deployment, replay protection requires a shared store.

See also: [Encryption migration](migration-encryption-v2.md).

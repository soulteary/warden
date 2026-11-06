# Service permissions

`WARDEN_SERVICE_POLICIES` optionally restricts authenticated HMAC keys to named
operations. Existing installations retain their current permissions when it is
unset. Setting a policy requires HMAC v2; API-key and client-certificate
fallbacks are rejected at startup. TLS server authentication remains supported.

```json
{
  "gateway": {"actions": ["directory:lookup"]},
  "directory-admin": {"actions": ["directory:lookup", "directory:list", "admin:loglevel"]},
  "monitor": {"actions": ["metrics:read"]}
}
```

Every policy key must exist in `WARDEN_HMAC_KEYS`. All configured key IDs must
have distinct secrets, including keys without a policy. Reusing a secret would
allow its holder to sign requests as another key ID. Unknown fields, empty policies,
unknown actions, and duplicate actions fail validation. Unmapped keys are denied.
`directory:lookup` covers `/user`, `/v1/user`, and `/v1/lookup`;
`directory:list` covers `/`, `/data.json`, and `/v1/users`.
`admin:loglevel` is separate from directory access. `metrics:read` applies when
metrics authentication is enabled; keep `WARDEN_METRICS_REQUIRE_AUTH=true` for
private scraping. A scraper using HMAC must sign each request through a trusted
adapter. Network restrictions remain necessary if metrics are explicitly public.

Deploy keys and policies together, then test each consumer before removing its
old credential. Do not distribute an administrator key to ordinary consumers.
In this mode a key ID, which HMAC v2 binds into the signature, identifies the
caller; unverified service-name headers grant no permissions.

These permissions separate operations, not tenants or users. A lookup credential
can query any record in this directory. Use separate instances and Redis
credentials for mutually untrusted directories. Redis logical databases alone
are not an authorization boundary.

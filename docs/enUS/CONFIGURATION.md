# Configuration

> 🌐 **Language / 语言**: [English](CONFIGURATION.md) | [中文](../zhCN/CONFIGURATION.md) | [Français](../frFR/CONFIGURATION.md) | [Italiano](../itIT/CONFIGURATION.md) | [日本語](../jaJP/CONFIGURATION.md) | [Deutsch](../deDE/CONFIGURATION.md) | [한국어](../koKR/CONFIGURATION.md)

This document provides detailed information about Warden's configuration options, including running modes, configuration file formats, environment variables, etc.

**Configuration priority**: Command line arguments > Environment variables > Configuration file (YAML) > Defaults.

For a **full option table** (YAML paths, env vars, defaults, validation rules), see [zhCN CONFIGURATION](../zhCN/CONFIGURATION.md). Summary:

| Category | YAML / Env | Notes |
|----------|------------|--------|
| Server | `server.*` / `PORT` | port, read_timeout, write_timeout, shutdown_timeout, idle_timeout, max_header_bytes |
| Redis | `redis.*` / `REDIS`, `REDIS_PASSWORD`, `REDIS_PASSWORD_FILE`, `REDIS_ENABLED` | addr, password, password_file, db; Redis enabled default `true` (except ONLY_LOCAL without REDIS) |
| Redis transport | — / `REDIS_USERNAME`, `REDIS_TLS_ENABLED`, `REDIS_TLS_CA_FILE`, `REDIS_TLS_SERVER_NAME` | **Env only**. ACL username (default user when empty) and TLS (default off) |
| Cache | `cache.ttl`, `cache.update_interval` | no env overrides; update_interval default 5s |
| Rate limit | `rate_limit.rate`, `rate_limit.authenticated_rate`, `rate_limit.window` / `RATE_LIMIT`, `RATE_LIMIT_AUTHENTICATED`, `RATE_LIMIT_WINDOW` | `rate`: per client IP for requests that did not authenticate, default 60; `authenticated_rate`: per credential, default 600; both per 1m window; environment overrides YAML |
| HTTP client | `http.*` / `HTTP_TIMEOUT`, `HTTP_MAX_IDLE_CONNS`, `HTTP_INSECURE_TLS` | timeout, max_idle_conns, insecure_tls, max_retries, retry_delay |
| Remote | `remote.*` / `CONFIG`, `KEY`, `MERGE_MODE`, `REMOTE_DECRYPT_ENABLED`, `REMOTE_RSA_PRIVATE_KEY_FILE`, `REMOTE_RSA_PRIVATE_KEY` | url, key, mode, decrypt_enabled, rsa_private_key_file |
| Task | `task.interval` / `INTERVAL` / `--interval` | Explicit CLI value overrides the environment, then YAML, then the default |
| App | `app.*` / `API_KEY`, `DATA_FILE`, `DATA_DIR`, `RESPONSE_FIELDS` | mode, api_key, data_file, data_dir, response_fields |
| Tracing | `tracing.enabled`, `tracing.endpoint` / `OTLP_ENABLED`, `OTLP_ENDPOINT` | The selected YAML file supplies tracing settings; `OTLP_ENABLED` and `OTLP_ENDPOINT` override them |
| Service auth | — / `WARDEN_HMAC_KEYS`, `WARDEN_HMAC_TIMESTAMP_TOLERANCE`, `WARDEN_TLS_*` | **Env only** (no YAML keys) |
| Health | — / `SNAPSHOT_MAX_AGE` | Maximum accepted snapshot age; Go duration, default `max(30s, 3 × task interval)` |
| Data policy | — / `EMPTY_RULESET_POLICY` | What a refresh does when every loaded record is rejected; `consistency-first` (default) or `availability-first`, see [Empty ruleset policy](#empty-ruleset-policy) |

## YAML schema and application wiring

Parsing a YAML key does not guarantee that the current binary uses it. The main
startup path projects configuration through `Config.ToCmdConfig` and `cmd.Config`.
The following schema fields are not wired into that application path:

- `server.read_timeout`, `server.write_timeout`, `server.shutdown_timeout`,
  `server.idle_timeout`, `server.max_header_bytes` (only `server.port` is projected).
- `redis.db`, `cache.ttl`, `cache.update_interval`, `http.max_retries`,
  `http.retry_delay`.

The tables/examples below include schema fields for reference, not promises that
changing those fields changes the running service. Current runtime values come
from application defaults: Redis snapshot/cache TTL is one hour; the parser-kit loader retry policy uses three retries with a one-second base
delay. Refresh scheduling uses `task.interval` / `INTERVAL` / `--interval`, not
`cache.update_interval`; valid intervals below five seconds are clamped to five
seconds by the application. SDK CacheTTL is a separate client option.

When the selected configuration file sets any of the fields listed above, Warden
logs one warning at startup that names them. It also logs a warning when the
refresh interval is below five seconds.

See `internal/config/config.go: ToCmdConfig`, `main.go: NewApp`,
`internal/cache/redis_cache.go`, and
`internal/loader/loader.go`. Wiring additional fields requires a separate behavior
change, not a documentation correction.


## Running Mode (MERGE_MODE)

The system supports 7 data merging modes, selected with `MERGE_MODE` (`MODE` is deprecated):

| Mode | Description | Use Case |
|------|-------------|----------|
| `DEFAULT` | Historical remote-first, tolerant behavior | Backward compatibility for deployments that never selected a mode |
| `REMOTE_FIRST` | Remote wins when loading succeeds; a remote error is fatal and the last-known-good snapshot is retained | Strict remote-authoritative deployments |
| `ONLY_REMOTE` | Use only remote data source | Fully dependent on remote configuration |
| `ONLY_LOCAL` | Use only local configuration file, **Redis disabled by default** (will be enabled if `REDIS` address is explicitly set or `REDIS_ENABLED=true`) | Offline environment or test environment |
| `LOCAL_FIRST` | Local-first, use remote data to supplement when local data doesn't exist | Local configuration as primary, remote as secondary |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | Remote-first, allow fallback to local when remote fails | High availability scenarios |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | Local-first, allow fallback to remote when local fails | Hybrid mode |

**No remote source**: `CONFIG` / `remote.url` has no default. When it is empty there is no remote source: `DEFAULT`, `LOCAL_FIRST` and the two `*_ALLOW_REMOTE_FAILED` modes load only the local data, make no remote request and are not reported as degraded, and the production rules for remote sources (HTTPS, encryption) do not apply. `ONLY_REMOTE` and `REMOTE_FIRST` refuse to start without a remote source.

### Snapshot freshness and remote failures

`REMOTE_FIRST` and `ONLY_REMOTE` are strict modes. If a scheduled remote refresh
fails, Warden keeps serving the last-known-good in-memory snapshot, records the
refresh failure, and does not advance the snapshot's `loaded_at`. The health
endpoint returns HTTP 503 after the snapshot exceeds `SNAPSHOT_MAX_AGE` (default
`max(30s, 3 × task interval)`). Set the value with a Go duration such as `2m`.

`REMOTE_FIRST_ALLOW_REMOTE_FAILED` explicitly falls back to validated local data,
marks the snapshot `degraded`, and remains serviceable with HTTP 200.
`LOCAL_FIRST` and `LOCAL_FIRST_ALLOW_REMOTE_FAILED` can remain healthy when their
local primary succeeds even if the remote supplement is unavailable. `DEFAULT`
retains the historical tolerant behavior for compatibility (including this
plaintext local-success behavior); select an explicit mode for new production
deployments. Encrypted remote failures that use a local fallback are reported as
degraded in every tolerant mode.

In multi-replica deployments every replica refreshes its process-local cache and
snapshot. A distributed Redis lock elects only the writer of the shared Redis
cache, so non-writers still advance their own snapshot freshness.

### Empty ruleset policy

Warden filters records in two layers. **Identity validation** (duplicate
identifiers, missing `user_id`) runs over the whole set: one bad record fails the
entire refresh and the last known good data is kept. **Per-record format
validation** (phone and mail format) lives in the cache: it silently drops the
records it rejects and keeps the rest.

Together they produce one peculiar state: **a load succeeds, returns records, and
not a single one of them reaches the effective rule set.** The usual cause is an
upstream format change — `13800138000` becoming `138-0013-8000`, say. The
upstream is perfectly healthy, the load reports success, and every record is
discarded one layer down.

From inside Warden that state is **ambiguous**, and the two readings are
indistinguishable from the data alone:

| Reading | Correct response |
|---------|------------------|
| "the upstream revoked everyone" | the empty set is correct and must take effect now |
| "the upstream changed format" | the empty set is an artifact and must not take effect |

Since Warden cannot tell them apart, the operator chooses, via the
`EMPTY_RULESET_POLICY` environment variable:

| Value | Behavior | When to choose it |
|-------|----------|-------------------|
| `consistency-first` (default) | Apply and publish the empty set: the in-memory cache empties, the shared Redis cache is updated to match, and the refresh counts as successful | The upstream is the single source of truth and **divergence is worse than unavailability**: a legitimate mass revocation propagates immediately, and an upstream format regression surfaces at once as "everything is denied" instead of hiding behind stale data |
| `availability-first` | Treat it as a failed refresh: keep the last known good rule set in memory; renew the shared cache only when this replica has a proven known-good snapshot, otherwise retry Redis or skip the write; advance the failure counter, let snapshot age keep growing, and report degraded health | **The availability loss outweighs the risk of wrongly allowing**: with an unstable data source and replicas that restart often, one upstream glitch would otherwise empty every replica and leave restarted ones with nothing to bootstrap from |

The default preserves the historical behavior. It is also the right bias for an
allow list: continuing to admit users the source of truth no longer contains is a
**security** failure, while denying users is an **availability** failure — loud,
obvious, and immediately noticed.

#### What the policy does not change

- **A load that genuinely returns zero records** is not ambiguous and always
  takes effect under both policies. Otherwise revoking every user would become
  impossible.
- **Startup** (`loadInitialData`) normally bootstraps from the shared Redis cache
  first, but one transient read failure does not mean its last-known-good set has
  disappeared. If the subsequent upstream/local load is non-empty but every record
  fails format validation, availability-first retries Redis after acquiring the
  writer lock. A recovered cache bootstraps the replica; if it remains unreadable,
  this replica stays empty but never writes an empty replacement over data that may
  still exist in Redis.

  `MERGE_MODE=ONLY_LOCAL` skips the normal Redis-first read, so it enters the same
  guarded flow directly. consistency-first keeps its historical startup behavior
  on every path: apply and publish the result after per-record validation.
- **A successful Redis bootstrap** creates a degraded snapshot with `source=redis`,
  load time, and content version. Strict remote modes can therefore measure
  `SNAPSHOT_MAX_AGE` from adoption instead of returning 503 immediately because the
  source is `none`. If startup never obtains valid data, the first background refresh
  retries Redis and still skips publishing the process's zero-value empty cache if the
  retry fails. If a legacy non-empty Redis payload is reduced to zero by current format
  validation, the configured policy still applies: consistency-first adopts the effective
  empty set, while availability-first rejects it as a bootstrap baseline and continues to
  the configured sources.
- **A genuinely empty successful startup load** is applied, snapshotted, and published
  to Redis immediately; a mass revocation does not wait for the first background cycle.
  On restart, Redis `Get` returning an empty slice is followed by `Exists` to distinguish
  a stored valid empty set from a missing key. The former becomes a known-good
  `source=redis` snapshot and preempts possibly stale upstream/local fallbacks; only the
  latter continues to other sources. Health checks use the same snapshot provenance to
  distinguish a legitimate empty set from data that has never loaded. Because `Get` and
  `Exists` are two round-trips taken before writer election, the value is re-read after
  `Exists` confirms presence: otherwise a peer publishing a rule set between the two calls
  would leave this replica pairing the pre-write empty slice with a post-write existence and
  serving an empty allow list until the next refresh.
- **Identity validation failures** (conflicts, missing `user_id`) are unrelated to
  this policy: under both values they keep the last known good data and record a
  refresh failure. If a set has both identity conflicts and per-record format
  errors, the identity-integrity failure takes precedence and is not classified as
  `all_records_rejected`.

#### Observability

| Policy | Log | `warden_refresh_failures_total{reason}` |
|--------|-----|------------------------------------------|
| `consistency-first` | `WARN` with `loaded_count` / `policy` | not counted (the refresh is successful) |
| `availability-first` | `WARN` with `loaded_count` / `kept_count` / `has_known_good` / `consecutive_failures` | `reason="all_records_rejected"` |

Under availability-first the snapshot version is deliberately not advanced, so the
next cycle still sees the broken load as a change and retries; once the upstream
recovers, the first successful refresh clears the failure counter and applies the
new data normally. Meanwhile snapshot age keeps growing and the health endpoint
returns 503 past `SNAPSHOT_MAX_AGE` — "still serving, but the data is stale" stays
visible rather than being silently papered over.

### Configuration Methods

You can set the running mode in the following ways:

**Command Line Arguments**:
```bash
go run . --mode DEFAULT
```

**Environment Variables**:
```bash
export MERGE_MODE=DEFAULT
# MODE remains a deprecated compatibility alias.
```

**Configuration File**:
```yaml
remote:
  mode: "DEFAULT"
# or
app:
  mode: "DEFAULT"
```

## Configuration File Format

### Local User Data File (`data.json`)

Local user data file `data.json` format (refer to `data.example.json`):

**Minimal format** (required fields only):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**Complete format** (with all optional fields):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write", "admin"],
        "role": "admin"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com",
        "status": "active",
        "scope": ["read"],
        "role": "user"
    }
]
```

**Field descriptions**:
- `phone` (required): User phone number
- `mail` (required): User email address
- `user_id` (optional): User unique identifier, auto-generated based on phone or mail if not provided
- `status` (optional): User status; omitted values fail closed as `"inactive"`. Set `"active"` explicitly to allow access.
- `scope` (optional): User permission scope array, defaults to empty array
- `role` (optional): User role, defaults to empty string

### Application Configuration File (`config.yaml`)

Supports YAML format configuration files, specified via the `--config-file` parameter:

```yaml
server:
  port: "8081"
  read_timeout: 5s
  write_timeout: 5s
  shutdown_timeout: 5s
  max_header_bytes: 1048576  # 1MB
  idle_timeout: 120s

redis:
  addr: "localhost:6379"
  password: ""  # Recommend using environment variable REDIS_PASSWORD or REDIS_PASSWORD_FILE
  password_file: ""  # Password file path (higher priority than password)
  db: 0

cache:
  ttl: 3600s
  update_interval: 5s

rate_limit:
  rate: 60                 # Requests per window, per client IP, that did not authenticate
  authenticated_rate: 600  # Requests per window, per credential
  window: 1m

http:
  timeout: 5s
  max_idle_conns: 20
  insecure_tls: false  # Development only
  max_retries: 3
  retry_delay: 1s

remote:
  url: ""  # Remote source URL; empty means no remote source
  key: ""
  mode: "DEFAULT"
  decrypt_enabled: false       # RSA decrypt remote response (use with rsa_private_key_file or REMOTE_RSA_PRIVATE_KEY)
  rsa_private_key_file: ""    # Path to PEM file (or use env REMOTE_RSA_PRIVATE_KEY for inline PEM)

task:
  interval: 5s

app:
  mode: "DEFAULT"  # Data merge mode; production policy is selected by ENVIRONMENT
  api_key: ""      # Recommend env API_KEY
  data_file: "./data.json"
  data_dir: ""     # Optional: merge all *.json in directory (can be used with data_file)
  response_fields: []  # Optional: API response field whitelist; empty = all fields

tracing:
  enabled: false
  endpoint: ""     # e.g. "http://localhost:4318"
```

**Configuration priority**: Command line arguments > Environment variables > Configuration file > Default values.

**Configuration file selection**: `--config-file` takes precedence over `CONFIG_FILE`. The selected file supplies both the main and tracing settings; environment variables can override YAML values. An explicitly selected file must exist and parse successfully. Without a file, `OTLP_ENABLED` + `OTLP_ENDPOINT` can enable tracing directly.

Refer to example file: [config.example.yaml](../../config.example.yaml).

## Command Line Arguments

```bash
go run . \
  --port 8081 \                    # Web service port (default: 8081)
  --redis localhost:6379 \         # Redis address (default: localhost:6379)
  --redis-password "password" \    # Redis password (optional; visible in the process list, prefer REDIS_PASSWORD or REDIS_PASSWORD_FILE)
  --redis-enabled=true \           # Enable/disable Redis (default: true)
  --config http://example.com/api \ # Remote configuration URL
  --key "Bearer token" \           # Remote configuration authentication header (visible in the process list, prefer KEY)
  --interval 5 \                   # Scheduled task interval (seconds, default: 5)
  --mode DEFAULT \                 # Running mode (see description above)
  --http-timeout 5 \               # HTTP request timeout (seconds, default: 5)
  --http-max-idle-conns 20 \      # Idle-connection pool of the remote-source client (default: 20)
  --http-insecure-tls \           # Skip TLS certificate verification (development only)
  --api-key "your-secret-api-key" \ # API Key for authentication (optional; visible in the process list, prefer API_KEY)
  --config-file config.yaml        # Configuration file path (supports YAML format)
```

**Notes**:
- Configuration file support: You can use the `--config-file` parameter to specify a YAML format configuration file
- Secrets on the command line: values passed with `--api-key`, `--key` and `--redis-password` are visible to other local users in the process list (`ps`, `/proc/<pid>/cmdline`) and may be saved in shell history. Use the environment variables `API_KEY`, `KEY` and `REDIS_PASSWORD`, or `REDIS_PASSWORD_FILE` for the Redis password. The flags remain supported
- TLS certificate verification: `--http-insecure-tls` is for development environments only, should not be used in production

## Environment Variables

Supports configuration via environment variables, with lower priority than command line arguments. For the full option table (including validation rules), see [zhCN CONFIGURATION](../zhCN/CONFIGURATION.md).

**Boolean values**: every boolean variable (`REDIS_ENABLED`, `REDIS_TLS_ENABLED`, `HTTP_INSECURE_TLS`, `REMOTE_DECRYPT_ENABLED`, `REMOTE_ENCRYPTION_REQUIRED`, `REQUIRE_EXPLICIT_USER_ID`, `WARDEN_TLS_REQUIRE_CLIENT_CERT`, `WARDEN_HMAC_ALLOW_V1`, `WARDEN_METRICS_REQUIRE_AUTH`, `OTLP_ENABLED`) accepts `true`/`false`, `1`/`0`, `yes`/`no`, `on`/`off` and `t`/`f`, in any letter case. Any other non-empty value stops startup with a message that names the variable.

**Numeric values**: `PORT` (1 to 65535), `INTERVAL` (seconds, at least 1), `HTTP_MAX_IDLE_CONNS` (0 or more; 0 selects the default) and `WARDEN_HMAC_TIMESTAMP_TOLERANCE` (seconds, at least 1) must be integers. `HTTP_TIMEOUT` is a whole number of seconds or a duration such as `30s` or `1m30s`, from 1 second to 1 hour; the same upper bound applies to `--http-timeout` and `http.timeout`. A value that does not meet these rules stops startup and the error names the variable. An empty variable is unset.

```bash
export PORT=8081
export REDIS=localhost:6379
export REDIS_PASSWORD="password"        # Redis password (optional)
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Redis password file path (optional; priority: REDIS_PASSWORD > REDIS_PASSWORD_FILE > config)
export REDIS_USERNAME=                  # Redis ACL username (optional; empty = default user)
export REDIS_TLS_ENABLED=false          # Connect to Redis over TLS (default: false)
export REDIS_TLS_CA_FILE=               # CA bundle for the Redis server certificate (optional; system roots when empty)
export REDIS_TLS_SERVER_NAME=           # Expected certificate name (optional; defaults to the Redis host)
export REDIS_ENABLED=true               # Enable/disable Redis (optional, default: true)
                                        # Note: In ONLY_LOCAL mode, default is false
                                        #       But if REDIS address is explicitly set, Redis will be enabled automatically
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5
export MERGE_MODE=DEFAULT
export DATA_FILE=./data.json          # Local user data file path
export DATA_DIR=                      # Optional: directory to merge all *.json (can be used with DATA_FILE)
export RESPONSE_FIELDS=               # Optional: API response field whitelist (comma-separated, e.g. phone,mail,user_id,status,name); empty = all
export REMOTE_DECRYPT_ENABLED=false   # Optional: decrypt remote response with RSA
export REMOTE_RSA_PRIVATE_KEY_FILE=   # Optional: path to RSA private key PEM (or use REMOTE_RSA_PRIVATE_KEY for inline PEM)
export REMOTE_RSA_PRIVATE_KEY=        # Optional: inline RSA private key PEM (used when REMOTE_RSA_PRIVATE_KEY_FILE is not set)
export HTTP_TIMEOUT=5                  # HTTP request timeout (seconds, or a duration such as 30s; 1s to 1h)
export RATE_LIMIT=60                   # Requests per window, per client IP, that did not authenticate
export RATE_LIMIT_AUTHENTICATED=600    # Requests per window, per credential (API key, HMAC key ID or client certificate)
export RATE_LIMIT_WINDOW=1m            # Rate limit window (duration or seconds, minimum 1s)
export HTTP_MAX_IDLE_CONNS=20          # Idle-connection pool of the remote-source client (default: 20)
export HTTP_INSECURE_TLS=false         # Whether to skip TLS certificate verification (true/false or 1/0)
export API_KEY="your-secret-api-key"   # API Key for authentication (strongly recommended)
export CONFIG_FILE=config.yaml         # Optional YAML file for main and tracing settings; --config-file takes precedence
export OTLP_ENABLED=false              # Enable OpenTelemetry
export OTLP_ENDPOINT=http://localhost:4318  # OTLP endpoint (required when OTLP_ENABLED is true)
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # Trusted proxy IP list (comma-separated), or "none" to trust no peer
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # Health check endpoint IP whitelist (optional)
export IP_WHITELIST="192.168.1.0/24"  # Global IP whitelist (optional)
export LOG_LEVEL="info"                # Log level (optional, default: info, options: trace, debug, info, warn, error, fatal, panic)
export WARDEN_HMAC_KEYS='{"key-id":"0123456789abcdef0123456789abcdef"}'  # Production secrets require at least 32 bytes
export WARDEN_HMAC_ALLOW_V1=false                # Default: false; set true only during a bounded v1 migration
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60     # HMAC timestamp tolerance (seconds)
export WARDEN_TLS_CERT=/path/to/warden.crt    # Service auth: server TLS cert (with KEY enables TLS)
export WARDEN_TLS_KEY=/path/to/warden.key     # Server TLS key
export WARDEN_TLS_CA=/path/to/ca.crt          # Client CA (mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true    # Require client certificate (mTLS)
```

**Environment Variable Priority**:
- Redis password: `REDIS_PASSWORD` > `REDIS_PASSWORD_FILE` > command line argument `--redis-password`
- When Redis is enabled and `REDIS_PASSWORD` is not set, a configured `REDIS_PASSWORD_FILE` (or `redis.password_file`) that cannot be read or is empty stops startup. Warden does not fall back to an unauthenticated connection or to `--redis-password`
- `REDIS_TLS_ENABLED` accepts a boolean; any other value stops startup. The server certificate is always verified. `REDIS_TLS_CA_FILE` and `REDIS_TLS_SERVER_NAME` require `REDIS_TLS_ENABLED=true`

**Security Configuration Notes**:
- `API_KEY`: Used to protect sensitive endpoints (`/`, `/log/level`), strongly recommended for production environments
- `TRUSTED_PROXY_IPS`: Comma-separated IP addresses or CIDR ranges of the reverse proxies whose `X-Forwarded-For` / `X-Real-IP` headers are trusted. An entry that is neither stops startup. The single value `none` (case-insensitive) trusts no peer: forwarded headers are ignored and the client address is the TCP peer; `none` combined with other entries stops startup. Default (empty): forwarded headers are trusted from every peer with a private or loopback address, so another host or container on the same network can present an allowlisted address to `IP_WHITELIST` or obtain a new per-IP rate-limit budget per request. While it is empty Warden logs a startup warning, and with `ENVIRONMENT=production` and `IP_WHITELIST` set it refuses to start. A later release will stop trusting forwarded headers when it is empty. See [SECURITY.md](SECURITY.md#trusted-proxies-and-forwarded-addresses)
- `HEALTH_CHECK_IP_WHITELIST`: Restrict health check endpoint access IPs (optional, supports CIDR ranges). It is compared with the TCP peer address; forwarded headers and `TRUSTED_PROXY_IPS` do not apply
- `RATE_LIMIT`, `RATE_LIMIT_AUTHENTICATED`, `RATE_LIMIT_WINDOW`: `RATE_LIMIT` bounds, per client IP, the requests that fail authentication or need none; `RATE_LIMIT_AUTHENTICATED` bounds each credential. Both must be integers of at least 1; any other value stops startup. See [SECURITY.md](SECURITY.md#rate-limiting)
- `IP_WHITELIST`: Global IP whitelist (optional, supports CIDR ranges)
- `RESPONSE_FIELDS`: Field allowlist for the list, user and lookup responses. Empty returns every field. On `/v1/lookup`, `mail` governs `destination.email`, `phone` governs `destination.phone`, and `name` and `dingtalk_userid` govern the fields of the same name. `user_id`, `status`, `channel_hint` and the `destination` object are always returned because the login gateway depends on them; `channel_hint` is `sms` only when `destination.phone` is returned. `scope` and `role` are not part of the lookup response. A gateway that delivers codes through `/v1/lookup` needs `mail` and/or `phone` in the allowlist, and `dingtalk_userid` for DingTalk delivery. Names are matched exactly and are case-sensitive. The accepted names are `phone`, `mail`, `user_id`, `status`, `scope`, `role`, `name` and `dingtalk_userid`; any other entry, such as `email` or `Mail`, stops startup. An allowlist without `status` is accepted and logs a startup warning: `/user` and the list responses then carry no status, so a client that decides on it treats every user as not active. The Go SDK's `CheckUserInList` returns `false` for every user and `CheckUserInListWithError` returns an `INVALID_RESPONSE` error. Include `status`, or decide through `/v1/lookup`, which always returns it

## Remote Configuration API Requirements

The remote configuration API should return a JSON array in the same format, with optional Authorization header authentication support.

The API response format should match the `data.json` file format:

```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write"],
        "role": "admin"
    }
]
```

If the `KEY` environment variable or `--key` parameter is configured, the `Authorization` header will be automatically added to requests:

```http
Authorization: Bearer your-token-here
```

## Optional Service Integration Configuration

If you choose to integrate with other services (such as Stargate), inter-service authentication can be configured. The following are relevant configuration items:

**Note**: If Warden is used standalone, the following configurations are optional.

### mTLS Configuration (Recommended)

Use mutual TLS certificates for inter-service authentication. **Only environment variables are supported** (no YAML keys in the application config):

```bash
# Warden server certificate
export WARDEN_TLS_CERT=/path/to/warden.crt
export WARDEN_TLS_KEY=/path/to/warden.key
export WARDEN_TLS_CA=/path/to/ca.crt

# Require client certificate (mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
```

### HMAC Signature Configuration

Use HMAC-SHA256 signature for inter-service authentication. **Only environment variables are supported** (no YAML keys):

```bash
# HMAC keys (JSON format, supports multiple keys for rotation)
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef","key-id-2":"abcdef0123456789abcdef0123456789"}'

# Timestamp tolerance (seconds), default 60 when HMAC keys are set
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60

# Legacy v1 is disabled by default. Enable only while migrating old callers.
export WARDEN_HMAC_ALLOW_V1=false
```

### Stargate Calling Configuration

Stargate needs to configure Warden service address and authentication information:

**Stargate Configuration Example** (Environment Variables):
```bash
# Warden service address
export STARGATE_WARDEN_BASE_URL=http://warden:8081

# Inter-service authentication method (mTLS or HMAC)
export STARGATE_WARDEN_AUTH_TYPE=hmac

# HMAC configuration (if using HMAC)
export STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
export STARGATE_WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef

# mTLS configuration (if using mTLS)
export STARGATE_WARDEN_TLS_CERT=/path/to/stargate.crt
export STARGATE_WARDEN_TLS_KEY=/path/to/stargate.key
export STARGATE_WARDEN_TLS_CA=/path/to/ca.crt
```

### Configuration Priority

1. **mTLS**: If TLS certificates are configured, mTLS is used first
2. **HMAC**: If mTLS is not configured, HMAC signature is used
3. **API Key**: If neither is configured, falls back to API Key authentication (not recommended for inter-service calls)

### Configuration Validation

When Warden starts, it checks inter-service authentication configuration:

- Rejects partial TLS configuration: certificate and key must be set together; mTLS also requires a client CA
- If HMAC is configured, verifies key format is correct
- In `ENVIRONMENT=production`, refuses to start unless API Key, HMAC, or mTLS authentication is configured
- In `ENVIRONMENT=production`, refuses to start when `API_KEY` is a documented placeholder value (for example `your-secret-api-key-here`), and logs a warning when it is shorter than 16 characters

## Detailed Configuration Documentation

For more detailed information about parameter parsing mechanisms, priority rules, and usage examples, please refer to:

- [Parameter Parsing Design Document](CONFIG_PARSING.md) - Detailed parameter parsing mechanism documentation
- [Architecture Design Document](ARCHITECTURE.md) - Understand overall architecture and configuration impact
- [Security Documentation](SECURITY.md) - Learn about inter-service authentication details

## Redis snapshot migration

The default Redis format remains unchanged. Opt-in `REDIS_SNAPSHOT_MODE` and
`REDIS_SNAPSHOT_NAMESPACE` settings, time semantics, rollout and rollback are
documented in [Redis snapshot migration](REDIS_SNAPSHOT_MIGRATION.md).

## Server data read freshness

`SNAPSHOT_READ_POLICY` defaults to `enforce` in production and `observe` in
development/test. Explicit `observe` preserves existing data reads, even when
health reports stale data. Use `enforce` to require
source freshness on all authenticated list, paginated list, user and lookup routes,
including their aliases. This setting also applies when configuration comes from YAML.
Invalid values fail startup validation.

Enforcement pins one immutable snapshot for both the freshness check and query.
A known source success time must be no later than the request observation time,
and the request must be strictly before `SourceSucceededAt + SNAPSHOT_MAX_AGE`.
The exact deadline, unknown source age, missing canonical view, or future source
time returns 503 before querying. A fresh snapshot still returns 404 for an absent
user, and an accepted empty set still returns 200 for lists. Authentication runs
first. Enforced data responses use `Cache-Control: no-store`.

Publication, TTL renewal and restart recovery do not extend this deadline. Legacy
Redis snapshots have no source-success evidence and therefore return 503 under
enforcement; use V2 snapshots with preserved source timestamps before enabling it.
Fresh local fallback remains valid: this policy checks the resolved dataset, not
whether every configured remote dependency succeeded. Metrics, merge precedence,
Redis storage format and the SDK default policy are unchanged. Explicit observe-mode
health behavior is unchanged; enforce-mode readiness is described below.

When `SNAPSHOT_READ_POLICY=enforce`, all existing health aliases include a critical
`read_ready` check using the same source freshness decision as data reads. It
checks the pinned canonical view and source-success deadline, including unknown
and future timestamps. The legacy `snapshot_freshness` check remains diagnostic
in this mode; its `LoadedAt` clock does not override source-based readiness.
Other critical checks (data availability and HMAC replay Redis, when applicable)
remain in effect, so readable data alone does not guarantee healthy status.
In observe mode, `read_ready` is absent and existing health behavior is unchanged.
Development health details expose a bounded reason and, when known, `valid_until`;
production continues to hide details. This adds no liveness endpoint and does not
change the meaning of source success to require every remote dependency.

## Following shared snapshots after local expiry

`REDIS_SNAPSHOT_RECOVERY=bootstrap` is the default and preserves existing recovery:
a V2 snapshot may fill an uninitialized replica, but does not replace its known-good
local set. The environment setting applies to both flag and YAML startup paths.
Invalid values fail startup validation.

Opt in with `REDIS_SNAPSHOT_RECOVERY=stale`; this requires Redis enabled and
`REDIS_SNAPSHOT_MODE=v2`. Following is attempted in the refresh path only after a
source error, identity rejection, or an availability-first all-records-rejected
refresh. It requires a recorded failure and a local canonical snapshot whose known
source-success deadline has expired under `SNAPSHOT_MAX_AGE`. It does not run on
HTTP reads, replace fresh or unknown-age data, or enable Redis reads in ONLY_LOCAL.

The shared candidate must pass the existing V2 dataset/policy/schema and effective
rules validation, have a strictly later source-success time, and remain fresh and
not future-dated at the final adoption check. Valid empty input is eligible.
A conditional publication rejects replacement if the local generation changed.
Background refresh remains serialized by the existing refresh mutex.

Recovery preserves SourceSucceededAt, PublishedAt, revision and refresh-failure
count/reason; only RestoredAt records recovery. It remains degraded with reason
`redis_follow_recovery`. The recovery helper does not publish or renew shared data;
an all-records-rejected cycle also skips renewal after successful adoption.
The bounded `follow_read` operation distinguishes recovery outcomes in snapshot
metrics. A missing/invalid/unavailable Redis snapshot leaves local data untouched.
No missing key is rebuilt from old memory. Initial bootstrap behavior, including
its handling of unknown or stale evidence, is unchanged.

This is an availability policy, independent of SNAPSHOT_READ_POLICY. Enable enforce
separately if reads must stop at the source deadline. A later source-success time
is evidence of a later accepted validation, not an ordered upstream business
version or proof that every remote source succeeded. Use distinct namespaces for
distinct datasets; a policy match does not infer tenant identity from source URLs.

## Enabling server and SDK freshness together

Deploy metadata-capable servers and SDKs before enabling constraints. For a bound
on data used by an SDK caller, configure both sides explicitly:

- Server: `SNAPSHOT_READ_POLICY=enforce` and, for example, `SNAPSHOT_MAX_AGE=2m`.
- SDK: `DefaultOptions().WithSourceFreshness(2*time.Minute, 5*time.Second)` before
  `NewClient`, alongside the existing URL and authentication options.

The SDK's cache deadline is the earlier of its own source-age limit and the
server's valid-until, minus the clock-skew allowance, and remains bounded by its
local CacheTTL. Publication, restart recovery and receiving a response do not
restart source age. Fresh missing users remain NOT_FOUND; freshness failure must
not be interpreted as an absent user or an accepted empty list.

Server enforcement only governs requests reaching the server. The default SDK
still uses its existing list-cache TTL and error mapping; `Cache-Control: no-store`
does not automatically disable this application cache. Enabling only the server
therefore does not impose an end-to-end source-age bound on already cached lists.
Enable SDK source freshness explicitly, and account for downstream gateway/session
caches separately. This bounds accepted validation age, not instantaneous
revocation or the upstream content's creation time.

Legacy Redis recovery has no source-success evidence. Migrate to V2 with preserved
source times before requiring freshness after restart. A fresh accepted local
fallback is still resolved-set evidence, not proof that every remote source
succeeded. Keep clocks synchronized and budget skew conservatively.

### Negotiated source 503 errors

An authenticated metadata-v1 request rejected by enforce mode receives HTTP 503,
`Warden-Snapshot-Metadata: v1`, and a bounded `Warden-Source-Error` value:
`expired` for a reached deadline, `unknown` for missing source evidence or its
canonical view, and `future` for future source time. Ordinary requests retain the
existing 503 response without these markers; authentication failures expose none.
Invalid server age configuration is not reported as unknown source evidence.

Only a source-bound SDK interprets these markers: SOURCE_FRESHNESS wraps
ErrSourceStale, ErrSourceUnknown or ErrSourceFuture, detectable with errors.Is.
The original SERVER_ERROR remains in the error chain. A plain 503, missing or
malformed/duplicate markers, an unsupported metadata version, or an unknown reason
keeps SERVER_ERROR. A marker on another HTTP status does not activate 503 mapping.
The default SDK keeps its existing cache and status mapping, even if markers arrive.
Classification happens after the existing HTTP retry policy; enabling it adds no
retry and does not skip configured 503 retries. Fresh 404 remains NOT_FOUND.

## Actual source-attempt diagnostics

Loader results now include per-call attempt diagnostics. Set `LOG_LEVEL=debug`
(or use the existing runtime log-level control) to emit `source attempt completed`
events during startup source loads and background refreshes. Default info-level
logging is unchanged. No additional source request, decode or normalization is
performed for diagnostics; configured but unread sources produce no event.

Each event includes a per-load attempt number, source-family ordinal such as
`local_1` or `remote_1`, file/plain/decrypt transport, start/finish timestamps,
`success`, `failure` or `absent` outcome, a bounded failure-stage reason, decoded
row count, `selected_input`, and `winning_rows`. Ordinals follow the existing
source builder (sorted directory files, then the explicit file), not stable global
source identities. Repeated fallback reads remain separate attempts with the same
ordinal. A remote source invocation includes its existing transport retries; this
is not a per-HTTP-retry trace. Discovery failures and configured sources never
opened are not fabricated as attempted sources.

Success means the source passed loader decoding/normalization (or the encrypted
load pipeline), not final cache validation. An allowed missing file is `absent`,
while explicit `[]` is a successful empty input. `selected_input` means an input
participated in successful loader resolution; a shadowed source or accepted empty
input may have zero winning rows. A successful but ignored empty local input is
not selected. `winning_rows` attributes final loader rows using the existing merge
callbacks and precedence, including identical duplicate records. A failed overall
load has no selected input, even if some source calls succeeded.

Diagnostics contain no source path/URL, credentials, payload, user identifiers or
raw error text. Events include counts, stable resource digests, manifest revisions, ordinals and timestamps. Resource digests are correlatable identifiers, not anonymization or secret storage. The diagnostic
maps used to track winners are local to a load and are not returned or persisted.
These events do not prove required-source coverage, upstream content age, or
remote revocation. They do not change the historical Source label, plaintext and
encrypted merge differences, fallback/degraded policy, LoadedAt, SourceSucceededAt,
readiness or SDK freshness. Subsequent cache validation may reject the resolved
candidate. No attempt evidence is added to Redis V2 envelopes or data/health APIs.

### Stable source identities and observed manifests

Optional `SOURCE_SET_ID` is a non-secret logical dataset identifier (up to 64
letters, digits, colons, underscores or hyphens, starting with a letter or digit). Empty remains unspecified.
Source IDs are versioned SHA-256 resource digests scoped by this value: lexical
absolute file paths (without resolving symlinks) or the configured remote URL.
Authorization headers are excluded; rotate the logical set ID when authentication
switches datasets at the same URL. Digests are identity tokens, not secret storage.
Existing ordinals are retained separately as `source_ordinal`.

Every actual source-building pass records its ordered manifest and directory
 discovery outcome. Repeated fallback enumeration remains unchanged; new members
change the manifest revision and have distinct stable resource IDs. A manifest
lists eligible configured sources, not successful coverage. No extra enumeration,
fetch or decode is added. Evidence remains diagnostic; Redis V2 and read policy
are unchanged.

### Evidence on accepted snapshots

Successful application acceptance binds a detached copy of observed manifests and
source attempts to the same immutable in-memory snapshot as its input/effective
versions and source-success time. Its generation is process-local diagnostics,
not an ordered upstream version. Failure diagnostics can change while this
accepted evidence stays attached to the last-known-good data. A genuine successful
recheck, including unchanged content or an accepted empty set, gets new evidence.
This does not assert per-source cache validity or enforce required coverage.

Redis V2 stores no such evidence: restoration leaves it unknown, rather than
inventing provenance from origin, publication or recovery time. No data/health API,
Redis format, read policy or source-age semantics change.
Directory discovery is indeterminate when filepath.Glob returns normally: it suppresses filesystem errors, so an empty or partial match is not proof of complete discovery. Only a reported pattern error is marked failed. No extra filesystem probe is added.


## Remote source connection policy

Plain and encrypted remote sources use the same connection policy. Redirects are
rejected, environment HTTP proxies are ignored, and each new connection resolves
the configured hostname, checks every returned address and dials a checked IP
directly. TLS still verifies the configured hostname. Loopback, link-local,
metadata, multicast and reserved destinations are rejected.

Public unicast destinations are accepted by default. A private source can be
allowed with `REMOTE_PRIVATE_ORIGINS`, a comma-separated list of exact HTTP(S)
origins such as `https://rules.internal:8443`. The exception includes scheme,
hostname and effective port; it never permits loopback, link-local or metadata
addresses and does not enable redirects. Startup URL checks use the same policy.
Production remote sources must use HTTPS with certificate verification. An
explicit private-origin exception does not relax this requirement. HTTP remains
supported in development/test. Both plaintext and encrypted source fetchers apply
the selected transport policy even when constructed without CLI validation.

A source whose hostname resolves to a refused address fails the load with
`remote host resolves to a prohibited address`. The refresh log line carries
`reason: address_policy`, and `warden_refresh_failures_total` counts it under the
same reason. The request is not retried within that load: the next refresh
resolves and checks the hostname again.

The Compose example requires `REDIS_PASSWORD` and places Redis on a separate
internal network. The gateway should join only `auth-internal`. Do not attach
untrusted containers to either network. For Redis ACL deployments, verify the
commands and key prefixes required by the chosen cache/snapshot mode, including
Lua and leases, before restricting the default user.

Before upgrading production, ensure snapshots have source-success timestamps.
Legacy Redis snapshots cannot establish freshness and return 503 under enforcement
until a source refresh succeeds; V2 snapshots preserve that evidence. An explicit
`SNAPSHOT_READ_POLICY=observe` retains the previous availability policy during a
planned migration, at the cost of allowing stale directory reads.

# Configuration

> 🌐 **Language / 语言**: [English](CONFIGURATION.md) | [中文](../zhCN/CONFIGURATION.md) | [Français](../frFR/CONFIGURATION.md) | [Italiano](../itIT/CONFIGURATION.md) | [日本語](../jaJP/CONFIGURATION.md) | [Deutsch](../deDE/CONFIGURATION.md) | [한국어](../koKR/CONFIGURATION.md)

This document provides detailed information about Warden's configuration options, including running modes, configuration file formats, environment variables, etc.

**Configuration priority**: Command line arguments > Environment variables > Configuration file (YAML) > Defaults.

For a **full option table** (YAML paths, env vars, defaults, validation rules), see [zhCN CONFIGURATION](../zhCN/CONFIGURATION.md). Summary:

| Category | YAML / Env | Notes |
|----------|------------|--------|
| Server | `server.*` / `PORT` | port, read_timeout, write_timeout, shutdown_timeout, idle_timeout, max_header_bytes |
| Redis | `redis.*` / `REDIS`, `REDIS_PASSWORD`, `REDIS_PASSWORD_FILE`, `REDIS_ENABLED` | addr, password, password_file, db; Redis enabled default `true` (except ONLY_LOCAL without REDIS) |
| Cache | `cache.ttl`, `cache.update_interval` | no env overrides; update_interval default 5s |
| Rate limit | `rate_limit.rate`, `rate_limit.window` | default 60/min, 1m window |
| HTTP client | `http.*` / `HTTP_TIMEOUT`, `HTTP_MAX_IDLE_CONNS`, `HTTP_INSECURE_TLS` | timeout, max_idle_conns, insecure_tls, max_retries, retry_delay |
| Remote | `remote.*` / `CONFIG`, `KEY`, `MODE`, `REMOTE_DECRYPT_ENABLED`, `REMOTE_RSA_PRIVATE_KEY_FILE`, `REMOTE_RSA_PRIVATE_KEY` | url, key, mode, decrypt_enabled, rsa_private_key_file |
| Task | `task.interval` | no env override when using config file; use `INTERVAL` only when not using config file |
| App | `app.*` / `API_KEY`, `DATA_FILE`, `DATA_DIR`, `RESPONSE_FIELDS` | mode, api_key, data_file, data_dir, response_fields |
| Tracing | `tracing.enabled`, `tracing.endpoint` / `OTLP_ENABLED`, `OTLP_ENDPOINT` | When using `--config-file`, tracing is not read from that file unless `CONFIG_FILE` is set to the same path |
| Service auth | — / `WARDEN_HMAC_KEYS`, `WARDEN_HMAC_TIMESTAMP_TOLERANCE`, `WARDEN_TLS_*` | **Env only** (no YAML keys) |

## Running Mode (MODE)

The system supports 6 data merging modes, selected via the `MODE` parameter:

| Mode | Description | Use Case |
|------|-------------|----------|
| `DEFAULT` / `REMOTE_FIRST` | Remote-first, use local data to supplement when remote data doesn't exist | Default mode, suitable for most scenarios |
| `ONLY_REMOTE` | Use only remote data source | Fully dependent on remote configuration |
| `ONLY_LOCAL` | Use only local configuration file, **Redis disabled by default** (will be enabled if `REDIS` address is explicitly set or `REDIS_ENABLED=true`) | Offline environment or test environment |
| `LOCAL_FIRST` | Local-first, use remote data to supplement when local data doesn't exist | Local configuration as primary, remote as secondary |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | Remote-first, allow fallback to local when remote fails | High availability scenarios |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | Local-first, allow fallback to remote when local fails | Hybrid mode |

### Configuration Methods

You can set the running mode in the following ways:

**Command Line Arguments**:
```bash
go run . --mode DEFAULT
```

**Environment Variables**:
```bash
export MODE=DEFAULT
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
- `status` (optional): User status, defaults to "active"
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
  rate: 60  # Requests per minute
  window: 1m

http:
  timeout: 5s
  max_idle_conns: 100
  insecure_tls: false  # Development only
  max_retries: 3
  retry_delay: 1s

remote:
  url: "http://localhost:8080/data.json"
  key: ""
  mode: "DEFAULT"
  decrypt_enabled: false       # RSA decrypt remote response (use with rsa_private_key_file or REMOTE_RSA_PRIVATE_KEY)
  rsa_private_key_file: ""    # Path to PEM file (or use env REMOTE_RSA_PRIVATE_KEY for inline PEM)

task:
  interval: 5s

app:
  mode: "DEFAULT"  # Options: DEFAULT, production, prod
  api_key: ""      # Recommend env API_KEY
  data_file: "./data.json"
  data_dir: ""     # Optional: merge all *.json in directory (can be used with data_file)
  response_fields: []  # Optional: API response field whitelist; empty = all fields

tracing:
  enabled: false
  endpoint: ""     # e.g. "http://localhost:4318"
```

**Configuration priority**: Command line arguments > Environment variables > Configuration file > Default values.

**Tracing note**: When using `--config-file`, the main program does not read the `tracing` section from that file unless the `CONFIG_FILE` environment variable is set to the same path, or you use `OTLP_ENABLED` + `OTLP_ENDPOINT`.

Refer to example file: [config.example.yaml](../../config.example.yaml).

## Command Line Arguments

```bash
go run . \
  --port 8081 \                    # Web service port (default: 8081)
  --redis localhost:6379 \         # Redis address (default: localhost:6379)
  --redis-password "password" \    # Redis password (optional, recommend using environment variables)
  --redis-enabled=true \           # Enable/disable Redis (default: true)
  --config http://example.com/api \ # Remote configuration URL
  --key "Bearer token" \           # Remote configuration authentication header
  --interval 5 \                   # Scheduled task interval (seconds, default: 5)
  --mode DEFAULT \                 # Running mode (see description above)
  --http-timeout 5 \               # HTTP request timeout (seconds, default: 5)
  --http-max-idle-conns 100 \     # HTTP maximum idle connections (default: 100)
  --http-insecure-tls \           # Skip TLS certificate verification (development only)
  --api-key "your-secret-api-key" \ # API Key for authentication (optional, recommend using environment variables)
  --config-file config.yaml        # Configuration file path (supports YAML format)
```

**Notes**:
- Configuration file support: You can use the `--config-file` parameter to specify a YAML format configuration file
- Redis password security: Recommend using environment variables `REDIS_PASSWORD` or `REDIS_PASSWORD_FILE` instead of command line arguments
- TLS certificate verification: `--http-insecure-tls` is for development environments only, should not be used in production

## Environment Variables

Supports configuration via environment variables, with lower priority than command line arguments. For the full option table (including validation rules), see [zhCN CONFIGURATION](../zhCN/CONFIGURATION.md).

```bash
export PORT=8081
export REDIS=localhost:6379
export REDIS_PASSWORD="password"        # Redis password (optional)
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Redis password file path (optional; priority: REDIS_PASSWORD > REDIS_PASSWORD_FILE > config)
export REDIS_ENABLED=true               # Enable/disable Redis (optional, default: true, supports true/false/1/0)
                                        # Note: In ONLY_LOCAL mode, default is false
                                        #       But if REDIS address is explicitly set, Redis will be enabled automatically
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5
export MODE=DEFAULT
export DATA_FILE=./data.json          # Local user data file path
export DATA_DIR=                      # Optional: directory to merge all *.json (can be used with DATA_FILE)
export RESPONSE_FIELDS=               # Optional: API response field whitelist (comma-separated, e.g. phone,mail,user_id,status,name); empty = all
export REMOTE_DECRYPT_ENABLED=false   # Optional: decrypt remote response with RSA
export REMOTE_RSA_PRIVATE_KEY_FILE=   # Optional: path to RSA private key PEM (or use REMOTE_RSA_PRIVATE_KEY for inline PEM)
export REMOTE_RSA_PRIVATE_KEY=        # Optional: inline RSA private key PEM (used when REMOTE_RSA_PRIVATE_KEY_FILE is not set)
export HTTP_TIMEOUT=5                  # HTTP request timeout (seconds)
export HTTP_MAX_IDLE_CONNS=100         # HTTP maximum idle connections
export HTTP_INSECURE_TLS=false         # Whether to skip TLS certificate verification (true/false or 1/0)
export API_KEY="your-secret-api-key"   # API Key for authentication (strongly recommended)
export CONFIG_FILE=config.yaml         # Optional; used to load tracing from YAML when not using --config-file, or to enable tracing from same file as --config-file
export OTLP_ENABLED=false              # Enable OpenTelemetry (true/false or 1/0)
export OTLP_ENDPOINT=http://localhost:4318  # OTLP endpoint (required when OTLP_ENABLED is true)
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # Trusted proxy IP list (comma-separated)
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # Health check endpoint IP whitelist (optional)
export IP_WHITELIST="192.168.1.0/24"  # Global IP whitelist (optional)
export LOG_LEVEL="info"                # Log level (optional, default: info, options: trace, debug, info, warn, error, fatal, panic)
export WARDEN_HMAC_KEYS='{"key-id":"secret"}'  # Service auth: HMAC keys (JSON)
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60     # HMAC timestamp tolerance (seconds)
export WARDEN_TLS_CERT=/path/to/warden.crt    # Service auth: server TLS cert (with KEY enables TLS)
export WARDEN_TLS_KEY=/path/to/warden.key     # Server TLS key
export WARDEN_TLS_CA=/path/to/ca.crt          # Client CA (mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true    # Require client certificate (mTLS)
```

**Environment Variable Priority**:
- Redis password: `REDIS_PASSWORD` > `REDIS_PASSWORD_FILE` > command line argument `--redis-password`

**Security Configuration Notes**:
- `API_KEY`: Used to protect sensitive endpoints (`/`, `/log/level`), strongly recommended for production environments
- `TRUSTED_PROXY_IPS`: Configure trusted reverse proxy IPs to correctly obtain client real IP
- `HEALTH_CHECK_IP_WHITELIST`: Restrict health check endpoint access IPs (optional, supports CIDR ranges)
- `IP_WHITELIST`: Global IP whitelist (optional, supports CIDR ranges)

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
export WARDEN_HMAC_KEYS='{"key-id-1":"secret-key-1","key-id-2":"secret-key-2"}'

# Timestamp tolerance (seconds), default 60 when HMAC keys are set
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60
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
export STARGATE_WARDEN_HMAC_SECRET=secret-key-1

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

- If mTLS is configured, verifies certificate files exist
- If HMAC is configured, verifies key format is correct
- If neither is configured, logs a warning (not recommended for production)

## Detailed Configuration Documentation

For more detailed information about parameter parsing mechanisms, priority rules, and usage examples, please refer to:

- [Parameter Parsing Design Document](CONFIG_PARSING.md) - Detailed parameter parsing mechanism documentation
- [Architecture Design Document](ARCHITECTURE.md) - Understand overall architecture and configuration impact
- [Security Documentation](SECURITY.md) - Learn about inter-service authentication details

# Configuration

> 🌐 **Language / 语言**: [English](CONFIGURATION.md) | [中文](../zhCN/CONFIGURATION.md)

This document provides detailed information about Warden's configuration options, including running modes, configuration file formats, environment variables, etc.

## Running Mode (MODE)

The system supports 6 data merging modes, selected via the `MODE` parameter:

| Mode | Description | Use Case |
|------|-------------|----------|
| `DEFAULT` / `REMOTE_FIRST` | Remote-first, use local data to supplement when remote data doesn't exist | Default mode, suitable for most scenarios |
| `ONLY_REMOTE` | Use only remote data source | Fully dependent on remote configuration |
| `ONLY_LOCAL` | Use only local configuration file, **Redis disabled by default** (can be explicitly enabled via `REDIS_ENABLED=true`) | Offline environment or test environment |
| `LOCAL_FIRST` | Local-first, use remote data to supplement when local data doesn't exist | Local configuration as primary, remote as secondary |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | Remote-first, allow fallback to local when remote fails | High availability scenarios |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | Local-first, allow fallback to remote when local fails | Hybrid mode |

### Configuration Methods

You can set the running mode in the following ways:

**Command Line Arguments**:
```bash
go run main.go --mode DEFAULT
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

task:
  interval: 5s

app:
  mode: "DEFAULT"  # Options: DEFAULT, production, prod
```

**Configuration Priority**: Command line arguments > Environment variables > Configuration file > Default values

Refer to example file: [config.example.yaml](../config.example.yaml)

## Command Line Arguments

```bash
go run main.go \
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

Supports configuration via environment variables, with lower priority than command line arguments:

```bash
export PORT=8081
export REDIS=localhost:6379
export REDIS_PASSWORD="password"        # Redis password (optional)
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Redis password file path (optional, higher priority than REDIS_PASSWORD)
export REDIS_ENABLED=true               # Enable/disable Redis (optional, default: true, supports true/false/1/0)
                                        # Note: In ONLY_LOCAL mode, default is false (unless explicitly set)
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5
export MODE=DEFAULT
export HTTP_TIMEOUT=5                  # HTTP request timeout (seconds)
export HTTP_MAX_IDLE_CONNS=100         # HTTP maximum idle connections
export HTTP_INSECURE_TLS=false         # Whether to skip TLS certificate verification (true/false or 1/0)
export API_KEY="your-secret-api-key"   # API Key for authentication (strongly recommended)
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # Trusted proxy IP list (comma-separated)
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # Health check endpoint IP whitelist (optional)
export IP_WHITELIST="192.168.1.0/24"  # Global IP whitelist (optional)
export LOG_LEVEL="info"                # Log level (optional, default: info, options: trace, debug, info, warn, error, fatal, panic)
```

**Environment Variable Priority**:
- Redis password: `REDIS_PASSWORD_FILE` > `REDIS_PASSWORD` > command line argument `--redis-password`

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

## Detailed Configuration Documentation

For more detailed information about parameter parsing mechanisms, priority rules, and usage examples, please refer to:

- [Parameter Parsing Design Document](CONFIG_PARSING.md) - Detailed parameter parsing mechanism documentation
- [Architecture Design Document](ARCHITECTURE.md) - Understand overall architecture and configuration impact

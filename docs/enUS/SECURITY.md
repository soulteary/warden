# Security Documentation

> 🌐 **Language / 语言**: [English](SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

This document explains Warden's security features, security configuration, and best practices.

## Implemented Security Features

1. **API Authentication**: Supports API Key authentication to protect sensitive endpoints
2. **SSRF Protection**: Strictly validates remote configuration URLs to prevent Server-Side Request Forgery attacks
3. **Input Validation**: Strictly validates all input parameters to prevent injection attacks
4. **Rate Limiting**: IP-based rate limiting to prevent DDoS attacks
5. **TLS Verification**: Production environments enforce TLS certificate verification
6. **Error Handling**: Production environments hide detailed error information to prevent information leakage
7. **Security Response Headers**: Automatically adds security-related HTTP response headers
8. **IP Whitelist**: Supports configuring IP whitelist for health check endpoints
9. **Configuration File Validation**: Prevents path traversal attacks
10. **JSON Size Limits**: Limits JSON response body size to prevent memory exhaustion attacks
11. **User Query Parameter Length Limit**: Single parameter (`phone`/`mail`/`user_id`) must not exceed 512 bytes to prevent DoS and log/cache bloat
12. **Audit Log PII Sanitization**: Identifier written to audit is sanitized for phone/mail to avoid PII exposure if audit storage is compromised

## Security Best Practices

### 1. Production Environment Configuration

**Required Configuration**:
- **Must** set `ENVIRONMENT=production` to enable production hardening.
- **Must** configure at least one service authentication mechanism: `API_KEY`, HMAC v2, or mTLS.
- **Must** configure `TRUSTED_PROXY_IPS` to correctly obtain client IP
- **Must** use `HEALTH_CHECK_IP_WHITELIST` to restrict health check access (or restrict `/health`, `/healthcheck` via network/reverse proxy)
- **Must** restrict `/metrics`. With `ENVIRONMENT=production` authentication is required by default; keep it that way (or restrict the path at the reverse proxy/network layer) and do **not** set `WARDEN_METRICS_REQUIRE_AUTH=false` in production.

**Configuration Example**:
```bash
export API_KEY="$(openssl rand -hex 32)"   # generate once and keep it in a secret store
export ENVIRONMENT=production
export WARDEN_METRICS_REQUIRE_AUTH=true
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"
```

### 2. Sensitive Information Management

**Recommended Practices**:
- ✅ Use environment variables to store passwords and keys
- ✅ Use password files (`REDIS_PASSWORD_FILE`) to store Redis passwords
- ✅ Use placeholders or comments in configuration files
- ✅ Ensure configuration file permissions are set correctly (e.g., `chmod 600`)

**Not Recommended**:
- ❌ Hardcode passwords in configuration files
- ❌ Pass passwords via command line arguments (will appear in process list)
- ❌ Commit configuration files containing sensitive information to version control

**Example**:
```yaml
# config.yaml
redis:
  addr: "localhost:6379"
  # password: ""  # Use environment variable REDIS_PASSWORD or REDIS_PASSWORD_FILE

app:
  # api_key: ""  # Use environment variable API_KEY
```

### 3. Network Security

**Required Configuration**:
- Production environments must use HTTPS
- Configure firewall rules to restrict access
- Regularly update dependencies to fix known vulnerabilities

**Recommended Configuration**:
- Use reverse proxy (such as Nginx) to handle SSL/TLS
- Configure `TRUSTED_PROXY_IPS` to correctly obtain client real IP
- Use strong passwords and API keys
- Disable `HTTP_INSECURE_TLS` (must be `false` in production)

### 4. Monitoring and Auditing

**Recommended Practices**:
- Monitor security event logs
- Regularly review access logs
- Use security scanning tools in CI/CD
- Set up alert mechanisms

**Log Level Management**:
- Production environments recommend using `info` or `warn` level
- All log level modification operations are recorded in security audit logs
- Log levels can be dynamically adjusted via `/log/level` API (requires the configured service authentication)

## API Security

### API Key Authentication

Some API endpoints require API Key authentication:

**Endpoints Requiring Authentication**:
- `GET /` - Get user list
- `GET /user` - Query single user
- `GET /log/level` - Get log level
- `POST /log/level` - Set log level

**Endpoints Not Requiring Authentication** (must be protected by other means in production):
- `GET /health` - Health check (**must** configure `HEALTH_CHECK_IP_WHITELIST` or network isolation)
- `GET /healthcheck` - Health check (same as above)
- `GET /metrics` - Prometheus metrics (**must** set API Key for scrape or restrict via reverse proxy/network; do not expose publicly)

**Authentication Methods**:
1. **X-API-Key Header**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer Header**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

**Key strength**: generate the key from a secure random source, for example
`openssl rand -hex 32`. With `ENVIRONMENT=production`, Warden refuses to start when
`API_KEY` is one of the placeholder values used in this documentation and the
examples (such as `your-secret-api-key-here` or `your-api-key-here`), and logs a
startup warning when the key is shorter than 16 characters.

### Rate Limiting

By default, API requests are protected by rate limiting:

- **Limit**: 60 requests per minute
- **Window**: 1 minute
- **Exceeded**: Returns `429 Too Many Requests`

Adjust it in the configuration file, or with the `RATE_LIMIT` and
`RATE_LIMIT_WINDOW` environment variables, which take precedence:

```yaml
rate_limit:
  rate: 60  # Requests per window, per client IP
  window: 1m
```

The limit is counted per client IP. A gateway such as Stargate reaches Warden
from a single address, so all of its lookups share one quota; size the limit for
the caller's total request rate, not for one end user.

The limit also applies to `/metrics` and `/log/level`, which share the same
per-client budget as the data endpoints. Health check endpoints are not rate
limited. A Prometheus scraper that shares a client address with other callers
consumes part of that address's quota.

### IP Whitelist

Supports two types of IP whitelist configuration:

1. **Global IP Whitelist** (`IP_WHITELIST`):
   - Restricts access to all endpoints
   - Supports CIDR range format

2. **Health Check IP Whitelist** (`HEALTH_CHECK_IP_WHITELIST`):
   - Only restricts `/health` and `/healthcheck` endpoints
   - Supports CIDR range format

**Configuration Example**:
```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

### Trusted Proxies and Forwarded Addresses

`IP_WHITELIST`, rate limiting and access logs use the client address. Warden reads
`X-Forwarded-For` / `X-Real-IP` only when the directly connected peer is a trusted
proxy; otherwise it uses the peer address.

- **`TRUSTED_PROXY_IPS` set**: only the listed addresses and CIDR ranges are trusted
  proxies. Forwarded headers from every other peer are ignored.
- **`TRUSTED_PROXY_IPS` empty (default)**: every peer with a private or loopback
  address (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.0/8`, IPv6
  loopback, link-local and unique-local) is treated as a trusted proxy. Any host
  or container on the same private network can then send
  `X-Forwarded-For: <allowlisted address>` to pass `IP_WHITELIST`, or vary the
  header to obtain a fresh rate-limit bucket per request.

Recommendation: whenever `IP_WHITELIST` is used, or Warden is reachable from other
workloads on a private network, set `TRUSTED_PROXY_IPS` to the exact addresses of
your reverse proxies. Warden logs a startup warning when `IP_WHITELIST` is set and
`TRUSTED_PROXY_IPS` is empty.

Each `TRUSTED_PROXY_IPS` entry must be an IP address or a CIDR range. Any other
value (a hostname, an address with a port) stops startup, because a dropped entry
could leave the list empty and restore the default above.

## Data Security

### Remote Configuration API Security

- Remote configuration APIs should use authentication mechanisms (Authorization header)
- Recommend using HTTPS protocol
- Verify remote API TLS certificates (required in production)

### Redis Security

- Redis should be configured with password protection
- Use `REDIS_PASSWORD` or `REDIS_PASSWORD_FILE` environment variables
- A configured `REDIS_PASSWORD_FILE` that is unreadable or empty stops startup instead of connecting without a password
- Use a dedicated ACL user with `REDIS_USERNAME` rather than the default user
- Set `REDIS_TLS_ENABLED=true` when Redis is not on a trusted private network; use `REDIS_TLS_CA_FILE` for a private CA. TLS is off by default
- Restrict Redis network access (only allow application server access)
- Regularly update Redis to fix known vulnerabilities

### Data File Security

- Ensure `data.json` file permissions are set correctly
- Do not commit sensitive data to version control
- Regularly backup data files

## Security Response Headers

Warden automatically adds the following security-related HTTP response headers:

- `X-Content-Type-Options: nosniff` - Prevents MIME type sniffing
- `X-Frame-Options: DENY` - Prevents clickjacking
- `X-XSS-Protection: 1; mode=block` - XSS protection

## Error Handling

### Production Mode

In production mode (`ENVIRONMENT=production`):

- Hide detailed error information to prevent information leakage
- Return generic error messages
- Detailed error information is only recorded in logs

### Development Mode

In development mode:

- Display detailed error information for debugging
- Include stack trace information

## Security Audit

For release hardening and verification guidance, see [Release Security](../RELEASE_SECURITY.md).

## Vulnerability Reporting

If you discover a security vulnerability, please report it through:

1. Create a private security Issue (if supported)
2. Send email to project maintainers
3. Do not publicly disclose the vulnerability until it is fixed

<a name="inter-service-authentication"></a>

## Inter-Service Authentication (Optional)

Warden supports mTLS, HMAC and API keys. Authentication is selected per request as described below. Standalone deployments still require configured authentication in production; only development/test may use the documented unauthenticated policy.

### mTLS (Recommended)

Use mutual TLS certificates for authentication, providing higher security.

**Configuration**:

1. **Generate Certificates**:
   ```bash
   # Generate CA certificate
   openssl genrsa -out ca.key 2048
   openssl req -new -x509 -days 365 -key ca.key -out ca.crt
   
   # Generate Warden server certificate
   openssl genrsa -out warden.key 2048
   openssl req -new -key warden.key -out warden.csr
   openssl x509 -req -days 365 -in warden.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out warden.crt
   
   # Generate Stargate client certificate
   openssl genrsa -out stargate.key 2048
   openssl req -new -key stargate.key -out stargate.csr
   openssl x509 -req -days 365 -in stargate.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out stargate.crt
   ```

2. **Warden Configuration** (Environment Variables):
   ```bash
   export WARDEN_TLS_CERT=/path/to/warden.crt
   export WARDEN_TLS_KEY=/path/to/warden.key
   export WARDEN_TLS_CA=/path/to/ca.crt
   export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
   ```

3. **Stargate Configuration**:
   - Configure client certificate path
   - Configure CA certificate path to verify Warden server certificate

### HMAC Signature

Use HMAC-SHA256 signature to verify requests, easier to deploy.

**Signature Algorithm**:
```text
canonical_v2 = METHOD + "\n" + ESCAPED_PATH_AND_QUERY + "\n" + KEY_ID + "\n" +
               TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(BODY)
signature = HEX(HMAC_SHA256(secret, canonical_v2))
```

**Request Headers**:
- `X-Signature`: HMAC signature value
- `X-Timestamp`: Unix timestamp (seconds)
- `X-Key-Id`: Key ID (included in the signature, for safe key rotation)
- `X-Nonce`: Unique 128-bit hexadecimal nonce
- `X-Signature-Version`: `v2`

**Warden Configuration** (Environment Variables):
```bash
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60  # Timestamp tolerance (seconds), default 60
export WARDEN_HMAC_ALLOW_V1=false          # Default; set true only during a bounded legacy migration
```

Production configuration requires every HMAC secret to contain at least 32 raw bytes.
Generate secrets with a cryptographically secure random source and store them in a
secret manager; do not reuse the illustrative value above.

The signed body is read before the signature can be checked, so that read is
capped at the request body limit (10 KB) on every route and method. A request
with complete HMAC headers and a larger body is rejected with 401.

**Go SDK Example**:
```go
client, err := warden.NewClient(warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET")))
```

**Verification Rules**:
- Warden verifies if timestamp is within tolerance range (default ±60 seconds)
- Warden verifies all canonical fields, including Key ID, and rejects reused nonces
- If signature verification fails, returns `401 Unauthorized`

### Configuration Priority

1. With `WARDEN_SERVICE_POLICIES`, require an authenticated HMAC v2 key ID and its permitted action. API keys and mTLS do not bypass this profile.
2. Without that profile, a request with a peer certificate accepted by the server TLS policy bypasses header authentication. Merely configuring a server certificate does not authenticate every request.
3. Otherwise, complete HMAC headers plus configured keys enter signature verification. A failed signature returns 401 without falling back to API key; legacy v1 requires the explicit compatibility switch.
4. Requests outside that complete HMAC branch go to the API-key middleware (`X-API-Key` or Bearer). Enabling HMAC does not by itself disable separately configured API-key access.

### Security Recommendations

1. **Production Environment**: Strongly recommend using mTLS for inter-service authentication
2. **Key Management**: Use key management services (such as HashiCorp Vault) to store keys and certificates
3. **Key Rotation**: Regularly rotate HMAC keys and TLS certificates
4. **Network Isolation**: When possible, use network policies to restrict access to Warden only from Stargate

## Related Documentation

- [Configuration Documentation](CONFIGURATION.md) - Learn about security-related configuration options
- [Deployment Documentation](DEPLOYMENT.md) - Learn about production environment deployment recommendations
- [API Documentation](API.md) - Learn about API security features
- [Architecture Documentation](ARCHITECTURE.md) - Learn about service integration architecture

## Diagnostic compatibility and historical logs

Upgrade requirements, diagnostic field semantics and historical log handling are
covered in the [v1.5.0 release notes](../releases/v1.5.0.md). Upgrading does not
rewrite previously stored logs. Masked identifiers and resource digests remain
correlatable; access and retention controls still apply.


## Remote source authenticity

Production rule sources require HTTPS with certificate verification, including
sources permitted by `REMOTE_PRIVATE_ORIGINS`. Migrate an existing HTTP source to
a verified HTTPS endpoint before upgrading. Decryption and required v2 envelopes
remain separate checks.

RSA-OAEP and AES-GCM provide confidentiality and integrity for the encrypted
envelope. They do not identify its publisher: anyone with the recipient public
key can create an envelope. HTTPS authenticates the configured delivery endpoint;
it does not prove the age or publisher identity of an object stored there. If
objects must remain verifiable independently of that endpoint, use a separately
designed publisher-signature and version/expiry protocol.

The Go SDK rejects every redirect, including same-origin redirects and HTTPS
downgrades. API keys, HMAC headers and client-certificate handshakes must remain
bound to the configured endpoint. Configure the final Warden URL directly rather
than relying on an HTTP redirect. This policy is local to the SDK client and does
not change the process-wide HTTP client or the generic http-kit defaults.

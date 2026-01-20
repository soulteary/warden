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

## Security Best Practices

### 1. Production Environment Configuration

**Required Configuration**:
- Must set `API_KEY` environment variable
- Set `MODE=production` to enable production mode
- Configure `TRUSTED_PROXY_IPS` to correctly obtain client IP
- Use `HEALTH_CHECK_IP_WHITELIST` to restrict health check access

**Configuration Example**:
```bash
export API_KEY="your-strong-api-key-here"
export MODE=production
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
- Log levels can be dynamically adjusted via `/log/level` API (requires API Key authentication)

## API Security

### API Key Authentication

Some API endpoints require API Key authentication:

**Endpoints Requiring Authentication**:
- `GET /` - Get user list
- `GET /user` - Query single user
- `GET /log/level` - Get log level
- `POST /log/level` - Set log level

**Endpoints Not Requiring Authentication**:
- `GET /health` - Health check (can be restricted via IP whitelist)
- `GET /healthcheck` - Health check (can be restricted via IP whitelist)
- `GET /metrics` - Prometheus metrics

**Authentication Methods**:
1. **X-API-Key Header**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer Header**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### Rate Limiting

By default, API requests are protected by rate limiting:

- **Limit**: 60 requests per minute
- **Window**: 1 minute
- **Exceeded**: Returns `429 Too Many Requests`

Can be adjusted via configuration file:

```yaml
rate_limit:
  rate: 60  # Requests per minute
  window: 1m
```

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

## Data Security

### Remote Configuration API Security

- Remote configuration APIs should use authentication mechanisms (Authorization header)
- Recommend using HTTPS protocol
- Verify remote API TLS certificates (required in production)

### Redis Security

- Redis should be configured with password protection
- Use `REDIS_PASSWORD` or `REDIS_PASSWORD_FILE` environment variables
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

In production mode (`MODE=production` or `MODE=prod`):

- Hide detailed error information to prevent information leakage
- Return generic error messages
- Detailed error information is only recorded in logs

### Development Mode

In development mode:

- Display detailed error information for debugging
- Include stack trace information

## Security Audit

For detailed security audit reports, please refer to [SECURITY_AUDIT.md](../SECURITY_AUDIT.md) (if exists).

## Vulnerability Reporting

If you discover a security vulnerability, please report it through:

1. Create a private security Issue (if supported)
2. Send email to project maintainers
3. Do not publicly disclose the vulnerability until it is fixed

## Related Documentation

- [Configuration Documentation](CONFIGURATION.md) - Learn about security-related configuration options
- [Deployment Documentation](DEPLOYMENT.md) - Learn about production environment deployment recommendations
- [API Documentation](API.md) - Learn about API security features

# Security Policy

## Supported Versions

We actively support security updates for the following versions of Warden:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest| :x:                |

## Reporting a Vulnerability

We take the security of Warden seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### How to Report

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

1. **GitHub Security Advisory** (Preferred)
   - Go to the [Security tab](https://github.com/soulteary/warden/security) in the repository
   - Click on "Report a vulnerability"
   - Fill out the security advisory form

2. **Email** (If GitHub Security Advisory is not available)
   - Send an email to the project maintainers
   - Include a detailed description of the vulnerability
   - Include steps to reproduce the issue (if applicable)
   - Include potential impact assessment

### What to Include

When reporting a security vulnerability, please include:

- **Description**: A clear description of the vulnerability
- **Impact**: The potential impact of the vulnerability
- **Steps to Reproduce**: Detailed steps to reproduce the issue (if applicable)
- **Affected Versions**: Which versions are affected
- **Suggested Fix**: If you have a suggested fix, please include it (optional)

### What to Expect

- **Acknowledgment**: You will receive an acknowledgment within 48 hours
- **Initial Assessment**: We will provide an initial assessment within 7 days
- **Updates**: We will keep you informed of our progress
- **Resolution**: We will notify you when the vulnerability is resolved
- **Disclosure**: We will coordinate with you on public disclosure timing

### Disclosure Policy

- We ask that you keep the vulnerability confidential until we have had a chance to address it
- We will work with you to coordinate public disclosure after a fix is available
- We will credit you in the security advisory (unless you prefer to remain anonymous)

## Security Features

Warden implements multiple security features to protect your data and infrastructure:

- **API Authentication**: API Key authentication for sensitive endpoints
- **SSRF Protection**: Strict validation of remote configuration URLs
- **Input Validation**: Comprehensive input validation to prevent injection attacks
- **Rate Limiting**: IP-based rate limiting to prevent DDoS attacks
- **TLS Verification**: Enforced TLS certificate verification in production
- **Security Headers**: Automatic security-related HTTP response headers
- **IP Whitelist**: Configurable IP whitelist for access control
- **Error Handling**: Production mode hides detailed error information

For detailed information about security features and best practices, please refer to:

- [Security Documentation (English)](docs/enUS/SECURITY.md)
- [Security Documentation (中文)](docs/zhCN/SECURITY.md)

## Security Best Practices

### Production Deployment

When deploying Warden in production:

1. **Set Strong API Keys**: Use a strong, randomly generated API key
2. **Enable Production Mode**: Set `MODE=production`
3. **Use HTTPS**: Always use HTTPS in production environments
4. **Configure Trusted Proxies**: Set `TRUSTED_PROXY_IPS` correctly
5. **Restrict Access**: Use IP whitelists where appropriate
6. **Secure Redis**: Use password protection for Redis
7. **Regular Updates**: Keep dependencies and the application up to date
8. **Monitor Logs**: Regularly review security event logs

### Configuration Security

- Use environment variables for sensitive information (API keys, passwords)
- Never commit configuration files with sensitive data to version control
- Set appropriate file permissions (e.g., `chmod 600` for config files)
- Use password files (`REDIS_PASSWORD_FILE`) instead of command-line arguments

For more detailed security configuration guidance, see the [Configuration Documentation](docs/enUS/CONFIGURATION.md).

## Security Updates

Security updates are released as soon as possible after a vulnerability is identified and fixed. We recommend:

- Subscribing to repository notifications for security advisories
- Regularly updating to the latest version
- Reviewing the [CHANGELOG](CHANGELOG.md) (if available) for security-related changes

## Acknowledgments

We would like to thank all security researchers and contributors who help keep Warden secure by responsibly reporting vulnerabilities.

---

**Note**: This security policy is subject to change. Please check back periodically for updates.

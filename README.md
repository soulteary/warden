# Warden

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

A high-performance AllowList user data service that supports data synchronization and merging from local and remote configuration sources.

![Warden](.github/assets/banner.jpg)

> **Warden** (The Gatekeeper) — The guardian of the Stargate who decides who may pass and who will be denied. Just as the Warden of Stargate guards the Stargate, Warden guards your allowlist, ensuring only authorized users can pass through.

## 📋 Project Overview

Warden is a lightweight HTTP API service developed in Go, primarily used for providing and managing allowlist user data (phone numbers and email addresses). The service supports fetching data from local configuration files and remote APIs, and provides multiple data merging strategies to ensure data real-time performance and reliability.

## ✨ Core Features

- 🚀 **High Performance**: Supports 5000+ requests per second with an average latency of 21ms
- 🔄 **Multiple Data Sources**: Supports both local configuration files and remote APIs
- 🎯 **Flexible Strategies**: Provides 6 data merging modes (remote-first, local-first, remote-only, local-only, etc.)
- ⏰ **Scheduled Updates**: Scheduled tasks based on Redis distributed locks for automatic data synchronization
- 📦 **Containerized Deployment**: Complete Docker support, ready to use out of the box
- 📊 **Structured Logging**: Uses zerolog to provide detailed access logs and error logs
- 🔒 **Distributed Locks**: Uses Redis to ensure scheduled tasks don't execute repeatedly in distributed environments
- 🌐 **Multi-language Support**: Supports 7 languages (English, Chinese, French, Italian, Japanese, German, Korean) with automatic language detection

## 🏗️ Architecture Design

Warden uses a layered architecture design, including HTTP layer, business layer, and infrastructure layer. The system supports multiple data sources, multi-level caching, and distributed locking mechanisms.

For detailed architecture documentation, please refer to: [Architecture Design Documentation](docs/enUS/ARCHITECTURE.md)

## 📦 Installation and Running

> 💡 **Quick Start**: Want to quickly experience Warden? Check out our [Quick Start Examples](example/README.md):
> - [Simple Example](example/basic/README.md) - Basic usage, local data file only
> - [Advanced Example](example/advanced/README.md) - Full features, including remote API and Mock service

### Prerequisites

- Go 1.25+ (refer to [go.mod](go.mod))
- Redis (for distributed locks and caching)
- Docker (optional, for containerized deployment)

### Quick Start

1. **Clone the project**
```bash
git clone <repository-url>
cd warden
```

2. **Install dependencies**
```bash
go mod download
```

3. **Configure local data file**
Create a `data.json` file (refer to `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

4. **Run the service**
```bash
go run main.go
```

For detailed configuration and deployment instructions, please refer to:
- [Configuration Documentation](docs/enUS/CONFIGURATION.md) - Learn about all configuration options
- [Deployment Documentation](docs/enUS/DEPLOYMENT.md) - Learn about deployment methods

## ⚙️ Configuration

Warden supports multiple configuration methods: command line arguments, environment variables, and configuration files. The system provides 6 data merging modes with flexible configuration strategies.

For detailed configuration documentation, please refer to: [Configuration Documentation](docs/enUS/CONFIGURATION.md)

## 📡 API Documentation

Warden provides a complete RESTful API with support for user list queries, pagination, health checks, and more. The project also provides OpenAPI 3.0 specification documentation.

For detailed API documentation, please refer to: [API Documentation](docs/enUS/API.md)

OpenAPI specification file: [openapi.yaml](openapi.yaml)

## 🌐 Multi-language Support

Warden supports complete internationalization (i18N) functionality. All API responses, error messages, and logs support internationalization.

### Supported Languages

- 🇺🇸 English (en) - Default
- 🇨🇳 Chinese (zh)
- 🇫🇷 French (fr)
- 🇮🇹 Italian (it)
- 🇯🇵 Japanese (ja)
- 🇩🇪 German (de)
- 🇰🇷 Korean (ko)

### Language Detection

Warden supports two language detection methods with the following priority:

1. **Query Parameter**: Specify language via `?lang=zh`
2. **Accept-Language Header**: Automatically detect browser language preference
3. **Default Language**: English if not specified

### Usage Examples

```bash
# Specify Chinese via query parameter
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=zh"

# Auto-detect via Accept-Language header
curl -H "X-API-Key: your-key" -H "Accept-Language: zh-CN,zh;q=0.9" "http://localhost:8081/"

# Use Japanese
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=ja"
```

For detailed multi-language documentation, please refer to: [Multi-language Documentation](docs/enUS/README.md#multi-language-support)

## 🐳 Docker Deployment

Warden supports complete Docker and Docker Compose deployment, ready to use out of the box.

### Quick Start with Pre-built Image (Recommended)

Use the pre-built image from GitHub Container Registry (GHCR) to get started quickly without local build:

```bash
# Pull the latest version image
docker pull ghcr.io/soulteary/warden:latest

# Run container (basic example)
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Tip**: Using pre-built images allows you to get started quickly without a local build environment. Images are automatically updated to ensure you're using the latest version.

### Using Docker Compose

> 🚀 **Quick Deployment**: Check the [Examples Directory](example/README.md) for complete Docker Compose configuration examples

For detailed deployment documentation, please refer to: [Deployment Documentation](docs/enUS/DEPLOYMENT.md)

## 📊 Performance Metrics

Based on wrk stress test results (30-second test, 16 threads, 100 connections):

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Average Latency: 21.30ms
Max Latency:     226.09ms
```

## 📁 Project Structure

```
warden/
├── main.go                 # Program entry point
├── data.example.json      # Local data file example
├── config.example.yaml    # Configuration file example
├── openapi.yaml           # OpenAPI specification file
├── go.mod                 # Go module definition
├── docker-compose.yml     # Docker Compose configuration
├── LICENSE                # License file
├── README.*.md            # Multi-language project documents (Chinese/English/French/Italian/Japanese/German/Korean)
├── CONTRIBUTING.*.md      # Multi-language contribution guides
├── docker/
│   └── Dockerfile         # Docker image build file
├── docs/                  # Documentation directory (multi-language)
│   ├── enUS/              # English documentation
│   └── zhCN/              # Chinese documentation
├── example/               # Quick start examples
│   ├── basic/             # Simple example (local file only)
│   └── advanced/          # Advanced example (full features, includes Mock API)
├── internal/
│   ├── cache/             # Redis cache and lock implementation
│   ├── cmd/               # Command line argument parsing
│   ├── config/            # Configuration management
│   ├── define/            # Constant definitions and data structures
│   ├── di/                # Dependency injection
│   ├── errors/            # Error handling
│   ├── logger/            # Logging initialization
│   ├── metrics/           # Metrics collection
│   ├── middleware/        # HTTP middleware
│   ├── parser/            # Data parser (local/remote)
│   ├── router/            # HTTP route handling
│   ├── validator/         # Validator
│   └── version/           # Version information
├── pkg/
│   ├── gocron/            # Scheduled task scheduler
│   └── warden/            # Warden SDK
├── scripts/               # Scripts directory
└── .github/               # GitHub configuration (CI/CD, Issue/PR templates, etc.)
```

## 🔒 Security Features

Warden implements multiple security features, including API authentication, SSRF protection, rate limiting, TLS verification, and more.

For detailed security documentation, please refer to: [Security Documentation](docs/enUS/SECURITY.md)

## 🔧 Development Guide

> 📚 **Reference Examples**: Check the [Examples Directory](example/README.md) for complete example code and configurations for different usage scenarios.

For detailed development documentation, please refer to: [Development Documentation](docs/enUS/DEVELOPMENT.md)

### Code Standards

The project follows Go official code standards and best practices. For detailed standards, please refer to:

- [CODE_STYLE.md](docs/enUS/CODE_STYLE.md) - Code style guide
- [CONTRIBUTING.en.md](CONTRIBUTING.en.md) - Contribution guide

## 📄 License

See the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Welcome to submit Issues and Pull Requests!

## 📞 Contact

For questions or suggestions, please contact via Issues.

---

**Version**: The program displays version, build time, and code version on startup (via `warden --version` or startup logs)


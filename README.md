# Warden

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/warden/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/warden)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/warden)](https://goreportcard.com/report/github.com/soulteary/warden)

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

A high-performance AllowList user data service that supports data synchronization and merging from local and remote configuration sources.

![Warden](.github/assets/banner.jpg)

> **Warden** (The Gatekeeper) — The guardian of the Stargate who decides who may pass and who will be denied. Just as the Warden of Stargate guards the Stargate, Warden guards your allowlist, ensuring only authorized users can pass through.

## 📋 Overview

Warden is a lightweight HTTP API service developed in Go, primarily used for providing and managing allowlist user data (phone numbers and email addresses). The service supports fetching data from local configuration files and remote APIs, and provides multiple data merging strategies to ensure data real-time performance and reliability.

Warden can be used **standalone** or integrated with other services (such as Stargate and Herald) as part of a larger authentication architecture. For detailed architecture information, see [Architecture Documentation](docs/enUS/ARCHITECTURE.md).

## ✨ Core Features

- 🚀 **High Performance**: 5000+ requests per second with 21ms average latency
- 🔄 **Multiple Data Sources**: Local configuration files and remote APIs
- 🎯 **Flexible Strategies**: 6 data merging modes (remote-first, local-first, remote-only, local-only, etc.)
- ⏰ **Scheduled Updates**: Automatic data synchronization with Redis distributed locks
- 📦 **Containerized Deployment**: Complete Docker support, ready to use out of the box
- 🌐 **Multi-language Support**: 7 languages with automatic language detection

## 🚀 Quick Start

### Option 1: Docker (Recommended)

The fastest way to get started is using the pre-built Docker image:

```bash
# Pull the latest image
docker pull ghcr.io/soulteary/warden:latest

# Create a data file
cat > data.json <<EOF
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
EOF

# Run the container
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Tip**: For complete examples with Docker Compose, see the [Examples Directory](example/README.md).

### Option 2: From Source

1. **Clone and build**
```bash
git clone <repository-url>
cd warden
go mod download
```

2. **Create data file**
Create a `data.json` file (refer to `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

3. **Run the service**
```bash
# Run directly
go run . --api-key your-api-key-here

# Or build then run
go build -o warden .
./warden --api-key your-api-key-here
```

## ⚙️ Essential Configuration

Warden supports configuration via command line arguments, environment variables, and configuration files. The following are the most essential settings:

| Setting | Environment Variable | Description | Required |
|---------|---------------------|-------------|----------|
| Port | `PORT` | HTTP server port (default: 8081) | No |
| API Key | `API_KEY` | API authentication key (recommended for production) | Recommended |
| Redis | `REDIS` | Redis address for caching and distributed locks (e.g., `localhost:6379`) | Optional |
| Data File | `DATA_FILE` | Local data file path (default: `./data.json`) | Yes* |
| Remote Config | `CONFIG` | Remote API URL for data fetching | Optional |

\* Required if not using remote API

For complete configuration options, see [Configuration Documentation](docs/enUS/CONFIGURATION.md).

## 📡 API Usage

Warden provides a RESTful API for querying user lists, pagination, and health checks. The service supports multi-language responses via query parameter `?lang=xx` or `Accept-Language` header.

**Example**:
```bash
# Query users
curl -H "X-API-Key: your-key" "http://localhost:8081/"

# Health check
curl "http://localhost:8081/health"
```

For complete API documentation, see [API Documentation](docs/enUS/API.md) or [OpenAPI Specification](openapi.yaml).

## 📊 Performance

Based on wrk stress test (30s, 16 threads, 100 connections):
- **Requests/sec**: 5038.81
- **Average Latency**: 21.30ms
- **Max Latency**: 226.09ms

## 📚 Documentation

### Core Documentation

- **[Architecture](docs/enUS/ARCHITECTURE.md)** - Technical architecture and design decisions
- **[API Reference](docs/enUS/API.md)** - Complete API endpoint documentation
- **[Configuration](docs/enUS/CONFIGURATION.md)** - Configuration reference and examples
- **[Deployment](docs/enUS/DEPLOYMENT.md)** - Deployment guide (Docker, Kubernetes, etc.)

### Additional Resources

- **[Development Guide](docs/enUS/DEVELOPMENT.md)** - Development environment setup and contribution guide
- **[Security](docs/enUS/SECURITY.md)** - Security features and best practices
- **[SDK](docs/enUS/SDK.md)** - Go SDK usage documentation
- **[Examples](example/README.md)** - Quick start examples (basic and advanced)

## 📄 License

See the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Welcome to submit Issues and Pull Requests! See [CONTRIBUTING.md](docs/enUS/CONTRIBUTING.md) for guidelines.

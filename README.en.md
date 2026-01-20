# Warden

> 🌐 **Language / 语言**: [English](README.en.md) | [中文](README.md)

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

## 🏗️ Architecture Design

Warden uses a layered architecture design, including HTTP layer, business layer, and infrastructure layer. The system supports multiple data sources, multi-level caching, and distributed locking mechanisms.

For detailed architecture documentation, please refer to: [Architecture Design Documentation](docs/ARCHITECTURE.md)

## 📦 Installation and Running

> 💡 **Quick Start**: Want to quickly experience Warden? Check out our [Quick Start Examples](example/README.en.md):
> - [Simple Example](example/basic/README.en.md) - Basic usage, local data file only
> - [Advanced Example](example/advanced/README.en.md) - Full features, including remote API and Mock service

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
- [Configuration Documentation](docs/CONFIGURATION.md) - Learn about all configuration options
- [Deployment Documentation](docs/DEPLOYMENT.md) - Learn about deployment methods

## ⚙️ Configuration

Warden supports multiple configuration methods: command line arguments, environment variables, and configuration files. The system provides 6 data merging modes with flexible configuration strategies.

For detailed configuration documentation, please refer to: [Configuration Documentation](docs/CONFIGURATION.md)

## 📡 API Documentation

Warden provides a complete RESTful API with support for user list queries, pagination, health checks, and more. The project also provides OpenAPI 3.0 specification documentation.

For detailed API documentation, please refer to: [API Documentation](docs/API.md)

OpenAPI specification file: [openapi.yaml](openapi.yaml)

## 🐳 Docker Deployment

Warden supports complete Docker and Docker Compose deployment, ready to use out of the box.

> 🚀 **Quick Deployment**: Check the [Examples Directory](example/README.en.md) for complete Docker Compose configuration examples

For detailed deployment documentation, please refer to: [Deployment Documentation](docs/DEPLOYMENT.md)

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
├── go.mod                 # Go module definition
├── docker-compose.yml     # Docker Compose configuration
├── docker/
│   └── Dockerfile         # Docker image build file
├── example/               # Quick start examples
│   ├── README.md          # Example documentation
│   ├── basic/             # Simple example (local file only)
│   └── advanced/          # Advanced example (full features)
├── internal/
│   ├── cache/             # Redis cache and lock implementation
│   ├── cmd/               # Command line argument parsing
│   ├── define/            # Constant definitions and data structures
│   ├── logger/            # Logging initialization
│   ├── parser/            # Data parser (local/remote)
│   ├── router/            # HTTP route handling
│   └── version/           # Version information
└── pkg/
    └── gocron/            # Scheduled task scheduler
```

## 🔒 Security Features

Warden implements multiple security features, including API authentication, SSRF protection, rate limiting, TLS verification, and more.

For detailed security documentation, please refer to: [Security Documentation](docs/SECURITY.md)

## 🔧 Development Guide

> 📚 **Reference Examples**: Check the [Examples Directory](example/README.en.md) for complete example code and configurations for different usage scenarios.

For detailed development documentation, please refer to: [Development Documentation](docs/DEVELOPMENT.md)

### Code Standards

The project follows Go official code standards and best practices. For detailed standards, please refer to:

- [CODE_STYLE.en.md](CODE_STYLE.en.md) - Code style guide
- [CONTRIBUTING.en.md](CONTRIBUTING.en.md) - Contribution guide

## 📄 License

See the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Welcome to submit Issues and Pull Requests!

## 📞 Contact

For questions or suggestions, please contact via Issues.

---

**Version**: The program displays version, build time, and code version on startup (via `warden --version` or startup logs)


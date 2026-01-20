# Documentation Index

Welcome to the Warden AllowList user data service documentation.

## 🌐 Multi-language Documentation

- [English](README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 Document List

### Core Documents

- **[README.md](../../README.en.md)** - Project overview and quick start guide
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Technical architecture and design decisions

### Detailed Documents

- **[API.md](API.md)** - Complete API endpoint documentation
  - User list query endpoints
  - Pagination functionality
  - Health check endpoints
  - Error response formats

- **[CONFIGURATION.md](CONFIGURATION.md)** - Configuration reference
  - Configuration methods
  - Required configuration items
  - Optional configuration items
  - Data merging strategies
  - Configuration examples
  - Configuration best practices

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Deployment guide
  - Docker deployment (including GHCR images)
  - Docker Compose deployment
  - Local deployment
  - Production environment deployment
  - Kubernetes deployment
  - Performance optimization

- **[DEVELOPMENT.md](DEVELOPMENT.md)** - Development guide
  - Development environment setup
  - Code structure explanation
  - Testing guide
  - Contribution guide

- **[SDK.md](SDK.md)** - SDK usage documentation
  - Go SDK installation and usage
  - API interface description
  - Example code

- **[SECURITY.md](SECURITY.md)** - Security documentation
  - Security features
  - Security configuration
  - Best practices

- **[CODE_STYLE.md](CODE_STYLE.md)** - Code style guide
  - Code standards
  - Naming conventions
  - Best practices

## 🚀 Quick Navigation

### Getting Started

1. Read [README.en.md](../../README.en.md) to understand the project
2. Check the [Quick Start](../../README.en.md#quick-start) section
3. Refer to [Configuration](../../README.en.md#configuration) to configure the service

### Developers

1. Read [ARCHITECTURE.md](ARCHITECTURE.md) to understand the architecture
2. Check [API.md](API.md) to understand the API interfaces
3. Refer to [Development Guide](../../README.en.md#development-guide) for development

### Operations

1. Read [DEPLOYMENT.md](DEPLOYMENT.md) to understand deployment methods
2. Check [CONFIGURATION.md](CONFIGURATION.md) to understand configuration options
3. Refer to [Performance Optimization](DEPLOYMENT.md#performance-optimization) to optimize the service

## 📖 Document Structure

```
warden/
├── README.md              # Main project document (English)
├── README.en.md           # Main project document (English)
├── docs/
│   ├── enUS/
│   │   ├── README.md       # Documentation index (English, this file)
│   │   ├── ARCHITECTURE.md # Architecture document (English)
│   │   ├── API.md          # API document (English)
│   │   ├── CONFIGURATION.md # Configuration reference (English)
│   │   ├── DEPLOYMENT.md   # Deployment guide (English)
│   │   ├── DEVELOPMENT.md  # Development guide (English)
│   │   ├── SDK.md          # SDK document (English)
│   │   ├── SECURITY.md     # Security document (English)
│   │   └── CODE_STYLE.md   # Code style (English)
│   └── zhCN/
│       ├── README.md       # Documentation index (Chinese)
│       ├── ARCHITECTURE.md # Architecture document (Chinese)
│       ├── API.md          # API document (Chinese)
│       ├── CONFIGURATION.md # Configuration reference (Chinese)
│       ├── DEPLOYMENT.md   # Deployment guide (Chinese)
│       ├── DEVELOPMENT.md  # Development guide (Chinese)
│       ├── SDK.md          # SDK document (Chinese)
│       ├── SECURITY.md     # Security document (Chinese)
│       ├── CODE_STYLE.md   # Code style (Chinese)
│       └── CONFIG_PARSING.md # Configuration parsing (Chinese)
└── ...
```

## 🔍 Find by Topic

### Configuration Related

- Environment variable configuration: [CONFIGURATION.md](CONFIGURATION.md)
- Data merging strategies: [CONFIGURATION.md](CONFIGURATION.md)
- Configuration examples: [CONFIGURATION.md](CONFIGURATION.md)

### API Related

- API endpoint list: [API.md](API.md)
- Error handling: [API.md](API.md)
- Pagination functionality: [API.md](API.md)

### Deployment Related

- Docker deployment: [DEPLOYMENT.md#docker-deployment](DEPLOYMENT.md#docker-deployment)
- GHCR images: [DEPLOYMENT.md#using-pre-built-image-recommended](DEPLOYMENT.md#using-pre-built-image-recommended)
- Production environment: [DEPLOYMENT.md#production-environment-deployment-recommendations](DEPLOYMENT.md#production-environment-deployment-recommendations)
- Kubernetes: [DEPLOYMENT.md#kubernetes-deployment](DEPLOYMENT.md#kubernetes-deployment)

### Architecture Related

- Technology stack: [ARCHITECTURE.md](ARCHITECTURE.md)
- Project structure: [ARCHITECTURE.md](ARCHITECTURE.md)
- Core components: [ARCHITECTURE.md](ARCHITECTURE.md)

## 💡 Usage Recommendations

1. **First-time users**: Start with [README.en.md](../../README.en.md) and follow the quick start guide
2. **Configure service**: Refer to [CONFIGURATION.md](CONFIGURATION.md) to understand all configuration options
3. **Deploy service**: Check [DEPLOYMENT.md](DEPLOYMENT.md) to understand deployment methods
4. **Develop extensions**: Read [ARCHITECTURE.md](ARCHITECTURE.md) to understand the architecture design
5. **Integrate SDK**: Refer to [SDK.md](SDK.md) to learn how to use the SDK

## 📝 Document Updates

Documentation is continuously updated as the project evolves. If you find errors or need additions, please submit an Issue or Pull Request.

## 🤝 Contributing

Documentation improvements are welcome:

1. Find errors or areas that need improvement
2. Submit an Issue describing the problem
3. Or directly submit a Pull Request

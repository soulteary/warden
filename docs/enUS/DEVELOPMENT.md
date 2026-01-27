# Development Guide

> 🌐 **Language / 语言**: [English](DEVELOPMENT.md) | [中文](../zhCN/DEVELOPMENT.md) | [Français](../frFR/DEVELOPMENT.md) | [Italiano](../itIT/DEVELOPMENT.md) | [日本語](../jaJP/DEVELOPMENT.md) | [Deutsch](../deDE/DEVELOPMENT.md) | [한국어](../koKR/DEVELOPMENT.md)

This document provides a development guide for Warden project developers, including project structure, development workflow, testing methods, etc.

## Project Structure

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
│   ├── loader/             # Data loader (parser-kit)
│   ├── router/            # HTTP route handling
│   └── version/           # Version information
└── pkg/
    └── gocron/            # Scheduled task scheduler
```

## Development Environment Setup

### 1. Clone the project

```bash
git clone <repository-url>
cd warden
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Run development server

```bash
go run main.go
```

## Adding New Features

### Code Organization

1. **Core Business Logic**: In the `internal/` directory
2. **Route Handling**: In the `internal/router/` directory
3. **Data Loading Logic**: In the `internal/loader/` directory (parser-kit)
4. **Public Packages**: In the `pkg/` directory

### Development Workflow

1. Create a feature branch
2. Implement the feature and write tests
3. Run tests to ensure they pass
4. Commit code and create a Pull Request

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests and view coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Testing Best Practices

- Write unit tests for each new feature
- Maintain test coverage at a reasonable level
- Use table-driven tests
- Test boundary conditions and error cases

## Code Standards

The project follows Go official code standards and best practices. For detailed standards, please refer to:

- [CODE_STYLE.md](CODE_STYLE.md) / [CODE_STYLE.md](../zhCN/CODE_STYLE.md) - Code style guide
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guide

### Code Formatting

```bash
# Format code
go fmt ./...

# Run static analysis tools
go vet ./...

# Use golangci-lint (if installed)
golangci-lint run
```

## API Documentation

The project provides complete OpenAPI 3.0 specification documentation:

- [openapi.yaml](../openapi.yaml) - OpenAPI specification file

You can use the following tools to view:

- [Swagger Editor](https://editor.swagger.io/) - Online viewing and editing
- [Redoc](https://github.com/Redocly/redoc) - Generate beautiful documentation pages
- Postman - Import and test APIs

### Updating API Documentation

When adding or modifying API endpoints, you need to synchronously update the `openapi.yaml` file.

## Logging

The service uses structured logging to record the following information:

- **Access Logs**: HTTP request method, URL, status code, response size, duration
- **Business Logs**: Data updates, rule loading, error information
- **System Logs**: Service startup, shutdown, version information

### Log Levels

Supported log levels: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

Can be set via the `LOG_LEVEL` environment variable or the `/log/level` API endpoint.

## Reference Examples

Check the [Examples Directory](../example/README.md) / [示例目录](../example/README.md) for complete example code and configurations for different usage scenarios.

## Performance Testing

### Using wrk for Stress Testing

```bash
# Install wrk
# macOS: brew install wrk
# Linux: apt-get install wrk

# Run stress test
wrk -t16 -c100 -d30s --latency http://localhost:8081/health
```

### Performance Benchmarks

Based on wrk stress test results (30-second test, 16 threads, 100 connections):

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Average Latency: 21.30ms
Max Latency:     226.09ms
```

## Debugging

### Enable Debug Logging

```bash
export LOG_LEVEL=debug
go run main.go
```

Or set dynamically via API:

```bash
curl -X POST http://localhost:8081/log/level \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"level":"debug"}'
```

### Using Debugger

```bash
# Use Delve debugger
dlv debug main.go
```

## Building

### Local Build

```bash
go build -o warden main.go
```

### Cross Compilation

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o warden-linux-amd64 main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o warden-darwin-amd64 main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o warden-windows-amd64.exe main.go
```

## Docker Development

### Build Docker Image

```bash
docker build -f docker/Dockerfile -t warden-dev .
```

### Development with Docker Compose

```bash
docker-compose up
```

## Related Documentation

- [Architecture Design Documentation](ARCHITECTURE.md) - Understand system architecture
- [Configuration Documentation](CONFIGURATION.md) - Learn about configuration options
- [API Documentation](API.md) - Learn about API endpoints
- [Security Documentation](SECURITY.md) - Learn about security features

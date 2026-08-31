# Simple Example - Quick Start

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md)

This is the simplest Warden usage example, using only local data files, suitable for quick testing and development environments.

## 📋 Prerequisites

- Go 1.27+ or Docker
- Redis (optional, for caching and distributed locks - disabled by default in ONLY_LOCAL mode)

## 🚀 Quick Start

### Method 1: Using Go

1. **Prepare Data File**

Create a `data.json` file:

```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com"
    }
]
```

2. **Run Warden** (Redis is optional in ONLY_LOCAL mode)

```bash
# Execute in project root directory (Redis disabled by default)
go run . \
  --port 8081 \
  --mode ONLY_LOCAL

# Or set Redis address (Redis will be enabled automatically, no need for --redis-enabled)
go run . \
  --port 8081 \
  --redis localhost:6379 \
  --mode ONLY_LOCAL
```

**Note**: If you want to use Redis, start it first:
```bash
# Start Redis using Docker (simplest)
docker run -d --name redis -p 6379:6379 redis:7.4-alpine

# Or use local Redis
redis-server
```

4. **Test Service**

```bash
# Get user list (requires API Key)
curl -H "X-API-Key: your-api-key" http://localhost:8081/

# Health check (no API Key required)
curl http://localhost:8081/health
```

### Method 2: Using Docker Compose

1. **Prepare Data File**

Copy the example data file to the current directory:

```bash
cp ../../data.example.json ./data.json
```

2. **Create Environment Variable File `.env`**

```env
PORT=8081
REDIS=warden-redis:6379
MERGE_MODE=ONLY_LOCAL
API_KEY=your-secret-api-key-here

# Optional: Docker image configuration
# WARDEN_IMAGE=ghcr.io/soulteary/warden
# WARDEN_IMAGE_TAG=latest
```

3. **Start Service**

```bash
# Pull the latest image (optional, docker-compose will pull automatically)
docker-compose pull

# Start services
docker-compose up -d
```

4. **Test Service**

```bash
# Get user list
curl -H "X-API-Key: your-secret-api-key-here" http://localhost:8081/

# Health check
curl http://localhost:8081/health
```

## 📝 Configuration

### Running Mode

This example uses `ONLY_LOCAL` mode, which means:
- ✅ Only reads data from local `data.json` file
- ❌ Does not use remote API
- ⚠️  **Redis is disabled by default** (will be enabled automatically if `REDIS` address is explicitly set)
- ✅ If Redis is enabled, data is cached in Redis for improved performance

### Data File Format

The `data.json` file must be in JSON array format, each element containing:
- `phone`: Phone number (string)
- `mail`: Email address (string)

Example:
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

## 🔍 Verify Service

### 1. Check Service Status

```bash
curl http://localhost:8081/health
```

Expected response:
```json
{
    "status": "ok",
    "service": "warden"
}
```

Development/test environments also include per-check details. Production hides
them and returns only the aggregate status and service name.

### 2. Get User List

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8081/
```

Expected response:
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com"
    }
]
```

### 3. Paginated Query

```bash
curl -H "X-API-Key: your-api-key" "http://localhost:8081/?page=1&page_size=1"
```

Expected response:
```json
{
    "data": [
        {
            "phone": "13800138000",
            "mail": "admin@example.com"
        }
    ],
    "pagination": {
        "page": 1,
        "page_size": 1,
        "total": 2,
        "total_pages": 2
    }
}
```

## 🛠️ Common Questions

### Q: Why is Redis needed?

A: Warden uses Redis for:
- Data caching (improve performance)
- Distributed locks (prevent scheduled tasks from executing repeatedly)
- Multi-instance data synchronization

Even when using only local files, Redis is required.

### Q: How to modify data?

A: After modifying the `data.json` file, the service will automatically load it on the next scheduled task execution (default every 5 seconds). You can also restart the service to take effect immediately.

### Q: How to set API Key?

A: Set via environment variable:
```bash
export API_KEY=your-secret-api-key-here
go run . --port 8081 --redis localhost:6379 --mode ONLY_LOCAL
```

## 📚 Next Steps

- Check [Advanced Example](../advanced/README.md) to learn how to use remote APIs
- Read [Complete Documentation](../../README.md) to learn more features
- Check [API Documentation](../../openapi.yaml) to learn all API endpoints

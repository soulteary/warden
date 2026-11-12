# Advanced Example - Full Feature Demonstration

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md)

This is Warden's complete feature example, demonstrating all core features, including:
- Local data files
- Remote API data sources
- Redis cache and distributed locks
- Scheduled tasks for automatic synchronization
- Multiple data merging strategies
- Complete Docker Compose deployment

This isolated mock example explicitly uses `ENVIRONMENT=development` and the exact private origin `http://mock-api:8080`. This is not a production source configuration. If you change `CONFIG`, review that origin setting too; production still requires HTTPS and its encryption policy. For manual runs, use the separate private-host origin settings below; the Compose hostname does not resolve on the host.

## 📋 Prerequisites

- Docker and Docker Compose
- Or Go 1.27.4+ and Redis

## 🏗️ Architecture Overview

This example includes the following components:

```
┌─────────────────┐
│   Warden API    │  ← Main service (port 8081)
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼───┐  ┌──▼──────┐
│ Redis │  │ Mock    │  ← Mock remote API (port 8080)
│ Cache │  │ API     │
└───────┘  └─────────┘
```

## 🚀 Quick Start

### Method 1: Using Docker Compose (Recommended)

1. **Prepare Environment**

```bash
cd example/advanced
cp env.example .env
# Edit .env file, set your configuration.
# API_KEY is required: Compose refuses to start without it. env.example ships the
# public demo key used by the commands below; replace it outside local testing.
```

2. **Start All Services**

```bash
# Pull the latest image from GHCR (optional, docker-compose will pull automatically)
docker-compose pull

# Start all services
docker-compose up -d
```

This will start:
- Warden main service (port 8081) - using `ghcr.io/soulteary/warden:latest` by default
- Redis cache service (port 6379)
- Mock remote API service (port 8080)

**Note**: The example uses the pre-built image from GitHub Container Registry (GHCR). You can customize the image by setting `WARDEN_IMAGE` and `WARDEN_IMAGE_TAG` in your `.env` file.

3. **Check Service Status**

```bash
# View all service logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f warden
docker-compose logs -f mock-api
```

4. **Test Service**

```bash
# Health check
curl http://localhost:8081/health

# Get user list (requires API Key)
curl -H "X-API-Key: your-secret-api-key-here" http://localhost:8081/

# View Prometheus metrics
curl http://localhost:8081/metrics
```

### Method 2: Local Running

1. **Start Redis**

```bash
docker run -d --name redis -p 6379:6379 redis:7.4-alpine
```

2. **Start Mock API Service**

```bash
cd example/advanced/mock-api
cp ../../../data.example.json data.json
PORT=8080 go run .
```

Mock API will serve at `http://localhost:8080/api/users`.

3. **Run Warden**

```bash
# In project root directory. Set this to an actual RFC1918 address on this host.
# Loopback (localhost/127.0.0.1) is rejected even in development.
export MOCK_API_HOST=192.168.1.10 # replace with your host address
ENVIRONMENT=development \
REMOTE_PRIVATE_ORIGINS="http://$MOCK_API_HOST:8080" \
API_KEY=your-secret-api-key-here go run . \
  --data-file ./data.example.json \
  --port 8081 \
  --redis localhost:6379 \
  --config "http://$MOCK_API_HOST:8080/api/users" \
  --key "Bearer mock-token" \
  --mode DEFAULT \
  --interval 10
```

### Test script and host ports

Compose keeps Warden listening on container port 8081. `PORT` selects the published host port; `MOCK_API_PORT` selects the mock's host port. Both are published on `127.0.0.1` only. The commands above use the default ports and demo API key. Adjust client URLs and credentials when changing them.

`test.sh` reads exported variables, not `.env`, and exits if `API_KEY` is not set. Pass values matching your Compose configuration explicitly:

```bash
cd example/advanced # from the project root
API_KEY=your-secret-api-key-here PORT=8081 MOCK_API_PORT=8080 ./test.sh
```

`BASE_URL` and `MOCK_API_BASE_URL` can override the complete observer URLs.

## 📝 Configuration

### Data Merging Strategy

This example demonstrates `DEFAULT` (remote-first) mode:

- ✅ Prioritize fetching data from remote API
- ✅ Use local data to supplement when remote data doesn't exist
- ✅ Scheduled tasks automatically synchronize every 10 seconds

### Environment Variable Configuration

Edit `.env` file:

```env
# Service Port
PORT=8081

# Docker Image Configuration (Optional)
# WARDEN_IMAGE=ghcr.io/soulteary/warden
# WARDEN_IMAGE_TAG=latest

# Redis Configuration
REDIS=warden-redis:6379
REDIS_PASSWORD=

# Remote API Configuration
CONFIG=http://mock-api:8080/api/users
KEY=Bearer mock-token

# Task Configuration
INTERVAL=10

# Running Mode
MERGE_MODE=DEFAULT

# API Authentication
API_KEY=your-secret-api-key-here

# HTTP Client Configuration
HTTP_TIMEOUT=5
HTTP_MAX_IDLE_CONNS=100
```

### Data Files

**Local Data File** (`data.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**Remote API Data** (provided by Mock API):
```json
[
    {
        "phone": "13900139000",
        "mail": "remote@example.com"
    },
    {
        "phone": "15000150000",
        "mail": "user@example.com"
    }
]
```

**Merged Result** (remote-first):
```json
[
    {
        "phone": "13900139000",
        "mail": "remote@example.com"
    },
    {
        "phone": "15000150000",
        "mail": "user@example.com"
    },
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

## 🔍 Feature Demonstration

### 1. Data Synchronization Flow

Observe how scheduled tasks automatically synchronize data:

```bash
# View Warden logs
docker-compose logs -f warden

# You will see output like:
# INFO Loaded data from remote API ✓ count=2
# INFO Background data update 📦 count=3 duration=0.123
```

### 2. Modify Remote Data

Modify Mock API's data file and observe automatic synchronization:

```bash
# Edit Mock API data
vim mock-api/data.json

# Wait 10 seconds (scheduled task interval), data will automatically update
```

### 3. Test Different Merging Modes

Modify `MERGE_MODE` in `.env` to test different modes (`MODE` is deprecated):

- `DEFAULT`: Historical remote-first behavior with tolerant fallback
- `REMOTE_FIRST`: Strict remote-first; failed remote refresh keeps the last-known-good snapshot
- `REMOTE_FIRST_ALLOW_REMOTE_FAILED`: Remote-first with a degraded local fallback
- `LOCAL_FIRST`: Local-first
- `ONLY_REMOTE`: Remote-only
- `ONLY_LOCAL`: Local-only

```bash
# Recreate the service with the updated .env configuration
docker-compose up -d warden
```

### 4. View Monitoring Metrics

```bash
# Prometheus metrics
curl http://localhost:8081/metrics | grep warden

# Health check (details are visible outside production)
curl http://localhost:8081/health | jq
```

### 5. Test API Functionality

```bash
# Get all users
curl -H "X-API-Key: your-secret-api-key-here" http://localhost:8081/

# Paginated query
curl -H "X-API-Key: your-secret-api-key-here" \
  "http://localhost:8081/?page=1&page_size=10"

# Dynamically adjust log level
curl -X POST -H "X-API-Key: your-secret-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{"level":"debug"}' \
  http://localhost:8081/log/level
```

## 🧪 Test Scenarios

### Scenario 1: Remote API Failure

1. Stop Mock API service:
```bash
docker-compose stop mock-api
```

2. Observe Warden automatically fallback to local data:
```bash
docker-compose logs -f warden
# Should see: Loaded data from local file
```

3. Restore Mock API:
```bash
docker-compose start mock-api
```

4. Observe automatic recovery:
```bash
# Wait for scheduled task execution, data will recover from remote
```

### Scenario 2: Redis Failure

1. Stop Redis:
```bash
docker-compose stop warden-redis
```

2. Observe service behavior:
```bash
# Warden will continue running, but cannot use Redis cache
# Distributed locks for scheduled tasks will fail (multi-instance scenario)
```

### Scenario 3: Data Conflict Test

1. Modify local and remote data to have overlaps:
   - Local: `phone: 13800138000`
   - Remote: `phone: 13800138000` (different email)

2. Observe merge result (depends on selected mode)

## 📊 Performance Testing

Use `wrk` for stress testing:

```bash
# Install wrk
# macOS: brew install wrk
# Linux: apt-get install wrk

# Run test
wrk -t4 -c100 -d30s \
  -H "X-API-Key: your-secret-api-key-here" \
  http://localhost:8081/
```

Expected results:
- Request rate: 5000+ req/s
- Average latency: < 25ms

## 🛠️ Troubleshooting

### Issue 1: Cannot Connect to Remote API

**Symptoms**: Logs show "Remote API load failed"

**Solution**:
1. Check if Mock API is running: `docker-compose ps`
2. Check network connection: `curl http://localhost:8080/api/users`
3. Check authentication header: Ensure `KEY` configuration is correct

### Issue 2: Redis Connection Failed

**Symptoms**: Shows "Redis connection failed" on startup

**Solution**:
1. Check if Redis is running: `docker-compose ps warden-redis`
2. Check Redis password configuration
3. Check Redis inside its Compose container (Redis is not published on host port 6379):

```bash
docker-compose exec warden-redis sh -c 'if [ -n "$REDIS_PASSWORD" ]; then REDISCLI_AUTH="$REDIS_PASSWORD" redis-cli ping; else redis-cli ping; fi'
```

### Issue 3: Data Not Updated

**Symptoms**: API returns old data after modifying data

**Solution**:
1. Check scheduled task interval configuration (`INTERVAL`)
2. Check logs to confirm if scheduled tasks are executing
3. Manually trigger: Restart service or wait for next scheduled task cycle

## 📚 Next Steps

- Read [Complete Documentation](../../README.md) to learn all features
- Check [API Documentation](../../openapi.yaml) to learn API details
- Refer to [Simple Example](../basic/README.md) to learn basic usage
- Check [Configuration Example](../../config.example.yaml) to learn all configuration options

## 🔗 Related Resources

- [Warden Main Documentation](../../README.md)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Redis Documentation](https://redis.io/docs/)

# Deployment Documentation

> 🌐 **Language / 语言**: [English](DEPLOYMENT.md) | [中文](../zhCN/DEPLOYMENT.md) | [Français](../frFR/DEPLOYMENT.md) | [Italiano](../itIT/DEPLOYMENT.md) | [日本語](../jaJP/DEPLOYMENT.md) | [Deutsch](../deDE/DEPLOYMENT.md) | [한국어](../koKR/DEPLOYMENT.md)

This document explains how to deploy the Warden service, including Docker deployment, local deployment, etc.

## Prerequisites

- Go 1.25+ (refer to [go.mod](../go.mod))
- Redis (for distributed locks and caching)
- Docker (optional, for containerized deployment)

## Docker Deployment

> 🚀 **Quick Deployment**: Check the [Examples Directory](../example/README.md) / [示例目录](../example/README.md) for complete Docker Compose configuration examples:
> - [Simple Example](../example/basic/docker-compose.yml) / [简单示例](../example/basic/docker-compose.yml) - Basic Docker Compose configuration
> - [Advanced Example](../example/advanced/docker-compose.yml) / [复杂示例](../example/advanced/docker-compose.yml) - Complete configuration including Mock API

### Using Pre-built Image (Recommended)

Warden provides pre-built Docker images that can be pulled directly from GitHub Container Registry (GHCR), no manual build required:

```bash
# Pull the latest version image
docker pull ghcr.io/soulteary/warden:latest

# Run container
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api/data.json \
  -e KEY="Bearer your-token-here" \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **Tip**: Using pre-built images allows you to get started quickly without a local build environment. Images are automatically updated to ensure you're using the latest version.

### Using Docker Compose

1. **Prepare environment variable file**
   
   If a `.env.example` file exists in the project root directory, you can copy it:
   ```bash
   cp .env.example .env
   ```
   
   If the `.env.example` file doesn't exist, you can manually create a `.env` file with the following content:
   ```env
   # Server Configuration
   PORT=8081
   
   # Redis Configuration
   REDIS=warden-redis:6379
   # Redis password (optional, recommend using environment variables instead of config file)
   # REDIS_PASSWORD=your-redis-password
   # Or use password file (more secure)
   # REDIS_PASSWORD_FILE=/path/to/redis-password.txt
   
   # Remote Data API
   CONFIG=http://example.com/api/data.json
   # Remote configuration API authentication key
   KEY=Bearer your-token-here
   
   # Task Configuration
   INTERVAL=5
   
   # Application Mode
   MODE=DEFAULT
   
   # HTTP Client Configuration (optional)
   # HTTP_TIMEOUT=5
   # HTTP_MAX_IDLE_CONNS=100
   # HTTP_INSECURE_TLS=false
   
   # API Key (for API authentication, required in production)
   API_KEY=your-api-key-here
   
   # Health Check IP Whitelist (optional, comma-separated)
   # HEALTH_CHECK_IP_WHITELIST=127.0.0.1,::1,10.0.0.0/8
   
   # Trusted Proxy IP List (optional, comma-separated, for reverse proxy environments)
   # TRUSTED_PROXY_IPS=127.0.0.1,10.0.0.1
   
   # Log Level (optional)
   # LOG_LEVEL=info
   ```
   
   > ⚠️ **Security Note**: The `.env` file contains sensitive information. Do not commit it to version control. The `.env` file is already ignored by `.gitignore`. Please use the above content as a template to create the `.env` file.

2. **Start the service**
```bash
docker-compose up -d
```

### Manual Image Build

```bash
docker build -f docker/Dockerfile -t warden-release .
```

### Run Container

```bash
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e PORT=8081 \
  -e REDIS=localhost:6379 \
  -e CONFIG=http://example.com/api \
  -e KEY="Bearer token" \
  warden-release
```

## Local Deployment

### 1. Clone the project

```bash
git clone <repository-url>
cd warden
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Configure local data file

Create a `data.json` file (refer to `data.example.json`):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**Note**: The `data.json` file supports the following fields:
- `phone` (required): User phone number
- `mail` (required): User email address
- `user_id` (optional): User unique identifier, auto-generated if not provided
- `status` (optional): User status, such as "active", "inactive", "suspended", defaults to "active"
- `scope` (optional): User permission scope array, such as `["read", "write"]`
- `role` (optional): User role, such as "admin", "user"

For a complete example, please refer to the `data.example.json` file.

### 4. Run the service

```bash
go run .
```

## Production Environment Deployment Recommendations

### 1. Use Reverse Proxy

It is recommended to use a reverse proxy such as Nginx or Traefik in production:

**Nginx Configuration Example**:
```nginx
upstream warden {
    server localhost:8081;
}

server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://warden;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. Use HTTPS

Production environments must use HTTPS. This can be achieved by:

- Using Let's Encrypt free certificates
- Using a reverse proxy (such as Nginx) to handle SSL/TLS
- Configuring the `TRUSTED_PROXY_IPS` environment variable to correctly obtain client real IP

### 3. Configure Monitoring

- Use Prometheus to collect metrics (via `/metrics` endpoint)
- Configure health checks (via `/health` endpoint)
- Set up log collection and analysis

### 4. High Availability Deployment

- Deploy multiple instances, use load balancer to distribute requests
- Use shared Redis instance to ensure data consistency
- Configure automatic restart and failover

### 5. Resource Limits

Configure resource limits in Docker Compose or Kubernetes:

```yaml
services:
  warden:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Kubernetes Deployment

### Basic Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: warden
spec:
  replicas: 3
  selector:
    matchLabels:
      app: warden
  template:
    metadata:
      labels:
        app: warden
    spec:
      containers:
      - name: warden
        image: warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "redis-service:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: warden-service
spec:
  selector:
    app: warden
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8081
  type: LoadBalancer
```

## Performance Optimization

### 1. Redis Configuration

- Use Redis persistence (RDB or AOF)
- Configure appropriate Redis memory limits
- Use Redis cluster (if needed)

### 2. Application Configuration

- Adjust `HTTP_MAX_IDLE_CONNS` to optimize connection pool
- Configure appropriate `INTERVAL` to balance real-time performance and efficiency
- Use appropriate running mode (`MODE`)

### 3. Monitoring and Tuning

Based on wrk stress test results (30-second test, 16 threads, 100 connections):

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Average Latency: 21.30ms
Max Latency:     226.09ms
```

Adjust configuration parameters based on actual load.

## Optional Integration Deployment (with Stargate/Herald)

Warden can be deployed and used standalone, or optionally integrated with Stargate and Herald. The following are optional integration deployment configuration examples.

**Note**: The following integration deployment scenarios are optional, and Warden can be deployed and used completely independently.

### Docker Compose Integration Example

Complete Stargate + Warden + Herald integration deployment configuration:

```yaml
version: '3.8'

services:
  # Warden Service
  warden:
    image: ghcr.io/soulteary/warden:latest
    container_name: warden
    ports:
      - "8081:8081"
    networks:
      - auth-network
    environment:
      - PORT=8081
      - REDIS=warden-redis:6379
      - API_KEY=${WARDEN_API_KEY}
      - MODE=DEFAULT
      # Inter-service authentication configuration (HMAC example)
      - WARDEN_HMAC_KEYS=${WARDEN_HMAC_KEYS}
      - WARDEN_HMAC_TIMESTAMP_TOLERANCE=60
    volumes:
      - ./warden-data.json:/app/data.json:ro
    healthcheck:
      test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
      interval: 10s
      timeout: 1s
      retries: 3
    depends_on:
      - warden-redis

  # Warden Redis
  warden-redis:
    image: redis:6.2.4
    container_name: warden-redis
    networks:
      - auth-network
    volumes:
      - warden-redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 1s
      retries: 3

  # Stargate Service (example configuration)
  stargate:
    image: ghcr.io/soulteary/stargate:latest
    container_name: stargate
    ports:
      - "8080:8080"
    networks:
      - auth-network
    environment:
      - STARGATE_WARDEN_BASE_URL=http://warden:8081
      - STARGATE_WARDEN_AUTH_TYPE=hmac
      - STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
      - STARGATE_WARDEN_HMAC_SECRET=${WARDEN_HMAC_SECRET}
      - STARGATE_HERALD_BASE_URL=http://herald:8082
    depends_on:
      - warden
      - herald

  # Herald Service (example configuration)
  herald:
    image: ghcr.io/soulteary/herald:latest
    container_name: herald
    ports:
      - "8082:8082"
    networks:
      - auth-network
    environment:
      - HERALD_REDIS_URL=redis://herald-redis:6379
    depends_on:
      - herald-redis

  # Herald Redis
  herald-redis:
    image: redis:6.2.4
    container_name: herald-redis
    networks:
      - auth-network
    volumes:
      - herald-redis-data:/data

networks:
  auth-network:
    driver: bridge

volumes:
  warden-redis-data:
  herald-redis-data:
```

### Environment Variable Configuration

Create `.env` file:

```bash
# Warden API Key
WARDEN_API_KEY=your-warden-api-key-here

# Warden HMAC keys (JSON format)
WARDEN_HMAC_KEYS='{"key-id-1":"your-hmac-secret-key-1"}'

# HMAC secret used by Stargate (corresponds to key in WARDEN_HMAC_KEYS)
WARDEN_HMAC_SECRET=your-hmac-secret-key-1
```

### Network Configuration

All services should be in the same Docker network for mutual communication:

- **Warden**: Listens on port `8081`, called by Stargate
- **Stargate**: Listens on port `8080`, serves as Traefik forwardAuth service
- **Herald**: Listens on port `8082`, called by Stargate

### Service Dependencies

- **Stargate** depends on **Warden** and **Herald**
- **Warden** depends on **warden-redis** (optional, if Redis is enabled)
- **Herald** depends on **herald-redis**

### Health Checks

All services should configure health checks to ensure normal operation:

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
  interval: 10s
  timeout: 1s
  retries: 3
```

### Production Environment Recommendations

1. **Use Independent Redis Instances**: Warden and Herald should use independent Redis instances to avoid data conflicts
2. **Configure Inter-Service Authentication**: Production environment must configure mTLS or HMAC signature
3. **Use Key Management Services**: Use HashiCorp Vault or similar services to manage keys and certificates
4. **Network Isolation**: Use Docker network policies to restrict inter-service access
5. **Monitoring and Logging**: Configure unified monitoring and log collection systems

### Kubernetes Integration Deployment

When deploying in Kubernetes, it is recommended to:

1. **Use Services**: Create Kubernetes Services for each service
2. **Use ConfigMap and Secret**: Store configuration and keys
3. **Use NetworkPolicy**: Restrict inter-service network access
4. **Use Ingress**: Configure Traefik Ingress to route to Stargate

Example Kubernetes configuration:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: warden
spec:
  selector:
    app: warden
  ports:
    - port: 8081
      targetPort: 8081
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: warden
spec:
  replicas: 3
  selector:
    matchLabels:
      app: warden
  template:
    metadata:
      labels:
        app: warden
    spec:
      containers:
      - name: warden
        image: ghcr.io/soulteary/warden:latest
        ports:
        - containerPort: 8081
        env:
        - name: PORT
          value: "8081"
        - name: REDIS
          value: "warden-redis:6379"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: api-key
        - name: WARDEN_HMAC_KEYS
          valueFrom:
            secretKeyRef:
              name: warden-secrets
              key: hmac-keys
```

## Related Documentation

- [Configuration Documentation](CONFIGURATION.md) - Learn about detailed configuration options
- [Security Documentation](SECURITY.md) - Learn about security configuration and best practices
- [Architecture Design Documentation](ARCHITECTURE.md) - Understand system architecture
- [API Documentation](API.md) - Learn about API interfaces and integration examples

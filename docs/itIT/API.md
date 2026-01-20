# API Documentation

> 🌐 **Language / 语言**: [English](../enUS/API.md) | [中文](../zhCN/API.md) | [Français](../frFR/API.md) | [Italiano](API.md) | [日本語](../jaJP/API.md) | [Deutsch](../deDE/API.md) | [한국어](../koKR/API.md)

This document provides detailed information about all API endpoints provided by Warden.

## OpenAPI Documentation

The project provides complete OpenAPI 3.0 specification documentation in the `openapi.yaml` file.

You can use the following tools to view and test the API:

1. **Swagger UI**: Open the `openapi.yaml` file using [Swagger Editor](https://editor.swagger.io/)
2. **Postman**: Import the `openapi.yaml` file into Postman
3. **Redoc**: Use Redoc to generate a beautiful API documentation page

## Authentication

Some API endpoints require API Key authentication. You can provide authentication information in two ways:

1. **X-API-Key Header**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer Header**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

The API Key can be configured via the `API_KEY` environment variable or the `--api-key` command line argument.

## API Endpoints

### Get User List

Get all users or paginated user list.

**Request**
```http
GET /
X-API-Key: your-secret-api-key

GET /?page=1&page_size=100
X-API-Key: your-secret-api-key
```

**Query Parameters**:
- `page` (optional): Page number, starting from 1, defaults to 1
- `page_size` (optional): Number of items per page, defaults to all data (no pagination)

**Note**: This endpoint requires API Key authentication.

**Response (no pagination)**
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

**Response (with pagination)**
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
        "page_size": 100,
        "total": 200,
        "total_pages": 2
    }
}
```

**Status Code**: `200 OK`

**Content-Type**: `application/json`

### Get Single User

Query a single user by phone number, email, or user ID.

**Request**
```http
GET /user?phone=13800138000
X-API-Key: your-secret-api-key

GET /user?mail=admin@example.com
X-API-Key: your-secret-api-key

GET /user?user_id=user-123
X-API-Key: your-secret-api-key
```

**Query Parameters** (must provide exactly one):
- `phone`: User phone number
- `mail`: User email address
- `user_id`: User unique identifier

**Note**: 
- This endpoint requires API Key authentication
- Only one query parameter (`phone`, `mail`, or `user_id`) is allowed

**Response (user exists)**
```json
{
    "phone": "13800138000",
    "mail": "admin@example.com",
    "user_id": "user-123",
    "status": "active",
    "scope": ["read", "write"],
    "role": "admin"
}
```

**Field Descriptions**:
- `phone`: User phone number
- `mail`: User email address
- `user_id`: User unique identifier (auto-generated if not provided)
- `status`: User status, possible values:
  - `"active"`: Active status, user can login and access the system
  - `"inactive"`: Inactive status, user cannot login
  - `"suspended"`: Suspended status, user cannot login
  - Defaults to `"active"` if not set
- `scope`: User permission scope array (optional), used for fine-grained authorization, e.g., `["read", "write", "admin"]`
- `role`: User role (optional), e.g., `"admin"`, `"user"`, `"guest"`

**Notes**:
- Only users with `status` of `"active"` can pass authentication checks
- `scope` and `role` fields are used by Stargate to set authorization headers (`X-Auth-Scopes` and `X-Auth-Role`) for downstream services

**Response (user not found)**
- **Status Code**: `404 Not Found`
- **Response Body**: `User not found`

**Error Response (missing parameter)**
- **Status Code**: `400 Bad Request`
- **Response Body**: `Bad Request: missing identifier (phone, mail, or user_id)`

**Error Response (multiple parameters)**
- **Status Code**: `400 Bad Request`
- **Response Body**: `Bad Request: only one identifier allowed (phone, mail, or user_id)`

### Health Check

Check service health status, including Redis connection status, data loading status, etc.

**Request**
```http
GET /health
GET /healthcheck
```

**Note**: This endpoint does not require authentication, but access IPs can be restricted via the `HEALTH_CHECK_IP_WHITELIST` environment variable.

**Response**
```json
{
    "status": "ok",
    "details": {
        "redis": "ok",
        "data_loaded": true,
        "user_count": 100
    },
    "mode": "DEFAULT"
}
```

**Status Code**: `200 OK`

**Response Field Descriptions**:
- `status`: Service status, `"ok"` indicates normal
- `details.redis`: Redis connection status, possible values:
  - `"ok"`: Redis is normal
  - `"unavailable"`: Redis connection failed (fallback mode) or Redis client is nil
  - `"disabled"`: Redis is explicitly disabled
- `details.data_loaded`: Whether data has been loaded
- `details.user_count`: Current user count
- `mode`: Current running mode

### Log Level Management

Dynamically get and set log levels.

#### Get Current Log Level

**Request**
```http
GET /log/level
X-API-Key: your-secret-api-key
```

**Response**
```json
{
    "level": "info"
}
```

**Note**: This endpoint requires API Key authentication.

#### Set Log Level

**Request**
```http
POST /log/level
Content-Type: application/json
X-API-Key: your-secret-api-key

{
    "level": "debug"
}
```

**Request Body**:
```json
{
    "level": "debug"
}
```

**Supported Log Levels**: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

**Response**
```json
{
    "level": "debug",
    "message": "Log level updated successfully"
}
```

**Note**: 
- This endpoint requires API Key authentication
- All log level modification operations are recorded in security audit logs

### Prometheus Metrics

Get Prometheus format monitoring metrics data.

**Request**
```http
GET /metrics
```

**Response**: Prometheus format metrics data

**Note**: This endpoint does not require authentication.

**Example Response**:
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/",status="200"} 1234

# HELP http_request_duration_seconds HTTP request duration in seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/",le="0.005"} 1000
http_request_duration_seconds_bucket{method="GET",path="/",le="0.01"} 1200
...
```

## Error Responses

All API endpoints may return the following error responses:

### 401 Unauthorized

Returned when API Key authentication fails:

```json
{
    "error": "Unauthorized",
    "message": "Invalid or missing API key"
}
```

### 429 Too Many Requests

Returned when requests exceed rate limit:

```json
{
    "error": "Too Many Requests",
    "message": "Rate limit exceeded"
}
```

### 500 Internal Server Error

Returned when server internal error occurs:

```json
{
    "error": "Internal Server Error",
    "message": "An internal error occurred"
}
```

In production mode, detailed error information is hidden to prevent information leakage.

## Rate Limiting

By default, API requests are protected by rate limiting:

- **Limit**: 60 requests per minute
- **Window**: 1 minute
- **Exceeded**: Returns `429 Too Many Requests`

Rate limiting can be adjusted via configuration file:

```yaml
rate_limit:
  rate: 60  # Requests per minute
  window: 1m
```

## IP Whitelist

IP whitelists can be configured via the following environment variables:

- `IP_WHITELIST`: Global IP whitelist (restricts access to all endpoints)
- `HEALTH_CHECK_IP_WHITELIST`: Health check endpoint IP whitelist (only restricts `/health` and `/healthcheck`)

Supports CIDR range format, multiple IPs or ranges separated by commas:

```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## Response Compression

All API responses support automatic compression (gzip). Clients can enable compression via the `Accept-Encoding: gzip` request header.

## Related Documentation

- [OpenAPI Specification](../openapi.yaml) - Complete OpenAPI 3.0 specification
- [Configuration Documentation](CONFIGURATION.md) - Learn how to configure API Key and other options
- [Security Documentation](SECURITY.md) - Learn about security features and best practices

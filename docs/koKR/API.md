# API 문서

> 🌐 **Language / 语言**: [English](../enUS/API.md) | [中文](../zhCN/API.md) | [Français](../frFR/API.md) | [Italiano](../itIT/API.md) | [日本語](../jaJP/API.md) | [Deutsch](../deDE/API.md) | [한국어](API.md)

이 문서는 Warden이 제공하는 모든 API 엔드포인트에 대한 자세한 정보를 담고 있습니다.

## OpenAPI 문서

이 프로젝트는 `openapi.yaml` 파일에 완전한 OpenAPI 3.0 명세를 제공합니다.

다음 도구로 API를 확인하고 테스트할 수 있습니다.

1. **Swagger UI**: [Swagger Editor](https://editor.swagger.io/)로 `openapi.yaml` 파일 열기
2. **Postman**: `openapi.yaml` 파일을 Postman으로 가져오기
3. **Redoc**: Redoc으로 보기 좋은 API 문서 페이지 생성하기

## 인증

일부 API 엔드포인트는 API 키 인증을 요구합니다. 인증 정보는 다음 두 가지 방법으로 전달할 수 있습니다.

1. **X-API-Key 헤더**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer 헤더**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

API 키는 환경 변수 `API_KEY` 또는 명령줄 인자 `--api-key`로 설정할 수 있습니다.

## 라우팅 계약

아래에 문서화된 엔드포인트가 Warden이 제공하는 전체 경로 집합입니다. 그 밖의 모든 경로는
사용자 데이터를 포함하지 않는 JSON 본문과 함께 `404 Not Found`를 반환합니다.

```http
GET /not-a-route
X-API-Key: your-secret-api-key
```

```json
{
  "error": "Requested resource does not exist"
}
```

> **동작 변경**: 루트 경로 `/`는 이전에 하위 트리 패턴으로 등록되어 있었기 때문에, 일치하지
> 않는 모든 경로(`/foo`, `/user/`, `/v1/`)를 사용자 목록 핸들러가 처리하여 **허용 목록
> 전체**를 반환했습니다. 이제 `/`는 정확히 일치하는 경로이며, 일치하지 않는 경로에는 위의
> 404가 반환됩니다. 임의의 경로가 사용자 데이터를 반환하는 데 의존하던 클라이언트는 `/`,
> `/data.json`, `/v1/users` 중 하나를 사용해야 합니다.

Go 라우터는 일치 처리 전에 경로를 정규화하므로, `/metrics/../user`는 `/user`로 해석되어
그쪽으로 리디렉션되며 404 핸들러에 도달하지 않습니다.

## API 엔드포인트

### 사용자 목록 조회

전체 사용자 또는 페이지 단위로 나뉜 사용자 목록을 조회합니다.

**요청**
```http
GET /
X-API-Key: your-secret-api-key

GET /?page=1&page_size=100
X-API-Key: your-secret-api-key
```

**쿼리 매개변수**:
- `page`(선택): 페이지 번호. 1부터 시작하며 기본값은 1
- `page_size`(선택): 페이지당 항목 수. 기본값은 전체 데이터(페이지 나눔 없음)

**참고**: 이 엔드포인트는 API 키 인증이 필요합니다.

**응답(페이지 나눔 없음)**
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

**응답(페이지 나눔 있음)**
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

**상태 코드**: `200 OK`

**Content-Type**: `application/json`

### 단일 사용자 조회

전화번호, 이메일 주소 또는 사용자 ID로 단일 사용자를 조회합니다.

**요청**
```http
GET /user?phone=13800138000
X-API-Key: your-secret-api-key

GET /user?mail=admin@example.com
X-API-Key: your-secret-api-key

GET /user?user_id=user-123
X-API-Key: your-secret-api-key
```

**쿼리 매개변수**(정확히 하나만 제공해야 함):
- `phone`: 사용자 전화번호
- `mail`: 사용자 이메일 주소
- `user_id`: 사용자 고유 식별자

**참고**:
- 이 엔드포인트는 API 키 인증이 필요합니다
- 쿼리 매개변수(`phone`, `mail`, `user_id`)는 하나만 허용됩니다

**응답(사용자가 존재하는 경우)**
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

**필드 설명**:
- `phone`: 사용자 전화번호
- `mail`: 사용자 이메일 주소
- `user_id`: 사용자 고유 식별자(제공하지 않으면 자동 생성)
- `status`: 사용자 상태. 가능한 값:
  - `"active"`: 활성 상태. 사용자가 로그인하여 시스템에 접근할 수 있음
  - `"inactive"`: 비활성 상태. 사용자가 로그인할 수 없음
  - `"suspended"`: 정지 상태. 사용자가 로그인할 수 없음
  - 설정하지 않으면 기본값은 `"inactive"`이며, 로그인을 허용하려면 `"active"`를 명시적으로 설정해야 합니다
- `scope`: 사용자 권한 범위 배열(선택). 세분화된 인가에 사용하며 예: `["read", "write", "admin"]`
- `role`: 사용자 역할(선택). 예: `"admin"`, `"user"`, `"guest"`

**참고 사항**:
- `status`가 `"active"`인 사용자만 인증 검사를 통과합니다
- `scope`와 `role` 필드는 Stargate가 다운스트림 서비스용 인가 헤더(`X-Auth-Scopes`, `X-Auth-Role`)를 설정하는 데 사용합니다

**선택적 연동 시나리오**:
다른 서비스(예: Stargate)와 연동하기로 했다면, 로그인 흐름에서 이 엔드포인트를 호출하여 사용자 정보를 조회할 수 있습니다.
1. 사용자가 식별자(이메일/전화번호/사용자명)를 입력하면 `GET /user?phone=xxx` 또는 `GET /user?mail=xxx`를 호출합니다
2. Warden이 사용자 정보(`user_id`, `email`, `phone`, `status` 포함)를 반환합니다
3. 사용자가 존재하고 상태가 `"active"`이면 이후 인증 흐름을 계속 진행할 수 있습니다
4. 반환된 `scope`와 `role`은 인가 헤더 설정에 사용할 수 있습니다

**응답(사용자를 찾을 수 없는 경우)**
- **상태 코드**: `404 Not Found`
- **응답 본문**: `User not found`

**오류 응답(매개변수 누락)**
- **상태 코드**: `400 Bad Request`
- **응답 본문**: `Bad Request: missing identifier (phone, mail, or user_id)`

**오류 응답(매개변수 중복)**
- **상태 코드**: `400 Bad Request`
- **응답 본문**: `Bad Request: only one identifier allowed (phone, mail, or user_id)`

### 상태 확인

Redis, 데이터 캐시, 스냅샷 출처 및 스냅샷 신선도를 확인합니다.

**요청**
```http
GET /health
GET /healthcheck
```

**참고**: 이 엔드포인트는 인증이 필요하지 않지만, 접근 IP는 환경 변수 `HEALTH_CHECK_IP_WHITELIST`로 제한할 수 있습니다. 운영 환경 응답은 개별 검사 결과를 숨깁니다.

**응답**
```json
{
    "status": "ok",
    "service": "warden",
    "checks": {
        "redis": {
            "name": "redis",
            "status": "ok",
            "latency_ms": 1,
            "timestamp": "2026-08-31T00:00:00Z"
        },
        "snapshot": {
            "name": "snapshot",
            "status": "ok",
            "latency_ms": 0,
            "timestamp": "2026-08-31T00:00:00Z",
            "metadata": {
                "source": "merged",
                "version": "a1b2c3d4",
                "age_seconds": 2.5
            }
        }
    },
    "timestamp": "2026-08-31T00:00:00Z",
    "total_latency_ms": 1
}
```

운영 환경 응답:

```json
{"status":"ok","service":"warden"}
```

**상태 코드**:

- `200 OK`: 종합 상태가 `ok` 또는 `degraded`입니다. `degraded`는 서비스가 여전히 동작 가능함을 의미합니다.
- `503 Service Unavailable`: 중요한 검사가 실패했습니다.
- `403 Forbidden`: 클라이언트가 `HEALTH_CHECK_IP_WHITELIST` 범위 밖에 있습니다.

**응답 필드 설명**:
- `status`: `ok`, `degraded`, `unhealthy` 중 하나입니다.
- `service`: 서비스 이름(`warden`).
- `checks`: 개발/테스트 환경에서만 제공되는 맵으로 `redis`, `data`, `snapshot`, `snapshot_freshness` 결과를 담습니다.
- `checks.snapshot.metadata`: 카디널리티가 낮은 출처/버전/경과 시간과 안정적인 갱신 사유 코드입니다. 원본 원격 오류, URL, 자격 증명은 결코 노출되지 않습니다.
- `timestamp`, `total_latency_ms`: 개발/테스트 환경에서만 제공되는 종합 소요 시간 필드입니다.

`REMOTE_FIRST`와 `ONLY_REMOTE`에서는 `snapshot_freshness`가 중요한 항목입니다. 출처를 알 수
없거나 경과 시간이 `SNAPSHOT_MAX_AGE`를 넘으면 503을 반환합니다. 관대한 모드에서는 검증된
로컬 데이터나 마지막으로 유효했던 스냅샷을 `degraded` 상태로 HTTP 200과 함께 제공할 수
있습니다.

### 로그 수준 관리

로그 수준을 동적으로 조회하고 설정합니다.

#### 현재 로그 수준 조회

**요청**
```http
GET /log/level
X-API-Key: your-secret-api-key
```

**응답**
```json
{
    "level": "info"
}
```

**참고**: 이 엔드포인트는 API 키 인증이 필요합니다.

#### 로그 수준 설정

**요청**
```http
POST /log/level
Content-Type: application/json
X-API-Key: your-secret-api-key

{
    "level": "debug"
}
```

**요청 본문**:
```json
{
    "level": "debug"
}
```

**지원하는 로그 수준**: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`

**응답**
```json
{
    "level": "debug",
    "message": "Log level updated successfully"
}
```

**참고**:
- 이 엔드포인트는 API 키 인증이 필요합니다
- 모든 로그 수준 변경 작업은 보안 감사 로그에 기록됩니다

### Prometheus 지표

Prometheus 형식의 모니터링 지표 데이터를 조회합니다.

**요청**
```http
GET /metrics
```

**응답**: Prometheus 형식의 지표 데이터

**인증**: 배포 환경에 따라 다릅니다.

| `ENVIRONMENT` | `/metrics` 기본값 |
| --- | --- |
| `production` | 인증 필요(데이터 엔드포인트와 동일한 방식) |
| `development`, `test`, 미설정 | 익명 스크레이핑 허용 |

`WARDEN_METRICS_REQUIRE_AUTH`는 기본값을 양방향으로 재정의합니다. 인증이 필요한 엔드포인트를
인증 없이 스크레이핑하면 `401 Unauthorized`가 반환됩니다. 응답에는 카디널리티가 낮고 민감하지
않은 시계열만 포함됩니다. `endpoint`와 `method` 레이블은 허용 목록을 기준으로 정규화되며,
인식되지 않는 값은 모두 `other`로 합쳐집니다.

**응답 예시**:
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

## 오류 응답

모든 API 엔드포인트는 다음과 같은 오류 응답을 반환할 수 있습니다.

### 401 Unauthorized

API 키 인증에 실패한 경우 반환됩니다.

```json
{
    "error": "Unauthorized",
    "message": "Invalid or missing API key"
}
```

### 429 Too Many Requests

요청이 속도 제한을 초과한 경우 반환됩니다.

```json
{
    "error": "Too Many Requests",
    "message": "Rate limit exceeded"
}
```

### 500 Internal Server Error

서버 내부 오류가 발생한 경우 반환됩니다.

```json
{
    "error": "Internal Server Error",
    "message": "An internal error occurred"
}
```

운영 모드에서는 정보 유출을 막기 위해 자세한 오류 정보를 숨깁니다.

## 속도 제한

기본적으로 API 요청은 속도 제한으로 보호됩니다.

- **제한**: 분당 60회 요청
- **윈도**: 1분
- **초과 시**: `429 Too Many Requests` 반환

속도 제한은 설정 파일로 조정할 수 있습니다.

```yaml
rate_limit:
  rate: 60  # 분당 요청 수
  window: 1m
```

## IP 허용 목록

IP 허용 목록은 다음 환경 변수로 설정할 수 있습니다.

- `IP_WHITELIST`: 전역 IP 허용 목록(모든 엔드포인트 접근을 제한)
- `HEALTH_CHECK_IP_WHITELIST`: 상태 확인 엔드포인트 IP 허용 목록(`/health`와 `/healthcheck`만 제한)

CIDR 범위 형식을 지원하며, 여러 IP나 범위는 쉼표로 구분합니다.

```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## 응답 압축

모든 API 응답은 자동 압축(gzip)을 지원합니다. 클라이언트는 요청 헤더 `Accept-Encoding: gzip`으로 압축을 활성화할 수 있습니다.

## 선택적 연동 예시

### 다른 서비스와의 연동 호출 예시(선택)

다른 서비스(예: Stargate)와 연동해야 한다면, 로그인 흐름에서 Warden의 `/user` 엔드포인트를 호출하여 사용자 정보를 조회할 수 있습니다.

**시나리오 1: 전화번호로 조회**

```bash
# Stargate가 Warden을 호출
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?phone=13800138000"
```

**응답 예시**:
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

**시나리오 2: 이메일로 조회**

```bash
# Stargate가 Warden을 호출
curl -H "X-API-Key: your-key" \
     "http://warden:8081/user?mail=admin@example.com"
```

### Go SDK 연동 예시

Stargate는 Warden Go SDK를 사용해 연동할 수 있습니다.

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // Warden 클라이언트 생성
    opts := warden.DefaultOptions().
        WithBaseURL("http://warden:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second)
    
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // 로그인 흐름에서 사용자 조회
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            // 사용자를 찾을 수 없어 로그인 거부
            fmt.Println("User not found in allowlist")
            return
        }
        panic(err)
    }
    
    // 사용자 상태 확인
    if !user.IsActive() {
        // 상태가 활성이 아니므로 로그인 거부
        fmt.Printf("User status is %s, cannot login\n", user.Status)
        return
    }
    
    // 사용자가 존재하고 상태가 활성이므로 로그인 흐름 계속 진행
    fmt.Printf("User found: %s, Status: %s, Role: %s, Scopes: %v\n",
        user.UserID, user.Status, user.Role, user.Scope)
    
    // 다음 단계: Herald를 호출해 인증 코드 전송
    // ...
}
```

### 전체 로그인 흐름 예시(선택적 연동 시나리오)

선택적 연동 시나리오에서 전체 로그인 흐름은 다음과 같을 수 있습니다.

1. **사용자가 식별자를 입력** → 인증 서비스가 수신
2. **인증 서비스 → Warden**: 사용자 정보 조회
   ```go
   user, err := wardenClient.GetUserByIdentifier(ctx, phone, mail, "")
   ```
3. **사용자 상태 검증**: `user.Status == "active"` 확인
4. **인증 서비스 → OTP 서비스**: 챌린지를 만들고 인증 코드 전송(선택)
5. **사용자가 인증 코드 제출** → 인증 서비스가 수신(선택)
6. **인증 서비스 → OTP 서비스**: 인증 코드 검증(선택)
7. **인증 서비스**: 세션을 발급하고 `user.Scope`와 `user.Role`로 인가 헤더 설정

**참고**: Warden은 단독으로 사용할 수 있으며, 위의 연동 흐름은 선택 사항입니다.

## 관련 문서

- [OpenAPI 명세](../../openapi.yaml) - 완전한 OpenAPI 3.1 명세
- [설정 문서](CONFIGURATION.md) - API 키 및 기타 옵션 설정 방법
- [보안 문서](SECURITY.md) - 보안 기능과 모범 사례
- [아키텍처 문서](ARCHITECTURE.md) - 서비스 연동 아키텍처

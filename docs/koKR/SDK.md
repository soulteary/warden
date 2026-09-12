# SDK 사용 문서

> 🌐 **Language / 语言**: [English](../enUS/SDK.md) | [中文](../zhCN/SDK.md) | [Français](../frFR/SDK.md) | [Italiano](../itIT/SDK.md) | [日本語](../jaJP/SDK.md) | [Deutsch](../deDE/SDK.md) | [한국어](SDK.md)

Warden은 다른 프로젝트에 쉽게 통합할 수 있도록 Go SDK를 제공합니다. SDK는 캐시와 인증 등을 지원하는 간결한 API를 제공합니다.

## 특징

- 🚀 **간단하고 쉬움**: 간결한 API를 제공합니다
- ⚡ **높은 성능**: 캐시 내장(GetUsers). 직접 조회(GetUserByIdentifier)로 API 호출을 줄입니다
- 🔒 **안전함**: API 키 인증을 지원하며, 오류 처리 과정에서 민감한 정보가 새지 않습니다
- 📦 **유연함**: 타임아웃, 캐시 TTL 등을 설정할 수 있습니다
- 🔌 **확장 가능**: 사용자 정의 로거 구현을 지원합니다
- 🎯 **스마트 폴백**: CheckUserInList는 전화번호를 찾지 못하면 자동으로 이메일로 폴백합니다

## SDK 설치

```bash
go get github.com/soulteary/warden/pkg/warden
```

## 빠른 시작

### 기본 사용법

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // 클라이언트 옵션 생성
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)
    
    // 클라이언트 생성
    client, err := warden.NewClient(opts)
    if err != nil {
        panic(err)
    }
    
    // 사용자 목록 조회
    ctx := context.Background()
    users, err := client.GetUsers(ctx)
    if err != nil {
        panic(err)
    }
    
    // 사용자가 목록에 있는지 확인(phone, mail 또는 둘 다 지정 가능)
    exists := client.CheckUserInList(ctx, "13800138000", "user@example.com")
    if exists {
        println("User is in the allow list and active")
    }
    
    // phone만 또는 mail만 사용할 수도 있습니다
    existsByPhone := client.CheckUserInList(ctx, "13800138000", "")
    existsByMail := client.CheckUserInList(ctx, "", "user@example.com")
    
    // 사용자 상세 정보 조회
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        panic(err)
    }
    fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
}
```

### 사용자 정의 로거 사용

SDK는 사용자 정의 로거 구현을 지원합니다. 예를 들어 logrus를 사용하는 경우는 다음과 같습니다.

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    logger := logrus.StandardLogger()
    
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithLogger(warden.NewLogrusAdapter(logger))
    
    client, err := warden.NewClient(opts)
    // ...
}
```

### 페이지 단위 조회

```go
// 페이지 단위 사용자 목록 조회
resp, err := client.GetUsersPaginated(ctx, 1, 10) // 1페이지, 페이지당 10건
if err != nil {
    panic(err)
}

fmt.Printf("Total users: %d\n", resp.Pagination.Total)
fmt.Printf("Total pages: %d\n", resp.Pagination.TotalPages)
for _, user := range resp.Data {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
}
```

### 단일 사용자 정보 조회

```go
// 전화번호로 사용자 정보 조회
user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
        println("User not found")
    } else {
        panic(err)
    }
} else {
    fmt.Printf("UserID: %s, Phone: %s, Mail: %s, Status: %s\n", 
        user.UserID, user.Phone, user.Mail, user.Status)
    if user.IsActive() {
        println("User is active")
    }
}

// 이메일로 사용자 정보 조회
user, err = client.GetUserByIdentifier(ctx, "", "user@example.com", "")

// 사용자 ID로 사용자 정보 조회
user, err = client.GetUserByIdentifier(ctx, "", "", "user123")
```

### 캐시 비우기

```go
// 클라이언트 캐시를 수동으로 비우기
client.ClearCache()

// 또는 별칭 사용
client.InvalidateCache()
```

### 사용자 정의 HTTP 트랜스포트

```go
import "net/http"

// 사용자 정의 트랜스포트 생성
customTransport := &http.Transport{
    MaxIdleConns: 100,
    IdleConnTimeout: 90 * time.Second,
}

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithTransport(customTransport)

client, err := warden.NewClient(opts)
```

### HMAC v2 요청 서명

```go
opts := warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET"))

client, err := warden.NewClient(opts)
```

SDK는 키 ID, 타임스탬프, 논스, 경로/쿼리, 메서드, 본문 해시에 서명합니다.
키 ID와 비밀 값은 모두 설정해야 합니다. 한쪽만 설정된 경우에는 서명 없는 요청을 보내지 않고
`ErrCodeInvalidConfig`를 반환합니다.

### 재시도 설정

```go
// 재시도 옵션 설정
retryOpts := warden.DefaultRetryOptions()
retryOpts.MaxRetries = 3
retryOpts.RetryDelay = 100 * time.Millisecond
retryOpts.MaxRetryDelay = 5 * time.Second
retryOpts.BackoffMultiplier = 2.0

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithRetry(retryOpts)

client, err := warden.NewClient(opts)
```

### 이벤트 기반 캐시 무효화

```go
// 캐시 무효화 이벤트용 채널 생성
invalidationCh := make(chan struct{}, 1)

opts := warden.DefaultOptions().
    WithBaseURL("http://localhost:8081").
    WithCacheInvalidationChannel(invalidationCh)

client, err := warden.NewClient(opts)
if err != nil {
    panic(err)
}
defer client.Close() // 중요: 닫아서 백그라운드 리스너를 중지

// 이후 외부 이벤트로 캐시 무효화를 트리거
invalidationCh <- struct{}{}

// 신호를 받으면 캐시가 자동으로 비워집니다
```

## API 레퍼런스

### Options

`Options` 구조체는 클라이언트를 설정하는 데 사용합니다.

- `BaseURL`: Warden 서비스 주소(필수)
- `APIKey`: API 키(선택)
- `Timeout`: HTTP 요청 타임아웃(기본값 10초)
- `CacheTTL`: 캐시 TTL(기본값 5분)
- `Logger`: 로거 인터페이스(선택. 기본값은 NoOpLogger)
- `Transport`: 사용자 정의 HTTP 트랜스포트(선택)
- `HMACKeyID` / `HMACSecret`: HMAC v2 서명 쌍(둘 다 설정하거나 둘 다 설정하지 않음)
- `TLSConfig`: TLS/mTLS 클라이언트 설정(선택)
- `Retry`: 재시도 설정(선택. 기본값은 재시도 없음)
- `CacheInvalidationChannel`: 이벤트 기반 캐시 무효화용 채널(선택)

### 클라이언트 메서드

#### `NewClient(opts *Options) (*Client, error)`

새로운 Warden 클라이언트를 생성합니다.

#### `GetUsers(ctx context.Context) ([]AllowListUser, error)`

전체 사용자 목록을 조회합니다. 캐시가 유효하면 캐시된 데이터를 바로 반환합니다.

#### `GetUsersPaginated(ctx context.Context, page, pageSize int) (*PaginatedResponse, error)`

페이지 단위 사용자 목록을 조회합니다.

- `page`: 페이지 번호(1부터 시작)
- `pageSize`: 페이지 크기

`PaginatedResponse`를 반환하며, 내용은 다음과 같습니다.
- `Data`: 사용자 목록
- `Pagination`: 페이지 정보(페이지 번호, 페이지 크기, 전체 건수, 전체 페이지 수)

**참고:** 이 메서드는 캐시를 사용하지 않습니다. 호출할 때마다 API에서 최신 데이터를 가져옵니다.

#### `GetUserByIdentifier(ctx context.Context, phone, mail, userID string) (*AllowListUser, error)`

식별자로 단일 사용자 정보를 조회합니다.

- `phone`: 사용자 전화번호(선택. 단, phone, mail, userID 중 하나는 반드시 제공해야 함)
- `mail`: 사용자 이메일(선택)
- `userID`: 사용자 고유 식별자(선택)

**중요:** `phone`, `mail`, `userID` 중 정확히 하나만 제공해야 합니다.

`*AllowListUser`와 오류를 반환합니다. 사용자가 존재하지 않으면 `ErrCodeNotFound` 오류를 반환합니다.

**참고:** 이 메서드는 캐시를 사용하지 않습니다. 호출할 때마다 API에서 최신 데이터를 가져옵니다.

#### `CheckUserInList(ctx context.Context, phone, mail string) bool`

사용자가 허용 목록에 있는지 확인합니다.

- `phone`: 사용자 전화번호(선택)
- `mail`: 사용자 이메일(선택)

사용자가 존재하면(전화번호 또는 이메일로 일치) `true`, 그렇지 않으면 `false`를 반환합니다.

**동작:**
- `phone`과 `mail`이 모두 제공되면 `phone`이 우선합니다
- `phone` 조회가 실패하고(`NotFound` 오류) `mail`이 비어 있지 않으면 자동으로 `mail` 조회로 폴백합니다
- `phone` 조회가 성공했지만 사용자 상태가 활성이 아니면 `mail`로 폴백하지 않습니다(이미 사용자를 찾았기 때문)
- `phone` 조회가 실패했고 오류가 `NotFound`가 아니면(예: 네트워크 오류) `mail`로 폴백하지 않습니다
- 입력은 자동으로 정규화됩니다. `phone`은 앞뒤 공백을 제거하고, `mail`은 공백을 제거한 뒤 소문자로 변환합니다
- 이 메서드는 조회에 `GetUserByIdentifier`를 사용하므로 사용자 목록을 순회하는 것보다 효율적입니다
- 상태가 "active"인 사용자만 `true`를 반환합니다

#### `ClearCache()`

클라이언트 내부 캐시를 비웁니다.

#### `InvalidateCache()`

`ClearCache()`의 별칭이며, 이벤트 기반 무효화와의 일관성을 위해 제공됩니다.

#### `Close()`

백그라운드 goroutine(예: 캐시 무효화 리스너)을 중지하고 리소스를 해제합니다.
클라이언트가 더 이상 필요하지 않을 때 호출해야 합니다.

## 타입 정의

### AllowListUser

```go
type AllowListUser struct {
    Phone  string   `json:"phone"`   // 사용자 전화번호
    Mail   string   `json:"mail"`    // 사용자 이메일 주소
    UserID string   `json:"user_id"` // 사용자 고유 식별자(선택. 없으면 자동 생성)
    Status string   `json:"status"`  // 사용자 상태(예: "active", "inactive", "suspended")
    Scope  []string `json:"scope"`   // 사용자 권한 범위(선택)
    Role   string   `json:"role"`    // 사용자 역할(선택)
}
```

**메서드:**
- `IsActive() bool`: 사용자 상태가 "active"인지 확인합니다
- `IsValid() bool`: 사용자 상태가 유효한지 확인합니다(현재는 "active"만 지원)

### PaginatedResponse

```go
type PaginatedResponse struct {
    Data       []AllowListUser `json:"data"`
    Pagination PaginationInfo  `json:"pagination"`
}

type PaginationInfo struct {
    Page       int `json:"page"`        // 현재 페이지 번호(1부터 시작)
    PageSize   int `json:"page_size"`   // 페이지 크기
    Total      int `json:"total"`       // 전체 레코드 수
    TotalPages int `json:"total_pages"` // 전체 페이지 수
}
```

## 오류 처리

SDK는 오류 코드와 상세 정보를 담은 사용자 정의 오류 타입을 사용합니다.

```go
if err != nil {
    if sdkErr, ok := err.(*warden.Error); ok {
        switch sdkErr.Code {
        case warden.ErrCodeUnauthorized:
            // 인증 오류 처리
        case warden.ErrCodeRequestFailed:
            // 요청 실패 처리
        case warden.ErrCodeNotFound:
            // "찾을 수 없음" 오류 처리
        case warden.ErrCodeServerError:
            // 서버 오류 처리
        // ...
        }
    }
}
```

### 오류 코드

- `ErrCodeInvalidConfig`: 잘못된 설정
- `ErrCodeRequestFailed`: 요청 실패
- `ErrCodeInvalidResponse`: 잘못된 응답 형식
- `ErrCodeUnauthorized`: 권한 없음
- `ErrCodeNotFound`: 찾을 수 없음
- `ErrCodeServerError`: 서버 오류

## 모범 사례

1. **클라이언트 재사용**: 클라이언트를 한 번만 만들고 애플리케이션 수명 주기 전체에서 재사용하세요
2. **캐시 TTL을 적절히 설정**: 데이터 갱신 빈도에 맞는 캐시 시간을 설정하세요
3. **Context 사용**: 컨텍스트를 전달해 취소와 타임아웃을 제어하세요
4. **오류 처리**: 항상 오류를 확인하고 처리하세요
5. **로깅**: 운영 환경에서는 적절한 로거 구현을 사용하세요
6. **클라이언트 닫기**: 클라이언트가 더 이상 필요 없으면 `Close()`를 호출해 백그라운드 goroutine을 중지하세요
7. **재시도 설정**: 운영 환경에서는 재시도를 활성화해 일시적인 장애에 대비하세요
8. **사용자 정의 트랜스포트**: 고급 시나리오(TLS, 프록시, 연결 풀 등)에는 사용자 정의 트랜스포트를 사용하세요

## 설계 문서

### 설계 원칙

1. **간단하고 쉬움**: 간결한 API를 제공합니다
2. **높은 성능**: 내장 캐시로 API 호출을 줄입니다
3. **스레드 안전**: 모든 메서드는 동시성에 안전합니다
4. **유연한 설정**: 타임아웃, 캐시, 로거 등을 사용자 정의할 수 있습니다

### 아키텍처 설계

#### 핵심 구성 요소

1. **Client**: HTTP 클라이언트 래퍼
2. **Cache**: 스레드 안전한 인메모리 캐시
3. **Options**: 설정 옵션(Builder 패턴)
4. **Logger**: 로거 인터페이스(다양한 로깅 라이브러리 지원)

#### 동시성 안전성

- `http.Client`는 동시성에 안전합니다
- `Cache`는 `sync.RWMutex`로 스레드 안전성을 보장합니다
- `Client`의 모든 필드는 생성 이후 읽기 전용입니다
- 모든 메서드는 스레드 안전하며 여러 goroutine에서 동시에 호출할 수 있습니다

#### 캐시 전략

1. **GetUsers()**: 캐시를 사용합니다
   - 먼저 캐시를 확인합니다
   - 캐시가 유효하면 바로 반환합니다
   - 캐시가 유효하지 않거나 없으면 API에서 가져와 캐시를 갱신합니다

2. **GetUsersPaginated()**: 캐시를 사용하지 않습니다
   - 이유: 페이지 매개변수가 다르면 결과도 다르기 때문입니다
   - 페이지 매개변수 단위로 캐시하는 것은 복잡합니다
   - 현재 설계: 매번 API에서 가져와 데이터 정확성을 보장합니다

3. **GetUserByIdentifier()**: 캐시를 사용하지 않습니다
   - 이유: 단일 사용자의 최신 정보를 가져와 데이터의 실시간성을 보장해야 하기 때문입니다
   - 호출할 때마다 API에서 가져와 캐시로 인한 불일치를 피합니다

4. **CheckUserInList()**: 캐시를 사용하지 않습니다
   - `GetUserByIdentifier()`로 단일 사용자를 직접 조회합니다
   - 호출할 때마다 API에 요청해 데이터의 실시간성을 보장합니다
   - 스마트 폴백 지원: 전화번호 조회가 실패(NotFound)하고 mail이 비어 있지 않으면 자동으로 mail 조회로 전환합니다
   - 성능 최적화: 단일 사용자를 직접 조회하는 편이 전체 사용자 목록을 순회하는 것보다 효율적입니다

#### CheckUserInList 구현 전략

`CheckUserInList()` 메서드는 다음 전략을 사용합니다.

1. **입력 정규화**: phone과 mail의 앞뒤 공백을 자동으로 제거하고, mail은 소문자로 변환합니다
2. **우선순위 전략**: phone과 mail이 모두 제공되면 phone이 우선합니다
3. **스마트 폴백**:
   - phone 조회가 `NotFound` 오류를 반환하고 mail이 비어 있지 않으면 자동으로 mail 조회로 전환합니다
   - phone 조회가 성공했지만 사용자 상태가 활성이 아니면 mail로 폴백하지 않습니다(이미 사용자를 찾았기 때문)
   - phone 조회가 그 밖의 오류(예: 네트워크 오류)를 만나면 mail로 폴백하지 않습니다
4. **상태 검증**: 상태가 "active"인 사용자만 `true`를 반환합니다
5. **성능 최적화**: `GetUserByIdentifier()`로 직접 조회하여 전체 사용자 목록을 가져오지 않습니다

### RetryOptions

`RetryOptions` 구조체는 재시도 동작을 설정합니다.

- `MaxRetries`: 최대 재시도 횟수(기본값 0, 재시도 없음)
- `RetryDelay`: 재시도 사이의 초기 지연(기본값 100ms)
- `MaxRetryDelay`: 재시도 사이의 최대 지연(기본값 5s)
- `BackoffMultiplier`: 지수 백오프 배수(기본값 2.0)
- `RetryableStatusCodes`: 재시도를 유발하는 HTTP 상태 코드(기본값: 5xx)

**참고:** 네트워크 오류는 항상 재시도 대상입니다. 클라이언트 오류(4xx)는 재시도하지 않습니다.

### 알려진 제한

1. **페이지 캐시**: `GetUsersPaginated()`는 캐시를 사용하지 않습니다
   - 데이터 정확성을 보장하기 위한 의도적인 설계입니다
   - 페이지 캐시가 필요하다면 더 복잡한 전략을 구현할 수 있습니다

2. **단일 사용자 조회 캐시**: `GetUserByIdentifier()`와 `CheckUserInList()`는 캐시를 사용하지 않습니다
   - 데이터의 실시간성을 보장하기 위한 의도적인 설계입니다
   - 캐시가 필요하다면 사용자 식별자에 기반한 전략을 구현할 수 있습니다

### 향후 개선 사항

1. 요청/응답 미들웨어 지원
2. 지표 수집 지원
3. 연결 풀 설정 지원
4. 서킷 브레이커 패턴 지원

## 전체 예시

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/soulteary/warden/pkg/warden"
)

func main() {
    // 클라이언트 생성
    opts := warden.DefaultOptions().
        WithBaseURL("http://localhost:8081").
        WithAPIKey("your-api-key").
        WithTimeout(10 * time.Second).
        WithCacheTTL(5 * time.Minute)

    client, err := warden.NewClient(opts)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // 모든 사용자 조회
    users, err := client.GetUsers(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total users: %d\n", len(users))

    // 전화번호로 단일 사용자 조회
    user, err := client.GetUserByIdentifier(ctx, "13800138000", "", "")
    if err != nil {
        if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound {
            fmt.Println("User not found")
        } else {
            log.Fatal(err)
        }
    } else {
        fmt.Printf("User: %s, Status: %s\n", user.UserID, user.Status)
    }

    // 페이지 단위 조회
    result, err := client.GetUsersPaginated(ctx, 1, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page 1: %d users\n", len(result.Data))

    // 사용자 확인
    exists := client.CheckUserInList(ctx, "13800138000", "admin@example.com")
    fmt.Printf("User exists and active: %v\n", exists)

    // 캐시 비우기
    client.ClearCache()
    fmt.Println("Cache cleared")
}
```

## 관련 문서

- [API 문서](API.md) - API 엔드포인트 상세 정보
- [설정 문서](CONFIGURATION.md) - 서버 설정 옵션

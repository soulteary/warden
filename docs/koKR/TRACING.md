# Warden OpenTelemetry Tracing

Warden 서비스는 서비스 간 호출 체인을 모니터링하고 디버깅하기 위한 OpenTelemetry 분산 추적을 지원합니다.

## 기능

- **자동 HTTP 요청 추적**: 모든 HTTP 요청에 대해 자동으로 span 생성
- **사용자 쿼리 추적**: `/user` 엔드포인트에 대한 상세한 추적 정보 추가
- **컨텍스트 전파**: W3C Trace Context 표준을 지원하며, Stargate 및 Herald 서비스와 원활하게 통합
- **구성 가능**: 환경 변수 또는 구성 파일을 통해 활성화/비활성화

## 구성

### 환경 변수

```bash
# OpenTelemetry 추적 활성화
OTLP_ENABLED=true

# OTLP 엔드포인트 (예: Jaeger, Tempo, OpenTelemetry Collector)
OTLP_ENDPOINT=http://localhost:4318
```

### 구성 파일 (YAML)

```yaml
tracing:
  enabled: true
  endpoint: "http://localhost:4318"
```

## 핵심 Span

### HTTP 요청 Span

모든 HTTP 요청은 다음 속성을 포함하는 span을 자동으로 생성합니다:
- `http.method`: HTTP 메서드
- `http.url`: 요청 URL
- `http.status_code`: 응답 상태 코드
- `http.user_agent`: 사용자 에이전트
- `http.remote_addr`: 클라이언트 주소

### 사용자 쿼리 Span (`warden.get_user`)

`/user` 엔드포인트에 대한 쿼리는 다음을 포함하는 전용 span을 생성합니다:
- `warden.query.phone`: 쿼리된 전화번호 (마스킹됨)
- `warden.query.mail`: 쿼리된 이메일 (마스킹됨)
- `warden.query.user_id`: 쿼리된 사용자 ID
- `warden.user.found`: 사용자가 발견되었는지 여부
- `warden.user.id`: 발견된 사용자 ID

## 사용 예제

### 추적을 활성화하여 Warden 시작

```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://localhost:4318
./warden
```

### 코드에서 추적 사용

```go
import "github.com/soulteary/warden/internal/tracing"

// 자식 span 생성
ctx, span := tracing.StartSpan(ctx, "warden.custom_operation")
defer span.End()

// 속성 설정
span.SetAttributes(attribute.String("key", "value"))

// 오류 기록
if err != nil {
    tracing.RecordError(span, err)
}
```

## Stargate 및 Herald와의 통합

Warden의 추적은 Stargate 및 Herald 서비스의 추적 컨텍스트와 자동으로 통합됩니다:

1. **Stargate**가 Warden을 호출할 때 HTTP 헤더를 통해 trace context를 전달합니다
2. **Warden**이 자동으로 추출하고 추적 체인을 계속합니다
3. 세 서비스의 모든 span이 동일한 trace에 표시됩니다

## 지원되는 추적 백엔드

- **Jaeger**: `OTLP_ENDPOINT=http://localhost:4318`
- **Tempo**: `OTLP_ENDPOINT=http://localhost:4318`
- **OpenTelemetry Collector**: `OTLP_ENDPOINT=http://localhost:4318`
- **기타 OTLP 호환 백엔드**

## 성능 고려사항

- 추적은 기본적으로 배치 내보내기를 사용하여 성능 영향을 최소화합니다
- 샘플링 속도를 통해 추적 데이터 양을 제어할 수 있습니다
- 프로덕션 환경에서는 샘플링 전략을 사용하는 것이 좋습니다 (현재는 전체 샘플링, 개발 환경에 적합)

## 문제 해결

### 추적이 활성화되지 않음

환경 변수 확인:
```bash
echo $OTLP_ENABLED
echo $OTLP_ENDPOINT
```

### 추적 데이터가 백엔드에 도달하지 않음

1. OTLP 엔드포인트가 액세스 가능한지 확인
2. 네트워크 연결 확인
3. Warden 로그의 오류 메시지 확인

### Span 누락

새로운 context를 생성하는 대신 요청 처리에서 `r.Context()`를 사용하여 컨텍스트를 전달해야 합니다.

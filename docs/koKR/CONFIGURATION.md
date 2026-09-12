# 설정

> 🌐 **Language / 语言**: [English](../enUS/CONFIGURATION.md) | [中文](../zhCN/CONFIGURATION.md) | [Français](../frFR/CONFIGURATION.md) | [Italiano](../itIT/CONFIGURATION.md) | [日本語](../jaJP/CONFIGURATION.md) | [Deutsch](../deDE/CONFIGURATION.md) | [한국어](CONFIGURATION.md)

이 문서는 실행 모드, 설정 파일 형식, 환경 변수 등 Warden의 설정 옵션을 자세히 설명합니다.

**설정 우선순위**: 명령줄 인자 > 환경 변수 > 설정 파일(YAML) > 기본값.

**전체 옵션 표**(YAML 경로, 환경 변수, 기본값, 검증 규칙)는 [zhCN CONFIGURATION](../zhCN/CONFIGURATION.md)을 참고하세요. 요약은 다음과 같습니다.

| 분류 | YAML / Env | 비고 |
|----------|------------|--------|
| 서버 | `server.*` / `PORT` | port, read_timeout, write_timeout, shutdown_timeout, idle_timeout, max_header_bytes |
| Redis | `redis.*` / `REDIS`, `REDIS_PASSWORD`, `REDIS_PASSWORD_FILE`, `REDIS_ENABLED` | addr, password, password_file, db. Redis는 기본적으로 활성(`true`)이며, REDIS가 없는 ONLY_LOCAL은 예외 |
| 캐시 | `cache.ttl`, `cache.update_interval` | 환경 변수로 재정의 불가. update_interval 기본값 5s |
| 속도 제한 | `rate_limit.rate`, `rate_limit.window` | 기본값 분당 60회, 윈도 1m |
| HTTP 클라이언트 | `http.*` / `HTTP_TIMEOUT`, `HTTP_MAX_IDLE_CONNS`, `HTTP_INSECURE_TLS` | timeout, max_idle_conns, insecure_tls, max_retries, retry_delay |
| 원격 | `remote.*` / `CONFIG`, `KEY`, `MERGE_MODE`, `REMOTE_DECRYPT_ENABLED`, `REMOTE_RSA_PRIVATE_KEY_FILE`, `REMOTE_RSA_PRIVATE_KEY` | url, key, mode, decrypt_enabled, rsa_private_key_file |
| 작업 | `task.interval` | 설정 파일 사용 시 환경 변수로 재정의 불가. `INTERVAL`은 설정 파일을 사용하지 않을 때만 적용 |
| 애플리케이션 | `app.*` / `API_KEY`, `DATA_FILE`, `DATA_DIR`, `RESPONSE_FIELDS` | mode, api_key, data_file, data_dir, response_fields |
| 추적 | `tracing.enabled`, `tracing.endpoint` / `OTLP_ENABLED`, `OTLP_ENDPOINT` | `--config-file` 사용 시 `CONFIG_FILE`이 같은 경로로 지정되지 않으면 해당 파일에서 tracing을 읽지 않습니다 |
| 서비스 간 인증 | — / `WARDEN_HMAC_KEYS`, `WARDEN_HMAC_TIMESTAMP_TOLERANCE`, `WARDEN_TLS_*` | **환경 변수 전용**(YAML 키 없음) |
| 상태 | — / `SNAPSHOT_MAX_AGE` | 허용되는 스냅샷 최대 경과 시간. Go duration 형식, 기본값 `max(30s, 작업 간격 × 3)` |

## 실행 모드(MERGE_MODE)

시스템은 7가지 데이터 병합 모드를 지원하며 `MERGE_MODE`로 선택합니다(`MODE`는 더 이상 권장되지 않음).

| 모드 | 설명 | 사용 사례 |
|------|-------------|----------|
| `DEFAULT` | 기존의 원격 우선, 관대한 동작 | 모드를 지정한 적이 없는 배포와의 하위 호환 |
| `REMOTE_FIRST` | 로드에 성공하면 원격이 우선. 원격 오류는 치명적이며 마지막으로 유효했던 스냅샷을 유지 | 원격이 권위를 갖는 엄격한 배포 |
| `ONLY_REMOTE` | 원격 데이터 소스만 사용 | 원격 설정에 전적으로 의존 |
| `ONLY_LOCAL` | 로컬 설정 파일만 사용하며 **Redis는 기본적으로 비활성**(`REDIS` 주소를 명시적으로 설정하거나 `REDIS_ENABLED=true`이면 활성화) | 오프라인 환경 또는 테스트 환경 |
| `LOCAL_FIRST` | 로컬 우선. 로컬 데이터가 없을 때 원격 데이터로 보완 | 로컬 설정이 주가 되고 원격이 보조가 되는 구성 |
| `REMOTE_FIRST_ALLOW_REMOTE_FAILED` | 원격 우선. 원격 실패 시 로컬로 폴백 허용 | 고가용성 시나리오 |
| `LOCAL_FIRST_ALLOW_REMOTE_FAILED` | 로컬 우선. 로컬 실패 시 원격으로 폴백 허용 | 하이브리드 모드 |

### 스냅샷 신선도와 원격 장애

`REMOTE_FIRST`와 `ONLY_REMOTE`는 엄격한 모드입니다. 예약된 원격 갱신이 실패하면 Warden은
마지막으로 유효했던 메모리 내 스냅샷을 계속 제공하고, 갱신 실패를 기록하며, 스냅샷의
`loaded_at`을 진행시키지 않습니다. 스냅샷이 `SNAPSHOT_MAX_AGE`(기본값 `max(30s, 작업 간격 × 3)`)를
넘어서면 상태 확인 엔드포인트는 HTTP 503을 반환합니다. 값은 `2m`과 같은 Go duration 형식으로
설정합니다.

`REMOTE_FIRST_ALLOW_REMOTE_FAILED`는 검증된 로컬 데이터로 명시적으로 폴백하고 스냅샷을
`degraded`로 표시하면서 HTTP 200으로 계속 서비스합니다. `LOCAL_FIRST`와
`LOCAL_FIRST_ALLOW_REMOTE_FAILED`는 원격 보완을 사용할 수 없더라도 주된 로컬 데이터 로드가
성공하면 정상 상태를 유지할 수 있습니다. `DEFAULT`는 호환성을 위해 기존의 관대한 동작(위의
평문 로컬 성공 의미 포함)을 유지합니다. 새로운 운영 배포에서는 모드를 명시적으로 선택하세요.
암호화된 원격이 실패하여 로컬로 폴백한 경우, 모든 관대한 모드에서 `degraded`로 보고됩니다.

다중 복제본 배포에서는 각 복제본이 자신의 프로세스 로컬 캐시와 스냅샷을 갱신합니다. 분산
Redis 잠금은 공유 Redis 캐시의 기록자만 선출하므로, 기록자가 아닌 복제본도 자신의 스냅샷
신선도를 계속 진행시킵니다.

### 설정 방법

실행 모드는 다음 방법으로 설정할 수 있습니다.

**명령줄 인자**:
```bash
go run . --mode DEFAULT
```

**환경 변수**:
```bash
export MERGE_MODE=DEFAULT
# MODE는 더 이상 권장되지 않는 호환용 별칭으로 남아 있습니다.
```

**설정 파일**:
```yaml
remote:
  mode: "DEFAULT"
# 또는
app:
  mode: "DEFAULT"
```

## 설정 파일 형식

### 로컬 사용자 데이터 파일(`data.json`)

로컬 사용자 데이터 파일 `data.json`의 형식(`data.example.json` 참고):

**최소 형식**(필수 필드만):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**전체 형식**(모든 선택 필드 포함):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write", "admin"],
        "role": "admin"
    },
    {
        "phone": "13900139000",
        "mail": "user@example.com",
        "status": "active",
        "scope": ["read"],
        "role": "user"
    }
]
```

**필드 설명**:
- `phone`(필수): 사용자 전화번호
- `mail`(필수): 사용자 이메일 주소
- `user_id`(선택): 사용자 고유 식별자. 제공하지 않으면 `phone` 또는 `mail`을 기반으로 자동 생성됩니다
- `status`(선택): 사용자 상태. 값이 없으면 안전한 쪽으로 `"inactive"`가 됩니다. 접근을 허용하려면 `"active"`를 명시적으로 설정하세요.
- `scope`(선택): 사용자 권한 범위 배열. 기본값은 빈 배열
- `role`(선택): 사용자 역할. 기본값은 빈 문자열

### 애플리케이션 설정 파일(`config.yaml`)

YAML 형식의 설정 파일을 지원하며 `--config-file` 매개변수로 지정합니다.

```yaml
server:
  port: "8081"
  read_timeout: 5s
  write_timeout: 5s
  shutdown_timeout: 5s
  max_header_bytes: 1048576  # 1MB
  idle_timeout: 120s

redis:
  addr: "localhost:6379"
  password: ""  # 환경 변수 REDIS_PASSWORD 또는 REDIS_PASSWORD_FILE 사용을 권장
  password_file: ""  # 비밀번호 파일 경로(password보다 우선)
  db: 0

cache:
  ttl: 3600s
  update_interval: 5s

rate_limit:
  rate: 60  # 분당 요청 수
  window: 1m

http:
  timeout: 5s
  max_idle_conns: 100
  insecure_tls: false  # 개발 환경 전용
  max_retries: 3
  retry_delay: 1s

remote:
  url: "http://localhost:8080/data.json"
  key: ""
  mode: "DEFAULT"
  decrypt_enabled: false       # 원격 응답을 RSA로 복호화(rsa_private_key_file 또는 REMOTE_RSA_PRIVATE_KEY와 함께 사용)
  rsa_private_key_file: ""    # PEM 파일 경로(인라인 PEM은 환경 변수 REMOTE_RSA_PRIVATE_KEY 사용)

task:
  interval: 5s

app:
  mode: "DEFAULT"  # 데이터 병합 모드. 운영 정책은 ENVIRONMENT로 선택합니다
  api_key: ""      # 환경 변수 API_KEY 사용을 권장
  data_file: "./data.json"
  data_dir: ""     # 선택: 디렉터리의 모든 *.json을 병합(data_file과 함께 사용 가능)
  response_fields: []  # 선택: API 응답 필드 허용 목록. 비어 있으면 모든 필드

tracing:
  enabled: false
  endpoint: ""     # 예: "http://localhost:4318"
```

**설정 우선순위**: 명령줄 인자 > 환경 변수 > 설정 파일 > 기본값.

**추적 관련 참고**: `--config-file`을 사용할 때 환경 변수 `CONFIG_FILE`이 같은 경로로 설정되어 있거나 `OTLP_ENABLED` + `OTLP_ENDPOINT`를 사용하지 않는 한, 메인 프로그램은 해당 파일의 `tracing` 섹션을 읽지 않습니다.

예시 파일을 참고하세요: [config.example.yaml](../../config.example.yaml).

## 명령줄 인자

```bash
go run . \
  --port 8081 \                    # 웹 서비스 포트(기본값: 8081)
  --redis localhost:6379 \         # Redis 주소(기본값: localhost:6379)
  --redis-password "password" \    # Redis 비밀번호(선택. 환경 변수 사용 권장)
  --redis-enabled=true \           # Redis 활성화/비활성화(기본값: true)
  --config http://example.com/api \ # 원격 설정 URL
  --key "Bearer token" \           # 원격 설정 인증 헤더
  --interval 5 \                   # 예약 작업 간격(초, 기본값: 5)
  --mode DEFAULT \                 # 실행 모드(위 설명 참고)
  --http-timeout 5 \               # HTTP 요청 타임아웃(초, 기본값: 5)
  --http-max-idle-conns 100 \     # HTTP 최대 유휴 연결 수(기본값: 100)
  --http-insecure-tls \           # TLS 인증서 검증 건너뛰기(개발 환경 전용)
  --api-key "your-secret-api-key" \ # 인증용 API 키(선택. 환경 변수 사용 권장)
  --config-file config.yaml        # 설정 파일 경로(YAML 형식 지원)
```

**참고**:
- 설정 파일 지원: `--config-file` 매개변수로 YAML 형식의 설정 파일을 지정할 수 있습니다
- Redis 비밀번호 보안: 명령줄 인자 대신 환경 변수 `REDIS_PASSWORD` 또는 `REDIS_PASSWORD_FILE` 사용을 권장합니다
- TLS 인증서 검증: `--http-insecure-tls`는 개발 환경 전용이며 운영 환경에서 사용해서는 안 됩니다

## 환경 변수

환경 변수를 통한 설정을 지원하며, 명령줄 인자보다 우선순위가 낮습니다. 전체 옵션 표(검증 규칙 포함)는 [zhCN CONFIGURATION](../zhCN/CONFIGURATION.md)을 참고하세요.

```bash
export PORT=8081
export REDIS=localhost:6379
export REDIS_PASSWORD="password"        # Redis 비밀번호(선택)
export REDIS_PASSWORD_FILE="/path/to/password/file"  # Redis 비밀번호 파일 경로(선택. 우선순위: REDIS_PASSWORD > REDIS_PASSWORD_FILE > 설정 파일)
export REDIS_ENABLED=true               # Redis 활성화/비활성화(선택. 기본값: true. true/false/1/0 지원)
                                        # 참고: ONLY_LOCAL 모드에서는 기본값이 false
                                        #       단, REDIS 주소를 명시적으로 설정하면 자동으로 활성화됩니다
export CONFIG=http://example.com/api
export KEY="Bearer token"
export INTERVAL=5
export MERGE_MODE=DEFAULT
export DATA_FILE=./data.json          # 로컬 사용자 데이터 파일 경로
export DATA_DIR=                      # 선택: 모든 *.json을 병합할 디렉터리(DATA_FILE과 함께 사용 가능)
export RESPONSE_FIELDS=               # 선택: API 응답 필드 허용 목록(쉼표 구분. 예: phone,mail,user_id,status,name). 비어 있으면 전체
export REMOTE_DECRYPT_ENABLED=false   # 선택: 원격 응답을 RSA로 복호화
export REMOTE_RSA_PRIVATE_KEY_FILE=   # 선택: RSA 개인 키 PEM 경로(인라인 PEM은 REMOTE_RSA_PRIVATE_KEY 사용)
export REMOTE_RSA_PRIVATE_KEY=        # 선택: 인라인 RSA 개인 키 PEM(REMOTE_RSA_PRIVATE_KEY_FILE 미설정 시 사용)
export HTTP_TIMEOUT=5                  # HTTP 요청 타임아웃(초)
export HTTP_MAX_IDLE_CONNS=100         # HTTP 최대 유휴 연결 수
export HTTP_INSECURE_TLS=false         # TLS 인증서 검증을 건너뛸지 여부(true/false 또는 1/0)
export API_KEY="your-secret-api-key"   # 인증용 API 키(강력히 권장)
export CONFIG_FILE=config.yaml         # 선택. `--config-file`을 쓰지 않을 때 YAML에서 추적 설정을 읽거나, `--config-file`과 같은 파일에서 추적을 활성화할 때 사용
export OTLP_ENABLED=false              # OpenTelemetry 활성화(true/false 또는 1/0)
export OTLP_ENDPOINT=http://localhost:4318  # OTLP 엔드포인트(OTLP_ENABLED가 true이면 필수)
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"  # 신뢰하는 프록시 IP 목록(쉼표 구분)
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"  # 상태 확인 엔드포인트 IP 허용 목록(선택)
export IP_WHITELIST="192.168.1.0/24"  # 전역 IP 허용 목록(선택)
export LOG_LEVEL="info"                # 로그 수준(선택. 기본값: info. 옵션: trace, debug, info, warn, error, fatal, panic)
export WARDEN_HMAC_KEYS='{"key-id":"0123456789abcdef0123456789abcdef"}'  # 운영 환경 비밀 값은 최소 32바이트여야 합니다
export WARDEN_HMAC_ALLOW_V1=false                # 기본값: false. 기간이 정해진 v1 마이그레이션 중에만 true로 설정
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60     # HMAC 타임스탬프 허용 오차(초)
export WARDEN_TLS_CERT=/path/to/warden.crt    # 서비스 간 인증: 서버 TLS 인증서(KEY와 함께 설정하면 TLS 활성화)
export WARDEN_TLS_KEY=/path/to/warden.key     # 서버 TLS 개인 키
export WARDEN_TLS_CA=/path/to/ca.crt          # 클라이언트 CA(mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true    # 클라이언트 인증서 요구(mTLS)
```

**환경 변수 우선순위**:
- Redis 비밀번호: `REDIS_PASSWORD` > `REDIS_PASSWORD_FILE` > 명령줄 인자 `--redis-password`

**보안 설정 참고 사항**:
- `API_KEY`: 민감한 엔드포인트(`/`, `/log/level`)를 보호합니다. 운영 환경에서 강력히 권장됩니다
- `TRUSTED_PROXY_IPS`: 신뢰하는 리버스 프록시 IP를 설정해 클라이언트의 실제 IP를 올바르게 얻습니다
- `HEALTH_CHECK_IP_WHITELIST`: 상태 확인 엔드포인트의 접근 IP를 제한합니다(선택. CIDR 범위 지원)
- `IP_WHITELIST`: 전역 IP 허용 목록(선택. CIDR 범위 지원)

## 원격 설정 API 요구 사항

원격 설정 API는 동일한 형식의 JSON 배열을 반환해야 하며, 선택적으로 Authorization 헤더 인증을 지원할 수 있습니다.

API 응답 형식은 `data.json` 파일 형식과 일치해야 합니다.

```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com",
        "user_id": "a1b2c3d4e5f6g7h8",
        "status": "active",
        "scope": ["read", "write"],
        "role": "admin"
    }
]
```

환경 변수 `KEY` 또는 `--key` 매개변수가 설정되어 있으면 요청에 `Authorization` 헤더가 자동으로 추가됩니다.

```http
Authorization: Bearer your-token-here
```

## 선택적 서비스 연동 설정

다른 서비스(예: Stargate)와 연동하기로 했다면 서비스 간 인증을 설정할 수 있습니다. 관련 설정 항목은 다음과 같습니다.

**참고**: Warden을 단독으로 사용한다면 아래 설정은 모두 선택 사항입니다.

### mTLS 설정(권장)

서비스 간 인증에 상호 TLS 인증서를 사용합니다. **환경 변수만 지원**합니다(애플리케이션 설정에 YAML 키 없음).

```bash
# Warden 서버 인증서
export WARDEN_TLS_CERT=/path/to/warden.crt
export WARDEN_TLS_KEY=/path/to/warden.key
export WARDEN_TLS_CA=/path/to/ca.crt

# 클라이언트 인증서 요구(mTLS)
export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
```

### HMAC 서명 설정

서비스 간 인증에 HMAC-SHA256 서명을 사용합니다. **환경 변수만 지원**합니다(YAML 키 없음).

```bash
# HMAC 키(JSON 형식. 키 교체를 위해 여러 키 지원)
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef","key-id-2":"abcdef0123456789abcdef0123456789"}'

# 타임스탬프 허용 오차(초). HMAC 키가 설정되면 기본값은 60
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60

# 레거시 v1은 기본적으로 비활성화되어 있습니다. 기존 호출자를 마이그레이션하는 동안에만 활성화하세요.
export WARDEN_HMAC_ALLOW_V1=false
```

### Stargate 호출 설정

Stargate 쪽에서는 Warden 서비스 주소와 인증 정보를 설정해야 합니다.

**Stargate 설정 예시**(환경 변수):
```bash
# Warden 서비스 주소
export STARGATE_WARDEN_BASE_URL=http://warden:8081

# 서비스 간 인증 방식(mTLS 또는 HMAC)
export STARGATE_WARDEN_AUTH_TYPE=hmac

# HMAC 설정(HMAC을 사용하는 경우)
export STARGATE_WARDEN_HMAC_KEY_ID=key-id-1
export STARGATE_WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef

# mTLS 설정(mTLS를 사용하는 경우)
export STARGATE_WARDEN_TLS_CERT=/path/to/stargate.crt
export STARGATE_WARDEN_TLS_KEY=/path/to/stargate.key
export STARGATE_WARDEN_TLS_CA=/path/to/ca.crt
```

### 설정 우선순위

1. **mTLS**: TLS 인증서가 설정되어 있으면 mTLS가 먼저 사용됩니다
2. **HMAC**: mTLS가 설정되어 있지 않으면 HMAC 서명이 사용됩니다
3. **API 키**: 둘 다 설정되어 있지 않으면 API 키 인증으로 폴백합니다(서비스 간 호출에는 권장되지 않음)

### 설정 검증

Warden은 시작할 때 서비스 간 인증 설정을 확인합니다.

- 불완전한 TLS 설정을 거부합니다. 인증서와 키는 반드시 함께 설정해야 하며, mTLS에는 클라이언트 CA도 필요합니다
- HMAC이 설정된 경우 키 형식이 올바른지 검증합니다
- `ENVIRONMENT=production`에서는 API 키, HMAC, mTLS 인증 중 어느 것도 설정되어 있지 않으면 기동을 거부합니다

## 자세한 설정 문서

매개변수 해석 방식, 우선순위 규칙, 사용 예시에 대한 자세한 내용은 다음을 참고하세요.

- [매개변수 해석 설계 문서](CONFIG_PARSING.md) - 매개변수 해석 방식에 대한 자세한 설명
- [아키텍처 설계 문서](ARCHITECTURE.md) - 전체 아키텍처와 설정의 영향 이해하기
- [보안 문서](SECURITY.md) - 서비스 간 인증 상세 내용

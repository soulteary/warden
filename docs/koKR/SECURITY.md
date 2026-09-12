# 보안 문서

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](SECURITY.md)

이 문서는 Warden의 보안 기능, 보안 설정, 모범 사례를 설명합니다.

## 구현된 보안 기능

1. **API 인증**: API 키 인증을 지원해 민감한 엔드포인트를 보호합니다
2. **SSRF 방어**: 원격 설정 URL을 엄격히 검증해 서버 측 요청 위조 공격을 막습니다
3. **입력 검증**: 모든 입력 매개변수를 엄격히 검증해 인젝션 공격을 막습니다
4. **속도 제한**: IP 기반 속도 제한으로 DDoS 공격을 막습니다
5. **TLS 검증**: 운영 환경에서는 TLS 인증서 검증을 강제합니다
6. **오류 처리**: 운영 환경에서는 자세한 오류 정보를 숨겨 정보 유출을 막습니다
7. **보안 응답 헤더**: 보안 관련 HTTP 응답 헤더를 자동으로 추가합니다
8. **IP 허용 목록**: 상태 확인 엔드포인트용 IP 허용 목록을 설정할 수 있습니다
9. **설정 파일 검증**: 경로 탐색 공격을 막습니다
10. **JSON 크기 제한**: JSON 응답 본문 크기를 제한해 메모리 고갈 공격을 막습니다
11. **사용자 조회 매개변수 길이 제한**: 단일 매개변수(`phone`/`mail`/`user_id`)는 512바이트를 넘을 수 없으며, DoS와 로그·캐시 비대화를 막습니다
12. **감사 로그의 개인정보 정제**: 감사에 기록되는 식별자는 phone/mail을 마스킹하여, 감사 저장소가 침해되더라도 개인정보가 노출되지 않도록 합니다

## 보안 모범 사례

### 1. 운영 환경 설정

**필수 설정**:
- 운영 환경 강화를 활성화하려면 `ENVIRONMENT=production`을 **반드시** 설정해야 합니다.
- 서비스 인증 방식을 **반드시** 하나 이상 설정해야 합니다: `API_KEY`, HMAC v2 또는 mTLS.
- 클라이언트 IP를 올바르게 얻으려면 `TRUSTED_PROXY_IPS`를 **반드시** 설정해야 합니다
- `HEALTH_CHECK_IP_WHITELIST`로 상태 확인 접근을 **반드시** 제한해야 합니다(또는 네트워크나 리버스 프록시에서 `/health`, `/healthcheck`를 제한하세요)
- `/metrics`를 **반드시** 제한해야 합니다. `ENVIRONMENT=production`에서는 기본적으로 인증이 필요하며, 그대로 유지하거나(또는 리버스 프록시·네트워크 계층에서 경로를 제한하고) 운영 환경에서 `WARDEN_METRICS_REQUIRE_AUTH=false`를 설정하지 **마세요**.

**설정 예시**:
```bash
export API_KEY="your-strong-api-key-here"
export ENVIRONMENT=production
export WARDEN_METRICS_REQUIRE_AUTH=true
export TRUSTED_PROXY_IPS="10.0.0.1,172.16.0.1"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,10.0.0.0/8"
```

### 2. 민감 정보 관리

**권장 방식**:
- ✅ 비밀번호와 키는 환경 변수에 저장하세요
- ✅ Redis 비밀번호에는 비밀번호 파일(`REDIS_PASSWORD_FILE`)을 사용하세요
- ✅ 설정 파일에는 자리 표시자나 주석을 사용하세요
- ✅ 설정 파일 권한이 올바른지 확인하세요(예: `chmod 600`)

**권장하지 않는 방식**:
- ❌ 설정 파일에 비밀번호를 하드코딩하기
- ❌ 명령줄 인자로 비밀번호 전달하기(프로세스 목록에 노출됩니다)
- ❌ 민감 정보가 담긴 설정 파일을 버전 관리에 커밋하기

**예시**:
```yaml
# config.yaml
redis:
  addr: "localhost:6379"
  # password: ""  # 환경 변수 REDIS_PASSWORD 또는 REDIS_PASSWORD_FILE 사용

app:
  # api_key: ""  # 환경 변수 API_KEY 사용
```

### 3. 네트워크 보안

**필수 설정**:
- 운영 환경에서는 반드시 HTTPS를 사용하세요
- 방화벽 규칙을 설정해 접근을 제한하세요
- 알려진 취약점을 수정하기 위해 의존성을 정기적으로 갱신하세요

**권장 설정**:
- 리버스 프록시(예: Nginx)에서 SSL/TLS를 처리하세요
- `TRUSTED_PROXY_IPS`를 설정해 클라이언트의 실제 IP를 올바르게 얻으세요
- 강력한 비밀번호와 API 키를 사용하세요
- `HTTP_INSECURE_TLS`를 비활성화하세요(운영 환경에서는 `false`여야 합니다)

### 4. 모니터링과 감사

**권장 방식**:
- 보안 이벤트 로그를 모니터링하세요
- 접근 로그를 정기적으로 검토하세요
- CI/CD에서 보안 스캔 도구를 사용하세요
- 경보 체계를 마련하세요

**로그 수준 관리**:
- 운영 환경에서는 `info` 또는 `warn` 수준을 권장합니다
- 모든 로그 수준 변경 작업은 보안 감사 로그에 기록됩니다
- 로그 수준은 `/log/level` API로 동적으로 조정할 수 있습니다(API 키 인증 필요)

## API 보안

### API 키 인증

일부 API 엔드포인트는 API 키 인증을 요구합니다.

**인증이 필요한 엔드포인트**:
- `GET /` - 사용자 목록 조회
- `GET /user` - 단일 사용자 조회
- `GET /log/level` - 로그 수준 조회
- `POST /log/level` - 로그 수준 설정

**인증이 필요 없는 엔드포인트**(운영 환경에서는 다른 방법으로 반드시 보호해야 함):
- `GET /health` - 상태 확인(`HEALTH_CHECK_IP_WHITELIST` 또는 네트워크 격리를 **반드시** 설정하세요)
- `GET /healthcheck` - 상태 확인(위와 동일)
- `GET /metrics` - Prometheus 지표(스크레이핑용 API 키를 **반드시** 설정하거나 리버스 프록시·네트워크에서 제한하세요. 공개하지 마세요)

**인증 방식**:
1. **X-API-Key 헤더**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer 헤더**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### 속도 제한

기본적으로 API 요청은 속도 제한으로 보호됩니다.

- **제한**: 분당 60회 요청
- **윈도**: 1분
- **초과 시**: `429 Too Many Requests` 반환

설정 파일로 조정할 수 있습니다.

```yaml
rate_limit:
  rate: 60  # 분당 요청 수
  window: 1m
```

### IP 허용 목록

두 가지 IP 허용 목록을 설정할 수 있습니다.

1. **전역 IP 허용 목록**(`IP_WHITELIST`):
   - 모든 엔드포인트 접근을 제한합니다
   - CIDR 범위 형식을 지원합니다

2. **상태 확인 IP 허용 목록**(`HEALTH_CHECK_IP_WHITELIST`):
   - `/health`와 `/healthcheck` 엔드포인트만 제한합니다
   - CIDR 범위 형식을 지원합니다

**설정 예시**:
```bash
export IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
export HEALTH_CHECK_IP_WHITELIST="127.0.0.1,::1,10.0.0.0/8"
```

## 데이터 보안

### 원격 설정 API 보안

- 원격 설정 API에는 인증 방식(Authorization 헤더)을 사용해야 합니다
- HTTPS 프로토콜 사용을 권장합니다
- 원격 API의 TLS 인증서를 검증하세요(운영 환경에서는 필수)

### Redis 보안

- Redis에는 비밀번호 보호를 설정해야 합니다
- 환경 변수 `REDIS_PASSWORD` 또는 `REDIS_PASSWORD_FILE`을 사용하세요
- Redis 네트워크 접근을 제한하세요(애플리케이션 서버만 허용)
- 알려진 취약점을 수정하기 위해 Redis를 정기적으로 갱신하세요

### 데이터 파일 보안

- `data.json` 파일 권한이 올바르게 설정되었는지 확인하세요
- 민감한 데이터를 버전 관리에 커밋하지 마세요
- 데이터 파일을 정기적으로 백업하세요

## 보안 응답 헤더

Warden은 다음과 같은 보안 관련 HTTP 응답 헤더를 자동으로 추가합니다.

- `X-Content-Type-Options: nosniff` - MIME 타입 스니핑 방지
- `X-Frame-Options: DENY` - 클릭재킹 방지
- `X-XSS-Protection: 1; mode=block` - XSS 방어

## 오류 처리

### 운영 모드

운영 모드(`ENVIRONMENT=production`)에서는 다음과 같습니다.

- 정보 유출을 막기 위해 자세한 오류 정보를 숨깁니다
- 일반적인 오류 메시지를 반환합니다
- 자세한 오류 정보는 로그에만 기록됩니다

### 개발 모드

개발 모드에서는 다음과 같습니다.

- 디버깅을 위해 자세한 오류 정보를 표시합니다
- 스택 추적 정보를 포함합니다

## 보안 감사

릴리스 강화와 검증 지침은 [Release Security](../RELEASE_SECURITY.md)를 참고하세요.

## 취약점 신고

보안 취약점을 발견했다면 다음 방법으로 신고해 주세요.

1. 비공개 보안 Issue를 생성합니다(지원되는 경우)
2. 프로젝트 관리자에게 이메일을 보냅니다
3. 수정되기 전까지 취약점을 공개하지 않습니다

## 서비스 간 인증(선택)

다른 서비스(예: Stargate)와 연동하기로 했다면 서비스 간 인증으로 보안을 확보할 수 있습니다. **mTLS와 HMAC이 구현되어 있으며**, 인증 우선순위는 **mTLS > HMAC > API 키**입니다. Warden은 다음 방식을 지원합니다.

**참고**: Warden을 단독으로 사용한다면 서비스 간 인증은 선택 사항입니다.

### mTLS(권장)

상호 TLS 인증서로 인증하여 더 높은 보안을 제공합니다.

**설정**:

1. **인증서 생성**:
   ```bash
   # CA 인증서 생성
   openssl genrsa -out ca.key 2048
   openssl req -new -x509 -days 365 -key ca.key -out ca.crt
   
   # Warden 서버 인증서 생성
   openssl genrsa -out warden.key 2048
   openssl req -new -key warden.key -out warden.csr
   openssl x509 -req -days 365 -in warden.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out warden.crt
   
   # Stargate 클라이언트 인증서 생성
   openssl genrsa -out stargate.key 2048
   openssl req -new -key stargate.key -out stargate.csr
   openssl x509 -req -days 365 -in stargate.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out stargate.crt
   ```

2. **Warden 설정**(환경 변수):
   ```bash
   export WARDEN_TLS_CERT=/path/to/warden.crt
   export WARDEN_TLS_KEY=/path/to/warden.key
   export WARDEN_TLS_CA=/path/to/ca.crt
   export WARDEN_TLS_REQUIRE_CLIENT_CERT=true
   ```

3. **Stargate 설정**:
   - 클라이언트 인증서 경로를 설정합니다
   - Warden 서버 인증서를 검증하기 위해 CA 인증서 경로를 설정합니다

### HMAC 서명

HMAC-SHA256 서명으로 요청을 검증합니다. 배포가 더 간편합니다.

**서명 알고리즘**:
```text
canonical_v2 = METHOD + "\n" + ESCAPED_PATH_AND_QUERY + "\n" + KEY_ID + "\n" +
               TIMESTAMP + "\n" + NONCE + "\n" + SHA256_HEX(BODY)
signature = HEX(HMAC_SHA256(secret, canonical_v2))
```

**요청 헤더**:
- `X-Signature`: HMAC 서명 값
- `X-Timestamp`: Unix 타임스탬프(초)
- `X-Key-Id`: 키 ID(서명에 포함되며 안전한 키 교체에 사용)
- `X-Nonce`: 고유한 128비트 16진수 논스
- `X-Signature-Version`: `v2`

**Warden 설정**(환경 변수):
```bash
export WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'
export WARDEN_HMAC_TIMESTAMP_TOLERANCE=60  # 타임스탬프 허용 오차(초). 기본값 60
export WARDEN_HMAC_ALLOW_V1=false          # 기본값. 기간이 정해진 레거시 마이그레이션 중에만 true로 설정
```

운영 환경 설정에서는 각 HMAC 비밀 값이 최소 32바이트의 원시 바이트를 포함해야 합니다.
암호학적으로 안전한 난수원으로 비밀 값을 생성하고 비밀 관리 서비스에 보관하세요.
위의 예시 값을 그대로 재사용하지 마세요.

**Go SDK 예시**:
```go
client, err := warden.NewClient(warden.DefaultOptions().
    WithBaseURL("https://warden:8081").
    WithHMAC("key-id-1", os.Getenv("WARDEN_HMAC_SECRET")))
```

**검증 규칙**:
- Warden은 타임스탬프가 허용 범위 안에 있는지 검증합니다(기본값 ±60초)
- Warden은 키 ID를 포함한 모든 정규화 필드를 검증하고, 재사용된 논스를 거부합니다
- 서명 검증에 실패하면 `401 Unauthorized`를 반환합니다

### 설정 우선순위

1. **mTLS**: TLS 인증서가 설정되어 있으면 mTLS가 먼저 사용됩니다
2. **HMAC**: mTLS가 설정되어 있지 않으면 HMAC 서명이 사용됩니다
3. **API 키**: 둘 다 설정되어 있지 않으면 API 키 인증으로 폴백합니다(서비스 간 호출에는 권장되지 않음)

### 보안 권장 사항

1. **운영 환경**: 서비스 간 인증에는 mTLS 사용을 강력히 권장합니다
2. **키 관리**: 키와 인증서 보관에는 키 관리 서비스(예: HashiCorp Vault)를 사용하세요
3. **키 교체**: HMAC 키와 TLS 인증서를 정기적으로 교체하세요
4. **네트워크 격리**: 가능하다면 네트워크 정책으로 Warden 접근을 Stargate에서만 허용하세요

## 관련 문서

- [설정 문서](CONFIGURATION.md) - 보안 관련 설정 옵션 알아보기
- [배포 문서](DEPLOYMENT.md) - 운영 환경 배포 권장 사항 알아보기
- [API 문서](API.md) - API 보안 기능 알아보기
- [아키텍처 문서](ARCHITECTURE.md) - 서비스 연동 아키텍처 알아보기

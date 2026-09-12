# 배포 문서

> 🌐 **Language / 语言**: [English](../enUS/DEPLOYMENT.md) | [中文](../zhCN/DEPLOYMENT.md) | [Français](../frFR/DEPLOYMENT.md) | [Italiano](../itIT/DEPLOYMENT.md) | [日本語](../jaJP/DEPLOYMENT.md) | [Deutsch](../deDE/DEPLOYMENT.md) | [한국어](DEPLOYMENT.md)

이 문서는 Docker 배포, 로컬 배포 등 Warden 서비스를 배포하는 방법을 설명합니다.

## 사전 요구 사항

- Go 1.27 이상([go.mod](../../go.mod) 참고)
- Redis(분산 잠금 및 캐시용)
- Docker(선택. 컨테이너 배포용)

## Docker 배포

> 🚀 **빠른 배포**: 완전한 Docker Compose 설정 예시는 [예제 디렉터리](../../example/README.md) / [示例目录](../../example/README.md)를 확인하세요.
> - [간단한 예시](../../example/basic/docker-compose.yml) / [简单示例](../../example/basic/docker-compose.yml) - 기본 Docker Compose 설정
> - [고급 예시](../../example/advanced/docker-compose.yml) / [复杂示例](../../example/advanced/docker-compose.yml) - 모의 API를 포함한 완전한 설정

### 사전 빌드된 이미지 사용(권장)

Warden은 사전 빌드된 Docker 이미지를 제공하며, GitHub Container Registry(GHCR)에서 바로 받을 수 있습니다. 직접 빌드할 필요가 없습니다.

```bash
# 최신 버전 이미지 받기
docker pull ghcr.io/soulteary/warden:latest

# 컨테이너 실행
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

> 💡 **팁**: 사전 빌드된 이미지를 사용하면 로컬 빌드 환경 없이도 바로 시작할 수 있습니다. 이미지는 자동으로 갱신되므로 항상 최신 버전을 사용할 수 있습니다.

### Docker Compose 사용

1. **환경 변수 파일 준비**
   
   프로젝트 루트 디렉터리에 `.env.example` 파일이 있다면 복사할 수 있습니다.
   ```bash
   cp .env.example .env
   ```
   
   `.env.example` 파일이 없다면 다음 내용으로 `.env` 파일을 직접 만들 수 있습니다.
   ```env
   # 서버 설정
   PORT=8081
   
   # Redis 설정
   REDIS=warden-redis:6379
   # Redis 비밀번호(선택. 설정 파일 대신 환경 변수 사용을 권장)
   # REDIS_PASSWORD=your-redis-password
   # 또는 비밀번호 파일 사용(더 안전함)
   # REDIS_PASSWORD_FILE=/path/to/redis-password.txt
   
   # 원격 데이터 API
   CONFIG=http://example.com/api/data.json
   # 원격 설정 API 인증 키
   KEY=Bearer your-token-here
   
   # 작업 설정
   INTERVAL=5
   
   # 애플리케이션 모드
   MERGE_MODE=DEFAULT
   
   # HTTP 클라이언트 설정(선택)
   # HTTP_TIMEOUT=5
   # HTTP_MAX_IDLE_CONNS=100
   # HTTP_INSECURE_TLS=false
   
   # API 키(API 인증용. 운영 환경에서는 필수)
   API_KEY=your-api-key-here
   
   # 상태 확인 IP 허용 목록(선택. 쉼표 구분)
   # HEALTH_CHECK_IP_WHITELIST=127.0.0.1,::1,10.0.0.0/8
   
   # 신뢰하는 프록시 IP 목록(선택. 쉼표 구분. 리버스 프록시 환경용)
   # TRUSTED_PROXY_IPS=127.0.0.1,10.0.0.1
   
   # 로그 수준(선택)
   # LOG_LEVEL=info
   ```
   
   > ⚠️ **보안 참고**: `.env` 파일에는 민감한 정보가 들어 있습니다. 버전 관리에 커밋하지 마세요. `.env` 파일은 이미 `.gitignore`로 제외되어 있습니다. 위 내용을 템플릿으로 삼아 `.env` 파일을 만드세요.

2. **서비스 시작**
```bash
docker-compose up -d
```

### 이미지 직접 빌드

```bash
docker build -f docker/Dockerfile -t warden-release .
```

### 컨테이너 실행

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

## 로컬 배포

### 1. 프로젝트 클론

```bash
git clone <repository-url>
cd warden
```

### 2. 의존성 설치

```bash
go mod download
```

### 3. 로컬 데이터 파일 설정

`data.json` 파일을 만듭니다(`data.example.json` 참고).
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

**참고**: `data.json` 파일은 다음 필드를 지원합니다.
- `phone`(필수): 사용자 전화번호
- `mail`(필수): 사용자 이메일 주소
- `user_id`(선택): 사용자 고유 식별자. 제공하지 않으면 자동 생성됩니다
- `status`(선택): 사용자 상태. "active", "inactive", "suspended" 등. 값을 생략하면 기본값은 "inactive"입니다
- `scope`(선택): 사용자 권한 범위 배열. 예: `["read", "write"]`
- `role`(선택): 사용자 역할. "admin", "user" 등

완전한 예시는 `data.example.json` 파일을 참고하세요.

### 4. 서비스 실행

```bash
go run .
```

## 운영 환경 배포 권장 사항

### 1. 리버스 프록시 사용

운영 환경에서는 Nginx나 Traefik 같은 리버스 프록시 사용을 권장합니다.

**Nginx 설정 예시**:
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

### 2. HTTPS 사용

운영 환경에서는 반드시 HTTPS를 사용해야 합니다. 다음 방법으로 구현할 수 있습니다.

- Let's Encrypt 무료 인증서 사용
- 리버스 프록시(예: Nginx)에서 SSL/TLS 처리
- 환경 변수 `TRUSTED_PROXY_IPS`를 설정해 클라이언트의 실제 IP를 올바르게 얻기

### 3. 모니터링 설정

- Prometheus로 지표 수집(`/metrics` 엔드포인트 이용)
- 상태 확인 설정(`/health` 엔드포인트 이용)
- 로그 수집과 분석 체계 구축

### 4. 고가용성 배포

- 여러 인스턴스를 배포하고 로드 밸런서로 요청을 분산
- 공유 Redis 인스턴스를 사용해 데이터 일관성 확보
- 자동 재시작과 장애 조치 구성

### 5. 리소스 제한

Docker Compose 또는 Kubernetes에서 리소스 제한을 설정합니다.

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

## Kubernetes 배포

### 기본 배포

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

## 성능 최적화

### 1. Redis 설정

- Redis 영속화 사용(RDB 또는 AOF)
- 적절한 Redis 메모리 제한 설정
- 필요하다면 Redis 클러스터 사용

### 2. 애플리케이션 설정

- `HTTP_MAX_IDLE_CONNS`를 조정해 연결 풀 최적화
- 적절한 `INTERVAL`을 설정해 실시간성과 효율의 균형 맞추기
- 알맞은 병합 모드(`MERGE_MODE`) 사용

### 3. 모니터링과 튜닝

wrk 부하 테스트 결과 기준(30초 테스트, 스레드 16개, 연결 100개):

```
Requests/sec:   5038.81
Transfer/sec:   38.96MB
Average Latency: 21.30ms
Max Latency:     226.09ms
```

실제 부하에 맞춰 설정 매개변수를 조정하세요.

## 선택적 연동 배포(Stargate/Herald와 함께)

Warden은 단독으로 배포해 사용할 수도 있고, 선택적으로 Stargate 및 Herald와 연동할 수도 있습니다. 아래는 선택적 연동 배포 설정 예시입니다.

**참고**: 아래의 연동 배포 시나리오는 선택 사항이며, Warden은 완전히 독립적으로 배포해 사용할 수 있습니다.

### Docker Compose 연동 예시

Stargate + Warden + Herald를 연동하는 완전한 배포 설정:

```yaml
version: '3.8'

services:
  # Warden 서비스
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
      - MERGE_MODE=DEFAULT
      # 서비스 간 인증 설정(HMAC 예시)
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

  # Warden용 Redis
  warden-redis:
    image: redis:7.4-alpine
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

  # Stargate 서비스(설정 예시)
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

  # Herald 서비스(설정 예시)
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

  # Herald용 Redis
  herald-redis:
    image: redis:7.4-alpine
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

### 환경 변수 설정

`.env` 파일을 만듭니다.

```bash
# Warden API 키
WARDEN_API_KEY=your-warden-api-key-here

# Warden HMAC 키(JSON 형식)
WARDEN_HMAC_KEYS='{"key-id-1":"0123456789abcdef0123456789abcdef"}'

# Stargate가 사용하는 HMAC 비밀 값(WARDEN_HMAC_KEYS의 키에 대응)
WARDEN_HMAC_SECRET=0123456789abcdef0123456789abcdef
```

### 네트워크 설정

서로 통신할 수 있도록 모든 서비스는 같은 Docker 네트워크에 있어야 합니다.

- **Warden**: `8081` 포트에서 수신하며 Stargate가 호출합니다
- **Stargate**: `8080` 포트에서 수신하며 Traefik의 forwardAuth 서비스 역할을 합니다
- **Herald**: `8082` 포트에서 수신하며 Stargate가 호출합니다

### 서비스 의존 관계

- **Stargate**는 **Warden**과 **Herald**에 의존합니다
- **Warden**은 **warden-redis**에 의존합니다(Redis를 활성화한 경우. 선택)
- **Herald**는 **herald-redis**에 의존합니다

### 상태 확인

정상 동작을 보장하기 위해 모든 서비스에 상태 확인을 설정해야 합니다.

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl --fail http://localhost:8081/healthcheck || exit 1"]
  interval: 10s
  timeout: 1s
  retries: 3
```

### 운영 환경 권장 사항

1. **독립된 Redis 인스턴스 사용**: 데이터 충돌을 피하기 위해 Warden과 Herald는 각각 독립된 Redis 인스턴스를 사용해야 합니다
2. **서비스 간 인증 설정**: 운영 환경에서는 mTLS 또는 HMAC 서명을 반드시 설정해야 합니다
3. **키 관리 서비스 사용**: HashiCorp Vault 등 유사한 서비스로 키와 인증서를 관리하세요
4. **네트워크 격리**: Docker 네트워크 정책으로 서비스 간 접근을 제한하세요
5. **모니터링과 로깅**: 통합된 모니터링 및 로그 수집 체계를 구성하세요

### Kubernetes 연동 배포

Kubernetes에 배포할 때는 다음을 권장합니다.

1. **Service 사용**: 각 서비스마다 Kubernetes Service를 만듭니다
2. **ConfigMap과 Secret 사용**: 설정과 키를 저장합니다
3. **NetworkPolicy 사용**: 서비스 간 네트워크 접근을 제한합니다
4. **Ingress 사용**: Traefik Ingress를 구성해 Stargate로 라우팅합니다

Kubernetes 설정 예시:

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

## 관련 문서

- [설정 문서](CONFIGURATION.md) - 자세한 설정 옵션 알아보기
- [보안 문서](SECURITY.md) - 보안 설정과 모범 사례 알아보기
- [아키텍처 설계 문서](ARCHITECTURE.md) - 시스템 아키텍처 이해하기
- [API 문서](API.md) - API 인터페이스와 연동 예시 알아보기

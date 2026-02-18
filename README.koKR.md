# Warden

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org)
[![codecov](https://codecov.io/gh/soulteary/warden/branch/main/graph/badge.svg)](https://codecov.io/gh/soulteary/warden)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/warden)](https://goreportcard.com/report/github.com/soulteary/warden)

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md) | [Français](README.frFR.md) | [Italiano](README.itIT.md) | [日本語](README.jaJP.md) | [Deutsch](README.deDE.md) | [한국어](README.koKR.md)

로컬 및 원격 구성 소스에서 데이터 동기화 및 병합을 지원하는 고성능 허용 목록(AllowList) 사용자 데이터 서비스입니다.

![Warden](.github/assets/banner.jpg)

> **Warden**（看守者）—— 스타게이트의 수호자로서 누가 통과할 수 있고 누가 거부될지 결정합니다. 스타게이트의 수호자가 스타게이트를 지키는 것처럼, Warden은 허용 목록을 지키며 승인된 사용자만 통과할 수 있도록 합니다.

## 📋 개요

Warden은 Go로 개발된 경량 HTTP API 서비스로, 주로 허용 목록 사용자 데이터(전화번호 및 이메일 주소)를 제공하고 관리하는 데 사용됩니다. 이 서비스는 로컬 구성 파일과 원격 API에서 데이터를 가져오는 것을 지원하며, 실시간 성능과 신뢰성을 보장하기 위한 여러 데이터 병합 전략을 제공합니다.

Warden은 **독립적으로 사용**할 수 있으며, 더 큰 인증 아키텍처의 일부로 다른 서비스(Stargate 및 Herald 등)와 통합할 수도 있습니다. 자세한 아키텍처 정보는 [아키텍처 문서](docs/enUS/ARCHITECTURE.md)를 참조하세요.

## ✨ 핵심 기능

- 🚀 **고성능**: 평균 지연 시간 21ms로 초당 5000개 이상의 요청 지원
- 🔄 **다중 데이터 소스**: 로컬 구성 파일과 원격 API
- 🎯 **유연한 전략**: 6가지 데이터 병합 모드(원격 우선, 로컬 우선, 원격 전용, 로컬 전용 등)
- ⏰ **예약 업데이트**: Redis 분산 잠금을 사용한 자동 데이터 동기화
- 📦 **컨테이너화 배포**: 완전한 Docker 지원, 즉시 사용 가능
- 🌐 **다국어 지원**: 7개 언어를 지원하며 자동 언어 감지

## 🚀 빠른 시작

### 옵션 1: Docker(권장)

가장 빠른 시작 방법은 사전 빌드된 Docker 이미지를 사용하는 것입니다:

```bash
# 최신 이미지 가져오기
docker pull ghcr.io/soulteary/warden:latest

# 데이터 파일 생성
cat > data.json <<EOF
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
EOF

# 컨테이너 실행
docker run -d \
  -p 8081:8081 \
  -v $(pwd)/data.json:/app/data.json:ro \
  -e API_KEY=your-api-key-here \
  ghcr.io/soulteary/warden:latest
```

> 💡 **팁**: Docker Compose를 사용한 전체 예제는 [예제 디렉토리](example/README.md)를 참조하세요.

### 옵션 2: 소스에서

1. **클론 및 빌드**
```bash
git clone <repository-url>
cd warden
go mod download
```

2. **데이터 파일 생성**
`data.json` 파일 생성(`data.example.json` 참조):
```json
[
    {
        "phone": "13800138000",
        "mail": "admin@example.com"
    }
]
```

3. **서비스 실행**
```bash
go run . --api-key your-api-key-here
```

## ⚙️ 필수 구성

Warden은 명령줄 인수, 환경 변수 및 구성 파일을 통한 구성을 지원합니다. 다음은 가장 필수적인 설정입니다:

| 설정 | 환경 변수 | 설명 | 필수 |
|------|----------|------|------|
| 포트 | `PORT` | HTTP 서버 포트(기본값: 8081) | 아니오 |
| API 키 | `API_KEY` | API 인증 키(프로덕션에 권장) | 권장 |
| Redis | `REDIS` | 캐싱 및 분산 잠금을 위한 Redis 주소(예: `localhost:6379`) | 선택 사항 |
| 데이터 파일 | - | 로컬 데이터 파일 경로(기본값: `data.json`) | 예* |
| 원격 구성 | `CONFIG` | 데이터 가져오기를 위한 원격 API URL | 선택 사항 |

\* 원격 API를 사용하지 않는 경우 필수

전체 구성 옵션은 [구성 문서](docs/enUS/CONFIGURATION.md)를 참조하세요.

## 📡 API 사용

Warden은 사용자 목록 쿼리, 페이지네이션 및 상태 확인을 위한 RESTful API를 제공합니다. 서비스는 쿼리 매개변수 `?lang=xx` 또는 `Accept-Language` 헤더를 통한 다국어 응답을 지원합니다.

**예제**:
```bash
# 사용자 쿼리
curl -H "X-API-Key: your-key" "http://localhost:8081/"

# 상태 확인
curl "http://localhost:8081/health"
```

전체 API 문서는 [API 문서](docs/enUS/API.md) 또는 [OpenAPI 사양](openapi.yaml)을 참조하세요.

## 📊 성능

wrk 스트레스 테스트 기반(30초, 16스레드, 100연결):
- **요청/초**: 5038.81
- **평균 지연 시간**: 21.30ms
- **최대 지연 시간**: 226.09ms

## 📚 문서

### 핵심 문서

- **[아키텍처](docs/enUS/ARCHITECTURE.md)** - 기술 아키텍처 및 설계 결정
- **[API 참조](docs/enUS/API.md)** - 전체 API 엔드포인트 문서
- **[구성](docs/enUS/CONFIGURATION.md)** - 구성 참조 및 예제
- **[배포](docs/enUS/DEPLOYMENT.md)** - 배포 가이드(Docker, Kubernetes 등)

### 추가 리소스

- **[개발 가이드](docs/enUS/DEVELOPMENT.md)** - 개발 환경 설정 및 기여 가이드
- **[보안](docs/enUS/SECURITY.md)** - 보안 기능 및 모범 사례
- **[SDK](docs/enUS/SDK.md)** - Go SDK 사용 문서
- **[예제](example/README.md)** - 빠른 시작 예제(기본 및 고급)

## 📄 라이선스

자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 🤝 기여

Issues 및 Pull Request 제출을 환영합니다! 가이드라인은 [CONTRIBUTING.md](docs/enUS/CONTRIBUTING.md)를 참조하세요.

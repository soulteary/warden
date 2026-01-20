# 문서 인덱스

Warden AllowList 사용자 데이터 서비스 문서에 오신 것을 환영합니다.

## 🌐 다국어 문서

- [English](../enUS/README.md) | [中文](../zhCN/README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](README.md)

## 📚 문서 목록

### 핵심 문서

- **[README.md](../../README.koKR.md)** - 프로젝트 개요 및 빠른 시작 가이드
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 기술 아키텍처 및 설계 결정

### 상세 문서

- **[API.md](API.md)** - 완전한 API 엔드포인트 문서
  - 사용자 목록 쿼리 엔드포인트
  - 페이지네이션 기능
  - 헬스 체크 엔드포인트
  - 오류 응답 형식

- **[CONFIGURATION.md](CONFIGURATION.md)** - 구성 참조
  - 구성 방법
  - 필수 구성 항목
  - 선택적 구성 항목
  - 데이터 병합 전략
  - 구성 예제
  - 구성 모범 사례

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - 배포 가이드
  - Docker 배포 (GHCR 이미지 포함)
  - Docker Compose 배포
  - 로컬 배포
  - 프로덕션 환경 배포
  - Kubernetes 배포
  - 성능 최적화

- **[DEVELOPMENT.md](DEVELOPMENT.md)** - 개발 가이드
  - 개발 환경 설정
  - 코드 구조 설명
  - 테스트 가이드
  - 기여 가이드

- **[SDK.md](SDK.md)** - SDK 사용 문서
  - Go SDK 설치 및 사용
  - API 인터페이스 설명
  - 예제 코드

- **[SECURITY.md](SECURITY.md)** - 보안 문서
  - 보안 기능
  - 보안 구성
  - 모범 사례

- **[CODE_STYLE.md](CODE_STYLE.md)** - 코드 스타일 가이드
  - 코드 표준
  - 명명 규칙
  - 모범 사례

## 🌐 다국어 지원

Warden은 완전한 국제화(i18N) 기능을 지원합니다. 모든 API 응답, 오류 메시지 및 로그가 국제화를 지원합니다.

### 지원되는 언어

- 🇺🇸 영어 (en) - 기본 언어
- 🇨🇳 중국어 (zh)
- 🇫🇷 프랑스어 (fr)
- 🇮🇹 이탈리아어 (it)
- 🇯🇵 일본어 (ja)
- 🇩🇪 독일어 (de)
- 🇰🇷 한국어 (ko)

### 언어 감지

Warden은 다음 우선순위로 두 가지 언어 감지 방법을 지원합니다:

1. **쿼리 매개변수**: URL 쿼리 매개변수 `?lang=ko`로 언어 지정
2. **Accept-Language 헤더**: 브라우저 또는 클라이언트 언어 기본 설정 자동 감지
3. **기본 언어**: 지정되지 않은 경우 영어

### 사용 예제

#### 쿼리 매개변수로 언어 지정

```bash
# 한국어 사용
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=ko"

# 일본어 사용
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=ja"

# 프랑스어 사용
curl -H "X-API-Key: your-key" "http://localhost:8081/?lang=fr"
```

#### Accept-Language 헤더로 자동 감지

```bash
# 브라우저가 자동으로 Accept-Language 헤더를 전송
curl -H "X-API-Key: your-key" \
     -H "Accept-Language: ko-KR,ko;q=0.9,en;q=0.8" \
     "http://localhost:8081/"
```

### 국제화 범위

다음 내용이 여러 언어를 지원합니다:

- ✅ API 오류 응답 메시지
- ✅ HTTP 상태 코드 오류 메시지
- ✅ 로그 메시지 (요청 컨텍스트 기반)
- ✅ 구성 및 경고 메시지

### 기술 구현

- 요청 컨텍스트를 사용하여 언어 정보를 저장하고 전역 상태를 피함
- 스레드 안전한 언어 전환 지원
- 영어로 자동 폴백 (번역을 찾을 수 없는 경우)
- 모든 번역이 코드에 내장되어 있으며 외부 파일이 필요 없음

### 개발 참고 사항

새로운 번역을 추가하거나 기존 번역을 수정하려면 `internal/i18n/i18n.go` 파일의 `translations` 맵을 편집하세요.

## 🚀 빠른 탐색

### 시작하기

1. [README.koKR.md](../../README.koKR.md)를 읽어 프로젝트 이해하기
2. [빠른 시작](../../README.koKR.md#빠른-시작) 섹션 확인하기
3. [구성](../../README.koKR.md#구성)을 참조하여 서비스 구성하기

### 개발자

1. [ARCHITECTURE.md](ARCHITECTURE.md)를 읽어 아키텍처 이해하기
2. [API.md](API.md)를 확인하여 API 인터페이스 이해하기
3. [개발 가이드](../../README.koKR.md#개발-가이드)를 참조하여 개발하기

### 운영

1. [DEPLOYMENT.md](DEPLOYMENT.md)를 읽어 배포 방법 이해하기
2. [CONFIGURATION.md](CONFIGURATION.md)를 확인하여 구성 옵션 이해하기
3. [성능 최적화](DEPLOYMENT.md#성능-최적화)를 참조하여 서비스 최적화하기

## 📖 문서 구조

```
warden/
├── README.md              # 프로젝트 메인 문서 (한국어)
├── README.koKR.md         # 프로젝트 메인 문서 (한국어)
├── docs/
│   ├── enUS/
│   │   ├── README.md       # 문서 인덱스 (영어)
│   │   ├── ARCHITECTURE.md # 아키텍처 문서 (영어)
│   │   ├── API.md          # API 문서 (영어)
│   │   ├── CONFIGURATION.md # 구성 참조 (영어)
│   │   ├── DEPLOYMENT.md   # 배포 가이드 (영어)
│   │   ├── DEVELOPMENT.md  # 개발 가이드 (영어)
│   │   ├── SDK.md          # SDK 문서 (영어)
│   │   ├── SECURITY.md     # 보안 문서 (영어)
│   │   └── CODE_STYLE.md   # 코드 스타일 (영어)
│   └── koKR/
│       ├── README.md       # 문서 인덱스 (한국어, 이 파일)
│       ├── ARCHITECTURE.md # 아키텍처 문서 (한국어)
│       ├── API.md          # API 문서 (한국어)
│       ├── CONFIGURATION.md # 구성 참조 (한국어)
│       ├── DEPLOYMENT.md   # 배포 가이드 (한국어)
│       ├── DEVELOPMENT.md  # 개발 가이드 (한국어)
│       ├── SDK.md          # SDK 문서 (한국어)
│       ├── SECURITY.md     # 보안 문서 (한국어)
│       └── CODE_STYLE.md   # 코드 스타일 (한국어)
└── ...
```

## 🔍 주제별 검색

### 구성 관련

- 환경 변수 구성: [CONFIGURATION.md](CONFIGURATION.md)
- 데이터 병합 전략: [CONFIGURATION.md](CONFIGURATION.md)
- 구성 예제: [CONFIGURATION.md](CONFIGURATION.md)

### API 관련

- API 엔드포인트 목록: [API.md](API.md)
- 오류 처리: [API.md](API.md)
- 페이지네이션 기능: [API.md](API.md)

### 배포 관련

- Docker 배포: [DEPLOYMENT.md#docker-배포](DEPLOYMENT.md#docker-배포)
- GHCR 이미지: [DEPLOYMENT.md#사전-구축된-이미지-사용-권장](DEPLOYMENT.md#사전-구축된-이미지-사용-권장)
- 프로덕션 환경: [DEPLOYMENT.md#프로덕션-환경-배포-권장-사항](DEPLOYMENT.md#프로덕션-환경-배포-권장-사항)
- Kubernetes: [DEPLOYMENT.md#kubernetes-배포](DEPLOYMENT.md#kubernetes-배포)

### 아키텍처 관련

- 기술 스택: [ARCHITECTURE.md](ARCHITECTURE.md)
- 프로젝트 구조: [ARCHITECTURE.md](ARCHITECTURE.md)
- 핵심 구성 요소: [ARCHITECTURE.md](ARCHITECTURE.md)

## 💡 사용 권장 사항

1. **처음 사용하는 사용자**: [README.koKR.md](../../README.koKR.md)부터 시작하여 빠른 시작 가이드를 따르세요
2. **서비스 구성**: [CONFIGURATION.md](CONFIGURATION.md)를 참조하여 모든 구성 옵션을 이해하세요
3. **서비스 배포**: [DEPLOYMENT.md](DEPLOYMENT.md)를 확인하여 배포 방법을 이해하세요
4. **확장 개발**: [ARCHITECTURE.md](ARCHITECTURE.md)를 읽어 아키텍처 설계를 이해하세요
5. **SDK 통합**: [SDK.md](SDK.md)를 참조하여 SDK 사용 방법을 배우세요

## 📝 문서 업데이트

문서는 프로젝트가 발전함에 따라 지속적으로 업데이트됩니다. 오류를 발견하거나 추가가 필요한 경우 Issue 또는 Pull Request를 제출하세요.

## 🤝 기여

문서 개선을 환영합니다:

1. 오류나 개선이 필요한 영역 찾기
2. 문제를 설명하는 Issue 제출
3. 또는 직접 Pull Request 제출

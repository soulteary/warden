# 기여 가이드

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](CONTRIBUTING.md)

Warden 프로젝트에 관심을 가져 주셔서 감사합니다! 모든 형태의 기여를 환영합니다.


## 📋 목차

- [기여 방법](#기여-방법)
- [개발 환경 설정](#개발-환경-설정)
- [코드 표준](#코드-표준)
- [커밋 표준](#커밋-표준)
- [Pull Request 프로세스](#pull-request-프로세스)
- [버그 보고 및 기능 요청](#버그-보고-및-기능-요청)

## 🚀 기여 방법

다음과 같은 방법으로 기여할 수 있습니다:

- **버그 보고**: GitHub Issues에서 문제 보고
- **기능 제안**: GitHub Issues에서 새로운 기능 아이디어 제안
- **코드 제출**: Pull Request를 통해 코드 개선 사항 제출
- **문서 개선**: 프로젝트 문서 개선에 도움
- **질문 답변**: Issues에서 다른 사용자 도움

이 프로젝트에 참여할 때는 모든 기여자를 존중하고, 건설적인 비판을 수용하며, 프로젝트에 가장 좋은 것에 집중해 주세요.

## 🛠️ 개발 환경 설정

### 필수 요구사항

- Go 1.25 이상
- Redis (테스트용)
- Git

### 빠른 시작

```bash
# 1. 프로젝트 포크 및 클론
git clone https://github.com/your-username/warden.git
cd warden

# 2. 업스트림 저장소 추가
git remote add upstream https://github.com/soulteary/warden.git

# 3. 종속성 설치
go mod download

# 4. 테스트 실행
go test ./...

# 5. 로컬 서비스 시작 (Redis가 실행 중인지 확인)
go run .
```

## 📝 코드 표준

다음 코드 표준을 따르세요:

1. **Go 공식 코드 표준 준수**: [Effective Go](https://go.dev/doc/effective_go)
2. **코드 포맷팅**: `go fmt ./...` 실행
3. **코드 검사**: `golangci-lint` 또는 `go vet ./...` 사용
4. **테스트 작성**: 새 기능에는 테스트가 포함되어야 합니다
5. **주석 추가**: 공개 함수 및 타입에는 문서 주석이 있어야 합니다
6. **상수 명명**: 모든 상수는 `ALL_CAPS` (UPPER_SNAKE_CASE) 명명 스타일을 사용해야 합니다

자세한 코드 스타일 가이드는 [CODE_STYLE.md](CODE_STYLE.md)를 참조하세요.

## 📦 커밋 표준

### 커밋 메시지 형식

[Conventional Commits](https://www.conventionalcommits.org/) 표준을 사용합니다:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### 타입

- `feat`: 새 기능
- `fix`: 버그 수정
- `docs`: 문서 업데이트
- `style`: 코드 형식 조정 (코드 실행에 영향을 주지 않음)
- `refactor`: 코드 리팩토링
- `perf`: 성능 최적화
- `test`: 테스트 관련
- `chore`: 빌드 프로세스 또는 보조 도구 변경

## 🔄 Pull Request 프로세스

### Pull Request 생성

```bash
# 1. 기능 브랜치 생성
git checkout -b feature/your-feature-name

# 2. 변경 사항 적용 및 커밋
git add .
git commit -m "feat: 새 기능 추가"

# 3. 업스트림 코드 동기화
git fetch upstream
git rebase upstream/main

# 4. 브랜치 푸시 및 PR 생성
git push origin feature/your-feature-name
```

### Pull Request 체크리스트

Pull Request를 제출하기 전에 다음을 확인하세요:

- [ ] 코드가 프로젝트 코드 표준을 따름
- [ ] 모든 테스트 통과 (`go test ./...`)
- [ ] 코드가 포맷팅됨 (`go fmt ./...`)
- [ ] 필요한 테스트가 추가됨
- [ ] 관련 문서가 업데이트됨
- [ ] 커밋 메시지가 [커밋 표준](#커밋-표준)을 따름
- [ ] 코드가 lint 검사를 통과함

모든 Pull Request는 코드 검토가 필요합니다. 검토 의견에 신속하게 응답해 주세요.

## 🐛 버그 보고 및 기능 요청

Issue를 생성하기 전에 기존 Issues를 검색하여 문제나 기능이 보고되지 않았는지 확인하세요.

## 🎯 시작하기

기여하고 싶지만 어디서 시작해야 할지 모르겠다면 다음에 집중할 수 있습니다:

- `good first issue`로 레이블이 지정된 Issues
- `help wanted`로 레이블이 지정된 Issues
- 코드의 `TODO` 주석
- 문서 개선 (오타 수정, 명확성 개선, 예제 추가)

질문이 있으면 기존 Issues와 Pull Requests를 확인하거나 관련 Issue에서 질문하세요.

---

Warden 프로젝트에 기여해 주셔서 다시 한 번 감사합니다! 🎉

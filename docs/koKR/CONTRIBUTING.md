# 기여 가이드

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](CONTRIBUTING.md)

Warden 프로젝트에 관심을 가져 주셔서 감사합니다. 모든 형태의 기여를 환영합니다.

## 📋 목차

- [기여 방법](#기여-방법)
- [개발 환경 설정](#개발-환경-설정)
- [코드 표준](#코드-표준)
- [번역 정책](#번역-정책)
- [커밋 표준](#커밋-표준)
- [풀 리퀘스트 절차](#풀-리퀘스트-절차)
- [버그 신고와 기능 요청](#버그-신고와-기능-요청)

## 🚀 기여 방법

다음과 같은 방법으로 기여할 수 있습니다.

- **버그 신고**: GitHub Issues에 문제를 신고합니다
- **기능 제안**: GitHub Issues에 새로운 기능 아이디어를 제안합니다
- **코드 제출**: 풀 리퀘스트로 코드 개선을 제출합니다
- **문서 개선**: 프로젝트 문서 개선을 돕습니다
- **질문에 답변**: Issues에서 다른 사용자를 도와줍니다

이 프로젝트에 참여할 때는 모든 기여자를 존중하고, 건설적인 비판을 받아들이며, 프로젝트에 가장 좋은 것에 집중해 주세요.

## 🛠️ 개발 환경 설정

### 사전 요구 사항

- Go 1.27 이상
- Redis(테스트용)
- Git

### 빠른 시작

```bash
# 1. 프로젝트를 포크하고 클론
git clone https://github.com/your-username/warden.git
cd warden

# 2. 업스트림 저장소 추가
git remote add upstream https://github.com/soulteary/warden.git

# 3. 의존성 설치
go mod download

# 4. 테스트 실행
go test ./...

# 5. 로컬에서 서비스 시작(Redis가 실행 중인지 확인)
go run .
```

## 📝 코드 표준

다음 코드 표준을 따라 주세요.

1. **Go 공식 코드 표준 준수**: [Effective Go](https://go.dev/doc/effective_go)
2. **코드 형식 정리**: `go fmt ./...` 실행
3. **코드 검사**: `golangci-lint` 또는 `go vet ./...` 사용
4. **테스트 작성**: 새로운 기능에는 반드시 테스트를 포함
5. **주석 추가**: 공개 함수와 타입에는 문서 주석이 필요
6. **상수 명명**: 모든 상수는 `ALL_CAPS`(UPPER_SNAKE_CASE) 스타일을 사용

자세한 코드 스타일 지침은 [CODE_STYLE.md](CODE_STYLE.md)를 참고하세요.

## 🌐 번역 정책

Warden은 문서와 런타임 메시지를 7개 언어로 제공합니다. 이 언어들은 **모두 같은 수준으로
관리되지 않으며**, 그렇지 않은 척하는 것이 바로 다섯 개 로케일이 번역 키 18개와 여러 문서를
소리 없이 놓치게 된 원인입니다.

**등급**

| 등급 | 언어 | 기대 수준 |
| --- | --- | --- |
| 정본 | 영어(`enUS`), 간체 중국어(`zhCN`) | 변경 사항과 같은 풀 리퀘스트에서 함께 갱신합니다. 동작을 바꾸면서 둘 다 갱신하지 않은 PR은 완성된 것이 아닙니다. |
| 최선 노력 | `deDE`, `frFR`, `itIT`, `jaJP`, `koKR` | 뒤처져도 됩니다. 뒤처진 문서에는 독자를 정본 버전으로 안내하는 배너를 표시합니다. |

**런타임 문자열은 최선 노력 대상이 아닙니다.** `locales/*.json`은 `go test ./locales/`로
강제되며, 어느 로케일이든 다음에 해당하면 실패합니다.

- `en.json`이 정의한 키가 빠져 있음(또는 거기에 없는 키를 정의함)
- `printf` 동사(`%s`, `%d`)의 순서가 영어 원본과 다름
- 값이 영어와 바이트 단위로 동일함(번역되지 않은 문자열)

키가 없으면 런타임에 영어로 폴백하므로 겉으로는 아무것도 깨지지 않습니다. 바로 그렇기 때문에
이 검사가 존재합니다. 사용자에게 보이는 메시지를 추가할 때는 같은 커밋에서 **일곱 개 전부**의
로케일 파일에 키를 추가하세요. 값이 정당하게 영어와 동일하다면(외래어나 프로토콜 이름 등),
검사를 지우는 대신 `locales/locales_test.go`의 `intentionallyIdentical`에 주석과 함께
등록하세요.

번역된 각 문서가 `enUS`에서 얼마나 벗어났는지는 `make docs-parity`로 확인할 수 있습니다.

## 📦 커밋 표준

### 커밋 메시지 형식

[Conventional Commits](https://www.conventionalcommits.org/) 표준을 사용합니다.

```
<type>(<scope>): <subject>

<body>

<footer>
```

### type 종류

- `feat`: 새로운 기능
- `fix`: 버그 수정
- `docs`: 문서 갱신
- `style`: 코드 형식 조정(실행 결과에 영향 없음)
- `refactor`: 코드 리팩터링
- `perf`: 성능 최적화
- `test`: 테스트 관련
- `chore`: 빌드 과정이나 보조 도구 변경

### 예시

```
feat(cache): Add Redis cache support

Implemented Redis-based distributed cache, supporting data persistence and multi-instance sharing.

Closes #123
```

```
fix(router): Fix pagination parameter validation issue

Fixed the issue where incorrect status code was returned when page_size exceeds maximum value.

Fixes #456
```

## 🔄 풀 리퀘스트 절차

### 풀 리퀘스트 만들기

```bash
# 1. 기능 브랜치 생성
git checkout -b feature/your-feature-name

# 2. 변경 후 커밋
git add .
git commit -m "feat: Add new feature"

# 3. 업스트림 코드 동기화
git fetch upstream
git rebase upstream/main

# 4. 브랜치를 푸시하고 PR 생성
git push origin feature/your-feature-name
```

### 풀 리퀘스트 체크리스트

풀 리퀘스트를 제출하기 전에 다음을 확인해 주세요.

- [ ] 코드가 프로젝트의 코드 표준을 따름
- [ ] 모든 테스트가 통과함(`go test ./...`)
- [ ] 코드 형식이 정리됨(`go fmt ./...`)
- [ ] 필요한 테스트가 추가됨
- [ ] 관련 문서가 갱신됨
- [ ] 커밋 메시지가 [커밋 표준](#커밋-표준)을 따름
- [ ] 코드가 lint 검사를 통과함

모든 풀 리퀘스트는 코드 리뷰를 거칩니다. 리뷰 의견에는 신속히 응답해 주세요.

## 🐛 버그 신고와 기능 요청

Issue를 만들기 전에 기존 Issue를 검색하여 해당 문제나 기능이 이미 신고되지 않았는지 확인해 주세요.

### 버그 신고 템플릿

```markdown
**설명**
버그를 명확하고 간결하게 설명해 주세요.

**재현 단계**
1. '...' 실행
2. 오류 확인

**기대한 동작**
기대했던 동작을 명확하고 간결하게 설명해 주세요.

**실제 동작**
실제로 일어난 일을 명확하고 간결하게 설명해 주세요.

**환경 정보**
- 운영 체제: [예: macOS 12.0]
- Go 버전: [예: 1.27]
- Redis 버전: [예: 7.0]
```

### 기능 요청 템플릿

```markdown
**기능 설명**
원하는 기능을 명확하고 간결하게 설명해 주세요.

**문제 설명**
이 기능은 어떤 문제를 해결하나요? 왜 필요한가요?

**제안하는 해결 방안**
어떻게 구현되기를 바라는지 명확하고 간결하게 설명해 주세요.
```

## 🎯 시작하기

기여하고 싶지만 어디서 시작할지 모르겠다면 다음을 살펴보세요.

- `good first issue` 라벨이 붙은 Issue
- `help wanted` 라벨이 붙은 Issue
- 코드 안의 `TODO` 주석
- 문서 개선(오타 수정, 명확성 향상, 예시 추가)

궁금한 점이 있으면 기존 Issue와 풀 리퀘스트를 확인하거나 관련 Issue에서 질문해 주세요.

---

Warden 프로젝트에 기여해 주셔서 다시 한번 감사합니다! 🎉

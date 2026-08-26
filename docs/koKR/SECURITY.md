# 보안 문서

> 🌐 **Language / 语言**: [English](../enUS/SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](SECURITY.md)

이 문서는 Warden의 보안 기능, 보안 구성 및 모범 사례를 설명합니다.


## 구현된 보안 기능

1. **API 인증**: 민감한 엔드포인트를 보호하기 위한 API 키 인증 지원
2. **SSRF 보호**: 서버 측 요청 위조 공격을 방지하기 위해 원격 구성 URL을 엄격하게 검증
3. **입력 검증**: 주입 공격을 방지하기 위해 모든 입력 매개변수를 엄격하게 검증
4. **속도 제한**: DDoS 공격을 방지하기 위한 IP 기반 속도 제한
5. **TLS 검증**: 프로덕션 환경에서 TLS 인증서 검증 강제
6. **오류 처리**: 프로덕션 환경에서 정보 유출을 방지하기 위해 상세한 오류 정보 숨김
7. **보안 응답 헤더**: 보안 관련 HTTP 응답 헤더를 자동으로 추가
8. **IP 화이트리스트**: 상태 확인 엔드포인트에 대한 IP 화이트리스트 구성 지원
9. **구성 파일 검증**: 경로 순회 공격 방지
10. **JSON 크기 제한**: 메모리 고갈 공격을 방지하기 위해 JSON 응답 본문 크기 제한

## 보안 모범 사례

### 1. 프로덕션 환경 구성

**필수 구성**:
- `API_KEY` 환경 변수를 설정해야 합니다
- `MODE=production`을 설정하여 프로덕션 모드 활성화
- `TRUSTED_PROXY_IPS`를 구성하여 클라이언트 IP를 올바르게 가져옴
- `HEALTH_CHECK_IP_WHITELIST`를 사용하여 상태 확인 액세스 제한

### 2. 민감한 정보 관리

**권장 사항**:
- ✅ 환경 변수를 사용하여 비밀번호 및 키 저장
- ✅ 비밀번호 파일(`REDIS_PASSWORD_FILE`)을 사용하여 Redis 비밀번호 저장
- ✅ 구성 파일에서 자리 표시자 또는 주석 사용
- ✅ 구성 파일 권한이 올바르게 설정되었는지 확인 (예: `chmod 600`)

### 3. 네트워크 보안

**필수 구성**:
- 프로덕션 환경은 HTTPS를 사용해야 합니다
- 방화벽 규칙을 구성하여 액세스 제한
- 알려진 취약점을 수정하기 위해 종속성을 정기적으로 업데이트

## API 보안

### API 키 인증

일부 API 엔드포인트에는 API 키 인증이 필요합니다.

**인증 방법**:
1. **X-API-Key 헤더**:
   ```http
   X-API-Key: your-secret-api-key
   ```

2. **Authorization Bearer 헤더**:
   ```http
   Authorization: Bearer your-secret-api-key
   ```

### 속도 제한

기본적으로 API 요청은 속도 제한으로 보호됩니다:
- **제한**: 분당 60개 요청
- **창**: 1분
- **초과**: `429 Too Many Requests` 반환

## 취약점 보고

보안 취약점을 발견한 경우 다음을 통해 보고하세요:

1. **GitHub Security Advisory** (권장)
   - 저장소의 [Security](https://github.com/soulteary/warden/security) 탭으로 이동
   - "Report a vulnerability" 클릭
   - 보안 권고 양식 작성

2. **이메일** (GitHub Security Advisory를 사용할 수 없는 경우)
   - 프로젝트 유지 관리자에게 이메일 보내기
   - 취약점에 대한 자세한 설명 포함

## 관련 문서

- [구성 문서](CONFIGURATION.md) - 보안 관련 구성 옵션에 대해 알아보기
- [배포 문서](DEPLOYMENT.md) - 프로덕션 환경 배포 권장 사항에 대해 알아보기
- [API 문서](API.md) - API 보안 기능에 대해 알아보기

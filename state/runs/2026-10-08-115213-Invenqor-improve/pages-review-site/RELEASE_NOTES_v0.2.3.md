# Invenqor Server v0.2.3 릴리즈 노트

릴리즈 일자: 2026-07-29
호환 Agent: v0.2.1

## 핵심 변경

### 운영 설정 화면

- PostgreSQL 현재 모드·대상·schema·설정 원천과 안전한 장애 요약을 표시합니다.
- 새 PostgreSQL DSN을 schema 변경 없이 연결 테스트하고, 성공한 값만
  AES-256-GCM 암호화 저장합니다.
- `INVENQOR_POSTGRES_DSN`, `POSTGRES_DSN`, `postgres_dsn` 환경변수를 우선순위에
  따라 지원합니다.
- 환경변수 우선 적용과 재기동 필요 상태를 화면에 명확하게 표시합니다.

### Keycloak SSO/OIDC

- Issuer/Realm, Client ID/Secret, Redirect/Logout URI, Scope, Claim, Email domain,
  기본 역할과 자동 사용자 생성을 관리하는 화면을 제공합니다.
- Role/Group mapping, `realm_access.roles` 같은 중첩 Claim과 사설 CA PEM을
  지원합니다.
- Issuer discovery와 TLS 신뢰를 저장 전에 테스트합니다.
- Client Secret 없이 활성화하거나 존재하지 않는 Invenqor 역할을 mapping하는
  설정을 거부합니다.
- SSO 로그인마다 프로필과 Keycloak 원천 역할을 동기화하고 회수된 역할을
  제거합니다.
- 비활성화·삭제된 SSO 사용자는 같은 subject로 자동 재생성하지 않습니다.
- 로컬 Session 폐기 후 Keycloak end-session logout을 지원합니다.

### 사용자와 권한 관리

- 로컬 사용자 생성, 검색, 프로필, 역할, 활성 상태, 잠금 해제, 비밀번호 초기화와
  삭제 기능을 제공합니다.
- 로컬 역할과 Keycloak 역할 원천을 분리하고 SSO 역할은 화면에서 명확하게
  식별합니다.
- 비활성화·삭제 시 Session과 사용자가 발급한 API key를 즉시 폐기합니다.
- 자기 비활성화·강등·삭제와 마지막 활성 Super Admin 제거를 차단합니다.
- 관리 작업을 감사 로그에 기록합니다.

### UI와 운영성

- 로그인 화면과 콘솔 상단에 실행 중인 Server 버전을 표시합니다.
- Keycloak이 활성화된 경우에만 SSO 로그인 버튼을 노출합니다.
- 사용자 권한에 따라 접근 가능한 메뉴만 표시합니다.
- 설정·사용자 화면을 실제 관리 API와 연결하고 실패 사유를 표시합니다.

## 문서와 API

- 사용자·관리자·임원 보고서와 Server 설치 가이드를 Markdown/PDF로 갱신했습니다.
- PostgreSQL, Keycloak, 사용자 관리 API를 `openapi.yaml`에 추가했습니다.
- Keycloak Client 생성, 중첩 Claim, Role/Group mapping, 사설 CA와 비상 접근
  운영 절차를 문서화했습니다.

## 검증

- Go 전체 테스트 및 `go vet`
- React/Vitest 6개 테스트와 production build
- npm audit 취약점 0건
- Redocly OpenAPI lint
- 온라인·오프라인 Docker Compose 렌더링
- 실제 PostgreSQL 17 + Server Docker E2E
- 소문자 `postgres_dsn`, 환경변수 초기 관리자, 사용자 RBAC, Keycloak 설정과
  내장 웹 콘솔 E2E

## 릴리즈 자산

- 오프라인 Docker 번들: `invenqor-0.2.3.tar.gz`
- 번들 체크섬: `invenqor-0.2.3.tar.gz.sha256`
- Helm chart: `invenqor-0.2.3.tgz`
- Helm chart 체크섬: `invenqor-0.2.3.tgz.sha256`
- `compose.offline.yaml`, `openapi.yaml`
- 사용자·관리자·임원·Server 설치·API/MCP 가이드 MD/PDF

오프라인 번들에는 `linux/amd64`용 `invenqor-server:0.2.3`과
`postgres:17-alpine` 이미지가 포함됩니다. Agent v0.2.1 배포 파일은 별도
[`invenqor-agents`](https://github.com/hkjang/invenqor-agents/releases/tag/v0.2.1)
릴리즈에서 제공합니다.

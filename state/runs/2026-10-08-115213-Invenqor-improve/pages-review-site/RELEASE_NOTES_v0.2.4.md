# Invenqor Server v0.2.4 릴리즈 노트

릴리즈 일자: 2026-07-29
호환 Agent: v0.2.2

## 핵심 변경

### Zero-touch Agent 등록

- Agent는 `config.toml`의 Server URL만으로 최초 통신 시 자동 등록됩니다.
- 장비별 UUID와 device claim을 로컬에 안전하게 보존하고, Server가 발급한
  장비 Token이 무효화되거나 유실되면 같은 장비로 자동 복구합니다.
- localhost, 사설 IP, 단일-label 내부 DNS와 `.internal`/`.local` HTTP를
  내부망 설치 용도로 지원하며 공인 주소 HTTP는 명시적 허용이 필요합니다.
- CentOS 7, Red Hat UBI 8/9, Ubuntu 22.04/24.04 LTS에서 URL-only 실제
  설치·수집·전송을 검증했습니다.

### 런타임 등록 정책과 Token 관리

- **설정 → Agent 등록**에서 신규 등록 정책을 재기동 없이 전환합니다.
  - `open`: 등록 Token 없이 Server URL만으로 자동 등록
  - `token`: 공용 등록 Token이 있어야 최초 등록
  - `disabled`: 신규 등록만 차단
- 등록 Token 발급·즉시 회전·폐기를 지원합니다.
- Token 원문은 발급 응답에서 한 번만 표시하고 DB에는 SHA-256 비교값만
  저장합니다.
- 정책과 버전을 공용 PostgreSQL에 저장하고 등록 요청마다 현재 값을 검증하므로
  Kubernetes 모든 Server Pod에 즉시 동일하게 적용됩니다.
- 정책 및 Token 변경은 전후 상태와 변경 사유를 감사 로그에 기록합니다.
- `AGENT_AUTO_ENROLLMENT`와 등록 Token 환경변수는 DB 정책이 없는 최초 기동의
  기본값으로 사용됩니다.

### 관리 콘솔 완성도

- 자산 CRUD, 삭제·복원, 원천·변경 이력, 관계, 병합·분리를 실제 API에
  연결했습니다.
- Agent 상태, 수동 등록, 장비 Token 회전, 차단·해제, mTLS 인증서와 서명
  업데이트 게시 기능을 연결했습니다.
- Query DSL 검증·실행, 일반 설정 버전·롤백, 감사 로그, API·MCP Key 전체
  수명주기를 연결했습니다.
- 운영 통계 화면에 자산 최신성, Agent 건전성, 24시간 수집 성공·실패,
  7일 추이와 유형·환경·중요도·원천 분포를 제공합니다.

### 인증·사용자·개인화

- 레거시 nullable Keycloak 설정을 안전하게 정규화해 OIDC 설정 화면의 빈 화면
  오류를 수정했습니다.
- 페이지별 Error Boundary를 추가해 렌더 오류를 복구 가능한 화면으로
  표시합니다.
- 사용자 생성 화면의 역할 선택을 설명·권한 수·선택 상태가 명확한 카드로
  개선했습니다.
- 우측 상단 프로필 메뉴에서 내 보안, 개인화와 로그아웃에 접근합니다.
- 사용자별 테마, 화면 밀도, 시작 화면, 통계 갱신 주기와 모션 축소 설정을
  브라우저에 분리 저장합니다.
- 로컬 로그인과 Keycloak callback의 CSRF 처리를 통일했습니다.

## API와 문서

- Agent 등록 정책 조회·변경과 Token 발급·회전·폐기 API를
  `openapi.yaml`에 추가했습니다.
- 사용자·관리자·임원·Server 설치·API/MCP 가이드를 Markdown과 PDF로
  갱신했습니다.
- URL-only 설치, Open/Token/Disabled 운영 기준과 멀티 Pod 적용 방식을
  문서화했습니다.

## 검증

- Go 전체 테스트 및 `go vet`
- Rust 22개 단위 테스트와 Clippy `-D warnings`
- React/Vitest 11개 테스트와 production build
- npm audit 취약점 0건
- Redocly OpenAPI lint
- 온라인·오프라인 Docker Compose와 Helm lint/template
- 실제 브라우저에서 설정 화면 렌더링·정책 전환·Token 1회 표시
- PostgreSQL 17 기반 두 Server Pod의 정책 즉시 공유 E2E
- 실제 Agent 자동 등록, 장비 자격 증명 복구, 자산 수집·저장과 서명 업데이트 E2E

## 릴리즈 자산

- 오프라인 Docker 번들: `invenqor-0.2.4.tar.gz`
- 번들 체크섬: `invenqor-0.2.4.tar.gz.sha256`
- Helm chart: `invenqor-0.2.4.tgz`
- Helm chart 체크섬: `invenqor-0.2.4.tgz.sha256`
- `compose.offline.yaml`, `openapi.yaml`
- 사용자·관리자·임원·Server 설치·API/MCP 가이드 MD/PDF

오프라인 번들에는 `linux/amd64`용 `invenqor-server:0.2.4`와
`postgres:17-alpine` 이미지가 포함됩니다. Agent v0.2.2 배포 파일은 별도
[`invenqor-agents`](https://github.com/hkjang/invenqor-agents/releases/tag/v0.2.2)
릴리즈에서 제공합니다.

# Invenqor Server v0.2.5 릴리즈 노트

릴리즈 일자: 2026-07-29
호환 Agent: v0.2.3

## 핵심 변경

### 네트워크 경계 기반 Zero-touch 자산 등록

- Agent는 `config.toml`에 Server URL만 설정하면 최초 호출에서 자동 등록됩니다.
- 등록 성공과 동시에 IP 기반 discovered 자산이 생성되므로 첫 inventory 전에도
  관리 콘솔에서 신규 장비를 확인할 수 있습니다.
- 첫 system inventory는 등록 placeholder를 승격하므로 중복 자산을 만들지
  않습니다.
- 관리자는 모든 IP 허용 또는 정확한 IP/CIDR allowlist를 선택할 수 있습니다.
- `X-Forwarded-For`는 직접 접속한 Ingress/프록시가 trusted proxy 대역에
  포함될 때만 사용해 전달 헤더 위조를 차단합니다.

### Keycloak 최소 정보 빠른 연동

- Keycloak URL, Realm, Client ID와 Client Secret만 입력하면 issuer,
  discovery, callback, logout, scopes와 기본 claim mapping을 자동 구성합니다.
- 실제 OIDC discovery와 TLS 검증이 성공한 경우에만 암호화 저장하고
  활성화합니다.
- 기존 고급 설정은 유지해 조직별 role/group claim을 세부 조정할 수 있습니다.

### 멀티 Pod Server 진단 로그

- 모든 Server Pod가 공용 PostgreSQL에 구조화된 운영 진단 이벤트를 기록합니다.
- 관리자 콘솔의 **Server 로그**에서 수준, 구성요소, Pod, request ID, Agent ID,
  IP와 오류 코드로 검색할 수 있습니다.
- Agent 오류에도 Server의 코드, 안전한 메시지, API path와 request ID를
  표시해 양쪽 로그를 직접 연결합니다.
- Token, 비밀번호, Authorization과 URL 자격증명은 저장 전에 마스킹합니다.
- 기본 보존 정책은 30일 또는 최근 10,000건이며 15초 자동 갱신을 제공합니다.

### 콘솔과 HTTPS Ingress

- 주 메뉴와 설정 하위 메뉴를 URL 및 사용자별 브라우저 상태에 동기화해
  새로고침과 뒤로/앞으로 이동 후에도 현재 화면을 유지합니다.
- 선택적 Helm Ingress를 제공해 외부 HTTPS 443을 내부 단일 Service 7070으로
  연결합니다.
- Agent 등록·이벤트·업데이트 경로는 같은 호스트의 HTTPS를 사용하며 NGINX
  body size와 timeout 기본 권고값을 포함합니다.

## API와 데이터베이스

- Agent 등록 정책에 `network_mode`, `allowed_networks`, `trusted_proxies`를
  추가했습니다.
- `POST /api/v1/admin/settings/keycloak/auto-configure`를 추가했습니다.
- `GET /api/v1/admin/diagnostics/logs`를 추가했습니다.
- PostgreSQL과 SQLite에 `diagnostic_logs` migration을 추가했습니다.
- OpenAPI와 사용자·관리자·임원·Server 설치·API/MCP 가이드를 갱신했습니다.

## 검증

- Go 전체 테스트
- Rust format 및 단위 테스트 23개
- React/Vitest 테스트 14개와 production build
- Redocly OpenAPI lint
- Helm Ingress template 렌더링
- 실제 PostgreSQL 기반 Server 2개 인스턴스의 정책·자산·진단 로그 공유 E2E
- CentOS 7, Red Hat UBI 8/9, Ubuntu 22.04/24.04 LTS, Alpine Agent E2E
- enrollment-only 즉시 자산 생성과 첫 inventory 중복 방지 검증

## 릴리즈 자산

- 오프라인 Docker 번들: `invenqor-0.2.5.tar.gz`
- 번들 체크섬: `invenqor-0.2.5.tar.gz.sha256`
- Helm chart: `invenqor-0.2.5.tgz`
- Helm chart 체크섬: `invenqor-0.2.5.tgz.sha256`
- `compose.offline.yaml`, `openapi.yaml`
- 사용자·관리자·임원·Server 설치·API/MCP 가이드 MD/PDF

오프라인 번들에는 `linux/amd64`용 `invenqor-server:0.2.5`와
`postgres:17-alpine` 이미지가 포함됩니다. Agent v0.2.3 배포 파일은 별도
[`invenqor-agents`](https://github.com/hkjang/invenqor-agents/releases/tag/v0.2.3)
릴리스에서 제공합니다.

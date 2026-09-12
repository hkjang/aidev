## 2026-09-13
- 선택: 캠페인 silent-sso-2026-09 — Keycloak 에 이미 로그인한 사용자가 로그인 화면 없이 콘솔로 들어오는 silent SSO(`prompt=none`), `auto_login` 설정 기본 꺼짐 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 이 저장소에는 Keycloak OIDC(`server/internal/auth/oidc.go`, `httpapi/keycloak.go`)가
  이미 있어 SILENT-SSO-STANDARD.md 를 그대로 따라 붙였다. Server: `OIDCSettings.AutoLogin`
  (`auto_login`, 기본 false), migration 008 로 `oidc_flows.silent` 열 추가, `Start()` 가
  `silent` 를 받되 `auto_login` 이 꺼져 있으면 조용히 평범한 로그인으로 낮추고 켜져 있을 때만
  `prompt=none` 을 보낸다. `RefusedSilently()` 는 콜백의 `error=` 가 이 Server 가 `prompt=none`
  으로 시작한(아직 안 쓰인·안 만료된) 흐름의 `login_required`·`interaction_required`·
  `consent_required` 일 때만 참을 돌려주고 그 state 를 소비한다 — 그때 `/?sso=none` 으로 보내며
  진단 로그에 남기지 않는다(이 SPA 는 `/login` 경로가 없고 로그인 화면이 `/` 이므로 표준의
  `/login?sso=none` 대신 기존 `/?auth_error=` 관례에 맞춰 `/?sso=none` 을 썼다). 평범한 흐름의
  `login_required` 나 silent 흐름의 `access_denied` 는 전처럼 `KEYCLOAK_PROVIDER_REJECTED`.
  `/api/v1/auth/methods` 에 `keycloak_auto_login`(ready && auto_login) 추가. 콘솔:
  `web/src/silentSso.ts` 에 세 겹 장치(sessionStorage 한 탭 한 번 · 로그아웃 억제 표시 ·
  주소의 `sso`/`auth_error` 표시)와 저장소 예외를 '이미 시도했다'로 읽는 fail-closed,
  `/api`·`/v1`·`/health`·`/mcp`·`/momento` 경로 제외, `return_to` 는 `/` 로 시작하고 `//`·`\`
  없는 값만. `main.tsx` 는 `/auth/me`·`/auth/methods`·bootstrap 상태가 모두 온 뒤에만 판단해
  최상위 `window.location.assign` 으로 이동하고, 그동안과 이동 중에는 로그인 폼을 그리지
  않는다(깜빡임 제거). 로그아웃은 `sessionStorage.clear()` 뒤에 억제 표시를 남기고 세션이 다시
  생기면 지운다. 설정 → Keycloak 에 토글 추가, `webui/dist` 재빌드, openapi.yaml(`auto_login`,
  `keycloak_auto_login`, `prompt` 파라미터, 콜백 설명), ADMIN_GUIDE.md 16.2 절(설정·동작·세 겹
  장치·Server 보호·확인 방법)과 ADMIN_GUIDE.pdf 재생성(`file://` 0건). 검증: Go 단위 4개(기본
  꺼짐이면 `prompt=none` 무시, 켜면 전송·버튼은 평범한 로그인, 거절 인식은 silent 흐름에 한하고
  재사용 불가, 깊은 링크 `/#/assets?asset=42` 로 복귀·`//evil` 등은 `/`), httpapi 통합 1개
  (설정 켜고 끄기에 따른 downgrade·`/?sso=none` 착지·세션 쿠키 없음·state 재사용 거부·진단
  로그 무기록·`access_denied` 는 여전히 실패), vitest 8개(한 탭 한 번, `sso=none`/`auth_error`
  주소, 로그아웃 억제·해제, 저장소 예외 fail-closed, 비페이지 경로, return_to 안전성)를 더해
  `go test ./...` 를 SQLite 와 실 PostgreSQL(`scripts/test-postgres.sh`) 양쪽 전 패키지 통과,
  `go vet`·`gofmt`·`go build`, vitest 142개 통과, redocly lint 경고 6개(모두 기존). 표준의
  검증 항목은 임시 Server(SQLite)와 가짜 TLS IdP(discovery·auth·token·jwks, 세션 쿠키로
  로그인 상태 모사)를 띄우고 headless Chrome 으로 실제 확인했다 — (1) `auto_login` 꺼짐:
  로그인 화면, IdP 에 auth 요청 0건 (2) 켜고 IdP 미로그인: `/auth?prompt=none` 정확히 1건 뒤
  로그인 화면, `/?sso=none#/assets` 를 3회 다시 열어도 auth 요청이 늘지 않음(루프 없음),
  Server 로그에 실패 기록 없음 (3) IdP 로그인 상태로 `/#/audit` 열기: 로그인 폼 없이 콘솔
  (`CONTROL PLANE`, `SSO User`)이 뜨고 auth·token·jwks 각 1건. 로그아웃 억제는 headless
  dump-dom 으로 클릭을 재현할 수 없어 vitest 단위 테스트로만 확인했다. 버전 범프·릴리즈
  노트는 하지 않았다.
- 보류 아이디어: 설정 → Keycloak 화면 캡처에 auto_login 토글이 없음 — 캡처 스크립트·PNG·가이드 세 곳 대조 테스트가 있어 방문 추적 캡처와 같은 회차에 묶어 재촬영 (가치 2 / 위험 1 / S) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · MCP `asset_relations` 가 상대 자산의 이름·종류를 주지 않아 edge 마다 `asset_get` 을 한 번 더 부르게 만듦 (가치 3 / 위험 2 / M) · 콘솔 Query DSL 화면이 새 `total`·`offset` 을 쓰지 않아 여전히 첫 페이지만 보여줌 (가치 3 / 위험 2 / M) · 콜백의 `RefusedSilently` 가 평범한 로그인의 provider 오류에서도 흐름을 소비하므로 `oidc_flows` 정리 SQL 의 `consumed_at` 조건과 함께 e2e-multipod 스크립트에 silent 흐름 케이스를 더함 (가치 2 / 위험 1 / S)

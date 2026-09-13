## 2026-09-14
- 선택: Silent SSO(OIDC prompt=none) + auto_login 설정 [캠페인 silent-sso-2026-09] (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: madi에는 Keycloak OIDC(`integrations_oidc.go`)가 이미 있어 SILENT-SSO-STANDARD.md 대로 구현했다. 서버는 `oidc_auto_login`(기본 꺼짐)이 켜진 경우에만 `?prompt=none`을 제공자에 전달하고, `oidc_attempts`에 silent/return_to를 기록해 거절(`login_required`)이면 `/login?sso=none`으로, 성공이면 같은 출처 경로(`/`로 시작·`//` 제외)로 돌려보낸다. 브라우저는 새 모듈 `web/src/auth/silentSso.ts`가 sessionStorage 1회 표시(저장소 예외 시 '이미 시도'로 간주), 로그아웃 억제, URL 표시, 콜백·로그인·API·MCP·헬스 경로 제외 규칙으로 최상위 이동 한 번만 시도한다. 관리자 설정 UI 토글과 docs/admin-guide.md(manuals HTML 재생성)에 설정·동작을 적었다. 검증: `npm run build`(tsc), `go build/vet/gofmt`, 새 `tests/silent-sso.mjs`(CI 목록에 등록), 임시 PostgreSQL 17 컨테이너로 `TestPostgresOIDCSilentLogin`·기존 OIDC/설정/백업 통합 테스트, 전체 비DB `go test ./...` 통과. 커밋 323950c.
- 보류 아이디어: 로그인 화면에 `?sso=none` 도착 시 "회사 계정 세션이 없어 로그인 화면을 표시" 안내 문구(가치 2/위험 1/S); Playwright 브라우저 시험에 silent SSO 루프 방지 시나리오 추가(가치 3/위험 2/M); SAML `/auth/saml/start`에도 같은 return_to 보존 적용(가치 2/위험 2/S); 비조용 OIDC 제공자 오류(401 JSON)를 `/login?sso=error`로 안내해 사용자 경험 통일(가치 3/위험 2/S); OpenAPI 목록에 `prompt`/`return_to` 쿼리 파라미터 설명 추가(가치 1/위험 1/S).


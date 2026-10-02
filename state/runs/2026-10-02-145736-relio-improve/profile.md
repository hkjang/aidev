# relio 프로필 (2026-10-02)
- 목적: 에어갭 사내 B2B CRM 을 Go 단일 바이너리의 REST·MCP·React 웹으로 제공한다.
- 스택: Go 1.24(net/http, pgx v5.7.5), PostgreSQL, React 19·TypeScript 5.9·Vite 7, Keycloak OIDC.
- 구조:
  - cmd/relio: 진입점. internal/server: REST·관리자·메일·VOC 핸들러.
  - internal/auth·oidc·apikey: 인증·SSO·개인 키. internal/audit: 감사 기록.
  - internal/crm·intelligence·relationship·voice·approval: 업무 서비스·분석·승인.
  - internal/mail·job·personal·admin·analytics: 메일·잡·개인 기능·관리·분석.
  - internal/mcp: 도구 스키마·dispatch·오류 정제. internal/api: OpenAPI.
  - internal/platform: httpx·database·secrets·timezone. internal/webui: SPA embed.
  - migrations: SQL embed, 마지막 017_mail_notifications.sql, 다음 번호는 018.
  - web/src/pages: React 화면. web/test: Node 내장 테스트 4파일(27건).
  - docs: 가이드·설계·릴리즈·로드맵. scripts: 계약·오프라인·업그레이드 검증.
- 빌드·테스트: go build ./... ; go test ./... ; go test -race ./... ; go vet ./... ; gofmt -l.
  - 프런트: npm --prefix web ci ; run typecheck ; test ; run build.
  - **make test 는 이제 Go test/vet + npm ci/typecheck + npm test 를 모두 돌린다(ed0352e).**
    이전 프로필의 "npm test 누락" 은 해결됨. 단 npm ci 가 네트워크를 타므로 에어갭·느린 환경에서 오래 걸리거나 ETIMEDOUT 이 난다.
  - ./scripts/check-env-contract.sh ; ./scripts/check-static-assets.sh ; ./scripts/previous-release-tag-test.sh .
  - Docker offline/upgrade 검증은 이미지·PostgreSQL 기동이 필요한 장시간 작업.
  - README 요구 도구: Go 1.24+, Node.js 24+, Docker.
- 관례: `type(scope): 요약` 커밋, 한국어 요약이 기본. 환경 변수 필수 3+선택 1, 런타임 설정은 system_settings.
  - 새 스키마는 새 번호 SQL 마이그레이션. 기능 설명은 docs/ADMIN_GUIDE.md 등과 함께 유지.
  - 불변식 테스트 관용이 정착해 있다 — go/ast 로 internal/ 을 순회하고, 실패 메시지에 path:line 을 찍고,
    "스캔이 표에 닿고 있음" 을 개수 가드(`seen < known` → t.Fatalf)로 증명하고, 상단 주석에 왜 이 불변식이
    있는지 적는다. 본보기: platform/database/rows_err_test.go, server/sqlstate_parity_test.go, crm/search_test.go.
- 위험 구역: internal/auth·oidc, internal/server/public.go, migrations/, .github/workflows/release.yml.
  - 감사 기록에 원문 비밀 저장 금지. AdminPages.tsx·mcp/server.go 대규모 포매팅 피할 것(한 줄이 매우 길다).
  - 감사 질의 admin_operations.go:99 의 ILIKE 5개는 인덱스 계획이 걸려 있어 lower(...) LIKE 로 통일하지 말 것.
- 자주 깨지는 곳: web build 의 embed 앵커 internal/webui/dist/README 와 web/public/README 유지.
  - 릴리즈 업그레이드 검증 실패 이력(2026-09-06, 되돌림 PR + 자율화 강등) — 릴리즈·빌드 경로 수정은 로컬 재현 필수.
- 검증 함정:
  - ci.yml 과 Makefile:test 는 npm test 를 돌리지만 release.yml 의 Test source 단계에는 아직 없다.
  - web/test/login.test.ts 는 esbuild 로 실제 Login.tsx 를 번들링해 React 서버 렌더링한다. 브라우저 E2E 아님.
  - 일반 Go 테스트는 SQL 동작을 증명하지 않는다 — LIKE·ESCAPE·대소문자 접기 불변식은 **소스** 불변식이다.
  - secrets DB 통합 테스트는 DSN 없으면 skip, 폐기용 DB 만 사용.
  - REST/MCP 는 SQLSTATE 표(7그룹/10코드)·ConnectError·context 취소·만료를 모두 일반 오류로 접고,
    두 표의 일치를 go/ast 대조 테스트가 지킨다. 연결 **성립 후** 단절(net.OpError/EOF)은 아직 미해결이며
    실제 동적 타입은 미확인 — 착수 전 폐기용 PostgreSQL probe 필요.
  - 자율 회차(10분·6파일)는 실 DB 기동이 필요한 과제를 끝낼 수 없다 — 그런 후보는 보류로 두는 것이 맞다.
- 이번 확인(2026-10-02, main@494d00f, 워크트리 수정 없음):
  - `go test ./internal/crm/ -run 'TestSearchPattern|TestEveryFreeTextSearch' -v` 4건 PASS (0.032s).
  - internal/ 비-테스트 .go 의 LIKE 비교는 8줄에 19개(크m/service.go:209 3개, admin_operations.go:99 5개 등).
    cmd/ 와 migrations/ 에는 LIKE 비교가 없다.
  - 전체 `go test ./...`·npm 계열·make test 는 이번 회차 미실행(미확인). npm ci 는 이전 회차에 ETIMEDOUT 이력.
- 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration 을
  Skill 도구로 불러 반영했다(이번 회차에는 Skill 도구가 실제로 노출됨).

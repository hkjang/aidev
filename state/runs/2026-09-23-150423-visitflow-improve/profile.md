# VisitFlow 프로필 (2026-09-23)

- 목적: 사내 방문 예약·승인·QR 방문증·로비 체크인·알림을 단일 컨테이너로 제공하는 방문 관리 시스템.
- 스택: Go 1.24(chi·pgx·go-oidc·excelize), PostgreSQL 14+, React 19·TypeScript·Vite·Material UI 7, vitest·Playwright.
- 기준: HEAD 4e0481d(v2.8.5). 이번 정찰에서 `git log -15`·미머지 브랜치 목록·`VisitFormPage.tsx` 전체·`visits.go` 25-33/430-500·`import.go` 235-275·`templates.go` grep·`schedule.ts`/`schedule.test.ts`·`vite.config.ts`·`package.json` 을 직접 열었다.

## 구조
- `cmd/visitflow/`: 서버 진입점. 임베드 `webdist`(기본 스텁 vs 실제 프런트 빌드 구분), `version` 기본값 "dev".
- `internal/app/`: HTTP·MCP·SSE·백그라운드. `visits.go:436 createVisitRecord` 가 신청 정책·일정·방문자 검증의 **단일 지점**이며 REST(`visits.go:390`)·현장(`visits.go:1289`)·MCP(`mcp.go:166`) 셋이 공유한다.
- `internal/app/import.go`: CSV/XLSX 파싱(첫 시트·앞 10행 헤더 탐색). 101번째 행에서 명시적 오류로 중단하므로 조용한 잘림은 없다.
- `internal/app/integration_test.go`: `newTestEnv` 가 테스트별 DB 생성·정리. `do`/`doForwarded`/업로드는 30초 요청 데드라인 사용.
- `internal/platform/`(config·mail·crypto), `internal/database/`(번호별 트랜잭션 마이그레이션, 최신 `0014_tracking.sql`).
- `web/src/`: `pages/VisitFormPage.tsx`(119줄, 아주 긴 한 줄 스타일)가 `/visits/new` 와 `/lobby/walk-in` 을 공유. 순수 검증 로직은 `src/schedule.ts` 처럼 페이지 밖 별도 모듈로 빼는 관례가 자리잡았다.
- `web/e2e/visit-flow.spec.ts`(실제 서버 Playwright), `web/screenshots/`, `docs/`(USER/ADMIN GUIDE·API_AND_MCP·ARCHITECTURE·PDF).

## 빌드·테스트
- `go test ./... -count=1` — `VISITFLOW_TEST_DSN` 없으면 PostgreSQL 통합이 SKIP된다(**PASS 가 통합 증거가 아님**). DSN 지정 시 internal/app 과거 약 45~57초.
- CI DSN: `postgres://visitflow:visitflow@127.0.0.1:5432/visitflow?sslmode=disable`. CREATE DATABASE 권한 필요.
- `cd web && npm ci && npm run lint && npm test && npm run build`. `lint`=`tsc -b`, `test`=`vitest run`. **워크트리에 `web/node_modules` 가 없으므로 매 회차 `npm ci` 부터 필요**(이번 정찰 미실행).
- vitest `include` 는 `src/**/*.test.ts` — **`.tsx` 테스트는 수집되지 않는다**. 현재 단위 테스트는 `silentSso.test.ts`(8) + `schedule.test.ts`(15) = 23개.
- `cd web && npm run test:e2e`(Playwright). 설정이 서버를 자동 기동하지 않는다 — `web/dist` 를 `cmd/visitflow/webdist` 에 복사하고 Go 서버·DB·Chromium 을 미리 띄워야 한다(준비 비용 큼).
- 값싼 대안: `npm run build` 산출물을 정적 서빙하고 `auth/config`·`auth/me`·`reference-data`·`visits` 만 스텁으로 답하는 `/tmp` 서버 + Chromium. 최근 두 회차가 실제로 이 방식으로 화면을 확인했다.
- `docker build -t visitflow:dev .`. CI test 잡: Go 테스트/vet → npm ci/test/build → Docker build.

## 관례
- 커밋은 영어 conventional(`fix(web): …`), 릴리즈는 `chore(release): VisitFlow vX.Y.Z`. UI·가이드·오류 메시지는 한국어.
- `writeError(w, status, snake_case_code, 한국어 메시지)`. 정책은 `settings` 표에 저장하고 `PUT /api/v1/settings` 로 변경. `getSetting` 은 5초 캐시이고 `updateSettings` 가 즉시 invalidate 하므로 테스트에서 SQL 만 바꾸면 옛 캐시를 본다 — 설정 API 를 쓸 것.
- 마이그레이션은 번호별 SQL 트랜잭션. 로더가 빈 번호를 허용해 미머지 브랜치와 번호를 건너뛸 수 있다.
- 화면 사전 검사 관례(2회 연속 채택): 순수 함수 모듈 → 필드 `error`/`helperText` · 제출 버튼 `disabled` · `submit()` 가드 **세 곳이 같은 값 하나를 읽게** 배선.

## 위험 구역
- `auth.go`(세션/OIDC)·`keys.go`/암호화·`internal/database/migrations/`·`settings.go`·`server.go`·감사/개인정보.
- `normalizePhone`(visits.go:25)은 `watchlist_entries.phone_hash` 의 입력이다 — 편의로 바꾸지 말 것.
- `createVisitRecord` 의 상태코드·에러 코드·문구는 REST·현장·MCP 공용 계약이다. 화면 편의로 바꾸지 말 것.
- HEAD 미머지 브랜치 3개(파일 겹치면 피할 것): `origin/auto/2026-09-16-1212`(메일: platform/mail·settings·AdminPage·integration_test), `origin/auto/2026-09-18-0533`(MCP OAuth: SettingsPage·mcpoauth), `origin/auto/2026-09-21-0654`(가져오기 중복 헤더: import.go·import_test.go·USER_GUIDE 가져오기 절·API_AND_MCP).

## 자주 깨지는 곳
- 변이 되돌리기에 `git checkout -- <파일>` 을 써서 미커밋 테스트를 두 번 잃었다 — `sed` 나 임시 복사본을 쓸 것.
- 화면과 서버가 같은 규칙을 따로 읽는 자리(회사명 필수·일정·방문자 100명 상한·전화 자릿수)에서 한쪽만 고쳐 불일치가 생긴다. 앞의 둘은 v2.8.4/v2.8.5 로 닫혔고 **뒤의 둘은 아직 열려 있다**.
- SSE/로비 스트림은 요청 컨텍스트 종료가 없으면 대기한다(헬퍼 데드라인으로 완화됨).

## 검증 함정
- Go PASS 는 DB 통합 실행 증거가 아니다. E2E 는 실제 빌드 UI 를 서빙하는 서버가 있어야 의미가 있다.
- 새 단위 테스트는 `.ts` 로 둘 것(`.tsx` 는 vitest 가 수집하지 않아 조용히 0개가 된다).
- `new Date("2026-02-31T10:00")` 는 V8 이 3월 3일로 롤오버한다 — 파싱 실패 전제를 세울 때 주의(직전 회차가 실제로 걸렸다).
- excelize `GetRows` 는 표시 서식을 적용한다(숫자 전화·지수 표기). 중복 헤더 경고는 HEAD 에 없고 미머지 브랜치에만 있다.
- 실제 Excel·Keycloak·SMTP 검증 인프라는 이 환경에 없다. PDF 도구는 저장소 밖 `aidev/tools/guide/md2pdf.mjs`.

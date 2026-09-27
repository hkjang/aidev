# VisitFlow 프로필 (2026-09-28)

- 목적: 사내 방문 예약·승인·QR 방문증·로비 체크인·알림을 단일 컨테이너로 제공하는 방문 관리 시스템.
- 스택: Go 1.24(chi·pgx·go-oidc·excelize), PostgreSQL 14+, React 19·TypeScript·Vite·Material UI 7, vitest·Playwright.
- 기준: HEAD 15456db(v2.8.7). 이번 정찰에서 직접 연 것: `git log -15`·미머지 브랜치 목록, `web/src/pages/VisitFormPage.tsx` 전체, `web/src/visitors.ts` 전체, `internal/app/visits.go:430-560`(createVisitRecord), `internal/app/import.go` grep(237-270), `web/src/api.ts:1-40`, reference-data 소비자 grep.

## 구조
- `cmd/visitflow/`: 서버 진입점. 임베드 `webdist`(기본 스텁 vs 실제 프런트 빌드 구분), `version` 기본값 "dev".
- `internal/app/`: HTTP·MCP·SSE·백그라운드. `visits.go:436 createVisitRecord` 가 신청 정책·일정·방문자 검증의 **단일 지점**이며 REST·현장(walk-in)·MCP 셋이 공유한다. 경계: 방문자 1~100명(`required_fields`), 종료>시작(`invalid_schedule`), 31일 초과(`schedule_too_long`), 이름 TrimSpace·`normalizePhone` 7자리·`Consent`(`invalid_visitor`), 회사명 정책(`company_required`), 반복 2~52회 & `occurrences*len(Visitors)<=500`(`invalid_recurrence`).
- `internal/app/import.go`: CSV/XLSX 파싱(첫 시트·앞 10행 헤더 탐색). 101번째 행에서 명시적 오류로 중단하고, 파싱된 방문자가 0명이면 미리보기를 오류로 돌려보낸다(`import.go:267`).
- `internal/app/integration_test.go`: `newTestEnv` 가 테스트별 DB 생성·정리. `do`/`doForwarded`/업로드는 30초 요청 데드라인 사용.
- `internal/platform/`(config·mail·crypto), `internal/database/`(번호별 트랜잭션 마이그레이션, 최신 `0014_tracking.sql`).
- `web/src/`: `pages/VisitFormPage.tsx`(129줄, 아주 긴 한 줄 스타일)가 `/visits/new` 와 `/lobby/walk-in` 을 공유. 순수 검증 로직은 `src/schedule.ts`·`src/visitors.ts` 처럼 페이지 밖 `.ts` 모듈로 빼는 관례가 확립됐다.
- `web/e2e/visit-flow.spec.ts`(실제 서버 Playwright), `web/screenshots/`, `docs/`(USER/ADMIN GUIDE·API_AND_MCP·ARCHITECTURE·PDF).

## 빌드·테스트
- `go test ./... -count=1` — `VISITFLOW_TEST_DSN` 없으면 PostgreSQL 통합이 SKIP된다(**PASS 가 통합 증거가 아님**). DSN 지정 시 internal/app 과거 약 45~57초.
- CI DSN: `postgres://visitflow:visitflow@127.0.0.1:5432/visitflow?sslmode=disable`. CREATE DATABASE 권한 필요.
- `cd web && npm ci && npm run lint && npm test && npm run build`. `lint`=`tsc -b`, `test`=`vitest run`. **워크트리에 `web/node_modules` 가 없으므로 매 회차 `npm ci` 부터 필요**(이번 정찰 미실행).
- vitest `include` 는 `src/**/*.test.ts` — **`.tsx` 테스트는 수집되지 않는다**. 현재 단위 테스트는 `silentSso.test.ts` + `schedule.test.ts` + `visitors.test.ts` = 47개(v2.8.7 기준, 이번 정찰에서 재실행하지는 않음).
- `cd web && npm run test:e2e`(Playwright). 설정이 서버를 자동 기동하지 않는다 — 준비 비용 큼.
- 최근 4회차가 실제로 쓴 화면 검증: `web/dist` → `cmd/visitflow/webdist` 복사 → `go build` → postgres 컨테이너 + nohup 서버 → `playwright-core` 의 chromium(`executablePath=/usr/bin/google-chrome`). 스텁 API 를 쓰지 않아 계약이 어긋날 위험이 없다.
- `docker build -t visitflow:dev .`. CI test 잡: Go 테스트/vet → npm ci/test/build → Docker build.

## 관례
- 커밋은 영어 conventional(`fix(web): …`), 릴리즈는 `chore(release): VisitFlow vX.Y.Z`. UI·가이드·오류 메시지는 한국어. 소스 주석도 한국어(왜를 설명).
- `writeError(w, status, snake_case_code, 한국어 메시지)`. 정책은 `settings` 표에 저장하고 `PUT /api/v1/settings` 로 변경. `getSetting` 은 5초 캐시이고 `updateSettings` 가 즉시 invalidate 하므로 테스트에서 SQL 만 바꾸면 옛 캐시를 본다 — 설정 API 를 쓸 것.
- 마이그레이션은 번호별 SQL 트랜잭션. 로더가 빈 번호를 허용해 미머지 브랜치와 번호를 건너뛸 수 있다.
- **화면 사전 검사 관례(4회 연속 채택, v2.8.4~v2.8.7)**: 순수 `.ts` 모듈 → 필드 `error`/`helperText` · 제출 버튼 `disabled` · `submit()` 가드 **세 곳이 같은 값 하나를 읽게** 배선. 서버 경계를 그대로 복제하고 서버는 건드리지 않는다.
- 데이터 로딩: `AdminPage.tsx:68` 처럼 이름 붙은 `const load = async () => {...}` + `useEffect(() => { void load(); }, [])` 가 재시도 가능한 형태의 선례다. `VisitFormPage`·`ScannerPage`·`LobbyPage` 는 아직 익명 `.then().catch()` 다.

## 위험 구역
- `auth.go`(세션/OIDC)·`keys.go`/암호화·`internal/database/migrations/`·`settings.go`·`server.go`·감사/개인정보.
- `normalizePhone`(visits.go:25)은 `watchlist_entries.phone_hash` 의 입력이다 — 편의로 바꾸지 말 것.
- `createVisitRecord` 의 상태코드·에러 코드·문구는 REST·현장·MCP 공용 계약이다. 화면 편의로 바꾸지 말 것.
- HEAD 미머지 브랜치 3개(파일 겹치면 피할 것): `origin/auto/2026-09-16-1212`(메일: platform/mail·settings·AdminPage·integration_test), `origin/auto/2026-09-18-0533`(MCP OAuth: SettingsPage·mcpoauth), `origin/auto/2026-09-21-0654`(가져오기 중복 헤더: import.go·import_test.go·USER_GUIDE·API_AND_MCP). 세 브랜치 모두 v2.8.7 시점에도 그대로 미머지다.

## 자주 깨지는 곳
- 변이 되돌리기에 `git checkout -- <파일>` 을 써서 미커밋 테스트를 두 번 잃었다 — `sed` 나 임시 복사본을 쓸 것.
- 화면과 서버가 같은 규칙을 따로 읽는 자리에서 한쪽만 고쳐 불일치가 생긴다. 회사명 필수(v2.8.4)·일정(v2.8.5)·이름/전화(v2.8.6)·방문자 100명 및 반복 500건(v2.8.7)은 **모두 닫혔다**. 아직 화면 안내가 없는 것: 개인정보 동의 해제·방문 유형 체크리스트·차량/장비 미기재(제출 버튼만 말없이 잠긴다).
- `cmd/visitflow/webdist/index.html` 스텁 복원을 잊으면 빌드 산출물이 커밋된다. `pkill -f 'vf-server'` 는 자기 셸까지 죽여 exit 144 가 난다. `ENCRYPTION_KEY` 는 정확히 32바이트(64자 hex).
- SSE/로비 스트림은 요청 컨텍스트 종료가 없으면 대기한다(헬퍼 데드라인으로 완화됨).

## 검증 함정
- Go PASS 는 DB 통합 실행 증거가 아니다. E2E 는 실제 빌드 UI 를 서빙하는 서버가 있어야 의미가 있다.
- 새 단위 테스트는 `.ts` 로 둘 것(`.tsx` 는 vitest 가 수집하지 않아 조용히 0개가 된다). React state/effect 자체는 단위 테스트 대상이 아니므로 브라우저 확인이 1급 증거다.
- `new Date("2026-02-31T10:00")` 는 V8 이 3월 3일로 롤오버한다 — 파싱 실패 전제를 세울 때 주의.
- excelize `GetRows` 는 표시 서식을 적용한다(숫자 전화·지수 표기). 중복 헤더 경고는 HEAD 에 없고 미머지 브랜치에만 있다.
- 실제 Excel·Keycloak·SMTP 검증 인프라는 이 환경에 없다. PDF 도구는 저장소 밖 `aidev/tools/guide/md2pdf.mjs`.

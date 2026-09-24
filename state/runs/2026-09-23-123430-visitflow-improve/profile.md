# VisitFlow 프로필 (2026-09-23)

- 목적: 사내 방문 예약·승인·QR 방문증·로비 체크인·알림을 단일 컨테이너로 제공하는 방문 관리 시스템.
- 스택: Go 1.24(go.mod), chi·pgx·go-oidc·excelize, PostgreSQL 14+, React 19·TypeScript·Vite·Material UI 7, vitest·Playwright.
- 기준: HEAD 7d3dccb(v2.8.4). 이번 정찰에서 git log 15·브랜치 목록·internal/app 파일 목록·VisitFormPage 전체·visits.go 검증부·web/vite.config.ts·package.json을 직접 열었다.

## 구조
- `cmd/visitflow/`: 서버 진입점. 임베드 `webdist`(기본 스텁 vs 실제 프런트 빌드 구분), `version` 기본값 "dev".
- `internal/app/`: HTTP·MCP·SSE·백그라운드. 파일 ~48개, 약 15.9k줄. `visits.go`(1696줄)의 `createVisitRecord`가 신청 정책·일정 검증의 단일 지점.
- `internal/app/import.go`·`import_test.go`: CSV/XLSX 파싱과 실제 파일/HTTP 회귀. 첫 시트·앞 10행 헤더 탐색 계약.
- `internal/app/integration_test.go`: `newTestEnv`가 테스트별 DB 생성·정리. `do`/`doForwarded`/업로드는 30초 요청 데드라인 사용.
- `internal/platform/`(config·mail·crypto), `internal/database/`(연결 + 번호별 트랜잭션 마이그레이션, 최신 `0014_tracking.sql`).
- `web/src/`: 화면·타입·vitest. `pages/VisitFormPage.tsx`(116줄, 아주 긴 한 줄 스타일)가 `/visits/new`와 `/lobby/walk-in`을 공유한다.
- `web/e2e/visit-flow.spec.ts`(실제 서버 Playwright), `web/screenshots/`(가이드 캡처), `docs/`(USER/ADMIN GUIDE·API_AND_MCP·ARCHITECTURE·PDF).

## 빌드·테스트
- `go test ./... -count=1` — `VISITFLOW_TEST_DSN` 없으면 PostgreSQL 통합이 SKIP된다(PASS가 통합 증거가 아님). DSN 지정 시 internal/app 과거 약 45~57초.
- CI DSN: `postgres://visitflow:visitflow@127.0.0.1:5432/visitflow?sslmode=disable`. DB 통합은 CREATE DATABASE 권한 필요.
- `cd web && npm ci && npm run lint && npm test && npm run build`. `lint`=`tsc -b`, `test`=`vitest run`. **워크트리에 `web/node_modules`가 없으므로 `npm ci`부터 필요**(이번 정찰 미실행).
- vitest `include`는 `src/**/*.test.ts` — `.tsx` 테스트는 수집되지 않는다. 현재 단위 테스트 파일은 `src/silentSso.test.ts` 하나(8개 케이스).
- `cd web && npm run test:e2e`(Playwright). 설정이 서버를 자동 기동하지 않는다 — `web/dist`를 `cmd/visitflow/webdist`에 복사하고 Go 서버·DB·Chromium을 미리 띄워야 한다(준비 비용 큼).
- `docker build -t visitflow:dev .`. CI test 잡: Go 테스트/vet → npm ci/test/build → Docker build. e2e 잡은 실제 빌드 UI로 별도 실행.

## 관례
- 커밋은 영어 conventional(`fix(import): …`), 릴리즈는 `chore(release): VisitFlow vX.Y.Z`. UI·가이드·오류 메시지는 한국어.
- `writeError(w, status, snake_case_code, 한국어 메시지)` 형태. 정책은 `settings` 표에 저장하고 `PUT /api/v1/settings`로 변경(admin/settings 아님). `getSetting`은 5초 캐시이고 `updateSettings`가 즉시 invalidate하므로, 테스트에서 SQL만 바꾸면 옛 캐시를 볼 수 있다 — 설정 API를 쓸 것.
- 마이그레이션은 번호별 SQL 트랜잭션. 로더가 빈 번호를 허용해 미머지 브랜치와 번호를 건너뛸 수 있다.

## 위험 구역
- `auth.go`(세션/OIDC)·`keys.go`/암호화·`migrations/`·`settings.go`·`server.go`·감사/개인정보. `normalizePhone`은 전화 해시 입력이라 가져오기 편의로 바꾸지 말 것.
- HEAD 미머지 브랜치 3개와 파일이 겹치면 피할 것: `origin/auto/2026-09-16-1212`(메일: platform/mail·settings·AdminPage·integration_test), `origin/auto/2026-09-18-0533`(MCP OAuth: SettingsPage·mcpoauth), `origin/auto/2026-09-21-0654`(가져오기 중복 헤더 8cc7374: import.go·import_test.go·USER_GUIDE·API_AND_MCP).
- 서버 검증 계약은 화면보다 넓게 쓰인다(REST·MCP 공용). `createVisitRecord`의 상태코드·코드·문구를 화면 편의로 바꾸지 말 것.

## 자주 깨지는 곳
- 변이 되돌리기에 `git checkout -- <파일>`을 써서 미커밋 테스트를 두 번 잃었다 — `sed`나 임시 복사본으로 되돌릴 것.
- SSE/로비 스트림은 요청 컨텍스트 종료가 없으면 대기한다(헬퍼 데드라인으로 완화됨).
- 화면과 서버가 같은 정책을 따로 읽는 자리(회사명 필수, 인원 상한, 일정 검증)에서 한쪽만 고쳐 불일치가 생기기 쉽다.

## 검증 함정
- Go PASS는 DB 통합 실행 증거가 아니다. E2E는 실제 빌드 UI를 서빙하는 서버가 있어야 의미가 있다.
- excelize `GetRows`는 표시 서식을 적용한다(숫자 전화·지수 표기). HEAD는 숫자 전화 복원과 잘린 지수 경고까지 있으나 **중복 헤더 경고는 HEAD에 없다**(미머지 브랜치에만 있음).
- 가져오기 경고 UI는 처음 5개와 나머지 개수만 표시한다.
- 실제 Excel·Keycloak·SMTP·브라우저 검증 인프라는 이 환경에서 미확인. PDF 도구는 저장소 밖 `aidev/tools/guide/md2pdf.mjs`.

## 현재 신청 화면 상태(2026-09-23 확인)
- `reference-data`가 `companyRequired`를 내려주고 `VisitFormPage`의 `companiesSatisfied`가 표시·버튼 차단·submit 가드까지 연결되어 있다(c9f53e3, v2.8.4로 머지 완료 — 2026-09-22 프로필의 "아직 없다"는 기술은 낡았다).
- 같은 화면에서 체크리스트(`checklistSatisfied`)·차량/장비 선언(`declarationsSatisfied`)도 사전 검사되지만, **방문 시작/종료 시각과 방문자 100명 상한은 사전 검사가 없다** — 이번 회차 과제와 차선 후보의 근거.

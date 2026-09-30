# 정찰 과제서 (2026-09-30, seaton)

- 과제: 직원 목록 조회의 늦은 응답이 최신 필터 결과를 덮지 않게 하고, 재조회 시 낡은 오류 배너를 지우기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `EmployeesPage.load`(web/src/pages/EmployeesPage.tsx:60-73)에는 요청 순번 검사가 없어, 조직/재직상태/배정상태 Select 을 연달아 바꾸면 먼저 보낸 느린 응답이 나중 응답을 덮어써 화면의 필터와 표의 내용이 어긋난다(같은 화면의 `setLoading(false)` 도 낡은 요청이 먼저 끄고, `setError` 도 낡은 실패가 올라온다). 같은 저장소의 `HistoryPage.load`(web/src/pages/HistoryPage.tsx:73-100)는 `requestRef` 순번과 시작 시 `setError("")` 를 이미 갖고 있으므로, 같은 처방을 직원 화면에 옮기면 필터와 표가 항상 같은 조건을 보이고 실패 뒤 재조회가 성공하면 빨간 배너가 스스로 사라진다.

- 수용 기준:
  1) 조직 Select 을 A→B 로 빠르게 바꾸었을 때 A 의 응답이 B 보다 늦게 도착해도 표에는 B 의 결과만 남는다(Select 표시값과 표의 행 수가 일치).
  2) 낡은 요청의 응답은 `items`·`loading`·`error` 중 어느 것도 바꾸지 않는다 — 특히 늦게 도착한 응답이 최신 요청의 로딩 스켈레톤을 먼저 끄지 않는다.
  3) 직원 조회가 실패해 빨간 배너가 뜬 뒤, 원인을 없애고 다시 조회해 성공하면 배너가 사라지고 표가 채워진다(사용자가 X 를 누를 필요가 없다).
  4) 테스트가 증명해야 하는 것: 위 1)·3) 을 **변경 전 이미지에서 먼저 붉게** 만든 뒤 수정본에서 통과. 1) 은 "늦은 응답이 실제로 나중에 도착"하는 상황을 실서버 + `page.route` 지연으로 만들어야 하고(손으로 만든 fetch 대역 금지), 3) 은 `route.abort("failed")` → `unroute` → 재검색으로 만든다.
  5) 기존 직원 화면 동작(URL 필터 복원·검색 제출·CSV 내보내기·가져오기 뒤 재조회)이 그대로 통과한다.

- 건드릴 파일 (프로덕션 1개):
  - `web/src/pages/EmployeesPage.tsx:60-73` — `load`: 함수 위에 `const requestRef = useRef(0)` 를 두고(현재 line 1 의 import 는 `useEffect, useMemo, useState, type FormEvent` 만 가져오므로 `useRef` 를 추가), `load` 시작에서 `const sequence = ++requestRef.current`, `setLoading(true)`, `setError("")`; `setItems`·`catch` 의 `setError` 앞에 `if (sequence !== requestRef.current) return;`, `finally` 는 `if (sequence === requestRef.current) setLoading(false);`. 꼴은 `HistoryPage.tsx:73-101` 을 그대로 따른다(줄 73 의 `requestRef`, 76 의 `sequence`, 92·97 의 조기 반환, 100 의 `finally` 게이트).
  - `web/e2e/employee-request.spec.ts` (새 파일) — 2건: ① 지연 역전 ② 실패 뒤 복구. 기존 `employee-filter.spec.ts` 의 `chooseOrganization` 헬퍼(줄 21-28: `getByRole("combobox", { name: "조직 필터" })` → `getByRole("option", { name, exact: true })`)와 같은 선택자를 쓰고, `login(page)` 는 `./helpers` 에서 가져온다. 지연은 `page.route("**/api/v1/employees?*", …)` 에 카운터를 두어 **두 번째** 매칭 요청만 `await new Promise(r => setTimeout(r, 1500))` 뒤 `route.continue()` 하고 나머지는 즉시 통과시킨다(첫 요청은 `goto` 직후의 초기 조회다). 같은 파일 줄 282·306 과 `login.spec.ts:10,17` 이 이 저장소의 `route`/`unroute` 선례다.
  - 문서: 사용자에게 보이는 새 기능이 아니므로 `docs/` 는 건드리지 않는다(동작 교정). USER_GUIDE 갱신 불필요.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm test && npm run lint && npm run build` — 이번 체크아웃에 `node_modules` 가 없어 `npm ci` 가 먼저 필요하다(이번 정찰 미실행, 이전 회차 159건 통과 기록). `npm run lint` 는 `tsc -b --pretty false`.
  - `gofmt -l .` · `go vet ./...` · `go test ./...` — 서버를 건드리지 않아도 회차 관례상 무출력 확인.
  - 실서버 E2E: `docker build --build-arg VERSION=e2e -t seaton:e2e .` 후 전용 PostgreSQL 16 + 앱을 Docker bridge 로 띄우고(09-29 회차가 host network 대신 bridge + published port 로 성공했다), `cd web && E2E_BASE_URL=http://127.0.0.1:8080 E2E_USERNAME=admin E2E_PASSWORD=ci-e2e-password-123 npx playwright test employee-request employee-filter employee-export`. 준비 확인은 `/readyz`(`/api/v1/health` 는 없다).
  - 역검증: 변경 전 `fa0dba2` 를 `git archive` 로 구워 `seaton:e2e-before` 로 띄우고 새 spec 2건이 붉은 것을 먼저 기록할 것.

- 위험과 피할 것:
  - `applyFilters`(86-)의 "같은 조건 재제출" 분기와 `useEffect`(74-79)의 `void load(filters)` 두 경로가 모두 `load` 를 부른다. 순번은 `load` **안에서만** 올려야 하며 호출부에 순번 로직을 복제하지 말 것(같은 값을 두 경로가 다르게 읽는 것이 이 저장소의 반복 실패다).
  - `upload`(107-)는 `await load()` 를 인수 없이 부른다 — 기본 인수 `filters = readEmployeeParams(searchParams)` 를 유지할 것.
  - `HistoryPage` 를 같이 리팩터하거나 순번 규칙을 `web/src/lib/` 헬퍼로 뽑지 말 것(이번 범위 밖). 이유: 파일이 늘고, 순수 헬퍼 단위테스트는 배선 결함을 증명하지 못한다 — 증명은 실서버 E2E 가 한다. 헬퍼 통합은 별도 후속으로 남긴다.
  - `@testing-library`/`jsdom` 은 `web/package.json` devDependencies 에 **없다**. 컴포넌트 단위테스트를 위해 의존성을 새로 넣지 말 것(범위 외). 따라서 이번 붉은-먼저 증거는 vitest 가 아니라 E2E 다.
  - 보호 경로 회피: `internal/app/auth.go`·`mcpoauth.go`, `internal/database/migrations.sql`, `internal/tracking`, `.github/workflows` 는 건드리지 않는다. 서버 `listEmployees` 도 손대지 않는다.
  - `page.route` 지연 테스트는 `page.unroute` 로 반드시 해제하고, 서버 상태를 바꾸지 않는다(필터만 만짐 — 되돌릴 것이 없다). 워커는 1개라 다른 spec 과 간섭하지 않는다.
  - 미확인: 1500ms 지연이 응답 역전을 안정적으로 만드는지 실제로 돌려보지 않았다(값은 조정 가능한 가정). `api` 헬퍼가 `route.abort("failed")` 에서 내는 오류 문구도 확인하지 않았다 — 배너 문구를 단정하기보다 `getByRole("alert")` 의 존재/부재로 단정할 것.

- 차선 후보: E2E 환경 의존 spec 2건(mcp-oauth·tracking) 준비 절차를 `web/e2e/README.md` 로 고정 (가치 2 / 위험 1 / S) — 문서만이라 위험 0 에 가깝고, 09-29 회차가 bridge + published port 로 실제 기동에 성공한 사실과 "`/api/v1/health` 는 없고 `/readyz` 를 쓴다"를 함께 굳힐 근거가 이미 있다.

## 견적 근거 (basis of estimate, bottom-up)
- 포함: `load` 순번·오류 초기화(≈12줄), 새 E2E spec 2건(≈70줄), 역검증용 before 이미지 빌드·기동, 로컬 검증 4종.
- 제외: 서버 변경, 문서 변경, `HistoryPage` 통합, 컴포넌트 테스트 하네스 도입.
- 범위: 30~55분(10회 중 8회). 변동의 대부분은 Docker 이미지 2개 빌드와 DB 기동이며, 그것이 막히면 E2E 를 한 이미지(수정본)만 돌리고 before 역검증을 코드 되돌림 패치로 대체하는 것이 예비안이다.

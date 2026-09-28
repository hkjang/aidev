- 과제: 직원 화면에 조직 필터 노출 (가치 3 / 위험 1 / 작업량 S)

- 왜: 서버 `listEmployees` 는 `organizationId` 쿼리를 이미 받아 `AND ($2='' OR e.organization_id=$2)` 로 거르는데(`internal/app/employees.go:62,69`), 화면의 `load` 는 `limit·q·status·assignment` 만 보낸다(`web/src/pages/EmployeesPage.tsx:58-61`). 그래서 관리자가 "영업팀 명단만" 을 볼 길이 화면에 없고, 자유 검색어(`q`)로 조직명을 쳐도 그것은 이름·사번·이메일까지 함께 훑는 LIKE 검색이라 정확한 조직 일치가 아니다. 필터를 붙이면 500건 상한(`EmployeesPage.tsx:449` 안내)에 걸리는 큰 조직에서도 팀 단위로 잘라 볼 수 있고, 09-27 에 붙인 `CSV 내보내기` 가 `items` 를 그대로 담으므로(`EmployeesPage.tsx:141-153`) 조직별 명단 파일이 서버 변경 없이 곧바로 나온다.

- 수용 기준:
  1) 직원 화면 필터 줄(`Paper`, EmployeesPage.tsx:297-352)에 조직 Select 가 생기고, 기본값은 `전체 조직` 이다. 고르면 `GET /api/v1/employees` 에 `organizationId=<id>` 가 실려 나가고 표에 그 조직 직원만 남는다.
  2) 그 상태에서 `CSV 내보내기` 를 누르면 걸러진 목록만 파일에 담긴다(`exportCsv` 가 `items` 를 쓰므로 추가 코드 없이 성립해야 한다 — E2E 로 확인만 한다). 조직을 고른 뒤 결과가 비면 내보내기 단추가 `disabled` 가 된다(기존 `disabled={!items.length || loading}`).
  3) vitest 가 쿼리 조립 규칙을 증명한다: 빈 값은 키를 아예 넣지 않고, 값이 있으면 `organizationId` 키로 넣으며, `limit=500` 은 항상 붙는다. 새 헬퍼 파일이 없는 상태에서 `Cannot find module` 로 먼저 붉은 것을 확인하고 구현할 것(이 저장소의 직전 세 회차가 쓴 방식).
  4) 새 E2E 가 실서버에서: 조직 Select 를 `getByRole("combobox", { name: "조직 필터" })` 로 집어 `영업팀` 을 고르면 표가 2행이 되고 모든 행의 '조직' 칸이 `영업팀` 이다(시드 PEOPLE — 개발팀 6·영업팀 2·인사팀 2, `web/e2e/seed.mjs:14-30`). 이어서 CSV 를 내려받아 데이터 행이 2행이고 `개발팀` 문자열이 없음을 단정한다.

- 건드릴 파일 (프로덕션 2개):
  - `web/src/lib/employeeQuery.ts` (신규) — `employeeQuery({ q, status, assignment, organizationId })` 가 `URLSearchParams` 를 짓는다. 빈 문자열은 키를 넣지 않고 `limit` 은 `"500"` 으로 고정. 이 저장소의 `silentSso.ts`·`seatMapLink.ts`·`employeeExport.ts` 와 같은 꼴 — "같은 값을 두 경로가 따로 짓지 않게 규칙을 lib 한 곳에 둔다" 는 반복된 교훈 때문이다.
  - `web/src/pages/EmployeesPage.tsx` —
    - `organizations` 상태와 `useEffect` 에서 `api<{ items: Organization[] }>("/api/v1/organizations")` 1회 로드. 이 엔드포인트는 인증만 요구하고(`internal/app/server.go:82`, `requireSeatManager` 그룹 밖) 이 화면은 이미 `Manager` 가드 안이다(`web/src/App.tsx:70-77`). 조직 로드가 실패해도 목록 조회는 죽지 않게 할 것(조직 Select 만 비고 나머지는 평소대로).
    - `organizationId` 상태 추가. `load` 의 네 번째 위치 인자로 넘긴다(기존 `load(query, nextStatus, nextAssignment)` 의 꼴을 유지). 본문의 수동 `URLSearchParams` 조립을 `employeeQuery` 호출로 교체.
    - 초기 `useEffect` 의 `void load("", "", "")` 도 새 인자에 맞춰 갱신.
    - `assignment` Select 뒤에 조직 `FormControl`+`Select` 추가. `displayEmpty`, 첫 항목 `<MenuItem value="">전체 조직</MenuItem>`, 나머지는 `organizations.map` (서버가 `ORDER BY name` 으로 정렬해 준다). `inputProps={{ "aria-label": "조직 필터" }}` — 이 꼴이 실제로 `getByRole("combobox", { name })` 에 잡히는 것은 `web/src/pages/UsersPage.tsx:260` 과 그것을 집는 `web/e2e/admin.spec.ts:150` 이 지금 통과하는 것으로 확인했다.
    - 덤(같은 파일이라 비용 0): 기존 두 Select 에도 `inputProps={{ "aria-label": "재직상태 필터" }}` / `"배정상태 필터"` 를 붙인다. **보이는 텍스트는 바꾸지 말 것** — `web/e2e/employee-export.spec.ts:52-55` 가 `getByRole("combobox").filter({ hasText: "전체 배정상태" })` 로 집으므로 `MenuItem` 라벨과 `displayEmpty` 표시는 그대로 두어야 한다.
  - 테스트·문서: `web/src/lib/employeeQuery.test.ts` (신규), `web/e2e/employee-filter.spec.ts` (신규 — 또는 `employee-export.spec.ts` 에 덧붙여도 된다), `docs/USER_GUIDE.md` 에 한 줄 + `python3 scripts/build-docs.py USER_GUIDE` 로 HTML 만 재생성(PDF 는 09-17 이후 관례대로 굽지 않는다).

- 검증 명령:
  - `cd web && npm test` (vitest, 현재 142건) → 새 헬퍼 없는 상태에서 먼저 붉은 것 확인
  - `cd web && npx tsc -b && npm run build`
  - `gofmt -l .` · `go vet ./...` · `go test ./...` (서버 무변경이지만 회귀 확인)
  - `git diff --check`
  - E2E(역검증 포함): `docker build -t seaton:e2e .` → PostgreSQL 16 과 함께 기동 → `cd web && E2E_BASE_URL=http://127.0.0.1:18781 E2E_USERNAME=admin E2E_PASSWORD=... npx playwright test employee-filter employee-export admin`. 변경 전 HEAD 를 `git archive` 로 구운 `seaton:e2e-before` 에서 새 spec 이 먼저 붉은 것(`조직 필터` combobox 부재)을 확인할 것 — 최근 성공 회차 네 번이 모두 쓴 방식이다.
  - `tracking`·`mcp-oauth` spec 7~8건은 수집기 호스트·가짜 Keycloak 미기동으로 원래 실패한다. 이번 변경과 무관하므로 돌리지 말거나, 돌렸다면 변경 전 이미지에서도 같은 수가 실패함을 함께 적을 것.

- 위험과 피할 것:
  - **서버를 고치지 말 것.** `listEmployees` 는 이미 `organizationId` 를 받는다. SQL·핸들러를 건드리면 "목록 핸들러가 `rows.Err()` 를 삼킨다"(3/2/M, 별도 과제)와 섞여 회차가 커진다.
  - `internal/app/auth.go`·`migrations.sql`·`.github/workflows` 미접촉.
  - 주소(`?organizationId=`)에 필터를 반영하는 것은 **이번 범위 밖**이다. 좌석맵의 `seatMapLink` 와 같은 일을 직원 화면에도 하려면 상태 네 개의 직렬화 규칙을 새로 정해야 한다 — 별도 과제로 남긴다.
  - 총 건수·페이지네이션(500건 상한)도 범위 밖. 서버 `COUNT` 가 필요하고 501명을 만들어야 관찰된다.
  - 조직 Select 를 바꿀 때 `q` 를 지우지 말 것. 기존 두 Select 가 `void load(q, ...)` 로 현재 검색어를 함께 넘기는 것과 같이 동작해야 한다(`EmployeesPage.tsx:325,340`).
  - 미확인: 조직이 0개인 설치(시드 전)에서 Select 를 어떻게 보일지는 정하지 않았다. `전체 조직` 한 항목만 남는 것으로 충분하다고 보지만 실제 화면은 확인하지 않았다.

- 차선 후보: E2E 환경 의존 spec 2건(mcp-oauth·tracking) 준비 절차를 `web/e2e/README.md` 로 고정 (2/1/S) — 네 회차 연속 "원래 안 돈다" 를 구전으로 판정했다. 문서만이라 코드 위험 0.

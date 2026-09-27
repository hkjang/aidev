# 과제서

- 과제: 직원 화면에서 조회 결과를 CSV 로 내보내기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/pages/EmployeesPage.tsx` 의 머리글 단추는 **양식 내려받기 두 개**(`downloadSeatTemplate`:112, `downloadTemplate`:121 — 둘 다 고정 문자열 한 줄짜리 양식)와 **가져오기 두 개**뿐이고, 화면에 띄운 직원 목록을 파일로 받아 갈 길이 없다. 변경 이력 화면은 같은 일을 `HistoryPage.exportCsv`(`HistoryPage.tsx:128-183`)로 이미 하고 공용 `toCSV`·`safeFileName`·`localDateStamp`(`web/src/lib/format.ts:59/64/101`)까지 있으므로, 같은 규칙을 직원 화면에 이어 주면 미배정자 명단·조직별 현황을 표를 눈으로 베끼지 않고 넘길 수 있다.
- 수용 기준:
  1) `/admin/employees` 머리글에 `CSV 내보내기` 단추가 있고, 목록이 비었거나 불러오는 중이면 눌리지 않는다(`disabled={!items.length || loading}` — 이력 화면과 같은 꼴). 누르면 `직원목록_YYYY-MM-DD.csv`(로컬 달력 날짜, `localDateStamp` 사용) 파일이 내려온다.
  2) 내보낸 CSV 는 **지금 화면에 보이는 그 목록**(검색어·재직상태·배정상태가 적용된 `items` 상태)을 그대로 담는다. 재조회하지 않으므로 표의 행 수와 파일의 데이터 행 수가 같다. 열은 표의 열과 맞춘다: 이름·사번·이메일·조직·직급·직책·근무지·좌석·재직상태.
  3) 표의 `상태` 칸과 CSV 의 재직상태 열이 **같은 함수 한 곳**에서 값을 얻는다 — `EmployeesPage.tsx:391-397` 의 `재직/휴직/퇴직` 삼항 분기를 공용 함수로 옮기고 표도 그 함수를 쓴다.
  4) 테스트가 증명할 것 — vitest: (a) 같은 `Employee` 배열에서 상태 라벨이 `재직/휴직/퇴직`이고 표·CSV 가 같은 함수를 거친다, (b) `seatNo`·`email`·`organizationName` 이 없는 직원은 좌석 칸이 `미배정`이고 나머지는 빈 칸(표의 `-` 를 CSV 에 넣지 말 것 — 스프레드시트에서 `-` 는 수식 방어 대상이라 `'-` 가 된다), (c) `toCSV` 를 거쳐 BOM·CRLF·수식 방어가 그대로 걸린다. E2E: (d) 실제 브라우저에서 내려받기 이벤트가 나고 `suggestedFilename()` 이 `직원목록_`으로 시작한다, (e) 배정상태 `미배정`으로 걸러 목록이 빈 상태에서는 단추가 `disabled` 다(시드 직원 10명은 전원 좌석이 있어 이 필터로 빈 목록을 만들 수 있다 — `web/e2e/seed.mjs:152-161`).
- 건드릴 파일 (프로덕션 2개):
  - `web/src/lib/employeeExport.ts` (신규) — `EMPLOYEE_CSV_HEADERS`, `employeeStatusLabel(status)`, `employeeCsvRows(items)`. 파일 첨머리에 `seatMapLink.ts`·`silentSso.ts` 처럼 "왜 규칙을 여기 한 곳에 두는가" 주석을 둘 것.
  - `web/src/lib/employeeExport.test.ts` (신규) — 위 (a)(b)(c).
  - `web/src/pages/EmployeesPage.tsx` — ① `PageHeader actions`(148-197) 에 `CSV 내보내기` 단추 추가(`DownloadRounded` 이미 import 되어 있다:25), ② `downloadTemplate`(121) 아래에 `exportCsv` 추가 — Blob→`document.createElement("a")`→`click()`→`revokeObjectURL` 은 `HistoryPage.tsx:163-172` 와 **같은 꼴**, `type: "text/csv;charset=utf-8;"`, ③ 표의 상태 칸(391-397)을 `employeeStatusLabel(employee.status)` 로 교체.
  - `web/e2e/employee-export.spec.ts` (신규) — `helpers.ts:19 login(page)` 으로 로그인 후 `page.goto("/admin/employees")`(라우트는 `web/src/App.tsx:71` 의 `admin/employees`), `const wait = page.waitForEvent("download")` 를 **클릭 전에** 걸고 클릭 → `suggestedFilename()` 확인.
  - `docs/USER_GUIDE.md` — `### 3.4 직원`(143행) 의 불릿 목록(149-152)에 한 줄 추가. 147행 캡션이 머리글 단추를 나열하고 있으니(`직원 양식·배정 양식·좌석 일괄 배정·직원 가져오기`) 그 문장에도 새 단추를 넣을 것. **PNG 재촬영(guide-shots)은 범위 밖**, HTML 만 재생성(`python3 scripts/build-docs.py USER_GUIDE`), **PDF 는 굽지 않음**(09-17 이후 관례).
- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` (`lint` 는 tsc 타입 검사)
  - `gofmt -l .` · `go vet ./...` · `go test ./...` (Go 변경 없음 — 초록 유지 확인용)
  - E2E: `docker build -t seaton:e2e .` → PostgreSQL 16 과 함께 기동(기본 URL `http://127.0.0.1:18781`) → `cd web && E2E_BASE_URL=http://127.0.0.1:18781 E2E_USERNAME=admin E2E_PASSWORD=… npx playwright test employee-export.spec.ts admin.spec.ts` (설정이 이미 `workers:1`, `fullyParallel:false`)
  - 역검증: 변경 전 HEAD 를 `git archive` 로 구운 `seaton:e2e-before` 에서 새 spec 이 붉은 것을 먼저 확인(09-23·09-24·09-26·09-27 성공 회차의 공통 방식). vitest 도 헬퍼 없는 상태에서 먼저 붉게 만들 것.
  - `git diff --check`
- 위험과 피할 것:
  - **서버는 건드리지 말 것.** `internal/app/employees.go:listEmployees`(60행) 는 `limit<=500` 만 받고 `total` 을 주지 않는다. 500명 상한 안내는 이미 화면에 있다(`EmployeesPage.tsx:423`). 상한 개선은 차선 후보로 남길 것.
  - 표와 CSV 가 상태·좌석 라벨을 각자 만들면 이 저장소가 반복해 깨진 자리(좌석 상세 '조직' vs 도면 색, '구역 불일치' 이중 정의)와 같은 어긋남이 된다. 값 규칙은 `web/src/lib/` 한 곳에서 짓고 양쪽이 import 할 것.
  - `URL.revokeObjectURL` 을 `click()` 직후 부르는 것은 `HistoryPage` 의 기존 방식이다. 바꾸지 말고 같게 둘 것(이번에 한쪽만 고치면 두 화면의 다운로드 동작이 갈린다).
  - **미확인**: Playwright 에서 blob URL + 분리된(detached) `<a>` 클릭 뒤 즉시 `revokeObjectURL` 하는 다운로드의 본문을 `download.path()`/`createReadStream()` 으로 읽을 수 있는지 이 저장소에서 확인한 적이 없다(다운로드를 다루는 E2E 가 하나도 없다 — `web/e2e` 전체에 `waitForEvent("download")` 없음). 본문 읽기가 불안정하면 E2E 는 다운로드 발생 + 파일명까지만 단정하고 내용은 vitest 로 증명할 것. `<a>` 를 `document.body` 에 붙였다 떼는 식으로 바꾸는 것도 허용하되, 그러면 `HistoryPage` 도 같은 헬퍼를 쓰게 해 두 화면을 갈라놓지 말 것(그 경우 프로덕션 파일 3개).
  - 보호 경로(`internal/app/auth.go`, `mcpoauth.go`, `internal/database/migrations.sql`, `.github/workflows`)는 건드리지 않는다.
  - 새 spec 은 좌석·배정 상태를 바꾸지 않으므로 복구가 필요 없다. 필터를 건드린 뒤 다른 spec 에 영향을 주지 않도록 페이지 상태만 쓰고 API 로 쓰기 하지 말 것.
- 차선 후보: 직원 목록 500건 상한에 `HistoryPage` 처럼 `total`·`더 보기` 붙이기 — `internal/app/employees.go:listEmployees` 에 COUNT 를 더해야 하고(파일 4~5개), DB 붙은 Go 테스트 하네스가 없어 증명 수단이 E2E 뿐이며 501명을 만들어야 관찰된다. 이번 회차에는 무겁다.

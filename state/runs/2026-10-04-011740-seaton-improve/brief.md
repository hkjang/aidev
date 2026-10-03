# 과제서 (2026-10-04, base main@61dbe3c / v1.4.14)

- 과제: 직원 가져오기가 재직상태 값을 검증하고, 저장 실패 사유에 DB 원문을 보이지 않게 한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `importEmployees`(internal/app/employees.go:229-248)는 파일의 `재직상태` 칸을 "재직/휴직/퇴직" 세 낱말만 코드로 바꾸고(231-237) 그 밖의 값은 그대로 INSERT 에 넣는다 — `employees.status` 에는 `CHECK (status IN ('active','leave','retired'))`(internal/database/migrations.sql:34)가 걸려 있으므로 "재직중"·"휴가"·"Active" 같은 흔한 오타 한 칸이 그 행을 DB 제약 위반으로 떨어뜨리고, 실패 사유로 `err.Error()` 를 그대로 담아(244) 화면의 "반영되지 않은 N행" 목록(web/src/pages/EmployeesPage.tsx:292-301, `item.error` 를 그대로 렌더)에 pgx 원문(제약 이름·SQLSTATE)이 뜬다. 관리자는 파일의 어디를 어떻게 고쳐야 하는지 알 수 없다. 같은 저장소의 인사 동기화는 이미 이것을 올바르게 한다 — `runEmployeeSync`(internal/app/sync.go:121-127)는 INSERT 앞에서 세 값만 받고 아니면 한국어 문장으로 되돌린다. 같은 값(`employees.status`)을 두 입력 경로가 다르게 읽는, 이 저장소에서 반복된 어긋남이다.

- 수용 기준:
  1) `재직상태` 칸에 세 코드(`active`/`leave`/`retired`)·세 낱말(`재직`/`휴직`/`퇴직`)·빈 값이 아닌 값이 적힌 행은 사람이 읽을 수 있는 한 문장(예: `재직상태 값을 알 수 없습니다: 재직중 (재직/휴직/퇴직)`)을 사유로 돌려주고, 그 사유에 `SQLSTATE`·`constraint`·`relation` 같은 DB 낱말이 없다. 같은 파일의 나머지 행은 그대로 반영된다(부분 성공 유지).
  2) `findOrganization` 이 돌려주는 사용자 문장(`조직코드 X 에 해당하는 조직이 없습니다`, `조직명 X 인 조직이 여러 개입니다…`)은 지금처럼 그 행의 사유로 계속 보인다 — 2026-10-02 회차가 만든 이 동작을 깨뜨리지 말 것. 그 밖의 오류(DB 오류)는 `저장하지 못했습니다` 같은 고정 문장으로 접는다.
  3) 테스트가 증명할 것: (a) 상태 정규화 함수의 Go 단위테스트 — 세 코드·세 낱말·앞뒤 공백·빈 값(→`active`)이 통과하고 모르는 값은 오류, (b) 실서버 E2E — 상태 오타가 든 CSV 를 올리면 실패 행에 그 한국어 문장이 뜨고 `SQLSTATE` 문자열은 화면에 없으며, 같은 파일의 정상 행은 반영된다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/app/employees.go` — ① `importEmployees` 안의 231-237 인라인 분기를 대신할 순수 함수(예: `normalizeEmployeeStatus(raw string) (string, error)`)를 파일 상단 또는 `findOrganization` 옆에 추가하고 행 루프에서 호출, 사번/이름 누락 검사와 같은 자리에서 `failures` 에 담는다. ② 사용자에게 보여도 되는 오류를 표시하는 타입 하나(예: `type inputError struct{ msg string }` + `Error()`, 또는 `errors.Is` 용 sentinel 래퍼)를 만들어 `findOrganization` 의 세 `fmt.Errorf`(117/144/146)와 새 상태 오류가 그것을 돌려주게 하고, 244 의 사유를 `errors.As`/`errors.Is` 로 갈라 사용자 문장만 통과시킨다. ③ (선택, 같은 파일이라 범위 안) `upsertEmployee`(196-200)가 지금은 모든 오류를 `409 employee_conflict` 고정 문장으로 덮어 단건 저장에서는 조직코드 오류 사유가 사라진다 — 같은 분기를 써서 사용자 오류는 `400`+그 문장으로 돌려주면 일관된다. 이번 정찰에서 `grep -rn "employee_conflict" web internal docs` 를 돌려 **employees.go:198 한 곳에만 있고 어떤 테스트·문서도 409 를 단정하지 않음**을 확인했다. 그래도 ③은 선택이다 — ①②만으로 수용 기준이 다 선다.
  - `internal/app/employees_test.go` (신규) — `normalizeEmployeeStatus` 단위테스트. 이 패키지에는 이미 DB 없는 `auth_test.go`·`analyzer_test.go`·`tracking_test.go`·`vlm_test.go` 가 있으니 그 꼴을 따른다(DB 하네스는 만들지 말 것 — 이 패키지에는 없고, 이번 과제는 순수 함수라 필요 없다).
  - `web/e2e/employee-import.spec.ts` — f32ea65 가 만든 기존 spec. 이번 정찰에서 열어 확인한 헬퍼를 **그대로 재사용**하라: `TARGET = "E007"`(36행), `csv(rows)`(51행), `EMPLOYEE_INPUT = "input[type=file] >> nth=1"`(58행), `fetchEmployee`(60행), `keepingEmployee(page, employeeNo, fn)`(76행 — 원래 값을 모두 되돌린다). 이것으로 "상태 오타 행 1개 + 정상 행 1개" CSV 업로드 케이스를 한 건 더한다. 새 직원을 만들지 말고 시드 직원(E007 등)을 쓰되 끝나고 원래 값으로 되돌릴 것.
  - `docs/USER_GUIDE.md` + `docs/USER_GUIDE.html` — 3.4 절(직원 가져오기)에 `재직상태` 칸이 받는 값 한 문장. HTML 은 `python3 scripts/build-docs.py USER_GUIDE` 로 다시 굽고 **PDF 는 건드리지 말 것**(09-17 이후 관례).

- 검증 명령:
  - `go test ./internal/app -run Status -v` → 변경 전 "no tests to run"/컴파일 실패가 먼저 붉은 것을 확인한 뒤 통과시킬 것.
  - `go test ./internal/app` — 이번 정찰에서 실제로 돌려 `ok github.com/hkjang/seaton/internal/app` 를 확인했다. `gofmt -l .` · `go vet ./...` · `go test ./...` 전체는 이번에 돌리지 않음(이전 회차 기록상 통과).
  - `cd web && npm ci && npm test && npm run lint && npm run build` (`lint`=`tsc -b`. 이 체크아웃에 `node_modules` 가 없다)
  - 역검증 포함 실서버 E2E (이전 회차가 쓴 절차 그대로):
    `git archive HEAD | docker build -t seaton:e2e-before -` 로 변경 전 이미지를 **새로** 굽고(동명 이미지가 낡아 있을 수 있다 — 10-02 회차가 그 함정에 걸렸다), 전용 bridge 네트워크 + `postgres:16-alpine` 으로 띄워 새 E2E 가 붉은 것을 먼저 확인 → 수정본 이미지에서 통과. 포트는 8080/18781 을 고정하지 말 것(이전 회차 컨테이너가 쥐고 있을 수 있다). 준비 확인은 `/readyz`. `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 필요.
    `cd web && E2E_BASE_URL=http://127.0.0.1:<port> E2E_USERNAME=admin E2E_PASSWORD=ci-e2e-password-123 npx playwright test employee-import employee-filter`
  - curl 로 결함 재현(권장, 변경 전 이미지): `재직상태` 가 `재직중` 인 1행 CSV 를 `POST /api/v1/employees/import` 에 올려 응답 `failures[0].error` 에 pgx 원문이 오는 것을 받아 적어라 — **이번 정찰은 서버를 띄우지 않았으므로 그 원문의 정확한 문장은 미확인이다**(CHECK 제약이 있다는 것까지만 확인). 실제로 어떤 문장이 오는지 보고 접기 전후를 기록할 것.

- 위험과 피할 것:
  - `internal/app/sync.go` 의 status 검증(121-127)은 건드리지 말 것 — 이미 올바르고, 인사 동기화는 트랜잭션 전체를 되돌리는 계약(한 행이 틀리면 `return nil, err`)이라 가져오기의 부분 성공 계약과 다르다. 두 경로를 합치려 들지 말고 값 규칙(세 코드)만 공유하라.
  - **부분 성공 계약을 깨지 말 것**: 상태가 틀린 행은 `continue` 로 건너뛰고 나머지는 저장돼야 한다. 전체를 400 으로 거절하면 회귀다.
  - 빈 `재직상태` 는 지금 `saveEmployee`(157-159)에서 `active` 가 된다. 새 함수가 빈 값을 오류로 만들면 조직/직급만 고치는 흔한 파일이 전부 거절된다 — 빈 값은 반드시 통과시켜라.
  - `employeeExport.ts` 는 CSV 에 `재직`/`휴직`/`퇴직` 한국어 라벨을 쓰고(`employeeStatusLabel`), 양식(`EmployeesPage.downloadTemplate`, 175행)은 예시로 `active` 를 쓴다. 두 입력 모양이 모두 계속 통과해야 한다 — 어느 한쪽만 받게 하면 양식이나 내보낸 파일이 깨진다.
  - 보호 경로 금지: `auth.go`·`mcpoauth.go`·`internal/database/migrations.sql`·`internal/tracking`·`.github/workflows`. 특히 `migrations.sql` 의 CHECK 를 완화해서 "고치지" 말 것 — 검증은 입력단에서 한다.
  - `seats.go` 의 `readSpreadsheet`(BOM 처리)는 10-02 회차가 고친 자리다. 이번 과제는 그 뒤의 값 해석만 다루므로 열지 말 것.
  - 사유 문장을 접을 때 `findOrganization` 의 세 문장이 함께 접히면 10-02 회차가 세운 동작이 조용히 사라진다 — E2E 에 조직코드 오류 케이스가 이미 있으니 그것이 계속 녹색인지 확인하라.

- 차선 후보: `importEmployees` 로 직원을 `retired` 로 바꿀 때 좌석이 그대로 남는 것 (가치 3 / 위험 3 / M) — `runEmployeeSync`(sync.go:135-145)는 퇴직자의 좌석을 자동 해제하고 이력을 남기는데 가져오기 경로는 하지 않아, 파일로 퇴직 처리한 직원이 대시보드의 `retiredAssignments` 로 남는다. 다만 "파일 한 장으로 좌석을 해제해도 되는가"는 동작 결정이 필요하고 좌석·이력 테이블을 건드리므로 위험이 더 높다. 1순위가 성립하지 않으면(예: 상태 검증이 이미 들어가 있다면) 이쪽 대신 `E2E 환경 의존 spec 2건(mcp-oauth·tracking) 준비 절차를 web/e2e/README.md 로 고정`(2/1/S, 문서만, 위험 0)을 고르라.

- 과제: 좌석 목록(`listSeats`)이 행 읽기 실패를 삼키고 부분 좌석맵을 200 으로 돌려주는 것을 멈춘다 (가치 3 / 위험 2 / 작업량 S)
- 왜: 직전 회차(a3cc4a5)가 직원·조직·이력 세 목록에 `rowScanner` + `scan*` 순수 함수 꼴을 깔았는데 `internal/app/seats.go:33-39` 의 `listSeats` 는 아직 옛 꼴 `if rows.Scan(...) == nil { append }` 이고 `rows.Err()` 를 전혀 보지 않는다(실제로 읽어 확인함). 좌석맵은 이 서비스의 중심 화면이라 조회가 중간에 끊기면 "좌석 40개" 대신 "좌석 12개" 가 200 으로 나가고, 관리자는 그것이 도면 전부라고 믿어 이미 있는 자리에 좌석을 또 만들거나 비어 있지 않은 자리를 비었다고 본다.
- 수용 기준:
  1) `listSeats` 가 행 Scan 에 실패하거나 `rows.Err()` 가 비지 않으면 부분 목록 200 이 아니라 `notFoundOrServer` 를 통해 `database_error` 500 으로 응답한다(직원·조직·이력 세 핸들러와 같은 처리).
  2) 정상 경로의 응답 계약이 한 글자도 바뀌지 않는다 — SQL·열 순서·`{"items":[...]}` 키 이름 그대로이고, 결과가 0건이면 JSON 이 `null` 이 아니라 `[]` 다(`items := []Seat{}` 유지).
  3) 새 Go 단위테스트가 세 가지를 단정한다: ① 모든 행이 그대로 담기고 열 순서 짝이 맞다(`seatNo`·`employeeNo`·`employeeOrganizationName`·`employeeWorkplace` 가 서로 섞이지 않음) ② Scan 오류가 그대로 올라오고 첫 실패에서 멈춘다(`rows.scans == 1`) ③ `rows.Err()` 가 비지 않으면 그 오류가 올라온다. 구현 전에 테스트만 먼저 써서 `undefined: scanSeats` 로 build failed 를 확인하고, 구현 뒤에는 옛 `== nil` 루프를 헬퍼 안에 잠깐 되돌려 넣어 같은 테스트가 실제로 붉어지는지(인과) 확인할 것.
- 건드릴 파일:
  - `internal/app/seats.go:18-41 listSeats` — 스캔 루프를 같은 파일 안의 새 순수 함수 `func scanSeats(rows rowScanner) ([]Seat, error)` 로 뽑고, 핸들러는 `items, err := scanSeats(rows); if err != nil { notFoundOrServer(w, err); return }` 로 바꾼다. `rowScanner` 인터페이스는 `internal/app/employees.go:20-24` 에 이미 있고 같은 패키지이므로 새로 만들지 말 것. 규약 주석도 employees.go:26-33 에 이미 있으니 복제하지 말고 참조만 할 것.
  - `internal/app/seats_test.go`(없으면 새로) 또는 `internal/app/employees_test.go` — `scanSeats` 테스트 3건. **`fakeRows`(employees_test.go:186-238)의 Scan 타입 스위치는 지금 `*string`·`**string`·`*any` 만 받는다.** `Seat` 는 `X/Y/Width/Height/Rotation`(float64)와 `Confidence`(*float64)를 쓰므로 `case *float64:` 와 `case **float64:` 두 가지를 그 스위치에 더해야 한다 — 이 두 줄이 이번 과제에서 가장 쉽게 빠뜨리는 자리다. 테스트를 새 파일에 두더라도 `fakeRows`/`ptr` 는 같은 패키지이니 재선언하지 말고 그대로 쓸 것.
  - 프런트 변경 없음 — 응답 계약이 그대로다. 500 이 났을 때의 표시는 이미 있다: `web/src/pages/SeatMapPage.tsx` 의 좌석 조회 네 자리(`load` 410, `chooseMap` 665, `reloadSeats` 794)가 모두 `try/catch`+`setError` 안에 있고 `reloadSeats` 의 유일한 호출자 `unassignSeat`(814)도 catch 를 갖고 있음을 실제로 읽어 확인했다.
- 검증 명령:
  - `go build ./... && go vet ./... && go test ./internal/app && go test ./... && gofmt -l .`(출력 없어야 함) && `git diff --check`
  - 역검증: 테스트 파일만 먼저 커밋 없이 써서 `go test ./internal/app` 가 `undefined: scanSeats`(build failed)로 떨어지는 것을 먼저 확인.
  - 프런트·E2E 는 돌릴 필요 없음(변경 없음). 돌린다면 `cd web && npm ci && npm test` 는 159건 기준.
  - (선택, 시간이 남을 때만) 실서버 대조: 전용 bridge + `postgres:16-alpine` + `-p 127.0.0.1:<빈포트>:8080`, 준비 확인은 `/readyz`. 직전 회차가 쓴 수법 그대로 — 풀 연결 여러 개를 데운 뒤 `ALTER TABLE seats ALTER COLUMN rotation TYPE integer` 로 캐시된 plan 을 깨면 pgx 가 `cached plan must not change result type (SQLSTATE 0A000)` 을 `rows.Err()` 에만 두고 `Next()` 는 false 로 돌린다. 변경 전 이미지는 `{"items":[]}` 200, 수정본은 `database_error` 500 이어야 한다. **변경 전 이미지는 반드시 `git archive` 로 새로 구울 것**(동명 이미지가 낡아 있던 사고가 있었다).
- 위험과 피할 것:
  - **다른 파일의 같은 패턴을 함께 고치지 말 것.** `maps.go:27·72·126`, `keys.go:28`, `detection.go:468`, `mcp.go:113·127·153`, `dashboard.go`, `grid.go:150`, `tracking.go`, `settings.go`, `mcpoauth.go` 에 같은 꼴이 남아 있지만 이번 회차는 `seats.go` 한 파일이다(파일 수가 재작업률을 가른다). 남은 것은 ideas.json 에 그대로 둔다.
  - SQL 문자열·`ORDER BY`·필터 파라미터($1~$4)·`limit` 류를 손대지 말 것. 이번 변경은 루프를 함수로 옮기는 것뿐이다.
  - 루프를 함수로 옮길 때 가장 조용히 깨지는 것이 **열 순서 19개의 짝**이다(`&item.EmployeeOrganizationID` 와 `&item.OrganizationID` 가 둘 다 `*string` 이라 바꿔 넣어도 컴파일된다). 수용 기준 3-①의 테스트가 그것을 막는 장치이니 생략하지 말 것.
  - Scan 오류로 루프를 중간에 끊을 때는 `rows.Err()` 를 보지 말고 Scan 오류 자체를 돌려줄 것(employees.go 의 규약 주석과 동일 — pgx 가 Scan 실패 시 rows 를 닫는다).
  - 보호 경로(auth.go·mcpoauth.go·migrations.sql·.github/workflows)는 건드리지 않는다. `internal/app/seats.go` 의 나머지(`performAssignment`·일괄 배정·`readSpreadsheet`)도 이번 범위 밖이다.
- 차선 후보: `importEmployees` 의 실패 행 사유에 어느 열이 비었는지 적기 (2/1/S) — `internal/app/employees.go` 의 `fail("사번/이름 누락")` 이 사번·이름 중 무엇이 빈지 가리지 않는다. 같은 파일의 `normalizeEmployeeStatus` 가 세운 "어느 값이 문제인지 적는다" 꼴에 맞추면 되고 프로덕션 1파일이다. 그 문장을 단정하는 테스트·E2E 가 없어 문장을 바꿔도 깨지는 것이 없음(네 회차 연속 차선으로 밀림).

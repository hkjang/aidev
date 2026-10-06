# 회차 노트 2026-10-06-191815-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:18] base pinned — main@e86f453
- [러너 19:18] autonomy release — 

## 정찰 노트
- 좌석맵은 이 서비스의 중심 화면인데 `listSeats`(seats.go:36)만 옛 `== nil` 꼴로 남아 있어 골랐다 — 직전 회차가 rowScanner/fakeRows 를 깔아 두어 프로덕션 1파일 S 로 끝나고, 같은 접근이 바로 전에 채택·릴리즈됐다(반려 이력 아님).
- 차선으로 밀린 "실패 행 사유에 어느 열이 비었는지"(2/1/S)와 "listEmployees 필터 미검증"(2/1/S)은 가치가 낮고, mcp.go 의 `_ = rows.Scan`(새 후보 3/2/S)은 같은 묶음이라 다음 회차로 미뤘다 — 한 회차 한 파일 규칙.
- 추측이 아닌 것: seats.go:33-39, rowScanner 위치, fakeRows 타입 스위치의 한계(*float64 없음), SeatMapPage 네 호출자의 try/catch 를 모두 열어 확인했고 `go build ./... && go test ./internal/app` 녹색 기준선도 확인했다.
- 미확인: 실서버에서 이 경로를 실제로 깨뜨려 보지 않았다 — 과제서의 `ALTER TABLE seats ALTER COLUMN rotation TYPE integer` 는 직전 회차가 organizations 로 성공시킨 수법의 유추이고 선택 검증으로 적었다. 주된 증명은 단위테스트다.
- 구현자가 조심할 것: 열 19개의 짝(`OrganizationID`·`EmployeeOrganizationID` 가 둘 다 `*string` 이라 뒤바뀌어도 컴파일된다)과 fakeRows 에 `*float64`/`**float64` 추가. 다른 파일의 같은 패턴은 손대지 말 것.
- [러너 19:21] scout done — 좌석 목록(`listSeats`)이 행 읽기 실패를 삼키고 부분 좌석맵을 200 으로 돌려주는 것을 멈춘다 (가치 3 / 위험

## 구현 노트
- `listSeats` 의 옛 `if rows.Scan(...) == nil { append }` 루프를 같은 파일의 순수 함수 `scanSeats(rows rowScanner) ([]Seat, error)` 로 뽑고 핸들러가 `notFoundOrServer` 로 500 을 내게 했다. 프로덕션 1파일(seats.go), 커밋 2b81dec. SQL·열 순서·응답 키는 건드리지 않았다 — Scan 인수 19개는 diff 가 보이는 대로 바이트 단위로 동일하다.
- 확신하는 것: 열 순서 가드가 실제 가드다(`OrganizationID`↔`EmployeeOrganizationID` 를 일부러 맞바꿔 테스트가 붉어지는 것을 보았다). 인과도 확인했다(옛 루프를 헬퍼에 되돌려 넣으면 같은 테스트가 붉어진다). 실서버 대조까지 했다 — 같은 깨진 DB 에서 변경 전 `{"items":[]}` 200 ↔ 수정본 `database_error` 500, 나머지 11회는 양쪽 동일. 원인은 DB 로그의 `cached plan must not change result type` 로 확정(추측 아님).
- 확신 없는 곳: 실서버 대조는 `rotation` 열 타입을 깨는 **한 가지** 방법으로만 했다. 다른 종류의 중단(연결 끊김·취소)에서 pgx 가 같은 자리에 오류를 두는지는 직접 보지 않았고 단위테스트의 `fakeRows` 로만 덮었다 — 비평가가 먼저 볼 자리는 여기다. (처음에 함께 적었던 "`notFoundOrServer` 가 404 로 가를 수 있는지"는 그 뒤 읽어서 닫았다: `pgx.ErrNoRows` 하나만 404 이고 그것은 `QueryRow` 전용 센티널이라 스캔 루프에서는 나오지 않는다 — 깨진 목록 조회는 항상 500 이고, 직원·조직·이력 세 핸들러도 같은 함수를 쓴다.)
- 일부러 하지 않은 것: 과제서 지시대로 `maps.go`·`keys.go`·`detection.go`·`mcp.go`·`dashboard.go` 등의 같은 패턴은 손대지 않았다(한 회차 한 파일). 프런트·문서·E2E 무변경 — 응답 계약이 그대로이고 500 표시는 `SeatMapPage.tsx` 의 네 호출자가 이미 `try/catch`+`setError` 로 갖고 있다. 프런트 테스트(`npm test`)는 web/ 변경이 없어 돌리지 않았다.
- 다음 역할이 조심할 것: 새 테스트 3건은 DB 없이 도는 순수 테스트다(`go test ./internal/app` 로 충분). `fakeRows`(employees_test.go)의 타입 스위치에 이번에 `*float64`·`**float64` 를 더했으니 다른 패키지 파일에서 재선언하지 말 것. `ptrFloat` 헬퍼는 seats_test.go 에 있고 `ptr`(string)은 employees_test.go 에 있다.
- [러너 19:28] brief accepted — 채택 — 과제서가 지목한 자리와 수단이 모두 지금 코드와 맞았다. `seats.go:33-39` 의 옛 루프, `rowScanner`(employees.go:20-24)와 �
- [러너 19:29] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 인과 역검증을 직접 재현했다(옛 `== nil` 루프로 되돌리면 ScanFailure·RowsErr 두 테스트가 원장과 같은 문구로 붉어진다). 열 순서 가드도 실제 가드다 — OrganizationID↔EmployeeOrganizationID, X↔Y 를 맞바꿔 각각 seats_test.go:56,59 / :48 이 붉어지는 것을 보았다. 열 19개를 SQL·types.go 와 하나씩 대조했고 `go build/vet/test ./... && gofmt -l .` 전부 녹색이다.
- 구현자의 "자신 없는 곳"은 결함이 아니었다: 헬퍼가 Scan 반환값과 rows.Err() 두 자리를 모두 보므로 중단 종류와 무관하게 500 이고, `pgx.ErrNoRows`→404 경로는 QueryRow 전용 센티널이라 닫혀 있다.
- 부분200→500 의 blast radius 도 닫혔다: 비포인터 스캔 열은 migrations.sql:123-142 에서 전부 NOT NULL/COALESCE 라 정상 데이터가 새로 500 을 내는 길이 없고, 프런트 호출자 3곳(SeatMapPage.tsx:410·665·795)은 모두 try/catch→setError 안이다.
- 못 본 것: 실서버 대조(깨진 DB 12회 조회)는 원장 기록을 믿었고 직접 재현하지 않았다. E2E·프런트 테스트도 돌리지 않았다(web/ 무변경).
- 승인 후 남는 것: 릴리즈 노트에 "좌석 목록 조회 실패 시 부분 목록 대신 500 + 오류 배너"를 적을 것. employees_test.go:231 의 `f, ok :=` 가 수신자 `f` 를 가린다(지금은 무해). 다음 회차 후보는 mcp.go 의 `_ = rows.Scan` — MCP 클라이언트에 조용한 부분 목록을 주는 경로다.
- [러너 19:32] review approved — 리뷰 승인 (risk=low)
- [러너 19:32] pr created — https://github.com/hkjang/seaton/pull/45
- [러너 19:37] ci passed — 검사 2개 모두 success
- [러너 19:37] merge done — 2b81dec
- [러너 19:48] release published — v1.4.17
- [러너 19:50] assets verified — v1.4.17 자산 1개 (이전 v1.4.16: 1)

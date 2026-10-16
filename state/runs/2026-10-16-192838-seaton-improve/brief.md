- 과제: 건물·층·도면 목록(`internal/app/maps.go`)이 행 읽기 실패를 삼키고 부분 목록을 200 으로 돌려주는 것을 멈춘다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `listBuildings`(maps.go:17-31)·`listFloors`(59-77)·`listFloorMaps`(105-131) 세 핸들러만 옛 꼴 `if rows.Scan(...) == nil { append }` 로 남아, 읽기에 실패한 행을 조용히 버리고 `rows.Err()` 를 전혀 보지 않는다. 건물·층 목록은 좌석맵·도면 업로드 화면의 선택 기준이라 한 행이 조용히 빠지면 관리자는 "그 건물에 층이 없다"고 믿고 같은 층을 다시 만들거나 도면을 다시 올린다. 직원·조직·이력(a3cc4a5)과 좌석(2b81dec)에는 이미 올바른 꼴이 깔려 있어 이 세 자리가 남은 마지막 REST 목록이다.
- 수용 기준:
  1) 세 목록이 Scan 실패와 `rows.Err()` 를 삼키지 않고 `notFoundOrServer` 를 거쳐 500 `database_error` 로 응답한다 (부분 목록을 200 으로 내보내지 않는다).
  2) 정상 경로의 응답 계약이 한 글자도 바뀌지 않는다 — SQL·`ORDER BY`·쿼리 파라미터(`buildingId`·`floorId`)·`{"items":[...]}` 와 모든 키 이름(`buildingId`·`buildingName`·`sortOrder`·`fileName`·`contentType`·`createdAt`·`floorName`·`buildingName`·`seatCount`·`reviewCount`·`contentUrl`·`previewUrl`·`overlayReady`·`grid`) 그대로. 목록은 계속 `[]map[string]any{}` 로 시작해 JSON 이 `null` 이 되지 않는다.
  3) `internal/app/maps_test.go`(신규)가 세 헬퍼마다 네 가지를 단정한다: (a) 돌려준 모든 행이 목록에 담기고 **열 짝이 맞다** — 특히 `listFloorMaps` 의 `fname`/`bname`(둘 다 string, 뒤바꿔도 컴파일됨)과 `seatCount`/`reviewCount`(둘 다 int)를 서로 다른 값으로 넣어 짝을 못 박는다 (b) Scan 실패를 오류로 돌려주고 첫 실패에서 멈춘다(`rows.scans == 1`) (c) `rows.Err()` 를 오류로 돌려준다 (d) 빈 결과가 `nil` 이 아니라 `[]` 다.
- 건드릴 파일 (프로덕션 1개 + 테스트 2개):
  - `internal/app/maps.go` — 세 루프를 같은 파일의 순수 함수 `scanBuildings(rows rowScanner) ([]map[string]any, error)`, `scanFloors(...)`, `scanFloorMaps(...)` 로 뽑는다. 핸들러는 `defer rows.Close()` 뒤에 `items, err := scanX(rows); if err != nil { notFoundOrServer(w, err); return }` + 기존 `writeJSON(w, 200, map[string]any{"items": items})`. **Scan 인수와 map 리터럴은 바이트 단위로 그대로 옮길 것** — 열 순서가 가장 조용히 깨지는 자리다. `listFloorMaps` 의 `parseSeatGrid(gridRaw)`(grid.go:42) 호출과 `overlayReady: width != nil && height != nil` 도 그대로 헬퍼 안으로 옮긴다.
  - `internal/app/employees_test.go:204-252` `fakeRows.Scan` — 타입 스위치에 지금 `*string`·`**string`·`*float64`·`**float64`·`*any` 다섯 case 만 있다. maps.go 가 쓰는 네 타입 `*int`(sortOrder·seatCount·reviewCount), `**int`(width·height), `*bool`(active), `*[]byte`(gridRaw) case 를 더해야 한다. 안 더하면 `default` 가 `fakeRows: dest[%d] 타입을 모른다` 로 떨어진다 — 2026-10-06 회차가 `*float64`·`**float64` 를 빠뜨려 같은 데 걸렸다. `*any`(created) 는 이미 있다.
  - `internal/app/maps_test.go` (신규) — 위 테스트. `seats_test.go:24-96` 의 꼴(행 생성 헬퍼 + 세 테스트 + `ptr()`)을 그대로 따른다. `errors.Is` 로 단정할 것.
- 검증 명령 (이 저장소에서 실제로 도는 것, 이번 정찰에서 `go test ./internal/app` 를 돌려 11.478s 로 통과 확인):
  - 역검증 먼저: `maps_test.go` 만 쓰고 `go test ./internal/app` → `undefined: scanBuildings` 로 build failed 를 확인한 뒤 구현한다.
  - 인과 확인: 구현 후 옛 `== nil` 루프를 헬퍼 안에 되돌려 넣어 같은 테스트가 실제로 붉어지는 것을 보고 되돌린다.
  - `go build ./...` / `go vet ./...` / `go test ./...` / `gofmt -l .`(출력 없음) / `git diff --check`
  - 프런트·문서는 응답 계약이 그대로이므로 손대지 않는다. `npm`·Playwright·Docker 는 이번 회차에 필요 없다.
- 위험과 피할 것:
  - `rowScanner` 인터페이스와 `scanX` 규약 주석은 `employees.go:20-33` 에 이미 있다. 같은 패키지이므로 **maps.go 에 다시 선언하지 말고 참조만 하라**(2026-10-06 회차가 그렇게 했다).
  - 다른 파일의 같은 패턴은 **손대지 말 것**: `mcp.go:113·127·153`, `keys.go:28`, `detection.go:468`, `settings.go:35`, `grid.go:150`, `dashboard.go:317`, `tracking.go:68`. 파일 수가 늘면 재작업률 규칙에 걸린다.
  - `dashboard.go:172·194·213·233` 은 이미 Scan 오류와 `rows.Err()` 를 처리한다 — 결함이 아니니 건드리지 말 것.
  - 보호 경로: `auth.go`·`mcpoauth.go`(인증), `internal/database/migrations.sql`(시작 시 DB 변경), `internal/tracking`(CSP), `.github/workflows`. 이번 과제는 전부 피한다.
  - `tracking.go:68`·`mcpoauth.go:82` 의 `err == nil` 은 설정 키/값 로더로 성격이 다르다 — 같은 결함으로 묶지 말 것.
  - 미확인: 이번 정찰에서 실서버로 이 경로를 깨뜨려 보지는 않았다. 직전 두 회차가 쓴 수법(같은 DB 에 변경 전/후 이미지를 붙이고 `ALTER TABLE <t> ALTER COLUMN <int열> TYPE integer` 로 캐시된 plan 을 깨 `cached plan must not change result type` 를 일으킴)이 `buildings`·`floors`·`floor_maps` 에도 통할 것으로 보이지만 **유추**다. 단위테스트만으로 수용 기준을 채울 수 있으니, 실서버 대조는 시간이 남을 때만 덧붙이고 그것 때문에 회차를 비우지 말 것(직전 두 회차가 no-change 로 끝났다).
- 차선 후보: 직원 가져오기 실패 행 사유에 어느 열이 비었는지 적기 — `internal/app/employees.go` 의 `fail("사번/이름 누락")` 이 사번·이름 중 무엇이 빈지 가리지 않는다. 프로덕션 1파일, 순수 함수 단위테스트로 증명 가능. (여섯 회차 연속 차선으로 밀려 있다.)

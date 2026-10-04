- 과제: 시스템 로그 DB·감사 내보내기가 콘솔과 같은 값을 읽게 한다 — 지금 `"error", err` 는 `{}` 로 저장된다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `internal/observability/logs.go` 의 `persistentHandler.Handle` 은 `attr.Value.Any()` 를 그대로 `map[string]any` 에 담아 `encoding/json` 으로 굳히는데, `slog` 의 JSONHandler 는 error 값을 `err.Error()` 문자열로 적는 특수 처리를 한다. 그래서 **같은 한 레코드**가 콘솔에는 `"error":"상위: 연결 거부"` 로, `system_logs.attributes` 에는 `"error":{}` 로 들어간다(아래 재현 참고). 프로덕션에서 `"error", err` 로 로그를 남기는 자리가 28곳이고(`rg '"error", err' cmd internal` 기준) 관리자 콘솔 로그 화면과 CSV 내보내기(`internal/httpapi/platform.go:233`)는 이 `attributes` 를 그대로 보여 주므로, 장애 원인이 적힌 유일한 영구 기록이 빈 객체다. 고치면 콘솔에서 보이던 원인이 DB 기록·감사 파일에도 같은 값으로 남는다.

- 재현(정찰이 실제로 돌려 확인함 — 프로덕션 타입 `persistentHandler` + 실제 `slog.New` 경로):
  ```
  message="request failed" attributes={"dur":3000000000,"error":{},"path":"/api/x"}
  console={"level":"ERROR","msg":"request failed","error":"상위: 연결 거부","path":"/api/x","dur":3000000000}
  message="grouped"        attributes={"g":[{"Key":"a","Value":{}}]}
  console={"level":"INFO","msg":"grouped","g":{"a":1}}
  ```
  (`logger.Error("request failed", "error", fmt.Errorf("상위: %w", errors.New("연결 거부")), ...)` 와 `logger.Info("grouped", slog.Group("g","a",1))` 두 줄. 정찰은 임시 테스트 파일로 확인한 뒤 지웠다 — 작업 트리는 깨끗하다.)

- 수용 기준:
  1) `"error", err` 로 남긴 로그의 `persistedRecord.attributes` 를 `json.Marshal` 한 결과가 `{"error":"상위: 연결 거부"}` 처럼 콘솔 JSONHandler 가 적은 것과 같은 문자열이다. `json.Marshaler` 를 구현한 error 는 지금처럼 그 Marshaler 결과를 쓴다(slog 과 같은 규칙).
  2) `slog.Group("g","a",1)` 은 `{"g":{"a":1}}` 중첩 맵으로 저장되고(지금은 `[{"Key":..,"Value":{}}]`), `slog.LogValuer` 는 `Resolve()` 한 값으로 저장된다. 문자열·숫자·bool·time·duration 등 지금 올바르게 저장되는 종류의 결과는 바뀌지 않는다(특히 `dur:3000000000` 그대로, `trace_id` 문자열 추출 그대로 동작).
  3) 테스트가 "콘솔이 적은 값과 DB 에 담길 값이 같다" 를 **한 레코드에서** 증명한다: 같은 `logger.Error(...)` 한 번으로 콘솔 버퍼의 JSON 과 큐에 담긴 `attributes` 를 둘 다 받아 키별로 비교한다(error·group·LogValuer·문자열·숫자·duration 표). 손으로 만든 가짜 handler 를 쓰지 말고 `persistentHandler` 자신을 `slog.New` 에 꽂을 것.

- 건드릴 파일 (프로덕션 1개):
  - `internal/observability/logs.go:Handle` — `record.Attrs(...)` 안에서 `attributes[attr.Key] = attr.Value.Any()` 대신 새 변환 함수(예: `attributeValue(slog.Value) any`)를 쓴다. 변환은 `value.Resolve()` 먼저 하고 Kind 로 가른다: `slog.KindGroup` → 자식 attr 로 중첩 `map[string]any`, `slog.KindAny` 이면서 `json.Marshaler` 가 아닌 `error` → `err.Error()`, 그 밖에는 `Any()` 그대로. 왜 그렇게 하는지(콘솔 JSONHandler 와 같은 규칙을 쓴다)를 저장소 관례대로 한국어 주석으로 남길 것.
  - `internal/observability/logs.go:NewLogger` — (선택, 테스트를 위해 권장) 생성을 `newPersistentHandler(console, pool)` 같은 작은 함수로 빼고 `NewLogger` 가 그것을 쓴 뒤 `go handler.run()` 을 돌린다. 테스트는 `run()` 없이 같은 생성 경로를 쓴다(`pool=nil` 로 `run()` 을 돌리면 `h.pool.Exec` 가 nil 역참조로 패닉한다 — 정찰 확인).
  - `internal/observability/logs_test.go` — 신규. 패키지 내부 테스트로 `slog.New(handler)` → 로그 1~2회 → `close(handler.queue)` 후 큐에서 `persistedRecord` 를 꺼내 `json.Marshal(record.attributes)` 를 콘솔 버퍼의 같은 키와 비교. 큐는 2048 버퍼라 `run()` 없이도 레코드가 남는다(정찰 확인).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/observability -run TestPersisted -v` (고치기 전 빨강 → 후 초록, 변환 한 줄 되돌리면 다시 빨강인지까지 확인)
  - `go test ./internal/observability -race -count=1`
  - `go test ./...` (정찰은 이번에 `./internal/observability` 만 돌렸다 — 전체는 미실행)
  - `go vet ./...` · `go build ./...` · `gofmt -l ./cmd ./internal ./pkg` (출력 없어야 함)
  - `./scripts/check-release-docs.sh` · `./scripts/check-commit-identities.sh HEAD`

- 위험과 피할 것:
  - **DB·마이그레이션은 건드리지 말 것.** `system_logs` 스키마는 그대로다(`attributes` 는 이미 JSON). 옛 행의 `{}` 를 고치는 백필은 범위 밖.
  - `Store.List`/`ListRange`/`Stream`/`PurgeBefore` 와 `logFilter.where()` 의 SQL, `internal/httpapi/platform.go` 의 CSV 내보내기는 손대지 말 것 — 이번 변경으로 값만 제대로 실린다.
  - **새로 로깅하는 값을 늘리지 말 것.** 이미 레코드에 들어와 콘솔(stdout)로 나가는 값을 같은 규칙으로 적는 것뿐이다. 운영자 규칙(감사에 스크럽 이전 원문을 넘기지 말 것)에 걸리지 않으려면 호출자가 넘기지 않은 것을 만들어 넣지 말 것.
  - `WithAttrs`/`WithGroup` 가 handler 수준 속성을 DB 행에 싣지 않는 것(=`logger.With("trace_id",...)` 가 DB 에서 사라짐)도 정찰이 같은 재현으로 확인했지만, 지금 프로덕션에 `logger.With(` 호출자가 없다(`rg '\.With\(' cmd internal` 결과 로거 쪽 무소득 — `cmd/api/main.go:54` 의 `NewLogger` 뿐). **이번 회차에서는 손대지 말고** 아이디어로 남길 것 — 프로덕션에서 관찰되지 않는 변경은 증명할 자리가 없다.
  - 테스트에서 `run()` 을 돌리지 말 것(nil pool 패닉). `closeFn` 을 두 번 부르는 것도 금지(`once` 는 공유된다).
  - auth·apikey·migrations·.github/workflows·mail·automation 은 이번 과제와 무관하다 — 열지 말 것. 문서·PDF 재생성도 불필요(사용자에게 보이는 규칙 변경이 아니다).

- 차선 후보: cron 필드가 `+5`·`-0` 같은 부호 붙은 수를 받아들인다 — `internal/automation/schedule.go` 의 `cronValue` 가 `strconv.Atoi` 로 부호를 허용해 `"+5 0 * * *"` 가 매일 00:05 로 돈다(이전 정찰 확인, 이번 미확인). 저장된 일정을 깨뜨릴 수 있어 단독 과제로만, 거절 문구는 기존 `ErrInvalid` 래핑을 유지할 것.

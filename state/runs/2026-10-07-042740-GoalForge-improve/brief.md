- 과제: `storage prune --older-than` 가 비유한 수·오버플로 기간을 조용히 삼키지 않게 한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `cmd/goalforge/main.go:dayDuration` 은 `d`/`w` 접미사를 떼고 `strconv.ParseFloat` 한 값을 범위 검사 없이 `time.Duration(count * float64(unit))` 로 변환한다. `ParseFloat` 은 `Inf`/`+Inf`/`NaN` 과 `1e300` 같은 값을 모두 성공으로 읽고, float→int64 변환이 범위를 벗어날 때의 결과는 Go 명세가 구현에 맡긴 값이다. 이 함수의 유일한 호출부(`storagePrune`)는 감사 기록을 **지우는** 명령의 경계 시각을 정하므로, 읽을 수 없어야 할 입력이 어떤 숫자로든 통과하면 지우는 범위가 사용자가 쓴 값과 무관해진다.
- 수용 기준:
  1) `dayDuration("Infd")`, `dayDuration("+Infd")`, `dayDuration("-Infd")`, `dayDuration("NaNd")`, `dayDuration("NaNw")` 가 모두 오류를 반환한다(값이 아니라 오류). 오류 문구는 기존 `기간 %q 를 읽을 수 없습니다 (예: 30d, 12w, 720h)` 계열을 따른다.
  2) `time.Duration` 으로 표현할 수 없는 크기(예: `1e10d`, `1e300w`)도 오류로 거절된다 — 조용히 다른 수가 되지 않는다.
  3) 기존 계약은 하나도 바뀌지 않는다: `cmd/goalforge/duration_test.go` 의 `TestDurationAcceptsDaysAndWeeks`(`30d`,`1d`,`12w`,`720h`,`90m`,`1.5d`) 와 `TestUnreadableDurationSaysWhatItAccepts`(`""`,`soon`,`30 days`,`d`) 가 그대로 통과한다. 음수(`-5d`)·0(`0d`) 의 처리는 지금처럼 `dayDuration` 이 아니라 `storagePrune` 의 `window <= 0` 이 계속 맡는다(여기서 바꾸지 말 것).
  4) 테스트는 "거절된다" 를 증명하는 것으로 충분하지 않다 — 비유한/오버플로 입력이 **거절되지 않았을 때 어떤 값이 되는지** 를 먼저 측정해 적고, 수정 후 그 값이 더는 나오지 않음을 보인다.
- **먼저 측정할 것 (이번 정찰에서 미확인)**: 이 환경에서 `go run`/임시 파일 실행이 승인되지 않아 **변환 결과를 실제로 측정하지 못했다**. 구현자는 프로덕션 코드를 고치기 전에 먼저 테스트를 하나 써서 `dayDuration("Infd")`·`"NaNd"`·`"1e10d"` 의 **반환값과 `d > 0` 여부**를 출력해 보고(`go test ./cmd/goalforge -run TestDuration -v -count=1`), 그 결과에 따라 갈라라:
  - **(A) 양수 또는 0 이 아닌 엉뚱한 값이 나온다** → 프로덕션 결함이다. `dayDuration` 안에서 `math.IsInf(count, 0) || math.IsNaN(count)` 와 범위 초과(`math.Abs(count)*float64(unit) > math.MaxInt64` 또는 동등한 검사)를 거절한다. 파일 1개 + 테스트 1개.
  - **(B) 모두 0 이하가 되어 `storagePrune` 의 `window <= 0` 이 이미 막는다** → 프로덕션 코드를 **고치지 말고**, 이 거절이 우연이 아니라 계약임을 `duration_test.go` 에 고정하는 테스트만 추가한다. 이때 테스트는 `dayDuration` 단위가 아니라 `storagePrune` 이 `--older-than Infd` 를 오류로 끝내는 것까지 보여야 한다(아래 "건드릴 파일" 의 e2e 경로). 2026-10-05 회차가 같은 형태(프로덕션 변경 0, 테스트만)로 채택됐으니 B 를 축소된 결과로 보지 말 것.
- 건드릴 파일 (최대 2개):
  - `cmd/goalforge/duration_test.go` — `TestDurationRefusesNonFiniteWindows` 추가. 기존 두 테스트 함수는 건드리지 않는다.
  - `cmd/goalforge/main.go:dayDuration` (4373 행, 주석은 4367 행부터) — **(A) 인 경우에만** 범위·비유한 검사 추가. `storagePrune`(4299 행 부근)의 `window <= 0` 과 `--older-than` 기본값 `"30d"`, 플래그 설명 문구는 바꾸지 말 것.
  - 분기 (B) 에서 CLI 경로까지 보이려면 기존 CLI 테스트 하네스를 쓸 것: `cmd/goalforge/main_e2e_test.go:19 runCLI` / `:28 runCLIWithError` (이번 정찰에서 존재 확인). **새 하네스를 만들지 말고 기존 것을 재사용**하고, 전역 stdout 을 바꾸므로 `t.Parallel()` 을 쓰지 말 것. `storagePrune` 은 열린 `*store.Store` 를 받으므로 단위 호출이 아니라 CLI 경로로 가야 DB 가 준비된다.
- 검증 명령:
  - `go test ./cmd/goalforge -run Duration -v -count=1`  (측정 단계·좁은 확인. 이번 정찰에서 실제로 돌려 **ok 0.004s** 확인. 주의: `-run TestDuration` 으로 쓰면 `TestUnreadableDurationSaysWhatItAccepts` 가 걸러져 한 개만 돈다 — 실측함)
  - `go test ./cmd/goalforge -count=1`  (이 저장소 기준 약 2~7초)
  - `go test ./... -count=1`  (전체. `internal/store/sqlite` 약 45초, `internal/observer` 약 19초로 느리다. push 가 막힌 환경에서는 테스트 4건이 skip 되는 것이 정상이며 그것을 실패로 읽지 말 것)
  - `go vet ./...` / `go build ./...` / `gofmt -l ./cmd ./internal`(무출력) / `go mod tidy` 후 `go.mod`·`go.sum` 드리프트 없음
- 위험과 피할 것:
  - **다른 기간 파서를 건드리지 말 것.** `internal/notify/suppress.go` 와 `internal/api/setup.go` 에도 기간 파서가 있고, `internal/model` 에 `ParseWindow` 는 **존재하지 않는다**(여섯 회차째 미머지). 운영자 지시대로 같은 종류의 파서를 통합하려 하지 말고 이번 과제는 `dayDuration` 한 곳만 좁혀 고친다.
  - `storagePrune` 의 삭제 동작·`s.Prune` 의 SQL·`--apply`/`--vacuum` 의미를 건드리지 말 것. 이 과제는 입력 거절만 다룬다.
  - 보호 경로(auth·migrations·.github/workflows·`internal/gitops`·`internal/observer` 승인 봉투)는 건드릴 일이 없다.
  - 에러 문구를 새로 만들어 늘리지 말 것 — 기존 두 문구 중 하나를 재사용해 사용자가 보는 어휘를 유지한다.
  - 소스 문자열 검사나 손으로 주입한 가짜 의존성을 근거로 삼지 말 것. 실제 `dayDuration` 호출과(가능하면) 실제 CLI 경로로 증명한다.
- 차선 후보: **같은 `CheckReadiness` 결과를 네 소비자가 같은 결론으로 번역하는지 end-to-end 로 고정한다** (가치 2 / 위험 1 / 작업량 S). `internal/diagnostics` 의 한 결과를 `cmd/goalforge` doctor·`internal/app/plan.go`(FAIL·WARN 을 모두 WARN 으로 올리고 접두사로만 구분)·`internal/mcp/tools.go:590 부근`(FAIL 개수로만 ready 판정)·`internal/api/setup.go` 가 서로 다르게 번역한다. 2026-10-06 변경으로 '전부 미측정' 설정에서 proof kind 가 WARN 으로 바뀌었으므로 번역 차이가 실제로 드러난다. **통합하려 하지 말고** 같은 설정이 두 표면에서 같은 결론(ready/not ready)을 주는지만 테스트로 고정할 것. 이 네 소비자의 코드는 이번 정찰에서 **직접 열어보지 못했다(미확인)** — 행 번호는 이전 회차 기록에서 옮긴 것이니 먼저 확인하라.

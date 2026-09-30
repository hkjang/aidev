# 과제서 — 2026-10-01 (base: main@0c9e799)

- **과제**: 무인 스윕이 "봉투는 켜져 있는데 아무것도 승인되지 않는" 프로젝트를 조용히 지나치는 것을 고친다 (가치 4 / 위험 1 / 작업량 S)

- **왜**: `internal/observer/tick.go:137` 이 `AutoApprove` 의 결과에서 `execution.Approved` 만 꺼내고 `execution.Detail`·`execution.Refused` 를 버린다(`AutoApproveMerges` 도 `merges.Approved` 만). 그래서 봉투가 켜져 있고 공급된 findings 도 있는데 하루 한도를 다 썼거나 전부 `admits()` 에 거절된 프로젝트는 `Ran=false, Approved=[], Merged=[], Note="", Err=nil` 이 되고, `TickResult.Acted()`(tick.go:37) 가 false 를 돌려 `sweepStandards`(cmd/goalforge/main.go:1122) 가 **한 줄도 찍지 않는다**. 운영자에게는 아무 일도 없는 프로젝트와 매 틱 영원히 거절당하는 프로젝트가 똑같이 침묵으로 보인다. 5b83c30 커밋 메시지는 "한도에 걸린 프로젝트는 실패가 아니라 설정대로 동작한 것이라 따로 보고한다" 고 적었는데, 그 처리는 기준 패스(`pass.Detail` → `tick.Note`, tick.go:119)에만 있고 자동 승인 쪽에는 없다. 고치면 무인 루프에서 막혀 있는 프로젝트가 보이게 된다 — 기록만 하고 아무도 읽지 않는 상태의 반대쪽 경우다.

- **수용 기준**:
  1) 봉투가 켜져 있고(`Enabled=true`) 공급된 findings 가 있는데 하루 한도를 다 쓴 프로젝트에 대해 `Tick` 결과의 해당 `ProjectTick` 이 그 사유를 담고, `TickResult.Acted()` 가 true 가 되며, `sweepStandards` 가 그 프로젝트에 대해 한 줄을 찍는다.
  2) 봉투가 켜져 있는데 공급된 findings 가 전부 거절된 경우(예: 정산 게이트 없음)도 같은 방식으로 보인다. 거절 사유 표시는 **결정적**이어야 한다 — `AutoDecisions.Refused` 는 map 이므로 순서에 의존하면 테스트가 흔들린다(작업 항목 ID 로 정렬해 대표 하나를 고르거나 건수만 표시).
  3) **저장된 봉투가 없거나 꺼져 있는 프로젝트는 여전히 아무 줄도 찍지 않는다.** `AutoApprove` 는 이 경우에도 `Detail="자동 승인이 켜져 있지 않습니다"` 를 채우므로, 그것을 그대로 보고하면 팩만 고정한 모든 프로젝트가 매 틱 한 줄씩 찍고 5b83c30 이 일부러 없앤 소음이 돌아온다. 기존 `tick_test.go:190` 대역(`envelope` 없음 / `Enabled=false`) 이 계속 통과해야 한다.
  4) 조용해야 할 경우가 조용하다는 것을 **테스트가 직접 증명**한다: 봉투 없음 → 출력 대상 아님, 봉투 켜짐+한도 소진 → 출력 대상임. 두 방향 모두 단정할 것(한쪽만 보면 항상 찍는 구현도 통과한다).

- **건드릴 파일** (프로덕션 2개):
  - `internal/observer/tick.go`
    - `ProjectTick`(:16) — `Note` 를 덮어쓰지 말고 별도 필드(예: `AutonomyNote string`)를 더한다. `Note` 의 주석은 "nothing happened" 의 사유라고 못박혀 있고 기준 패스가 이미 쓰고 있다. 같은 필드에 두 의미를 넣으면 갈라진다.
    - `Acted()`(:37) — 새 필드가 채워진 경우를 true 에 포함.
    - `tickProject`(:123~150) — `execution := AutoApprove(...)` 뒤에서 **`policy.Enabled` 플래그로** 판정한다(`Detail` 문자열 비교로 켜짐/꺼짐을 판정하지 말 것 — 운영자 규칙). `policy.Enabled && len(execution.Approved)==0` 일 때만 `execution.Detail` 또는 `execution.Refused` 요약을 새 필드에 담는다. `AutoMerge` 켜진 경우 `merges` 쪽도 같은 규칙으로.
    - 참고: :166 에 `var _ = time.Now` 가 남아 있다. `time` 이 실제로 쓰이게 되면 지울 수 있지만 이번 과제와 무관하면 건드리지 말 것.
  - `cmd/goalforge/main.go:sweepStandards`(:1122~1149) — `project.Approved`/`Merged` 출력 옆에 새 필드 한 줄을 더한다. `Note` 는 지금 **어디서도 찍히지 않는다**(확인함) — 이번에 `Note` 까지 찍게 만들면 "예정 아님" 이 매 틱 출력되어 기준 3 을 깬다. 손대지 말 것.
  - `internal/observer/tick_test.go` — 기준 1·2·4 의 테스트. 기존 픽스처 `tickFixture`/`tickProjectIn`/`enrol`(:191 부근)과 `db.SetGateSettles`, `db.SaveAutonomyConfig` 를 그대로 쓴다. 한도 소진은 `DailyLimit:1` 로 저장하고 `Tick` 을 두 번 돌려 실제 `AutoApprovalsSince` 경로를 지나게 만든다(대역 주입 금지 — 실제 `*store.Store` 와 실제 `AutoApprove` 로).
  - (선택) `cmd/goalforge/` 쪽 출력 단정이 어려우면 프로덕션 판정은 `tick.go` 에 두고 `main.go` 는 그 필드를 그대로 찍는 한 줄로만 유지한다. 분기 로직을 `main.go` 에 두면 테스트가 닿지 않는다.

- **검증 명령** (이 저장소에서 실제로 도는 것, 전부 실측 확인):
  - `go test ./internal/observer/ -count=1 -v -run TestTick` — 패키지 전체는 약 13.4s
  - `go test ./internal/observer/ ./cmd/goalforge/ -count=1`
  - `go build ./...`
  - `go vet ./...`
  - `gofmt -l ./cmd ./internal` (무출력이어야 함)
  - `go test ./... -count=1` — **기준선은 exit 0 이다. 이번 회차에 실측했다**(약 60s; `internal/store/sqlite` 25.9s, `internal/observer` 13.4s, `internal/api` 8.7s, `internal/app` 8.5s). 이전 프로필의 "6건 실패" 는 옛 얘기다 — 새로 깨진 것이 하나라도 있으면 이번 변경 탓이다.

- **위험과 피할 것**:
  - **소음이 진짜 위험이다.** 이 과제는 "안 보이는 것을 보이게" 인데, 과하게 하면 5b83c30 이 의도적으로 없앤 프로젝트당 매 틱 한 줄이 돌아오고 그게 바로 그 커밋이 고친 문제다. 기준 3 이 이 과제의 진짜 합격선이다.
  - `AutoDecisions.Refused` 는 `map[string]string` 이다. 순회 순서에 의존한 출력·단정은 무작위로 깨진다.
  - `AutoApprove`/`admits`(`internal/observer/autonomy.go`)의 **판정 로직은 건드리지 말 것.** 봉투가 무엇을 허용하는지는 `AutoApprove` 한 곳에만 있어야 한다고 5b83c30 이 명시했고(같은 검사를 스윕에 복제했다가 지운 이력이 있다), 이번 과제는 보고만 바꾼다.
  - `internal/policy/*`, `internal/gitops/commit.go`, `internal/store/sqlite/store.go` 의 스키마, `.github/workflows/*` 는 이번 과제에서 열 이유가 없다. 마이그레이션 불필요(새 컬럼 없음).
  - `store.AutonomyConfig` → `observer.AutonomyPolicy` 8필드 매핑이 `tick.go:133` 과 `cmd/goalforge/standards.go:402` 두 곳에 복제되어 있다. 지금은 양쪽이 8필드 모두 옮기므로 결함은 아니다(확인함). **이번 회차에 통합하지 말 것** — 과제가 커지고 봉투 경로를 건드린다.

- **차선 후보**: MCP `activity_report` 가 광고하는 기간 단위를 실제로 받아들이게 한다 — `internal/mcp/tools.go:73` 이 `"Duration such as 24h or 7d (default 24h)."` 라고 광고하는데 :657 은 `time.ParseDuration(args.Since)` 라서 `7d` 를 반드시 거절한다(main@0c9e799 에서 두 줄 모두 확인함). 일/주를 아는 파서는 `cmd/goalforge/main.go:4167 dayDuration` 에 이미 있고 `cmd/goalforge/duration_test.go` 가 그것을 테스트한다.
  **경고(미확인)**: 2026-09-29 회차가 정확히 이 문제를 `internal/model/window.go` 의 `ParseWindow` 로 뽑아 3개 표면을 통일하는 방식으로 고쳤고 구현 판정은 "채택" 이었는데, **그 변경은 main@0c9e799 에 없다**(`internal/model/` 에는 `model.go` 뿐이고 `ParseWindow` 는 저장소 어디에도 없다). 사람이 반려했는지 그냥 유실됐는지 이 세션에서는 확인할 수 없었다(`gh pr list` 가 샌드박스에서 막혔다). 운영자 규칙상 반려된 접근의 재제출은 금지이므로 **새 공용 패키지로 뽑는 그 방식은 다시 쓰지 말고**, 이 후보를 고를 경우 MCP 처리기 한 곳만 고치는 최소 변경으로 하고 `internal/notify/suppress.go:52`·`internal/api/setup.go:289,304` 는 폴백 의미가 달라 손대지 말 것.

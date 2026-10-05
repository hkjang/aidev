- 과제: doctor 가 게이트가 없는 완료 조건까지 "알맞은 게이트로 측정됩니다" 로 세지 않게 한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/diagnostics/readiness.go:kindChecks` 는 종류를 요구하는 조건마다 `demanded++` 를 먼저 하고 그 다음에 `gateKinds[criterion]` 의 `covered` 를 보고 `continue` 하므로, 게이트가 아예 없는 조건도 "검증 종류를 요구하는 완료 조건 N개가 알맞은 게이트로 측정됩니다" 라는 OK 집계에 들어간다. 종류를 요구하는 조건이 전부 미측정이면 `demanded > 0` 이 성립해 그 OK 를 반환하고 뒤의 behavioural warn 분기까지 건너뛰므로, doctor 가 증거 경로가 하나도 없는 설정을 proof kind OK 로 설명한다.
- 수용 기준:
  1) 종류를 요구하는 조건에 같은 이름의 게이트가 없으면 그 조건은 proof kind 의 OK 집계에 포함되지 않는다. (coverage FAIL 은 그대로 유지 — 중복 보고를 다시 만들지 말 것)
  2) 종류를 요구하는 조건이 전부 미측정이면 proof kind 는 "N개가 알맞은 게이트로 측정됩니다" OK 를 내지 않는다. 그 경우 동작 확인 게이트 유무 판단(`behavioural` 분기)이 정상적으로 평가되어야 한다.
  3) 측정되는 조건과 미측정 조건이 섞여 있으면 OK 집계 수는 측정되는 조건 수만 센다 (예: 2개 요구 중 1개만 게이트 있음 → "1개").
  4) 테스트가 증명할 것: 실제 `CheckReadiness` 호출 결과 `[]Check` 에서 `coverage` FAIL 과 `proof kind` 의 레벨·Detail 조합을 함께 단정한다. 손으로 만든 대역 없이 `ReadinessInput` 값만으로 프로덕션 함수를 지난다.
- 건드릴 파일:
  - `internal/diagnostics/readiness.go:kindChecks` — `demanded++` 를 `covered` 확인 뒤로 옮긴다(= 측정되는 조건만 센다). 미측정 조건은 coverage 체크가 이미 보고하므로 여기서 또 말하지 않는다는 기존 주석의 의도를 유지한다.
  - `internal/diagnostics/readiness_test.go` — 위 세 경우(전부 미측정 / 혼합 / 기존 정상)를 `CheckReadiness` 로 단정하는 테스트 추가. 기존 테스트 6개를 직접 읽어 확인했다: `TestReadinessCatchesUnmeasurableCriteria`(`CriterionKinds` 를 주지 않음), `TestReadinessRejectsProofOfTheWrongKind`, `TestReadinessRejectsAJudgementBehindAnObjectiveCriterion`, `TestReadinessWarnsWhenNothingChecksBehaviour`, `TestReadinessBlocksConfigurationsThatCannotFinish`, `TestReadinessWarnsWithoutBlocking` — **어느 것도 종류를 요구하면서 게이트가 없는 조합을 쓰지 않으므로 기존 기대값은 바뀌지 않을 것으로 본다**(실행으로는 미확인). 헬퍼 `levelFor(checks, name)` 가 이미 있고 OK 집계 수를 보려면 `Detail` 을 직접 봐야 한다.
  - 그 외 프로덕션 파일은 건드리지 않는다. 프로덕션 1개 + 테스트 1개로 끝나야 한다.
- 검증 명령:
  - `go test ./internal/diagnostics -count=1` (빠름)
  - `go test ./internal/diagnostics ./internal/policy ./cmd/goalforge -count=1` (CLI doctor 경로 동반 확인; CLI 약 7초)
  - `go test ./... -count=1` / `go vet ./...` / `go build ./...` / `gofmt -l ./cmd ./internal`(무출력) / `go mod tidy` 드리프트 없음
  - 인과 고정: 수정을 되돌리면 새 테스트가 다시 실패하는 것을 한 번 확인하고 즉시 원복.
- 위험과 피할 것:
  - `policy.EvidenceSatisfies` 와 `offLadderKinds`(2026-10-04 회차에서 고친 자리)를 건드리지 말 것. 이번 과제는 집계 순서만 바꾸는 것이며 무엇이 무엇을 충족하는지의 판정은 그대로 둔다.
  - 미측정 조건을 proof kind 에서 FAIL 로 새로 보고하지 말 것 — `criteria coverage` 가 이미 FAIL 로 말하고 있어 같은 결함이 두 줄이 되면 코드 주석이 경고한 "finding 이 묻히는" 문제가 생긴다.
  - `demanded` 를 옮기면 "전부 미측정" 경우에 `behavioural` warn 이 새로 등장한다. 그것이 의도한 결과임을 테스트로 단정하고, 기존 doctor 출력 스냅샷류 테스트가 있으면 함께 갱신할 것.
  - 보호 경로(auth, store 마이그레이션, gitops commit/merge/push, `.github/workflows`)는 건드리지 않는다. 이 과제는 그 어디도 지나지 않는다.
  - `internal/model/window.go`(ParseWindow)는 main 에 없다. 그것을 전제로 하는 코드를 쓰지 말 것.
  - 소비자 4곳을 직접 열어 확인했으므로 이 수정이 차단 판정을 바꾸지 않는다: `cmd/goalforge/main.go:966`(doctor 출력), `internal/api/setup.go:63`, `internal/app/plan.go:192`(OK 가 아닌 모든 체크를 일괄 `WARN` 으로 올리며 FAIL 만 접두사가 다르다 — WARN 추가는 가산적), `internal/mcp/tools.go:590`(`blocking`/`ready` 를 **FAIL 개수로만** 계산). 이 과제는 FAIL 을 하나도 새로 만들지 않으므로 `ready` 가 뒤집히지 않는다. 그래도 FAIL 을 새로 추가하는 방향으로 설계를 바꾸지 말 것 — MCP `ready` 와 plan 접두사가 함께 변한다.
  - 프로덕션 도달성 확인: `internal/store/sqlite/readiness.go:ReadinessInput` 은 `Criteria`/`CriterionKinds` 를 `goal.Criteria` 의 `Type`·`RequiredKind` 에서, `Gates` 는 별도의 `ListGates` 에서 채운다. 두 출처가 독립이므로 "종류를 요구하는데 게이트가 없는" 상태는 실제 프로젝트에서 도달 가능하다(실제 DB 로 재현해 보지는 않았다 — 미확인).
- 차선 후보: retention `dayDuration` 의 비유한 수·오버플로 거절 (`cmd/goalforge/main.go:dayDuration` — `strconv.ParseFloat` 결과를 범위 검사 없이 `time.Duration` 으로 변환). 단, 호출부가 이미 `<= 0` 로 거절하는지 먼저 실행으로 확인하고, 거절된다면 프로덕션 수정 없이 계약 테스트만 추가하는 쪽으로 좁힐 것.

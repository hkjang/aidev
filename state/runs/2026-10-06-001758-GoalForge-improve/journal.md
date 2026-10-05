# 회차 노트 2026-10-06-001758-GoalForge-improve — GoalForge
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:18] base pinned — main@37bbc7f
- [러너 00:18] autonomy release — 

## 정찰 노트
- 보류 목록 중 유일하게 프로덕션 결함을 코드로 확정할 수 있고 파일 2개로 끝나는 것을 골랐다: `readiness.go:126` 의 `demanded++` 가 covered 확인보다 앞서 proof kind OK 집계를 부풀린다. DailyLimit·승인 철회·헬퍼 정리는 설계가 선행하는 M 이라 제쳤고, ParseWindow 통일은 선행 PR 미머지 + 계약 상이로 `rejected` 로 내렸다.
- 확신 없는 곳: 실제 SQLite 프로젝트로 "종류를 요구하는데 게이트 없음" 상태를 런타임 재현하지는 않았다(`ReadinessInput` 이 criteria 와 gates 를 독립 출처에서 채운다는 코드 근거만). 기존 readiness 테스트 6개가 안 깨질 것이라는 판단도 읽기 기반이며 실행으로 확인하지 않았다.
- 구현자가 조심할 것: 미측정 조건을 proof kind 에서 **FAIL 로 새로 보고하지 말 것**. `criteria coverage` 가 이미 FAIL 이고, FAIL 을 늘리면 `mcp/tools.go:590` 의 `ready` 와 `app/plan.go:192` 의 접두사가 함께 바뀐다. WARN 추가는 소비자 4곳 모두 가산적이라 안전하다.
- 기준선 실측: `go test ./internal/diagnostics ./internal/policy -count=1` exit 0 (각 0.02s). 전체 suite·vet·build 는 이번에 돌리지 않았다.
- [러너 00:21] scout done — doctor 가 게이트가 없는 완료 조건까지 "알맞은 게이트로 측정됩니다" 로 세지 않게 한다 (가치 3 / 위험 1 / 

## 구현 노트
- `kindChecks` 의 집계를 `covered` 확인 뒤로 옮겼다(변수명 `demanded`→`measured`). 게이트가 없는 조건은 측정되는 게 아니므로 proof kind OK 집계에서 빠지고, 종류를 요구하는 조건이 전부 미측정이면 OK 를 반환하며 조기 종료하던 경로가 없어져 `behavioural` 분기가 정상 평가된다. 프로덕션 1개 + 테스트 1개.
- 확신 없는 곳·검증 못 한 것: 실제 SQLite 프로젝트로 "종류를 요구하는데 게이트 없음" 상태를 **런타임 재현하지 않았다** — 정찰과 동일하게 `ReadinessInput` 이 criteria/gates 를 독립 출처에서 채운다는 코드 근거만 있다. 테스트는 `ReadinessInput` 값으로 `CheckReadiness` 를 직접 호출하며 CLI `doctor` 출력을 끝까지 돌려 눈으로 본 적은 없다(`cmd/goalforge` 패키지 테스트는 통과). 소비자 4곳(plan/mcp/api/CLI)이 새로 생기는 WARN 을 어떻게 보여주는지도 코드 읽기로만 판단했고 HTTP/MCP 응답을 실제로 비교하지 않았다. Windows/macOS 는 돌리지 않았다(이 변경에 OS 의존 경로는 없다).
- 일부러 하지 않은 것: 미측정 조건을 proof kind 에서 FAIL 로 보고하지 않았다 — `criteria coverage` 가 이미 FAIL 로 말하므로 중복 보고가 되고, FAIL 이 늘면 `mcp/tools.go:590` 의 `ready` 와 `app/plan.go:192` 의 접두사가 함께 바뀐다. `policy.EvidenceSatisfies`·`offLadderKinds` 는 손대지 않았다(무엇이 무엇을 충족하는지의 판정은 불변).
- 다음 역할이 조심할 것: 새 테스트 2개는 DB 도 git 도 필요 없고 `go test ./internal/diagnostics -count=1` 로 0.02s 에 돈다. 인과는 집계 순서만 되돌려 양방향 확인했다. doctor 출력 스냅샷류 테스트는 없다 — `"proof kind"`/`"알맞은 게이트로 측정"` 을 테스트 전체에서 검색해 두 파일만 나왔고, `internal/store/sqlite/evalfromfailure_test.go` 쪽은 산문 주석 일치일 뿐 `Criterion.RequiredKind` 보존을 단정하는 무관한 테스트다(diagnostics 의 Detail 문자열을 보지 않는다). 전체 suite 도 `go test ./... -count=1` exit 0, 실패·에러 줄 없음.
- [러너 00:25] brief accepted — 채택 — 지목한 두 줄(`demanded++` 가 `covered` 확인보다 앞)이 지금 코드와 정확히 일치했고, 수용 기준 4개·검증 명령·"FAIL �
- [러너 00:25] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인: readiness.go 만 HEAD~1 로 되돌려 새 테스트 2개가 원장의 '실패 재현' 과 **글자까지 같은 메시지**로 실패하는 것을 직접 재현했다(수정본으로는 통과). 테스트는 CheckReadiness 를 실제로 지나고 Detail 문자열을 단언한다.
- 확인: 구현자가 비워 둔 자리 — `sqlite/readiness.go:26-47` 이 criteria(RequiredKind)/gates 를 독립 출처에서 채우므로 미측정 상태는 도달 가능하고, 소비자 4곳(mcp:593·api:65·plan:197·main:968)은 모두 LevelFail 만 세므로 새 WARN 은 가산적이다. ready·접두사·종료코드 불변.
- 못 본 것: 실제 SQLite 프로젝트로 doctor 를 끝까지 돌린 출력, HTTP/MCP 응답 비교, Windows/macOS. 코드 읽기와 패키지 테스트로만 판단했다.
- 승인이어도 남는 우려(릴리즈): 종류를 요구하는 조건이 전부 미측정 + 동작 확인 게이트가 있으면 `proof kind` 줄이 **아예 사라진다**(이전엔 거짓 OK). 차단에는 영향 없으나 출력 변화라 릴리즈 노트에 적을 것. 또 `readiness.go:133` 의 `measured++` 가 `EvidenceSatisfies` 앞이라, 불일치를 치명적이지 않게 바꾸는 다음 변경은 같은 결함을 되살린다.
- 다음 회차: `go test ./... -count=1` 첫 실행에서 `internal/app/TestResumePausedValidatesCheckpointAndVerifies` 가 `saved [] current [verify]` 로 1회 실패했고 재실행 4회는 통과 — readiness 경로를 지나지 않는 기존 플레이크로 판단했다. CI 에서 재발하면 여기서 시작하라.
- [러너 00:31] review approved — 리뷰 승인 (risk=low)
- [러너 00:31] pr created — https://github.com/hkjang/goalforge/pull/81
- [러너 00:39] ci passed — 검사 5개 모두 success
- [러너 00:39] merge done — d348ffe
- [러너 00:50] release published — v0.53.0
- [러너 01:00] assets verified — v0.53.0 자산 7개 (이전 v0.52.0: 7)

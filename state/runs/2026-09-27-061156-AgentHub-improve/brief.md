- 과제: Pod 게이트웨이가 보낸 dlp.tool 보고의 문자열 길이에 상한을 두어 감사 details·로그가 Pod 가 정한 크기를 그대로 받지 않게 한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/api/dlp.go:reportDLPEvent` 는 Pod 가 보낸 `event.server`·`event.tool`·`event.direction` 과 각 finding 의 `class`·`label`·`action`·`sample` 을 길이 검사 없이 `details` 맵(→ audit_events.details JSON)과 `s.logger.Warn` 에 그대로 싣는다. 본문 상한은 `respond.go:99` 의 `http.MaxBytesReader(w, r.Body, 1<<20)` 뿐이라 한 건의 보고가 1MB 에 가까운 임의 문자열을 감사 행과 운영 로그에 넣을 수 있고, 같은 서버의 `mcp.tool_call` 은 이미 `mcp.go:216 mcpClientIDLimit = 200` 으로 `client` 를 룬 단위로 자르고 있어 두 감사 경로의 규칙이 엇갈린다. 게이트웨이 토큰은 에이전트 코드가 도는 Pod 안에 있으므로 이 문자열은 신뢰 경계 밖의 입력이며, `sample` 은 "마스킹된 예시" 라는 계약만 있고 컨트롤 플레인이 검증하지 않는다.
- 수용 기준:
  1) 1MB 짜리 `server`/`tool`/`direction` 과 1MB 짜리 `sample` 을 담은 보고를 프로덕션 라우터로 POST 했을 때, 응답 상태·`outcome` 분류는 지금과 같고 `audit_events.details` 에 저장된 각 문자열이 정해진 상한(룬 기준) 이하다.
  2) 상한보다 짧은 정상 보고의 `details` 는 한 글자도 바뀌지 않는다(기존 `TestAGatewaysFindingReachesTheTrail` 가 그대로 통과).
  3) 절단은 룬 경계에서 일어나 한글이 깨지지 않는다(멀티바이트 문자로 채운 입력으로 증명).
  4) `outcome` 은 절단 **전** 값으로 계산된다 — `dlp.Result.Outcome()` 이 `finding.Action == dlp.Redact` 문자열 비교에 의존하므로(`internal/dlp/dlp.go:234-247`), 자른 `Action` 으로 분류하면 판정이 바뀔 수 있다. 테스트가 이 순서를 고정해야 한다.
  5) `findings` 개수 상한(`maxReportedFindings = 32`)의 기존 동작은 그대로다.
- 건드릴 파일:
  - `internal/api/dlp.go:reportDLPEvent` (156-210행) — `outcome := result.Outcome()` 계산 **뒤에**, `details` 맵과 `s.logger.Warn` 에 넣는 문자열을 상한에 맞춰 자른다. `findings` 는 `maxReportedFindings` 로 자른 슬라이스를 복사한 뒤 각 원소의 문자열 필드를 자른다(원본 `result.Findings` 를 제자리에서 고치지 말 것 — `result` 는 `Outcome()`·`Reportable()` 이 이미 읽은 값이고 같은 슬라이스를 공유한다).
  - `internal/api/dlp.go` — 상한 상수와 절단 헬퍼를 이 파일에 새로 둔다(예: `maxReportedTextLen`, `clampReported`). `mcp.go:224 mcpCallDetails` 의 룬 절단 방식을 그대로 따르되 **`internal/dlp` 패키지는 건드리지 말 것**(런타임 base 이미지 소스라 BASE_VERSION 상향이 따라온다). `mcp.go` 의 기존 상수를 재사용해도 되지만 의미가 다른 값이므로 새 상수를 권한다.
  - `internal/api/dlpreport_live_test.go` — 기존 `gatewayDeployment(t)` 헬퍼(같은 파일 136행, DSN·암호화 키가 없으면 `t.Skip`)를 써서 과대 문자열 보고 케이스를 추가하고 `db.AuditTrail(ctx, store.AuditFilter{Action: "dlp.tool", ResourceID: runtime.AgentID, Limit: 10})` 로 실제 저장된 details 를 읽어 확인한다(기존 테스트 105·124·205행이 같은 방식).
  - (선택) DB 없이 도는 단위 테스트를 같은 패키지에 하나 더 둬서 룬 절단·상한 이하 무변경을 고정한다. 단위 테스트만으로 끝내지 말 것 — 프로덕션 라우터를 통과하는 live 테스트가 수용 기준 1·2 의 증거다.
- 검증 명령:
  - `go build ./... && go vet ./internal/api`
  - `go test ./internal/api ./cmd/runtime-proxy` (DSN 없으면 live 테스트는 skip — 이것만으로는 수용 기준 1 의 증거가 아니다)
  - DSN 이 있으면: `AGENTHUB_TEST_DSN=… AGENTHUB_ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64) go test -race -p 1 -run 'TestAGatewaysFinding|TestDLP|Gateway' ./internal/api`
  - 마지막에 `go test -race ./cmd/... ./internal/...` (CI 와 같은 명령)
  - 수정 **전** 코드에서 새 테스트가 옳은 이유로 실패하는 것(저장된 details 문자열이 상한을 넘는다)을 먼저 확인해 기록할 것.
  - 미확인: 정찰 시점 이 환경에는 `AGENTHUB_TEST_DSN` 이 없어 `internal/api` 의 live 테스트가 전부 skip 됐다. `go test ./internal/api ./cmd/runtime-proxy` 는 이번 회차에 실제로 돌려 통과했으나(1.6s), 그 통과는 DB 경로의 증명이 아니다.
- 위험과 피할 것:
  - `internal/dlp` 패키지와 `cmd/runtime-proxy` 는 손대지 말 것 — 런타임 base 이미지 소스이고 BASE_VERSION 상향이 따라온다. 상한은 컨트롤 플레인(`internal/api`) 쪽에서만 적용한다.
  - `internal/api/auth.go`·`mcpoauth.go`·`store` 마이그레이션·`.github/workflows` 는 건드리지 않는다. 이 과제는 그 어느 것도 필요하지 않다.
  - 교훈(2026-09-09): 감사에 넘기는 값은 식별자만 — 이 과제는 **새 값을 추가하는 것이 아니라 이미 넘어가는 값을 줄이는 것**이다. `sample` 을 아예 빼는 쪽으로 범위를 넓히지 말 것(기존 테스트와 DLP 화면 계약이 sample 을 읽는다. 감사 화면이 sample 을 읽는지는 미확인 — 빼는 변경은 이번 범위 밖).
  - 교훈(2026-09-09): `Result` 를 값으로 받는 함수의 복사 주의 — `dlpReport.result()` 는 `report.Event.Findings` **슬라이스를 공유**한다. 슬라이스 원소를 제자리에서 고치면 `input.Event` 쪽 값도 같이 바뀌므로, `s.logger.Warn` 에 넘기는 `input.Event.Server`/`Tool` 도 같은 절단값을 쓰도록 한 곳에서 계산해 두 경로가 같은 값을 보게 할 것(운영자 지침: 같은 값을 읽는 경로가 둘이면 둘을 함께 확인).
  - grep 으로 문자열이 있다는 것을 증거로 제출하지 말 것. 실제로 DB 에 저장된 details 를 읽어 보일 것.
  - 상한 값은 `mcpClientIDLimit = 200` 과 맞추되 `sample`·`tool` 은 사람이 읽어야 하므로 지나치게 짧게 잡지 말 것(200~512 룬 범위에서 고르고 상수 주석에 이유를 남길 것).
- 차선 후보: 복원 실패가 guide-shots 의 problems 요약 출력을 건너뛰게 하는 것을 고친다 (가치 2 / 위험 1 / 작업량 S). `web/scripts/guide-shots.mjs:130-144` — `withGuideSettings` 가 던지면 138행 `if (problems.length)` 요약 블록과 `process.exitCode = 1` 이 건너뛰어지고 운영자는 스택만 본다(복원 실패 시 `guide-settings-check.mjs:60` 의 `note('복원 …', false, …)` 가 이미 problems 에 쌓아 두었는데도). 요약을 `finally` 로 옮기거나 `withGuideSettings` 호출을 try/catch 로 감싸 요약을 찍은 뒤 원본 오류를 재전파하면 된다. 주의: `web/scripts/guide-settings-check.test.mjs` 하니스의 소스 슬라이스 끝 앵커가 `  if (problems.length)` 라, 이 줄을 옮기면 앵커도 함께 옮겨야 한다(프로필의 "검증 함정" 항목).

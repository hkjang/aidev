- 과제: 설정의 정수 한도를 저장값과 집행값이 같은 하나의 값으로 고정 (가치 4 / 위험 2 / 작업량 S)

- 왜: `validateSettings`(internal/app/settings.go:233)는 `ai`·`agents`·`security` 그룹의 숫자 8개를 `asInt`(:72, `int(x)` 절삭)로 **검사만** 하고 받은 값을 그대로 저장한다 — 형제 검증기들(`validateFindingOpsSettings`:59 는 `n != math.Trunc(n)` 로 거절, `validateScope`/`validatePolicy`/`validateContributionPoints` 는 검사한 int 를 `m[key] = n` 으로 되쓴다)과 달리 이 세 그룹만 소수를 **조용히 수락**한다. 그래서 `max_tool_calls: 40.9` 는 집행은 40(`agents_tools.go:45`,`:89` 의 `asInt`)인데 저장·표시·실행 스냅샷은 40.9 가 되어, 관리자 화면과 **에이전트 보고서의 "실행에 기록된 한도"**(`agent_reports.go:115-122` 가 `v.Limits[k]` 원값을 `reportText` 로 인쇄)가 실제로 집행되지 않은 한도를 적는다. 2026-10-07 회차가 `contribution_points` 에 적용해 머지된 "한 값으로 저장" 수정과 같은 계약·같은 모양이다.

- 도달 경로(코드로 확인함):
  1. `PUT /api/settings/agents {"max_tool_calls": 40.9}` → 핸들러(settings.go:150-200)가 기존 설정 맵 `v` 에 원값을 넣고 `validateSettings` 통과 후 **그 맵을 그대로** `json.Marshal(v)` 로 저장(:199).
  2. 실행 생성 `agents.go:293-296`: `limits := cloneMap(cfg)` + `limits["max_tokens"]=ai["max_tokens"]`·`context_window` → 소수가 **불변 실행 스냅샷**(`agent_runs.limits` jsonb)에 동결되고 `agentRun.Limits map[string]any`(agents.go:30, json `limits`)로 상세 API 에도 그대로 나간다.
  3. 집행 독자는 전부 `asInt`: `agents_tools.go:42`·45·89, `agents_llm.go:34`, `agents_runner.go:112`·233, `agents_control.go:167`, `ai.go:80`, `auth.go:106`(session_hours), `keys.go:71`(key_max_days).
  4. 원값 독자: `agent_reports.go:115-122`(오프라인 한글 보고서), `GET /api/settings`(:132-148, DB 원값 반환) → 관리자 폼.

- 수용 기준:
  1) `validateSettings` 가 통과시킨 뒤 `v["max_iterations"]`·`max_model_calls`·`max_tool_calls`·`timeout_minutes`(agents), `max_tokens`·`context_window`(ai), `session_hours`·`key_max_days`(security)는 **집행 독자의 `asInt` 결과와 같은 정수**다(예: 40.9 → 40, 15.9 → 15). 즉 저장되는 값과 집행되는 값이 하나다.
  2) **새 거절을 만들지 않는다.** 소수를 400 으로 바꾸지 말 것 — 이미 소수가 저장된 기존 행은 관리자 폼이 그룹 전체를 그대로 PUT 하므로(settings.tsx:405 `settingsGroupValues` → 그룹 값 원형 PUT) 거절로 바꾸면 그 그룹 저장이 전부 막힌다(2026-10-16 회차의 grandfathering 교훈과 같은 자리). 정규화는 다음 저장에서 기존 행도 조용히 고친다.
  3) 정수 입력의 동작·저장값·오류 문구는 한 글자도 바뀌지 않는다: 기존 범위 경계(agents 6~100 / 5~200 / 1~200 / 1~60, ai 1~262144 과 1024~262144 및 `max_tokens ≤ context_window`, security 1~168 / 1~365)와 세 한국어 오류 문구(settings.go:264,273,287)를 그대로 유지한다.
  4) 테스트가 증명할 것: (a) 세 그룹 각각 소수 입력이 통과한 뒤 맵에 남은 값이 `asInt` 와 **같은 정수 타입/값**이다 (b) 같은 소수에 대해 `reportDocument` 가 인쇄하는 한도 문자열(`agent_reports.go` 의 `reportText(v.Limits[k],200)` 경로)이 집행값과 일치한다 — 보고서가 집행되지 않은 수치를 적지 못한다 (c) 경계 수락/거절이 수정 전과 동일하다(6·100·262144·1023·168·365 와 그 ±1).
  5) `go vet ./...` 0, `gofmt -l` 변경 파일 빈 출력.

- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `internal/app/settings.go:262-288` — `validateSettings` 의 `case "ai"`·`case "agents"`·`case "security"`: 각 키를 `n := asInt(v[k])` 로 한 번 읽어 범위를 검사하고 **통과 시 `v[k] = n` 으로 되쓴다**. `ai` 는 현재 같은 `asInt` 를 5번 호출하므로 두 지역변수로 묶으면 읽기도 좋아진다(오류 문구·판정 순서는 보존). `asInt`(:72)·`defaultSettings`(:16)·핸들러·`validateFindingOpsSettings`·`validateInventorySettings` 는 손대지 않는다.
  - `internal/app/core_test.go:43 TestSettingsLimits` 에 사례를 더하거나 같은 패키지에 새 DB-free 테스트 추가 — `validateSettings` 는 DB 없이 직접 호출 가능(core_test.go:43-60, handoff_test.go:13, mcp_oauth_test.go:189 이 이미 그렇게 한다). 보고서 일치 단언은 `agent_reports.go` 의 인쇄 경로(`reportText`)를 실제로 지나게 쓸 것 — 소스 문자열 검사로 대신하지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```sh
  go test -run '^TestSettingsLimits$' -count=1 -v ./internal/app      # DB 불필요
  go test -run '^TestAgentReport' -count=1 -v ./internal/app          # 보고서 경로(일부 testApp 는 DSN 없으면 SKIP)
  go test -count=1 ./internal/app
  go vet ./...
  gofmt -l internal/app/settings.go internal/app/core_test.go
  go build ./cmd/hunter
  node scripts/verify-pentagi.mjs
  git diff --check
  ```
  웹 변경이 없으므로 `npm --prefix web ...` 와 `internal/webassets/dist` 재복사는 불필요하다. `HUNTER_TEST_DSN` 없으면 `testApp` 통합시험은 SKIP 이며 **SKIP 을 통과로 보고하지 말 것**. 먼저 배선(되쓰기 없는 상태)에서 새 테스트가 **실제로 실패**하는 red 를 받고, 되쓰기 한 줄을 다시 제거해 그 테스트만 실패하는 인과를 확정할 것.

- 위험과 피할 것:
  - **같이 건드리지 말 것**: `web/src/settings.tsx`. 2026-10-08 회차의 `integer: true` 변경은 **아직 main 에 없다**(`grep -n integer web/src/settings.tsx` 무출력, 그 파일의 마지막 커밋은 daff3bc). 미머지 PR 과 충돌하므로 이번 회차는 서버만 고친다.
  - `sla`·`risk`·`inventory` 는 이미 소수를 거절한다(`finding_ops.go:59`, `operations.go:16`). 그 규칙을 느슨하게 만들거나 거기에 정규화를 추가하지 말 것.
  - `security` 그룹은 auth 인접이다. `auth.go:106`·`keys.go:71` 은 수정하지 않는다 — 두 곳 다 이미 `asInt` 라 세션 길이·키 만료의 실제 동작은 이 변경으로 바뀌지 않는다(저장값만 집행값과 같아진다).
  - 되쓰기는 값 타입을 `float64`→`int` 로 바꾼다. 이 키들에 `.(float64)` 단정을 하는 독자는 없음을 확인했다(`internal/app` 전체에서 `.(float64)` 는 `agent_memory_engine.go:414`·477, `mcp_oauth.go:323` 뿐, 모두 무관). 새로 그런 단정을 만들지 말 것.
  - 거대한 소수(예: 1e300)는 `int(x)` 변환이 Go 사양상 정의되지 않는다. 지금도 범위 검사에서 걸러지므로 **새로 다루려 하지 말고** 기존 판정 순서를 유지할 것.
  - 보고서 문구·`reportText` 한도(200)·PDF 폰트 경로는 건드리지 않는다. 감사 details 에 설정 원값 문자열을 새로 넣지 말 것.
  - 마이그레이션·워크플로·`third_party/pentagi` 는 무관하다. 커밋 메시지는 영어 conventional commit.

- 차선 후보: 소프트웨어 구성(inventory) 그룹 `stale_after_days` 입력에 `integer` 선언 — **다만 `web/src/settings.tsx` 는 위 미머지 PR 과 같은 파일이므로, 1순위가 성립하지 않을 때만 그 파일을 열고 inventory 그룹 한 곳만 바꿀 것**. 3순위: `web/src/settings.tsx:333` 의 "Hunter 도구 7종" 과 실제 도구 수(AGENTS.md §6 은 8종: service_context·list_findings·request_scan·scan_result·record_candidate·remember·recall·search_reference — `internal/app/agents_tools.go` 에서 실제 등록 수를 먼저 셀 것)의 불일치 정정.

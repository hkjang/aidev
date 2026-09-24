# 과제서 (정찰, 2026-09-24, base main@54e94f3)

- **과제**: MCP 도구 스키마의 불리언 플래그 3개(`open`·`overdue`·`mine`)가 `"type":"string"` 으로 선언되어 진짜 JSON 불리언을 보내면 필터가 조용히 꺼지는 것을 고치기 (가치 4 / 위험 1 / 작업량 S)

- **왜**: `internal/mcp/server.go:675` 의 `voiceProps` 는 같은 도구(`list_customer_voices`) 안에서 `reviewPending` 은 `boolean("…")` 으로 선언하고 `boolArg` 로 읽는데, 바로 옆의 `open`·`overdue` 는 `str("true면 …")` 으로 선언하고 `:972` 에서 `strArg(a,"open")=="true"` 로 읽는다 — 같은 도구의 같은 성격의 플래그가 두 가지 타입을 요구한다. `:690` 의 `get_recommendations` `mine` 도 같다. `strArg` 은 `v,_ := args[key].(string)` 이라 클라이언트가 스키마의 `boolean` 형제를 보고 `{"open": true, "reviewPending": true}` 를 보내면 `reviewPending` 만 먹고 `open` 은 빈 문자열이 되어 **오류 없이 필터가 꺼진다**. 그 결과 해결·종결된 VOC 가 섞인 목록이 "미해결 건" 으로, 남의 추천이 "본인 추천" 으로 에이전트에게 전달된다. 고치면 세 플래그가 스키마대로 동작하고, 기존에 문자열 `"true"` 를 보내던 클라이언트도 그대로 돌아간다.

- **수용 기준**:
  1. 프로덕션 도구 목록(`(*Server).tools`, server.go:589)에서 `list_customer_voices` 의 `open`·`overdue` 와 `get_recommendations` 의 `mine` 의 `inputSchema.properties[*].type` 이 `"boolean"` 이다(같은 도구의 `reviewPending`·`unreadOnly` 와 동일).
  2. 세 플래그를 읽는 자리가 진짜 JSON 불리언 `true` 를 켜짐으로 읽는다. 그리고 **하위호환**: 옛 스키마를 따라 보내던 문자열 `"true"` 도 여전히 켜짐이고, `false`·`"false"`·`""`·키 없음·`1` 은 모두 꺼짐이다.
  3. 테스트가 증명할 것 — (a) 스키마 타입 3자리가 `boolean` 임을 **실제 `tools()` 출력**에서 확인, (b) 인자 해석 헬퍼가 위 7가지 입력을 정확히 분류, (c) `internal/mcp` 안에 `strArg(a, "…") == "true"` 형태의 비교가 하나도 남아 있지 않음(재발 방지 불변식). 고치기 전에 (a)·(c) 가 실제로 red 인 것을 먼저 볼 것.

- **건드릴 파일**:
  - `internal/mcp/server.go:675` `voiceProps` — `"open"`·`"overdue"` 를 `str(...)` → `boolean(...)` 로. 설명 문구에서 "true면" 을 불리언에 맞게 다듬을 것(예: `boolean("미해결 건만")`).
  - `internal/mcp/server.go:690` `get_recommendations` 스키마 — `"mine"` 을 `boolean("본인에게 배정된 추천만")` 으로.
  - `internal/mcp/server.go:746` 근처 — `boolArg` 옆에 하위호환 헬퍼를 하나 추가(예: `flagArg(args, key)`): `bool` 이면 그 값, `string` 이면 `== "true"`, 그 외·부재면 false. `boolArg` 자체를 고쳐도 되나 그러면 `reviewPending`·`unreadOnly` 의 계약도 넓어지므로 **한 헬퍼로 통일할지 새 헬퍼를 쓸지 먼저 정하고 한 가지로만** 갈 것(권장: `boolArg` 를 문자열도 받도록 넓히고 호출부를 전부 `boolArg` 로 통일 — 읽는 경로가 하나가 된다).
  - `internal/mcp/server.go:972` `list_customer_voices` dispatch — `OpenOnly`/`Overdue` 를 새 헬퍼로.
  - `internal/mcp/server.go:998` `get_recommendations` dispatch — `Mine` 을 새 헬퍼로.
  - 새 테스트: `internal/mcp/arguments_test.go` 에 헬퍼 케이스 추가, 스키마 타입 검증과 AST 불변식은 `internal/mcp/server_test.go` 또는 새 파일. AST 불변식은 이 저장소에 이미 있는 관용을 그대로 베낄 것 — `internal/platform/database/rows_err_test.go` 가 `go/ast`·`go/parser` 로 소스를 훑어 `path:line` 으로 찍는다.
  - 문서: `docs/api-mcp.md` 에 `open`/`overdue`/`mine` 의 타입이 문자열로 적혀 있으면 함께 고칠 것(**미확인** — 정찰에서 이 파일의 해당 대목을 열어 보지 못했다. grep 으로 확인하고 없으면 건드리지 말 것).

- **검증 명령**:
  - `go test ./internal/mcp/` (red → green)
  - `go test ./...` — 정찰에서 base 전체 통과 확인함
  - `go test -race ./...`
  - `go vet ./...`, `go build ./...`, 변경한 Go 파일에 `gofmt -l`(CI 에 gofmt 단계 있음 — cc288f3)
  - 프런트 무변경이면 web 빌드 불필요

- **위험과 피할 것**:
  - `strArg`·`intArg` 의 계약은 건드리지 말 것. 숫자 파서를 통합하거나 넓히지 말라는 운영자 지시가 있다 — 여기서도 **불리언만** 손대고 `intArg(a,"limit",…)` 류는 그대로 둔다.
  - **하위호환을 깨지 말 것**: 문자열 `"true"` 를 계속 받아야 한다. 스키마를 `boolean` 으로 바꿔 놓고 문자열 수용을 빼면 이미 붙어 있는 클라이언트가 조용히 필터를 잃는다(지금과 정반대 방향의 같은 버그).
  - 읽는 경로를 한쪽만 넓히지 말 것 — 스키마(선언)와 dispatch(해석) **양쪽을 같이** 바꾸고, 세 플래그 모두 같은 헬퍼를 지나게 할 것. 하나만 고치면 도구 안에서 타입이 또 갈린다.
  - `internal/mcp/server.go` 에는 OAuth 리소스 서버 검증(e74e752)과 도구 허용목록·권한 검사가 같이 들어 있다. 이번 diff 는 스키마 3자리 + 헬퍼 1개 + dispatch 3자리로 **작게** 유지하고 인증/권한 코드는 건드리지 말 것.
  - 보호 경로(auth·migrations·.github/workflows) 를 전혀 건드리지 않는 과제다. 그대로 유지할 것.
  - `internal/mcp/server.go` 는 한 줄이 매우 긴 스타일이다. 파일 전체 포매팅·줄바꿈 정리 금지, 바꾼 자리만 최소 수정.
  - **미확인**: `(*Server).tools(ctx, p)` 가 DB 없이 테스트에서 호출 가능한지 확인하지 못했다(`approvalsEnabled`·`toolAllowlist` 가 DB 를 볼 수 있음). 기존 `internal/mcp/server_test.go` 가 `tools()` 를 어떻게 부르는지 먼저 보고, DB 가 필요하면 수용 기준 3(a) 는 AST 불변식 3(c) 로 대체하지 말고 `schema()` 를 만드는 자리에서 검증하는 형태로 바꿀 것 — 다만 손으로 만든 대역으로 스키마를 재구성해 검사하는 것은 증거로 삼지 말 것.

- **차선 후보**: `internal/mail` 이 `rows_err_test.go` 의 `scannedPackages` 에 빠져 있다 — mail 은 rows.Err 회차(9f713a0) **뒤에** main 에 머지되었고(e5f9231), 현재 3개 루프(`digest.go:46`, `service.go:283`·`:299`)는 이미 전부 검사하고 있으므로 목록에 `"mail"` 을 더하면 곧바로 green 이다. 동작 변화가 없는 대신 다음에 들어올 mail 루프를 잡는 가드다. 1순위가 성립하지 않을 때만, 그리고 "효과 없는 변경 금지" 지시를 감안해 **회차 노트에 가드 목적임을 분명히 적고** 고를 것.

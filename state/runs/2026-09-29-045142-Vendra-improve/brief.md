- 과제: get_expiring_contracts 가 공개하는 days 범위를 실제로 답하는 범위와 맞추고, 숫자 인자의 공개 범위와 실행 상한이 어긋나면 테스트가 잡게 하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `get_expiring_contracts` 는 `tools/list` 로 `days.maximum: 730` 을 공개하는데 실행은 `intArg(name, args, "days", 180, 3650)` 이고, **이미 통과하고 있는 테스트가 그 사실을 증명한다** — `mcptools_test.go:122` 의 `{"days":3650}` 케이스는 3650일 뒤 만료되는 계약을 포함한 4건을 돌려받는다. 즉 서버는 10년 창을 답하는데 스키마는 2년까지만 물어보라고 말하고 있어, 스키마를 검증하는 MCP 클라이언트는 `days:1095` 를 전송 전에 거절하고 모델은 도구가 실제로 할 수 있는 일을 영원히 모른다. 공개 범위와 실행 상한을 잇는 테스트가 없어서 이 불일치가 여섯 회차 동안 보류 목록에만 남아 있었다.
- 수용 기준:
  1) `mcpTools` 의 `get_expiring_contracts.days.maximum` 이 실행 상한 3650 과 같다. **운영자 지시로 180/3650 정책 자체는 유지 — 고치는 쪽은 스키마다.**
  2) 새 가드 테스트가 숫자 인자를 가진 네 자리(`search_suppliers.limit` 100, `analyze_spend.limit` 100, `recommend_suppliers.limit` 50, `get_expiring_contracts.days` 3650)에 대해 `mcpTools` 의 `maximum` 과 `runMCPTool` 이 `intArg` 에 넘기는 max 가 같음을 확인한다. 이 테스트는 **지금 코드에서 먼저 돌려 `get_expiring_contracts` 한 건만 실패**해야 한다(나머지 셋은 현재 이미 일치 — 확인함). 고친 뒤 전부 통과.
  3) `TestExpiringContractsToolReturnsContracts`(`{}`→2, `{"days":30}`→1, `{"days":3650}`→4, `{"days":0.5}`→2, `{"days":1e100}`→4)와 `TestIntArgStaysInRange`, `TestMCPToolAnswersAreBounded`, `mcp_intargs_integration_test.go` 가 **한 글자도 안 고치고** 그대로 통과한다 — 기본값 180·상한 3650·LIMIT 100 동작이 불변이라는 뜻.
- 건드릴 파일:
  - `internal/httpapi/integrations.go:419` — `mcpTools` 의 `get_expiring_contracts` inputSchema 에서 `"maximum": 730` → `3650`. **이번 회차의 유일한 프로덕션 변경(파일 1개, 한 줄).**
  - `internal/httpapi/integrations.go` — `:634` 의 `intArg(name, args, "days", 180, 3650)` 은 **그대로 둔다**. 왜 스키마 쪽을 고쳤는지를 그 case 또는 `mcpTools` 근처에 이 저장소 문체(「왜 이렇게 되어 있나」를 서술하는 긴 주석, `intArg`:949-968 이 본보기)로 남길 것: 응답이 `LIMIT 100`·`ORDER BY end_date` 로 묶여 있어 창이 넓어도 가까운 100건만 나가므로 넓은 창의 비용이 작다는 점, 그리고 730 은 어떤 코드도 적용하지 않는 숫자였다는 점.
  - `internal/httpapi/mcptools_test.go` — `TestIntArgStaysInRange` 바로 아래에 DB 불필요 가드 테스트 하나 추가. `map[string]struct{arg string; max int}{"search_suppliers":{"limit",100}, "analyze_spend":{"limit",100}, "recommend_suppliers":{"limit",50}, "get_expiring_contracts":{"days",3650}}` 를 표로 두고 `mcpTools` 를 순회해 `inputSchema["properties"].(map[string]any)[arg].(map[string]any)["maximum"]` 과 대조. 표에 없는 도구에 숫자 인자가 새로 생기면 알아채도록, `maximum` 을 가진 속성이 표에 없으면 실패시킬 것(양방향).
- 검증 명령:
  - DB 불필요(먼저, 고치기 전에): `go test ./internal/httpapi/ -run 'TestIntArgStaysInRange|TestMCPTool|TestStringArg' -count=1` — 이번 정찰에서 현재 트리 0.051s 통과를 확인했으므로, 새 가드가 여기서 **실패하는 것**이 결함의 증거다.
  - 전용 `postgres:16-alpine` + 전용 DB 3개, 세 DSN(`VENDRA_TEST_DSN`, `VENDRA_TEST_MIGRATE_DSN`, `VENDRA_TEST_UPGRADE_DSN`; 뒤 둘은 빈 DB)으로 `go test ./internal/httpapi/ -run 'TestMCP|TestExpiringContracts|TestASupplierTool' -count=1` — **SKIP 0** 을 확인할 것. `TestExpiringContractsToolReturnsContracts` 가 3650 이 진짜 실행 상한이라는 배선 증거이며, 이미 있으므로 새로 쓸 필요 없다.
  - 전체: `go test ./internal/... ./cmd/... -count=1`(httpapi 약 32초), `go vet ./internal/... ./cmd/...`, `gofmt -l internal cmd`(무출력). diff 에 `gate.py secrets` 를 먼저 돌릴 것.
- 위험과 피할 것:
  - **`intArg` 의 `f < 1 → def` 동작을 이번에 건드리지 말 것.** `limit:0`·`days:0` 이 기본값(=limit 세 자리에서는 상한)으로 넓어지는 것은 2026-09-28 회차가 의도적으로 분리한 별건이고, 왜 그렇게 두었는지가 `integrations.go:964-968` 에 어제 병합된 주석으로 적혀 있다. **새 가드 테스트는 `maximum` 만 대조하고 `minimum` 은 주장하지 말 것** — 지금 `minimum:1` 과 `f<1→def` 는 일부러 어긋나 있고, `TestIntArgStaysInRange` 의 zero·negative·below one 세 케이스가 그 동작을 계약으로 고정하고 있다.
  - `get_expiring_contracts` 의 기본값 180 과 SQL 의 `LIMIT 100` 을 바꾸지 말 것. `{}`→2건 같은 기존 기대값이 흔들린다.
  - 스키마 값을 `intArg` 호출에서 읽어 오도록 **프로덕션 코드를 리팩터하지 말 것**(상수를 한 곳으로 모으는 유혹). 이번 범위는 한 줄 + 테스트이며, 배선을 바꾸면 파일 수와 위험이 함께 커진다.
  - 보호 경로(auth/session/migrations/`.github/workflows`)와 `stringValue`/`numberValue`(REST 쓰기 경로 수십 곳이 공유) 는 건드리지 말 것.
  - `docs/USER_GUIDE.md` 4.6 도구표는 도구별 한 줄 설명뿐이고 `730`·`3650` 이라는 문자열이 문서 어디에도 없다(이번에 grep 으로 확인) — 문서 변경 불필요.
- 차선 후보: MCP 권한 게이트의 `required` 맵(`integrations.go:476-482`, 7개 도구)과 `datascope_test.go` 의 역할 permission 을 대조하는 DB 불필요 가드 테스트 — 2026-09-26 에 `get_supplier_issues`/`issue.read` 로 실제로 터진 자리(테스트가 게이트에 막혀 도구에 닿지도 못했고, 그것이 그 도구에 호출 테스트가 하나도 없던 진짜 이유였다)를 다시 막는다. 프로덕션 변경 0, 위험 1.

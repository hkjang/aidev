- 과제: MCP 인자 읽기에 남은 `stringValue` 두 자리를 이 파일이 세운 계약에 맞추기 — `recommend_suppliers.category` 와 `supplierIDArg` (가치 3 / 위험 2 / 작업량 S)

- 왜: f41ecf8(2026-09-26)·a2caced(2026-09-27) 가 「문자열이 아닌 인자를 조용히 `""` 로 읽지 않는다」는 계약을 세웠는데, **같은 연산을 하는 호출부 두 곳이 아직 `stringValue` 위에 있다**(`internal/httpapi/integrations.go:657`, `:791` — 이번에 열어 확인). `recommend_suppliers {"category":5}` 는 `$1=''` 로 품목 조건을 통째로 잃은 전체 추천 목록을 「그 품목의 추천 업체」로 답하고(f41ecf8 가 "답을 전부로 넓히는 것" 이라고 판정한 결함), `get_supplier_risk {"supplierId":42}` 는 인자를 준 호출에 "requires supplierId" 라고 답한다(a2caced 가 "엉뚱한 실수를 지목하는 것" 이라고 판정한 결함). 고치면 MCP 열한 도구의 인자 읽기가 전부 같은 계약 위에 서고, 모델이 자기 실수를 고칠 수 있게 된다.

- 수용 기준:
  1) `recommend_suppliers` 에 `{"category":5}`·`{"category":true}`·`{"category":["IT"]}` 를 주면 값을 들어 거절한다(`recommend_suppliers category must be text: 5` 형식 — `stringArg` 가 이미 내는 문구). 지금은 추천 목록이 그대로 돌아온다.
  2) `get_supplier_risk`·`get_supplier_score`·`get_supplier_issues` 에 `{"supplierId":42}`·`{"supplierId":true}`·`{"supplierId":{"name":"x"}}` 를 주면 **"requires supplierId" 가 아닌** 문구로, 값을 들어 거절한다(`search_contracts`/`search_purchase_orders` 가 이미 같은 입력에 내는 문구와 같은 모양 — `mcp_stringargs_integration_test.go:91,93` 참조).
  3) 없음/`null`/`""` 는 **그대로** 「필터 없음」·「인자 없음」이다: `{"category":""}`·`{}` 는 지금과 똑같이 동작하고(`mcp_recommend_integration_test.go:108,177` 이 이미 고정), `get_supplier_risk {}` 는 여전히 "get_supplier_risk requires supplierId"(`mcptools_test.go:330-337` 이 이미 고정).
  4) 테스트는 **고치기 전에 실제 응답 본문으로 실패해야** 한다 — 1) 은 "거절 대신 행이 돌아왔다", 2) 는 "answered 'requires supplierId': 인자는 주어졌으므로 이것은 틀린 실수를 지목한다"(`mcp_supplierid_integration_test.go:142-149` 가 `get_supplier` 로 쓴 바로 그 모양을 형제 세 도구로 옮긴 것).
  5) 정상 인자 동작 불변: `{"category":"rec","minScore":80,"maxRisk":"HIGH"}` 와 `get_supplier_risk {"supplierId":"<uuid>"}` 가 고치기 전과 같은 결과(대조군 테스트를 고치기 전에도 통과시켜 보일 것).

- 건드릴 파일 (프로덕션 1개):
  - `internal/httpapi/integrations.go:657` — `runMCPTool` 의 `recommend_suppliers` 분기, `category := stringValue(args, "category")` → `stringArg(name, args, "category")` + err 반환. 같은 분기의 `numberArg`/`riskCeilingArg`/`intArg` 와 같은 모양.
  - `internal/httpapi/integrations.go:789-799` — `supplierIDArg(tool, args)` 내부 `id := stringValue(args, "supplierId")` → `stringArg(tool, args, "supplierId")` + err 반환. 그 뒤 `id == ""` → "requires supplierId", `!validUUID(id)` → "must be a record id, not a name" 순서는 **그대로 둘 것**(세 도구의 기존 문구가 거기 걸려 있다).
  - 주석: 두 자리 모두 「왜 `stringValue` 가 아니라 `stringArg` 인가」를 이 저장소 문체(길게, 피해를 서술)로 남길 것. `supplierArg` 의 "stringArg draws the same line for the text filters" 단락이 본보기이고, 그 문장이 지키지 못한 자리가 바로 `supplierIDArg` 였다고 적으면 정확하다.
  - 테스트: 새 통합 테스트 1개(예 `internal/httpapi/mcp_remaining_stringargs_integration_test.go`). 하네스는 `newScopeWorld(t)` → `w.deptToken`, `callMCPTool(t, w, token, tool, args)`(`mcptools_test.go:13`), `toolFailure`(:58)/`toolRows`(:25). **`get_supplier_issues` 는 게이트가 `issue.read` 를 요구하고 `newScopeWorld` 의 역할에 없다** — `mcp_stringargs_integration_test.go:25-32` 의 `UPDATE roles SET permissions=permissions||'["issue.read"]'::jsonb WHERE code LIKE 'scope%'` 를 그대로 쓸 것(2026-09-26 회차가 이것 때문에 걸렸다).
  - 기존 테스트 파일은 **고치지 않아도 통과해야 한다**(이번에 전수 확인: 비-문자열 `supplierId` 를 `supplierIDArg` 세 도구에 주는 케이스도, 비-문자열 `category` 케이스도 트리에 없다). 고쳐야 할 것이 나오면 그 자리가 옛 동작을 계약으로 고정하고 있다는 뜻이므로 과제서 범위를 다시 볼 것.

- 검증 명령:
  - DB 불필요(먼저): `go test ./internal/httpapi/ -run 'TestSupplierArg|TestSupplierListArg|TestStringArg|TestIntArg|TestNumberArg|TestMCPToolSchemas|TestMCPNumericSchema' -count=1` — 이번 정찰에서 0.052s 전부 PASS 확인.
  - DB 통합(본 증거): 전용 `postgres:16-alpine` 컨테이너 + 전용 DB 3개 + 세 DSN(`VENDRA_TEST_DSN`, `VENDRA_TEST_MIGRATE_DSN`, `VENDRA_TEST_UPGRADE_DSN`; 뒤 둘은 빈 DB)으로 `go test ./internal/httpapi/ -run 'TestMCP|TestASupplierTool|TestRecommend' -count=1 -v` — **SKIP 0 을 확인할 것**(DSN unset 이면 초록이어도 아무것도 검증하지 않는다).
  - 전체: `go test ./internal/... ./cmd/... -count=1`(httpapi 약 31~33초), `gofmt -l internal cmd`(무출력), `go vet ./internal/... ./cmd/...`.
  - 커밋 전 `gate.py secrets` 를 diff 에 돌릴 것(비밀번호 리터럴 게이트가 전체 실행을 떨군 적 있음).

- 위험과 피할 것:
  - **`stringValue` 자체를 손대지 말 것.** `objects.go:665-` 에 있고 REST 쓰기 경로(suppliers.go·analytics.go·portal.go·workflows.go) 수십 곳이 쓴다. MCP 계약은 `stringArg` 로 따로 두는 것이 이 저장소의 방식이고, 「계약이 다른 파서를 통합하려 하지 말 것」은 운영자 지시다.
  - 빈 문자열을 거절로 바꾸지 말 것 — `riskCeilingArg`·`stringArg` 주석이 「템플릿을 채우는 모델은 값 없는 칸에 "" 를 쓴다」를 계약으로 적어 두었고 테스트가 고정한다.
  - `minimum`·범위 밖 **숫자**(limit:0 등)는 이번 범위가 아니다(`TestIntArgStaysInRange` 가 계약으로 고정, 보류 아이디어로 남아 있다).
  - 보호 경로를 건드릴 필요 없음: auth/session/migrations/.github/workflows 무관. 권한 게이트의 `required` 맵(`:476-482`)도 손대지 말 것 — 테스트 역할에 permission 을 **SQL 로 부여**하는 것이 기존 방식이다.
  - `recommend_suppliers` 의 SQL 인자 순서($1..$8)와 `ceiling` 조건부 덧붙임은 건드리지 말 것 — dc31174 가 「인자 없는 호출의 SQL 이 글자 그대로 그대로」인 것을 설계로 삼았다.
  - 증거로 grep 결과를 내지 말 것, 손으로 만든 대역 대신 실제 로그인 세션 → `App.Handler` → `POST /mcp` 로 증명할 것(운영자 지시 2건).

- 차선 후보: MCP 권한 게이트의 `required` 맵과 `mcpTools`·역할 permission 을 대조하는 DB 불필요 가드 테스트 (2/1/S) — `required` 맵이 요구하는 permission 이 도구 이름 집합 안에 있고(철자 오류 없음), 게이트가 요구하는 것이 `newScopeWorld` 역할에 있는지 대조한다. 프로덕션 변경 0, `TestMCPNumericSchemaMaximumsMatchEnforcedCeilings` 가 본보기.

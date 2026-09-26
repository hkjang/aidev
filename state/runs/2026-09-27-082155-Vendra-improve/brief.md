- 과제: compare_suppliers 가 비교 대상 하나를 조용히 버리지 않게 하고, 형식이 틀린 supplierId 를 「인자를 안 줬다」로 답하지 않게 하기 (가치 3 / 위험 2 / 작업량 S~M)

- 왜: `compare_suppliers` 의 ID 목록은 `supplierListArg` → `stringSlice`(integrations.go:750) 를 지나는데 이 헬퍼는 **문자열이 아닌 원소를 조용히 버린다** — `{"supplierIds":["<uuid-A>",42,"<uuid-B>"]}` 는 A·B 두 건만 비교하고, `{"supplierIds":["<uuid-A>",42]}` 는 스키마가 `minItems:2` 로 공개한 도구가 한 건짜리 「비교」를 정상 응답으로 돌려준다. 모델은 빠진 업체를 「그 업체는 없습니다/접근할 수 없습니다」로 사람에게 옮긴다. 같은 자리에서 목록 전체가 숫자면(`{"supplierIds":[1,2]}`) 응답은 "compare_suppliers requires supplierIds" — 준 값이 틀렸다는 말이 아니라 안 줬다는 말이고, `get_supplier` 도 `supplierArg`(:710, `stringValue` 기반)를 써서 `{"supplierId":42}` 를 "get_supplier requires supplierId" 로 답한다. 이번 수정은 2026-09-23(e0aa306)·09-26(f41ecf8)에서 형제 도구들에 이미 적용한 계약을 마지막으로 남은 두 도구에 맞추는 것이고, 고치면 MCP 도구 열한 개가 같은 인자 실수에 같은 문구로 답한다.

- 수용 기준:
  1) `{"supplierIds":["<scope 안 uuid>",42]}` 는 한 건짜리 비교 결과가 아니라 `isError` 응답이며, 문구가 문제된 값(42)을 들어 보여 준다(`json.Marshal` 로 렌더 — `stringArg`/`numberArg` 와 같은 방식).
  2) `{"supplierIds":[1,2]}`·`{"supplierId":42}` 의 응답에 "requires supplierId" 계열 문구가 더는 쓰이지 않는다(인자를 준 호출이므로). 값이 실제로 빠진 `{}` 는 지금 문구 그대로 유지.
  3) `get_supplier` 가 이름을 받으면 형제 도구와 같이 record id 가 아니라고 말한다(현재는 "supplier not found" — 존재하지 않는다는 말로 읽힌다). `supplierId`/`id`, `supplierIds`/`ids` 별칭과 `{"supplierId":"   ","id":"<uuid>"}` 같은 빈 값 넘기기는 **그대로 동작해야 한다**(하드코딩한 클라이언트 호환).
  4) 테스트는 고치기 전에 실패해야 하며, 실패가 실제 응답 본문(구조화 결과 행 수 또는 isError 문구)으로 드러나야 한다. 정상 인자 호출(uuid 두 개 비교, `get_supplier` uuid) 대조군은 고치기 전에도 통과해야 한다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/httpapi/integrations.go`
    - `supplierArg`(:700-717) — 이름 목록을 돌며 값이 존재하고 nil 이 아닌데 문자열이 아니면 `mcpToolError` 로 값을 들어 거절. 문자열이면 TrimSpace 후 비어 있으면 다음 이름으로(기존 동작 유지). 시그니처는 `(string, error)` 로. 호출부는 `get_supplier`(:536) 한 곳뿐.
    - `supplierListArg`(:740-748)·`stringSlice`(:750-758) — 원소 하나라도 문자열이 아니면 거절하는 계약으로. `v` 가 배열이 아닌 경우(`"not a list"`)도 거절 대상인지 결정해 주석에 이유를 남길 것. 두 헬퍼 모두 MCP 밖에서 쓰이는 곳이 없음(`grep` 으로 확인: `stringSlice`/`supplierArg`/`supplierListArg` 의 비-테스트 호출부는 integrations.go 뿐).
    - `get_supplier` 분기(:534-549) — 이름(비-UUID)에 대해 `validUUID`(app.go:147) 검사를 앞세워 `supplierId must be a record id, not a name: %q`(형제 문구와 동일)로. 실제로 없는 uuid 는 "supplier not found" 그대로, 옆 부서 uuid 는 `canAccessSupplier` 의 "data scope denied" 그대로.
  - `internal/httpapi/mcpargs_test.go` — 지금 **틀린 동작을 고정하고 있다**: `TestSupplierArgAcceptsEitherName`(:25 부근)이 `{"supplierId":42}` → `""` 를, `TestSupplierListArgAcceptsEitherName` 이 `{"supplierIds":"not a list"}` → 빈 목록을 기대한다. 새 계약으로 갱신(에러를 기대).
  - 새 통합 테스트(예 `internal/httpapi/mcp_supplierid_integration_test.go`) — `newScopeWorld(t)` + `callMCPTool`(mcptools_test.go:13) + `toolRows`/`toolFailure` 로 실제 세션 쿠키·`POST /mcp` 경유. 본보기: `mcp_stringargs_integration_test.go`, `mcptools_test.go:242`.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 전용 postgres 컨테이너 + 전용 DB 3개, DSN 셋 `VENDRA_TEST_DSN`/`VENDRA_TEST_MIGRATE_DSN`/`VENDRA_TEST_UPGRADE_DSN`(뒤 둘은 빈 DB).
  - 먼저 고치기 전: `go test ./internal/httpapi/ -run 'TestCompare|TestASupplierTool|TestMCP' -count=1 -v` 로 새 테스트의 실패 본문을 남길 것.
  - 고친 뒤: `go test ./internal/httpapi/ -run 'TestSupplierArg|TestSupplierListArg|TestMCP|TestASupplierTool' -count=1`(SKIP 0 확인), `go test ./internal/... ./cmd/... -count=1`, `gofmt -l internal cmd`(무출력), `go vet ./internal/... ./cmd/...`.
  - 참고: 이번 정찰에서 `go test ./internal/httpapi/ -run 'TestSupplierArg|TestSupplierListArg' -count=1` 은 DSN 없이 통과(0.0x초) — 이 두 개는 DB 없이 돈다.

- 위험과 피할 것:
  - 별칭(`id`/`ids`)과 빈 문자열 넘기기 동작을 깨지 말 것 — `mcpargs_test.go` 의 통과 케이스 네 개가 그 계약이다. `stringValue` 자체는 REST 쓰기 경로(suppliers.go/analytics.go/portal.go 등 수십 곳)에서 쓰이므로 **절대 손대지 말 것**(f41ecf8 이 `stringArg` 를 새로 만든 이유).
  - `compare_suppliers` 의 `maxItems`/응답 크기 상한은 이번 범위 밖(보류 아이디어로 남겨 둘 것). `minItems:2` 를 서버에서 강제하는 것도 범위 밖 — 정당한 한 건 호출을 깨뜨릴 수 있다.
  - 인자 실수의 원문을 감사 로그로 새로 흘리지 말 것. `mcpCall` 은 이미 `arguments` 를 감사에 기록하므로 추가 기록을 넣지 말 것.
  - 손으로 만든 대역·Principal 직접 대입으로 증명하지 말 것. SQL 소스 문자열 grep 도 증거가 아니다. 보호 경로(auth/oidc/migrations/.github/workflows) 무관 — 건드릴 필요 없다.
  - `newScopeWorld` 는 이름·번호·사업자번호를 같게 시드하므로 목록 테스트는 uuid 두 개(`w.mySupplier`, 필요하면 `w.theirSupplier`)를 쓸 것.

- 차선 후보: MCP 권한 게이트(`mcpCall` 의 `required` 맵, integrations.go:476-482)가 요구하는 permission 전부를 `mcpTools` 와 대조하는 DB 불필요 가드 테스트 — 2026-09-26 회차에서 `get_supplier_issues` 의 `issue.read` 가 `newScopeWorld` 역할에 없어 그 도구를 부르는 테스트가 트리에 하나도 없었다는 사실이 드러났고, 같은 공백을 다시 막는다.

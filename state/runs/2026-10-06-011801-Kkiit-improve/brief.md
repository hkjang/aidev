- 과제: MCP `submit_requirement` 가 주문 양식 검증을 건너뛰어 판매자가 받은 요구사항을 지우고도 주문을 READY 로 올린다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `orders.requirements` 를 쓰는 경로가 둘인데 계약이 다르다. 생성 경로 `createOrder`(internal/httpapi/orders.go:126)는 `normalizeRequirements(ctx, tx, talentID, in.Requirements, true)` 를 통과시켜 필수 항목 누락(400 `requirements_incomplete`)·항목당 5000자/전체 40000자 초과를 거절하고 키를 **판매자가 보는 라벨**로 정규화해 저장한다. 반면 MCP 경로 `mcpSubmitRequirement`(internal/httpapi/mcp.go:401-411)는 `args["requirements"]` 맵을 검증·정규화 없이 그대로 `UPDATE orders SET requirements=$3, state=CASE WHEN state='REQUIREMENT_PENDING' THEN 'READY' ELSE state END ... WHERE id=$1::uuid AND buyer_id=$2 AND state IN ('CREATED','PAID','REQUIREMENT_PENDING')` 로 덮어쓴다. 그래서 `orders.buy` 스코프를 가진 AI Agent 구매자가 `{}` 나 아무 키를 담아 호출하면 생성 때 검증된 답변이 통째로 사라지고, 주문이 REQUIREMENT_PENDING 이었다면 "양식 작성 완료" 를 뜻하는 READY 로 올라간다 — 판매자는 작업 지시가 빈 주문을 착수 가능 상태로 받는다. 길이 상한도 없어 무제한 JSON 이 들어간다.
- 수용 기준:
  1) 필수 요구사항 항목을 비운 `submit_requirement` 호출은 실패하고, 주문의 `requirements` 와 `state` 가 호출 전과 **똑같이 남는다**(REQUIREMENT_PENDING 이 READY 로 올라가지 않는다).
  2) 정상 입력(필수 항목을 채운 호출)은 종전처럼 성공하고, 저장된 `orders.requirements` 의 키가 `createOrder` 가 저장하는 것과 같은 **라벨** 형태다. REQUIREMENT_PENDING → READY 승격은 그대로 동작한다.
  3) 테스트가 실제 MCP 엔드포인트(POST /mcp, JSON-RPC `tools/call`) → 실제 PostgreSQL 왕복으로 (1)(2)를 증명한다. 수정 전에 먼저 실패시키고, 고친 뒤 통과시킨 로그를 남긴다.
- 건드릴 파일 (프로덕션 1개):
  - `internal/httpapi/mcp.go:401` `mcpSubmitRequirement` — ①`args["order_id"]` 를 Go 에서 `uuid.Parse` 로 먼저 파싱해 거절(지금은 비-uuid 문자열이 PG 22P02 를 일으켜 "요구사항을 갱신할 수 있는 주문이 아닙니다" 로 둔갑한다). ②`s.DB.Begin` 으로 tx 를 열고 `SELECT talent_id FROM orders WHERE id=$1 AND buyer_id=$2 AND state IN ('CREATED','PAID','REQUIREMENT_PENDING')` 로 대상 주문과 talent 를 잡고, ③`normalizeRequirements(ctx, tx, talentID, requirements, true)` 의 결과를 `UPDATE` 에 넣고, 실패 메시지는 normalizeRequirements 가 돌려주는 한국어 문구를 그대로 쓴다(새 문구 만들지 말 것). ④커밋.
  - `internal/httpapi/integration_test.go` — 기존 `mcpClient.tool` / `mcpClient.toolError`(1237, 1248행)와 `TestIntegration…` 관례로 새 테스트 추가.
  - (선택) `docs/mcp.md` 에 submit_requirement 가 주문 양식 검증을 받는다는 한 줄. 없으면 생략.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 단독: `go test ./internal/httpapi/ -run TestIntegration -v` (KKIIT_TEST_DSN 필요, 없으면 SKIP — SKIP 으로 끝났으면 증명이 아니다)
  - 통합: `make test-integration KKIIT_TEST_DSN=postgres://…` (전용 폐기 DB, 과거 80~100초)
  - 전체: `go test ./cmd/... ./internal/...`, `go vet ./cmd/... ./internal/...`, `gofmt -l cmd internal`
- 위험과 피할 것:
  - `normalizeRequirements` 의 시그니처·동작을 바꾸지 말 것. `createOrder`(requireAll=true)와 `rfq.go:408`(requireAll=false) 두 호출자가 서로 다른 계약으로 쓰고 있다 — 통합하려 하지 말고 MCP 호출만 추가한다.
  - `orders.go`·`finance.go`(원장), `orderTransitions` 표, `auth.go`/`middleware.go`, `internal/database/migrations`, `.github/workflows` 는 건드리지 말 것.
  - `requireAll=true` 로 하면 **기존 Agent 호출이 거절될 수 있다**(호환성 변경). 이것이 의도한 수정이지만, 과하다고 판단되면 승격이 일어나는 경우(state='REQUIREMENT_PENDING')에만 requireAll=true 를 쓰고 CREATED/PAID 에서는 false 를 쓰는 좁은 변형도 수용 기준을 만족한다. 둘 중 무엇을 골랐는지 커밋 메시지에 적을 것.
  - 통합 테스트 준비물(확인됨): `integration_test.go:2582` 가 `feature_flags` 의 `agent_marketplace` 를 켠다. POST /mcp 는 `s.require("mcp.use", s.requireFeature("agent_marketplace", …))`(router.go:180)이고 submit_requirement 자체는 `orders.buy` 스코프(mcp.go:141)를 요구한다. `integrationServer` 는 `apiUnderTest` 전역을 공유하므로 **테스트를 병렬화하지 말 것**.
  - **미확인**: ⓐ필수 `talent_requirements` 항목을 가진 상품을 만들고 그 주문을 REQUIREMENT_PENDING 까지 옮기는 기존 통합 도우미가 있는지(상태 전이는 `PATCH /orders/{id}/state` 계열을 직접 호출해야 할 수 있다) ⓑ`mcpClient` 를 만드는 기존 테스트가 쓰는 API 키 발급 흐름의 정확한 도우미 이름 ⓒ`docs/mcp.md` 의 현재 submit_requirement 기술 내용. 세 가지는 구현자가 `integration_test.go` 에서 `mcpClient{` 를 쓰는 기존 테스트 하나를 찾아 그대로 본뜨는 것이 가장 빠르다.
  - 증거 규칙: grep 으로 "검증 호출이 생겼다" 를 증명하지 말 것. 반드시 HTTP→DB 왕복으로 state·requirements 값을 다시 읽어 보일 것. 수정 후 분기를 한 번 되돌려 테스트가 다시 실패하는 것(변이 확인)까지 보일 것.
- 차선 후보: `createOrder` 의 추가 옵션 가격 합산 int64 오버플로 — `price += optionPrice`(orders.go:95)에 상한이 없고 `validateTalent`(talents.go:265,283)는 음수만 거절하므로 base_price 와 옵션 가격을 각각 2^62 로 올리면 `price` 가 음수로 돌아간다. `orders.amount` 는 `CHECK (amount >= 0)`(migrations/001_initial.sql:252)이라 INSERT 가 터져 500 "주문을 만들지 못했습니다." 가 나가고, 그 전에 음수 `price` 가 `resolveCoupon`·`reserveBudget`(orders.go:131,159)에 그대로 들어간다. (지난 회차가 1<<57 가격의 상품 등록·공개를 실제로 성공시킨 전례가 있다. 다만 orders.go 는 원장 경로라 위험이 한 단계 높다 — 1순위가 성립하지 않을 때만.)

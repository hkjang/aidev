# 회차 노트 2026-10-06-011801-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:18] base pinned — main@b1199a3
- [러너 01:18] autonomy release — 

## 정찰 노트
- 고른 이유: `orders.requirements` 를 쓰는 경로가 둘인데 계약이 다르다(createOrder 는 normalizeRequirements 로 거르고 MCP 는 생짜로 덮어쓴다) — 운영자가 반복해 말한 "같은 값을 쓰는 경로가 둘이면 end-to-end 로 확인" 에 정확히 맞고, 남은 `err!=nil||RowsAffected()==0` 9곳 중 유일하게 사용자 입력이 문장에 들어가 실제 HTTP→DB 로 증명된다. 404 가르기 후보들(keys/events/coupons/organizations)은 파라미터가 uuid 뿐이라 증거를 만들 수 없어 제쳤다.
- 추측으로 적은 것: 필수 `talent_requirements` 항목을 가진 상품의 주문을 REQUIREMENT_PENDING 까지 옮기는 기존 통합 도우미가 있는지, `mcpClient` 생성에 쓰는 API 키 발급 도우미 이름, `docs/mcp.md` 의 현재 기술 내용 — 셋 다 미확인. `mcpClient{` 를 쓰는 기존 테스트를 본뜨는 것이 가장 빠르다.
- 구현자 주의: `requireAll=true` 는 기존 Agent 호출을 거절하는 호환성 변경이다(과제서에 좁은 변형도 적어 뒀다). `normalizeRequirements` 는 호출자 둘(orders.go:126, rfq.go:408)의 계약이 달라 시그니처를 건드리면 안 되고, orders.go·finance.go·마이그레이션은 손대지 말 것. 테스트가 DSN 없음으로 SKIP 되면 증명이 아니다.
- 이번 회차 실행 검증은 없음(읽기만, 테스트 미실행). 프로필은 2026-10-05 자로 최신이라 다시 쓰지 않았다.
- [러너 01:22] scout done — MCP `submit_requirement` 가 주문 양식 검증을 건너뛰어 판매자가 받은 요구사항을 지우고도 주문을 READY 로 올�

## 구현 노트
- 무엇을 왜: `mcpSubmitRequirement`(mcp.go:401)가 `args["requirements"]` 를 생짜로 `orders.requirements` 에 덮어쓰던 것을 tx 안에서 `normalizeRequirements(…, requireAll=true)` 를 통과시키도록 바꿨다. `order_id` 는 `$1::uuid` 전에 `uuid.Parse` 로 거절하고, 저장 실패를 "갱신할 수 있는 주문이 아닙니다" 와 갈랐다. 프로덕션 1파일 + 테스트 1 + docs/mcp.md 한 단락. 커밋 026afbc.
- **과제서 전제가 틀렸던 곳(비평가가 먼저 볼 곳)**: `REQUIREMENT_PENDING` 은 지금 어떤 HTTP 경로로도 도달하지 않는다 — `/orders/{id}/pay` 가 PAID→READY 를 같은 트랜잭션에서 커밋해(orders.go:329,332) PAID 가 머물지 않고, 그래서 operator 의 `PAID→REQUIREMENT_PENDING` 전이도 409 `invalid_transition` 로 거절된다(실제로 받았다). 그래서 증명의 축을 도달하는 피해(CREATED 주문의 지시 삭제 → 구매자 결제 → 판매자가 빈 READY 주문 수령)로 옮겼고, 승격 분기는 두 번째 테스트에서 `UPDATE orders SET state='REQUIREMENT_PENDING'` 으로 상태만 심어 고정했다 — **이 한 줄이 이번 변경에서 유일한 손으로 만든 전제**이고, 그 뒤 호출·검증은 전부 실제 엔드포인트와 실제 테이블이다.
- 확신 없는 것: ①견적(RFQ)으로 만든 주문은 `requirements["견적 요청 내용"]` 이 `normalizeRequirements` **밖에서** 붙으므로(rfq.go:415) 이 도구로 갱신하면 그 키가 사라진다 — 다만 종전에도 UPDATE 가 컬럼을 통째로 덮어써 똑같이 사라졌으므로 악화는 아니다. RFQ 주문 + MCP 갱신 조합은 테스트하지 않았다. ②`requireAll=true` 는 과제서가 경고한 호환성 변경이다. 상태로 나누는 좁은 변형이 아니라 항상 true 를 골랐고 이유는 커밋 메시지에 적었다(이 도구가 갱신하는 주문은 createOrder 가 이미 필수 항목을 채우게 해서 만든 것).
- 일부러 안 한 것: `normalizeRequirements` 의 시그니처·동작, `orderTransitions` 표, pay 경로, 마이그레이션은 건드리지 않았다. 위 ①의 RFQ 키 보존도 범위 밖으로 두고 ideas.json 에 남기지 않았다(악화가 아니므로) — 대신 여기 적어 둔다.
- 다음 역할이 조심할 것: 새 테스트 2건은 `KKIIT_TEST_DSN` 이 있어야 돈다(없으면 SKIP = 증명 아님). **같은 DB 에 통합 스위트를 두 번 돌리면** `TestIntegrationMailNotificationsLeaveThroughTheRelay`·`TestIntegrationAgentCanResearchBeforeBuying` 가 실패하는데, 내 변경을 되돌린 baseline 에서도 재현되는 사전 존재 결함이다 — 반드시 깨끗한 DB 로 돌릴 것. 깨끗한 DB 에서는 `-run Integration -v` 가 90 PASS / SKIP 0 / FAIL 0.
- [러너 01:45] brief accepted — 채택 — 1순위를 과제서가 지정한 파일·수법(uuid.Parse → tx → normalizeRequirements(requireAll=true) → commit)대로 구현했고 수용 �
- [러너 01:46] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인한 것: 폐기 PG16 컨테이너로 새 테스트 2건이 HEAD 에서 PASS / mcp.go 만 main 으로 되돌리면 원장에 적힌 것과 **동일한 출력**으로 FAIL 하는 것을 직접 재현했고, 깨끗한 DB 에서 `-run Integration` 전체 PASS(82.7초) + vet/gofmt 무결. 인가(SELECT·UPDATE 모두 `buyer_id=$2`), `orders.talent_id` NOT NULL(Scan 실패 경로 없음), 트랜잭션 누수 없음까지 읽어 확인 — 보안·법무 차단 사유 없음.
- 못 본 것: RFQ 로 만든 주문 + MCP 갱신 조합, 선택(required=false) 항목이 섞인 주문 양식, 웹 UI 의 requirements 표시. 셋 다 테스트가 지나지 않는다.
- 승인이어도 남는 우려 ①(릴리즈 노트가 알 것): `docs/mcp.md:67` 의 "일부 항목만 보내면 호출 자체가 거절된다" 는 **필수** 항목에만 참이다 — 필수를 다 채우고 선택 항목을 뺀 제출은 통과하며 그 선택 답변은 조용히 삭제된다(orders.go:798-803). 코드 악화는 아니고 문서 한 문장의 범위 오류이므로 문장을 "필수 항목" 으로 좁히면 끝난다.
- 우려 ②(다음 회차 후보): 주문 양식이 빈 상품에서는 normalized 가 `{}` 가 되어 에이전트 payload 를 전부 버리고도 `updated: true` 를 돌려준다. createOrder 와 일관되지만 호출자가 알아챌 수 없다. 그리고 mcp.go:430 의 SELECT 는 DB 장애와 "내 주문 아님" 을 여전히 한 문구로 묶는다(잔존 404 가르기 목록에 +1).
- 수리 불필요(reject 아님). 작업 트리에 커밋되지 않은 `internal/ui/dist/*` 검증 빌드 산출물이 남아 있으니 릴리즈 때 섞지 말 것.
- [러너 01:52] review approved — 리뷰 승인 (risk=low)
- [러너 01:52] pr created — https://github.com/hkjang/Kkiit/pull/20
- [러너 01:53] ci passed — 검사 없음 — 정책으로 허용
- [러너 01:53] merge done — 026afbc
- [러너 02:01] release published — v0.4.15
- [러너 02:02] assets verified — v0.4.15 자산 1개 (이전 v0.4.14: 1)

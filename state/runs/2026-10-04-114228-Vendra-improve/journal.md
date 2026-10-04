# 회차 노트 2026-10-04-114228-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:42] base pinned — main@8887dc2
- [러너 11:42] autonomy release — 

## 구현 노트
- 무엇을 왜: `suppliers.annual_spend` 를 여는 권한 목록이 두 벌이었다 — `redactSupplier`(상세·목록·포털·`get_supplier`)는 `supplier.financial.read` 를 받고, 그 옆 요약 질의 네 자리(integrations.go:160·531, analytics.go:663)는 받지 않아, 그 권한만 가진 역할이 같은 세션 한 호출 차이로 실제 금액과 0 을 읽었다. `canReadSupplierSpend(p)` 한 곳으로 모았다(commit b50d3f3, 프로덕션 3파일 +29/-4).
- 확신 없는 곳: ① 이 변경은 `supplier.financial.read` 보유자에게 MCP·공급망의 지출을 **넓힌다**. 안전 근거는 「같은 값을 `get_supplier`·REST 상세·목록에서 이미 읽을 수 있다」는 것(새로 닿는 데이터 0)이지만, 그 권한의 의도를 「상세만」으로 보는 해석이 있다면 방향이 반대다 — 비평가가 먼저 볼 곳. ② 각 자리의 `hasPermission(p, "*")` 를 뺀 것은 `permissionMatches` 의 `got=="*"` 에 기대며, `["*"]`·`["*.read"]`·`["spend.*"]` 역할 행으로 단정했다.
- 검증 못 한 것: 웹 스위트·`npm run build`(diff 에 web 파일 0개라 돌리지 않았다), 원격 CI, 릴리즈 경로.
- 일부러 안 한 것: 가려진 지출이 `ELSE 0`(0) 과 `END`(null) 로 갈리는 불일치 — `redactSupplier` 자신이 `AnnualSpend = 0` 이라 0 이 관례이고, MCP 만 null 로 바꾸면 같은 가림을 두 표면이 다르게 답한다. ideas.json 에 3/3/M 로 넘겼다.
- 다음 역할이 조심할 것: 새 테스트 2건은 **DB 가 있어야 돈다**(세 DSN; 전용 `vendra-1004-improve-pg`, 127.0.0.1:55471). `grantOnly` 가 `scope%` 역할의 permissions 를 **덮어쓰므로** 같은 fixture 를 쓰는 테스트와 병렬 실행 금지. httpapi 전체 SKIP 5건은 전부 기존·환경성이다.
- [러너 11:53] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: `canReadSupplierSpend` 의 세 wording 과 `permissionMatches`(context.go:96-100, `*`·`*.read`·`spend.*` 전부 통과) / `redactSupplier` 호출 지점 6곳이 이미 같은 값을 내주므로 MCP 확대는 새로 닿는 데이터가 0 — security·legal 차단 사유 없음 / 원장의 `- 실패 재현:` 본문과 코드상 수정 전 실패 경로가 일치, 테스트 헬퍼는 전부 기존 파일.
- 거절 사유: `annual_spend` 를 공급업체별로 내보내는 **다섯 번째 자리**가 남았다 — analytics.go:14 `showAmounts` 가 옛 목록이라 :56-76 의 `topSuppliers[].annualSpend` 가 `supplier.financial.read` 역할에 0 을 내고(Dashboard.tsx:230-247 이 그 0 을 /suppliers 링크 옆에 렌더), suppliers.go:328-350 주석과 커밋 제목의 "every surface" 는 그만큼 사실이 아니다.
- 수리가 먼저 볼 파일: internal/httpapi/analytics.go. :14 를 그대로 치환 금지 — 같은 플래그가 `contractValue`(business_objects.amount)도 가려서 치환하면 계약 금액까지 넓어진다. 플래그를 둘로 쪼개거나, 범위를 좁힐 생각이면 주석·커밋 주장을 실제 범위로 고치고 :14 를 남은 자리로 명시하면 해제된다.
- 못 본 것: DB 통합 테스트를 직접 돌리지 않았다(컨테이너 미기동, 구현자 기록으로 대체). 러너 verify 의 `go test ./...` 는 2초 = DB 통합 전부 SKIP 이므로 이 변경은 게이트가 지키지 않는다.
- 남는 우려: 가려진 지출이 0(`ELSE 0`)과 null(`END`)로 갈리는 불일치는 ideas.json 대로 보류가 맞으나, 새 테스트가 둘을 모두 0 으로 읽으므로 그 회차에도 초록이다 — 통일할 때 단언을 같이 손봐야 한다.
- [러너 11:58] review rejected — 리뷰 거절: internal/httpapi/analytics.go:14 `dashboard` 의 `showAmounts` 는 고쳐지지 않은 **다섯 번째 자리**다 — 같은 파일 :663 은 `canReadSupplierSpend(p)` 로 바꾸면서 

## 수리 노트
- 맞았던 지적: `analytics.go:14` 은 정말 고쳐지지 않은 다섯 번째 자리였고, `suppliers.go:328-` 주석/커밋 제목의 "every surface" 가 코드보다 넓었다. 틀린 지적은 없다. 「:14 를 그대로 치환 금지 — `contractValue` 까지 넓어진다」는 경고도 정확했다(같은 플래그가 `business_objects.amount` 를 덮고 있었다).
- 고친 방법: 플래그를 `showSpend`(= `canReadSupplierSpend`, annual_spend 두 자리)와 `showContractValue`(옛 목록 그대로, amount 한 자리)로 쪼갰다. 주석에 대시보드 두 자리를 열거에 넣고 범위 문장(「annual_spend 뿐, activeContractValue 는 자기 문 유지」)을 추가했다.
- 검증: 새 `TestDashboardReadsTheSameSupplierSpendDoor` 가 수정 전 `supplier.financial.read` 벌에서만 3줄 실패 → 수정 후 통과(실패를 먼저 확인했다). DB 세 DSN 으로 `go test ./internal/... ./cmd/...` 전부 ok, SKIP 5건 기존과 동일, gofmt/vet/gate secrets 깨끗.
- 여전히 확신 없는 곳: ① 구현자의 ① 그대로 — `supplier.financial.read` 의 의도를 「상세만」으로 읽는 해석이면 방향이 반대다(이번에 대시보드 집계 KPI 까지 넓어졌다; 근거는 같은 열을 /suppliers 목록에서 이미 전건 읽을 수 있어 합도 유도 가능하다는 것). ② `activeContractValue` 를 좁은 쪽에 남긴 것은 범위 보존 판단이며, 「대시보드 금액은 하나의 문」이라는 해석이면 재검토 대상이다.
- 손대지 않은 것: 웹(`Dashboard.tsx` 는 API 가 준 값을 `money()` 로 렌더할 뿐 — 서버 수정으로 해결, web diff 0). 0 vs null 불일치는 앞 회차대로 ideas.json 보류.
- [러너 12:03] repair done — # 수리 요약 (efafce8)  - 지적은 맞았다. `analytics.go:14` 의 `showAmounts` 한 플래그가 **두 열**을 덮고 있었다 — `suppliers.annual_spend`(KPI `annualSpend` + `topSuppliers[].

## 비평 노트
- 확인한 것: 수정 전 어구로 `canReadSupplierSpend` 를 되돌리고 `redactSupplier` 를 인라인 3항으로 복원한 상태에서 신규 3건 전부 FAIL(문구가 원장의 증상과 일치) → HEAD 에서 전부 PASS. 세 DSN(`vendra-1004-improve-pg`)으로 `go test ./internal/... ./cmd/...` 전부 ok(httpapi 27.1s), gofmt/vet 공백. `hasPermission(p,"*")` 제거는 permissionMatches 의 `got=="*"` 와 `["*"]`·`["*.read"]`·`["spend.*"]` 역할 행으로 실증됨. 헬퍼 전부 기존 파일, scopeWorld 사용 파일에 t.Parallel 없음.
- 판정: approve, risk low, blocking 없음 (security·legal 차단 사유 없음).
- 남는 우려 ①: 확대 범위가 노트의 안전 근거보다 조금 넓다 — MCP(:485 supplier.read|*.read)·supplier-network 는 새 데이터 0 이지만, **대시보드(dashboard.read 만)와 /api/v1/ai/analyze(ai.use 만)** 는 `supplier.financial.read` 를 갖고 `supplier.read` 가 없는 커스텀 역할에게 지출 합계·Top5·AI 요약 금액을 새로 보여준다. 출하 역할 중 해당 권한 보유자는 0 이라 수용.
- 남는 우려 ②: `suppliers.go:329` 의 "every surface" 는 아직 `analyze_spend`(integrations.go:668)·`GET /api/v1/spend`(analytics.go:526-)를 포함하지 않는다. 둘은 문이 `spend.read` 라서 0 대신 거절이 나가므로 피해 구조가 다름 — 문구 정밀도만.
- 릴리즈/다음 회차: 0 vs null 통일 시 `spendByName`(테스트 :253)의 단언을 같이 고쳐야 한다. CI 는 ci.yml:52-54 에서 세 DSN 을 주므로 이 테스트는 원격에서 실제로 돈다. 마이그레이션 0 — revert 는 커밋 두 개로 끝.
- [러너 12:08] review approved — 리뷰 승인 (risk=low)
- [러너 12:08] pr created — https://github.com/hkjang/Vendra/pull/139
- [러너 12:10] ci passed — 검사 2개 모두 success
- [러너 12:10] merge done — efafce8

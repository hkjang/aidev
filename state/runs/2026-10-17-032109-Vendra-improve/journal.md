# 회차 노트 2026-10-17-032109-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:21] base pinned — main@e030b63
- [러너 03:21] autonomy release — 

## 정찰 노트
- 웹(Objects.tsx)으로 네 회차를 연달아 갔고 011c5a3 이 아직 HEAD 조상이 아니어서, 같은 파일에 또 들어가는 대신 Go 쪽 미해결 절반을 골랐다: 가려진 annualSpend 가 MCP 두 도구에서 0, 형제 둘에서 null. 세 회차 연속 "계약 결정 선행" 으로 밀렸던 3/3/M 아이디어를 **MCP 전용 경로(supplierSummaryRows)로 좁혀** S 로 만든 것이 이번 판단의 핵심 — REST/웹 타입(types.ts annualSpend: number)은 손대지 않는다.
- 제쳤다: compare_suppliers 의 조용한 ID 누락(기존 테스트 :187 이 그 동작을 의도로 고정 → 기각), mcpObjects 의 LIMIT 100 절단 신호(get_supplier_risk 주석이 "조용히 자른다"를 의도로 명시해 저장소 입장과 충돌 + 배열→객체 계약 변경), 저장된 보기 실패(차선으로 유지, 같은 파일의 미병합 변경 때문).
- 추측으로 적은 것: ① `jsonb_build_object` 가 null 값을 키째로 남긴다는 것은 recommend_suppliers 의 코드 모양에서 추론했고 **응답 본문으로 실측하지 않았다**(DB 미실행) — 그래서 기준 1 을 recommend_suppliers 와 나란히 비교하도록 적었다. ② 기존 테스트가 안 깨진다는 예상은 spendByName 의 `.(float64)` 읽기에 근거한 것이고 실행으로 확인하지 않았다.
- 구현자가 조심할 것: `spendByName` 으로 새 테스트를 쓰면 null 이 0 으로 읽혀 결함을 구조적으로 못 본다 — `map[string]any` 로 nil 을 직접 볼 것. `grantOnly` 는 테이블 각 행 안에서 부를 것(루프 밖이면 엉뚱한 403). DSN 3개 없이 통과한 Go 테스트는 증거가 아니다.
- [러너 03:28] scout done — MCP `search_suppliers`·`compare_suppliers` 가 **가려진 연간지출을 `annualSpend: 0`** 으로 답하는 것 고치기 — 같은 파

## 구현 노트
- 무엇·왜: MCP `search_suppliers`·`compare_suppliers` 의 `CASE WHEN $5 THEN annual_spend ELSE 0 END` → `... END` 로, `supplierSummaryRows` 의 `spend` 를 `*float64`(옆의 `grade`·`score` 와 같은 이유)로. 가려진 금액이 0 이라는 숫자로 나가 모델이 「거래액 0원 업체」라는 허구의 사실을 사람에게 전달했다. 프로덕션 1파일, REST·웹·마이그레이션·감사 무변경.
- 과제서 기준 5 를 한 곳 못 지켰다(가장 먼저 볼 곳): 기존 `mcp_limits_integration_test.go:190` 의 `amount permissions` 가 `search_suppliers`=`float64(0)` / `recommend_suppliers`=`nil` 로 **옛 모양을 못 박아** 기준 1 과 양립 불가였다(과제서는 대조군 세 개만 감사하고 이 파일을 빠뜨렸다). 그 조건을 `if amount != nil` 로 좁혀 두 도구를 같은 모양으로 묶고 사유 주석을 남겼다 — 그 하위 테스트가 심는 행은 `annual_spend=1000-g` 이라 실제 검사는 「진짜 금액 누출」이고 엄격도는 동일하며, 되돌림 실행에서 이 줄도 같이 빨강이 되어 유효함을 확인했다. 과제서가 지목한 대조군 3개는 무수정 통과.
- 검증 못 한 것: MCP 를 쓰는 실제 모델/클라이언트가 `annualSpend:null` 을 어떻게 읽는지(스키마에 타입 선언이 없어 깨질 자리는 못 찾았지만 실제 클라이언트로 돌려보지는 않았다). 웹·`npm test` 는 돌리지 않았다 — 이 경로(`supplierSummaryRows`)는 MCP 전용이라 TS 타입이 닿지 않는다(`git diff` 에 web/ 없음).
- 일부러 안 한 것: REST `redactSupplier`(`s.AnnualSpend=0`)·대시보드 KPI·`analyze_spend` 의 도구 접근 권한. 웹 `types.ts` 가 `annualSpend: number` 라 응답 모양 결정이 선행돼야 하고, 운영자 지시대로 두 문을 합치지 않았다.
- 다음 역할 주의: 신규 `mcp_spend_null_integration_test.go` 는 DB 가 있어야 돈다(DSN 3개). `spendByName`(기존 헬퍼)은 `.(float64)` 로 null 을 0 으로 읽으므로 이 결함을 증명하지 못한다 — 새 헬퍼 `spendValueByName` 이 `map[string]any` 로 nil 을 직접 본다. 컨테이너 정리: `docker rm -f vendra-1017-improve-pg`(패턴 pkill 금지).
- [러너 03:36] brief accepted — 채택 — 결함 근거가 코드와 좌표까지 그대로 맞았고(`:542`/`:591` 의 `ELSE 0`, `:695`/`:194` 의 `END`, `supplierSummaryRows` 의 호출처 
- [러너 03:37] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 승인. 독립 재검증: 새 컨테이너(vendra-1017-review-pg, 55511)·DSN 3개로 `go test ./internal/... ./cmd/...` 전부 ok(httpapi 30.062s), SKIP 정확히 5건·이름까지 기존 환경성과 일치, gofmt/vet 깨끗. 컨테이너는 `docker rm -f` 로 정리했다.
- 테스트가 경로를 지나는 것 확인: assets/spend-null-red.log 가 수정 전 `:56`/`:78`/`:89` 네 단정 빨강이고 문 열린 쪽 단정은 같은 실행에서 초록(대조군). `spendValueByName` 이 `map[string]any` 로 nil 을 직접 봐서 `spendByName` 의 붕괴를 피했고, 정찰이 추측으로 남긴 jsonb null 키 보존은 `<nil>` 응답으로 실측됐다. 기존 `mcp_limits_integration_test.go:190` 수정은 관례 이탈이지만 새 조건이 **더 엄격**하다.
- 보안·법무 차단 없음: 인가 코드·`canReadSupplierSpend` 무변경, `$5` 는 바인드 파라미터, 출력에서 값을 빼는 방향이라 권한을 넓히지 않음. 개인정보·의존성·외부 약속 문구 변경 없음.
- 남는 우려(릴리즈 노트·다음 회차): MCP 안에 불일치 하나가 **방향만 바뀌어** 남았다 — `get_supplier` 는 `redactSupplier`(suppliers.go:368) 때문에 여전히 0 을 답한다(변경 전 0:3/null:2 → 변경 후 0:1/null:4). `TestSupplierSpendDoorOpensTheSameOnEveryTool` 가 이를 못 잡는 이유는 `spendByName` 이 null 을 0 으로 읽기 때문이다. REST/웹 `types.ts` 모양 결정과 함께 끝낼 일.
- 못 본 것: 실제 MCP 클라이언트의 null 해석(outputSchema 가 없어 깨질 계약은 없음), 웹 스위트(diff 에 web/ 없음).
- [러너 03:43] review approved — 리뷰 승인 (risk=low)
- [러너 03:43] pr created — https://github.com/hkjang/Vendra/pull/146
- [러너 03:46] ci passed — 검사 2개 모두 success
- [러너 03:46] merge done — 375797d
- [러너 03:50] release published — v0.7.73
- [러너 03:51] assets verified — v0.7.73 자산 1개 (이전 v0.7.72: 1)

# 회차 노트 2026-10-07-042750-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:27] base pinned — main@7cd2c6c
- [러너 04:27] autonomy release — 

## 정찰 노트
- 옵션 가격 합산 오버플로를 골랐다. 두 회차 연속 차선이었고, 읽어 보니 피해가 "불명확한 500" 보다 무겁다 — 합계가 두 번 감싸면(2^62 옵션 4개 = 2^64) price 가 0 이 되어 주문이 201 로 성공하고 원장·감사·timeline 에 거짓 금액이 커밋된다. 그래서 위험을 3→2 로 낮추고 1순위로 올렸다.
- 제친 후보: `err!=nil||RowsAffected()==0` 남은 8곳(증거를 댈 HTTP 경로가 없어 404 회귀까지만 — 이 계열은 소진됐다), REQUIREMENT_PENDING 死상태(제품 결정 선행 + 결제·상태표 동시 변경), openapi 응답 코드 대조(계약 설계가 한 세션을 넘김), AdminPage 승인 조건(브라우저 검증·dist 번들 필요).
- 추측으로 적은 것: ⓐ 2^62 가격 옵션이 `POST /talents` 를 실제로 통과하는지 미확인(코드상 상한이 없고 2026-10-05 가 1<<57 을 성공시킨 전례로 추정) ⓑ `POST /talents` 응답이 옵션 id 를 돌려주는지 미확인 — 둘 다 과제서에 대안(pool 로 직접 SELECT)과 함께 미확인으로 적었다. DB 는 이번에 띄우지 않았다(순수 읽기).
- 구현자가 조심할 것: `rows.Next()` 루프 안에서 `writeError` + `return` 하면 pgx 행 집합이 닫히지 않는다 — 플래그를 세워 `rows.Close()` 뒤에 거절할 것. 같은 루프의 `days += extraDays` 와 `rows.Err()` 누락은 범위 밖으로 뺐다(한 가드씩 증명해야 변이 검증이 섞이지 않는다; 후자는 ideas.json 에 다음 회차 후보로 남겼다).
- [러너 04:32] scout done — 주문 생성의 추가 옵션 가격 합산 int64 오버플로를 400 으로 거절 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- `createOrder` 의 옵션 합산에 상한 검사를 넣어 2^64 합계가 `price == 0` 으로 감싸 `amount: 0` 주문을 201 로 커밋하던 길을 400 `order_amount_too_large` 로 막았다(56ebc35, 프로덕션 1파일). 과제서의 경고대로 루프 안에서 return 하지 않고 플래그를 세워 `rows.Close()` 뒤에 거절한다.
- 확신 없는 곳: `TestIntegrationMailNotificationsLeaveThroughTheRelay` 가 깨끗한 DB 의 `-run Integration` 전체 실행에서 **1회 실패**했다(mail_integration_test.go:160, `digest subject="[Kkiit] 수정 요청 …"` — 두 알림이 한 통으로 묶이지 않음). 내 변경과 무관하다고 판단한 근거는 ⓐ 같은 명령 재실행 93 PASS·FAIL 0 ⓑ 전체 `go test ./cmd/... ./internal/...` 두 차례 통과 ⓒ 새 테스트 3건을 이 테스트 바로 앞에 붙여 `-count=1` 로 3회 연속 통과 ⓓ 새 분기가 `len(optionIDs) > 0` 안에 있고 이 테스트의 주문은 전부 `"options": []` 라 분기 자체를 지나지 않음. 그래도 비평가는 이 테스트를 먼저 보기 바란다 — 종전 기록보다 범위가 넓은(깨끗한 DB 에서도 간헐) 사전 존재 경합일 수 있다.
- 검증 못 한 것: `npm --prefix web test`(프런트 미변경, 이 기계 PATH Node 20 문제는 2026-10-03 기각), `make check`, `make build`.
- 일부러 하지 않은 것: 같은 루프의 `rows.Err()`/Scan 삼킴과 `days += extraDays` 오버플로 — 한 세션에 같은 루프를 두 번 바꾸면 변이 검증이 섞인다. 둘 다 ideas.json 에 남겼다(전자는 다음 회차 유력 후보). 쿠폰·예산·상태표·pay 경로와 마이그레이션은 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트 3건은 DB 가 있어야 돈다(`KKIIT_TEST_DSN` 없으면 SKIP). 판정은 반드시 DROP/CREATE 한 깨끗한 DB 로. 도우미 `publishTalentWithOptions` 는 옵션 uuid 를 `pool` 로 직접 읽는다 — `POST /talents` 응답이 `{id,status}` 뿐이기 때문이고, 옵션은 `active` 여야 `createOrder` 의 `id=ANY` 조회에 걸린다.
- [러너 05:02] brief accepted — 채택 — 과제서가 지정한 파일·수법(루프 안 플래그 → `rows.Close()` 뒤 `writeError`, `math` import 추가)대로 구현하고 수용 기�
- [러너 05:03] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 직접 재현했다: 폐기 PG16 을 DROP/CREATE 한 뒤 HEAD 테스트 + main 의 orders.go 로 `TestIntegrationOrderRejectsOptionTotalPastInt64` 가 `status=201 want=400 body={"amount":0,...}` 로 실패, HEAD 에서는 3건 통과. 원장의 실패 출력과 증상이 일치한다. 가드 산술도 검토 — 세 price 열 모두 `CHECK (>= 0)` 이라 우변 `MaxInt64-price` 가 먼저 감싸지 않고, 비음수 누적이라 행 순서와 무관하게 총합 기준으로만 발동한다.
- 구현자가 의심한 mail 테스트: 깨끗한 DB `-run Integration` 전체 90.9초 통과로 이번엔 재현되지 않았다. diff 가 worker·mail 을 건드리지 않고 해당 테스트가 `queued` 를 전역으로 세는 구조라 사전 존재 간섭으로 본다 — 다음 회차에서 또 보이면 그 쪽을 의심할 것.
- 못 본 것: `npm --prefix web test`·`make check`·`make build`(프런트 미변경이라 생략), rfq.go 의 다른 주문 INSERT 경로(옵션을 쓰지 않음).
- 승인이어도 남는 우려: `reserveBudget`(organizations.go:461) 의 `consumed+payable > amount` 가 같은 계열의 오버플로다(결과는 409 라 조용한 손상은 아님) — ideas.json 의 `rows.Err()` 건과 함께 다음 회차 후보. 테스트 3 의 `due_at::date - created_at::date` 는 자정 교차 시 6 이 될 수 있는 밀리초 폭 flake.
- 릴리즈 단계가 볼 것: 작업 트리에 커밋되지 않은 `internal/ui/dist/*` 산출물이 남아 있다. 브랜치 diff(3파일)에는 없으니 `git add -A` 로 쓸어 담지 말 것.
- [러너 05:09] review approved — 리뷰 승인 (risk=low)
- [러너 05:09] pr created — https://github.com/hkjang/Kkiit/pull/21
- [러너 05:10] ci passed — 검사 없음 — 정책으로 허용
- [러너 05:10] merge done — 56ebc35
- [러너 05:22] release published — v0.4.16
- [러너 05:23] assets verified — v0.4.16 자산 1개 (이전 v0.4.15: 1)

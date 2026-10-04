# 과제서 — 2026-10-04 (base main@98c5081, v0.9.66)

- 과제: `usableEntitlement` 가 `GetContractScope` 조회 오류를 "계약 없음" 과 함께 `continue` 로 묶어, 서버 오류가 403 `contract_scope_inactive` 로 보고되는 문제 수정 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/proxy/dataworks_runtime.go:252-255`(실제로 열어 확인)는
  `contract, found, err := s.db.GetContractScope(ctx, ent.ContractKey)` 다음 줄이
  `if err != nil || !found || contract.ProductKey != product.ProductKey { continue }` 이라,
  **조회 실패를 "그 계약이 없다" 와 똑같이 취급**한다. API 키가 한 상품에 엔타이틀먼트를 둘 이상
  가질 수 있고(:239-245 주석이 그 설계를 명시), 아무 후보도 "사용 가능" 으로 뽑히지 않으면 호출부는
  `ent = candidates[0]`(:70)로 되돌아가 그 후보의 사유로 응답한다 — 그래서 **다른 후보의 DB 읽기
  실패가 403 으로 바뀌어** 운영자는 "계약이 비활성" 이라는 틀린 사유를 본다. 고치면 이 저장소가
  v0.9.66·2026-10-01·2026-10-03 회차에서 세 번 적용한 규칙("조회 실패를 부재로 숨기지 않는다")이
  마지막 런타임 경로에도 적용되고, 같은 실패에 단일 후보 경로가 이미 내는 500
  `contract_lookup_failed`(:100-104)과 응답이 일치한다.

- 수용 기준:
  1) 후보가 둘 이상이고 **아무 후보도 사용 가능하지 않으며** 그중 한 후보의 계약 조회가 실패하면
     `POST /v1/data-products/{key}/query` 가 **500 + code `contract_lookup_failed`** (지금은 403
     `contract_scope_inactive`). 메시지·code 는 :103 과 동일한 것을 재사용하고 새 code 를 만들지 않는다.
  2) **가용성 회귀 없음**: 조회가 실패하는 후보가 목록에서 *앞* 에 있고 그 뒤에 사용 가능한 후보가
     있으면 여전히 **200**(그 사용 가능한 후보의 `contract_key`·`entitlement_id` 로 응답). 즉 오류는
     즉시 반환하지 말고 **"끝까지 돌려 보고, 사용 가능한 후보가 하나도 없을 때만" 반환**해야 한다.
     이 분기가 이 과제의 핵심 설계다(아래 "구현 지침" 참조).
  3) 조회가 전부 정상일 때의 선택 결과·감사 문자열·응답 본문은 **바이트 단위로 불변**(기존
     `DataProductQuery` 계열 테스트가 전부 통과해야 한다).
  4) 테스트는 손으로 만든 대역 없이 **실제 `store.SQLStore`(SQLite) + `NewServer(...).Routes()`** 의
     HTTP 경로만 써서 (1)과 (2)를 증명하고, 프로덕션 파일만 되돌리면 (1)이 다시 실패해야 한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개 + 문서 1개):
  - `internal/proxy/dataworks_runtime.go:246 usableEntitlement` — 시그니처를
    `(store.APIEntitlement, bool, error)` 로 바꾸고, `:252` 의 `err` 를 `!found` /
    `ProductKey` 불일치와 분리한다. 루프 안에서 **첫 오류를 지역 변수에 보관하고 계속 진행**하다가,
    사용 가능한 후보를 찾으면 그 후보와 `nil` 오류를 반환하고, 끝까지 못 찾으면 보관한 오류를 반환한다.
    주석(:239-245)에 "조회 실패는 부재가 아니다 / 사용 가능한 후보가 있으면 오류는 삼키지 않고 버린다"
    는 판단 근거를 한두 줄 덧붙인다.
  - `internal/proxy/dataworks_runtime.go:71` 호출부 —
    `usable, ok, err := s.usableEntitlement(...)` 로 받고 `if err != nil` 이면
    `writeOpenAIError(w, http.StatusInternalServerError, err.Error(), "server_error", "contract_lookup_failed")`
    후 return. **주의**: 이 지점은 :89 의 과금 `defer` 보다 앞이고 `contractKey` 가 아직 "" 이므로
    `IncrementUsageMetering` 은 호출되지 않는다(= 과금·`errCode` 배선을 건드릴 필요 없음). 감사 로그를
    남길지는 구현자 판단이나, 남긴다면 기존 `s.auditDataProductQuery` 형식을 따르고 **식별자만**
    넘긴다(원문 오류 메시지를 감사에 넣지 말 것 — 운영자 규칙).
  - `docs/OPERATIONS.md` — 500 `contract_lookup_failed` 가 후보가 여럿일 때도 나온다는 운영 문단 2~4줄.
    (정본 PDF 없음 — `ls docs/*.pdf` 로 재확인할 것. PDF 재생성 불필요.)
  - 새 테스트는 `internal/proxy/` 의 기존 네이밍을 따라 추가(예:
    `dataworks_runtime_entitlement_test.go` 또는 기존 DataProductQuery 테스트 파일에 함수 추가).

- 실패를 재현하는 법 (**한 행만 읽기 실패**하게 만들어야 한다 — 테이블 rename 은 쓸 수 없다):
  - 테이블 전체를 rename 하면 `candidates[0]` 의 조회도 실패해 `:100` 이 이미 500 을 내므로
    **결함이 재현되지 않는다**. 그래서 한 contract_key 만 읽기 실패해야 한다.
  - **1순위 레버(확실, affinity 무관)**: `dw_contract_scopes` 를 `dw_contract_scopes_real` 로 rename 하고
    같은 이름의 **뷰**를 만들어 깨뜨릴 행의 `rate_limit` 만 텍스트로 바꾼다. `GetContractScope` 는
    `rate_limit` 을 `int`(store/dataworks_operations.go:45 `RateLimit int`, scan 은 :379-381)로 스캔하므로
    텍스트 값이면 `rows.Scan` 이 오류를 반환한다. 뷰에는 affinity 가 없어 값이 TEXT 로 그대로 나온다:
    ```sql
    ALTER TABLE dw_contract_scopes RENAME TO dw_contract_scopes_real;
    CREATE VIEW dw_contract_scopes AS SELECT contract_key, product_key, customer_key, allowed_fields,
      CASE WHEN contract_key = 'ctr-broken' THEN 'not-a-number' ELSE rate_limit END AS rate_limit,
      valid_from, valid_to, purpose, restrictions, status, created_by, created_at, updated_at, masking_policy
      FROM dw_contract_scopes_real;
    ```
    `SELECT` 열 이름 14개는 `GetContractScope`(:363-365)가 뽑는 그대로여야 한다. cleanup 에서 뷰를 DROP
    하고 테이블을 원래 이름으로 되돌린 뒤 **같은 요청이 다시 200/403 기준선으로 돌아오는지** 단언한다.
    쓰기는 뷰 설치 **전** 에 프로덕션 admin 라우트로 끝내 둘 것(뷰에는 INSERT 가 안 된다).
  - **2순위 레버(미확인)**: 뷰 없이 `UPDATE dw_contract_scopes SET rate_limit='not-a-number' WHERE …`.
    컬럼이 `rate_limit INTEGER NOT NULL DEFAULT 0`(store/sqlstore.go:1964, 3473)이라 SQLite 의 INTEGER
    affinity 가 숫자 아닌 텍스트는 TEXT 로 저장할 것으로 보이지만 **이 환경에서 실행으로 확인하지 못했다**
    (정찰 세션에서 `sqlite3` CLI 실행 권한이 없었다). 이쪽이 되면 더 단순하니 먼저 1분만 시험해 보고,
    값이 0 으로 강제되면 1순위 레버로 갈 것.
  - **후보 순서 고정**: `ListAPIEntitlementCandidates`(store/dataworks_operations.go:477-498)는
    `ORDER BY updated_at DESC LIMIT 50` 이다. 두 엔타이틀먼트를 admin 라우트로 만든 뒤 raw SQL 로
    `updated_at` 을 서로 다른 값으로 UPDATE 해 순서를 **결정적으로** 만들 것(같은 시각이면 순서 미정).
  - 두 사례를 만들 것:
    - **사례 A(수용 기준 1)**: 깨진 계약을 가리키는 후보 + 창이 이미 닫힌(`valid_to` 과거) 계약을 가리키는
      후보. 후자가 `candidates[0]` 이 되게 순서를 잡는다 → 수정 전 **403 `contract_scope_inactive`**,
      수정 후 **500 `contract_lookup_failed`**.
    - **사례 B(수용 기준 2)**: 깨진 계약 후보가 **앞**, 완전히 유효한 계약 후보가 **뒤** → 수정 전후 모두
      **200** 이고 응답의 `contract_key`·`entitlement_id` 가 유효한 쪽. (오류를 즉시 반환하는 구현이면
      이 사례가 500 으로 깨진다 — 설계 선택을 잡아 주는 테스트다.)

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go build ./...`
  - `go vet ./...`
  - `go test ./internal/proxy -run 'DataProductQuery|Entitlement' -count=1` (proxy 패키지 전체는 35~40초)
  - `go test ./... -count=1` (전체는 분 단위; proxy ~38s, store ~15s)
  - `go run ./cmd/api-surface-audit` — 누락 0 이어야 함(550 routes / 612 OpenAPI paths). 새 라우트·code 를
    만들지 않으므로 수치가 변하면 안 된다.
  - `gofmt -l internal/proxy/dataworks_runtime.go <새 테스트 파일>` — 출력 없어야 함. 다만 이 저장소의
    기존 CRLF 때문에 HEAD 에서도 파일명이 나오는 사례가 있으니 **줄 끝은 건드리지 말 것**.
  - 프로덕션 파일만 `git checkout` 으로 되돌려 사례 A 가 다시 실패하는지 확인한 뒤 복원.
  - 웹 검증(`npm` 계열)은 **하지 말 것**: web 변경이 없고, 이 워크트리에는 `web/node_modules` 가 없어
    설치 없이 돌리면 exit 127 이다.

- 위험과 피할 것:
  - **수용 기준 2 를 깨는 구현(오류 즉시 반환)이 가장 흔한 함정이다.** :66-69 주석이 명시한 설계
    ("다른 후보가 유효하면 그 후보로 서비스한다")를 되돌리지 말 것.
  - `!found` 와 `contract.ProductKey != product.ProductKey` 는 **지금처럼 조용히 continue** 로 남겨야
    한다(그 둘은 진짜 "이 후보는 못 쓴다" 다). 오류만 분리하는 것이 이 과제의 전부다.
  - `entitlementActive`·`entitlementAllowsQuery`·`contractScopeActive`·`contractScopeCanServe` 의 판정
    로직은 손대지 말 것. 파서를 통합하거나 트림 규칙을 바꾸지 말 것(판정에는 트림 값, 저장·응답에는 원문).
  - `internal/store` 쪽은 수정하지 말 것(마이그레이션 위험 구역). 재현은 테스트 안의 raw SQL 로만.
  - 보호 경로 금지: `internal/proxy/keycloak*.go`·`mcp_oauth.go`, `.github/workflows/`, `web/` 전체.
    특히 `web/` 인프라 변경은 최근 세 PR(1d63b7c·5ed6876·f927cea)이 main 에 들어오지 않았다 — 건드리지 말 것.
  - 과금 `defer`(:89-98)와 `errCode` 흐름을 재배치하지 말 것. 새 500 은 그 defer 가 설치되기 전에 반환된다.

- 차선 후보: **액션 센터·런타임이 보는 "계약 비활성" 사유를 감사 문자열로 구분하기** 가 아니라,
  `dataWorksTimestampOK` 를 통과한 타임스탬프를 네 읽기 경로(`contractScopeActive` / 액션 센터 루프 /
  `store.EntitlementActive` / `bestApprovalStatus`)가 같은 값으로 읽는지 end-to-end 계약 테스트로 고정
  (2/1/S, 프로덕션 0파일). 단, 즉시 통과하는 테스트는 결함을 증명하지 못하므로 **먼저 네 경로가 어긋나는
  입력을 찾고**(예: 오프셋 표기·`Z` 없는 값·나노초 자릿수), 어긋나는 것이 하나도 없으면 이 차선도 버리고
  "퍼블리시 게이트 조립의 선택적 조회 오류를 '증거 없음' 과 구분"(admin_dataworks.go:1750·1758·1769·1775,
  2/3/M)으로 갈 것.

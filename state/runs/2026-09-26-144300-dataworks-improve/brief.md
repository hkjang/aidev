- 과제: 액션 센터가 `valid_from` 이 깨져 모든 조회를 막는 계약도 `contract_expiring` 으로 보고하게 하기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `contractScopeActive`(internal/proxy/dataworks_runtime.go:317~330)는 `valid_from` 이 비어 있지 않은데 `time.Parse(time.RFC3339Nano, …)` 로 읽히지 않으면 `false` 를 돌려주므로, 그 계약의 모든 런타임 조회가 영구히 `403 contract_scope_inactive` 가 된다. 그런데 액션 센터의 계약 루프(internal/proxy/admin_dataworks.go:251~281)는 `valid_to` 만 보고 `valid_to` 가 비어 있으면 `continue` 하므로, 이 "죽은 계약" 은 운영 화면 어디에도 뜨지 않는다 — 같은 실패(창을 해석할 수 없어 모든 조회가 막힘)를 `valid_to` 쪽에서는 이미 severity `high` 인 `contract_expiring` 으로 보고하고 있다(그 코드의 주석 270~272, docs/OPERATIONS.md:216). 고치면 운영자가 고객 문의를 받기 전에 화면에서 원인을 본다.

- 수용 기준:
  1) `status` 가 활성(`contractScopeStatusActive`)이고 `valid_from` 이 공백 제거 후 비어 있지 않으며 `time.Parse(time.RFC3339Nano, …)` 로 읽히지 않는 Contract Scope 는, `valid_to` 가 비어 있든 먼 미래든 상관없이 액션 센터 응답의 `actions` 에 `type: "contract_expiring"`, `severity: "high"` 로 나오고 `summary.expiring_contracts` 가 1 증가한다.
  2) 한 계약이 두 개의 액션으로 중복되어 나오지 않는다. `valid_to` 도 `valid_from` 도 읽을 수 없는 계약은 `contract_expiring` 이 **정확히 1건**이고 `summary.expiring_contracts` 도 1만 증가한다.
  3) 기존 동작이 하나도 바뀌지 않는다: 활성이 아닌 계약(draft/suspended/revoked)은 `valid_from` 이 깨져 있어도 여전히 보고하지 않는다. `valid_from` 이 정상(과거/미래/빈 값/공백만)인 계약의 판정·severity·개수는 종전과 같다. `valid_from` 이 **미래**(창이 아직 안 열림)인 경우는 이번 과제에서 **건드리지 않는다**(별도 보류 아이디어).
  4) 응답에 실리는 `valid_to`(그리고 새로 싣는다면 `valid_from`)는 저장 원문 그대로다 — trim 한 값을 응답에 쓰지 않는다. 저장된 행도 바뀌지 않는다.
  5) 테스트가 증명할 것: 수정 전 코드(변경을 되돌린 상태)에서 "활성 + `valid_from` = `2026-01-01` + `valid_to` 빈 값" 계약에 대해 런타임 조회는 `403 contract_scope_inactive` 인데 액션 센터는 `expiring_contracts: 0` 이라는 것, 수정 후에는 같은 픽스처가 `expiring_contracts: 1` + severity `high` 가 된다는 것.

- 건드릴 파일:
  - `internal/proxy/admin_dataworks.go` — `handleDataWorksActionCenter` 의 계약 루프(251~281행). 현재 구조는
    `validTo == "" → continue` / `err == nil && !expiresAt.Before(deadline) → continue` 두 개의 continue 뒤에 append 가 오는데, 이대로는 `valid_from` 조건을 끼워 넣을 자리가 없다. 각 scope 마다 "보고할지 / severity 가 무엇인지" 를 먼저 계산하고 append 는 한 번만 하도록 루프 본문을 정리한 뒤, `valid_from` 이 비어 있지 않고 `time.Parse(time.RFC3339Nano, strings.TrimSpace(scope.ValidFrom))` 가 실패하면 보고 + severity `high` 를 켜는 조건을 더한다. 계약당 append 1회를 유지할 것(수용 기준 2). 액션 맵에 `"valid_from": scope.ValidFrom`(원문) 을 추가한다.
  - `internal/proxy/admin_dataworks_action_center_test.go` — 실제 `store.SQLStore` + `NewServer(...).Routes()` 를 쓰는 기존 HTTP 회귀 테스트 파일(이미 존재, 159행 부근에 같은 계열 주석이 있다). 여기에 표 사례를 더한다: (a) 활성 + `valid_from="2026-01-01"` + `valid_to` 빈 값 → 액션 1건·high, 같은 계약/API 키로 런타임 조회는 403 `contract_scope_inactive`, (b) 활성 + `valid_from="2026-01-01"` + `valid_to` 도 `"2026-12-31"` → 액션 **1건**만, (c) `status="revoked"` + 깨진 `valid_from` → 액션 0건, (d) 활성 + 공백 낀 정상 `valid_from`(` 2020-01-01T00:00:00Z `) + 먼 미래 `valid_to` → 종전대로 액션 0건이고 런타임 조회 200. 각 사례에서 `GET …/contract-scopes` 의 `valid_from`·`valid_to` 원문이 그대로인지 함께 단언한다.
  - `docs/OPERATIONS.md` — 5절, 200~217행 부근에 `valid_to` 의 해석 불가 → `high` 규칙을 설명하는 문단이 이미 있다. `valid_from` 도 같은 규칙으로 보고된다는 2~3줄을 같은 자리에 더한다(이 문서는 PDF 정본이 없다 — `ls docs/*.pdf` 로 확인할 것).
  - 프로덕션 파일은 Go 1개 + 문서 1개. 웹은 손대지 않는다(새 action type·새 summary key 를 만들지 않으므로 `web/src/lib/labels.ko.ts`·`secondary-pages.tsx`·`types/dataworks.ts` 가 그대로 동작한다).

- 검증 명령:
  - `go test ./internal/proxy/ -run ActionCenter -v` (먼저 이것으로 반복)
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...` (internal/proxy 는 30~40초대)
  - `go run ./cmd/api-surface-audit` — gap 0 유지(라우트를 더하지 않으므로 바뀌면 안 된다)
  - `gofmt -l internal/proxy/admin_dataworks.go` — 이 파일은 CRLF 가 아니므로 출력이 없어야 한다(주의: 같은 패키지의 `dataworks_runtime.go` 는 저장소의 기존 CRLF 때문에 HEAD 에서도 파일명을 출력한다. 줄 끝은 건드리지 말 것)
  - web 을 바꾸지 않았다면 `npm` 계열 검증은 불필요

- 위험과 피할 것:
  - **런타임 판정(`contractScopeActive`)을 고치지 말 것.** 이번 과제는 순수하게 "이미 막혀 있는 상태를 화면에 보고" 하는 것이다. 런타임 게이트를 바꾸면 접근 제어 변경이 되어 범위를 벗어난다.
  - **`contractScopeCanServe`(dataworks_runtime.go:271)를 고치지 말 것.** 해석 불가 `valid_from` 에 대해 `true` 를 돌려주는 것이 `dataworks_masking_policy_test.go:252` 의 `{"unparseable valid_from", …, canServe: true}` 사례로 이미 고정돼 있고, 이 값은 publish gate 의 masking 증거(admin_dataworks.go:1738)를 **막는** 쪽으로 쓰이므로 `false` 로 바꾸면 민감 상품의 게이트를 느슨하게 만든다. 이번 회차에 이 방향은 검토 후 기각했다.
  - **새 action type·새 summary key 를 만들지 말 것.** 만들면 `web/src/lib/labels.ko.ts`, `web/src/pages/secondary/secondary-pages.tsx:199` 의 `typeMap`, `web/src/pages/home/home-page.tsx:41`, `web/src/types/dataworks.ts:122` 까지 같이 바꿔야 해서 파일 수가 6개를 넘고 한 세션에 끝나지 않는다. 기존 `contract_expiring` / `expiring_contracts` 를 재사용한다.
  - **중복 집계 주의.** 루프를 재구성할 때 `valid_to` 조건과 `valid_from` 조건이 각각 append 하면 한 계약이 두 줄로 나오고 `summary.expiring_contracts` 도 2가 된다(수용 기준 2가 이것을 잡는다).
  - **원문 보존.** 이 저장소는 trim 한 값을 판정에만 쓰고 저장·응답에는 원문을 쓴다(1e9045a, 36c665c). 같은 규칙을 지킬 것.
  - **손으로 만든 대역을 쓰지 말 것.** 기존 액션 센터 테스트처럼 실제 `store.SQLStore` 와 `NewServer(...).Routes()` 로 HTTP 를 태워 증명하고, 소스 문자열 grep 을 증거로 제출하지 말 것.
  - **수정을 되돌린 상태에서 새 테스트가 실제로 실패하는지 확인할 것**(이 저장소의 직전 4회차가 모두 그렇게 검증해 채택됐다).
  - 보호 경로(`internal/proxy/keycloak*.go`, `mcp_oauth.go`, `internal/store` 마이그레이션, `.github/workflows/ci.yml`, `web/embed.go`)는 건드리지 않는다.
  - 릴리즈 커밋은 남기지 않는다.

- 차선 후보: **액션 센터가 아직 열리지 않은 계약 창(`valid_from` 이 미래)을 보고하지 않는 문제** (가치 2 / 위험 2 / 작업량 M) — `contractScopeActive` 는 `validFrom.After(now)` 면 모든 조회를 403 `contract_scope_inactive` 로 막지만, 이는 오류가 아니라 "예정된 계약" 이므로 `contract_expiring` 으로 보고하면 의미가 어긋난다. 새 action type(`contract_not_yet_active`)과 summary 키·웹 문구·문서를 함께 바꿔야 하므로 1순위가 성립하지 않을 때만 고르고, 고른다면 웹 4파일을 포함해 범위를 다시 잡을 것.

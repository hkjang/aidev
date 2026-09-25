- 과제: 런타임 계약 게이트가 `valid_from` 을 `valid_to` 와 같은 규칙으로 읽도록 수정 (가치 3 / 위험 1 / 작업량 S)
- 왜: `contractScopeActive`(internal/proxy/dataworks_runtime.go:317)는 `strings.TrimSpace(scope.ValidFrom) != ""` 로 빈 값만 판정한 뒤 **트림하지 않은** 원문을 `time.Parse(time.RFC3339Nano, scope.ValidFrom)` 에 넘긴다. 같은 함수가 호출하는 `contractScopeCanServe`(같은 파일 271~283)는 `valid_to` 를 `strings.TrimSpace` 한 값으로 파싱하고, `store.EntitlementActive`(internal/store/dataworks_operations.go:541)와 액션 센터(internal/proxy/admin_dataworks.go:262·1e9045a)도 트림 후 파싱한다 — 이 한 곳만 규칙이 다르다.
- 왜(계속): 그래서 `valid_from` 에 공백이 섞인 계약 행은 창이 이미 열려 있고 `valid_to` 가 먼 미래여도 파싱 실패로 `contractScopeActive` 가 false 가 되어 모든 상품 조회가 `403 contract_scope_inactive` 로 막히고(`usableEntitlement`, dataworks_runtime.go:256 도 그 계약을 건너뛴다), 액션 센터의 `contract_expiring` 루프는 `valid_to` 만 보므로 아무 경고도 내지 않아 운영자에게 신호가 없다.

- 수용 기준:
  1) `valid_from` 이 ` 2020-01-01T00:00:00Z `(앞뒤 공백)이고 `valid_to` 가 미래인 활성 계약 + 활성 엔타이틀먼트로 런타임 상품 조회가 **200** 을 반환한다(수정 전에는 `403 contract_scope_inactive`).
  2) 아직 열리지 않은 창은 그대로 막힌다: `valid_from` 이 미래(공백 포함/미포함 모두)면 여전히 `403 contract_scope_inactive`. 해석 불가한 `valid_from`(예: `not-a-time`)도 여전히 거부된다(fail-closed 유지).
  3) 저장된 원문과 API 응답의 `valid_from` 문자열은 바뀌지 않는다(관리 API `GET …/contract-scopes` 가 공백 포함 원문을 그대로 돌려준다) — 트림은 판정에만 쓴다.
  4) 테스트: 트림을 되돌리면 1) 의 사례가 실제로 403 으로 실패하고 2) 의 사례는 계속 통과함을 구현자가 확인한다(변이 확인).

- 건드릴 파일:
  - `internal/proxy/dataworks_runtime.go:317 contractScopeActive` — `raw := strings.TrimSpace(scope.ValidFrom)` 로 한 번 트림한 지역 변수를 만들어 빈 값 검사와 `time.Parse` 에 같이 쓴다(바로 위 `contractScopeCanServe` 의 `if raw := strings.TrimSpace(scope.ValidTo); raw != ""` 형태를 그대로 따를 것). 주석 한 줄로 "런타임·액션 센터·EntitlementActive 와 같은 규칙" 을 남길 것. `scope` 는 값 전달이므로 호출자가 보는 문자열은 변하지 않는다 — 구조체 필드에 대입하지 말고 지역 변수만 쓸 것.
  - 회귀 테스트: `internal/proxy/admin_dataworks_action_center_test.go` 의 `TestDataWorksActionCenterMatchesRuntimeContractExpiry`(307행~) 를 형판으로 표 테스트를 하나 더 추가할 것(예: `TestRuntimeContractGateReadsValidFromLikeValidTo`). 이 테스트는 이미 필요한 배선을 전부 갖고 있다 — `newAccessWindowTestServer(t)`(`internal/proxy/admin_dataworks_access_window_test.go:15`, 실제 `store.SQLStore` + `NewServer(...).Routes()` + `dw_credit_score` 상품), 공백 픽스처 헬퍼 `paddedDate(days)`(` \t` + RFC3339Nano + `\r\n `), `db.UpsertContractScope`/`db.UpsertAPIKey`(`hashProxyKey("bank-secret")`)/`db.UpsertAPIEntitlement`, 런타임 호출 `postJSON(t, srv.URL+"/v1/data-products/dw_credit_score/query", "bank-secret", map[string]any{"fields": []string{"score"}})`, 원문 보존 확인 `db.GetContractScope`. **기존 테스트의 계약에는 `ValidFrom` 이 비어 있으므로** 새 표는 `ValidFrom` 을 축으로 삼고 `ValidTo` 는 먼 미래(또는 빈 값)로 고정할 것. 표 사례 권장: 공백 포함 과거(200) / 공백 없는 과거(200) / 빈 값(200) / 공백만(200) / 공백 포함 미래(403) / 공백 없는 미래(403) / ` not-a-date `(403). 각 사례에서 `db.GetContractScope` 로 `valid_from` 원문 불변도 단언할 것.
  - 문서는 필요하면 `docs/OPERATIONS.md` 5절(만료 예정 계약·권한 확인)에 한두 줄. PDF 없는 문서이므로 PDF 재생성 불필요(`ls docs/*.pdf` 로 확인 가능).

- 공백 행을 만드는 방법(중요 — 대역을 쓰지 말 것): 관리 HTTP POST 경로는 `handleDataWorksContractScopes`(admin_dataworks.go:1356~1358)가 `valid_from`·`valid_to` 를 **이미 트림해서 저장**하므로 HTTP 로는 공백 행을 만들 수 없다. 프로덕션 store 메서드 `store.(*SQLStore).UpsertContractScope`(internal/store/dataworks_operations.go:301)는 `ContractKey`·`ProductKey`·`Status` 만 트림하고 `valid_from`·`valid_to` 는 원문 그대로 INSERT 하므로, 테스트에서 실제 store 로 공백 포함 행을 심고(레거시·직접 DB 쓰기와 같은 상태) **그 뒤의 검증은 실제 HTTP 라우트로** 하면 된다. FakeStore·직접 구조체 주입은 쓰지 말 것.

- 검증 명령:
  - `cd /home/hkjang/.cache/auto-improve-wt/dataworks`
  - `go test ./internal/proxy/ -run 'ContractScope|AccessWindow|ValidFrom|RuntimeContractGate' -count=1 -v` (이 정찰에서 `-run 'ContractScope|AccessWindow'` 를 실제로 돌려 0.857초에 ok 확인. proxy 패키지 전체는 약 34초)
  - `go build ./... && go vet ./... && go test ./... -count=1`
  - `go run ./cmd/api-surface-audit` (gap 0 이어야 함)
  - `gofmt -l internal/proxy/dataworks_runtime.go` (출력 없음). 주의: 저장소에 CRLF 파일이 섞여 있다 — `dataworks_runtime.go` 의 줄 끝을 바꾸지 말 것.
  - web 변경이 없으면 웹 체크는 불필요(과거 회차 관례). `web/dist/.gitkeep` 은 v0.9.57 의 vite 플러그인이 지키므로 손대지 말 것.

- 위험과 피할 것:
  - 판정만 바꾸고 **저장·JSON 원문은 유지**할 것(1e9045a 회차와 같은 규칙). 마이그레이션으로 기존 행을 일괄 트림하지 말 것 — `internal/store` 마이그레이션은 보호 경로다.
  - `contractScopeCanServe`·`contractScopeStatusActive`·`store.EntitlementActive` 는 이미 트림하므로 **건드리지 말 것**. 파서를 하나로 합치려 하지 말 것(계약이 다른 파서를 통합하지 말라는 운영자 지시).
  - `handleDataWorksContractScopes` 의 400 검증(`dataWorksTimestampOK`·`dataWorksAccessWindowOrdered`)을 느슨하게 하지 말 것 — 새 쓰기는 지금처럼 트림 후 엄격 검증이 맞다.
  - auth/session(`internal/proxy/keycloak*.go`, `mcp_oauth.go`), `.github/workflows/ci.yml`, `web/embed.go` 는 건드리지 말 것.
  - 도달 경로에 대한 정직한 표기: 현재 HTTP 쓰기 경로로는 공백 행이 생기지 않는다(레거시 행·직접 DB 쓰기·`store.UpsertContractScope` 직접 호출에서만 생긴다). 이 저장소에는 정규화 이전 행이 실재한다는 근거가 코드 주석에 있다(admin_dataworks.go:284~287 의 `" active "` 설명). 이 사실을 커밋 메시지·PR 본문에 그대로 적고 "원격에서 흔히 발생" 처럼 부풀리지 말 것.

- 차선 후보: 액션 센터가 아직 열리지 않은 계약 창(`valid_from` 이 미래)을 경고로 내지 않는 문제 — `contractScopeActive` 는 그 계약으로 오는 모든 조회를 `403 contract_scope_inactive` 로 막는데 `admin_dataworks.go` 의 `contract_expiring` 루프(약 252~281행)는 `valid_to` 만 보므로 화면에 아무 것도 뜨지 않는다. 새 action type(`contract_not_yet_active` 등)을 더하면 요약 키·웹 문구·문서를 함께 바꿔야 하므로 1순위보다 작업량이 크다(가치 2 / 위험 2 / 작업량 M).

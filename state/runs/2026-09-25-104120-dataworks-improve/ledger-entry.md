## 2026-09-25
- 선택: 런타임 계약 게이트가 `valid_from` 을 `valid_to` 와 같은 규칙으로 읽도록 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `contractScopeActive`(dataworks_runtime.go:317)는 `TrimSpace(scope.ValidFrom)` 로 빈 값만 검사한 뒤 트림하지 않은 원문을 `time.Parse` 에 넘겨, 공백이 섞인 `valid_from` 행은 창이 이미 열려 있고 `valid_to` 가 먼 미래여도 파싱 실패로 모든 조회가 `403 contract_scope_inactive` 가 되었다(같은 함수의 `valid_to`·액션 센터·`store.EntitlementActive` 는 모두 트림 후 파싱). 한 번 트림한 지역 변수를 빈 값 검사와 파싱에 함께 쓰도록 고쳤고 `scope` 는 값 전달·필드 미대입이므로 저장 원문과 API 응답은 그대로다. 실제 `store.SQLStore` + `NewServer(...).Routes()` 로 7사례 표 테스트(공백 과거·공백없는 과거·빈 값·공백만 → 200, 공백 미래·공백없는 미래·`not-a-date` → 403 + `contract_scope_inactive`, 각 사례마다 `GetContractScope` 와 `GET …/contract-scopes` 원문 불변)를 추가했고, 트림을 되돌린 상태(수정 전)에서 공백 과거 사례만 403 으로 실패하고 나머지 6사례는 통과함을 실행으로 확인했다. `go build ./...`·`go vet ./...`·`go test ./...` 전체 통과(proxy 36.080s), `go run ./cmd/api-surface-audit` gap 0(550 routes / 612 OpenAPI paths), `docs/OPERATIONS.md` 5절 3줄 추가(PDF 없음). `gofmt -l internal/proxy/dataworks_runtime.go` 는 저장소의 기존 CRLF 때문에 HEAD 에서도 파일명을 출력한다 — CRLF 를 제거한 사본은 클린임을 확인했고 줄 끝은 건드리지 않았다.
- 보류 아이디어:
  - 액션 센터가 아직 열리지 않은 계약 창(`valid_from` 미래)을 경고로 내지 않음 (가치 2 / 위험 2 / 작업량 M)
  - 상품이 사라진 고아 Contract Scope·Entitlement 를 전용 운영 경고로 분류 (가치 2 / 위험 2 / 작업량 M)
  - 동일 API 키에 활성 엔타이틀먼트가 둘 이상일 때 운영 화면 경고 (가치 2 / 위험 2 / 작업량 S)
  - 이미 지난 `expires_at` 으로 발급되는 죽은 엔타이틀먼트를 쓰기 경로에서 거부 (가치 2 / 위험 2 / 작업량 S)
- 과제서: 채택 — `valid_from` 만 트림 규칙이 다르다는 근거가 현재 코드와 정확히 일치했고 수용 기준 1~4 를 모두 실행으로 확인했다(다만 과제서의 `gofmt -l` "출력 없음" 기대는 CRLF 파일에선 HEAD 에서도 성립하지 않는다).

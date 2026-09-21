- 과제: MCP가 공개한 limit 인자를 세 공급업체 조회 도구에서 실제로 적용하기 (가치 3 / 위험 2 / 작업량 M)
- 왜: `internal/httpapi/integrations.go:mcpTools`는 search_suppliers·analyze_spend·recommend_suppliers에 limit을 공개하지만 `runMCPTool`의 SQL은 각각 LIMIT 100·100·50으로 고정되어 한 건을 요청해도 여러 건을 반환한다. 공개된 인자를 실제 조회에 연결하면 AI 클라이언트가 필요한 크기로 답변을 받고 불필요한 컨텍스트 사용을 줄일 수 있다.
- 수용 기준:
  1) 세 도구 모두 유효한 양의 정수 limit을 반영하며 기존 정렬상 앞선 N건을 반환한다. search_suppliers/analyze_spend는 기본값·최댓값 100, recommend_suppliers는 기본값·최댓값 50을 유지한다. 생략·null·문자열·0·음수는 기본값, 상한 초과는 상한으로 처리한다. recommend_suppliers의 tools/list 스키마에도 minimum=1, maximum=50을 명시한다. 기존 두 스키마의 1~100 범위는 유지한다.
  2) 스코프·소프트 삭제·추천의 상태/category/minScore 조건·금액 노출·기존 정렬은 그대로다. analyze_spend의 share는 limit 이전에 접근 가능한 전체 공급업체 지출을 분모로 삼아야 한다. limit=1로 바꿔도 같은 공급업체의 share는 기본 요청 때와 같아야 하며, SQL 윈도 함수보다 먼저 자르는 서브쿼리를 만들지 않는다.
  3) PostgreSQL과 실제 로그인 세션→App.Handler→POST /mcp→tools/call을 통과하는 회귀 테스트로 세 도구 모두 수정 전 limit=1 실패, 수정 후 통과를 확인한다. 각 도구에 대해 limit=1·2, 생략, 상한, 상한 초과를 확인하고 기본값/상한 검증은 적격 공급업체 101개 이상을 시드하여 실제 잘림을 증명한다. JSON structuredContent와 content[0].text를 각각 파싱해 동일한 행·ID·순서인지 검사한다. 다른 부서/own 범위 밖/삭제된 업체가 앞순위에 있어도 노출되거나 적격 N건을 밀어내지 않는지 확인한다. recommendation fixture에는 active·비CRITICAL·서로 다른 score를 사용한다.
  4) 숫자를 SQL LIMIT에 연결하며 0.5 또는 1e100 같은 JSON 수치가 LIMIT 0/음수/내부 오류로 새지 않도록 한다. 기존 intNumber를 재사용한다면 정수 변환 전에 상한을 검사하고 1 미만 양수는 기본값으로 처리한다(1 이상 소수는 기존 절삭 방식을 유지). 실제 MCP 요청에서도 0.5·1e100을 보내 기본값·상한 행 수와 오류 없음으로 증명한다. helper 변경 시 기존 get_expiring_contracts의 기본 180·상한 3650 및 정상 days 동작은 유지한다.
- 건드릴 파일:
  - `internal/httpapi/integrations.go:runMCPTool` — search_suppliers는 LIMIT $6, analyze_spend는 LIMIT $4, recommend_suppliers는 LIMIT $7에 검증된 정수를 바인딩한다. 해당 인자 위치는 현재 쿼리 기준이며 기존 파라미터 순서를 보존한다.
  - `internal/httpapi/integrations.go:mcpTools` — 추천 도구 limit 스키마의 범위를 실제 상한과 일치시킨다. `intNumber` — 위 경계 처리에 필요한 최소 수정만 허용; 다른 숫자 파서나 days 정책은 바꾸지 않는다.
  - `internal/httpapi/mcptools_test.go:callMCPTool, toolRows, TestIntNumberStaysInRange` — 기존 helper 활용 및 숫자 경계 회귀. toolRows는 structuredContent만 반환하므로 텍스트 결과 검사는 별도로 추가해야 한다.
  - `internal/httpapi/mcp_limits_integration_test.go` (신규 제안) — `TestMCPAdvertisedLimitsAreApplied`라는 실제 DB/API 회귀 테스트. `internal/httpapi/datascope_test.go:newScopeWorld, wipe`와 `internal/httpapi/login_integration_test.go:newTestApp`를 활용하되 이 공용 fixture 구현은 가능하면 변경하지 않는다.
- 검증 명령:
  - 저장소 루트에서 `go test ./internal/httpapi -run 'Test(MCPAdvertisedLimitsAreApplied|IntNumberStaysInRange|ExpiringContractsToolReturnsContracts|MCPToolAnswersAreBounded|MCPToolResultsAreBounded|MCPSupplierNumberSearchMatchesREST)$' -count=1 -v` — 반드시 구현자 전용 PostgreSQL의 VENDRA_TEST_DSN을 먼저 설정하고 신규/기존 DB 테스트가 SKIP되지 않았음을 확인한다. 신규 테스트명은 위 제안대로 작성한다.
  - `go test ./internal/... ./cmd/... -count=1` — CI와 같은 검증은 서로 다른 전용 DB 세 개에 VENDRA_TEST_DSN, VENDRA_TEST_MIGRATE_DSN, VENDRA_TEST_UPGRADE_DSN을 설정한다. 뒤 두 DB는 빈 DB에서 시작해야 한다.
  - `go vet ./internal/... ./cmd/...`, `gofmt -l internal cmd`, `git diff --check`.
  - 정찰 실행 결과: DSN 세 개가 없는 상태로 전체 Go 테스트 통과(httpapi 1.459초, cmd 테스트 없음). 따라서 DB 회귀 및 위 신규 테스트는 정찰에서 실행·재현하지 않았음. Docker ps는 동작하지만 Vendra 전용 DB는 준비하지 않았음. 타 프로젝트 컨테이너를 재사용하거나 수정하지 말 것.
- 위험과 피할 것: auth·session·migrations·.github/workflows·프런트·가이드 PDF는 범위 밖이다. 과거 반려된 통화 합산/재해석 접근은 반복하지 말고 annualSpend 및 share 계산식도 변경하지 않는다. 결과를 Go에서 모두 읽은 뒤 자르지 말고 기존 WHERE/ORDER BY 뒤 SQL LIMIT만 매개변수화한다. 추천 상한을 100으로 확대하거나 limit 없는 다른 도구에 새 인자를 추가하지 않는다. 테스트용 공급업체 번호는 SC- 접두사를 유지하고 종료 시 context.Background()로 wipe를 호출하며 테스트를 병렬화하지 않는다. 공용 fixture의 이름/사업자번호/번호가 같은 점은 기존 번호 검색 회귀를 흐릴 수 있으므로 기존 테스트는 유지한다. 스코프 및 share 검증은 권한 밖 업체의 큰 지출이 분모에 섞이지 않는 것도 확인한다.
- 차선 후보: 사용자 가이드 MCP 도구표와 실제 tools/list 응답을 양방향 비교하는 회귀 테스트 (가치 2 / 위험 1 / 작업량 S) — 1순위가 현재 트리에서 이미 해결되어 성립하지 않을 때만 선택. `guide_docs_test.go`, `docs/USER_GUIDE.md` 4.6 표, `mcpTools`를 확인했으며 도구 이름 집합은 현재 11개다. 소스 문자열 검사나 mcpTools 직접 비교만 하지 말고 App.Handler의 실제 tools/list 응답을 사용한다. 전체 Handler를 거치려면 newTestApp 기반 DB·세션 준비가 필요하므로 DSN 없는 검사라고 주장하지 않는다.

실행 순서·예산: 8분 fixture와 실패 재현 → 10분 세 SQL 및 스키마/경계 처리 → 17분 회귀·기존 테스트 → 10분 전체 검증과 여유, 총 45분. DB를 준비하지 못하면 성공으로 표시하지 말고 검증 차단 사유를 남긴다. 정찰의 확정 근거는 스키마와 SQL의 불일치이며 실제 DB 재현은 미확인이다.
스킬 확인: 요청한 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 노출 도구와 로컬 스킬 경로에서 찾지 못했다. 해당 스킬 고유 절차·반환 형식은 미확인이며 위 산정/실행 계획은 사용자 지시에 따라 독립 작성했다.

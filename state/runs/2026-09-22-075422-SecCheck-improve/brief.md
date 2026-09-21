- 과제: MCP 심의 리포트의 잘못된 JSON 타입을 필터 생략으로 처리하지 않기 (가치 4 / 위험 1 / 작업량 M)
- 왜: `mcpTools`는 from·to·department를 string으로 선언하지만 `mcpReviewReport`는 `stringValue`가 숫자·불리언·배열·객체·null을 빈 문자열로 바꾸는 탓에 잘못 보낸 필터를 조용히 버린다. 호출자에게 인자 오류를 돌려주면 기간·부서를 제한했다고 믿으며 전체 집계를 사용하는 일을 막는다.
- 수용 기준: 1) 권한 있는 사용자의 `seccheck.review_report` 호출에서 from/to/department 중 존재하는 값이 문자열이 아니면 HTTP 200의 JSON-RPC error.code=-32602를 반환하고 result/structuredContent를 반환하지 않는다(null도 거부; 오류는 필드명과 문자열 요구만 표시하며 원문 값을 포함하지 않는다). 2) 인자 생략·빈 객체·문자열 빈값·공백만 있는 문자열의 기존 무필터 의미와 정상 문자열의 TrimSpace 동작은 유지한다. 정상 날짜·부서 필터가 실제 행을 제외하며 같은 필터의 REST JSON과 MCP structuredContent가 일치한다. 3) 전용 실 PostgreSQL과 기존 newHarness/NewServer·로그인·실 HTTP POST /mcp를 쓰는 회귀 테스트가 각 세 필드×숫자/불리언/배열/객체/null을 거부함을 증명한다. 수정 전에는 해당 잘못된 호출이 성공 집계를 내므로 실패해야 하며, REQUESTER의 기존 권한 오류(-32001)는 유지한다. 잘못된 인자가 성공 MCP_TOOL_CALL 감사 이벤트로 남지 않는 것도 확인한다.
- 건드릴 파일: `internal/web/mcp.go:callMCPTool` — review_report case의 기존 역할 확인 뒤, mcpReviewReport 호출 전에 세 인자의 존재 여부와 string 타입만 검사하고 rpcError 반환. `internal/web/mcp_report_arguments_test.go`(새 파일, package web_test) — `internal/web/integration_test.go:newHarness, harness.user, harness.login, client.do, client.send, client.createReview, adminOf`를 재사용하는 런타임 테스트. 기존 `internal/web/admin.go:stringValue`는 여러 도구가 공유하므로 변경하지 않는다. `mcpReviewReport`, `mcpTools`, `internal/web/server.go`의 POST /mcp·GET /api/v1/reports/reviews 등록은 실제 확인한 배선 참고 지점이다.
- 검증 명령: 저장소 루트에서 `go test ./internal/web -run '^TestMCPReviewReportArguments$' -count=1 -v`(새 테스트를 이 이름으로 작성), `go test ./internal/web -run 'TestMCPReviewReportArguments|TestReviewReportSummarisesAPeriod' -count=1 -v`, `go test -race ./internal/web -run '^TestMCPReviewReportArguments$' -count=1`, `go test ./...`, `go vet ./...`. 모든 DB 검증 명령에는 구현자가 마련한 전용 테스트 DB의 TEST_POSTGRES_DSN을 설정한다. `test -n "$TEST_POSTGRES_DSN"`로 먼저 확인하고 SKIP는 통과로 세지 않는다. 현재 정찰에서는 두 번째 명령의 기존 테스트 하나를 단독 실행해 컴파일/명령 유효성만 확인했고 DSN 부재로 SKIP였다. 전체·race·새 테스트는 미실행이다.
- 위험과 피할 것: auth/, internal/auth/, migrations/, .github/workflows/, go.mod/go.sum, 프런트, PDF 및 날짜 형식·달력일·기간 역전 검증은 범위 밖. 09-21의 350d36c 날짜 검증 과제는 기록상 성공했지만 현재 HEAD eb2a3b0에는 없다. 그 커밋을 다시 구현하거나 cherry-pick하지 않는다. 최신 구현 기준에서 이미 타입 검증이 있으면 실제 HTTP로 확인하고 차선으로 전환한다. reportFilter의 반환형이 바뀌었어도 이 과제는 callMCPTool 진입부만 수정하여 독립성을 유지한다. 전역 stringValue, 모든 MCP 도구의 스키마 검증기, URL query 병합 동작까지 넓히지 않는다. 인자 오류를 감사/로그에 기록하려고 원문을 새로 넘기지 않는다. 기존 정상 호출 감사 정책의 전면 개편도 범위 밖. 외부 vulndb/CI 최신 상태는 미확인; 과거 게이트 실패를 이 수정 탓이라고 단정하거나 게이트를 완화하지 않는다.
- 차선 후보: MCP 리포트의 HTTP query가 도구 인자를 몰래 보충하는 동작 차단 (가치 3 / 위험 2 / M) — `mcpReviewReport`가 r.URL.Query()에서 시작하므로 `/mcp?department=다른부서`가 arguments={}에 섞인다. 동일 세션·DB에서 POST /mcp와 POST /mcp?department=…의 같은 arguments 집계 차이를 먼저 재현하고, JSON arguments만 보고서 필터의 원천으로 삼도록 좁힌다. 우선 과제와 함께 구현하지 않는다.

범위·대안 판단
- 목표는 MCP 호출자가 잘못된 타입으로 보낸 필터를 성공한 조건으로 오해하지 않게 하는 것이다. 현재 inputSchema와 일치시키는 세 필드 검사만 선택한다.
- 최소 변경(선택): 해당 도구 case에서 타입을 검사. 기존 프로토콜 파라미터 오류(-32602) 통로를 재사용하고 새 의존성이 없다.
- 전체 MCP JSON Schema 검증: 다른 도구까지 일관되지만 호환성·의존성·테스트 범위가 커져 45분에 부적합.
- 클라이언트에서만 수정/현상 유지: 호출자는 외부 에이전트까지 포함되어 서버가 계약을 보장하지 못한다.
- 가장 중요한 가정: 명시적 null은 스키마상 string이 아니므로 생략과 구분해 거절한다. 기존 null 사용 클라이언트 유무는 미확인이나 서버가 선언한 계약에는 부합한다.

실행 순서와 체크포인트 (각 상태: pending; 사람 승인 체크포인트 없음)
1. [pending] 기존 newHarness 패턴으로 TestMCPReviewReportArguments 작성. 정상/누락 인자는 통과하고 잘못된 타입은 수정 전 실패하는 것을 첫 검증 명령으로 기록한다. 실패가 DB/CSRF/인증 오류라면 결함 재현으로 세지 않는다.
2. [pending] callMCPTool의 review_report case 역할 검사 뒤에서 키 존재와 타입을 검사한다. 같은 첫 명령으로 전 케이스 통과 확인 후 다음 단계로 간다. 역할 검사를 앞으로 이동하거나 우회하지 않는다.
3. [pending] 한 fixture에 서로 다른 부서·기간의 심의를 생성하고 created_at 등은 기존 TestReviewReportSummarisesAPeriod처럼 전용 DB에서 결정적으로 맞춘다. REST JSON과 MCP structuredContent 비교, 권한 거부, 감사 성공 기록 부재를 두 번째 검증 명령으로 확인한다. 문자열 날짜는 2026-01-01 같은 유효 값만 사용하여 미병합 날짜 과제와 분리한다.
4. [pending] 제한 race, 전체 go test, vet를 차례로 실행하고 PASS/FAIL/SKIP를 구분한다. 사실이 과제서와 다르면 과제서를 수정해 근거를 남긴 뒤 진행한다.
- MCP 재현 payload: {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"seccheck.review_report","arguments":{"from":123}}}. 기본 client.do는 실제 쿠키/CSRF를 붙인다. 헤더 없는 기존 호환 모드도 프로덕션 배선이다. 최신 프로토콜 케이스도 하나 확인하려면 validateMCPHeaders에 맞춰 MCP-Protocol-Version=2026-07-28, Mcp-Method=tools/call, Mcp-Name=seccheck.review_report 및 params._meta["io.modelcontextprotocol/protocolVersion"]="2026-07-28"를 설정한다. 런타임 응답으로 증명하고 소스 문자열 검사·가짜 세션/스토어 대역만으로 대체하지 않는다.

추정 근거 (pmo:estimating-and-contingency 적용)
- 방식: bottom-up. 준비/실패 재현 7~10분, 타입 검사 3~5분, 정상/권한/감사 회귀 9~12분, 제한 race·전체 테스트·vet 5~8분 = 순수 작업 24~35분.
- 알려진 불확실성 예비 5~10분: 전용 DB 준비와 fixture/프로토콜 헤더 조정. 합계 29~45분, 중간 수준의 주관적 확신이며 통계적으로 보정된 80% 구간은 아니다.
- 관리 예비: 별도 0분 배정; 범위 밖 문제가 생기면 이 과제에 흡수하지 않고 기록한다. 환경 설치 장기화는 45분 완료 가정의 실패다.
- 비교 근거: 전 회차 실 DB 전체 Go 테스트가 약 160초였다는 운영 기록으로 검증 시간의 규모만 교차 점검했다. 유사 변경의 총 작업시간 기록은 없어 두 번째 독립 추정값은 만들지 않았다.
- 적용 스킬: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 technology/skills/{implementation-planning,solution-exploration}/SKILL.md를 직접 읽었다. Skill 호출 도구는 발견되지 않았다. pmo references/sources.md도 읽었으나 외부 기관의 예비율·효익 산식을 인용하지 않았으며 위 숫자는 실제 읽은 변경 범위와 회차 기록에 대한 정찰 추정이다.

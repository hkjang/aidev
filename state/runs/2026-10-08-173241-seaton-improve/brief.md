- 과제: MCP 직원·좌석 조회가 행 읽기 오류를 성공 목록으로 반환하지 않게 한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: internal/app/mcp.go의 executeMCPTool은 세 조회 도구에서 rows.Scan 오류를 버린 채 빈 값이나 불완전한 값을 목록에 넣고 rows.Err도 확인하지 않는다. 오류를 기존 mcpToolCall로 전달하면 MCP 클라이언트는 불완전한 좌석·직원 정보를 성공으로 받는 대신 isError:true와 안전한 고정 문장을 받는다.
- 수용 기준: 1) search_employees, list_available_seats, get_floor_map 모두 Scan 또는 rows.Err 오류를 반환한다. 2) 정상 SQL·키·순서·빈 배열·도면 URL은 유지한다. 3) 기존 rowScanner/fakeRows로 정상 열 매핑, 빈 결과, Scan 실패 즉시 중단, 행 전후 rows.Err를 증명한다.
- 건드릴 파일: internal/app/mcp.go:executeMCPTool — 세 루프를 rowScanner를 받는 scanMCP* 함수로 추출하고 오류를 반환; internal/app/mcp_test.go(신규) — employees_test.go의 fakeRows 재사용 회귀 테스트. 프로덕션 1파일.
- 검증 명령: go test ./internal/app; go test ./...; go build ./...; go vet ./...; git diff --check (이번 정찰 실행 결과는 후속 확인 예정).
- 위험과 피할 것: HTTP 500으로 바꾸지 말 것. MCP는 기존 HTTP 200 + result.isError:true 계약을 유지한다. DB 오류를 errMCP로 감싸면 원문이 노출되므로 그대로 반환한다. auth/mcpoauth/migrations/workflows·SQL·limit·배정은 범위 밖. 실서버 오류 재현은 미확인.
- 차선 후보: MCP search_employees의 limit을 공개 스키마와 맞추기 — 스키마는 기본10/최대50인데 실행은 공통 기본20/최대100. 구체 테스트 가능성 확인 후 보완 예정.

초안: 실제 확인한 mcp.go, employees.go, seats_test.go 기준. 추가 검증으로 덮어쓴다.

- 과제: 심의 리포트 날짜 필터를 REST·Excel·MCP 공통으로 검증 (가치 4 / 위험 2 / 작업량 M)
- 왜: `internal/web/reports.go:reportFilter`는 from/to를 TrimSpace한 뒤 검증 없이 PostgreSQL date 캐스트에 넘기며, `reviewReport`는 쿼리 오류를 500 QUERY_FAILED로 처리한다(소스로 확인, 잘못된 날짜의 HTTP 재현은 미확인). 입력 오류를 명확히 반환하고 REST JSON·Excel·MCP가 같은 날짜 계약을 사용하면 잘못된 요청을 서버 장애로 오인하지 않는다.
- 수용 기준: 1) GET /api/v1/reports/reviews의 JSON 및 format=xlsx 모두 잘못된 from/to(문자열, 2026-02-30, 0000-01-01, 비정규 날짜)와 from>to에 422 VALIDATION_FAILED 및 해당 필드 details를 반환한다. 2) 공백 제거 후 빈 값은 무제한, 한쪽 날짜만 지정·같은 날·윤년의 유효한 날짜는 허용하며 기존 display_day_start 기반 종료일 포함 의미를 유지한다; seccheck.review_report도 같은 기준으로 거절하되 기존 MCP result.isError 도구 오류 형식으로 안전한 날짜 안내를 제공하고 정상 결과는 유지한다. 3) 실제 NewServer·newHarness·로그인 세션·PostgreSQL을 통한 회귀 테스트가 REST JSON/Excel 및 POST /mcp tools/call의 오류와 정상 경로를 증명한다; XLSX 성공은 실제 워크북을 열어 기간 내 심의 집계를 JSON 및 MCP와 비교한다. 소스 문자열 검사·가짜 DB·검증 함수만의 단위 테스트는 수용 증거가 아니다.
- 건드릴 파일: internal/web/reports.go:reportFilter,reviewReport — 날짜 파싱 및 오류 전달, buildReport 앞에서 검증; internal/web/mcp.go:mcpReviewReport,callMCPTool — 공통 필터 오류 전달과 알려진 검증 오류만 안전한 도구 오류로 변환; internal/web/report_filter_test.go(신규, package web_test) — 기존 integration_test.go:newHarness,client.do,client.send,client.createReview를 재사용하는 실제 HTTP 회귀 테스트. 기존 integration_test.go:TestReviewReportSummarisesAPeriod는 정상 동작의 참고이며 가급적 수정하지 않는다.
- 검증 명령: `go test ./internal/web -run '^(TestReportDateFilter|TestReviewReportSummarisesAPeriod)' -count=1 -v`; `go test -race ./internal/web -run '^(TestReportDateFilter|TestReviewReportSummarisesAPeriod)' -count=1`; `go test ./...`; `go vet ./...`. 앞의 모든 통합 테스트에는 전용 테스트 PostgreSQL의 TEST_POSTGRES_DSN을 반드시 설정한다. DSN이 없으면 internal/testdb:open이 SKIP하므로 통과로 보고하지 않는다. 정찰에서 기존 TestReviewReportSummarisesAPeriod 명령은 exit 0/SKIP을 확인했고, CSV 관련 세 테스트는 실제 PASS했다. 새 TestReportDateFilter 접두 테스트는 구현자가 작성할 이름이며 아직 존재하지 않는다.
- 위험과 피할 것: auth/·migrations/·.github/workflows/·go.mod/go.sum 변경 금지. SQL 시간대 함수와 날짜 포함 범위는 그대로 유지하고 감사로그·심의 목록의 다른 날짜 필터까지 확장하지 않는다. MCP의 일반 DB 오류 원문을 노출하지 말고, 이 과제로 새 감사로그에 사용자 원문을 추가하지 않는다. reportFilter 호출자는 reviewReport와 mcpReviewReport 두 곳이므로 반드시 함께 수정한다. 기존 PR #9~#13 내용을 cherry-pick하거나 동일 취약점 게이트 재판정 과제를 반복하지 않는다. 과거 govulncheck 외부 차단의 최신 상태는 이번 회차 미확인; 로컬 기능 검증과 CI/릴리즈 가능성을 구별하고 게이트는 완화하지 않는다.
- 차선 후보: docs/user-guide.md의 대체된 옛 사용자 가이드 제거 — 현재 파일 머리말에 대체 선언이 있고 README/docs/scripts/internal/web/src/.github에서 파일명 참조 0건 확인. 1순위가 실제로 이미 해결되어 있을 때만 적용하고 `go test ./internal/web -run 'Docs|Guide' -count=1` 및 그림/PDF 참조 확인; 외부 CI 차단만을 이유로 차선으로 바꾸지 않는다.

범위·대안 비교
- 최소 REST만 검증: 짧지만 MCP와 입력 계약이 갈라져 제외.
- 공통 reportFilter에서 검증(선택): 두 호출자와 새 통합 테스트로 범위가 닫히고 실제 오류 응답이 달라진다.
- 모든 목록/감사 날짜 파서를 통합: 더 많은 API 계약을 바꾸므로 45분 범위 밖.
- 문서 정리/아무 변경 없음: 문서 정리는 가치가 작고, CI 외부 차단 재판정은 반복 실패 이력이 있어 우선순위에서 제외. CI 차단 상태 자체를 해결했다고 주장하지 않는다.

실행 순서와 체크포인트 (시작 시 모두 pending; 사람이 없는 회차이므로 사람 승인 대기 없음)
1. [pending] 전용 DB와 기존 하네스를 확인하고 동일 명령으로 TestReviewReportSummarisesAPeriod가 SKIP 없이 PASS하는지 확인. HTTP/MCP 테스트를 추가하여 기존 코드의 실패 응답을 기록한다. 증명: 첫 검증 명령(-v). 전제와 다르면 과제서에 이유를 먼저 적고 수정한다.
2. [pending] reports.go 공통 검증과 MCP 호출자/오류 매핑을 한 번에 바꿔 컴파일 가능한 상태로 마친다. YYYY-MM-DD 고정 형식·실제 달력일·연도 1~9999·순서를 확인하고 SQL에 넘기는 값은 정규 형식으로 유지한다. 증명: 첫 검증 명령에서 신규 테스트 및 기존 정상 테스트 모두 PASS. MCP 요청 헤더는 mcp.go:validateMCPHeaders를 따르거나 기존 호환 프로토콜 2025-11-25를 명시하고 실제 인증 미들웨어를 통과시킨다.
3. [pending] 제한된 race 테스트→전체 Go 테스트→vet 순으로 실행하고 PASS/SKIP/실패를 각각 기록한다. 정상 기간의 JSON 집계·워크북 집계·MCP structuredContent 동일성, 유효하지 않은 요청의 오류 형식을 확인한다. 새 실패가 없으면 검사 반복이나 프런트 변경 없이 종료한다.

작업량 산정 (스킬 적용)
- bottom-up 기본: 하네스/재현 6분 + 공통 검증/MCP 전달 10분 + 세 경로 회귀 테스트 14분 + 검증/결과 기록 7분 = 37분.
- 알려진 불확실성 contingency 8분: MCP 요청 헤더·워크북 셀 확인 조정. 합계 37~45분, 통계적 신뢰구간이 아닌 중간 확신의 작업 추정. 전용 DB와 Go 캐시가 준비되어 있다는 가정이며 환경 준비 시간이 초과하면 범위를 몰래 줄이지 말고 재산정한다.
- 유사 사례 교차 확인: 기존 TestReviewReportSummarisesAPeriod가 로그인·JSON·XLSX·권한을 이미 다루므로 하네스 신규 구축은 필요 없으나 MCP 테스트 시간은 실측 자료가 없다. 역사적 작업시간 자료가 없어 두 방식의 수치 비교는 미확인으로 둔다.
- management reserve는 별도 미배정(0분); 새로운 기능·전역 날짜 통합은 범위 변경이며 이번 회차에 흡수하지 않는다.
- 적용 스킬 원문: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md. Skill 전용 도구가 없어 실제 파일을 읽었다. pmo references/sources.md도 확인했으며 외부 기관의 정량 신뢰도/비용 기준을 이 추정의 근거로 주장하지 않는다.

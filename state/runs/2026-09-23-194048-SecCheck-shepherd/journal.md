# PR 처리기 노트 2026-09-23-194048-SecCheck-shepherd — SecCheck PR #16
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-23-183433-SecCheck-improve)
# 회차 노트 2026-09-23-183433-SecCheck-improve — SecCheck
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:34] base pinned — main@eb2a3b0
- [러너 18:34] autonomy release — 

## 정찰 노트
- 고른 이유: 도구 계약 위반 + 감사 불일치가 한 함수(mcpReviewReport)에 갇혀 있어 45분 M 으로 맞고, 보호 경로·워크플로·go.mod 를 전혀 건드리지 않는다. 월 시작 timezone·필터 경쟁·심의번호는 여러 화면 공통 계약이라 한쪽만 고치지 말라는 지침에 걸려 제쳤고, 문서 삭제류(user-guide.md)는 과거 반복 유형이라 뒤로 뺐다.
- 확신 없는 곳: DB 를 띄워 재현하지 않았다. `POST /mcp?include_done=1` 이 실제로 결과를 바꾸는지는 코드 경로 판단이며, 구현자는 수정 전 재현을 먼저 해야 한다. `audit_logs` 테이블명은 09-22 회차 기록을 인용했고 이번에 직접 확인하지 않았다.
- 구현자 주의: 저장소에 tools/call 테스트가 하나도 없으므로 JSON-RPC 헬퍼를 직접 만들고, MCP-Protocol-Version 헤더는 붙이지 말 것(붙이면 Mcp-Method/Mcp-Name/_meta 까지 요구됨). 09-21(날짜 검증)·09-22(인자 타입 검증)는 끝난 작업이니 재구현 금지, reportFilter 시그니처도 그대로 둘 것.
- 게이트: Go vulnerability scan(GO-2026-6452)은 origin/main 에서도 빨강이며 이 과제와 무관하다. 이번 회차엔 vulndb 를 조회하지 않았고 우회 시도도 금지.
- [러너 18:37] scout done — MCP `seccheck.review_report` 가 도구 인자 대신 `/mcp` URL 쿼리로 리포트 범위가 바뀌지 않게 하기 (가치 4 / 위험 1 

## 구현 노트
- 무엇/왜: `internal/web/mcp.go` 의 `mcpReviewReport` 가 리포트 범위를 `r.URL.Query()` 에서 시작하던 것을 빈 `url.Values{}` 로 바꿔, 도구 스키마에 선언된 `from`/`to`/`department` 인자만 필터가 되게 했다. 커밋 f4df4c2 (`internal/web/mcp.go` + 새 `internal/web/mcp_report_test.go`), 다른 파일 변경 없음.
- 재현 → 수정 → 통과 순서를 지켰다. 실 PostgreSQL 16(docker, TEST_POSTGRES_DSN 설정)에서 수정 전 테스트가 4건 실패: `?include_done=1` 만으로 등록부가 1→2행, `?from/to/department` 가 `structuredContent` 를 바꿈, 그런데 `audit_logs.after_value` 는 `arguments:{}` 뿐이었다. 수정 후 전부 PASS.
- 확신 없는 곳: (1) `format=xlsx` 가 `followUpLimit` 을 500→50000 으로 올리는 경로는 **직접 관측하지 못했다** — 픽스처 행이 2행뿐이라 두 한도가 같은 결과를 낸다. 코드상 성립하지만 이 테스트는 `include_done` 과 날짜/부서 누수로만 결함을 증명한다. (2) 원격 CI·`Go vulnerability scan`(GO-2026-6452)·vulndb 상태는 이번 회차에 **조회하지 않았다**. 초록을 주장하지 않는다.
- 일부러 하지 않은 것: `reportFilter` 시그니처·`format`/`include_done` 처리는 REST 엑셀 내보내기가 같은 키를 쓰므로 손대지 않았다(과제서 지침). 09-21 날짜 형식 검증·09-22 인자 타입 검증은 재구현하지 않았다. 감사 details 에는 URL 쿼리 원문을 넣지 않았다 — `p.Arguments` 가 이제 곧 적용된 필터다.
- 다음 역할 주의: `internal/web/mcp_report_test.go` 는 **DB 가 있어야 돈다**(`TEST_POSTGRES_DSN` 없으면 조용히 SKIP — SKIP 을 PASS 로 읽지 말 것). 감사 details 컬럼은 `details` 가 아니라 `audit_logs.after_value` 다. 새 헬퍼 `(*client).callTool` 은 MCP-Protocol-Version 헤더를 일부러 보내지 않는다(보내면 `validateMCPHeaders` 가 `Mcp-Method`/`Mcp-Name`/`_meta` 를 요구).
- 검증 출력: `go test ./internal/web -run 'MCP|Report' -count=1 -v` → 18 PASS/0 FAIL/0 SKIP; `go test ./... -count=1`(실 DSN) → 전 패키지 ok, web 167.009s; `go vet ./...` exit 0; `bash scripts/precheck.sh` 통과(gitleaks "no leaks found"), 단 프런트엔드 단계는 node_modules 부재로 건너뛰어 따로 `npm ci`+`npm test`(2 파일 11개 PASS)+`tsc -b`(exit 0) 를 돌렸다.
- [러너 18:47] brief accepted — 채택 — `mcpReviewReport` 가 `r.URL.Query()` 에서 출발한다는 전제와 `include_done`·`format`·`from`/`to`/`department` 가 실제로 결과를 바
- [러너 18:47] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: 실 PostgreSQL 16(docker)에서 새 테스트 PASS → `query := url.Values{}` 를 `r.URL.Query()` 로 되돌리니 같은 테스트가 4건 실패(재현성 입증), 트리 원상복구·`git status` clean. 이어 실 DSN `go test ./... -count=1` 전 패키지 ok, `go vet` 0, `gofmt -l internal/web` 무출력. 소비자 검색 결과 URL 쿼리로 `/mcp` 를 부르는 코드·문서·프런트는 없음. 판정 approve / risk low / blocking 없음.
- 못 본 것: 원격 CI, Go vulnerability scan(GO-2026-6452), npm audit·race·이미지·DAST, 프런트 테스트. 로컬 Go 검증만으로 CI 초록을 주장하지 않는다.
- 남는 우려 1: `mcp_report_test.go` 의 `format=xlsx` 단언은 공허하다 — follow-up 픽스처가 2행뿐이라 500 과 exportRowCap 이 같은 결과를 내므로 그 키만의 회귀는 잡히지 않는다(수정 자체는 by-construction 으로 막음).
- 남는 우려 2: `stringValue`(admin.go:545)가 비문자열을 조용히 `""` 로 만들어, `{"department":123}` 은 필터 없이 감사에만 남는다. mcp.go:257 주석의 "what ran is what was logged" 는 아직 완전하지 않다. 이번 변경이 만든 결함은 아니며 tools/call 은 inputSchema 대조 검증이 전혀 없다 — 다음 회차 후보.
- 릴리즈 노트: "MCP `seccheck.review_report` 의 범위는 도구 인자만으로 정해진다(URL 쿼리 무시)" 를 동작 변경으로 명시할 것.
- [러너 18:54] review approved — 리뷰 승인 (risk=low)
- [러너 18:54] pr created — https://github.com/hkjang/SecCheck/pull/16
- [러너 19:00] ci failed — 성공이 아닌 검사: test-build-scan=failure

## 수리 노트
- 맞은 것: CI 실패는 진짜다 — HEAD 에서 `govulncheck ./...` exit 3, `GO-2026-6452`, `internal/store/seed.go:66:25` 로 CI 로그와 동일하게 재현했다.
- 틀린 것: 이 PR 의 결함이라는 전제. 같은 스캔이 기준점 `origin/main`(eb2a3b0) 에서도 exit 3·같은 트레이스로 실패하고, diff 는 `internal/web/mcp.go`+`mcp_report_test.go` 뿐(`internal/store/`·`go.mod`·`go.sum` 0건)이다. 정찰 노트의 "origin/main 도 빨강" 주장을 이번에 직접 확인했다.
- 고친 방법: 없음 — 커밋하지 않았다. `Fixed in: N/A` 이고 `go list -m -versions` 의 최신이 이미 쓰는 `excelize v2.11.0` 이라 올릴 버전이 없다. 남은 길은 ci.yml 완화(검증 명령 수정 금지)나 무관한 `ExtractWorkbookDefaults` 수술(범위 밖)뿐이라 절대 규칙대로 멈췄다. 근거는 fix-summary.md.
- PR 자체 검증: 실 PostgreSQL 16 으로 CI 와 같은 `go test -race -coverprofile=coverage.out ./...` exit 0(internal/web 197.199s), `go vet` exit 0, 새 테스트는 SKIP 아닌 1.03s PASS. ci.yml 상 vuln 단계는 테스트 단계 *다음*이라 CI 에서도 테스트는 통과한 뒤 여기서 멈춘 것이다.
- 확신 없는 곳: 이 게이트는 사람이 정책으로 풀어야 한다(excelize 상류 패치 대기, 또는 다른 gate 들처럼 unfixed 를 제외하는 결정). 나머지 CI 단계(프런트·gitleaks·이미지·SBOM·DAST)는 vuln 단계에서 잡 이 끊겨 이번에도 원격에서 실행된 적이 없고, 로컬에서도 돌리지 않았다 — 초록을 주장하지 않는다.

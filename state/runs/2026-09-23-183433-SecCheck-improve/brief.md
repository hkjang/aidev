# 과제서 (2026-09-23 정찰)

- 과제: MCP `seccheck.review_report` 가 도구 인자 대신 `/mcp` URL 쿼리로 리포트 범위가 바뀌지 않게 하기 (가치 4 / 위험 1 / 작업량 M)

## 왜
`internal/web/mcp.go:253 mcpReviewReport` 는 `query := r.URL.Query()` 로 **POST /mcp 요청 URL 의 쿼리스트링을 출발점으로 삼고**, 도구 인자(`from`/`to`/`department`)는 비어 있지 않을 때만 `query.Set` 으로 덮는다. 그래서 `POST /mcp?include_done=1&format=xlsx&department=인프라` 처럼 URL 에 붙인 값이 그대로 `reportFilter`(internal/web/reports.go:31)로 흘러 들어가, 도구 스키마(`mcpTools()` 의 `review_report`, `additionalProperties:false`, 속성은 from/to/department 뿐)에 없는 `include_done`·`format` 까지 결과를 바꾼다.
결과는 두 가지다. (1) `format=xlsx` 는 `scope.followUpLimit` 을 500 → `exportRowCap`(=50000, internal/web/admin.go:672)로 올리고 `include_done=1` 은 완료·취소 심의의 보완조치까지 포함시켜, tools/list 가 선언한 적 없는 입력으로 응답 크기와 의미가 달라진다. (2) `callMCPTool` 의 감사 기록(`MCP_TOOL_CALL`, details `{"arguments": p.Arguments}`)에는 URL 쿼리가 전혀 남지 않아, **실제로 적용된 필터와 감사에 남은 인자가 어긋난다.** 고치면 MCP 도구의 입력이 선언된 스키마 하나로 정해지고, 감사 기록이 실제 적용 범위와 일치한다.

권한 상승은 아니다(REST `GET /api/v1/reports/reviews` 도 같은 4개 역할: server.go:121, mcp.go 의 `seccheck.review_report` 역할 검사와 동일). 고치는 것은 **도구 계약과 감사 정합성**이다.

## 수용 기준
1. `POST /mcp?from=2020-01-01&to=2020-01-02&department=<존재하는 부서>&include_done=1&format=xlsx` 로 `tools/call seccheck.review_report` 를 인자 없이(`arguments:{}`) 호출하면, 응답 `structuredContent` 가 **쿼리 없는 같은 호출의 응답과 동일**하다(특히 `from`/`to` 가 빈 문자열, `by_department` 가 전 부서, `follow_ups` 에 완료/취소 항목이 없음).
2. 도구 인자로 넘긴 `from`/`to`/`department` 는 지금처럼 그대로 동작한다(수정 전후 동일). 인자 생략 시 전체 기간·전 부서로 집계된다.
3. 테스트가 증명할 것: (a) 수정 전이라면 URL 쿼리가 결과를 바꿨다는 것(회귀 테스트를 수정 되돌린 트리에서 돌려 실패하는지 확인), (b) 수정 후 URL 쿼리 유무와 무관하게 동일 응답, (c) 도구 인자 경로는 여전히 필터가 걸린다(인자로 `department` 를 준 응답이 전체 응답과 다름 — 실제 행 수로 확인), (d) 감사 기록(`audit_logs`, 테이블명은 audit_events 가 아님 — 09-22 회차 확인)의 `MCP_TOOL_CALL` details 가 실제 적용 인자와 일치.
4. `go vet ./...` 과 아래 검증 명령이 통과하고, 작업 트리에 `.github/workflows/`·`go.mod`·`go.sum`·`internal/auth`·`internal/store/migrations` 변경이 없다.

## 건드릴 파일
- `internal/web/mcp.go:253 mcpReviewReport` — `r.URL.Query()` 대신 **빈 `url.Values{}`** 로 시작해 도구 인자 `from`/`to`/`department` 만 `Set` 한다(현재의 `strings.TrimSpace(stringValue(args[key])) != ""` 조건은 유지). `scoped := r.Clone(r.Context())` 후 `scoped.URL.RawQuery = query.Encode()` 는 그대로. `net/url` import 추가.
- `internal/web/mcp_report_test.go` (새 파일; 기존 `internal/web` 패키지 테스트와 같은 위치) — `newHarness`(integration_test.go:51) → `h.user(..., "SECURITY_REVIEWER")` → `h.login(...)` → `c.do(http.MethodPost, "/mcp?...", body)` 로 JSON-RPC `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"seccheck.review_report","arguments":{}}}` 를 보낸다. `c.do` 의 path 에 쿼리스트링을 그대로 붙일 수 있다(integration_test.go:149 가 `server.URL+path` 로 만듦). MCP-Protocol-Version 헤더는 보내지 말 것 — 보내면 `validateMCPHeaders` 가 `Mcp-Method`/`Mcp-Name`/`_meta` 까지 요구한다(mcp.go:46~56, 107~122). 현재 저장소에 `tools/call` 을 호출하는 테스트는 **하나도 없으므로** 헬퍼를 직접 만들어야 한다.
- 픽스처: 부서가 다른 심의 2건 이상 + `follow_up` 이 있고 `follow_up_done_at` 이 채워진 항목 1건(= `include_done` 유무가 실제로 행 수를 바꾸도록). `c.createReview`(integration_test.go:197) 와 직접 SQL(`h.db.Pool.Exec`) 을 섞어 쓰는 것이 기존 테스트 관례다.

## 검증 명령
```
export TEST_POSTGRES_DSN=<실 PostgreSQL DSN>            # 없으면 통합 테스트는 조용히 SKIP 된다 — SKIP 을 PASS 로 보고하지 말 것
go test ./internal/web -run 'MCP|Report' -count=1 -v    # 새 테스트가 PASS/SKIP 중 무엇인지 -v 로 확인
go test ./... -count=1                                   # 실 DSN 아래 전 패키지 (internal/web 는 2~3분)
go vet ./...
bash scripts/precheck.sh                                 # gofmt/vet/Go 테스트/프런트/그림/PDF/비밀 검사
```
DSN 없이 돌리면 `internal/web` 통합 테스트는 전부 건너뛴다. **DSN 없는 PASS 를 DB 검증으로 보고하지 말 것**(이 저장소의 반복 실패 지점).

## 위험과 피할 것
- **같이 고치지 말 것**: `reportFilter` 의 날짜 형식 검증(2026-09-21 회차 350d36c)과 MCP 인자의 JSON 타입 검증(2026-09-22 회차 837d1e8)은 이미 각각 끝난 작업이고 지금 HEAD(eb2a3b0)에는 없을 뿐이다. **재구현 금지.** 이번 변경은 `mcpReviewReport` 본문 한 곳에 가두고 `reportFilter` 시그니처는 건드리지 않는다(그래야 저 브랜치들과 충돌이 최소).
- `format=xlsx` 를 MCP 쪽에서 별도로 막거나 `reportFilter` 에서 `format` 처리를 바꾸지 말 것 — REST 엑셀 내보내기가 같은 키를 쓴다(reports.go:44~47, 82). URL 쿼리를 아예 상속하지 않으면 그 자체로 해결된다.
- 보호 경로 금지: `internal/auth/*`, `internal/web/core_handlers.go` 의 인증 함수, `internal/store/migrations/*`, `.github/workflows/*`.
- 외부 게이트: `Go vulnerability scan`(GO-2026-6452) 은 origin/main 에서도 빨강이다(09-19~09-20 회차 3회 확인). 이 과제와 무관하며, **워크플로 완화·replace·vendor·excelize 호출 분리로 우회하지 말 것.** 원격 CI 초록을 이번 과제의 성공 조건으로 삼지 않는다.
- 감사 기록에 URL 쿼리 원문을 새로 집어넣지 말 것(운영자 지침: 감사 details 에는 원문 문자열 말고 식별자). 감사는 지금처럼 `p.Arguments` 만 남기면 되고, 이번 수정으로 그것이 곧 실제 적용 필터가 된다.
- 미확인: 실제 DB 를 띄워 이 결함을 재현하지는 않았다(이번 회차는 읽기만 했고 DSN 도 확인하지 않았다). 코드 경로상 성립하는 것까지가 확인 범위이므로, 구현자는 **먼저 수정 없이 테스트를 돌려 URL 쿼리가 실제로 결과를 바꾸는지 재현**한 뒤 고칠 것. 재현되지 않으면(예: 미들웨어가 /mcp 의 쿼리를 지움) 그 사실을 적고 차선 후보로 넘어갈 것.

## 차선 후보
`seccheck.list_reviews` / `seccheck.search_controls` 의 `limit` 이 스키마 범위(1~100)를 벗어나면 조용히 50 으로 되돌아가는 문제(`internal/web/mcp.go:343 numberValue`) — `limit:1000` 이나 `limit:"5"` 가 오류 없이 기본값으로 처리되어 호출자가 잘린 줄 모른다. 같은 harness 로 검증 가능하고 작업량 S.
(그 다음: `docs/user-guide.md` 제거해 `USER_GUIDE.md` 하나만 정본으로 — 참조 0건은 09-20 회차에서 확인되었으나 이번 회차에 재확인하지 않았음.)

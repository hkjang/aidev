- 과제: MCP `orbit_create_memory` 의 title/content 누락이 "요청을 처리하지 못했습니다." 로 나가는 것을, DB 앞에서 무엇이 빠졌는지 말해 주는 메시지로 바꾸기 (가치 2 / 위험 1 / 작업량 S)
- 왜: `mcpCreateMemory`(mcp.go:292~297)는 `title`·`content` 가 비면 `errors.New("title and content required")` 를 돌려주고, `mcpCall` 의 오류 매핑(mcp.go:239~246)은 `pgx.ErrNoRows` 가 아닌 모든 오류를 "요청을 처리하지 못했습니다." 한 문장으로 덮는다 — 호출한 외부 모델은 무엇이 빠졌는지도, 애초에 자기 잘못인지(서버 장애인지)도 알 수 없어 스스로 고칠 수 없다. v0.7.10 이 같은 arm 에 `mcpPersonIDError` 가드를 깔아 두었으므로 같은 모양으로 한 칸 올리면 호출자 실수가 고칠 수 있는 메시지가 되고 DB 장애의 일반 메시지와도 구분된다.
- 수용 기준:
  1) `tools/call orbit_create_memory` 에 ① title 누락 ② content 누락 ③ 둘 다 누락 ④ 공백만 있는 값(`"  "`)을 주면 HTTP 200 + JSON-RPC `result.isError:true` 로 **빠진 필드 이름을 담은 한국어 메시지**가 나온다(`mcpResultText` 로 봉투까지 확인, 텍스트에 `title`/`content` 가 들어 있는지 검사). 지금은 네 경우 모두 "요청을 처리하지 못했습니다." 다.
  2) 그 거부는 **DB 에 닿기 전에** 끝난다 — `callWithoutStore((&Server{}).mcp, req)` 의 `panicked` 가 거짓이어야 한다.
  3) person_id 가드와의 순서가 유지된다 — uuid 모양이 어긋난 person_id 와 빈 title 을 함께 주면 **person_id 메시지**가 먼저 나온다(새 가드를 `mcpPersonIDError` 뒤에 두면 그대로 성립).
  4) 회귀: `TestMCPLetsWellFormedPersonIDReachDB`·`TestMCPAllowsEmptyPersonIDWhereOptional`·`TestMCPRejectsMalformedPersonIDBeforeDB`·`TestMCPToolsDeclarePersonIDFormat` 이 고치기 전과 후 모두 PASS. (확인함: 이 시험들의 `orbit_create_memory` 케이스는 전부 `title:"제목", content:"본문"` 을 넘긴다 — mcp_test.go:71, 176~177 — 그래서 새 가드가 이들을 삼키지 않는다.)
  5) `normalizeTags`(data.go:256, REST 와 공유)·`TrimSpace` 를 지난 정상 경로의 값이 지금과 한 글자도 다르지 않다.
- 건드릴 파일 (프로덕션 1파일):
  - `internal/server/mcp.go`
    - ① `mcpPersonIDError`(mcp.go:263~277) 바로 아래에 같은 모양(문제없으면 `""`)의 헬퍼를 추가: `func mcpMemoryFieldError(title, content string) string`. `strings.TrimSpace` 후 빈 것을 보고 title 만 / content 만 / 둘 다에 각각 필드 이름을 담은 한국어 문장을 돌려준다. `strings` 는 이미 import 돼 있다.
    - ② `orbit_create_memory` arm(mcp.go:222~235)의 `mcpPersonIDError` 가드 **바로 뒤**, `result, err = s.mcpCreateMemory(...)` 앞에 `if message := mcpMemoryFieldError(args.Title, args.Content); message != "" { s.rpcWrite(w, req.ID, map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": message}}}, nil); return }` — 바로 위 person_id 가드와 같은 리터럴 모양을 복사할 것.
    - ③ `mcpCreateMemory` 안의 `if title == "" || content == ""` 는 **지우지 말 것**. 그 함수의 사전조건이고, 이제 도달하지 않는 방어선이 된 이유를 한국어 주석으로 남길 것.
  - `internal/server/mcp_test.go` — 기존 `mcpToolCall`·`mcpResultText`·`callWithoutStore` 를 그대로 재사용해 기준 1~3 의 하위 시험을 추가(표 기반). 새 헬퍼의 순수 함수 시험도 같은 파일에. **DB 불필요** — 새 시험은 CI 에서 실제로 돈다.
  - `tools/list` 스키마는 이미 `required: ["title","content"]` 이고 `TestMCPToolsDeclarePersonIDFormat` 이 그것을 고정하고 있다(확인함) — 건드릴 것 없음.
- 검증 명령 (이 저장소에서 실제로 돌려 확인함 — 기준선 초록):
  - `cd /home/hkjang/.cache/auto-improve-wt/orbit`
  - `gofmt -l internal/server` (무출력이어야 함)
  - `go vet ./... && go build ./...`
  - `go test -race -count=1 ./...` — DSN 없이 전부 ok(DB 시험 SKIP). 2026-10-05 `go test -count=1 ./...` 로 기준선 초록 확인함(config·secure·server·scripts ok).
  - 집중: `go test -race -count=1 -v ./internal/server -run 'TestMCP'`
  - 인과 고정: ② 의 가드 블록만 지운 빌드에서 새 시험이 빨개지고 되살리면 초록인 것을 확인할 것.
- 빨강을 먼저 볼 수 있다 (코드 경로를 읽어 확인함, 실행 전):
  `mcpCreateMemory` 는 `normalizeTags`·`TrimSpace`(순수 함수) 다음 곧바로 `title==""||content==""` 에서 반환하고, 첫 DB 접촉인 `SELECT EXISTS`(mcp.go:301, 그것도 `personID != ""` 일 때만)·`activeDataKey`(mcp.go:308)에는 닿지 않는다. 따라서 **store nil 로도** 패닉 없이 "요청을 처리하지 못했습니다." 가 나오고, 그것이 고치기 전 빨강이다 — DB 없이 결함을 증명할 수 있다.
- 위험과 피할 것:
  - `mcpCreateMemory` 의 시그니처·반환 계약(`(map[string]any, error)`)·`normalizeTags`→`TrimSpace` 순서를 바꾸지 말 것. 호출자가 하나뿐이어도 함수 안의 검사를 지우면 사전조건이 사라진다.
  - 기존 isError 네 자리(mcp.go:133 권한 거부, 175·211·228 의 person_id, 243 의 일반 매핑)를 공통 헬퍼로 묶는 리팩터를 **하지 말 것** — 2026-10-05 회차가 같은 판단으로 남겨 두었다.
  - 공유 함수·질의(`normalizeTags`·`queryMemories`·`queryMemoriesLimit`·`personConnections`·`escapeLike`)의 시그니처·SQL 은 한 글자도 바꾸지 말 것(REST·AI·MCP 공유). `orbit_search_people` arm 의 `rows.Err()` 가드도 건드리지 말 것.
  - 새 가드를 person_id 가드보다 **앞에** 두지 말 것 — 기준 3 이 깨진다.
  - 보호 경로(auth.go·throttle.go·internal/secure·internal/store/migrations·.github/workflows)는 열지 말 것. `requestHasScope` 도 그대로.
  - 손으로 만든 대역을 쓰지 말고 프로덕션 배선(`(&Server{}).mcp` 에 실제 JSON-RPC 본문 + `userContextKey`)으로 부를 것 — `mcpToolCall`(mcp_test.go:17)·`callWithoutStore`(workflow_test.go:17)가 이미 그 모양이다. `authContextKey` 는 넣지 않아도 된다(`requestHasScope` 는 authInfo 가 없으면 통과).
- 차선 후보: `docs/API.md` 에 v0.7.8 의 `/personal/export` `complete:false`·`failed_section` 계약을 문서화 (가치 2 / 위험 1 / 작업량 S). 코드 변경 0. 경로·메서드는 `internal/server/server.go` 라우팅에서, 본문 모양은 `internal/server/export.go:58~90` 과 `export_db_test.go` 의 실제 출력에서 옮겨 적을 것. `docs/cru-manual.md` 의 API 설명은 실제와 다르다는 기록이 있으니 근거로 쓰지 말 것.

## 2026-09-09
- 선택: MCP 의 has_more 가 페이지 크기 추측이라 마지막 페이지에서 거짓말함 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `asset_search` 와 `agents_list` 는 `has_more` 를 `len(items) == limit` 으로
  계산해, 결과가 정확히 한도에서 끝나면(기본 50 인 상태에서 에이전트가 정확히
  50대, 또는 `type` 필터가 요청한 만큼만 맞는 경우) 다음 페이지가 있다고
  보고했고, 그 말을 믿은 클라이언트는 빈 페이지를 한 번 더 가져왔다.
  `agents_list` 는 offset 자체가 없어 `has_more:true` 가 애초에 가져올 수 없는
  페이지를 약속했다 — 더 있다는 말을 들은 호출자가 할 수 있는 일은 같은 요청을
  반복하는 것뿐이었다. 둘 다 한 행을 더 읽고 버리도록 바꿔 `has_more` 가 실제로
  다음 행이 존재하는지를 말하게 했고, `agents_list` 에 `offset` 입력과
  `offset`·`next_offset` 응답을 추가해 나머지에 도달할 수 있게 했다(`/api/v1/assets`
  와 query/execute 가 이미 쓰는 lookahead 와 같은 방식). 검증:
  `server/internal/httpapi/mcp_pagination_test.go` 추가 — 3행을 limit 3 으로 읽으면
  `has_more` false, limit 2 면 2행·`has_more` true·`next_offset` 2, 그
  `next_offset` 으로 다시 물으면 남은 1행이 오고 `has_more` false 인지, lookahead
  행이 페이지에 섞여 나오지 않는지, 그리고 `agents_list` 스키마가 `offset` 을
  실제로 광고하는지를 확인한다. 헛돌지 않는지 보려고 lookahead 와 비교식을 옛
  방식으로 되돌려 두 테스트가 정확히 "경계에서 끝난 결과에 has_more true" 로
  실패하는 것을 확인한 뒤 복구했다. `go test ./...` 를 SQLite fallback 과 실
  PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지 통과, `go vet`·
  `go build`·`gofmt` 통과. `docs/API_MCP_GUIDE.md` 의 도구 표에 `agents_list` 의
  `offset` 을 넣고 `has_more` 가 추측이 아니라는 점과 `next_offset` 사용법을
  한 문단 적었다. web·Rust·`openapi.yaml`(MCP 는 도구별 스키마를 기술하지 않음)은
  손대지 않아 `npm`·`cargo`·`redocly` 는 돌리지 않았다. 버전 범프·릴리즈 노트는
  하지 않았다.
- 보류 아이디어: `attributes.*` 의 배열·객체 값이 두 저장 모드에서 다른 텍스트로 렌더링됨(SQLite 은 공백 없는 JSON, PG `#>>` 는 `{"k": "v"}`) — 컨테이너를 가리키는 절을 아예 거절하는 편이 나을 수도 (가치 3 / 위험 2 / S) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · Query DSL 실행에 offset 이 없어 상한 500 을 넘는 나머지를 받아낼 방법이 아예 없음 — `/api/v1/assets` 처럼 offset·total 을 주는 것이 대안 (가치 3 / 위험 2 / M) · `listAgents`·설정 목록/이력·자산 상세 루프가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — 드라이버 fault injection 없이는 테스트 불가 (가치 3 / 위험 1 / M) · MCP `asset_search` 가 0건일 때 실제 존재하는 type·status 값을 함께 돌려주어 '자산 없음'과 '필터 값 없음'을 구분하게 함 (가치 3 / 위험 2 / M)

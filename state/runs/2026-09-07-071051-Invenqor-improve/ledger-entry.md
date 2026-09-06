## 2026-09-07
- 선택: MCP `asset_search`·`agents_list` 의 `has_more` 가 마지막 페이지를 "더 있음"으로 알림 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 두 MCP 도구는 `has_more` 를 `len(items) == limit` 로 계산했다. 정확히
  limit 개에서 끝나는 페이지는 크기만으로는 잘린 페이지와 구분되지 않으므로,
  완전한 답이 잘린 것으로 보고됐다 — 모델은 빈 페이지를 한 번 더 읽고 재고가
  일관되지 않다고 말하거나 아무것도 돌려주지 않을 offset 을 계속 넘긴다.
  `agents_list` 는 offset 인자 자체가 없어서 "더 있다"는 호출자가 해소할 수 없는
  주장이었다. 이제 두 도구 모두 한 행을 더 읽고 버려 `has_more` 를 확정한다
  (`software_inventory` 가 이미 실제 COUNT 로 확정하던 것과 같은 성질의 답을
  lookahead 로 얻는다). 같은 자리에서 `rows.Err()` 를 응답 전에 확인하도록 바꿔,
  반복 중 오류로 짧아진 목록이 `has_more: false` 인 성공 응답으로 나가지 않게 했다.
  검증: 자산 3건·에이전트 3건에 대해 limit 2(잘림)·limit 3(정확히 채운 마지막
  페이지)·limit 10(부분 페이지)·offset 2 로 도달한 마지막 페이지를 확인하는 테스트
  2개 추가(`mcp_paging_test.go`, `next_offset` 과 `count` 일관성 포함).
  `go test ./...` 를 SQLite fallback 과 실 PostgreSQL(`scripts/test-postgres.sh`)
  양쪽에서 전 패키지 통과, `go vet`·`go build`·`gofmt` 통과. web·Rust·openapi.yaml
  은 건드리지 않아(MCP 도구 출력은 openapi 에 기술돼 있지 않다) `npm`·`cargo`·
  `redocly` 는 돌리지 않았다. 문서 `.md` 와 버전 범프·릴리즈 노트는 하지 않았다:
  `docs/*.md` 는 릴리즈 커밋에서 PDF 와 함께만 갱신되고, 병렬 브랜치가 이미 다음
  버전 번호를 만들어 두어 같은 번호를 쓰면 병합 시 버전 파일이 통째로 충돌한다.
- 보류 아이디어: `/api/v1/external/query/*` API key 경로의 rate limit(429)·감사 기록 principal 에 테스트가 하나도 없음 (가치 3 / 위험 1 / M) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · `listAgents`·설정 목록/이력·자산 상세의 sources/history/relations 루프가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 (가치 3 / 위험 1 / M) · MCP `asset_search` 의 `type`·`status` 가 검증 없이 통과해 오타가 "해당 자산 없음"으로 답해짐(`software_inventory` 는 `mustBeOneOf` 로 거절) (가치 3 / 위험 1 / S) · API key 로 한 행위도 감사 기록의 `actor_type` 이 `user` 라 소유자가 콘솔에서 한 일과 구분되지 않음 (가치 2 / 위험 3 / S)

## 2026-09-10
- 선택: MCP `asset_relations` 가 관계 목록에 상한을 두지 않아 무한정 큰 응답을 돌려줌 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `mcpAssetRelations` 만 유일하게 LIMIT 이 없어 자산의 활성 관계를 전부
  돌려줬다 — process 상관관계가 돈 서버에서 `runs_on` 자식을 수천 개 가진 host
  하나면 도구 호출 한 번의 응답이 모델 context 를 통째로 차지하고, 잘라내는 쪽은
  클라이언트였다. 다른 읽기 도구와 같은 `limit`(기본 50, 최대 100)·`offset` 을
  받고, 한 행을 더 읽어 버리는 lookahead 로 `has_more`·`next_offset` 을 사실로
  보고하게 했다. 기존 `relations` 키를 그대로 유지해 응답 호환은 깨지지 않는다.
  겸사겸사 가이드의 MCP 도구 표를 `mcpTools` 스키마와 대조하는 테스트를 추가했고
  (openapi.yaml 대 라우터를 대조하는 `TestOpenAPIRouteCoverage` 와 같은 방식),
  그 테스트가 실제로 기존 드리프트인 `software_inventory` 의 `vendor` 누락을
  잡아내 문서를 함께 고쳤다. 검증: `mcp_pagination_test.go` 에 두 테스트 추가 —
  3개 관계를 limit 3 으로 읽으면 `has_more` false, limit 2 면 2건·`has_more`
  true·`next_offset` 2 이고 그 offset 으로 나머지 1건이 오며 세 페이지 합이 각
  edge 정확히 1회(lookahead 행이 섞이지 않음), 관계 55개에서 limit 미지정이면
  50건·`has_more` true. 헛돌지 않는지 보려고 lookahead 와 비교식을 옛 방식으로
  되돌려 두 테스트가 정확히 그 지점에서 실패하는 것을 확인한 뒤 복구했다. 문서
  대조 테스트도 표를 고치기 전에 `asset_relations`·`software_inventory` 두
  드리프트를 차례로 잡는 것을 확인했다. `go test ./...` 를 SQLite fallback 과 실
  PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지 통과, `go vet`·
  `go build`·`gofmt` 통과. web·Rust·`openapi.yaml`(MCP 는 도구별 스키마를 기술하지
  않음)은 손대지 않아 `npm`·`cargo`·`redocly` 는 돌리지 않았다. 버전 범프·릴리즈
  노트는 하지 않았다.
- 보류 아이디어: Query DSL 실행에 offset 이 없어 상한 500 을 넘는 나머지를 받아낼 방법이 아예 없음 — `/api/v1/assets` 처럼 offset·total 을 주는 것이 대안 (가치 3 / 위험 2 / M) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · MCP `asset_search` 가 0건일 때 실제 존재하는 type·status 값을 함께 돌려주어 '자산 없음'과 '필터 값 없음'을 구분하게 함 (가치 3 / 위험 2 / M) · `listAgents`·설정 목록/이력·자산 상세 루프가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — storage 에 테스트용 공개 생성자가 없어 fault injection 불가 (가치 3 / 위험 1 / M) · `attributes.*` 의 배열·객체 값이 두 저장 모드에서 다른 텍스트로 렌더링됨 — 컨테이너를 가리키는 절을 아예 거절하는 편이 나을 수도 (가치 3 / 위험 2 / S)

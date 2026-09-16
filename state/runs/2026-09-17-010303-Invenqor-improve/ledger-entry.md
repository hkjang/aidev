## 2026-09-17
- 선택: MCP `asset_relations` 가 상대 자산의 이름·종류·상태와 방향을 각 행에 담아 edge 마다 `asset_get` 을 다시 부르지 않게 함 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `mcpAssetRelations` 가 `assets` 를 source·target 으로 두 번 join 해 행마다 `source_asset`·`target_asset`
  (`id`,`name`,`type`,`status` — 콘솔 관계 목록과 같은 모양)과 질문한 `asset_id` 기준 `direction`
  (`outbound`/`inbound`)을 더해 돌려준다. 삭제·병합된 자산의 edge 는 그대로 남으므로 상대 `status` 로
  `asset_get` 이 실패할 자산임을 미리 알린다. 기존 `source_asset_id`·`target_asset_id` 열과 페이지 필드는
  그대로 유지해 호환을 지켰고, 도구 설명·API_MCP_GUIDE.md 표와 6장 설명(예시 JSON)을 갱신했다(PDF 는
  릴리즈 회차에서 재생성되는 관례라 건드리지 않음). 같은 회차에 보류돼 있던 `TestMCPToolTableMatchesDeclaredSchemas`
  의 Scope 열 대조도 붙였다(일부러 표의 scope 를 틀리게 바꿔 실패하는 것을 확인 뒤 되돌림). 검증: 새 단위
  2개(호스트 기준 outbound/inbound 양 끝 이름·종류·상태·UUID 유지·반대편에서 보면 방향 반전, 삭제된 상대의
  `status=deleted`), `go test ./...` SQLite 전 패키지 통과, 실 PostgreSQL(`scripts/test-postgres.sh`,
  다른 세션 컨테이너와 겹치지 않게 별도 이름·포트) 전 패키지 통과, `go vet`·`gofmt`·`go build`. web 변경
  없음. 버전 범프·릴리즈 노트는 하지 않았다.
- 보류 아이디어: MCP `asset_get` 이 병합된 자산에 "asset not found" 만 답함 — asset_changes 의 merged 기록에서 primary 를 찾아 "merged_into" 로 안내 (가치 3 / 위험 2 / M) · 콘솔·외부 REST `assetRelations` 는 여전히 상한 없이 모든 edge 를 돌려줌 — MCP 와 같은 `limit`/`offset`·`has_more` 도입, 콘솔 관계 패널이 소비하므로 web 수정 동반 (가치 3 / 위험 2 / M) · MCP `asset_relations` 에 `relation_type` 필터 인자 추가 — 스키마·가이드 표·대조 테스트를 함께 고침 (가치 2 / 위험 1 / S) · 여러 목록 핸들러가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — listAgents·settings·자산 상세 sources/history/relations (가치 3 / 위험 1 / M) · 콘솔 Query DSL 화면이 새 `total`·`offset` 을 쓰지 않아 첫 페이지만 보여줌 — webui/dist 재빌드 필요 (가치 3 / 위험 2 / M)

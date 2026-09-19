- 과제: MCP `asset_get` 이 병합된(secondary) 자산에 "asset not found" 만 답하던 것을 `merged_into` 로 primary 자산을 안내하게 한다 (가치 3 / 위험 2 / 작업량 M)
- 왜: 콘솔에서 자산 A 를 B 로 병합하면 A 는 `status='merged', deleted_at=now` 가 되는데(`server/internal/httpapi/assets.go:647 mergeAssets`), MCP `asset_get` 은 `deleted_at IS NULL` 로 걸러 `errors.New("asset not found")` 만 돌려준다(`server/internal/httpapi/mcp.go:691 mcpAssetGet`). 모델은 예전 대화·`asset_search` 결과·다른 도구 응답에 남은 A 의 UUID 로 다시 묻고 "자산이 없다" 고 결론내리므로, 병합 사실과 primary UUID 를 함께 알려 주면 한 번 더 호출해 정답에 닿는다. 같은 자산을 REST `GET /api/v1/assets/{id}`(`assets.go:260 getAsset`) 는 `status:"merged"` 로 돌려주므로 두 경로가 같은 상태를 같게 읽게 되는 것이기도 하다.
- 수용 기준:
  1) 병합된 secondary UUID 로 `asset_get` 을 부르면 오류가 아니라 `{"asset": null 또는 생략, "merged_into": "<primary UUID>", "merged_at": <deleted_at>, "message": "This asset was merged into <primary>; call asset_get with that id."}` 모양의 결과가 돌아온다(필드 이름은 이 세 개를 기준으로 하되 `asset` 을 빼려면 가이드 표에 그대로 적을 것). 존재한 적 없는 UUID 는 예전과 글자까지 같은 `asset not found` 를 유지한다.
  2) primary 자체가 나중에 다시 다른 자산으로 병합된 경우(A→B, B→C) 도 한 단계만 안내한다(B 를 돌려줌). 체인을 끝까지 따라가지 않는다 — 다음 호출이 다시 안내를 받으므로 충분하고, 순환 방지 코드가 필요 없다.
  3) 테스트가 증명할 것: 실제 `Server`(`testServer`, SQLite) 에서 자산 두 개를 넣고 **실제 REST 경로 `POST /api/v1/assets/merge`** 를 `performAuthenticatedJSON`(`admin_test_helpers_test.go:84`) 로 호출해 병합한 뒤 `server.mcpAssetGet` 을 부르면 (a) secondary 는 `merged_into == primary`, (b) primary 는 그대로 `asset` 이 온다, (c) 무작위 UUID 는 `asset not found`. `asset_changes` 를 손으로 INSERT 해서 흉내내지 말 것(운영자 규칙: 대역이 아니라 프로덕션 배선을 통과할 것). `TestMCPToolTableMatchesDeclaredSchemas`(`mcp_tool_docs_test.go`) 와 `go test ./...` 가 SQLite·PostgreSQL 양쪽에서 통과.
- 건드릴 파일:
  - `server/internal/httpapi/mcp.go:mcpAssetGet` — `deleted_at IS NULL` 로 없을 때 한 번 더 조회: `SELECT deleted_at FROM assets WHERE id=$1 AND status='merged'` 가 있으면 `asset_changes` 에서 `change_type='merged'` 이고 `after_json.secondary_ids` 배열에 그 id 가 든 행의 `asset_id`(=primary) 를 찾는다(가장 최근 `occurred_at DESC LIMIT 1`). 저장 모드별 SQL 이 다르다: PostgreSQL(`after_json` 은 JSONB) 은 `after_json->'secondary_ids' ? $1` 또는 `after_json @> jsonb_build_object('secondary_ids', jsonb_build_array($1::text))`, SQLite(TEXT) 는 `EXISTS (SELECT 1 FROM json_each(after_json, '$.secondary_ids') WHERE value = $1)`. 모드 판별은 `query.go:40` 이 쓰는 `s.database.Mode() != storage.ModeSQLiteFallback` 을 그대로 쓸 것. JSONB 에 LIKE 를 쓰면 42883 으로 터진다(CI 주석에 기록된 과거 사고).
  - `server/internal/httpapi/mcp.go:49` `asset_get` 의 `Description` 에 병합 안내를 한 줄 덧붙일지는 선택(붙이면 `mcp_tool_docs_test.go` 가 가이드 표와 대조하는 항목인지 먼저 확인 — 표는 이름·scope·인자·설명 열을 본다, 미확인).
  - `docs/API_MCP_GUIDE.md:368` `asset_get` 행의 설명에 "병합된 자산은 `merged_into` 로 primary 를 안내" 를 추가. `TestMCPToolTableMatchesDeclaredSchemas` 가 이 표를 소스와 대조하므로 열 구조는 바꾸지 말 것.
  - `server/internal/httpapi/mcp_pagination_test.go` 옆에 새 테스트 파일(예: `mcp_asset_get_merged_test.go`) — 기존 `insertSoftwareTestAsset`·`testServer`·`newRuntime`·`authenticateInitialAdmin` 재사용.
  - `openapi.yaml` 은 MCP 도구 응답을 스키마화하지 않으므로(미확인 — `asset_get` 으로 grep 해서 없으면 손대지 않음) 건드리지 않는다.
- 검증 명령:
  - `cd server && go test ./internal/httpapi -run 'TestMCPAssetGet|TestMCPToolTable' -count=1`
  - `cd server && go test ./... && go vet ./... && gofmt -l .`
  - `scripts/test-postgres.sh` (이 환경에 docker 가 있음 — 2026-09-20 `docker ps` 확인; JSONB 연산자는 반드시 실 PostgreSQL 로 검증)
  - 콘솔·webui/dist 는 바뀌지 않으므로 `npm run build` 불필요; `git diff --exit-code -- server/internal/webui/dist` 가 비어 있어야 함.
- 위험과 피할 것:
  - `mergeAssets`·`asset_changes` 쓰기 경로·마이그레이션(`server/migrations/*`) 은 손대지 않는다. `assets` 표에 `merged_into` 열을 더하는 방식은 마이그레이션이 필요하고 과거 병합 데이터를 못 채우므로 금지.
  - REST `getAsset` 은 이미 merged 자산을 200 으로 돌려주므로 바꾸지 않는다(범위 밖).
  - `asset not found` 문자열은 기존 테스트·가이드가 참조할 수 있으니 존재하지 않는 UUID 에 대해서는 그대로 유지.
  - SQLite 와 PostgreSQL 에서 `deleted_at` 이 각각 문자열/`time.Time` 으로 스캔되므로 응답에 넣을 때는 다른 곳처럼 `any` 로 받아 `apiTime(...)` 로 정규화(`assets.go` getAsset 의 sources 루프 참고).
  - 실제 값이 바뀌지 않는 장식 변경(설명만 고치기 등) 만으로 끝내지 말 것 — 응답 모양이 실제로 달라져야 한다.
- 차선 후보: Query DSL 화면의 limit 입력이 0·빈값·501 을 그대로 보내 Server 가 100/500 으로 고쳐 돌려주는 것을 입력 시점에 1~500 으로 맞추고 되돌려 준 `limit` 을 편집기에도 반영 (가치 2 / 위험 1 / S) — `web/src/.../operationsPages.tsx` 의 QueryResultPanel 주변, vitest 추가와 `npm run build` → webui/dist 동기화 동반.

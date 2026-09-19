# 회차 노트 2026-09-20-064412-Invenqor-improve — Invenqor
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:44] base pinned — main@39f61f6
- [러너 06:44] autonomy release — 

## 정찰 노트
- 선택: MCP `asset_get` 의 병합 자산 `merged_into` 안내(3/2/M). 순수 Server 변경이라 webui/dist 재빌드가 없고, 실제 REST merge 경로로 테스트해 "대역 금지" 규칙을 만족시키기 쉬워 assetRelations 페이지네이션(콘솔 동반)·rows.Err()(실패 주입 선행)·cron 워크플로(보호 경로)를 제쳤다.
- 확신 없는 곳: `mcp_tool_docs_test.go` 가 가이드 표의 설명 열까지 대조하는지 미확인(도구 이름·scope·인자만 볼 수도 있음). `openapi.yaml` 에 MCP 도구 응답 스키마가 없다는 것도 grep 으로 확인 필요. `Mode() != ModeSQLiteFallback` 이 PostgresDegraded 를 postgres 로 치는 의미는 query.go 를 그대로 따르면 됨.
- 조심할 것: `after_json` 은 PostgreSQL 에서 JSONB — LIKE 금지, `?`/`@>`/`json_each` 로 방언 분기하고 반드시 `scripts/test-postgres.sh` 로 확인. `asset not found` 문자열은 미존재 UUID 에 대해 그대로 유지.
- [러너 06:47] scout done — MCP `asset_get` 이 병합된(secondary) 자산에 "asset not found" 만 답하던 것을 `merged_into` 로 primary 자산을 안내하게 �

## 구현 노트
- 무엇/왜: MCP `asset_get` 이 병합된 secondary 에 `asset not found` 만 답하던 것을, `status='merged'` 확인 후 `asset_changes(change_type='merged')` 의 `after_json.secondary_ids` 에서 primary 를 찾아 `{merged_into, merged_at, message}` 로 답하게 함(커밋 145f932). `asset` 키는 생략. 한 단계만 안내. 도구 Description·가이드 표 `asset_get` 행 갱신.
- 확신 없는 곳·검증 못 한 것: 실제 MCP 클라이언트(Claude/Cursor)가 `asset` 없는 응답을 받아 `merged_into` 로 다시 호출하는지는 이 환경에 클라이언트가 없어 미확인(응답은 `structuredContent` 로 그대로 나감). `secondary_ids` 가 클라이언트가 보낸 글자 그대로 저장되므로 대문자 UUID 로 병합된 옛 데이터는 조회가 못 찾아 예전대로 `asset not found` 가 남는다(콘솔은 소문자를 보내므로 실사용 영향 작음, ideas 에 기록). 두 저장 모드 모두 실 DB(SQLite·postgres:17-alpine 컨테이너)로 테스트했고 PostgreSQL 분기는 일부러 뒤집어 빨강이 되는 것까지 확인했다.
- 일부러 하지 않은 것: `assets` 에 `merged_into` 열 추가(마이그레이션·과거 데이터 미채움 — 과제서 금지), `mergeAssets`·마이그레이션·REST `getAsset` 수정, 체인 끝까지 따라가기, 병합 외 이유로 soft-delete 된 자산에 대한 안내(범위 밖), openapi.yaml(MCP 도구 응답을 스키마화하지 않음 — grep 으로 확인), webui 재빌드(콘솔 변경 없음).
- 다음 역할이 조심할 것: `mcp_asset_get_merged_test.go` 는 `authenticateInitialAdmin` + 실제 REST merge 를 거치므로 `testServer` 가 필요하고 PostgreSQL 에서도 돌아야 한다(`scripts/test-postgres.sh`; 기본 포트 55432 는 이 머신에서 다른 컨테이너가 쓰고 있어 `POSTGRES_CONTAINER`/`POSTGRES_PORT` 를 따로 줬음). 조회 SQL 의 `after_json` 은 JSONB — LIKE 로 바꾸면 `TestNoTextOperatorIsAppliedToAJsonbColumnWithoutACast` 와 실 PostgreSQL 이 함께 깨진다. 릴리즈(버전·노트·PDF)는 하지 않았다.
- [러너 06:58] brief accepted — 채택 — 과제서의 근거(`mcpAssetGet` 의 `deleted_at IS NULL` 필터, `mergeAssets` 의 asset_changes 기록 모양, 문서 대조 테스트가 입력 �
- [러너 06:58] verify passed — 검증 8개 통과 (auto)

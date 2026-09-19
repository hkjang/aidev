## 2026-09-20
- 선택: MCP `asset_get` 이 병합된(secondary) 자산에 "asset not found" 만 답하던 것을 `merged_into` 로 primary 자산을 안내하게 함 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 콘솔에서 A 를 B 로 병합하면 A 는 `status='merged', deleted_at=now` 가 되는데 MCP `asset_get` 은
  `deleted_at IS NULL` 로 걸러 존재한 적 없는 UUID 와 글자까지 같은 `asset not found` 만 돌려줬고, 예전 대화·
  검색 페이지·관계 edge 에 남은 A 의 UUID 로 다시 묻는 모델은 자산이 사라졌다고 결론냈다(REST GET 과 콘솔은
  같은 자산을 merged 로 보여 줌). 이제 live 조회가 비면 `status='merged'` 인지 한 번 더 보고 `mergeAssets` 가
  쓰는 `asset_changes(change_type='merged', after_json.secondary_ids)` 에서 가장 최근 primary 를 찾아
  `{merged_into, merged_at, message}` 로 답한다(`asset` 키는 생략, 가이드 표에 명시). A→B→C 체인은 한 단계만
  안내(A 는 B 를 받음 — 다음 호출이 다시 안내를 받으므로 순환 방지 불필요). `after_json` 은 PostgreSQL 에서
  JSONB 라 `@> jsonb_build_object('secondary_ids', jsonb_build_array($1::text))`, SQLite 는 `json_each` 로
  `query.go` 와 같은 `Mode() != ModeSQLiteFallback` 분기. 조회 실패는 not found 로 접지 않고 오류로 돌려
  방언 실수가 숨지 않게 했다. 존재하지 않는 UUID·병합 외 이유로 삭제된 자산은 예전 그대로 `asset not found`.
  도구 `Description` 과 `docs/API_MCP_GUIDE.md` 도구 표 `asset_get` 행에 응답 모양을 적었다(표 대조 테스트는
  입력 열만 보므로 열 구조 유지). 검증: 새 테스트 2개(`mcp_asset_get_merged_test.go`)는 실제 `Server`(SQLite/
  PostgreSQL 공용 `testServer`)에서 자산을 넣고 **실제 REST `POST /api/v1/assets/merge`** 로 병합한 뒤
  `mcpAssetGet` 을 불러 secondary→`merged_into==primary`·`merged_at` RFC3339·`asset` 없음, primary 는 그대로
  `asset`, 무작위 UUID 는 `asset not found`, 두 단계 체인은 한 단계만을 확인. 수정 전 두 테스트 모두 `asset not
  found` 로 빨강 확인. 방언 분기를 일부러 뒤집어 PostgreSQL 에서 빨강이 되는 것도 확인(그 분기가 실제로 타는
  증거). `go test ./...`(SQLite), `scripts/test-postgres.sh`(postgres:17-alpine, 별도 컨테이너·포트 55461 —
  기본 55432 는 다른 컨테이너가 점유) 전 패키지, `go vet`, `gofmt -l` 빈 출력, `go build`. webui/dist 변경 없음.
  버전 범프·릴리즈 노트·PDF 재생성은 하지 않았다.
- 보류 아이디어: mergeAssets 가 primary·secondary 존재를 확인하지 않아 없는 secondary 는 조용히 성공·없는 primary 는 PostgreSQL 에서 FK 500/SQLite 성공으로 두 저장 모드가 갈림 — 트랜잭션 첫머리 존재 확인 (가치 3 / 위험 2 / S) · 콘솔·외부 REST `assetRelations` 가 상한 없이 모든 edge 를 돌려줌 — MCP 와 같은 `limit`/`offset`·`has_more`, 콘솔 관계 패널 수정 동반 (가치 3 / 위험 2 / M) · 여러 목록 핸들러가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — 실패 주입 수단이 선행 과제 (가치 3 / 위험 1 / M) · Query DSL 화면의 limit 입력이 0·빈값·501 을 그대로 보내 Server 가 100/500 으로 고쳐 돌려줌 — 입력 시점에 1~500 으로 맞추고 되돌려 준 `limit` 을 편집기에 반영 (가치 2 / 위험 1 / S) · mergeAssets 가 secondary_ids 를 보낸 글자 그대로 저장해 대문자 UUID 병합은 `merged_into` 조회가 못 찾음 — 쓰기 시 소문자 정규화 (가치 2 / 위험 1 / S)
- 과제서: 채택 — 과제서의 근거(`mcpAssetGet` 의 `deleted_at IS NULL` 필터, `mergeAssets` 의 asset_changes 기록 모양, 문서 대조 테스트가 입력 열만 보는 것)가 모두 코드와 일치해 그대로 구현했고, 유일한 이탈은 조회 오류를 not found 로 접지 않고 돌려주는 것.

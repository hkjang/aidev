## 2026-10-03
- 선택: `POST /api/v1/assets/merge` 가 비정규형 UUID 철자를 받아 두 방언이 갈리고 비정규형이 `asset_changes` 에 남아 MCP `merged_into` 가 못 찾던 것을 기존 `canonicalUUID` 로 막음 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `openapi.yaml:2229-2235` 는 `primary_id`·`secondary_ids` 를 `format: uuid` 로, 오류를 400 하나로만
  약속하는데 `mergeAssets` 는 `missingAssetID` 의 `uuid.Parse` 모양 검사만 거쳐 보낸 글자를 그대로 SQL 로
  넘겼다. `assets.id`·`asset_sources.asset_id` 가 PostgreSQL 에서 UUID 열이고 SQLite 폴백은 TEXT 라
  `uuid.Parse` 가 받아들이는 느슨한 철자가 방언별로 갈렸다. 2026-10-01 회차가 `splitAsset` 에 넣은
  `canonicalUUID` 를 `decodeJSON` 직후·`BeginTx` **전**에 merge 에도 적용하고(그래서 tx 안에서
  `internalError` 를 불러 폴백이 교착하는 구역을 건드리지 않는다), 이후의 쿼리·자기 병합 스킵·
  `asset_changes` metadata·audit 기록이 모두 지역 변수 `primaryID`·`secondaryIDs` 만 읽게 바꿨다.
  `missingAssetID` 안의 모양 검사는 호출 전 정규화로 닿지 않게 되어 지우고 주석으로 전제를 적었다.
  프로덕션 파일은 `assets.go` 한 개. 검증: 새 테스트 4건이 실제 라우터를 통과하며 응답 코드뿐 아니라
  `assets.status`·`asset_sources.asset_id`·`asset_changes.after_json`·`audit_logs` 행 수를 직접 세고,
  대문자 병합 뒤 MCP `asset_get` 이 `merged_into` 로 primary 를 가리키는 것까지 확인한다.
  `go test ./...`(SQLite 전체 ok), `go vet ./...`, `gofmt -l .` 빈 출력,
  `go build ./cmd/invenqor-server`, `scripts/test-postgres.sh`(postgres:17-alpine 전체 패키지 ok, 포트
  55497 — 기본 55432 는 점유). 기존 merge 3건·split 7건·`mcp_asset_get_merged_test.go` 2건 그대로 통과.
  openapi 는 응답 코드 집합이 그대로라 손대지 않았고 버전 범프·릴리즈 노트·PDF 재생성도 하지 않았다.
- 실패 재현: PostgreSQL 에서 고치기 전 — 과제서의 예측이 전부 실측으로 확인됐다.
  `asset_merge_validation_test.go:171: merge into primary "urn:uuid:0fb04211-…" = 500 "INTERNAL_ERROR", want 400 INVALID_MERGE`,
  `asset_merge_validation_test.go:171: merge into primary "{35061cc6-…}" = 200 "", want 400 INVALID_MERGE`(하이픈 없는 32자도 200),
  `asset_merge_validation_test.go:203:` 같은 세 줄이 `secondary_ids` 쪽에서도 났고,
  대문자는 PostgreSQL 에서 병합은 됐으나
  `:261: audit_logs 'asset.merge' rows for the canonical primary = 0, want 1` 과
  `:296: asset_get on the merged asset returned an error: asset not found` 로 실패했다 — 비정규형이
  `asset_changes`·`audit_logs` 에 그대로 저장돼 `merged_into` 조회가 못 찾는다는 가설 그대로다.
  SQLite 폴백에서는 대문자 2건만 `= 400 "INVALID_MERGE", want 200` 으로 실패하고 비정규형 6건은
  고치기 전에도 통과했다(TEXT 비교는 어차피 못 맞춘다) — 기본 `go test` 만으로는 이 결함이 보이지
  않는다는 과제서의 경고가 다시 실측으로 확인됐다.
- 보류 아이디어: 트랜잭션 안에서 `internalError` 를 부르면 SQLite 폴백이 교착한다 — `mergeAssets` 745행
  이후와 `splitAsset` 의 나머지가 그대로이고 막는 것은 여전히 프로덕션 경로로 오류를 터뜨릴 재현 수단
  (가치 4 / 위험 3 / M) · 목록 핸들러 `rows.Err()` 확인, 실패 주입 수단이 없어 증명이 어려움
  (가치 3 / 위험 1 / M) · 콘솔·외부 REST `assetRelations` 페이지네이션 (가치 3 / 위험 2 / M) ·
  `mergeAssets` 가 `secondary_ids` 에 같은 id 를 중복으로 받았을 때의 `source_ids`·`asset_changes`
  (가치 2 / 위험 1 / S) · Query DSL limit 입력을 1~500 으로 맞추고 응답 limit 반영 (가치 2 / 위험 1 / S)
- 과제서: 채택 — 지목한 파일·행·헬퍼·검증 명령이 모두 들어맞았고, '미확인' 으로 남겨 둔 PostgreSQL 의
  실제 응답 코드(대문자 200 / `urn:uuid:` 500 / 중괄호·하이픈 없는 32자 200)를 고치기 전에 직접 돌려
  전부 확인했다.

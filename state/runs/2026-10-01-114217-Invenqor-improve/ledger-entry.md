## 2026-10-01
- 선택: `POST /api/v1/assets/{assetId}/split` 이 형식이 틀린 UUID 를 PostgreSQL 에서 500 으로 터뜨리고 이 엔드포인트에 테스트가 전무하던 것을 두 방언 모두 400 으로 맞추고 첫 회귀 테스트를 넣음 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `openapi.yaml:2263` 은 이 엔드포인트의 오류를 `"400": Invalid split or source does not belong to the
  original asset` 하나로만 약속하는데, `splitAsset` 은 경로 파라미터 `assetID` 와 `source_ids` 를 검사 없이
  `UPDATE asset_sources SET asset_id=$1 WHERE id=$2 AND asset_id=$3` 에 그대로 넣었다. `asset_sources.id`·
  `asset_sources.asset_id` 는 PostgreSQL 에서 UUID 열(`migrations/postgres/001_initial.sql:216-218`)이라 형식이
  틀린 입력은 22P02 로 쿼리 자체가 실패해 500 이 되고, SQLite 폴백은 TEXT 비교라 0행 → 400 이었다. 같은
  입력에 두 방언이 다른 코드를 답하고 공개 계약도 어겼다. 이제 `decodeJSON` 직후·`BeginTx` 전에
  `originalID` 와 `source_ids` 각 원소를 `uuid.Parse` 로 보아 400 `INVALID_SPLIT`·`INVALID_SOURCE` 로
  끝낸다(2026-09-29 회차가 `mergeAssets` 에 넣은 `missingAssetID` 와 같은 모양). 트랜잭션을 열기 전이므로
  보류 아이디어 4/3/M(트랜잭션 내 `internalError` → SQLite 교착)을 건드리지 않고 비켜가며, 오류 코드는
  SQLite 가 이미 답하던 것을 그대로 두어 폴백의 동작 변화가 없다. 프로덕션 파일은 `assets.go` 한 개다.
  검증: 실제 라우터 `POST /api/v1/assets/{assetID}/split` 를 통과하는 새 테스트 4건(형식이 틀린 assetId,
  형식이 틀린 source_ids, 정상 split 1건, 남의 자산 source 거절)이 응답 코드뿐 아니라 `asset_sources.asset_id`
  이동·`asset_changes(change_type='split')`·`audit_logs(action='asset.split')`·새 자산 행 수를 직접 세어
  확인한다. `go test ./...`(SQLite 전체 ok), `go vet ./...`, `gofmt -l .` 빈 출력,
  `go build ./cmd/invenqor-server`, `scripts/test-postgres.sh`(postgres:17-alpine 전체 패키지 ok, 포트 55493 —
  기본 55432 는 점유). 기존 merge 테스트 3건·`mcp_asset_get_merged_test.go` 2건 그대로 통과. openapi 는
  응답 코드 집합이 그대로라 손대지 않았고 버전 범프·릴리즈 노트·PDF 재생성도 하지 않았다.
- 실패 재현: PostgreSQL 에서 고치기 전 —
  `asset_split_validation_test.go:110: split of a malformed asset id = 500 "INTERNAL_ERROR", want 400 INVALID_SPLIT`,
  `asset_split_validation_test.go:134: split with a malformed source id = 500 "INTERNAL_ERROR", want 400 INVALID_SOURCE`.
  SQLite 폴백에서는 첫 테스트만 실패하고(`= 400 "INVALID_SOURCE", want 400 INVALID_SPLIT`) 두 번째는
  통과했다 — 22P02 는 PostgreSQL 전용이라 기본 `go test` 로는 보이지 않는다는 과제서의 지적이 실측으로
  확인됐다. 정상 split·남의 자산 source 거절 2건은 고치기 전에도 통과하는 기존 동작 보존 테스트다.
- 보류 아이디어: 트랜잭션 안에서 `internalError` 를 부르면 SQLite 폴백이 교착한다 — `splitAsset` 의 809·820·
  846·850 과 `mergeAssets` 의 724 이후가 그대로이고, 막는 것은 여전히 프로덕션 경로로 오류를 터뜨릴 재현
  수단이다 (가치 4 / 위험 3 / M) · 목록 핸들러 `rows.Err()` 확인, 실패 주입 수단이 없어 증명이 어려움
  (가치 3 / 위험 1 / M) · 콘솔·외부 REST `assetRelations` 페이지네이션 (가치 3 / 위험 2 / M) ·
  `mergeAssets` 가 `secondary_ids` 를 보낸 글자 그대로 `asset_changes` 에 저장해 대문자 UUID 는 `merged_into`
  조회가 못 찾음 (가치 2 / 위험 1 / S) · Query DSL limit 입력을 1~500 으로 맞추고 응답 limit 반영
  (가치 2 / 위험 1 / S)
- 과제서: 채택 — 근거(openapi 의 400 단독 약속, `splitAsset` 에 형식 검증이 전무, UUID/TEXT 방언 차이, split
  테스트 0건)를 모두 코드와 실측으로 확인했고, 지정한 파일·헬퍼·검증 명령이 그대로 들어맞았다.

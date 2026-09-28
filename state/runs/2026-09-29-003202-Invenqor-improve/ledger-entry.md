## 2026-09-29
- 선택: `POST /api/v1/assets/merge` 가 primary·secondary 자산의 존재를 확인하지 않아 openapi 가 약속한 400 대신 200·무한 대기로 답하던 것을 400 `INVALID_MERGE` 로 맞춤 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `openapi.yaml:2234` 는 이 엔드포인트에 `"400": Primary or secondary assets missing` 을 공개 계약으로 적어 놓았지만
  `mergeAssets` 는 입력 UUID 가 실제 자산인지 한 번도 보지 않고 곧바로 `UPDATE asset_sources` / `UPDATE assets SET
  status='merged'` 를 돌렸다. 이제 `tx` 를 연 직후 primary 와 모든 secondary 를 `SELECT id FROM assets WHERE id=$1`
  로 단건 확인하고(방언 차이를 피하려 `IN` 대신 반복, 시간 열은 읽지 않음, `deleted_at IS NULL` 은 넣지 않아 이미
  병합된 자산을 다시 secondary 로 주는 기존 동작·체인 병합이 그대로다), 하나라도 없으면 아무 것도 쓰지 않고
  `writeAPIError(400, "INVALID_MERGE")` 로 끝낸다. `assets.id` 가 PostgreSQL 에서는 UUID 열, SQLite 폴백에서는
  TEXT 라 형식이 틀린 id 는 한쪽 방언에서만 조회 자체가 오류가 되므로 조회 전에 `uuid.Parse` 로 모양을 먼저
  보아 두 방언이 같은 입력에 같은 코드를 답하게 했다. 검증: 새 테스트 3개가 실제 라우터
  `POST /api/v1/assets/merge` 를 통과하며 응답 코드뿐 아니라 `asset_changes(change_type='merged')` 와
  `audit_logs(action='asset.merge')` 를 직접 세어 0 임을, 같은 요청의 정상 secondary 가 `status='active'` 로
  남아 있음을 확인한다. 기존 `mcp_asset_get_merged_test.go` 3개 테스트 그대로 통과. `go test ./...`(SQLite,
  19개 패키지 ok), `go vet ./...`, `gofmt -l .` 빈 출력, `go build ./cmd/invenqor-server`,
  `scripts/test-postgres.sh`(postgres:17-alpine, 포트 55491 — 기본 55432 는 다른 컨테이너가 점유). 버전 범프·
  릴리즈 노트·PDF 재생성은 하지 않았다.
- 실패 재현: 없는 secondary — `asset_merge_validation_test.go:112: merge with a missing secondary = 200 "", want 400 INVALID_MERGE`,
  형식이 틀린 id — `asset_merge_validation_test.go:138: merge with a malformed secondary id = 200 "", want 400 INVALID_MERGE`.
  없는 primary 는 정찰이 예측한 500 이 **아니라** SQLite 폴백에서 응답이 영원히 돌아오지 않았다:
  `panic: test timed out after 1m0s / running tests: TestMergeAssetsRejectsPrimaryThatIsNotAnAsset (1m0s)`.
  스택으로 원인을 확인함 — `mergeAssets` → (asset_changes FK 위반) `internalError` → `recordDiagnostic` →
  `database/sql.(*DB).conn` 이 커넥션을 기다리는데 열린 트랜잭션이 SQLite 폴백의 유일한 커넥션을 쥐고 있다
  (`storage/runtime.go:273` `SetMaxOpenConns(1)`). PostgreSQL(25개)에서는 정찰의 예측대로 500 이 된다.
- 보류 아이디어: 트랜잭션 안에서 `internalError` 를 부르면 SQLite 폴백이 교착한다 — 이번에 스택으로 실측한 새 결함으로
  `splitAsset`·ingest 등 tx 안에서 오류를 보고하는 모든 핸들러에 남아 있다 (가치 4 / 위험 3 / M) · `splitAsset` 도
  `assetID`·`source_ids` 의 존재를 확인하지 않아 같은 모양의 결함일 가능성 (가치 3 / 위험 2 / S) · 목록 핸들러
  `rows.Err()` 확인 (가치 3 / 위험 1 / M) · 콘솔·외부 REST `assetRelations` 페이지네이션 (가치 3 / 위험 2 / M) ·
  Query DSL limit 입력을 1~500 으로 맞추고 응답 limit 반영 (가치 2 / 위험 1 / S)
- 과제서: 채택 — 근거(openapi 의 400 약속, `mergeAssets` 에 존재 확인이 아예 없음, FK·`PRAGMA foreign_keys`)가 모두 코드와 맞았고, 유일한 이탈은 없는 primary 의 실제 증상이 500 이 아니라 SQLite 에서의 무한 대기였다는 것이다.

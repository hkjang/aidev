- 과제: `POST /api/v1/assets/merge` 가 primary·secondary 자산의 존재를 확인하지 않아, openapi 가 공개 계약으로 약속한 400 대신 조용한 200 성공(거짓 감사 기록)·500 을 돌려준다 (가치 4 / 위험 2 / 작업량 S)
- 왜: `openapi.yaml:2234` 는 이 엔드포인트의 응답으로 `"400": Primary or secondary assets missing` 을 공개 계약에 적어 놓았지만, `mergeAssets`(`server/internal/httpapi/assets.go:647`) 는 입력 UUID 가 실제 자산인지 한 번도 확인하지 않고 곧바로 `UPDATE asset_sources` / `UPDATE assets SET status='merged'` 를 돌린다. 그래서 존재하지 않는 secondary 는 0행 갱신으로 200 을 돌려주면서 `asset_changes(change_type='merged')` 와 `audit_log(action='asset.merge', result='success')` 에 "병합했다" 는 거짓 기록만 남기고, 존재하지 않는 primary 는 `asset_changes.asset_id` 의 FK(두 방언 모두 `REFERENCES assets(id)`, SQLite 는 `PRAGMA foreign_keys = ON` — `server/internal/storage/runtime.go:276`) 때문에 500 으로 새어 나간다. 고치면 문서와 구현이 같아지고, 오타 난 UUID 를 보낸 통합 클라이언트가 "성공했다" 는 잘못된 응답과 감사 기록 대신 고칠 수 있는 400 을 받는다.

- 수용 기준:
  1) 존재하지 않는 `primary_id` 로 병합하면 400 `INVALID_MERGE` (openapi 의 `"400"` 설명과 일치), 500 이 아니다.
  2) `secondary_ids` 중 하나라도 존재하지 않으면(또는 이미 `deleted_at`/`status='merged'` 인지에 대한 판단은 아래 "위험" 참고) 400 이고, **같은 요청의 나머지 정상 secondary 도 병합되지 않는다** — 트랜잭션이 전체 롤백되어 `assets.status` 가 그대로다.
  3) 실패한 요청은 `asset_changes(change_type='merged')` 행도 `audit_log(action='asset.merge')` 행도 남기지 않는다(응답 코드만 보는 테스트로는 불충분 — 실제로 두 표를 세어 0 임을 확인).
  4) 정상 병합의 200 응답 모양(`moved`/source_ids)과 기존 테스트 `mcp_asset_get_merged_test.go`(3개 테스트: 단일 병합, 체인, not-found)가 그대로 통과한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `server/internal/httpapi/assets.go:mergeAssets` — `tx` 를 열고 루프에 들어가기 **전에** primary 와 모든 secondary 의 존재를 한 번 확인하고(예: `SELECT id FROM assets WHERE id = $1` 를 각각, 또는 `IN` 한 번 — PostgreSQL 에서 `IN` 에 파라미터 배열을 쓸 때 방언 차이가 나므로 **단건 `SELECT` 반복이 안전하다**), 없으면 `writeAPIError(response, request, 400, "INVALID_MERGE", ...)` 로 끝내고 아무 것도 쓰지 않는다. `primary_id == secondary_id` 인 항목을 건너뛰는 기존 동작(assets.go:666)은 유지.
  - 새 테스트 파일 `server/internal/httpapi/asset_merge_validation_test.go` — 아래 검증 참고. 기존 헬퍼 `newRuntime`, `testServer`, `authenticateInitialAdmin`, `insertSoftwareTestAsset`, `performAuthenticatedJSON` 를 그대로 쓰고(모두 이 패키지에 이미 있다), **실제 라우터 `POST /api/v1/assets/merge`** 를 통과시킬 것. 손으로 만든 대역이나 `mergeAssets` 직접 호출은 금지.
  - `openapi.yaml` 은 이미 400 을 약속하므로 **고치지 않아도 된다**. 문구를 다듬고 싶으면 `openapi_routes_test.go` 가 라우터와 대조하는 것을 깨지 않는 선에서만.

- 검증 명령:
  - `cd server && go test ./internal/httpapi/` (SQLite 폴백. 이 환경에서 패키지 첫 컴파일이 5분 넘게 걸렸다 — 타임아웃을 넉넉히 두고 백그라운드로 돌릴 것)
  - `cd server && go vet ./... && gofmt -l . && go build ./cmd/invenqor-server`
  - `scripts/test-postgres.sh` (postgres:17 컨테이너, 수 분. 기본 포트 55432 가 점유돼 있으면 다른 포트를 쓸 것 — 2026-09-20 회차 기록)
  - 수정 **전에** 새 테스트가 빨강인 것을 먼저 확인할 것(없는 secondary → 200, 없는 primary → 500 이 그때 보여야 한다). 빨강을 못 보면 전제가 틀린 것이니 멈추고 실제 응답을 확인.

- 위험과 피할 것:
  - **이미 병합된(`status='merged'`, `deleted_at` 있음) 자산을 secondary 로 다시 주는 경우를 400 으로 막지 말 것** — 존재 확인 SQL 에 `deleted_at IS NULL` 을 넣으면 기존 동작(재병합·체인 병합)이 바뀌어 `mcp_asset_get_merged_test.go` 의 A→B→C 체인 테스트가 깨질 수 있다. 확인은 **행이 있는지만** 보는 것으로 좁힐 것(즉 `WHERE id = $1` 만). 참고로 체인 테스트(`TestMCPAssetGetFollowsOneMergeHopOnly`, 같은 파일 113~142행)를 실제로 읽어 보니 `b<-a` 다음 `c<-b` 로 두 번째 병합의 secondary `b` 는 아직 살아 있어(그 전에는 primary 였다) `deleted_at IS NULL` 을 넣어도 이 테스트 자체는 통과한다 — 그래도 의미상 넓히지 말 것. 이미 병합된 자산을 다시 secondary 로 주는 경로에 대한 기존 동작은 **미확인**이다.
  - `auth.go`/`auth/`, `server/migrations/`, `.github/workflows`, `server/internal/webui/dist` 는 건드리지 말 것.
  - `splitAsset` 은 이번 범위 밖(같은 파일이지만 별 과제로 남겨 두었다).
  - 시간 열은 방언별로 문자열/`time.Time` 로 스캔되므로 존재 확인에서 시간 열을 읽지 말 것 — `id` 만 `SELECT`.
  - PostgreSQL 의 `assets.id` 는 `UUID` 타입이라 UUID 형식이 아닌 문자열을 `$1` 로 넣으면 존재 확인 쿼리 자체가 오류가 된다. 형식이 잘못된 ID 를 500 대신 400 으로 접을지는 구현자 판단 — 접는다면 그것도 `INVALID_MERGE` 로 하고 SQLite 에서도 같은 코드가 나오는지 테스트할 것(두 방언이 같은 입력을 같은 코드로 답해야 한다).

- 차선 후보: 여러 목록 핸들러가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려주는 것 — `listAgents`·settings·자산 상세 sources/history/relations. 다만 실패를 주입할 수단이 없으면 증명이 어려워 M 이고, 1순위가 성립하면 그쪽을 할 것.

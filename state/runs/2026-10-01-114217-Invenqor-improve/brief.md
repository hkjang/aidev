- 과제: `POST /api/v1/assets/{assetId}/split` 이 형식이 틀린 UUID 를 PostgreSQL 에서 500 으로 터뜨리고, 이 엔드포인트에 테스트가 하나도 없다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `openapi.yaml:2263` 은 이 엔드포인트의 오류를 `"400": Invalid split or source does not belong to the original asset` 로만 약속하는데, `splitAsset` 은 `assetID` 와 `source_ids` 를 검사 없이 `UPDATE asset_sources SET asset_id=$1 WHERE id=$2 AND asset_id=$3` 에 그대로 넣는다. `asset_sources.id`·`asset_sources.asset_id` 는 PostgreSQL 에서 UUID 열(`server/migrations/postgres/001_initial.sql:216-218`)이라 형식이 틀린 입력은 22P02 로 쿼리 자체가 실패해 500 이 되고, SQLite 폴백은 TEXT 비교라 0행 → 400 `INVALID_SOURCE` 가 된다. 같은 입력에 두 방언이 다른 코드를 답하고 공개 계약도 어긴다. 게다가 `grep -i split` 로 `server/internal/httpapi/*_test.go` 를 봐도 이 엔드포인트를 통과하는 테스트가 **하나도 없다** — 자산을 새로 만들고 source 증거를 옮기고 `asset_changes`·`audit_logs` 를 쓰는 쓰기 경로가 무검증 상태다.
- 수용 기준:
  1) 형식이 틀린 `assetId` 경로 파라미터(예: `not-a-uuid`)로 split 하면 두 방언 모두 400 (`INVALID_SPLIT`) 이고 트랜잭션을 열기 전에 끝난다.
  2) `source_ids` 에 형식이 틀린 값이 하나라도 있으면 두 방언 모두 400 (`INVALID_SOURCE` 또는 `INVALID_SPLIT` — 하나를 골라 openapi 설명과 맞을 것) 이고, 새 자산(`assets` 행)·`asset_changes`·`audit_logs` 에 아무 것도 남지 않는다.
  3) 정상 split 1건이 201 과 `asset_id` 를 돌려주고, 지정한 source 의 `asset_sources.asset_id` 가 새 자산으로 바뀌며 원래 자산에 `asset_changes(change_type='split')` 1행과 `audit_logs(action='asset.split')` 1행이 남는다 — 이 엔드포인트의 첫 회귀 테스트.
  4) 남의 자산에 속한 source 를 넘기면 기존대로 400 `INVALID_SOURCE` 이고 새 자산이 남지 않는다(기존 동작 보존 증명).
- 건드릴 파일:
  - `server/internal/httpapi/assets.go:779 splitAsset` — 입력 검증 블록(`decodeJSON` 직후, `BeginTx` 전) 에 `uuid.Parse(originalID)` 와 `source_ids` 각 원소의 `uuid.Parse` 를 추가. 트랜잭션 밖이므로 `internalError` 교착 문제와 무관하다. 2026-09-29 회차가 `mergeAssets` 에 넣은 것과 같은 모양(`asset_merge_validation_test.go` 참고)으로 맞출 것.
  - `server/internal/httpapi/asset_split_validation_test.go` (신규) — 실제 라우터 `POST /api/v1/assets/{assetID}/split` 를 통과하는 테스트 4건. 손으로 만든 대역을 새로 만들지 말고 기존 헬퍼를 그대로 재사용할 것(모두 실제로 열어 확인함):
    - `newRuntime(t)` / `testServer(t, runtime)` / `authenticateInitialAdmin(t, server, runtime)` — `asset_merge_validation_test.go:78-80` 가 쓰는 조합. 초기 admin 이 `assets.merge` 권한을 갖는다(merge 테스트가 그 전제로 통과 중).
    - `performAuthenticatedJSON(...)` — `admin_test_helpers_test.go:84`. 세션 쿠키·CSRF 헤더를 붙여 실제 라우터로 요청한다.
    - `insertSoftwareTestAsset(t, server, key, name, "host", "{}", 1, now)` — `software_inventory_test.go:133`. 원래 자산을 만든다.
    - `asset_sources` 행은 `classification_test.go:140-146` 의 INSERT 형태를 그대로 따를 것(`id, asset_id, category, source_asset_id, source_name, payload_json, collected_at, first_seen_at, last_seen_at`). 이 테이블을 넣는 기존 헬퍼는 없으므로 새 테스트 파일 안에 작은 헬퍼로 두면 된다.
    - `countRows(t, server, query, args...)` — `asset_merge_validation_test.go:39`. 같은 패키지이므로 그대로 호출해 `asset_changes`/`audit_logs`/`assets` 행 수를 셀 것. **이름이 겹치는 헬퍼를 새로 정의하면 패키지가 컴파일되지 않는다.**
- 검증 명령:
  - `cd server && go test ./internal/httpapi/ -run 'Split|Merge' -count=1`
  - `cd server && go test ./... && go vet ./... && gofmt -l .` (gofmt 는 빈 출력이어야 함)
  - `cd server && go build ./cmd/invenqor-server`
  - `./scripts/test-postgres.sh` — **이 과제의 핵심 증거**. 기본 `go test` 는 SQLite 폴백이라 22P02 를 못 본다. 기본 포트 55432 가 점유돼 있으면(2026-09-29 회차가 겪음) 포트를 바꿔 돌릴 것.
  - `npx @redocly/cli@2.47.0 lint openapi.yaml` — openapi 설명 문구를 손대는 경우만.
- 위험과 피할 것:
  - `openapi_routes_test.go` 가 라우터와 openapi 를 대조하므로 라우트·operationId 를 바꾸지 말 것. 응답 코드 집합을 바꾸지 않으면(400 그대로) openapi 수정은 불필요하다 — 설명 문구만 다듬고 싶더라도 테스트가 요구하지 않으면 건드리지 않는 쪽이 안전하다.
  - **트랜잭션 안에서 `internalError` 를 부르는 기존 지점(`assets.go` 의 809·820·846·850, `mergeAssets` 의 724·737·745·761·769·773)은 이번에 손대지 말 것.** SQLite 폴백(`storage/runtime.go` `SetMaxOpenConns(1)`)에서 교착하는 별건의 결함이고(보류 아이디어 4/3/M), 같이 고치면 파일·위험이 커진다. 검증은 트랜잭션을 열기 전에 끝내는 것으로 이 경로를 **회피**한다.
  - `mergeAssets` 의 기존 검증 코드를 리팩터해 공용 헬퍼로 뽑으려 하지 말 것 — 두 핸들러의 오류 코드(`INVALID_MERGE` / `INVALID_SPLIT`)와 계약이 다르다.
  - `assets.id` 가 존재하는지까지 확인하는 것은 **이번 범위 밖**이다. `source_ids` 가 원래 자산에 속해야 한다는 기존 `RowsAffected()==0` 검사가 사실상 존재 확인을 대신하고 있으므로, 형식 검증만 넣고 동작 변화를 최소로 유지할 것.
  - 보호 경로(`auth.go`, `auth/`, `migrations/`, `.github/workflows`, `server/internal/webui/dist`) 는 건드리지 않는다. 버전 범프·릴리즈 노트·PDF 재생성도 하지 않는다(별도 릴리즈 회차).
- 차선 후보: 목록 핸들러 `rows.Err()` 확인 — `listAgents`·settings·자산 상세 `sources`/`history`/`relations` 가 `rows.Err()` 를 보지 않아 스캔 중 끊긴 결과를 200 으로 돌려준다(가치 3 / 위험 1 / M). 단, 실패를 주입할 수단이 없어 프로덕션 배선으로 증명하기 어렵다는 것이 지난 회차의 판단이므로, 1순위가 성립하지 않을 때만 고를 것.

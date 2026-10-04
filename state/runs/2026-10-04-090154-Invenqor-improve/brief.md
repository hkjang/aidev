- 과제: `/api/v1/assets/{assetId}` 계열 핸들러가 경로 파라미터를 날것으로 SQL 에 넘겨 PostgreSQL 에서 500 을 내고 비정규형 철자에서 두 방언이 갈리는 것을 기존 `canonicalUUID` 로 막는다 (가치 3 / 위험 2 / 작업량 M)
- 왜: `openapi.yaml` 은 `assetId`·`relationId` 를 `format: uuid` 로 선언하고(`2489-2490` 의 공용 파라미터), `GET`·`PATCH`·`DELETE`·`restore` 의 오류를 404 로만(`2067-2138`, 외부 API 쪽도 같음 `1782-1850`), `history`·`relations` GET 은 200 으로만(`2139-2186`) 약속하는데, `getAsset`·`updateAsset`·`setAssetDeleted`·`assetHistory`·`assetRelations` 는 `chi.URLParam(request,"assetID")` 를 그대로 `WHERE id = $1` 에 넣는다. PostgreSQL 에서 `assets.id`(`migrations/postgres/001_initial.sql:217` 부근 — `id UUID PRIMARY KEY`), `asset_changes.asset_id`(`:268`), `asset_relations.source_asset_id`·`target_asset_id`(`:246`·`:248`)가 모두 UUID 열이고 SQLite 폴백은 TEXT 라, 형식이 틀린 id 는 PostgreSQL 에서 22P02 → `s.internalError` → 500 INTERNAL_ERROR 이고 SQLite 에서는 404/빈 목록이며, 대문자·중괄호·하이픈 없는 32자는 PostgreSQL 이 조용히 접어 자산을 찾아 주지만 SQLite 는 못 찾는다. 2026-10-01(split)·2026-10-03(merge) 회차가 같은 결함을 같은 모양(`canonicalUUID`)으로 고쳤고 남은 가장 큰 구멍이 이 계열이다.
- 수용 기준:
  1) 형식이 틀린 `assetId`(예: `not-a-uuid`)로 `GET /api/v1/assets/{id}`·`PATCH`·`DELETE`·`POST .../restore` 를 부르면 두 방언 모두 404 `ASSET_NOT_FOUND` 를 준다(500 이 사라진다).
  2) `GET /api/v1/assets/{id}/history`·`GET /api/v1/assets/{id}/relations` 는 형식이 틀린 id 에서 두 방언 모두 200 `{"items":[]}` 를 준다(openapi 가 404 를 선언하지 않았으므로 코드 집합을 바꾸지 않는다).
  3) `uuid.Parse` 는 받지만 openapi 의 정규형이 아닌 철자(대문자 36자, `{...}`, 하이픈 없는 32자, `urn:uuid:...`)로 **존재하는** 자산을 조회·수정·삭제·복원하면 두 방언 모두 같은 결과를 준다 — 과제서는 "정규화해서 받아 준다"(200/정상 동작)로 통일할 것을 지시한다. merge·split 이 이미 그 선택을 했으므로 같은 쪽으로 맞춘다.
  4) 테스트가 응답 코드만 보지 말고, 정규화 수용 케이스에서 `assets.deleted_at`·`assets.status` 행 상태와 `asset_changes` 행 수를 직접 세어 실제로 같은 자산에 작용했음을 증명한다.
- 건드릴 파일:
  - `server/internal/httpapi/assets.go` (유일한 프로덕션 파일) — 새 헬퍼 `assetIDParam(request *http.Request) (string, bool)`(`canonicalUUID`(661행) 바로 아래에 두고 `chi.URLParam(request,"assetID")` 를 `canonicalUUID` 에 통과) 추가. 다음 다섯 함수가 날것 대신 이 헬퍼를 쓰게 바꾼다 — `getAsset`(261행, 265행의 `chi.URLParam`), `updateAsset`(370행, 371행), `setAssetDeleted`(489행, 492행 — `deleteAsset`(481)·`restoreAsset`(485)가 공유), `assetHistory`(523행, 529행), `assetRelations`(553행, 554행). 404 를 선언한 앞 세 곳은 `!ok` 에서 `writeAPIError(response, request, 404, "ASSET_NOT_FOUND", "The asset does not exist.")`, 200 만 선언한 뒤 두 곳은 `writeJSON(response, 200, map[string]any{"items": []any{}})` 로 즉시 반환.
  - 이 다섯 핸들러는 `server/internal/httpapi/server.go:407-425`(세션 콘솔 경로)와 `:211-232`(`/api/v1/external/...` API key 경로)에 **둘 다** 배선돼 있다 — 한 번 고치면 두 경로가 같이 고쳐지므로 `server.go` 는 건드리지 않는다. 외부 경로의 openapi 도 404 를 선언하므로(`1782-1850` 확인) 계약이 어긋나지 않는다.
  - `server/internal/httpapi/asset_id_param_test.go`(신규) — 실제 라우터·실제 `Runtime` 으로 위 수용 기준을 검증. 기존 헬퍼 `newRuntime`/`testServer`/`authenticateInitialAdmin`/`performAuthenticatedJSON`(`admin_test_helpers_test.go:84`)·`countRows`(`asset_merge_validation_test.go:39`)를 **재정의하지 말고 그대로 쓸 것**(같은 이름을 새 파일에 다시 선언하면 패키지 컴파일이 깨진다).
  - `openapi.yaml` 은 건드리지 않는다(응답 코드 집합이 그대로다).
- 검증 명령:
  - `cd server && go test ./internal/httpapi/ -run 'Merge|Split|Asset' -count=1` (2026-10-04 실측: 7.6초, 고치기 전 ok — 새 테스트 이름에 `Asset` 을 넣으면 이 선택에 걸린다)
  - `cd server && go test ./... && go vet ./... && gofmt -l .`(gofmt 은 빈 출력이어야 함)
  - `cd server && go build ./cmd/invenqor-server`
  - **필수** `scripts/test-postgres.sh`(docker postgres:17-alpine. 기본 포트 55432 가 점유돼 있을 수 있어 포트를 지정할 것. 기본 `go test` 는 SQLite 폴백이라 22P02 결함을 **보지 못한다** — 이 결함은 PostgreSQL 에서만 500 으로 드러난다)
- 위험과 피할 것:
  - `assetHistory`·`assetRelations` 를 빈 목록 200 으로 바꾸는 것은 **판단**이다. openapi 가 그 두 경로에 404 를 선언하지 않았으므로 코드 집합을 늘리지 않는 쪽을 골랐다. 404 를 새로 내보내려면 `openapi.yaml` 과 `openapi_routes_test.go` 까지 건드려야 하므로 이번 회차에서는 하지 말 것.
  - **트랜잭션 안에서 `s.internalError` 를 부르면 SQLite 폴백이 교착한다**(`SetMaxOpenConns(1)`). 이번 과제의 다섯 핸들러는 `BeginTx` 를 쓰지 않고 `s.database.DB()` 를 직접 쓰므로 그 구역을 비켜간다 — `mergeAssets`/`splitAsset` 의 트랜잭션 구간은 이번에 손대지 말 것.
  - `createAssetRelation`(595 부근, `assetID` 와 본문 `target_asset_id`)·`deleteAssetRelation`(629, `relationID`)·`classification.go:425` 의 관계 검토·`mcp.go` 의 도구 경로는 **이번 범위 밖**이다. 같은 결함이 남아 있을 가능성이 높지만(미확인) 파일·판단 범위를 늘리지 말고 아이디어 파일에 남긴다.
  - 보호 경로(`auth.go`·`auth/`·`migrations/`·`.github/workflows`·`webui/dist`)는 건드리지 않는다. 버전 범프·릴리즈 노트·PDF 재생성은 구현 회차에서 하지 않는다.
  - 손으로 만든 대역(가짜 DB·직접 주입한 객체)으로 증명하면 반려다 — 반드시 실제 라우터를 통과하는 요청으로 증명할 것.
  - **미확인**: PostgreSQL 에서 이 다섯 경로가 실제로 어떤 코드를 내는지는 이번 정찰에서 직접 돌려 보지 않았다(코드 읽기로 500/200·404 발산을 추론했다). 구현자는 고치기 **전에** `scripts/test-postgres.sh` 로 새 테스트를 한 번 돌려 실패 모양을 기록할 것 — 2026-10-01·10-03 회차 모두 그렇게 해서 과제서의 예측을 실측으로 확인했다. `assetRelations` 는 `$1` 을 두 열에 재사용하므로(`assets.go:563`) PostgreSQL 타입 추론이 다르게 걸릴 수 있다 — 여기만 다른 오류 코드가 나와도 조치는 같다.

- 차선 후보: `createAssetRelation` 이 본문 `target_asset_id` 와 경로 `assetID` 를 날것으로 INSERT 해, PostgreSQL 에서 형식이 틀린 uuid 가 22P02 로 터지고 그것을 `err != nil` 가 409 `RELATION_CONFLICT` 로 뭉개는 것(openapi 는 400 `INVALID_RELATION` 을 선언) — 한 함수·한 파일로 끝나는 S 과제.

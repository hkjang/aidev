## 2026-10-04
- 선택: `/api/v1/assets/{assetId}` 계열 다섯 핸들러가 경로 파라미터를 날것으로 SQL 에 넘겨 PostgreSQL 에서 500 을 내고 비정규형 철자에서 두 방언이 갈리던 것을 기존 `canonicalUUID` 로 막음 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `openapi.yaml` 은 공용 파라미터 `{assetId}`(`2489`)를 `format: uuid` 로 선언하고
  `GET`·`PATCH`·`DELETE`·`restore` 의 실패를 404 하나로만(`2067-2138`, 외부 API 도 동일),
  `history`·`relations` GET 은 200 하나로만(`2139-2186`) 약속하는데 `getAsset`·`updateAsset`·
  `setAssetDeleted`·`assetHistory`·`assetRelations` 는 `chi.URLParam(request,"assetID")` 를 그대로
  `WHERE id = $1` 에 넘겼다. `canonicalUUID`(`assets.go:661`)를 통과시키는 헬퍼
  `assetIDParam` 을 그 아래에 두고 다섯 핸들러가 쿼리 **전에** 그것을 쓰게 했다 — 404 를 선언한
  앞 세 곳은 `ASSET_NOT_FOUND`, 200 만 선언한 뒤 두 곳은 빈 목록 200 으로 즉시 반환해 응답 코드
  집합을 넓히지 않는다. 다섯 핸들러 모두 `BeginTx` 밖에서 판정하므로 SQLite 폴백이 교착하는
  구역을 비켜간다. 프로덕션 파일은 `assets.go` 한 개이고 콘솔(`server.go:407-425`)·외부 API
  key(`:211-232`) 경로가 같은 핸들러를 공유해 `server.go` 를 건드리지 않고 두 경로가 함께 고쳐졌다.
  새 테스트 3건은 실제 라우터·실제 `Runtime` 을 지나며 응답 코드뿐 아니라 `assets.status`·
  `deleted_at`·`assets.name` 행 상태와 `asset_changes` 행 수를 직접 센다. 검증:
  `go test ./...`(SQLite 전체 ok)·`go vet ./...`·`gofmt -l .` 빈 출력·
  `go build ./cmd/invenqor-server`·`scripts/test-postgres.sh`(postgres:17-alpine 전체 패키지 ok,
  포트 55494 — 기본 55432 는 점유). 기존 merge 3건·split 7건 그대로 통과. `openapi.yaml` 은
  응답 코드 집합이 그대로라 손대지 않았고 버전 범프·릴리즈 노트·PDF 재생성도 하지 않았다.
- 실패 재현: PostgreSQL 에서 고치기 전 — 과제서의 예측보다 결함이 **더 나빴다**. 500 만이 아니라
  비정규형 철자가 실제로 자산을 고치고 삭제했다.
  `asset_id_param_test.go:95: get with a not a uuid at all asset id ("not-a-uuid") = 500 "INTERNAL_ERROR", want 404 ASSET_NOT_FOUND`,
  `:95: patch with a brace wrapped asset id ("{7bab57a5-…}") = 200 "", want 404` — 응답 본문이
  `"name":"renamed by a bad id"` 로 **이름이 실제로 바뀐 것**을 보여 주고,
  `:95: delete with a unhyphenated asset id ("7bab57a5cf97…") = 200 "", want 404` 는 `{"deleted":true}`,
  `:108: asset_changes rows written by the refused requests = 6, want 0`.
  `history`·`relations` 는 `:131: … = 500, want 200` 이 네 줄(`not-a-uuid`·`urn:uuid:`),
  `:144: /relations with a brace wrapped asset id returned 1 items, want 0`.
  SQLite 폴백에서는 이 두 테스트가 고치기 전에도 **통과**하고 대문자 케이스 한 줄만 실패했다 —
  `asset_id_param_test.go:172: get with an upper-case id = 404 "ASSET_NOT_FOUND", want 200`.
  기본 `go test` 로는 이 결함이 보이지 않는다는 과제서의 경고가 세 회차 연속 실측으로 확인됐다.
- 보류 아이디어: `createAssetRelation` 이 경로 `assetID` 와 본문 `target_asset_id` 를 날것으로
  INSERT 해 PostgreSQL 22P02 를 409 `RELATION_CONFLICT` 로 뭉갠다(openapi 는 400
  `INVALID_RELATION` 선언), `deleteAssetRelation` 의 `relationID` 도 같은 날것 경로 — 이번 회차가
  옆에서 확인해 가장 익은 다음 조각 (가치 3 / 위험 1 / S) · 트랜잭션 안에서 `internalError` 를
  부르면 SQLite 폴백이 교착한다, 막는 것은 여전히 프로덕션 경로로 오류를 터뜨릴 재현 수단
  (가치 4 / 위험 3 / M) · 목록 핸들러 `rows.Err()` 확인, 실패 주입 수단이 없어 증명이 어려움
  (가치 3 / 위험 1 / M) · 콘솔·외부 REST `assetRelations` 페이지네이션, `LIMIT` 가 없다
  (가치 3 / 위험 2 / M) · `classification.go:425` 의 관계 검토·`mcp.go` 도구 인자·`users.go`·
  `api_keys.go`·`agents.go` 의 UUID 경로 파라미터가 아직 미점검 (가치 3 / 위험 2 / M)
- 과제서: 채택 — 지목한 파일·행·헬퍼·검증 명령이 모두 그대로 들어맞았고 '미확인' 으로 남겨 둔
  PostgreSQL 의 실제 응답 코드를 고치기 전에 직접 돌려 확인했다. 다만 수용 기준 3) 의 괄호 주장
  ("merge·split 이 비정규형을 정규화해서 받아 준다")은 코드와 어긋난다 — `canonicalUUID` 는
  `len(value) != 36` 으로 중괄호·하이픈 없는 32자·`urn:uuid:` 를 **거절**하고, merge·split 은 그
  셋에 400 을 주는 기존 테스트를 갖고 있다. 기준의 본래 목적(두 방언이 같은 결과)과 "건드릴 파일"
  이 지정한 `canonicalUUID` 통과를 따라, 선언된 형태인 대문자 36자만 정규화해 받고 나머지 셋은
  두 방언 모두 404(또는 빈 목록 200)로 일치시켰다. `canonicalUUID` 를 느슨하게 바꾸면 merge·split
  의 기존 계약까지 번져 이번 범위를 벗어난다.

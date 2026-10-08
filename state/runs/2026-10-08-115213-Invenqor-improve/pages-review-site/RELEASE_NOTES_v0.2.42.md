# Invenqor Server·Agent v0.2.42 릴리즈 노트

릴리즈 일자: 2026-10-04
호환 Agent: v0.2.42 (Linux·Windows)

이번 릴리즈는 결함 하나를 고칩니다. 자리는 **자산 단건 API 다섯 곳**
(`GET`·`PATCH`·`DELETE /api/v1/assets/{assetId}`,
`POST /api/v1/assets/{assetId}/restore`, `GET .../history`, `GET .../relations`)
이고, 문제는 **`openapi.yaml` 이 `format: uuid` 로 선언한 모양이 아닌 id 를
받았을 때 PostgreSQL 과 SQLite fallback 이 서로 다른 답을 했다**는 점입니다.
공개 계약은 앞의 네 곳의 실패를 404 하나로, `history`·`relations` 는 200
하나로만 약속하는데, 운영 기본값인 PostgreSQL 에서는 계약에 없는 500 이 되거나
**거절해야 할 요청으로 자산을 실제로 고치고 삭제했습니다.** v0.2.40 이 `split`
에서, v0.2.41 이 `merge` 에서 닫은 것과 같은 결함이 `{assetId}` 경로 파라미터를
쓰는 다섯 핸들러에 남아 있었습니다. 이제 경로 파라미터를 쿼리에 닿기 전에
정규형 UUID 로만 받습니다. 콘솔과 Agent 는 손대지 않았습니다.

## 1. 같은 요청에 방언마다 다른 답을 했습니다

`getAsset` · `updateAsset` · `setAssetDeleted`(삭제·복원) · `assetHistory` ·
`assetRelations` 는 `chi.URLParam(request, "assetID")` 가 돌려준 글자를 검사
없이 `WHERE id = $1` 로 넘겼습니다. 이 다섯이 닿는 `assets.id` ·
`asset_changes.asset_id` · `asset_relations.source_asset_id` ·
`target_asset_id` 는 PostgreSQL 에서 모두 UUID 열이고 SQLite fallback 에서는
TEXT 라, 계약이 선언하지 않은 철자가 한쪽에서만 오류가 되거나 조용히
정규화되었습니다. 증상이 세 갈래였고 셋 다 나빴습니다.

- **`urn:uuid:` 표기와 UUID 가 아닌 글자는 PostgreSQL 에서 500 이었습니다.**
  UUID 열 입력 변환이 SQLSTATE 22P02 로 거절해 조회 자체가 실패했고,
  `internalError` 를 거쳐 **계약에 없는 500 `INTERNAL_ERROR`** 가 되었습니다.
  같은 입력에 SQLite fallback 은 TEXT 비교로 0행을 만나 404 를, `history` ·
  `relations` 는 빈 목록 200 을 답했습니다.
- **중괄호·하이픈 없는 32자 표기는 PostgreSQL 에서 자산을 실제로 고쳤습니다.**
  이쪽이 가장 나빴습니다. 두 표기 모두 `openapi.yaml` 이 선언하지 않은 모양인데
  PostgreSQL 이 **조용히 정규화해 실제 행을 찾아냈습니다** — 즉 404 여야 할
  요청으로 `PATCH` 가 자산의 이름을 바꾸고 `DELETE` 가 삭제 표시를 남겼습니다.
  고치기 전 `postgres:17-alpine` 에서 직접 재현했고, 그 경로로 `asset_changes`
  6행이 쓰였습니다. 같은 요청에 SQLite fallback 은 모두 404 였습니다.
- **대문자 36자 UUID 는 방향이 반대였습니다.** 대문자는 계약이 선언한 정규형인데
  PostgreSQL 은 접어서 자산을 찾아 주고 SQLite 의 TEXT 비교는 접지 않아 404 를
  답했습니다 — 계약대로 보낸 호출자가 fallback 에서만 자산을 못 찾았습니다.

콘솔 경로와 API key 로 들어오는 `/api/v1/external/...` 경로가 같은 핸들러를
공유하므로, 두 입구 모두 같은 증상을 보였습니다.

## 2. 쿼리 전에 정규형 UUID 로만 받습니다

v0.2.40 이 `splitAsset` 을 위해 들여오고 v0.2.41 이 `mergeAssets` 에 적용한
`canonicalUUID` 를 이 다섯 핸들러에도 씁니다.

- **`assetIDParam` 헬퍼를 두었습니다.** `chi.URLParam(request, "assetID")` 를
  `canonicalUUID` 에 통과시켜 36자 하이픈 표기만 받고, 아니면 거절을 알립니다.
  다섯 핸들러가 첫 줄에서 이것을 부르므로 **어떤 쿼리도 실행되기 전에**
  판정이 끝납니다.
- **응답 코드 집합을 넓히지 않았습니다.** 404 를 선언한
  `GET`·`PATCH`·`DELETE`·`restore` 는 404 `ASSET_NOT_FOUND` 로, 200 만 선언한
  `history`·`relations` 는 빈 목록 200 으로 즉시 돌아갑니다. 두 방언이 이제
  같은 코드를 답하며, 그 코드는 모두 `openapi.yaml` 이 이미 적어 둔 것입니다.
- **트랜잭션 밖에서 판정합니다.** 다섯 모두 `BeginTx` 전에 거절하므로 거절된
  요청은 `assets` · `asset_changes` · `audit_logs` 어느 것도 건드리지 않고,
  트랜잭션 안에서 오류를 보고하면 SQLite fallback 이 교착하는 구역으로도
  들어가지 않습니다.
- **콘솔과 외부 API 가 함께 고쳐졌습니다.** 두 경로가 같은 핸들러를 공유하므로
  프로덕션 파일 한 개(`server/internal/httpapi/assets.go`)의 변경으로 두 입구가
  같이 닫혔습니다. 라우팅은 손대지 않았습니다.

## 검증

- **고치기 전에 실패를 먼저 확인했습니다.** `scripts/test-postgres.sh` 로
  PostgreSQL 에서 `urn:uuid:` 가 500 `INTERNAL_ERROR` 가 되는 것과, 중괄호·하이픈
  없는 32자 표기로 `PATCH` 가 이름을 바꾸고 `DELETE` 가 삭제 표시를 남기는 것을
  실제로 재현한 뒤 고쳤습니다. 기본 `go test` 는 SQLite fallback 이라 이 결함을
  보지 못합니다 — 방언 차이가 그대로 드러나는 자리라 `scripts/test-postgres.sh`
  없이는 재현이 반쪽입니다.
- **회귀 테스트 3건을 더했습니다.** 대역 없이 실제 라우터와 실제 `Runtime` 을
  지나 다섯 경로를 부르고, 응답 코드만 보지 않고 `assets.status` ·
  `assets.deleted_at` · `assets.name` 행 상태와 `asset_changes` 행 수를 직접
  세어 **거절된 요청이 같은 자산에 아무 것도 하지 않았음**을 확인합니다.
- **두 저장 엔진 모두에서 통과합니다.** `go test ./...`(SQLite fallback)와
  `scripts/test-postgres.sh`(postgres:17-alpine) 전 패키지가 통과합니다.
- **Server.** `go vet` · `gofmt -l` 빈 출력 · `go build` 가 통과합니다. 콘솔
  정적 파일(`server/internal/webui/dist`)은 바뀌지 않았습니다.
- **Agent.** Rust 코드는 이번 릴리즈에서 바뀌지 않았습니다. 릴리즈 커밋에서
  `cargo fmt --all -- --check` · `cargo clippy --all-targets -D warnings` ·
  `cargo test --all-targets` 가 통과합니다.

## 호환성

- **데이터베이스 마이그레이션이 없습니다.** 열을 더하거나 바꾸지 않고, 쿼리에
  닿기 전에 경로 파라미터의 모양을 한 번 더 볼 뿐입니다.
- **OpenAPI 문서가 바뀌지 않았습니다.** 이미 적혀 있던 `format: uuid` 와 각
  경로의 응답 코드 집합을 구현이 뒤늦게 지킨 것이라, 계약 쪽은 손대지
  않았습니다.
- **정규형 UUID 를 쓰던 호출자에게는 아무 변화가 없습니다.** 200·404 응답의
  모양도, 권한과 CSRF 검증도 이전과 같습니다. 달라지는 것은 **계약이 선언하지
  않은 모양으로 id 를 보내던 요청**뿐입니다 — PostgreSQL 에서 500 을 받던
  호출자는 이제 404(또는 빈 목록 200)를 받고, 중괄호나 하이픈 없는 표기로
  **자산이 고쳐지거나 삭제되던** 호출자는 이제 404 를 받습니다.
- **대문자 id 를 쓰던 호출자는 이제 두 방언에서 같게 동작합니다.** 대문자는
  계약이 선언한 정규형이라 계속 받아들이며, 소문자 정규형으로 접은 뒤 조회하므로
  SQLite fallback 에서도 자산을 찾습니다.
- **`merge`·`split` 의 계약은 그대로입니다.** 그 두 곳은 v0.2.40·v0.2.41 이
  정한 대로 비정규형에 400 을 답합니다. 이번 변경은 `{assetId}` 경로 파라미터를
  쓰는 다섯 핸들러에만 닿습니다.
- **알려진 제한.** 트랜잭션 안에서 오류를 보고하면 SQLite fallback 이
  교착하는 문제는 그대로 남아 있습니다. 이번 수정은 트랜잭션을 열기 전에
  거절하므로 이 다섯 경로의 입력 검증이 그 길로 들어가지 않게 되었을 뿐이고,
  다른 핸들러의 `internalError` 는 v0.2.41 과 같습니다. 운영 기본값인
  PostgreSQL 에서는 500 이 되며, fallback 을 쓰는 평가·개발 환경에만
  해당합니다.
- **콘솔이 바뀌지 않았습니다.** 임베디드 정적 파일은 v0.2.41 과 같습니다.
- **Agent 의 동작 변경이 없습니다.** 설정 파일, 큐, 전송 프로토콜은 이전과
  같고 버전 문자열만 올라갑니다. 기존 Agent 는 서두르지 않아도 Server 와의
  호환에 영향이 없습니다.

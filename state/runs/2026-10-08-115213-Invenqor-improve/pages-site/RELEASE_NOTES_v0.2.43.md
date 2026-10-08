# Invenqor Server·Agent v0.2.43 릴리즈 노트

릴리즈 일자: 2026-10-05
호환 Agent: v0.2.43 (Linux·Windows)

이번 릴리즈는 결함 하나를 고칩니다. 자리는 **자산 관계 API 두 곳**
(`POST /api/v1/assets/{assetId}/relations`,
`DELETE /api/v1/assets/{assetId}/relations/{relationId}`)이고, 문제는
**`openapi.yaml` 이 `format: uuid` 로 선언한 모양이 아닌 id 를 받았을 때
PostgreSQL 과 SQLite fallback 이 서로 다른 답을 했다**는 점입니다. 공개 계약은
`POST` 의 실패를 400 으로, `DELETE` 의 실패를 404 로 약속하는데, 운영 기본값인
PostgreSQL 에서는 계약에 없는 500 이 되거나 **거절해야 할 요청으로 관계를 실제로
만들고 끊었습니다.** v0.2.40 이 `split` 에서, v0.2.41 이 `merge` 에서, v0.2.42 가
`{assetId}` 단건 다섯 곳에서 닫은 것과 같은 결함이 관계 쓰기 두 곳에 남아
있었습니다. 이제 세 개의 id — 경로의 `assetId`, 본문의 `target_asset_id`,
경로의 `relationId` — 를 쿼리에 닿기 전에 정규형 UUID 로만 받습니다. 콘솔과
Agent 는 손대지 않았습니다.

## 1. 같은 요청에 방언마다 다른 답을 했습니다

`createAssetRelation` 은 `chi.URLParam(request, "assetID")` 와 본문의
`target_asset_id` 를 빈 값인지만 보고 그대로 `INSERT INTO asset_relations` 로
넘겼고, `deleteAssetRelation` 은 `chi.URLParam(request, "relationID")` 를 검사
없이 `UPDATE asset_relations ... WHERE id=$2` 로 넘겼습니다. 이 두 핸들러가 닿는
`asset_relations.id` · `source_asset_id` · `target_asset_id` 는 PostgreSQL 에서
모두 UUID 열이고 SQLite fallback 에서는 TEXT 라, 계약이 선언하지 않은 철자가
한쪽에서만 오류가 되거나 조용히 정규화되었습니다. 증상이 세 갈래였고 셋 다
나빴습니다.

- **`DELETE` 는 UUID 가 아닌 글자와 `urn:uuid:` 표기에 500 을 답했습니다.**
  UUID 열 입력 변환이 SQLSTATE 22P02 로 거절해 `UPDATE` 자체가 실패했고,
  `s.internalError` 를 거쳐 **이 경로가 선언하지 않은 500 `INTERNAL_ERROR`** 가
  되었습니다. 같은 입력에 SQLite fallback 은 TEXT 비교로 0행을 만나 404
  `RELATION_NOT_FOUND` 를 답했습니다.
- **중괄호·하이픈 없는 32자 표기로 `DELETE` 가 관계를 실제로 끊었습니다.**
  이쪽이 가장 나빴습니다. 두 표기 모두 `openapi.yaml` 이 선언하지 않은 모양인데
  PostgreSQL 이 **조용히 정규화해 실제 행을 찾아냈습니다** — 즉 404 여야 할
  요청이 `valid_to` 를 써서 관계를 끝내고 `relation.delete` 감사 로그를 남겼고,
  같은 요청에 SQLite fallback 은 404 였습니다. `POST` 쪽도 같은 방향으로
  어긋났습니다. 같은 표기로 보낸 `assetId` 와 `target_asset_id` 를 PostgreSQL 이
  정규화해 외래 키를 만족시켜 **400 이어야 할 요청으로 수동 관계를 201 로
  만들었고**, SQLite fallback 은 TEXT 외래 키가 맞는 자산을 못 찾아 409
  `RELATION_CONFLICT` 로 끝냈습니다.
- **대문자 36자 UUID 는 방향이 반대였습니다.** 대문자는 계약이 선언한 정규형인데
  PostgreSQL 은 접어서 자산과 관계를 찾아 주고 SQLite 의 TEXT 비교는 접지 않아,
  `POST` 는 409 로, `DELETE` 는 404 로 끝났습니다 — 계약대로 보낸 호출자가
  fallback 에서만 실패했습니다.

감사 로그도 함께 어긋났습니다. `createAssetRelation` 은 호출자가 보낸 철자를 그대로
`audit_logs.after_json` 에 적었으므로, PostgreSQL 이 정규화해 저장한
`asset_relations` 행과 그 행을 설명하는 감사 기록이 **다른 글자로 같은 자산을
가리켰습니다.**

콘솔 경로와 API key 로 들어오는 `/api/v1/external/...` 경로가 같은 핸들러를
공유하므로, 두 입구 모두 같은 증상을 보였습니다.

## 2. 쿼리 전에 정규형 UUID 로만 받습니다

v0.2.40 이 `splitAsset` 을 위해 들여오고 v0.2.41 이 `mergeAssets` 에,
v0.2.42 가 `assetIDParam` 으로 단건 다섯 곳에 적용한 `canonicalUUID` 를 이 두
핸들러에도 씁니다.

- **`POST` 는 세 id 를 모두 쓰기 전에 봅니다.** 경로의 `assetId` 는 v0.2.42 의
  `assetIDParam` 으로, 본문의 `target_asset_id` 는 `canonicalUUID` 로 36자
  하이픈 표기만 받습니다. 둘 중 하나가 아니면 **`INSERT` 가 실행되기 전에**
  400 `INVALID_RELATION` 으로 돌아갑니다.
- **`DELETE` 는 `relationId` 를 봅니다.** 정규형이 아니면 `UPDATE` 에 닿지 않고
  404 `RELATION_NOT_FOUND` 로 돌아갑니다.
- **응답 코드 집합을 넓히지 않았습니다.** `POST` 는 이미 선언된 400 으로,
  `DELETE` 는 이미 선언된 404 로 거절합니다. 두 방언이 이제 같은 코드를 답하며,
  그 코드는 모두 `openapi.yaml` 이 이미 적어 둔 것입니다.
- **정규화한 값만 저장합니다.** `INSERT` 와 감사 로그가 모두 소문자 36자 표기를
  쓰므로, 대문자로 보낸 요청도 행과 `audit_logs.after_json` 이 같은 글자를
  가리킵니다. `relation.delete` 의 `resource_id` 도 정규형입니다.
- **409 의 뜻이 좁아졌습니다.** 이전에는 PostgreSQL 에서 잘못된 모양의 id 도
  409 `RELATION_CONFLICT` 로 보일 수 있었습니다. 이제 409 는 **없는 자산과
  중복 관계**라는 본래의 뜻만 남고, 모양 문제는 400 으로 갈라집니다.
- **콘솔과 외부 API 가 함께 고쳐졌습니다.** 두 경로가 같은 핸들러를 공유하므로
  프로덕션 파일 한 개(`server/internal/httpapi/assets.go`)의 변경으로 두 입구가
  같이 닫혔습니다. 라우팅은 손대지 않았습니다.

## 검증

- **회귀 테스트 4건을 더했습니다.** 대역 없이 실제 라우터와 실제 `Runtime` 을
  지나 콘솔·외부 두 입구로 같은 요청을 보내고, 응답 코드만 보지 않고
  `asset_relations` 행 수와 `source_asset_id` · `target_asset_id` · `valid_to`
  행 상태, `audit_logs` 행 수를 직접 세어 **거절된 요청이 아무 것도 쓰지
  않았음**을 확인합니다. 같은 `DELETE` 를 두 번 보내 두 번째가 `valid_to` 를
  덮어쓰지 않는 것과, 409 를 남겨 두어야 하는 세 경우(없는 source, 없는 target,
  중복)가 여전히 409 인 것도 함께 봅니다.
- **두 저장 엔진 모두에서 통과합니다.** `go test ./...`(SQLite fallback)와
  `scripts/test-postgres.sh`(postgres:17-alpine) 전 패키지가 통과합니다. 기본
  `go test` 는 SQLite fallback 이라 이 결함을 보지 못합니다 — 방언 차이가 그대로
  드러나는 자리라 `scripts/test-postgres.sh` 없이는 재현이 반쪽입니다.
- **Server.** `go vet` · `gofmt -l` 빈 출력 · `go build` 가 통과합니다. 콘솔
  정적 파일(`server/internal/webui/dist`)은 바뀌지 않았습니다.
- **OpenAPI.** `@redocly/cli lint openapi.yaml` 이 통과합니다.
- **Agent.** Rust 코드는 이번 릴리즈에서 바뀌지 않았습니다. 릴리즈 커밋에서
  `cargo fmt --all -- --check` · `cargo clippy --all-targets -D warnings` ·
  `cargo test --all-targets` 가 통과합니다.

## 호환성

- **데이터베이스 마이그레이션이 없습니다.** 열을 더하거나 바꾸지 않고, 쿼리에
  닿기 전에 id 의 모양을 한 번 더 볼 뿐입니다.
- **OpenAPI 문서가 바뀌지 않았습니다.** 이미 적혀 있던 `format: uuid` 와 두
  경로의 응답 코드 집합을 구현이 뒤늦게 지킨 것이라, 계약 쪽은 손대지
  않았습니다.
- **정규형 UUID 를 쓰던 호출자에게는 아무 변화가 없습니다.** 201·200·404·409
  응답의 모양도, 권한과 CSRF 검증도 이전과 같습니다. 달라지는 것은 **계약이
  선언하지 않은 모양으로 id 를 보내던 요청**뿐입니다 — `DELETE` 에서 500 을 받던
  호출자는 이제 404 를 받고, 중괄호나 하이픈 없는 표기로 **관계가 만들어지거나
  끊어지던** 호출자는 이제 400 또는 404 를 받습니다.
- **대문자 id 를 쓰던 호출자는 이제 두 방언에서 같게 동작합니다.** 대문자는
  계약이 선언한 정규형이라 계속 받아들이며, 소문자 정규형으로 접은 뒤 쓰므로
  SQLite fallback 에서도 관계를 만들고 끊습니다.
- **감사 로그의 글자가 달라질 수 있습니다.** 대문자나 비정규형으로 보낸 요청의
  `audit_logs.after_json` 과 `resource_id` 가 이제 소문자 36자 표기로 적힙니다.
  이미 쌓인 기록은 고치지 않습니다 — 과거 감사 기록을 글자 그대로 맞추어 찾는
  외부 도구가 있다면 정규형으로 한 번 더 찾아야 합니다.
- **관계 읽기와 자동 분류는 그대로입니다.** `GET .../relations` 는 v0.2.42 가
  정한 대로 동작하고, `classify` 가 만드는 자동 관계와 `ingest` ·
  `softwarecatalog` 의 관계 정리는 호출자 입력을 받지 않아 이번 변경에 닿지
  않습니다.
- **`merge`·`split`·단건 다섯 곳의 계약은 그대로입니다.** 그 일곱 곳은
  v0.2.40 · v0.2.41 · v0.2.42 가 정한 대로 답합니다. 이번 변경은 관계 쓰기 두
  곳에만 닿습니다.
- **알려진 제한.** 트랜잭션 안에서 오류를 보고하면 SQLite fallback 이
  교착하는 문제는 그대로 남아 있습니다. 다른 핸들러의 `internalError` 는
  v0.2.42 와 같습니다. 운영 기본값인 PostgreSQL 에서는 500 이 되며, fallback 을
  쓰는 평가·개발 환경에만 해당합니다.
- **콘솔이 바뀌지 않았습니다.** 임베디드 정적 파일은 v0.2.42 와 같습니다.
- **Agent 의 동작 변경이 없습니다.** 설정 파일, 큐, 전송 프로토콜은 이전과
  같고 버전 문자열만 올라갑니다. 기존 Agent 는 서두르지 않아도 Server 와의
  호환에 영향이 없습니다.

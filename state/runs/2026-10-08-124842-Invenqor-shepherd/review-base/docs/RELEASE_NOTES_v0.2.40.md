# Invenqor Server·Agent v0.2.40 릴리즈 노트

릴리즈 일자: 2026-10-01
호환 Agent: v0.2.40 (Linux·Windows)

이번 릴리즈는 결함 하나를 고칩니다. 자리는 **자산 분리 API
`POST /api/v1/assets/{assetId}/split`** 이고, 문제는 **`openapi.yaml` 이
`format: uuid` 로 선언한 모양이 아닌 id 를 받았을 때 PostgreSQL 과 SQLite
fallback 이 서로 다른 답을 했다**는 점입니다. 공개 계약은 이 엔드포인트의 오류를
400 하나로만 약속하는데, 운영 기본값인 PostgreSQL 에서는 500 이 되거나 **거절해야
할 요청을 201 로 수행했습니다.** 이제 경로 파라미터와 `source_ids` 의 모든 원소를
트랜잭션을 열기 전에 정규형 UUID 로만 받고, 아니면 아무 것도 쓰지 않은 채 400 으로
거절합니다. 콘솔과 Agent 는 손대지 않았습니다.

## 1. split 이 같은 요청에 방언마다 다른 코드를 답했습니다

`splitAsset` 은 `{assetId}` 경로 파라미터와 `source_ids` 를 검사 없이
`UPDATE asset_sources SET asset_id=$1 WHERE id=$2 AND asset_id=$3` 에 그대로
넣었습니다. `asset_sources.id` 와 `asset_sources.asset_id` 는 PostgreSQL 에서
UUID 열이고 SQLite fallback 에서 TEXT 라, 모양이 다른 id 는 한쪽에서만 조회
자체가 오류가 되거나 조용히 정규화되었습니다. 증상이 두 갈래였고 둘 다
나빴습니다.

- **UUID 가 아닌 id 는 PostgreSQL 에서 500 이었습니다.** `not-a-uuid` 같은 입력은
  UUID 열 입력 변환이 SQLSTATE 22P02 로 거절해 쿼리 자체가 실패했고,
  `internalError` 를 거쳐 **계약에 없는 500 `INTERNAL_ERROR`** 가 되었습니다.
  같은 입력에 SQLite fallback 은 TEXT 비교로 0행을 만나 400 을 답했습니다.
- **`uuid.Parse` 가 받아 주는 비정규 표기는 PostgreSQL 에서 분리를
  수행했습니다.** 이쪽이 더 나빴습니다. `urn:uuid:` 접두, 중괄호 포장,
  하이픈 없는 32자 표기는 모두 `openapi.yaml` 이 선언하지 않은 모양인데
  `uuid.Parse` 는 세 가지를 모두 통과시킵니다. PostgreSQL 은 `urn:` 표기만
  22P02 로 거절하고 **중괄호와 하이픈 없는 표기는 조용히 정규화해 실제 행을
  찾아냈습니다** — 즉 거절되어야 할 요청이 201 로 성공하며 `asset_sources` 를
  옮기고 새 자산과 `asset_changes` · `audit_logs` 를 남겼습니다. 같은 요청에
  SQLite fallback 은 세 가지 모두 400 이었습니다.
- **대문자 UUID 는 기록되는 글자가 방언마다 달랐습니다.** 대문자는 계약이
  선언한 정규형이고 PostgreSQL 은 접어서 같은 행을 찾지만, SQLite 의 TEXT
  비교는 접지 않습니다. 보낸 글자가 그대로 `asset_changes.after_json` 과
  감사 로그에 들어가 같은 분리가 저장소에 따라 다르게 적혔습니다.

이제 **`canonicalUUID` 가 36자 하이픈 표기만 받습니다.**

- **모양 검사를 트랜잭션 전에 합니다.** `decodeJSON` 직후·`BeginTx` 전에
  `{assetId}` 와 `source_ids` 각 원소를 확인하고, 아니면 각각 400
  `INVALID_SPLIT` · `INVALID_SOURCE` 로 끝냅니다. 트랜잭션을 열기 전이므로
  `assets` · `asset_sources` · `asset_changes` · `audit_logs` 어느 것도
  건드리지 않습니다.
- **`uuid.Parse` 단독으로는 모양 검사가 아닙니다.** 그 godoc 이 직접 그렇게
  적어 둔 대로입니다. `canonicalUUID` 는 파싱에 더해 **길이 36자**를 함께 보아
  `urn:uuid:` · 중괄호 · 하이픈 없는 표기를 모두 거절합니다.
- **SQL 에는 정규형만 들어갑니다.** 호출자가 보낸 글자가 아니라 파싱 결과를
  소문자 정규형으로 되돌려 쓰므로, 대문자로 보낸 id 도 두 방언에서 같게
  동작하고 `asset_changes.after_json` 에 정규형으로 적힙니다.
- **폴백의 동작은 바뀌지 않습니다.** 오류 코드는 SQLite fallback 이 이미
  답하던 400 `INVALID_SPLIT` · `INVALID_SOURCE` 를 그대로 두었으므로, 달라지는
  것은 PostgreSQL 쪽의 500 과 잘못된 201 뿐입니다.

## 검증

- **이 엔드포인트는 테스트가 하나도 없었습니다.** 대역 없이 실제 라우터로
  `POST /api/v1/assets/{assetId}/split` 를 부르는 회귀 테스트 7건을
  추가했습니다 — UUID 가 아닌 `assetId`, UUID 가 아닌 `source_ids`, 네 가지
  비정규 표기(`urn:uuid:` · 중괄호 · 하이픈 없음 · UUID 아님)를 `assetId` 와
  `source_ids` 양쪽에 각각 적용, 대문자 `source_ids` 의 정규형 접힘, 정상 분리
  1건(새 자산 · source 이동 · `asset_changes` · `audit_logs` 확인), 남의 자산
  source 거절(기존 동작 보존). 거절 경로는 **source 가 원래 자산에 그대로
  남아 있는 것과 아무 것도 기록되지 않은 것**까지 확인합니다.
- **고치기 전에 실패를 먼저 확인했습니다.** PostgreSQL 에서 UUID 가 아닌 두
  입력이 500 `INTERNAL_ERROR` 로, 중괄호·하이픈 없는 표기가 201 로 분리를
  수행하는 것을 확인한 뒤 고쳤습니다. SQLite fallback 에서는 같은 테스트 일부가
  고치기 전에도 통과했습니다 — 방언 차이가 그대로 드러나는 자리라,
  `scripts/test-postgres.sh` 없이는 이 결함의 재현이 반쪽입니다.
- **두 저장 엔진 모두에서 통과합니다.** `go test ./...`(SQLite fallback)와
  `scripts/test-postgres.sh`(postgres:17-alpine) 전 패키지가 통과합니다.
- **Server.** `go vet` · `gofmt -l` 빈 출력 · `go build` 가 통과합니다. 콘솔
  정적 파일(`server/internal/webui/dist`)은 바뀌지 않았습니다.
- **Agent.** Rust 코드는 이번 릴리즈에서 바뀌지 않았습니다. 릴리즈 커밋에서
  `cargo fmt --all -- --check` · `cargo clippy --all-targets -D warnings` ·
  `cargo test --all-targets` 가 통과합니다.

## 호환성

- **데이터베이스 마이그레이션이 없습니다.** 열을 더하거나 바꾸지 않고, 쿼리에
  닿기 전에 입력의 모양을 한 번 더 볼 뿐입니다.
- **OpenAPI 문서가 바뀌지 않았습니다.** 이미 적혀 있던 `format: uuid` 와
  `"400": Invalid split or source does not belong to the original asset` 을
  구현이 뒤늦게 지킨 것이라, 계약 쪽은 손대지 않았습니다.
- **정상 분리의 응답은 그대로입니다.** 201 과 `asset_id` 의 모양도,
  `assets.merge` 권한과 CSRF 검증도 이전과 같습니다. 달라지는 것은 **계약이
  선언하지 않은 모양으로 id 를 보내던 요청**뿐입니다 — PostgreSQL 에서 500 을
  받던 호출자는 이제 400 을 받고, 중괄호나 하이픈 없는 표기로 **분리가 되던**
  호출자는 이제 400 을 받습니다. 정규형 36자 UUID 를 보내던 호출자에게는
  아무 변화가 없습니다.
- **알려진 제한.** 트랜잭션 안에서 오류를 보고하면 SQLite fallback 이
  교착하는 문제는 그대로 남아 있습니다. 이번 수정은 트랜잭션을 열기 전에
  거절하므로 split 의 입력 검증 경로가 그 길로 들어가지 않게 되었을 뿐이고,
  `splitAsset` 의 트랜잭션 안쪽과 ingest 등 다른 핸들러의 `internalError` 는
  v0.2.39 와 같습니다. 운영 기본값인 PostgreSQL 에서는 500 이 되며, fallback 을
  쓰는 평가·개발 환경에만 해당합니다.
- **콘솔이 바뀌지 않았습니다.** 임베디드 정적 파일은 v0.2.39 와 같습니다.
- **Agent 의 동작 변경이 없습니다.** 설정 파일, 큐, 전송 프로토콜은 이전과
  같고 버전 문자열만 올라갑니다. 기존 Agent 는 서두르지 않아도 Server 와의
  호환에 영향이 없습니다.

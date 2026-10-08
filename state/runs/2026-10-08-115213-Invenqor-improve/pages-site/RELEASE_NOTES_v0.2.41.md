# Invenqor Server·Agent v0.2.41 릴리즈 노트

릴리즈 일자: 2026-10-03
호환 Agent: v0.2.41 (Linux·Windows)

이번 릴리즈는 결함 하나를 고칩니다. 자리는 **자산 병합 API
`POST /api/v1/assets/merge`** 이고, 문제는 **`openapi.yaml` 이 `format: uuid` 로
선언한 모양이 아닌 id 를 받았을 때 PostgreSQL 과 SQLite fallback 이 서로 다른
답을 했다**는 점입니다. 공개 계약은 이 엔드포인트의 오류를 400 하나로만
약속하는데, 운영 기본값인 PostgreSQL 에서는 500 이 되거나 **거절해야 할 요청을
200 으로 수행했습니다.** v0.2.40 이 `split` 에서 닫은 것과 같은 결함이 `merge`
에 남아 있었습니다. 이제 `primary_id` 와 `secondary_ids` 의 모든 원소를
트랜잭션을 열기 전에 정규형 UUID 로만 받고, 아니면 아무 것도 쓰지 않은 채
400 으로 거절합니다. 콘솔과 Agent 는 손대지 않았습니다.

## 1. merge 가 같은 요청에 방언마다 다른 코드를 답했습니다

`mergeAssets` 는 `primary_id` 와 `secondary_ids` 를 `uuid.Parse` 모양 검사
(`missingAssetID`)만 거쳐 보낸 글자 그대로 SQL 로 넘겼습니다. `assets.id` 와
`asset_sources.asset_id` 는 PostgreSQL 에서 UUID 열이고 SQLite fallback 에서
TEXT 라, `uuid.Parse` 가 받아 주는 비정규형 철자는 한쪽에서만 조회 자체가
오류가 되거나 조용히 정규화되었습니다. 증상이 세 갈래였고 셋 다 나빴습니다.

- **`urn:uuid:` 표기는 PostgreSQL 에서 500 이었습니다.** UUID 열 입력 변환이
  SQLSTATE 22P02 로 거절해 조회 자체가 실패했고, `internalError` 를 거쳐
  **계약에 없는 500 `INTERNAL_ERROR`** 가 되었습니다. 같은 입력에 SQLite
  fallback 은 TEXT 비교로 0행을 만나 400 을 답했습니다.
- **중괄호·하이픈 없는 32자 표기는 PostgreSQL 에서 병합을 수행했습니다.**
  이쪽이 더 나빴습니다. 두 표기 모두 `openapi.yaml` 이 선언하지 않은 모양인데
  `uuid.Parse` 는 통과시키고, PostgreSQL 은 **조용히 정규화해 실제 행을
  찾아냈습니다** — 즉 거절되어야 할 요청이 200 으로 성공하며 `asset_sources`
  를 옮기고 `assets.status='merged'` 와 `asset_changes` · `audit_logs` 를
  남겼습니다. 같은 요청에 SQLite fallback 은 모두 400 이었습니다.
- **대문자 UUID 는 병합 결과를 MCP 가 찾지 못하게 만들었습니다.** 대문자는
  계약이 선언한 정규형이고 PostgreSQL 은 접어서 같은 행을 찾지만, SQLite 의
  TEXT 비교는 접지 않습니다. 더구나 보낸 대문자가 그대로
  `asset_changes.after_json` 과 `audit_logs` 에 들어가, 문자 일치로 묻는 MCP
  `merged_into` 조회(`mcp.go:747`·`752`, 두 방언 모두)가 **병합된 자산을
  못 찾았습니다** — v0.2.38 이 세운 안내가 이 경로에서만 조용히 깨져
  있었습니다.

이제 **`canonicalUUID` 가 36자 하이픈 표기만 받습니다.**

- **모양 검사를 트랜잭션 전에 합니다.** `decodeJSON` 직후·`BeginTx` 전에
  `primary_id` 와 `secondary_ids` 각 원소를 확인하고, 아니면 400
  `INVALID_MERGE` 로 끝냅니다. 트랜잭션을 열기 전이므로 `assets` ·
  `asset_sources` · `asset_changes` · `audit_logs` 어느 것도 건드리지
  않습니다.
- **정규형만 아래로 흐릅니다.** 존재 확인 쿼리, 자기 병합 스킵,
  `asset_sources` 이동, `asset_changes` 의 `secondary_ids` metadata, 감사
  기록이 모두 지역 변수 `primaryID` · `secondaryIDs` 만 읽습니다. 호출자가
  보낸 철자가 저장소에 남는 자리는 더 이상 없으므로, 대문자로 병합해도 MCP
  `merged_into` 가 primary 를 가리킵니다.
- **`missingAssetID` 의 모양 검사를 지웠습니다.** 호출자가 하나뿐이고 그
  호출자가 호출 전에 정규화를 끝내므로 더는 닿지 않는 코드였습니다. 그 전제를
  주석으로 적어 두었습니다.
- **폴백의 동작은 바뀌지 않습니다.** 오류 코드는 SQLite fallback 이 이미
  답하던 400 `INVALID_MERGE` 를 그대로 두었으므로, 달라지는 것은 PostgreSQL
  쪽의 500 과 잘못된 200 뿐입니다.

## 검증

- **고치기 전에 실패를 먼저 확인했습니다.** `scripts/test-postgres.sh` 로
  PostgreSQL 에서 `urn:uuid:` 가 500 `INTERNAL_ERROR` 로, 중괄호·하이픈 없는
  32자 표기가 200 으로 병합을 수행하는 것, 대문자 철자가 PostgreSQL 에서만
  병합되는 것을 실제로 재현한 뒤 고쳤습니다. SQLite fallback 에서는 같은
  테스트 일부가 고치기 전에도 통과합니다 — 방언 차이가 그대로 드러나는
  자리라, `scripts/test-postgres.sh` 없이는 이 결함의 재현이 반쪽입니다.
- **회귀 테스트 4건을 더했습니다.** 대역 없이 실제 라우터로
  `POST /api/v1/assets/merge` 를 부릅니다 — 비정규형 `primary_id`, 비정규형
  `secondary_ids`(`urn:uuid:` · 중괄호 · 하이픈 없음 각각), 대문자 id 의
  정규형 접힘, 그리고 대문자로 병합한 뒤 MCP `asset_get` 이 `merged_into` 로
  primary 를 가리키는 것. 응답 코드만 보지 않고 `assets.status` ·
  `asset_sources.asset_id` · `asset_changes.after_json` · `audit_logs` 행
  수를 직접 셉니다. 거절 경로는 **아무 것도 기록되지 않은 것**까지
  확인합니다. v0.2.39 가 추가한 존재 확인 테스트 3건은 그대로 통과합니다.
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
  `"400": Primary or secondary assets missing` 을 구현이 뒤늦게 지킨 것이라,
  계약 쪽은 손대지 않았습니다. 응답 코드 집합도 그대로입니다.
- **정상 병합의 응답은 그대로입니다.** 200 과 본문의 모양도, `assets.merge`
  권한과 CSRF 검증도 이전과 같습니다. 달라지는 것은 **계약이 선언하지 않은
  모양으로 id 를 보내던 요청**뿐입니다 — PostgreSQL 에서 500 을 받던 호출자는
  이제 400 을 받고, 중괄호나 하이픈 없는 표기로 **병합이 되던** 호출자는 이제
  400 을 받습니다. 정규형 36자 UUID 를 보내던 호출자에게는 아무 변화가
  없습니다.
- **대문자 id 로 병합하던 호출자는 이제 두 방언에서 같게 동작합니다.** 대문자는
  계약이 선언한 정규형이라 계속 받아들이지만, 저장되는 글자가 소문자 정규형으로
  바뀝니다. 이전에 PostgreSQL 에서 대문자로 병합해 둔 자산의
  `asset_changes.after_json` 과 `audit_logs` 에는 대문자가 남아 있고, 그
  과거 행을 고치지는 않습니다 — MCP `merged_into` 안내는 `assets` 테이블을
  보므로 과거 병합에도 영향이 없습니다.
- **알려진 제한.** 트랜잭션 안에서 오류를 보고하면 SQLite fallback 이
  교착하는 문제는 그대로 남아 있습니다. 이번 수정은 트랜잭션을 열기 전에
  거절하므로 merge 의 입력 검증 경로가 그 길로 들어가지 않게 되었을 뿐이고,
  `mergeAssets` 의 트랜잭션 안쪽과 ingest 등 다른 핸들러의 `internalError` 는
  v0.2.40 과 같습니다. 운영 기본값인 PostgreSQL 에서는 500 이 되며, fallback 을
  쓰는 평가·개발 환경에만 해당합니다.
- **콘솔이 바뀌지 않았습니다.** 임베디드 정적 파일은 v0.2.40 과 같습니다.
- **Agent 의 동작 변경이 없습니다.** 설정 파일, 큐, 전송 프로토콜은 이전과
  같고 버전 문자열만 올라갑니다. 기존 Agent 는 서두르지 않아도 Server 와의
  호환에 영향이 없습니다.

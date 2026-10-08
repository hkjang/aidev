# Invenqor Server·Agent v0.2.39 릴리즈 노트

릴리즈 일자: 2026-09-29
호환 Agent: v0.2.39 (Linux·Windows)

이번 릴리즈는 결함 하나를 고칩니다. 자리는 **자산 병합 API
`POST /api/v1/assets/merge`** 이고, 문제는 **존재하지 않는 자산 UUID 를 받아도
그대로 병합을 진행했다**는 점입니다. `openapi.yaml` 은 이 경우를 400 으로
공개 계약에 적어 두었지만 구현에는 존재 확인이 아예 없었습니다. 이제 트랜잭션을
연 직후 primary 와 모든 secondary 가 실제 자산인지 확인하고, 하나라도 없으면
아무 것도 쓰지 않고 400 `INVALID_MERGE` 로 거절합니다. 콘솔과 Agent 는
손대지 않았습니다.

## 1. 병합이 없는 자산을 받아도 성공했다고 답했습니다

`mergeAssets` 는 요청의 `primary_id` · `secondary_ids` 가 자산인지 한 번도
묻지 않고 곧바로 `asset_sources` 와 `assets` 를 갱신했습니다. 그래서 오타가 난
UUID, 이미 하드 삭제된 자산의 UUID, 다른 환경에서 복사해 온 UUID 가 들어오면
어느 쪽이냐에 따라 증상이 달랐고, 셋 다 나빴습니다.

- **없는 secondary 는 200 이었습니다.** 갱신할 행이 없으니 `UPDATE` 는 0행을
  건드리고 끝났는데, 그 뒤로 `asset_changes`(`change_type='merged'`)와
  `audit_logs`(`action='asset.merge'`, `result='success'`)에는 **병합했다는
  기록이 남았습니다.** 감사 로그를 읽는 사람에게는 일어나지 않은 병합이
  일어난 것으로 보였고, 호출자는 성공 응답을 받았습니다.
- **없는 primary 는 PostgreSQL 에서 500 이었습니다.** `asset_changes.asset_id`
  의 외래 키가 트랜잭션 도중에 거절했습니다 — 데이터가 깨지지는 않았지만,
  계약이 약속한 400 대신 서버 오류였습니다.
- **SQLite fallback 에서는 응답이 아예 돌아오지 않았습니다.** 위 500 을
  보고하는 `internalError` 가 진단 레코드를 쓰려고 **새 커넥션**을 요구하는데,
  열려 있는 트랜잭션이 fallback 의 유일한 커넥션(`SetMaxOpenConns(1)`)을 쥐고
  있어 서로를 기다렸습니다. 요청은 타임아웃까지 매달렸습니다.

이제 **트랜잭션을 연 직후 양쪽을 모두 확인합니다.**

- **하나라도 없으면 아무 것도 쓰지 않습니다.** primary 와 모든 secondary 를
  단건 `SELECT id FROM assets WHERE id=$1` 로 확인하고, 없는 것이 하나라도
  있으면 `asset_sources` · `assets` · `asset_changes` · `audit_logs` 어느 것도
  건드리지 않은 채 400 `INVALID_MERGE` 로 끝납니다. 거짓 감사 기록이 남지
  않습니다.
- **이미 병합된 자산을 다시 secondary 로 주는 동작은 그대로입니다.** 확인은
  행이 있는지만 묻고 `deleted_at` 이나 `status` 를 보지 않으므로, 체인 병합과
  재병합은 이전과 똑같이 동작합니다.
- **UUID 가 아닌 id 도 같은 400 입니다.** `assets.id` 는 PostgreSQL 에서 UUID
  열, SQLite fallback 에서 TEXT 라 형식이 틀린 id 는 한쪽에서만 조회 자체가
  오류가 됩니다. 조회 전에 모양을 먼저 보아 **두 방언이 같은 입력에 같은 코드를
  답합니다.**
- **오류 보고가 교착하지 않습니다.** 확인 질의가 실패하면 `internalError` 를
  부르기 전에 트랜잭션을 먼저 되돌려 커넥션을 놓아 줍니다 — 위의 SQLite
  무한 대기와 같은 길로 들어가지 않기 위해서입니다.

## 검증

- **실제 병합 경로로 테스트했습니다.** 대역 없이 REST
  `POST /api/v1/assets/merge` 를 불러 없는 primary(400 · 자산 상태 불변),
  없는 secondary 가 섞인 요청(400 · 정상 secondary 도 병합되지 않음 ·
  `asset_changes` 와 `audit_logs` 에 기록 없음), UUID 가 아닌 id(400) 를
  확인합니다. 수정 전에는 두 번째가 200 으로, 첫 번째가 SQLite 에서 응답 없이
  실패했습니다.
- **두 저장 엔진 모두에서 통과합니다.** `go test ./...`(SQLite fallback)와
  `scripts/test-postgres.sh`(postgres:17-alpine) 전 패키지가 통과합니다.
- **Server.** `go vet` · `gofmt -l` 빈 출력 · `go build` 가 통과합니다. 콘솔
  정적 파일(`server/internal/webui/dist`)은 바뀌지 않았습니다.
- **Agent.** Rust 코드는 이번 릴리즈에서 바뀌지 않았습니다. 릴리즈 커밋에서
  `cargo fmt --all -- --check` · `cargo clippy --all-targets -D warnings` ·
  `cargo test --all-targets` 가 통과합니다.

## 호환성

- **데이터베이스 마이그레이션이 없습니다.** 열을 더하거나 바꾸지 않고, 쓰기
  전에 읽기 한 번을 더 할 뿐입니다.
- **OpenAPI 문서가 바뀌지 않았습니다.** 이미 적혀 있던 `"400": Primary or
  secondary assets missing` 을 구현이 뒤늦게 지킨 것이라, 계약 쪽은 손대지
  않았습니다.
- **정상 병합의 응답은 그대로입니다.** 200 의 모양도, `assets.merge` 권한과
  CSRF 검증도 이전과 같습니다. 달라지는 것은 **이전에 200 이었던 잘못된
  요청**뿐이므로, 없는 UUID 를 보내던 호출자가 있었다면 이제 400 을 받습니다 —
  그 200 은 아무 것도 병합하지 않은 200 이었습니다.
- **알려진 제한.** 트랜잭션 안에서 오류를 보고하면 SQLite fallback 이
  교착하는 문제는 병합 경로에서만 막았습니다. 같은 모양이
  `POST /api/v1/assets/{assetId}/split` 과 ingest 등 트랜잭션 안에서
  `internalError` 를 부르는 다른 핸들러에 남아 있습니다. 운영 기본값인
  PostgreSQL 에서는 500 이 되며, fallback 을 쓰는 평가·개발 환경에만
  해당합니다.
- **콘솔이 바뀌지 않았습니다.** 임베디드 정적 파일은 v0.2.38 과 같습니다.
- **Agent 의 동작 변경이 없습니다.** 설정 파일, 큐, 전송 프로토콜은 이전과
  같고 버전 문자열만 올라갑니다. 기존 Agent 는 서두르지 않아도 Server 와의
  호환에 영향이 없습니다.

# Invenqor Server·Agent v0.2.38 릴리즈 노트

릴리즈 일자: 2026-09-20
호환 Agent: v0.2.38 (Linux·Windows)

이번 릴리즈는 결함 하나를 고칩니다. 자리는 **MCP 도구 `asset_get`** 이고, 문제는
콘솔에서 다른 자산으로 **병합된 자산의 UUID 로 물으면 존재한 적 없는 UUID 와
글자까지 같은 `asset not found` 만 돌려주었다**는 점입니다. 이제 병합된 자산은
`merged_into` 로 어느 자산(primary)으로 합쳐졌는지와 언제 합쳐졌는지를 답하므로,
모델은 그 UUID 로 다시 불러 지금의 자산에 닿을 수 있습니다. 콘솔과 Agent 는
손대지 않았습니다.

## 1. MCP `asset_get` 이 병합된 자산을 없는 자산과 같게 답했습니다

콘솔에서 자산 A 를 B 로 병합하면 A 는 `status='merged'` 가 되고 `deleted_at` 이
찍힙니다. REST `GET /api/v1/assets/{id}` 와 콘솔 상세 화면은 그 자산을 여전히
"병합됨" 으로 보여 주지만, MCP `asset_get` 은 `deleted_at IS NULL` 로만 걸러
**`asset not found`** 라고 답했습니다. 이전 대화, `asset_search` 결과, 관계 edge
에 남아 있던 A 의 UUID 로 다시 묻는 모델은 자산이 사라졌다고 결론냈고, B 로
가는 길은 어디에도 없었습니다.

이제 live 조회가 비면 **그 UUID 가 병합된 자산인지 한 번 더 봅니다.**

- **병합된 secondary 는 primary 를 안내합니다.** `asset` 키 없이
  `{ "merged_into": "<primary UUID>", "merged_at": "<RFC 3339>", "message": "..." }`
  를 돌려줍니다. `merged_into` 로 `asset_get` 을 다시 부르면 상세가 옵니다.
  primary 는 예전 그대로 `asset` 으로 답합니다.
- **한 단계만 안내합니다.** A→B→C 로 두 번 병합되었으면 A 는 B 를 받고, B 를 다시
  물으면 C 를 받습니다. 체인을 따라가지 않으므로 순환이 생길 수 없습니다.
- **존재하지 않는 UUID 는 예전 그대로 `asset not found` 입니다.** 병합 표시가
  없는 삭제 자산도 같습니다.
- **조회 실패는 숨기지 않습니다.** primary 는 `asset_changes` 의 `merged` 변경
  행(`after_json.secondary_ids`)에서 찾는데, 이 열은 PostgreSQL 에서 JSONB,
  SQLite fallback 에서 TEXT 라 엔진마다 다른 JSON 연산(`@>` · `json_each`)으로
  묻습니다. 그 문장이 실패하면 오류를 그대로 돌려주고 `asset not found` 로 접지
  않습니다 — SQLite 테스트만 통과한 채 운영에서 조용히 틀리는 길을 막기
  위해서입니다.

도구 설명(`Description`)에 병합 자산의 응답 모양을 적어 MCP 클라이언트가 도구
목록에서 바로 읽을 수 있게 했고, API·MCP 가이드의 도구 표 `asset_get` 행도 같은
내용으로 고쳤습니다.

## 검증

- **실제 병합 경로로 테스트했습니다.** 대역 없이 REST `POST /api/v1/assets/merge`
  로 병합한 뒤 `asset_get` 을 불러 secondary(`merged_into`==primary ·
  `merged_at` RFC 3339 · `asset` 없음), primary(그대로 `asset`), 무작위 UUID
  (`asset not found`), 두 단계 체인(한 단계만)을 확인합니다. 수정 전에는 두 테스트
  모두 `asset not found` 로 실패했습니다.
- **두 저장 엔진 모두에서 통과합니다.** `go test ./...`(SQLite fallback)와
  `scripts/test-postgres.sh`(postgres:17-alpine) 전 패키지가 통과합니다. PostgreSQL
  분기를 일부러 뒤집어 그쪽에서 빨강이 되는 것까지 확인해, 그 분기가 실제로
  실행된다는 것을 보았습니다.
- **Server.** `go vet` · `gofmt -l` 빈 출력 · `go build` 가 통과합니다. 콘솔
  정적 파일(`server/internal/webui/dist`)은 바뀌지 않았습니다.
- **Agent.** Rust 코드는 이번 릴리즈에서 바뀌지 않았습니다. 릴리즈 커밋에서
  `cargo fmt --all -- --check` · `cargo clippy --all-targets -D warnings` ·
  `cargo test --all-targets` 가 통과합니다.

## 호환성

- **데이터베이스 마이그레이션이 없습니다.** `assets` 에 열을 더하지 않고 이미
  기록되는 `asset_changes` 를 읽습니다.
- **REST API 응답 형식 변경이 없습니다.** 바뀐 것은 MCP `asset_get` 의 병합 자산
  응답뿐이고, 그 경우는 이전에 오류였으므로 성공 응답의 모양은 그대로입니다.
  scope 는 그대로 `assets.read` 입니다.
- **알려진 제한.** `secondary_ids` 는 병합 요청이 보낸 글자 그대로 저장되므로,
  콘솔이 아닌 호출자가 대문자 UUID 로 병합한 옛 자산은 조회가 찾지 못해 예전처럼
  `asset not found` 로 남습니다. 콘솔은 소문자를 보냅니다.
- **콘솔이 바뀌지 않았습니다.** 임베디드 정적 파일은 v0.2.37 과 같습니다.
- **Agent 의 동작 변경이 없습니다.** 설정 파일, 큐, 전송 프로토콜은 이전과 같고
  버전 문자열만 올라갑니다. 기존 Agent 는 서두르지 않아도 Server 와의 호환에
  영향이 없습니다.

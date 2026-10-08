# Invenqor Server·Agent v0.2.24 릴리즈 노트

릴리즈 일자: 2026-09-04
호환 Agent: v0.2.24 (Linux·Windows)

이번 릴리즈는 결함 하나를 고칩니다. 자리는 **API Key의 scope**이고, 증상은
v0.2.23까지와 같은 종류 — **오류 없이 받아들여진 뒤 나중에 틀린 답으로 드러나는
것** — 입니다. 콘솔에서 체크박스 하나를 지우면 만들 수 없어야 할 상태의 키가
남았고, 그 키는 죽은 것도 산 것도 아니었습니다.

## 1. API Key가 scope 하나 없는 상태로 남을 수 있었습니다

키를 **만들 때는** 언제나 scope가 최소 하나 필요했습니다. scope가 없는 키는
비활성화된 키가 아니기 때문입니다 — 그런 키도 인증에는 성공하고, `last_used_at`을
기록하고, rate limit을 통과한 뒤, **모든 엔드포인트에서 scope 검사에 걸려**
거절됩니다. 호출하는 쪽에서 보면 폐기된 자격 증명이 아니라 고장 난 API로 읽히고,
콘솔의 키 목록은 그 키를 아무 설명 없이 활성으로 보여줍니다.

그런데 **이미 있는 키를 바꾸는 두 경로**가 그 규칙을 건너뛰었습니다.

- `PATCH /api/v1/admin/api-keys/{id}`에 `{"scopes": []}`를 보내면 목록이 빈
  목록으로 교체되고 **HTTP 200**이 돌아왔습니다.
- `DELETE /api/v1/admin/api-keys/{id}/scopes/{scope}`로 **마지막 남은 scope**를
  지워도 마찬가지였습니다.

콘솔은 scope마다 체크박스를 하나씩 그리고 그 체크박스를 이 DELETE로 토글하므로,
마지막 상자를 지우는 클릭 한 번이면 생성 폼이 거부하는 상태에 도달했습니다.

이제 목록 전체를 다루는 호출자들이 검사 하나를 공유합니다. `RequireScopes`가
결과 목록을 검증하고 빈 목록을 거절하며, `Create`·`Update`·낙관적 scope 변경이
모두 이 함수를 지납니다. `ValidScopes`는 빈 목록을 계속 받아들입니다 —
`AddScopes`와 `RemoveScope`는 결과가 아니라 조각을 넘기기 때문입니다. 세 거절은
모두 **400 `INVALID_SCOPES`** 하나로 보고되므로, 스크립트가 규칙 하나에 대한 세
가지 답을 구분할 필요가 없습니다. 공개 계약(OpenAPI)에도 생성 시 `scopes`가
`minItems: 1`이고 scope 제거가 400을 낼 수 있다고 적혔습니다.

## 검증

- **저장 계층에서.** 이미 만들어진 키에 대해 목록 교체와 마지막 scope 제거가
  모두 `ErrScopesRequired`로 거절되고, **거절된 뒤 키의 scope가 그대로 남아
  있는지**를 고정했습니다. 증상이 "성공한 척하는 200"이었으므로 오류 코드만
  보지 않고 저장된 상태까지 확인합니다.
- **HTTP 경로까지.** 새 테스트가 생성 → 마지막 scope DELETE → 빈 목록 PATCH →
  scope 없는 생성까지 엔드포인트 너머에서 돌며, 네 자리 모두 400
  `INVALID_SCOPES`인지와 키가 원래 scope를 유지하는지를 확인합니다.
- **두 저장 모드에서.** `go test ./...`이 SQLite fallback과
  `scripts/test-postgres.sh`의 실제 PostgreSQL 양쪽에서 전 패키지를 통과하고,
  `go vet`·`go build`·`gofmt`, Agent 쪽 `cargo fmt`·`cargo clippy`·`cargo test`,
  콘솔 빌드와 OpenAPI 계약 검사가 통과합니다.

## 호환성

- 데이터베이스 마이그레이션이 없습니다.
- API 응답 형식 변경이 없습니다.
- **동작이 달라집니다.** 빈 `scopes` 목록으로 보내던 `PATCH`와 마지막 scope를
  지우던 `DELETE`는 이제 200 대신 400 `INVALID_SCOPES`를 돌려줍니다. 키를
  폐기하려면 scope를 비우지 말고 `DELETE /api/v1/admin/api-keys/{id}`로
  revoke하십시오.
- 이전 릴리즈에서 이미 scope가 비워진 키가 있다면 그대로 남아 있습니다. 그 키는
  어떤 요청도 통과시키지 못하므로, scope를 다시 부여하거나 revoke하십시오.

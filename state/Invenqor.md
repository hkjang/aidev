# Invenqor 자동 개선 기록

## 2026-09-02
- 선택: Query DSL의 시간 값을 DB에 넘기기 전에 시각으로 해석 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `querydsl`은 `"now - 24h"` 상대 시간만 `time.Time`으로 바꾸고 나머지 시간 값은
  문자열 그대로 SQL 파라미터로 넘겼다. PostgreSQL에서는 `last_seen_at < "2026-13-45"`가
  statement를 실패시켜 HTTP 500(실제로 재현 확인)이 되고, SQLite fallback에서는
  `"2026-01-31T09:00:00Z"`가 컬럼에 저장된 `"2026-01-31 09:00:00 +0000 UTC"`와 바이트
  비교되어 같은 날짜의 모든 행이 경계 반대편으로 넘어갔다. 이제 `now`, `now - <duration>`,
  드라이버가 만들어내는 절대 시각 레이아웃만 받아 `time.Time`으로 컴파일하고, 해석되지
  않는 값은 값 자체를 인용한 400 `INVALID_QUERY`로 거절한다. `/query/validate`도 파싱만
  하지 않고 컴파일까지 하도록 맞춰 실행 결과와 판정이 갈리지 않게 했다. 검증: 신규
  querydsl 단위 테스트와 `/api/v1/query/execute`·`/validate` 통합 테스트(엔드포인트에
  기존 테스트가 전무했음)를 추가하고 수정 전에 실패·수정 후 통과를 SQLite와 실제
  PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 확인, 전체 `go test ./...`(두 모드),
  `go vet`, `go build`, `gofmt` 통과.
- 보류 아이디어:
  - Query DSL에 OR/괄호 지원 추가 (가치 3 / 위험 4 / L — 문법·평가기 변경 범위가 큼)
  - `attributes.*` 경로의 숫자 비교가 텍스트 비교로 처리되는 문제 (가치 2 / 위험 3 / M)
  - 콘솔 쿼리 패널의 문법 참조 렌더링 테스트 보강 (가치 2 / 위험 1 / S)
  - `/api/v1/external/query/*` API key 경로의 한도·감사 로그 커버리지 (가치 3 / 위험 2 / M)

## 2026-09-02
- 선택: 요청 한도 카운터가 도착한 모든 주소를 영구히 기억하던 문제 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `agentRateLimiter.entries` 맵은 항목을 한 번도 지우지 않았다. 자격 증명이 필요 없는
  `/v1/agent/enroll`·`/v1/agent/preflight`의 한도는 출처 주소를 키로 쓰므로, 오래 떠 있는
  Server는 지금까지 접속한 모든 주소의 레코드를 그대로 들고 있었고 IPv6 /64를 가진 호스트
  하나가 운영자가 눈치채기 전에 메모리를 소진시킬 수 있었다. 창(window)이 지난 카운터는
  아무것도 결정하지 않으므로 `Allow`에서 만료 항목을 쓸어내되, 서로 다른 키가 몰릴 때
  매 요청마다 맵 전체를 훑지 않도록 창당 최대 1회만 수행한다. 테스트가 실제 창을 sleep으로
  기다리지 않도록 시계를 필드로 주입했다. 검증: 신규 `rate_limiter_test.go` 3개(만료 항목
  제거, 창당 1회 sweep, 한도 자체 규칙 — 한도 규칙에 대한 테스트도 기존에 없었음)를 추가해
  수정 전 실패·수정 후 통과를 확인하고, `go test ./...`, `go test -race`, `go vet`,
  `go build`, `gofmt` 통과.
- 보류 아이디어:
  - `/api/v1/external/query/*` API key 경로의 한도·감사 로그 커버리지 (가치 3 / 위험 2 / M)
  - API key 생성 시 잘못된 scope가 400 INVALID_SCOPES 대신 403 SCOPE_ESCALATION으로 반환됨 (가치 2 / 위험 1 / S)
  - 마지막 scope를 제거하면 scope가 하나도 없는 키가 남음(생성은 1개 이상을 요구) (가치 2 / 위험 2 / S)
  - `attributes.*` 경로의 숫자 비교가 텍스트 비교로 처리되는 문제 (가치 2 / 위험 3 / M)
- 릴리즈: v0.2.19 (2026-09-02)

## 2026-09-03
- 선택: 속성 조회가 찾으려는 키를 소문자로 바꿔버리던 문제 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `querydsl.Parse`가 컬럼을 대소문자 구분 없이 받으려고 필드명 전체를
  `strings.ToLower`로 접었다. 그런데 `attributes.` 뒤는 컬럼이 아니라 저장된 JSON
  문서의 키이고 JSON 키는 대소문자를 구분한다. 자산 API·MCP 도구·타 시스템 임포트로
  만든 자산은 호출자가 쓴 키를 그대로 갖고 있어 대문자가 흔한데,
  `attributes.assetTag`는 `'{assettag}'`(PostgreSQL `#>>`)와 `'$.assettag'`(SQLite
  `json_extract`)로 컴파일되어 어느 문서에도 없는 경로를 물었다. 결과는 오류 없는
  HTTP 200 빈 목록이었고 `/query/validate`는 그 표현식을 이미 valid라고 답한 뒤였다.
  이제 `attributes.` 접두사만 접고 키는 입력된 대소문자를 유지하며(컬럼명은 그대로
  대소문자 무시), 콘솔이 렌더링하는 문법 참조에도 대소문자 구분을 명시했다. 검증:
  querydsl 단위 테스트 2개와 `/api/v1/query/execute`·`/validate` 통합 테스트 2개를
  추가해 수정 전 실패·수정 후 통과를 확인하고, `go test ./...`를 SQLite fallback과
  실제 PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 통과, `go vet`, `go build`,
  `gofmt` 통과.
- 보류 아이디어:
  - `attributes.*` 경로의 숫자 비교가 텍스트 비교로 처리되는 문제 (가치 3 / 위험 3 / M)
  - 잘못된 scope로 키를 만들면 400 INVALID_SCOPES가 아니라 403 SCOPE_ESCALATION이 반환됨 (가치 2 / 위험 1 / S)
  - 마지막 scope를 제거하면 scope가 하나도 없는 키가 남음(생성은 1개 이상을 요구) (가치 2 / 위험 2 / S)
  - `/api/v1/external/query/*` API key 경로의 한도·감사 로그 커버리지 (가치 3 / 위험 2 / M)
- 릴리즈: v0.2.20 (2026-09-03)

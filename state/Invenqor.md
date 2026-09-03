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

## 2026-09-03
- 선택: 속성 경로의 크기 비교가 숫자를 문자열로 비교하던 문제 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `attributes.*` 경로는 저장된 JSON 문서에서 텍스트로 추출되므로 `<`, `<=`,
  `>`, `>=`가 문자 단위 비교였다. `'1' < '2'`라서 `"16000000000"`이
  `"2000000000"`보다 앞서고, 결과적으로 16 GB 호스트가
  `attributes.memory_bytes >= 2000000000`에서 빠졌으며
  `attributes.cpu_count > 9`는 10인 장비를 제외했다. 오류 없이 HTTP 200과 짧은
  목록만 돌아왔고 `/query/validate`는 그 표현식을 이미 valid라고 답한 뒤였다.
  이제 값이 숫자로 읽히는 크기 비교는 CASE로 컴파일되어 JSON 숫자로 저장된 값은
  숫자로(PostgreSQL은 `::double precision`, SQLite fallback은 `json_extract`가
  이미 돌려주는 수치 타입) 비교하고 그 외에는 지금처럼 텍스트로 비교한다. 덕분에
  `attributes.os_version > "20.04"`처럼 현재 텍스트로 정렬되는 절이 행을 잃지
  않는다. 등호·부등호(`=`, `!=`)는 그대로 두었다. 텍스트 비교가 이미 옳고, 거기서
  `"1.10"`을 숫자로 읽으면 버전을 담은 속성이 자기 자신과 일치하지 않게 된다.
  검증: querydsl 단위 테스트 2개(컴파일 결과·비대상 연산자)와
  `/api/v1/query/execute` 통합 테스트 2개(숫자 비교 교정·텍스트 정렬 무회귀)를
  추가해 수정 전 실패·수정 후 통과를 확인하고, `go test ./...`를 SQLite fallback과
  실제 PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 16개 패키지 모두 통과,
  `go vet`, `go build`, `gofmt` 통과.
- 보류 아이디어:
  - 잘못된 scope로 키를 만들면 400 INVALID_SCOPES가 아니라 403 SCOPE_ESCALATION이 반환됨(생성·추가 경로가 서로 다른 코드를 돌려줌) (가치 2 / 위험 1 / S)
  - 마지막 scope를 제거하거나 `PATCH {"scopes":[]}`를 보내면 scope가 하나도 없는 키가 남음(생성은 1개 이상을 요구) (가치 2 / 위험 2 / S)
  - `/api/v1/external/query/*` API key 경로의 한도·감사 로그 커버리지 (가치 3 / 위험 2 / M)
  - Query DSL에 OR/괄호 지원 추가 (가치 3 / 위험 4 / L)
- 릴리즈: v0.2.21 (2026-09-03)

## 2026-09-03
- 선택: `id` 절이 입력된 값을 그대로 DB에 넘기던 문제 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `assets.id`는 PostgreSQL에서 `UUID` 컬럼인데 `querydsl`은 `id` 절의 값을
  타이핑된 그대로 파라미터로 넘겼다. `id = "web-01"`은 statement 자체를 실패시켜
  HTTP 500(실제로 재현 확인)이 되고 값이 무엇이었는지는 응답 어디에도 남지
  않았으며, SQLite fallback에서는 텍스트 비교라 오류 없이 200 + 빈 목록이라
  "그런 자산이 없다"처럼 보였다. 두 모드는 표기만 다른 UUID에서도 갈렸다.
  PostgreSQL은 대문자·하이픈 없는 표기를 같은 값으로 읽고 fallback의 텍스트
  비교는 그렇지 않아서, UUID를 대문자로 출력하는 보고서에서 복사한 식별자가
  운영에서는 자산을 찾고 fallback에서는 아무것도 찾지 못했다. 이제 `id` 절은
  컴파일 시점에 `uuid.Parse`로 해석해 UUID가 아니면 값을 인용한 400
  `INVALID_QUERY`로 거절하고, UUID면 저장된 정규 표기로 접어서 두 모드가 같은
  답을 준다. 콘솔이 렌더링하는 문법 참조의 `id` 항목도 거절 대상이던 생략형
  예시(`"0d0f…"`) 대신 완전한 UUID 예시와 `uuid` 종류로 바꿨다. 검증: querydsl
  단위 테스트 2개와 `/api/v1/query/execute`·`/validate` 통합 테스트 2개를 추가해
  수정 전 실패(PostgreSQL 500, SQLite 빈 목록)·수정 후 통과를 확인하고,
  `go test ./...`를 SQLite fallback과 실제 PostgreSQL 양쪽에서 전 패키지 통과,
  `go vet`, `go build`, `gofmt` 통과.
- 보류 아이디어:
  - `/api/v1/external/query/*` API key 경로의 한도·감사 로그 커버리지 (가치 3 / 위험 2 / M)
  - 잘못된 scope로 키를 만들면 400 INVALID_SCOPES가 아니라 403 SCOPE_ESCALATION이 반환됨(생성·추가 경로만 검증 순서가 PATCH와 다름) (가치 2 / 위험 1 / S)
  - `attributes.*`에 대한 `!=`가 해당 키가 아예 없는 자산을 제외함(SQL NULL 의미) (가치 2 / 위험 3 / M)
  - `/api/v1/admin/api-keys` HTTP 계층에 테스트가 전무함(서비스 계층에만 존재) (가치 3 / 위험 1 / M)
- 릴리즈: v0.2.22 (2026-09-03)

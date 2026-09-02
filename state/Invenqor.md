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

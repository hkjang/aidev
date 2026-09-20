## 2026-09-20
- 선택: 실행 경로 SQL 정리 헬퍼 통합 — WrapLimit/Count/Explain이 trailing 주석·세미콜론으로 깨진 SQL을 만들지 않게 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `ValidateReadOnlySQL`은 주석을 마스킹한 뒤 검사해 `SELECT … FROM T -- note`·`…; -- note`를 통과시키지만 `WrapLimit`은 `TrimSuffix(";")`만 하고 같은 줄에 `)`를 붙여 괄호가 주석에 먹히거나 `;`가 남았고, `Count`·`ExplainPlan`도 같은 TrimSuffix를 복제하고 있었다. `internal/oracle.TrimStatement`(전방 스캔으로 문장 끝의 공백·`;`·line/block 주석만 제거, 리터럴·중간 주석 보존)를 추가해 세 호출부를 모두 이 헬퍼로 바꾸고 WrapLimit/Count는 본문을 별도 줄에 둔다. 검증: 먼저 표 테스트를 추가해 옛 WrapLimit에서 `(SELECT A FROM T -- note) WHERE ROWNUM` 형태로 실패하는 것을 확인(red) → 구현 후 `go build ./... && go vet ./... && go test ./...` 전부 통과(커밋 1507e00). 실제 Oracle 실행은 stub 드라이버라 Count/Explain은 헬퍼 단위 테스트까지만 증명.
- 보류 아이디어: ValidateReadOnlySQL 거부 키워드 정규식 사전 컴파일(2/1/S, 벤치마크 필요) · Manager.db()의 openDB를 변수화해 database/sql 테스트 드라이버로 Execute/Count/Explain 실제 전송 SQL 검증(3/2/M, 이번에 실행 e2e가 막힌 원인) · CI 워크플로 부재(3/1/S, 보호 경로라 운영자 확인 후) · gofmt 드리프트 2파일(2/1/S) · stripSQL 큰따옴표 식별자 오탐(2/3/S, 정책 경계라 보류)
- 과제서: 채택 — 과제서의 근거(세 경로의 TrimSuffix 복제, 검증/실행 불일치)가 코드와 정확히 일치했고 수용 기준 1~3을 그대로 구현; 단 제안된 "stripSQL 마스킹 길이로 자르기" 방식은 `SELECT 'x'`처럼 끝이 리터럴이면 리터럴이 잘려(마스크가 리터럴 공백과 주석 공백을 구분 못 함) 전용 스캐너로 대체함.


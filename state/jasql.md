## 2026-09-20
- 선택: 실행 경로 SQL 정리 헬퍼 통합 — WrapLimit/Count/Explain이 trailing 주석·세미콜론으로 깨진 SQL을 만들지 않게 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `ValidateReadOnlySQL`은 주석을 마스킹한 뒤 검사해 `SELECT … FROM T -- note`·`…; -- note`를 통과시키지만 `WrapLimit`은 `TrimSuffix(";")`만 하고 같은 줄에 `)`를 붙여 괄호가 주석에 먹히거나 `;`가 남았고, `Count`·`ExplainPlan`도 같은 TrimSuffix를 복제하고 있었다. `internal/oracle.TrimStatement`(전방 스캔으로 문장 끝의 공백·`;`·line/block 주석만 제거, 리터럴·중간 주석 보존)를 추가해 세 호출부를 모두 이 헬퍼로 바꾸고 WrapLimit/Count는 본문을 별도 줄에 둔다. 검증: 먼저 표 테스트를 추가해 옛 WrapLimit에서 `(SELECT A FROM T -- note) WHERE ROWNUM` 형태로 실패하는 것을 확인(red) → 구현 후 `go build ./... && go vet ./... && go test ./...` 전부 통과(커밋 1507e00). 실제 Oracle 실행은 stub 드라이버라 Count/Explain은 헬퍼 단위 테스트까지만 증명.
- 보류 아이디어: ValidateReadOnlySQL 거부 키워드 정규식 사전 컴파일(2/1/S, 벤치마크 필요) · Manager.db()의 openDB를 변수화해 database/sql 테스트 드라이버로 Execute/Count/Explain 실제 전송 SQL 검증(3/2/M, 이번에 실행 e2e가 막힌 원인) · CI 워크플로 부재(3/1/S, 보호 경로라 운영자 확인 후) · gofmt 드리프트 2파일(2/1/S) · stripSQL 큰따옴표 식별자 오탐(2/3/S, 정책 경계라 보류)
- 과제서: 채택 — 과제서의 근거(세 경로의 TrimSuffix 복제, 검증/실행 불일치)가 코드와 정확히 일치했고 수용 기준 1~3을 그대로 구현; 단 제안된 "stripSQL 마스킹 길이로 자르기" 방식은 `SELECT 'x'`처럼 끝이 리터럴이면 리터럴이 잘려(마스크가 리터럴 공백과 주석 공백을 구분 못 함) 전용 스캐너로 대체함.

## 2026-09-21
- 선택: 명시적 숫자 날짜의 달력 유효성을 공통 파서에서 검증 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: ParseTimeExpressions의 raw 숫자 날짜를 기존 ymd와 time.Parse로 검증하여 불가능한 날짜만 제외하고 유효한 윤일·월말·혼합 입력의 expression과 YYYYMMDD를 유지했다(3b9fcb1). 파서 표 테스트와 TempDir JSON을 catalog.Load로 읽은 실제 Server의 /mcp tools/call 테스트로 resolve_time/get_schema_context/build_sql_skeleton의 5개 날짜 타입·4개 표기에서 수정 전 실패를 확인하고 수정 후 통과했다. 지정 부분 테스트, go build ./..., go vet ./..., go test ./... 및 git diff --check 통과; 요청된 technology 부서 스킬/Skill 도구는 검색에서 찾지 못해 원문 절차·반환 형식 적용은 미확인이며 로컬 superpowers 디버깅·TDD 스킬을 대체 참고했다.
- 보류 아이디어:
  - 개발자 가이드 Go 버전·의존성·MCP 도구 수 갱신 (3/1/S): 차선 유지, 이번 범위 밖.
  - ValidateReadOnlySQL 정규식 사전 컴파일 (2/1/S): 효과 측정 벤치마크 필요.
  - CI 워크플로 추가 (3/1/S): 보호 경로로 보류.
  - cmd/* 플래그·env 파싱 테스트 (2/1/M): 별도 과제로 유지; 나머지 기존 항목도 ideas.json에 보존.
- 과제서: 채택 — 현재 코드의 검증 없는 raw 날짜 추가 및 세 도구의 공통 파서 배선이 정찰 근거와 일치했다.

## 2026-09-21
- 선택: jasql-goldgen의 음수 -keep/-n을 파일 접근 전에 거부 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: flag.Parse 직후 두 음수 인자를 stderr 메시지와 비정상 종료로 거부하도록 최소 가드를 추가하고 0 이상 제약을 문서화했다(fd6ba2a). 임시 fixture와 실제 CLI subprocess 28개 조합으로 수정 전 panic·덮어쓰기·입력 오류 우선순위 실패를 재현한 뒤, 수정 후 기본/명시적 출력의 바이트 보존·미생성 및 0/양수 생성·보존·clamp 통과를 확인했다. go test ./cmd/jasql-goldgen -count=1, go build ./..., go vet ./..., go test ./..., git diff --check 모두 통과했으며 기존 패키지 전체 테스트는 캐시 결과이고 요청한 technology 스킬 3개/Skill 도구는 제공 목록·로컬 검색에서 찾지 못해 원문 절차와 반환 형식 적용은 미확인이다.
- 보류 아이디어:
  - 개발자 가이드 Go 버전·의존성·MCP 도구 수 갱신 (3/1/S): 차선 유지.
  - ValidateReadOnlySQL 거부 키워드 정규식 사전 컴파일 (2/1/S): 효과 측정 필요.
  - CI 워크플로 추가 (3/1/S): 이번 범위 밖.
  - 나머지 cmd/* 플래그·env 테스트 (2/1/M): goldgen 외 공백 유지; 나머지 기존 항목과 정찰 신규 아이디어는 ideas.json에 보존.
- 과제서: 채택 — 현재 코드가 정찰의 음수 인자 문제와 일치하여 로드·선별·저장 알고리즘 변경 없이 조기 검증과 실제 CLI 테스트를 구현했다.


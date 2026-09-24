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

## 2026-09-22
- 선택: jasql-eval 상세 출력에서 컬럼·SQL 등 진단이 있는 케이스를 MISS로 표시 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 기존 세 bool 판정에 Missing 진단 유무를 추가하여 컬럼·SQL 오류도 MISS로 출력하고, 문서에 상세 상태와 요약 종료 기준의 차이를 명시했다(caf7a00). t.TempDir JSON 메타데이터·골든셋과 한 번 빌드한 실제 CLI의 7개 fixture × verbose/비verbose 테스트에서 수정 전 컬럼·SQL 케이스만 실패하고 수정 후 모두 통과했으며 정상·선택 검사 생략·테이블/지표/조인 실패·JSON 요약·기존 종료 코드를 검증했다. go test ./cmd/jasql-eval -count=1, go build ./..., go vet ./..., go test ./..., git diff --check 통과(기존 패키지 전체 테스트는 캐시); 실 Oracle exec:/rows: 재현과 요청된 technology 스킬 원문 적용은 미확인이다.
- 보류 아이디어:
  - 개발자 가이드 Go 버전·의존성·MCP 도구 수 갱신 (3/1/S): 차선 유지.
  - ValidateReadOnlySQL 거부 키워드 정규식 사전 컴파일 (2/1/S): 효과 벤치마크 필요.
  - CI 워크플로 추가 (3/1/S): 이번 보호 경로 범위 밖.
  - cmd/* 플래그·env 파싱 테스트 (2/1/M): eval 출력 테스트 외 공백 유지; 나머지 기존 항목은 ideas.json 보존.
- 과제서: 채택 — 현재 CLI 표시 조건과 공통 Missing 생산 경로가 정찰 근거와 일치하여 카탈로그·요약·종료 정책 변경 없이 구현했다.

## 2026-09-24
- 선택: 보고 단위(일별/월별/분기별/연도별) TimeRange를 질문 등장 순서로 내보내 aggregation_level 무작위화 제거 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `ParseTimeExpressions`의 보고 단위 루프가 map 리터럴 range라 TimeRange append 순서가 실행마다 달라졌고, `AnalyzeQuestion`이 이 슬라이스를 last-wins로 읽어 같은 질문의 `aggregation_level`이 day/month/year로 갈렸다. 고정 슬라이스 + `strings.Index` 위치 기준 `sort.SliceStable`로 등장 순서 append로 바꾸고(동률은 일별,월별,분기별,연도별,년도별 순), analyze.go에는 "마지막 언급 보고 단위" 계약 주석만 달고 선택 로직은 그대로 뒀다(c712da1). 검증: 신규 `internal/catalog/timeparse_order_test.go`(200회 바이트 동일성·위치 순서 표·AnalyzeQuestion 100회 안정성)와 `internal/mcp/analyze_order_test.go`(newFixtureServer + 실제 `/mcp` POST 25회, 보고 단위 1개 질문의 기존 출력 회귀)로 수정 전 red 확인 후 green; `go build ./... && go vet ./... && go test ./...`(catalog 57.3s, mcp 10.1s) 및 `git diff --check` 통과.
- 보류 아이디어: analyze_question이 복수 보고 단위 중 하나만 남겨 다단 GROUP BY 의도를 잃음(3/2/S, 골든 임계값 영향 미확인) · docs/development.md의 Go 버전·의존성·MCP 도구 수 갱신(3/1/S, 차선 후보) · CI 워크플로 부재(3/1/S, 보호 경로) · Manager.db()의 openDB 변수화로 실제 전송 SQL 검증(3/2/M) · internal/oracle의 gofmt 드리프트 2파일(profile.go, oracle_test.go — 이번 확인, 범위 밖이라 미수정)(2/1/S)
- 과제서: 채택 — 과제서의 근거(timeparse.go:114 map range → analyze.go:213 last-wins)가 현재 코드와 정확히 일치했고 수용 기준 1~3을 그대로 구현했다.

## 2026-09-25
- 선택: GitHub Actions CI 워크플로 추가 — build·vet·test 를 PR 마다 자동 실행 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 저장소에 `.github` 디렉터리 자체가 없어 러너가 올리는 PR 이 전부 "CI 검사 없음" 으로 막혀 있었다. `.github/workflows/ci.yml` 한 파일만 추가해 `push`(main)·`pull_request` 에서 `actions/checkout@v4` + `actions/setup-go@v5`(`go-version-file: go.mod`) 로 Go 를 깔고 `go build ./...` → `go vet ./...` → `go test ./...` 를 이 순서로 돌린다(ubuntu-latest 단일 잡, timeout-minutes 15, `permissions: contents: read`, 시크릿·외부 배포 없음). 검증은 문자열 검사가 아니라 실행으로 했다 — 워크플로를 쓰기 전에 현재 HEAD(e9f3fb2)에서 세 명령을 직접 돌려 green 을 확인했고(`go build ./...` exit 0, `go vet ./...` exit 0, `go test ./...` → catalog 54.365s / mcp 9.626s / meta·oracle cached / cmd 3 개 no test files, exit 0), 워크플로 자체는 `python3 -c "import yaml…"` 파싱과 `actionlint`(설치 후 실행, exit 0) 로 확인했다. `gofmt -l ./internal ./cmd` 는 `internal/oracle/oracle_test.go`, `internal/oracle/profile.go` 드리프트를 실제로 재현해(정찰이 미확인으로 남긴 항목) gofmt 게이트를 넣지 않은 판단이 옳았음을 확인했다 — 넣었으면 첫 실행부터 빨갛다.
- 보류 아이디어: docs/development.md 의 Go 버전·의존성·MCP 도구 수를 소스와 동기화 (3/1/S, 5 회차 연속 차선) · internal/oracle 의 gofmt 드리프트 2 파일 정리 (2/1/S, 이번에 실재 확인 — CI 가 들어왔으니 다음 회차 우선순위) · analyze_question 이 복수 보고 단위 중 하나만 남겨 다단 GROUP BY 의도를 잃음 (3/2/S) · Manager.db() 의 openDB 를 변수화해 실제 전송 SQL 검증 (3/2/M) · ValidateReadOnlySQL 거부 키워드 정규식 사전 컴파일 (2/1/S)
- 과제서: 채택 — 근거(`.github` 부재, origin 이 실제 GitHub 저장소, 5 회차의 '성공' 커밋이 pinned base 에 하나도 없음)가 현재 코드와 정확히 일치했고 수용 기준 1~3 을 그대로 구현했다.


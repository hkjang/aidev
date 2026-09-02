# relio 자율 개선 기록

## 2026-09-02
- 선택: 검색어의 `%`·`_` 를 와일드카드가 아닌 글자로 다루기 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 고객·영업기회·리드·제품·담당자·협업자 검색과 관리자 Audit Log 검색이 사용자가 입력한 문자열을 그대로 `LIKE '%'||lower($n)||'%'` 에 이어 붙여, Postgres 가 `%` 와 `_` 를 와일드카드로 읽었습니다. "50%" 로 검색하면 "50" 이 든 모든 고객이, `_` 는 임의의 한 글자가, `%` 하나는 테이블 전체가 나왔습니다. `crm.SearchPattern` 이 Go 쪽에서 백슬래시·`%`·`_` 를 이스케이프한 패턴을 만들고 7개 질의가 모두 `ESCAPE '\'` 를 선언하도록 바꿨습니다. 빈 질의는 빈 패턴을 그대로 돌려주므로 기존 `$n=''` 무필터 가드가 유지됩니다. 검증은 새 단위 테스트 3개(이스케이프 결과, 공백 질의, `internal/` 전체를 훑어 `ESCAPE` 없는 LIKE/ILIKE 를 잡는 회귀 가드)와 `go test -race ./...`, `go vet ./...`, `web` typecheck·build, `check-env-contract.sh`, `check-static-assets.sh` 전체 통과. 커밋 a49fd95.
- 보류 아이디어:
  - MCP 요청 본문이 1MB 를 넘으면 조용히 잘려 "Parse error" 가 되므로 413 으로 구분 (가치 3 / 위험 1 / S)
  - `internal/api`(OpenAPI 문서), `internal/audit`, `internal/job` 은 테스트가 하나도 없음 (가치 3 / 위험 1 / M)
  - `internal/server` 는 파일 23개에 테스트 4개뿐 — 순수 헬퍼(projection, 필터 파싱)부터 보강 (가치 3 / 위험 1 / M)
  - CI 에 `gofmt -l` 검사 추가 — 지금은 포맷 위반이 통과함 (가치 2 / 위험 1 / S)
- 릴리즈: v1.11.8 (2026-09-02)

## 2026-09-02
- 선택: OpenAPI 문서를 실제 라우터와 일치시키고 양방향 회귀 가드 추가 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `/api/openapi.json` 이 REST·MCP 클라이언트의 유일한 계약인데 라우터와 비교하는 장치가 없어 어긋나 있었습니다. 문서는 `PUT /opportunities/{id}/playbook` 을 약속했지만 실제 경로는 `PUT /opportunities/{id}/playbook/{itemId}` 라 405 가 났고, 로그인 흐름 전체(`/auth/status`, `/auth/login`, `/auth/logout`, `/auth/me`, OIDC start·callback)와 `/me/password`, `/dashboard` 등 8개 엔드포인트가 문서에 아예 없어 문서만 읽는 클라이언트는 인증 방법조차 알 수 없었습니다. 문서를 고치고, `internal/server/openapi_contract_test.go` 가 go/ast 로 패키지 소스에서 `mux.Handle*` 라우트 표를 읽어 문서와 양방향 비교하도록 했습니다(새 `/api/v1` 라우트는 문서화 전까지, 문서에만 있는 경로는 라우팅 전까지 빌드 실패). 검증은 문서 한 줄을 지워 테스트가 실제로 실패하는지 확인한 뒤 `go test -race ./...`, `go vet ./...`, `gofmt -l`, web typecheck·build, `check-env-contract.sh`, `check-static-assets.sh` 전체 통과. 커밋 4daf66f.
- 추가: CSP 보고에 directive 가 하나도 없으면 엔드포인트가 죽던 문제 (가치 4 / 위험 1 / 작업량 S)
  - `internal/server/analytics.go` 가 `strings.Fields(violated + " ")[0]` 로 첫 단어를 꺼냈는데, 뒤의 공백은 가드처럼 보이지만 아무 역할도 못 합니다 — `strings.Fields` 는 공백뿐인 문자열에 빈 슬라이스를 돌려주므로 `blocked-uri` 만 있고 `effective-directive`·`violated-directive` 가 둘 다 없는 보고(브라우저가 실제로 보낼 수 있는 형태)에서 index out of range 로 패닉했습니다. `/api/v1/csp-report` 는 인증이 없어 누구나 500 과 스택트레이스 로그를 반복 유발할 수 있었습니다. 테스트로 패닉을 먼저 재현한 뒤 `reportedDirective` 헬퍼로 분리해 고쳤고(빈 값은 `RecordViolation` 이 이미 버림), 단위 테스트 4케이스와 핸들러 테스트를 추가했습니다. 커밋 68529d3.
- 보류 아이디어:
  - `internal/intelligence/queries.go` 의 `GetSignal`/`GetRisk`/`GetRecommendation` 등은 ID 조건 없이 상위 200건만 받아 선형 탐색하므로, 레코드가 200건을 넘으면 조회뿐 아니라 `IgnoreSignal`·`AcceptRisk`·`AcceptRecommendation` 같은 쓰기까지 "not found" 로 실패 (가치 5 / 위험 2 / M) — 다음 세션 1순위
  - `internal/server/today.go:87` 의 `time.Now().Truncate(24*time.Hour)` 는 UTC 자정이라 KST 배포에서 하루 지난 연체 건이 HIGH 대신 WARNING 으로 분류됨 (가치 4 / 위험 2 / S)
  - `internal/intelligence/engine.go:333` 계약 만료 `D-N` 라벨이 정수 절삭 탓에 하루 짧고, D-90 창이 실제로는 D-91 까지 걸림 (가치 3 / 위험 2 / S)
  - `internal/approval/service.go:214` 의 `CONTAINS` 는 양쪽이 숫자처럼 보이면 숫자 분기 default 로 빠져 `==` 로 평가됨 → 승인 정책이 조용히 무시되고 결재가 우회됨 (가치 4 / 위험 2 / M)
  - MCP 요청 본문이 1MB 를 넘으면 조용히 잘려 "Parse error" 가 되므로 413 으로 구분 (가치 3 / 위험 1 / S)
  - CI 에 `gofmt -l` 검사 추가 — 지금은 포맷 위반이 통과함 (가치 2 / 위험 1 / S)
- 릴리즈: v1.11.9 (2026-09-02)

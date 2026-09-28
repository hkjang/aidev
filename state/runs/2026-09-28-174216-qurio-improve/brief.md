# 과제서 (2026-09-28, base d5eaf8d / VERSION 1.4.7 — 정찰이 실행으로 검증 완료)

- 과제: sqlsafe PostgreSQL 게이트가 `WHERE a.b = evil_fn(...)` 처럼 **점 표기 좌변 뒤의 함수 호출을 alias 컬럼 목록으로 오인해 통째로 면제**하는 fail-open 수정 (가치 5 / 위험 2 / 작업량 S)

- 왜: `internal/domain/sqlsafe/sqlsafe.go:770-771`(`postgresAliasColumnList` 의 마지막 `return`, "AS 없는 `schema.relation alias(column,…)`" 면제)가 **FROM 문맥을 요구하지 않는다.** `scanOraclePolicy` 는 연산자를 토큰으로 내보내지 않으므로 `WHERE public.events.name = dblink_exec(...)` 도 `word . word 이름(` 이라는 똑같은 토큰 형태가 되어 456행에서 허용목록 검사를 `continue` 로 건너뛴다. 실측에서 `dblink_exec('host=internal','DROP TABLE victim')` 이 `readOnly=true / risk=low` 로 **완전 통과**했고 `pg_read_file('/etc/passwd')` 는 blocked 가 아니라 medium(관리자 확인)으로 내려갔다. 440aa9f(`)` 분기에 FROM 문맥 요구)·89b3e76(`=(` 인접성 판정)가 같은 뿌리를 두 번 고쳤는데 세 번째 분기만 남아 있었다.

- 정찰이 실제로 실행해 본 결과 (base d5eaf8d, 임시 테스트 파일로 확인 후 삭제 — 작업 트리는 clean):
  - 수정 전 통과(= 결함): 
    - `SELECT id FROM public.events WHERE public.events.name = dblink_exec('host=internal','DROP TABLE victim')` → `readOnly=true risk=low reasons=[단일 읽기 전용 SQL 문입니다.]`
    - `SELECT id FROM public.events e WHERE e.name = pg_read_file('/etc/passwd')` → `risk=medium`(blocked 이어야 함)
    - `SELECT id FROM events WHERE events.name = custom_reporting_fn(1)` → `readOnly=true risk=low`
    - `SELECT id FROM events e JOIN other o ON e.id = custom_reporting_fn(o.id)` → `readOnly=true risk=low`
  - 대조군(원래도 정상 차단, 판정이 바뀌면 안 됨): `… WHERE name = custom_reporting_fn(1)`, `… AND dblink_exec('a','b') IS NOT NULL`
  - **제안한 한 줄 수정(아래 "건드릴 파일")을 임시로 적용해 본 결과: 위 4건 모두 blocked, 정상 질의 14건 전부 통과 유지, `go test ./internal/domain/sqlsafe -count=1` 통과, `go test ./... -count=1` 전 패키지 통과.** 확인 후 수정과 임시 테스트를 되돌렸다.

- 수용 기준:
  1) 위 4개 질의가 `ReadOnly=false`·`Risk="blocked"` 이고 이유에 해당 함수명이 담긴다. 대조군 2건의 판정은 그대로다.
  2) 정상 형태가 계속 통과한다 — 최소한 다음 14건: `FROM public.events ev(id, name)`, `FROM public.events AS ev(id, name)`, `FROM (SELECT 1) t(x)`, `FROM unnest(ARRAY[1,2]) g(v)`, `WITH cte(a, b) AS (SELECT 1, 2) SELECT a FROM cte`, `FROM public.orders a JOIN public.items b(id, sku) ON a.id = b.id`, `FROM public.docs d, public.tags t(id, label) WHERE d.id = t.id`, `FROM public.t, LATERAL unnest(t.arr) x(v)`, `SELECT lower(e.name) … WHERE e.id = 1`, `SELECT count(*) … GROUP BY e.name`, `… WHERE e.id IN (SELECT id FROM public.ids)`, `… WHERE s.ts > now() - interval '1 day'`, `… WHERE j.payload->>'k' = 'v'`, `… WHERE p.name LIKE upper(p.code)` (정찰이 이 14건으로 통과를 확인했다 — 테스트에 그대로 넣어도 된다).
  3) 기존 `internal/domain/sqlsafe` 테스트 전체 + `go test ./...` 가 통과한다. 가능하면 `internal/domain/dbexec`(뷰 정의·파티션 식·RLS 정책을 이 게이트로 검사)를 통합 태그로도 돌린다.
  4) red→green→인과 확인: 새 테스트가 수정 전 실패하고, 새로 붙인 문맥 검사만 `true` 로 고정하면 다시 실패한다.
  5) 게이트를 조이는 변경이므로 현실 질의 스윕으로 "내 변경으로 새로 막히는 정상 질의 0건" 을 보고한다(수정 전/후 판정을 같은 목록으로 비교).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개, 그 이상 벌리지 말 것):
  - `internal/domain/sqlsafe/sqlsafe.go:751-772 postgresAliasColumnList` — 마지막 `return nameIndex >= 3 && …` 분기에 `&& index.fromItemContext(nameIndex-1)` 를 붙여 760행의 `)` 분기와 같은 FROM 문맥 요구를 적용한다. 정찰이 확인한 사실: `fromItemContext`(869행)는 word 토큰에서 시작해도 동작하고(897행 switch 가 모든 kind 처리), 스트림 소진·미분류는 `false`(=fail-closed, 허용목록 검사로 복귀)다. `FROM public.events ev(…)` 는 `events → . → public → FROM` 으로 걸어가 true 가 되고, `WHERE public.events.name = fn(` 은 `name → . → events → . → public → WHERE`(clauseBoundary)에서 끊겨 false 가 된다. 주석도 `)` 분기(757-759행)와 같은 논조로 한 줄 갱신할 것.
  - `internal/domain/sqlsafe/sqlsafe_test.go` — 차단 케이스 4건과 수용 기준 2의 통과 케이스를 기존 표 형식에 추가.

- 검증 명령:
  - `go test ./internal/domain/sqlsafe -count=1 -v`
  - `go test ./... -count=1`
  - `go vet ./...` · `make check-go-format`
  - (DB 있을 때) 폐기 PostgreSQL 17 bootstrap 후 `go test -race -p=1 -tags=integration ./internal/domain/dbexec ./internal/legacyapi -count=1` — 포트 55432/55433 이 점유될 수 있으니 충돌하면 다른 포트로.

- 위험과 피할 것:
  - 허용목록(`postgresPureFunctions`·`postgresParenthesisKeywords`)·Oracle 경로·`postgresCallAdjacent`(723행)·`)` 분기(760행)·CTE 분기(`postgresCTEColumnList`, 939행)는 한 줄도 바꾸지 말 것. 이번 변경은 마지막 분기 한 곳이다.
  - 763행의 `AS` 분기(직전 토큰이 `AS` 면 무조건 면제)에도 같은 냄새가 있으나 **이번 범위 밖**이다 — `AS 이름(` 은 PostgreSQL 문법상 실행 불가라 악용 경로를 정찰이 확인하지 못했다(미확인). 같은 회차에 넓히지 말고 별건으로 남길 것.
  - `FROM unnest(a) WITH ORDINALITY g(v,i)`(AS 없는 형태)는 이 수정과 무관하게 **이전부터 오탐으로 차단**된다. 선재 결함이며 이번 범위 밖(같이 고치려 들면 범위가 터진다).
  - 미머지 PR a8f0775(runtimeapi·intelligenceapi·platformapi 통합 테스트)와 파일이 겹치지 않는다. 보호 경로(auth/migrations/workflows)도 건드리지 않는다.
  - 감사·로그에 SQL 원문을 새로 흘리지 말 것(이유 문자열은 기존 `boundedToken` 관례 그대로).

- 차선 후보: 픽스처 DELETE 를 등록하지 않는 통합 테스트 3개 파일(`runtimeapi/agent_instructions_integration_test.go`·`runtimeapi/publication_integration_test.go`·`intelligenceapi/repository_test.go`, 잔존 13행)에 정리 추가 — 단 앞 두 파일은 미머지 a8f0775 가 만지므로 충돌 주의.

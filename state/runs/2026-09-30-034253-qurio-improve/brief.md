- 과제: sqlsafe PostgreSQL 게이트가 `FROM events e(id, name)` 처럼 **AS 없는 비수식 테이블 별칭 + 컬럼 목록**을 비허용 함수 호출로 오판해 차단하는 오탐 수정 (가치 3 / 위험 3 / 작업량 S)

- 왜: `postgresAliasColumnList`(`internal/domain/sqlsafe/sqlsafe.go:751`)는 별칭 컬럼 목록을 네 형태로만 면제한다 — 닫는 괄호 뒤(760행), `WITH ORDINALITY` 뒤(766행), `AS` 뒤(771행), 점 표기 `schema.relation alias(`(781행). 표준 문법 `FROM table [AS] alias [(column,…)]` 중 **AS 생략 + 스키마 수식 없음** 조합만 어느 분기에도 안 걸려 별칭 `e` 를 함수 호출 `e()` 로 읽고 차단한다. 같은 질의에 `AS` 한 단어를 넣거나 `public.` 을 붙이면 통과하므로 게이트가 자기와 어긋나 있고, 평범한 읽기 전용 질의가 막힌다.
- 실측(이 정찰이 임시 테스트로 실제 실행해 확인, 확인 후 삭제 — 작업 트리 clean):
  - 차단(오탐): `SELECT * FROM events e(id, name)` → `ro=false risk=blocked` / 이유 `PostgreSQL SELECT의 비허용 함수 호출 e()이 포함되어 있습니다.`
  - 차단(오탐): `SELECT e.id FROM events e(id, name) WHERE e.id > 1`, `SELECT * FROM only events e(id)`, `SELECT * FROM events e(id) JOIN users u(uid) ON e.id = u.uid`(`e()`·`u()` 둘 다)
  - 통과(대조군): `FROM public.events e(id, name)`, `FROM events AS e(id, name)`, `FROM (SELECT 1) e(id)`, `FROM generate_series(1,3) g(v)`, `FROM events e` 는 전부 `ro=true risk=low`

- 수용 기준:
  1) `SELECT * FROM events e(id, name)`, `SELECT e.id FROM events e(id, name) WHERE e.id > 1`, `SELECT * FROM events e(id) JOIN users u(uid) ON e.id = u.uid`, `SELECT * FROM only events e(id)` 가 `ReadOnly=true`·`Risk != "blocked"` 로 통과한다.
  2) FROM 안의 **실제 함수 호출**은 계속 차단된다 — 새 분기가 `FROM`/`JOIN`/`LATERAL` 바로 뒤의 이름을 별칭으로 읽으면 fail-open 이다. 현재 차단되고 수정 후에도 차단이어야 하는 실측 케이스: `SELECT * FROM t CROSS JOIN dblink_exec('host=internal', 'DROP TABLE victim')`, `SELECT * FROM t, dblink_exec('host=internal', 'DROP TABLE victim')`, `SELECT * FROM t JOIN dblink('host=x', 'select 1') AS d(a text) ON true`, `SELECT * FROM events e(id) WHERE e.id = dblink_exec('host=internal','DROP TABLE victim')`(여기서 `e()` 는 통과하되 `dblink_exec()` 는 blocked).
  3) 새 red 테스트가 **수정 전 실패 → 수정 후 통과**하고, 새 판별(FROM 문맥 확인)을 `return true` 로 고정하면 2)의 차단 케이스가 다시 red 가 된다(인과 증명). 기존 sqlsafe 테스트(특히 `TestAnalyzeDialectRejectsPostgresStoredAndExtensionFunctions`, `…RejectsPostgresFunctionCallAfterClosingParenthesis`, `…RejectsPostgresCallAfterDottedOperand`, `…OrdinalityDoesNotExemptFunctionCalls`, `…RejectsPostgresCallsSeparatedOnlyByTrivia`)는 한 건도 깨지지 않는다.
  4) 차별 스윕: 현실 질의 목록(이전 회차들이 쓴 방식 — 50건 이상 × postgres·oracle 두 방언)의 판정을 base `3f7cf65` 파일과 수정본에서 각각 덤프해 diff 하고, **내 변경으로 새로 막히는 질의 0건**·Oracle 판정 불변을 보인다.

- 건드릴 파일 (프로덕션 1개, 테스트 1개):
  - `internal/domain/sqlsafe/sqlsafe.go` — `postgresAliasColumnList`(751-784행)의 마지막 `return`(781-783행) 앞 또는 그 `return` 확장으로 **비수식 별칭 분기** 추가. 설계 권고: `previous.kind == oraclePolicyWord && !previous.quoted` 이고 `previous.value` 가 `FROM`/`JOIN`/`LATERAL` 이 **아니며**(그 뒤의 이름은 별칭이 아니라 관계 = 테이블 함수 호출이다), `postgresValueExpressionKeywords`·`postgresClauseBoundaryKeywords` 에도 없을 때, `index.fromItemContext(nameIndex-2)` 가 true 면 별칭으로 본다. (`nameIndex-1` 이 아니라 `nameIndex-2` 에서 시작해야 `FROM t CROSS JOIN evil(1)` 이 `CROSS → t → FROM` 으로 걸어가 true 가 되는 fail-open 을 피할 수 있다 — 두 방어(키워드 제외 + `nameIndex-2` 시작)를 모두 넣을 것.) 이미 같은 파일에 있는 `fromItemContext`(881행)·`postgresTokenIndex` 를 그대로 쓰고 새 워커를 만들지 말 것(메모이제이션이 선형성을 보장한다).
  - `internal/domain/sqlsafe/sqlsafe_test.go` — `TestAnalyzeDialectAllowsPostgresAliasColumnLists`(224행)에 1)의 정상 케이스를, `TestAnalyzeDialectRejectsPostgresStoredAndExtensionFunctions`(132행) 또는 새 `Test…UnqualifiedAliasDoesNotExemptFunctionCalls` 에 2)의 차단 케이스를 추가.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/domain/sqlsafe -count=1 -v`
  - `go test ./...` (통합 태그 없이 28 패키지)
  - `go vet ./...` / `make check-go-format` / `go build ./...`
  - 통합(선택, 이 게이트를 뷰 정의·파티션 식·RLS 정책에 쓰는 `internal/domain/dbexec` 확인용): 폐기 PostgreSQL 17 을 도커로 띄워 bootstrap 후 `go test -race -p=1 -tags=integration ./internal/domain/dbexec ./internal/legacyapi -count=1`. 포트는 55432/55433/55439 가 점유된 적이 많으니 비어 있는 포트를 고르고 끝나면 컨테이너를 지울 것.

- 위험과 피할 것:
  - 이 과제는 **게이트를 넓히는 방향**이다. 허용목록(`postgresPureFunctions`)·Oracle 경로·기존 네 분기(760/766/771/781행)·`postgresCallAdjacent`(723행)는 한 줄도 건드리지 말 것. 분기 하나만 추가한다.
  - `fromItemContext` 를 요구하지 않는 면제는 이 파일에서 이미 네 번(440aa9f, 89b3e76, e1af783, 7cda7f5) fail-open 으로 드러났다. 새 분기도 반드시 FROM 문맥을 요구하고, 판단이 안 되면 fail-closed(허용목록 검사로 복귀)여야 한다.
  - 보호 경로(auth/session/migrations/.github/workflows)는 건드리지 않는다. 통합 테스트 픽스처(`defer pool.Close()`)도 이번 과제와 섞지 말 것.
  - **미확인으로 남기는 것**: `AS` 분기(771행)는 FROM 문맥 없이 무조건 면제해 `SELECT * FROM events AS dblink_exec('host=internal','DROP TABLE victim')` 이 `ro=true risk=low` 로 통과한다(실측). 다만 PostgreSQL 은 별칭 컬럼 목록에 문자열 리터럴을 허용하지 않아 실제 실행 가능한 악용 질의를 찾지 못했다 — **이번 회차 범위 밖**이고, 별건으로 남긴다. 같이 고치려 들면 범위가 커지고 증명이 안 된다.

- 차선 후보: `internal/legacyapi` 통합 테스트 10개 파일(`git_settings`·`embedding_materialization`·`advanced`·`durable_tasks`·`job_authorization`·`offline_parity`·`analytics_detail`·`code_remote_skills`×2·`integration_test`)의 `defer pool.Close()` → `t.Cleanup(pool.Close)` 순서 역전 수정. 단, 파일 수가 많으니 한 회차에 절반만 담고, 이미 같은 수정을 한 `runtimeapi`·`intelligenceapi`·`platformapi`(미머지 a8f0775)는 건드리지 말 것.

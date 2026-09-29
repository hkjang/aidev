# PR 처리기 노트 2026-09-30-042819-qurio-shepherd — qurio PR #29
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-034253-qurio-improve)
# 회차 노트 2026-09-30-034253-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:42] base pinned — main@3f7cf65
- [러너 03:42] autonomy release — 

## 정찰 노트
- sqlsafe 별칭 면제 계열을 다시 골랐다: 임시 테스트를 실제로 돌려 `FROM events e(id, name)` 가 `e()` 로 blocked 되고 `AS`·`public.` 을 붙인 동형 질의는 통과하는 것을 실측했다 — 추측이 아니다. legacyapi `pool.Close` 확장은 10파일·실 DB 재현이 필요해 한 세션·파일 수 기준에 맞지 않아 차선으로 뺐다.
- 확신 없는 곳: 구현 설계(`fromItemContext(nameIndex-2)` + FROM/JOIN/LATERAL 제외)는 토큰 walk 를 손으로 따라간 것이고 실행으로 확인하지 않았다. 구현자는 이 설계를 그대로 믿지 말고 red→green→헬퍼를 `true` 로 고정해 다시 red 로 인과를 확인할 것.
- 조심할 것: 이 과제는 게이트를 **넓히는** 방향이다. `FROM t CROSS JOIN dblink_exec(...)` / `FROM t, dblink_exec(...)` 가 계속 blocked 인지를 수용 기준 2 로 반드시 고정할 것. `AS` 분기(771행)의 무조건 면제도 실측으로 확인했으나 실행 가능한 악용 질의를 못 찾아 범위 밖으로 남겼다 — 같이 고치지 말 것.
- 작업 트리는 clean 상태로 남겼다(임시 테스트 파일 생성 후 삭제, `git status --porcelain` 무출력 확인).
- [러너 03:47] scout done — sqlsafe PostgreSQL 게이트가 `FROM events e(id, name)` 처럼 **AS 없는 비수식 테이블 별칭 + 컬럼 목록**을 비허용 함�

## 구현 노트
- `postgresAliasColumnList` 에 비수식 별칭 분기 하나(+헬퍼 하나, 프로덕션 1파일 +53줄)를 더해 `FROM events e(id, name)` 오탐을 없앴다. 면제 조건은 두 겹이다 — ① 별칭 왼쪽 단어가 관계 도입어(FROM/JOIN/LATERAL/ONLY/WITH/RECURSIVE/TABLE)·절 키워드가 아니고 ② 그 관계 이름이 FROM/JOIN/ONLY 직후이거나 `fromItemContext` 가 FROM 항목 구분자로 증명한 콤마 직후일 때만. 둘 중 하나를 `return true` 로 고정하면 차단 케이스가 다시 red 가 되는 것을 실행으로 확인했다.
- 과제서의 설계 권고(`fromItemContext(nameIndex-2)` 단독)는 그대로 쓰지 않았다 — 기존 테스트 `FROM unnest(ARRAY[1]) "WITH" ORDINALITY custom_reporting_fn(1)` 이 통과해 버리는 fail-open 이었다(실행으로 확인). 그래서 콤마 외에는 FROM/JOIN/ONLY 직후만 인정하도록 좁혔다.
- 확신 없는 곳: `WITH`/`RECURSIVE`/`TABLE` 을 관계 도입어에 넣은 것은 문법 추론이고 실행 가능한 악용 질의로 확인하지 않았다(해당 형태는 PostgreSQL 이 구문 오류로 거부한다고 판단). 또 이 분기는 관계 이름이 `postgresValueExpressionKeywords`/`postgresClauseBoundaryKeywords` 에 있으면 fail-closed 이므로, `FROM set s(x)` 처럼 **예약어를 인용 없이 테이블명으로 쓴** 질의는 여전히 차단된다(의도한 fail-closed, 실측하지 않음).
- 범위 관찰(비평가가 먼저 볼 곳): `internal/domain/dbexec` 는 자체 정책(`ErrUnqualifiedRelation`, search_path=pg_catalog)으로 비수식 *테이블* 을 금지하므로, dbexec 실행 경로의 비수식 테이블 별칭은 이 수정 후에도 다른 이유로 거부된다. 통합 테스트는 dbexec 이 비수식으로 허용하는 유일한 FROM 항목인 CTE 참조로 red→green 을 증명했다. 이득은 sqlsafe 를 직접 호출하는 게이트(mcpserver·runtimeapi/sql.go·legacyapi topology/dataset_autofill·platformapi evaluation_worker)에서 난다.
- 일부러 하지 않은 것: `AS` 분기(771행)의 무조건 면제는 범위 밖(악용 질의 미확인). 허용목록·Oracle 경로·기존 네 분기·`postgresCallAdjacent` 는 한 줄도 건드리지 않았다. legacyapi `pool.Close` 는 이번에 섞지 않았다.
- 다음 역할이 조심할 것: 새 `dbexec` 테스트(`TestPostgresUnqualifiedAliasColumnListValidationAndPaging`)는 `-tags=integration` + `QURIO_TEST_POSTGRES_DSN` 이 필요하고 마이그레이션이 적용된 폐기 DB 를 요구한다(`users` 에 시스템 사용자 1행이 있어야 한다). 이번 검증은 도커 PostgreSQL 17 을 127.0.0.1:55482 에 띄워 했고 컨테이너는 제거했다.
- [러너 04:01] brief accepted — 채택 — 지목한 행(751-784의 네 분기, 881 `fromItemContext`)이 코드와 정확히 일치했고 수용 기준 1~4 를 그대로 충족했다. 다만 
- [러너 04:04] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: diff 3파일 정독 + sqlsafe.go:751-835·fromItemContext·호출측(450-475) 흐름 추적 + base(main)/HEAD 양쪽에서 프로브 질의 11건 실제 실행 비교.
- **거절(security 차단)**: sqlsafe.go:806-816 의 introducer FROM/JOIN/ONLY 무조건 `return true` 가 fail-open 이다. PostgreSQL 은 `extract/substring/trim/overlay` 인자 문법에도 `FROM` 을 쓰고, 스캐너가 연산자를 버리므로 `FROM src + evil(...)` 가 `FROM <word> <word>(` 로 붕괴한다. 실측: `SELECT substring(label FROM id + query_to_xml('DELETE FROM users',false,true,'')::text::int) FROM events` → main blocked, HEAD `readonly=true risk=low`, RequireReadOnlyDialect nil. `query_to_xml` 은 pg_catalog 내장이라 dbexec 의 search_path 제한도 못 막는다.
- 수리가 먼저 볼 파일: `internal/domain/sqlsafe/sqlsafe.go:801-816`(FROM/JOIN/ONLY 특례 제거 → fromItemContext 통일, 단 구현 노트의 `"WITH" ORDINALITY` fail-open 을 별도로 닫을 것) 과 `internal/domain/sqlsafe/sqlsafe_test.go:573`(extract/substring/trim/overlay 인자 문법 차단 케이스 추가).
- 못 본 것: dbexec 통합 테스트 미실행(단위 수준에서 확정), Oracle 판정 불변 미검증 — 수리 후 프로필이 요구하는 base-vs-수정본 현실 질의 덤프 diff 를 반드시 수행할 것. 작업 트리는 clean 으로 복원했다.
- [러너 04:08] review rejected — 리뷰 거절: internal/domain/sqlsafe/sqlsafe.go:806-816 새 비수식 별칭 분기가 introducer 가 FROM/JOIN/ONLY 일 때 `fromItemContext` 를 호출하지 않고 무조건 `return true` 한다.
- [러너 04:08] review blocked — 검토 부서 차단 소견(security) — 수리·중재 없이 운영자의 위험 수용(risk-accepted) 필요
- [러너 04:08] pr created — https://github.com/hkjang/qurio/pull/29

## 심사 노트
- 확인: 비평가의 fail-open 을 손으로 믿지 않고 base(origin/main) 와 HEAD 양쪽에서 임시 테스트로 동일 질의를 실제 실행해 대조했다 — `substring(label FROM id + query_to_xml('DELETE FROM users',...)::text::int)` 외 2건이 main blocked → HEAD `readonly=true risk=low`, RequireReadOnlyDialect nil. 원인은 sqlsafe.go:806-816 의 FROM/JOIN/ONLY 무조건 `return true`(스캐너가 연산자를 버려 `FROM <word> <word>(` 로 붕괴). 프로덕션 배선도 확인(mcpserver/server.go:174, runtimeapi/services.go:753).
- 확인: 과제 목적인 `FROM events e(id, name)` 오탐 제거는 실측으로 성립(base blocked → HEAD 통과), `go test ./internal/domain/sqlsafe/` 통과·`go vet` 무출력, 보호 파일·마이그레이션·공개 경로·비밀값 변경 없음, 범위 이탈 없음.
- 못 본 것: 새 dbexec 통합 테스트는 `-tags=integration`+`QURIO_TEST_POSTGRES_DSN`+마이그레이션 적용 DB 가 필요해 실행하지 못했다. Oracle 판정 불변은 코드 정독(변경이 postgres* 함수 한정)으로만 확인했고 프로필이 요구하는 질의 덤프 diff 는 하지 않았다. 작업 트리는 임시 테스트 삭제 후 clean 으로 복원.
- 권고: reject / human, risk=high. 읽기 전용으로 광고된 경로에서 임의 쓰기가 가능해지는 신규 회귀라 security 차단 소견이며 수리·중재로 풀 수 없다.
- 다음 수리가 할 일: FROM/JOIN/ONLY 특례를 없애고 comma 분기와 같이 `fromItemContext` 로 통일하되 구현 노트가 적은 `"WITH" ORDINALITY` fail-open 을 함께 닫을 것. sqlsafe_test.go:573 차단 테이블에 substring/extract 인자 FROM 문법 3건을 추가하고, 프로필이 요구하는 base-vs-수정본 현실 질의 덤프 diff 를 첨부할 것.

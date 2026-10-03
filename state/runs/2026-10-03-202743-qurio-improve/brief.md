# 과제서 — 2026-10-03 (base main@814482a, VERSION 1.4.10)

- 과제: sqlsafe Oracle 게이트가 `WITH name (col, …) AS (…)` 서브쿼리 팩토링의 컬럼 별칭 목록을 함수 호출로 오판해 차단하는 오탐 수정 (가치 5 / 위험 2 / 작업량 S)

- 왜: Oracle 표준 문법인 서브쿼리 팩토링 컬럼 별칭(`WITH recent (id, total) AS (SELECT …) SELECT * FROM recent`)이 `Oracle SELECT의 비허용 함수 호출 recent()이 포함되어 있습니다.` 라는 허위 사유로 blocked 된다 — 이 세션에서 `AnalyzeDialect` 로 실측했다(아래 "실측 기록"). PostgreSQL 게이트는 같은 질의를 `postgresCTEColumnList`(sqlsafe.go:952)로 면제해 ro=true/low 로 통과시키므로, **같은 스캐너(`scanOraclePolicy`)가 만든 같은 토큰 스트림을 읽는 두 경로 중 Oracle 쪽만 면제가 없는 비대칭**이다. 4b29d08(`postgresCallAdjacent` → `callAdjacent`)이 고친 것과 같은 뿌리이며, 고치면 Oracle 사용자가 CTE 를 쓰는 가장 흔한 분석 질의가 드라이버에 닿는다.

## 실측 기록 (이 세션, 임시 테스트로 확인 후 되돌림)

base 814482a 의 `AnalyzeDialect(q, "oracle"|"postgres")` 판정:

| 질의 | oracle | postgres |
|---|---|---|
| `WITH recent (id, total) AS (SELECT id, SUM(amount) FROM reporting.sales GROUP BY id) SELECT * FROM recent` | **blocked** `recent()` | ro=true low |
| `WITH a (x) AS (SELECT 1 FROM dual), b (y) AS (SELECT 2 FROM dual) SELECT * FROM a, b` | **blocked** `a()`·`b()` | ro=true low |
| `WITH recent AS (SELECT id FROM reporting.sales) SELECT * FROM recent` (컬럼 목록 없음) | ro=true low | ro=true low |

즉 Oracle 에서 **컬럼 별칭 목록을 붙이는 순간에만** 차단된다 — 게이트가 자기와 어긋난 상태다.

## 설계 (이 세션에서 실제로 패치해 돌려 본 것 — 그대로 통했다)

`internal/domain/sqlsafe/sqlsafe.go` 의 `oracleReadOnlyViolations`(1004행) 루프는 `postgresReadOnlyViolations`(418행) 루프와 모양이 같고, 차이는 Postgres 쪽에만 456행 `postgresAliasColumnList` 가 있다는 점뿐이다. 두 가지만 하면 된다:

1. 루프 앞에 `tokenIndex := postgresTokenIndex{tokens: tokens}` 를 선언한다(Postgres 쪽 449행과 동일 — 지연 `build()` 라 후보 없는 질의는 비용 0).
2. 허용목록 검사 **앞**(1021행 `callAdjacent` 호출 바로 위)에 CTE 면제를 넣는다:
   ```go
   if postgresCTEColumnList(&tokenIndex, index-1) {
       continue
   }
   ```

`postgresCTEColumnList` 는 `WITH`/`WITH RECURSIVE` 직후이거나 같은 depth 의 top-level 콤마 뒤에서 `WITH` 까지 거슬러 올라가 확인되는 이름만, 그리고 **닫는 괄호 다음 토큰이 `AS` 일 때만** 면제한다(963행) — 이미 fail-closed 로 좁혀져 있어 새 판별 로직을 쓸 필요가 없다.

권고: 선례(4b29d08 의 `postgresCallAdjacent` → `callAdjacent`)대로 **함수 본문은 한 줄도 바꾸지 않고** 이름만 `postgresCTEColumnList` → `cteColumnList` 로 바꿔 두 게이트가 같은 규칙을 공유하는 것을 드러내라(`postgresAliasColumnList:775` 의 호출부도 같이 갱신). 이름을 그대로 두고 호출만 추가해도 동작은 같으니, 이름 변경이 리뷰에서 걸리면 포기해도 된다.

**`postgresAliasColumnList` 전체를 Oracle 에 붙이지 말 것.** ① Oracle 은 `FROM t alias(col)` 문법이 없어 필요하지 않고 ② 그 안의 `AS` 분기(772-774행)는 FROM 문맥 없이 무조건 면제하므로 Oracle 로 옮기면 fail-open 이 된다. 범위는 CTE 분기 하나로 한정한다.

## 수용 기준

1. Oracle 에서 `WITH recent (id, total) AS (SELECT id, SUM(amount) FROM reporting.sales GROUP BY id) SELECT * FROM recent` 가 `ReadOnly=true`, `Risk="low"` 로 통과한다.
2. 다중 CTE `WITH a (x) AS (…), b (y) AS (…) SELECT * FROM a, b` 와 `WITH RECURSIVE` 형태도 통과한다.
3. 차단 유지: `SELECT side_effect_fn(1) FROM dual`, `WITH a AS (SELECT side_effect_fn(1) FROM dual) SELECT * FROM a`(CTE 안의 실제 호출), `SELECT a (x) FROM dual` 처럼 `)` 뒤에 `AS` 가 없는 형태, 인용 식별자 `"A" (x) AS (…)`. 새 red 테스트 → 수정 → green → `postgresCTEColumnList` 호출만 `return`/`true` 로 고정해 다시 red → 복원으로 인과를 보여라.
4. **PostgreSQL 판정 불변**: 변경이 `oracleReadOnlyViolations` 안에만 있으므로 구조적으로 보장되지만, 이 저장소의 확립된 증명 방식대로 현실 질의 60건+ × 2방언 판정을 base `814482a` 의 sqlsafe.go 와 수정본에서 각각 덤프해 기계적으로 diff 하고 **PostgreSQL 행 전부 바이트 단위 동일 / 내 변경으로 새로 막히는 질의 0건** 을 보여라.
5. 프로덕션 배선으로도 red→green 을 증명하라: `dbexec.Connection{Dialect: Oracle, AllowedSchemas: []string{"REPORTING"}}` 로 `internal/domain/dbexec` 의 `analyzeExecutionPolicy` 를 호출하는 2026-10-02 회차(`policy_test.go`)와 같은 방식. 손으로 만든 대역 대신 실제 타입을 쓸 것.

## 건드릴 파일 (프로덕션 1개)

- `internal/domain/sqlsafe/sqlsafe.go`
  - `oracleReadOnlyViolations`(1004-1046) — `postgresTokenIndex` 선언 + 1021행 앞에 CTE 면제 `continue` 추가 (+4줄)
  - (선택) `postgresCTEColumnList`(952) 이름을 `cteColumnList` 로, 호출부 775행 갱신
- `internal/domain/sqlsafe/sqlsafe_test.go` — Oracle 허용/차단 테이블 테스트 추가
- `internal/domain/dbexec/policy_test.go` — 수용 기준 5 의 프로덕션 배선 테스트

## 검증 명령 (이 저장소에서 실제로 도는 것)

```
go test ./internal/domain/sqlsafe -count=1        # 이 세션에서 패치 후 ok 확인
go test ./... -count=1                            # 28 패키지
go vet ./...
go vet -tags=integration ./...
make check-go-format
go build ./...
```
통합(선택, 느림): 빈 포트의 폐기 PostgreSQL 17 컨테이너를 bootstrap 한 뒤
`go test -race -p=1 -tags=integration ./... -count=1` 2회 연속. **포트 55432/55433/55439/55462/55472/55482/55492/55502 는 과거 점유 이력이 있으니 다른 번호를 고르고 컨테이너는 반드시 제거할 것.**

## 위험과 피할 것

- 허용목록 `oraclePureFunctions`·`oracleParenthesisKeywords`·`postgresPureFunctions` 은 한 줄도 바꾸지 말 것. 이 수정은 면제 경로만 건드린다.
- `callAdjacent`(724) 본문, `postgresAliasColumnList`(752) 의 기존 네 분기, `fromItemContext`(882), `scanOraclePolicy` 는 건드리지 말 것.
- `postgresAliasColumnList` 를 Oracle 에 붙이지 말 것(위 설계 참조 — fail-open).
- **미머지 PR 충돌**: `433a183`(비수식 별칭 분기, `postgresAliasColumnList` 수정)과 `12223b0`(`ci.yml`/`release.yml` `-count=1` + `scripts/ci_workflow_test.go`)가 main 에 아직 없다. 433a183 은 `postgresAliasColumnList` 를 건드리므로 **이름 변경(`cteColumnList`)을 택하면 775행에서 텍스트 충돌이 날 수 있다** — 충돌이 부담되면 이름을 그대로 두고 Oracle 호출부만 추가하라. `oracleReadOnlyViolations` 자체는 두 미머지 PR 과 겹치지 않는다.
- 보호 경로(`internal/httpapi`·`oidcauth`·`mcpoauth`·`agentapi`·`migrations`·`.github/workflows`)는 이번 과제에서 전혀 건드리지 않는다.
- **실 Oracle 데이터베이스는 이 환경에 없다.** 증명은 게이트 판정 수준(sqlsafe + dbexec 정책)까지이고, 수정이 질의를 *통과*시키는 방향이므로 Oracle 서버가 최종 문법 심판이라는 점을 보고에 정직하게 적어라(2026-10-02 회차와 동일한 한계).

## 차선 후보

`internal/legacyapi` 통합 테스트의 `defer pool.Close()` → `t.Cleanup(pool.Close)` 순서 역전 수정 — main 에 53곳/31파일 잔존(이 세션 실측). legacyapi 10곳(`git_settings`·`durable_tasks`·`advanced`·`embedding_materialization`·`job_authorization`·`offline_parity`·`analytics_detail`·`code_remote_skills`×2·`integration_test`). **한 회차에 5파일만** 담을 것. 미머지 `a8f0775` 가 runtimeapi·intelligenceapi·platformapi 19곳을 다루므로 그 세 패키지는 피할 것.

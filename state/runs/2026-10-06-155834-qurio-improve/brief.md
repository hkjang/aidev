# 과제서 — 2026-10-06-155834-qurio-improve (qurio)

base: `afd3882` (chore: release Qurio v1.4.13), VERSION `1.4.13`

- **과제**: sqlsafe 위반 사유 누적(`appendUnique`)의 quadratic 제거 — 양 방언 + 병합 루프 (가치 4 / 위험 2 / 작업량 S) — **재시도 과제**

- **왜**: `internal/domain/sqlsafe/sqlsafe.go:192` `appendUnique` 는 append 마다 이미 모은 사유 전부와 문자열 비교를 한다. 사유 문자열이 **질의에서 뽑은 식별자**(함수 이름·cast 형식·문자열 상수 형식)를 품는 자리가 네 곳(pg 430/436/444/473/479/499 중 444·473·499, oracle 1096)이라 사유 카디널리티가 질의 길이로만 묶이고, 그 n 개를 `AnalyzeDialect` 병합 루프(131-133)가 `result.Reasons` 로 다시 `appendUnique` 로 옮겨 **같은 quadratic 을 한 번 더** 치른다. 진입점 `internal/runtimeapi/sql.go:41` `(*API).validateSQL` 은 인증된 principal 만 있으면 DB 권한·승인 없이 최대 `1<<20` 바이트를 그대로 넘기므로, 요청 한 건이 코어 하나를 10초 넘게 점유한다(직전 회차 실측: 1,048,570B/서로 다른 이름 81,512개 = **pg 10.9s / oracle 11.0s**; 4000/8000/16000/32000/64000개 = pg 20/92/241ms/1.07s/5.33s, oracle 20/97/236ms/1.07s/5.41s).

- **이 과제가 "재시도" 인 이유 (반드시 먼저 읽을 것)**: 2026-10-06 **이전 회차가 이 과제를 이미 설계·구현·증명했고 회차 결과는 `verify-failed`** 였다. 그 커밋은 main 에 **없다** — 이번 base 에서 `grep -n reasonSet internal/domain/sqlsafe/sqlsafe.go` 가 0건이고, `git diff bd3f310 afd3882 -- internal/domain/sqlsafe/` 가 **무출력**(sqlsafe 가 바이트 단위로 v1.4.12 와 같다)임을 이번 정찰이 직접 확인했다. 즉 위 실측치는 **이 base 에 그대로 유효**하다.
  - verify 가 왜 깨졌는지는 **미확인**(CI 로그를 못 봤다). 다만 **같은 날 다음 회차**(커밋 `ab53169`, main 에 머지됨)가 `cd web && npm test` 게이트를 base 5회 중 2회 깨뜨리던 테스트 레이스 2건(`web/src/pages/LegacyParityOperations.test.tsx:70`, `web/src/components/AdminAskOriginalLauncher.test.tsx:110`)을 고쳤다. 그 flake 가 당시 verify 를 깨뜨렸을 가능성이 가장 높고, **지금은 고쳐져 있다**. 사람이 반려한 PR 이 아니라 자동 게이트 실패이므로 재제출 금지 대상이 아니다.
  - **구현자가 할 일**: 설계를 처음부터 다시 고민하지 말고 아래 "설계" 를 그대로 쓰되, **증명은 다시 수행**하라(이전 회차 산출물은 트리에 없다).

- **수용 기준**:
  1. 서로 다른 사유를 n 개 만드는 질의에서 `AnalyzeDialect` 가 **선형**이 된다. 새 스케일 테스트 3개(pg 함수 이름 / oracle 함수 이름 / pg cast 형식, 각 **서로 다른 이름 64000개**, 테스트 예산 1초)가 **base 에서 red**(각 5.3~5.8s)이고 수정 후 **green**(0.1s 수준)이다. ⚠ 16000개(약 240ms)는 절대 예산을 통과해버려 red 가 되지 않는다 — **64000개로 재라**.
  2. **판정 불변**: 현실 질의 90건 이상 × 방언 표기 4종(`postgres`/`oracle`/`mysql`/빈값)의 `Analysis`(`StatementType`·`ReadOnly`·`MultipleStatements`·`Risk`·`Reasons` **순서와 문자열**·`Tokens`)를 base `afd3882` 의 `sqlsafe.go` 사본과 수정본에서 각각 덤프해 비교하면 **바이트 단위 동일**(diff 0). 덤프·사본·퍼즈 파일은 검증 후 전부 삭제하고 `git status` 로 확인한다.
  3. 사유를 **자르거나 요약하지 않는다**. 사유 개수·내용·순서는 전부 그대로다(사유 개수 상한은 별개 과제이며 기준 2 를 깨므로 섞지 말 것).
  4. 인과 확인: 임계값 상수를 maxint 로 고정해 **항상 선형 탐색으로 폴백**시키면 **스케일 테스트 3개만** red 가 되고 정합성 테스트는 전부 통과한다 → 순수 성능 변경임이 코드로 고정된다. 임계값 1·8 에서는 전부 통과.
  5. 중복 제거·순서를 고정하는 단위 테스트를 새로 쓰고, 그 테스트가 **base 에서도 통과**함을 먼저 확인한다(= 기존 동작을 고정한 것이지 바꾼 것이 아님).
  6. `go test ./... -count=1` exit 0 / 28 패키지 ok, `go test -race ./internal/domain/sqlsafe/` ok, `go vet ./...`, `go vet -tags=integration ./...`, `make check-go-format`, `go build ./...` 전부 exit 0.

- **설계 (이전 회차가 통과시킨 형태 — 그대로 쓸 것)**:
  - `sqlsafe.go` 에 **순서 보존 슬라이스 + lazy 멤버십 맵**을 담은 작은 타입 `reasonSet`(메서드 `add(string)`, `list() []string`)을 추가한다.
  - **슬라이스가 순서의 유일한 출처이고 맵은 멤버십만 답한다.** `list()` 가 맵을 순회해 사유를 만들게 하면 Go 맵 순회가 무작위라 기준 2 가 즉시 깨진다 — 이전 회차가 이것을 네거티브 컨트롤로 써서 덤프 diff 가 실제로 잡는 것을 확인했다.
  - 사유 수가 `reasonSetIndexThreshold`(8) 미만이면 맵을 만들지 않고 기존 선형 스캔 그대로 둔다(평범한 질의 전부 = 할당 증가 0).
  - **두 계층을 모두 고쳐라.** 방언 게이트만 고치면 병합 루프가 quadratic 을 그대로 다시 치른다(직전 측정: 16000개 합 250ms = 게이트 118ms + 병합 128ms).
  - 병합(`AnalyzeDialect` 131-133)은 `reasonSet` 을 **`result.Reasons` 로 시드**한 뒤 dialect 사유를 add 하라. 시드 없이 dialect 사유를 먼저 넣으면 순서가 바뀐다(`pg_sleep` 케이스에서 결정적으로 드러난다).
  - `appendUnique`(192) **자체는 지우지 말고 남겨라** — `Analyze` 의 호출부 두 곳(76·88)은 사유 카디널리티가 키워드 표 크기로 묶여 있어 고칠 필요가 없다.

- **건드릴 파일** (프로덕션 1개):
  - `internal/domain/sqlsafe/sqlsafe.go` — `reasonSet` 타입 + `reasonSetIndexThreshold` 상수 추가(이전 회차 규모 +70/-14);
    `postgresReadOnlyViolations`(418)의 `appendUnique` 6곳(430·436·444·473·479·499) → `reasonSet.add`, 반환은 `list()`;
    `oracleReadOnlyViolations`(1067)의 2곳(1096·1105) → 같은 교체(1070 의 database link `append` 도 집합으로 흡수);
    `AnalyzeDialect`(102)의 114·122 와 병합 루프 131-133 → `result.Reasons` 시드 + add.
  - `internal/domain/sqlsafe/sqlsafe_test.go` — 스케일 테스트 3개(64000개) + 중복 제거·순서 고정 테스트 1개 추가. **기존 스케일 테스트 2개가 그대로 쓸 수 있는 템플릿이다**: `TestAnalyzeDialectScalesLinearlyOverChainedPostgresCalls`(531), `TestAnalyzeDialectScalesLinearlyOverAliasedPostgresCalls`(614) — 같은 모양(질의 생성 → `time.Since` → 예산 초과 시 `t.Fatalf`)으로 쓰라. 이번 정찰 확인: base 의 `sqlsafe_test.go` 에 `TestAnalyzeDialectDeduplicatesAndOrdersViolationReasons` 는 **없다**(미머지 커밋이 추가했던 것) — 새로 써라.
  - 그 외 파일은 건드리지 말 것. `removeReason`(201)·허용목록(`postgresPureFunctions`·`oraclePureFunctions`·`oracleParenthesisKeywords`)·`scanOraclePolicy`·`callAdjacent`(724)·`postgresAliasColumnList`(752)·`postgresCTEColumnList`(964)·`cteWithList`(998)·`cteMemoFor`(1038)·`runtimeapi`·웹·`.github/workflows`·`migrations` 는 한 줄도 바꾸지 않는다.

- **검증 명령** (이 저장소에서 실제로 도는 것):
  - `go test ./internal/domain/sqlsafe/ -count=1 -run 'TestAnalyze' -v`
  - `go test -race ./internal/domain/sqlsafe/ -count=1`
  - `go test ./... -count=1` (28 패키지)
  - `go vet ./...` / `go vet -tags=integration ./...` / `make check-go-format` / `go build ./...`
  - 통합(권장, Go 변경이므로): 폐기 PostgreSQL 17 컨테이너를 **비어 있는 포트**에 띄워(과거 점유: 55432/55433/55439/55482/55492/55502/55517/55601/55617) `POSTGRES_DSN`·`QURIO_INTEGRATION_DSN`·`QURIO_TEST_POSTGRES_DSN` 세 env 를 같은 DB 로 주고 `cmd/qurio` 의 `TestIntegrationFreshInstallDatabase`→`TestIntegrationDatabaseMigrations` 로 bootstrap 후 `go test -race -p=1 -tags=integration ./... -count=1` (30 패키지). 끝나면 **컨테이너 제거**.
  - 웹: 변경이 Go 전용이면 `cd web && npm test --silent` 는 선택이지만, verify 게이트가 이것을 돌리므로 **최소 1회 돌려 exit 0 을 확인**하라. main 의 직전 커밋 `ab53169` 에서 직전 회차가 같은 명령 5회 연속 exit 0(`27 files / 134 tests`)을 확인했고 `afd3882` 는 VERSION 범프뿐이므로 녹색이 기대값이다(**이번 정찰은 재실행하지 않았다 — 미확인**). 깨지면 네 변경이 아니라 잔존 flake 이니 고치지 말고 보고하라.

- **위험과 피할 것**:
  - **맵 순회로 사유를 만들지 말 것**(위 설계 참조) — 기준 2 를 즉시 깬다.
  - **사유 개수 상한을 섞지 말 것** — 1MiB 질의가 여전히 사유 8만여 개(약 6.8MB)를 응답에 담는 것은 별개 `pending` 항목이고, 자르면 기준 2·3 이 깨진다.
  - 보호 경로(`internal/httpapi`·`oidcauth`·`mcpoauth`·`agentapi`·`migrations`·`.github/workflows`)는 건드릴 일이 없다.
  - **실 Oracle 인스턴스가 이 환경에 없다.** Oracle 증명은 게이트 판정 수준(sqlsafe 단위 테스트 + 판정 덤프)까지다 — 이번 변경은 판정이 바이트 단위로 불변이므로 이 한계는 가볍다. 솔직하게 그렇게 기록하라.
  - 선재 flake 2건: `intelligenceapi/credential_race_integration_test.go:236` 이 **같은 폐기 DB 2회차**에서 `qurio_secrets_check`(23514)로 깨질 수 있다(직전 회차 3회 연속 녹색으로 재현 실패). 나오면 DB 를 DROP/CREATE 하고 다시 돌려 네 변경이 아님을 보이고(`go list -deps -tags=integration ./internal/intelligenceapi` 에 sqlsafe 가 0건) **고치지 말고 보고**하라.
  - 이 환경의 bash 는 heredoc(`<<EOF`)을 거부한다 — 임시 덤프·퍼즈 파일 생성은 Write 도구로 하라.
  - **임시 파일을 반드시 지우고** `git status` 로 변경이 sqlsafe.go + sqlsafe_test.go 2개뿐임을 확인하라.

- **차선 후보**: **sqlsafe Oracle 게이트에 `WITH name (col, …) AS (…)` 컬럼 목록 면제 추가**(미머지 커밋 `3bf8799` 의 재시도, 가치 4 / 위험 3 / 작업량 S). 이번 정찰이 base 에서 재확인했다 — `oracleReadOnlyViolations`(1067) 본문에 `postgresCTEColumnList` 호출이 **없고**(호출부는 775 의 PostgreSQL 한 곳뿐), 따라서 `WITH recent (id, total) AS (…) SELECT * FROM recent` 가 Oracle 에서만 허위 사유 `Oracle SELECT의 비허용 함수 호출 recent()` 로 blocked 되는 방언 비대칭이 그대로다. 수정 형태는 PostgreSQL 호출부(775)와 **같은 위치·같은 모양**으로 `oracleReadOnlyViolations` 의 허용목록 검사 **앞**(위 본문의 `oracleParenthesisKeywords` 분기 앞)에 지연 `postgresTokenIndex` + 면제 `continue` 를 넣는 것이고, 헬퍼 본문은 한 줄도 바꾸지 않는다. `postgresAliasColumnList`(752) 전체를 Oracle 로 옮기는 것은 **금지**(그 `AS` 분기는 FROM 문맥 없이 무조건 면제해 Oracle 에서 fail-open 이 된다). 1순위와 **같은 파일**이므로 두 과제를 한 회차에 섞지 말 것.

# 과제서 — 2026-10-05 (base main@201bf2e, VERSION 1.4.11)

- 과제: `postgresCTEColumnList` 의 역방향 WITH 추적 루프를 메모이제이션해 PostgreSQL 게이트의 quadratic 제거 (가치 4 / 위험 2 / 작업량 S)

- 왜: `internal/domain/sqlsafe/sqlsafe.go:976-987` 의 역방향 루프가 함수 호출 후보 **사이트마다** 토큰 스트림을 0번까지 다시 훑는다. 같은 파일의 `fromItemContext`(882)는 이미 이 문제를 memo+trail 로 닫았고 그 주석(821-828)이 "a syntactically valid query well inside the payload limit of the validate endpoint pinned a core for minutes, reachable without any database privilege" 라고 사유를 적어 두었는데, 이 루프만 그 memo 밖에 남아 같은 결함이 다른 질의 모양으로 살아 있다. **이번 회차에 이 base 에서 직접 측정했다**(아래 수치).

## 측정 (내가 이 base 에서 실행한 결과 — 임시 테스트 파일로 재며, 작업 트리는 되돌렸다)
`SELECT upper (x) AS a0, upper (x) AS a1, … FROM public.events` (허용목록 함수라 `risk=low ro=true reasons=1` — 차단 경로의 비용이 섞이지 않는 순수 측정):

| terms | bytes | AnalyzeDialect(postgres) |
|---|---|---|
| 4000 | 78,914 | 111.9 ms |
| 8000 | 158,914 | 446.2 ms |
| 16000 | 324,914 | **1.759 s** |

바이트 2배마다 정확히 4배 = quadratic. validate 엔드포인트의 1MiB 한도까지 밀면 산술 추정 **~18초**(추정치, 미측정)이고 **DB 권한 없이** 도달한다.

인과는 측정으로 좁혔다: 같은 모양에서 `AS a_i` 만 떼면(`f0 (x), f1 (x), …`) 963행의 `AS` 접미사 검사가 먼저 false 를 돌려 이 루프가 아예 돌지 않고, 16000 terms 에서 1.996 s → 0.257 s 로 떨어진다(같은 모양·차단 경로 기준). 즉 이 루프가 그 질의 비용의 **약 87%** 다.

## 수용 기준
1) `internal/domain/sqlsafe/sqlsafe_test.go` 에 새 scale 테스트(위 표의 `upper (x) AS a{i}` 모양, terms=16000, 예산 **500ms**)를 추가하면 수정 전 `1.7s+` 로 **red**, 수정 후 **green**(수백 ms 가 아니라 수십 ms 가 될 것). 기존 `TestAnalyzeDialectScalesLinearlyOverChainedPostgresCalls`(sqlsafe_test.go:530)와 같은 형식·같은 "loose budget" 주석 관례를 따를 것.
2) 4000→8000→16000 terms 시간 증가가 선형에 가깝다(수정 후 doubling 당 ~2배, 4배가 아님). 테스트에 단정까지 넣지 않아도 되지만 회차 보고에 세 수치를 적을 것.
3) **판정 불변**: 현실 질의 70건+ × `postgres`·`oracle` = 140+ 판정을 base `201bf2e` 의 sqlsafe.go 와 수정본에서 각각 덤프해 기계적으로 비교 — **diff 0**(이 저장소의 확립된 증명 방식; 2026-09-30·10-02·10-03 회차가 모두 이 방식으로 통과했다). 특히 `WITH a (x) AS (…)`, `WITH RECURSIVE t (n) AS (…)`, `WITH a (x) AS (…), b (y) AS (…)`(두 번째 CTE 가 바로 이 루프를 타는 경로), `SELECT fn (x) AS a, side_effect_fn (y) AS b`(계속 blocked) 를 반드시 포함할 것.
4) 메모를 `return true` / `return false` 로 고정하면 기존 `postgresCTEColumnList` 관련 차단·허용 테스트가 깨지는 것을 보여 memo 가 실제 하중을 받는 것을 확인(2026-09-30·10-03 회차의 인과 확인 관례).
5) `go test ./... -count=1`, `go vet ./...`, `make check-go-format` 통과.

## 건드릴 파일 (프로덕션 1개 + 테스트 1개)
- `internal/domain/sqlsafe/sqlsafe.go:952 postgresCTEColumnList` — 976-987행의 역방향 루프 결과를 재사용. **같은 걸음을 그대로 걷고 답만 재사용**해야 한다(걸음을 바꾸면 판정이 바뀐다).
- `internal/domain/sqlsafe/sqlsafe.go:829 postgresTokenIndex` — memo 저장 필드 추가. 지금 구조체에는 `depths`/`matchingLeft`/`matchingRight`/`memo`/`trail` 이 있고 `build()`(846)에서 한 번에 만든다.
- `internal/domain/sqlsafe/sqlsafe_test.go` — 새 scale 테스트(530행 테스트 바로 아래가 자연스러운 자리).

### 설계 권고 (상태 공간을 정확히 보라 — 여기서 틀리면 fail-open 이 된다)
루프의 답은 **(시작 커서, columnDepth)** 두 값의 함수다. `columnDepth = depths[nameIndex]` 이고, 루프는 `depths[cursor] != columnDepth` 인 토큰을 **건너뛰기만** 한다 — 바깥 그룹으로 빠져나가 형제 그룹 안의 같은 깊이 토큰까지 보는 것이 현재 동작이다. 그래서 **커서만으로 키를 잡으면 깊이가 다른 두 사이트가 서로의 답을 훔쳐 판정이 바뀐다.**
- 권고: `fromItemContext`(882-949)와 같은 lazy memo+trail 패턴을, **깊이별로** 둘 것 — 예: `cteMemo map[int][]byte`(키 = columnDepth, 값 = 길이 `len(tokens)` 의 0/1/2 배열). 한 번의 걸음 동안 columnDepth 는 고정이므로 안전하고, 걸음이 커서 단조 감소이므로 trail 전체를 같은 답으로 칠할 수 있다(882의 주석이 그 논증을 이미 적어 두었다 — 재사용하라).
- 메모리 상한을 둘 것: 질의된 서로 다른 깊이가 많으면(예: 8 초과) 메모 없이 기존 걸음으로 돌아가게 하라. 어느 쪽이든 **답은 동일**하므로 판정에 영향이 없고, 깊게 중첩된 적대적 질의에서 O(n²) 메모리를 막는다. 측정한 실제 모양은 전부 깊이 0 이라 캐시 1개로 끝난다.
- `build()` 에서 전부 미리 계산하는 쪽을 고르고 싶다면, 위의 "형제 그룹까지 본다" 동작을 **정확히** 재현하는지 먼저 증명하라. 증명하지 못하면 lazy memo 를 쓸 것.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
go test ./internal/domain/sqlsafe/ -count=1 -v -run 'Scale|CTE|ColumnList|Ordinality|Alias'
go test ./internal/domain/sqlsafe/ -count=1
go test ./... -count=1          # 28 패키지, 통합 테스트는 -tags=integration 이라 제외
go vet ./...
make check-go-format
```
통합 테스트까지 돌릴 여유가 있으면 폐기 PostgreSQL 17 을 **비어 있는 포트**(55432/55433/55439/55482/55492/55502/55517 은 과거 점유 사례 — 피할 것)로 띄우고 `POSTGRES_DSN`·`QURIO_INTEGRATION_DSN`·`QURIO_TEST_POSTGRES_DSN` 세 env 를 같은 DB 로 지정한 뒤 fresh-install → migrations → `go test -race -p=1 -tags=integration ./... -count=1`. 컨테이너는 반드시 제거할 것.

## 위험과 피할 것
- **허용목록(`postgresPureFunctions`·`postgresParenthesisKeywords`)·`callAdjacent`(724)·`fromItemContext`(882)·`postgresAliasColumnList`(752)의 네 분기 본문은 한 줄도 건드리지 말 것.** 이 영역은 fail-open 으로 네 번 깨졌다(440aa9f·89b3e76·e1af783·7cda7f5). 이번 과제는 **판정을 바꾸지 않는 순수 성능 변경**이다 — 판정이 한 건이라도 달라지면 실패로 본다.
- **Oracle 경로는 손대지 말 것.** `oracleReadOnlyViolations`(1004)는 `postgresCTEColumnList` 를 호출하지 않는다 — 2026-10-03 회차가 그 비대칭을 고쳤으나(3bf8799) **이 base 에 미머지**다(이번 회차에 확인: `cteColumnList` 심볼 없음, `postgresCTEColumnList` 만 952행에 있음). 그 과제는 verify-failed 로 끝났으니 여기에 끼워 넣지 말 것.
- **같이 고치지 말 것 — 이번 회차 범위 밖의 별개 quadratic**: `postgresReadOnlyViolations`(418)이 사유를 `appendUnique`(473행 등)로 모으는데, 서로 다른 비허용 함수 이름이 수천 개인 차단 질의에서 `reasons` 가 그만큼 자라 append 당 O(len) 이 된다 — 16000개 이름에서 약 0.26s 로 실측했고 Oracle 게이트도 같다. 선재 결함이며 별도 과제다(ideas.json 에 올렸다).
- 보호 경로(`internal/httpapi`·`oidcauth`·`mcpoauth`·`migrations`·`.github/workflows`)는 건드릴 일이 없다.
- 과거 교훈: 성능 과제를 들고 와서 판정 증명을 생략하면 비평에서 막힌다. 70건+ 덤프 diff 를 **반드시** 남길 것.

## 차선 후보
`ci.yml:63`·`release.yml:99` 의 `go test -race -p=1 -tags=integration ./...` 에 `-count=1` 추가 + `scripts/ci_workflow_test.go` 회귀 테스트 (가치 4 / 위험 1 / 작업량 S). 2026-10-01 회차의 커밋 12223b0 이 **미머지**임을 이번 회차에 재확인했다 — `scripts/` 에 `ci_workflow_test.go` 가 없고(파일 목록: dev.sh·fake-ai-provider.go·offline-install.sh·release.sh·validate-release-contract.sh·verify-offline-image.sh), 두 워크플로의 그 줄에 `-count=1` 이 여전히 없다(같은 잡의 표적 `go test` 4곳은 모두 있다). `actions/setup-go` 캐시가 GOCACHE(테스트 결과 캐시 포함)를 보존하므로 통합 게이트가 DB 를 검증하지 않고 녹색이 될 수 있다. 워크플로가 보호 경로라 1순위로 올리지 않았다.

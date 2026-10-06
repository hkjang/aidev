# 과제서 — 2026-10-06 (base `bd3f310`, v1.4.12)

- 과제: sqlsafe 위반 사유 누적(`appendUnique`)의 quadratic 제거 — 서로 다른 비허용 이름이 많은 질의 하나가 **양 방언 모두** 11초의 단일 코어 CPU 와 7.3MB 의 사유를 만든다 (가치 4 / 위험 2 / 작업량 S)

## 왜
`appendUnique`(sqlsafe.go:192)는 append 마다 기존 값 전부와 문자열 비교를 하는 O(n) 함수인데, 사유 문자열이 **무한 카디널리티**인 자리 네 곳에서 토큰 루프 안에서 호출된다 — `postgresReadOnlyViolations`(418)의 함수 이름(473)·cast 형식(499)·문자열 상수 형식(444), `oracleReadOnlyViolations`(1067)의 함수 이름(1096). 모든 사유가 긴 공통 접두사("PostgreSQL SELECT의 비허용 함수 호출 ")를 공유해 비교마다 ~40바이트를 훑는다. 게다가 `AnalyzeDialect`(102)의 병합 루프(131-133)가 그 n 개를 또 `appendUnique` 로 옮겨 **같은 quadratic 을 한 번 더** 치른다.

프로덕션 진입점 `runtimeapi.(*API).validateSQL`(internal/runtimeapi/sql.go:41)은 인증된 principal 만 있으면 **DB 권한·승인 없이** 최대 `1<<20` 바이트(sql.go:50)를 그대로 `AnalyzeDialect` 에 넘긴다(`mcpserver/server.go:274` 도 같은 한도). 같은 파일의 `fromItemContext` 주석(821-828)이 이미 "a core pinned for minutes, reachable without any database privilege" 를 이 저장소의 수정 사유로 못박아 두었고, 직전 회차(5d8a1b5)가 같은 기준으로 머지됐다. 이 자리만 그 기준 밖에 남아 있다.

## 이번 회차 실측 (base `bd3f310`, 추정 아님)
| 입력 | PostgreSQL | Oracle |
|---|---|---|
| 서로 다른 이름 2000개 (20,914 B) | 8 ms | 5 ms |
| 서로 다른 이름 4000개 (42,914 B) | 20 ms | 21 ms |
| 서로 다른 이름 8000개 (86,914 B) | 89 ms | 87 ms |
| 서로 다른 이름 16000개 (180,914 B) | **233 ms** | **228 ms** |
| **같은 이름 16000번** (208,024 B, 대조군) | 14 ms | 12 ms |
| 서로 다른 **cast 형식** 16000개 (`x::t0, x::t1, …`) | **231 ms** | 8 ms |
| **요청 한도 그대로 1,048,562 B, 서로 다른 이름 88,304개** | **10.93 s** | **10.95 s** |

- 대조군이 핵심이다: 바이트가 **더 큰데도** 14ms — 비용은 길이가 아니라 **서로 다른 사유 개수**에서 온다.
- 1MiB 한도에서 사유 **88,304개 / 7,318,122 바이트**가 응답으로 나간다 (1MiB 요청 → ~7MB JSON 증폭).
- cast 행은 함수 이름 자리와 **독립된 두 번째 무한 카디널리티 자리**가 실재함을 보인다(Oracle 은 cast 를 검사하지 않아 8ms).
- 두 계층의 비중 실측(16000개, 합 250ms): `Analyze` 4ms(선형) / `postgresReadOnlyViolations` **118ms** / `AnalyzeDialect` 병합 루프 **128ms**. → **한쪽만 고치면 절반만 준다. 반드시 둘 다 고칠 것.**

## 수용 기준
1. 서로 다른 이름 4000/8000/16000개에서 소요 시간이 선형(doubling 당 약 2배). 기존 `TestAnalyzeDialectScalesLinearlyOverAliasedPostgresCalls`(sqlsafe_test.go:614)과 같은 형식의 red 테스트를 **먼저** 쓰고(base 에서 fail), 수정 후 green.
2. **양 방언 모두** 선형이어야 한다 — Oracle(228ms)도 같은 결함을 가지므로 PostgreSQL 전용 수정은 기준 미달.
3. `Analysis` 전체(`ReadOnly`/`Risk`/`Reasons` **순서와 문자열까지**)가 base 와 **바이트 단위로 동일**. 이 저장소의 확립된 방식대로 현실 질의 70건+ × postgres·oracle 판정을 base 파일과 수정본에서 덤프해 diff 0 을 보일 것. 사유를 **자르거나 개수를 제한하지 말 것**(기준 3 을 깨고 별개 과제다 — 아래 보류 항목 참조).
4. 인과 확인: 새 seen 집합을 우회(항상 선형 탐색으로 폴백)하도록 고정하면 스케일 테스트가 다시 red 가 되고 **정합성 테스트는 전부 통과**해야 한다 = 순수 성능 변경임을 코드로 고정. 직전 회차 교훈: 기존 스위트에 이 경로 커버리지가 없을 수 있으니, **없으면 사유 중복 제거(같은 이름이 여러 번 나올 때 사유 1개)와 사유 순서를 고정하는 테스트를 새로 쓰고 그것이 base 에서도 통과함**을 확인할 것.
5. `go test ./... -count=1` 0 FAIL, `go vet ./...`, `go vet -tags=integration ./...`, `make check-go-format` 통과.

## 건드릴 파일 (프로덕션 1개)
- `internal/domain/sqlsafe/sqlsafe.go` 만. 권고 형태: **순서 보존 슬라이스 + `map[string]struct{}` seen** 을 담는 작은 타입(예: `reasonSet` — `add(string)`, `list() []string`) 을 추가하고
  - `postgresReadOnlyViolations`(418) 의 `reasons` 를 그것으로 교체 (호출 네 곳: 430·436·444·473·479·499)
  - `oracleReadOnlyViolations`(1067) 의 `reasons` 를 그것으로 교체 (1072 의 `append`·1096·1105) — **1072 의 database link 사유가 첫 번째라는 순서를 유지**할 것
  - `AnalyzeDialect`(102) 의 병합 루프(131-133) — `result.Reasons` 를 **기존 값으로 시드한** seen 집합으로 병합. `Analyze` 가 넣은 사유와의 중복 제거 동작이 바뀌면 안 된다.
- `appendUnique`(192) 자체는 **지우지 말 것**: 48-95 의 `Analyze` 호출부(76·88)는 키워드 표 크기로 묶여 있어 그대로 두는 것이 변경면을 줄인다. `removeReason`(201)은 O(n) 이므로 손대지 않는다.
- 테스트: `internal/domain/sqlsafe/sqlsafe_test.go`.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
go test ./internal/domain/sqlsafe/ -count=1 -run TestAnalyzeDialect -v
go test -race ./internal/domain/sqlsafe/ -count=1
go test ./... -count=1
go vet ./... && go vet -tags=integration ./...
make check-go-format
go build ./...
```
통합까지 보려면 폐기 PostgreSQL 17 을 **비어 있는 포트**로 띄우고(과거 점유: 55432/55433/55439/55482/55492/55502/55517/55601) `POSTGRES_DSN`·`QURIO_INTEGRATION_DSN`·`QURIO_TEST_POSTGRES_DSN` 세 env 를 같은 DB 로 지정 → fresh-install → migrations → `go test -race -p=1 -tags=integration ./... -count=1`. 컨테이너는 반드시 제거.

## 위험과 피할 것
- **사유 문자열·순서가 바뀌면 안 된다.** UI 와 기존 테스트가 문자열을 그대로 본다. map 을 순회해 사유를 만들면 순서가 무작위가 되어 기준 3 이 깨진다 — 반드시 슬라이스가 순서의 유일한 출처여야 한다.
- 허용목록(`postgresPureFunctions`·`oraclePureFunctions`·`postgresParenthesisKeywords`·`oracleParenthesisKeywords`), `callAdjacent`(724), `postgresAliasColumnList`(752), `postgresCTEColumnList`(964), `postgresTokenIndex.fromItemContext`(894), `scanOraclePolicy` 는 **한 줄도 건드리지 말 것** — 이 다섯은 fail-open 오탐으로 다섯 번(440aa9f·89b3e76·e1af783·7cda7f5·433a183) 고쳐진 자리다. 이번 과제는 판정 로직이 아니라 사유 **수집**만 바꾼다.
- 보호 경로 회피: `.github/workflows`, `migrations`, `internal/httpapi`, `internal/oidcauth`, `internal/mcpoauth` 를 건드리지 않는다.
- 통합 스위트를 **같은 DB 에 두 번** 돌리면 `intelligenceapi/credential_race_integration_test.go:236` 이 `qurio_secrets_check` 위반으로 깨질 수 있다 — **선재 결함**이며 이 과제와 무관하다(`go list -deps -tags=integration ./internal/intelligenceapi` 에 sqlsafe 0건). 고치려 들지 말고 그렇게 기록할 것.
- 실 Oracle 인스턴스가 이 환경에 없다. Oracle 증명은 게이트 판정 수준까지만 가능하다(순수 성능 변경이므로 이번에는 한계가 가볍다).

## 차선 후보
**`intelligenceapi` credential_race 통합 테스트의 픽스처 격리 (3/2/S)** — `TestSSEDiscardsQueuedDeltaAfterProviderCredentialChange/delete` 가 같은 폐기 DB 2회차 실행에서 `qurio_secrets_check` 위반으로 깨진다(1회차·단독·DB 재생성 후에는 통과). `credential_race_integration_test.go` 의 provider/secret 픽스처에 `t.Cleanup(DELETE…)` 를 등록하거나 suffix 를 `qurio_secrets` 행까지 전파. 파일 1~2개, 프로덕션 코드 0줄. 3순위는 `legacyapi` 통합 테스트 `defer pool.Close()` → `t.Cleanup(pool.Close)` (한 회차에 5파일만, `runtimeapi`·`intelligenceapi`·`platformapi` 는 a8f0775 미머지 충돌 위험으로 제외).

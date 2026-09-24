# 정찰 과제서 — 2026-09-24 (base main@88a90a2, VERSION 1.4.2)

- 과제: `internal/store` 통합 테스트의 `defer pool.Close()` → `t.Cleanup(DELETE…)` 순서 역전 바로잡기 (가치 3 / 위험 1 / 작업량 S)

- 왜: Go 는 테스트 함수의 `defer` 를 먼저 실행하고 그 다음에 `t.Cleanup` 등록 함수를 실행한다(이번 세션에서 최소 재현 테스트로 확인: `seq=[body defer cleanup]`). 그래서 `internal/store` 의 네 통합 테스트는 픽스처를 지우는 `t.Cleanup` 이 **이미 닫힌 풀** 위에서 돌고, `_, _ = pool.Exec(...)` 로 오류를 버리기 때문에 아무 소리 없이 실패한다 — 정리 코드가 사실상 죽은 코드이고 `users`·`authentications`·`api_keys`·`evaluation_task_logs` 행이 매 실행마다 통합 DB 에 남는다. 고치면 정리 코드가 실제로 돌아 오래 쓰는 로컬 통합 DB 가 픽스처로 부풀지 않는다.

- 수용 기준:
  1) 아래 네 테스트를 실제 PostgreSQL 에 대고 돌린 뒤, 각 테스트가 만든 `users` 행(그리고 딸린 `authentications`/`api_keys`/`evaluation_task_logs`)이 DB 에 **0건** 남는다. 수정 전에는 같은 절차에서 남는 것을 먼저 보여 red 를 만든다(psql 카운트 또는 별도 검증 테스트 중 편한 쪽. 소스 문자열 검사는 증거로 쓰지 말 것).
  2) 정리는 성공하고, 풀 종료는 여전히 테스트 끝에서 일어난다 — 즉 `pool.Close` 가 **가장 마지막**에 실행된다(`t.Cleanup` 은 LIFO 이므로 `pool.Close` 를 DELETE 정리보다 **먼저** 등록해야 한다).
  3) 네 테스트의 기존 단언(레이트 리스 공유·회전 계보·프로필 선호값 저장/삭제)은 그대로 통과하고, 테스트 본문의 검증 로직은 바뀌지 않는다.
  4) `go vet -tags=integration ./internal/store` 와 `gofmt -l internal/store` 가 깨끗하다.

- 건드릴 파일 (수정 대상은 이 두 개뿐):
  - `internal/store/api_key_rate_integration_test.go`
    - `TestDurableAPIKeyRateIsSharedAndSurvivesTaskDeletionPostgres` — 28행 `defer pool.Close()` / 38행 `t.Cleanup(DELETE api_keys, evaluation_task_logs, users)`
    - `TestDurableAPIKeyConcurrentLeaseIsSharedAcrossProcessesPostgres` — 119행 / 129행
    - `TestDurableAPIKeyRateFollowsRotationLineagePostgres` — 226행 / 236행
  - `internal/store/user_preferences_integration_test.go`
    - `TestUserProfilePreferencesPostgres` — 31행 `defer pool.Close()` / 46행 `t.Cleanup(DELETE authentications, users)`
  - 고치는 법(권장, 최소 변경): `defer pool.Close()` 를 **같은 자리에서** `t.Cleanup(pool.Close)` 로 바꾼다. 같은 패턴을 이미 쓰는 선례가 저장소에 있다 — `internal/authzread/guard_integration_test.go:29,151`, `internal/legacyapi/admin_access_boundary_integration_test.go:33`, `internal/legacyapi/agent_feedback_admin_integration_test.go:29`. 등록 순서상 `pool.Close` 가 먼저 등록되므로 LIFO 로 마지막에 실행되어 DELETE 정리가 살아 있는 풀 위에서 돈다.
  - 참고: `t.Cleanup` 안의 `_, _ =` 를 오류 검사로 바꿀지는 선택. 바꾼다면 `t.Errorf` 가 아니라 `t.Logf` 로만 남겨 기존 통과/실패 판정을 흔들지 말 것(수용 기준 3).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - DB 없이: `go vet -tags=integration ./internal/store`, `gofmt -l internal/store`, `go test ./internal/store`(integration 태그 없는 단위 테스트), `go build ./...`
  - 실제 DB (이 환경은 도커로 가능함이 과거 회차에서 확인됨):
    1. `docker run -d --rm --name qurio-recon-pg -e POSTGRES_USER=qurio -e POSTGRES_PASSWORD=qurio-test -e POSTGRES_DB=qurio -p 127.0.0.1:55432:5432 postgres:17-alpine` (55432 가 점유돼 있으면 다른 포트를 쓰고 DSN 을 맞출 것 — 2026-09-20 회차에 실제로 점유 충돌이 있었다)
    2. `export POSTGRES_DSN=postgres://qurio:qurio-test@127.0.0.1:55432/qurio?sslmode=disable` (이 두 파일은 아직 `POSTGRES_DSN` 을 읽고 기본값이 `…@127.0.0.1:55432/qurio` 다 — 이번 과제에서 이 env 이름은 **바꾸지 말 것**, 아래 위험 참조)
    3. 부트스트랩: `go test -tags=integration ./cmd/qurio -run '^TestIntegrationDatabaseMigrations$' -count=1`
    4. `go test -race -p=1 -tags=integration ./internal/store -count=1` 2회 연속
    5. 잔존 확인: `docker exec qurio-recon-pg psql -U qurio -d qurio -c "SELECT count(*) FROM users WHERE name LIKE 'profile-%' OR name LIKE 'ratekey-%';"` → 0. (실제 nonce 접두사는 테스트 소스에서 확인할 것. `user_preferences` 는 `"profile-"+nonce` 로 확인됨, `api_key_rate` 쪽 접두사는 **미확인**이니 읽고 맞출 것.)
    6. 끝나면 `docker rm -f qurio-recon-pg`
  - 가능하면 마지막에 `go test -race -p=1 -tags=integration ./...` 한 번.

- 위험과 피할 것:
  - **DSN env 이름을 건드리지 말 것.** 이 두 파일이 읽는 `POSTGRES_DSN` 은 프로덕션 설정 키(`internal/config/config.go`)와 같은 이름이라 따로 손볼 가치가 있지만, 저장소의 **33개** 통합 테스트 파일이 같은 관례를 쓰고 CI(`.github/workflows/ci.yml:29-31`)가 세 DSN 을 모두 같은 폐기 DB 에 주고 있다. 한 패키지만 바꾸면 세 번째 관례가 생긴다 — 별도 과제로 남기고 이번엔 순서만 고친다.
  - **연결 실패 → `t.Skip`** 분기(24-26, 115-117, 222-224, 27-29행)도 같은 이유로 이번 범위 밖이다(dbexec 는 9a9577b 에서 이미 Fatal 로 바꿨지만 store 는 33개 파일과 같은 관례를 공유한다).
  - **`defer func(){ …DELETE… }()` 를 쓰는 파일은 정상이다 — 고치지 말 것.** `internal/store/api_keys_integration_test.go`, `approvals_integration_test.go`, `auth_oidc_integration_test.go`, `passwords_integration_test.go` 네 파일은 DELETE 를 `defer pool.Close()` **뒤에** 등록해 LIFO 로 먼저 돈다. 이미 맞다.
  - 같은 순서 역전은 다른 패키지에도 있다(확인된 곳: `cmd/qurio/migration_integration_test.go`, `internal/platformapi/query_history_integration_test.go`, `internal/intelligenceapi/credential_race_integration_test.go`, `internal/agentapi/repository_integration_test.go`, `internal/runtimeapi/imports_integration_test.go`). **이번 회차에 같이 고치지 말 것** — 한 세션에 실제 DB 로 증명할 수 있는 범위를 넘고, 보호 경로(httpapi/agentapi)로 번진다. ideas.json 에 후속 항목으로 남겼다.
  - 프로덕션 코드(`internal/store/*.go`), migrations, `.github/workflows`, auth/session 경로는 손대지 않는다. 이 과제는 `_integration_test.go` 두 파일만 바꾼다.
  - 과거 교훈: 손으로 만든 대역이나 소스 문자열 검사로 증명하지 말 것. 반드시 진짜 PostgreSQL 에 붙여 red → green → 되돌려 red 를 확인할 것.

- 차선 후보: `internal/store` 통합 테스트 여섯 파일의 DSN 선택을 `QURIO_TEST_POSTGRES_DSN` 전용으로 바꾸고 연결 오류를 `t.Fatal` 로 전환(9a9577b 가 `internal/domain/dbexec` 에 한 것과 같은 모양, 회귀 테스트는 `internal/domain/dbexec/integration_config_test.go` 를 본뜰 것) — 1순위가 이미 고쳐져 있거나 실제 DB 를 띄울 수 없을 때 고른다. 단 이 경우 33개 파일 중 store 6개만 바뀌는 부분 통일이 되므로, 구현자는 나머지 27개도 같은 관례를 따르도록 후속 과제를 ideas 에 명시적으로 남길 것.

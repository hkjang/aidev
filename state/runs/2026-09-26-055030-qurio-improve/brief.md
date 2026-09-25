- 과제: `defer pool.Close()` → `t.Cleanup(pool.Close)` 순서 역전을 runtimeapi·intelligenceapi·platformapi 통합 테스트로 확장 (가치 3 / 위험 1 / 작업량 M)

- 왜: 522f0e5 가 `internal/store` 네 곳에서 고친 것과 **정확히 같은 결함**이 세 패키지에 그대로 남아 있다 — Go 는 테스트 본문의 `defer` 를 `t.Cleanup` 보다 먼저 실행하므로, `defer pool.Close()` 를 쓰는 테스트의 `t.Cleanup(func(){ ... DELETE FROM ... })` 픽스처 정리는 **이미 닫힌 풀** 위에서 돌고 `_, _ =` 로 오류가 버려져 조용히 죽는다. 결과로 공유 테스트 DB 에 users·api_keys·qurio_ai_providers·qurio_secrets·sql_projects·meta_word_dict 행이 계속 쌓여 CI(세 DSN 이 같은 DB 를 가리킴)에서 뒤 테스트가 다른 이유로 흔들린다.

- 수용 기준:
  1) 아래 "건드릴 파일" 의 **19개 지점 전부**가 `t.Cleanup(pool.Close)` 로 바뀐다. 한 패키지 안에 고치다 만 파일을 남기지 말 것 — 남기면 수용 기준 3 의 "패키지 실행 후 잔존 0" 을 만족할 수 없다. 각 테스트 본문의 검증 로직·DSN 선택·Skip 분기·`cleanupCtx` 정의·프로덕션 코드는 한 줄도 바꾸지 않는다(diff 는 19줄).
  2) **수정 전에 red 를 실제 PostgreSQL 로 만들어 둘 것**(grep 이나 "고쳤으니 될 것" 은 증거가 아니다): 깨끗한 폐기 DB 를 bootstrap 한 뒤 세 패키지를 각각 돌리고 `psql` 잔존 카운트가 0 이 아님을 기록한다. 최소한 `internal/runtimeapi/imports_integration_test.go`(meta_word_dict/user_meta_logical_models)와 `internal/intelligenceapi/credential_race_integration_test.go`(qurio_ai_providers/qurio_secrets/api_keys/users)에서 잔존이 나와야 한다.
  3) 수정 후 같은 절차에서 세 패키지 각각 통합 테스트 PASS + 잔존 카운트 0.
  4) 인과 증명: 한 파일(예: `imports_integration_test.go`)만 되돌려 다시 돌리면 그 파일이 만든 행만 다시 남는 것을 확인하고 복원한다.
  5) `go test -race -p=1 -tags=integration ./internal/platformapi ./internal/intelligenceapi ./internal/runtimeapi -count=1` 이 2회 연속 통과하고, 매 회차 뒤 잔존 0.

- 건드릴 파일 (이 회차에 실제로 grep/열어 확인한 행 번호 — 편집 전 재확인할 것):
  - `internal/platformapi/query_history_integration_test.go:30` — `TestListQueriesUnionsOnlyOwnedAgentRunsWithScopesAndRestorationPath`. 78~82행 `t.Cleanup` 이 qurio_agent_conversations / qurio_query_runs / sql_projects / users 를 지운다. (platformapi 에서 `defer pool.Close()` 는 이 한 곳뿐 — 나머지 platformapi 통합 파일은 이 패턴을 쓰지 않는다.)
  - `internal/intelligenceapi/credential_race_integration_test.go:40, 151, 273, 360, 485, 617, 775` — 7개 테스트(`TestSSEDiscardsQueuedDeltaAfterAPIKeyRevocation`, `TestSSEDiscardsQueuedDeltaAfterProviderCredentialChange`, `TestAdminProviderScopeDeniesDifferentProvider`, `TestIntelligenceListsIntersectProfileAndLegacyModelScopesBeforePagination`, `TestCompletionRejectsCredentialRotationAndDeletionDuringProviderCall`, `TestAuthorizedProviderCredentialMutationRejectsConcurrentKeyPolicyChange`, `TestAuthorizedCredentialMutationRevalidatesExactSessionAndEffectiveRole`). 각 `t.Cleanup` 이 qurio_ai_usage / api_keys / qurio_ai_providers / qurio_secrets / authentications / users / qurio_roles 를 지운다.
  - `internal/runtimeapi/imports_integration_test.go:33, 97` — `TestSpreadsheetCommitAgainstMigratedPostgres`, `TestSpreadsheetCommitRevalidatesAPIKeyScopesAndRollsBack`. 51·64행 등 `t.Cleanup` 이 meta_word_dict / user_meta_logical_models 를 지운다.
  - `internal/runtimeapi/services_integration_test.go:38, 109, 192, 374, 516` — `t.Cleanup(func…)` 4개 존재.
  - `internal/runtimeapi/publication_integration_test.go:81` — `t.Cleanup(func…)` 1개.
  - `internal/runtimeapi/mcp_integration_test.go:27` — `t.Cleanup(func…)` 1개.
  - `internal/runtimeapi/database_drafts_integration_test.go:43`, `internal/runtimeapi/agent_instructions_integration_test.go:31` — 이 둘은 `t.Cleanup(func…)` 이 0개라 지금 당장 잃는 정리는 없지만, 같은 패키지 안에서 관례를 갈라 두면 다음 사람이 또 틀린다. 같이 바꿀 것.
  - 변경은 각 행에서 `defer pool.Close()` → `t.Cleanup(pool.Close)` 한 줄뿐. `t.Cleanup` 은 LIFO 라 가장 먼저 등록된 풀 종료가 마지막에 실행된다 — **등록 위치를 옮기지 말 것**(기존 `defer` 와 같은 자리에 둬야 픽스처 정리보다 먼저 등록된다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 폐기 DB: `docker run -d --rm --name qurio-it -e POSTGRES_USER=qurio -e POSTGRES_PASSWORD=qurio-test -e POSTGRES_DB=qurio -p 127.0.0.1:55432:5432 postgres:17-alpine` (다른 세션이 55432 를 쓰면 55433 등으로 옮길 것 — 2026-09-20 회차에 실제로 충돌했다.)
  - 세 env 를 같은 DSN 으로: `export QURIO_TEST_POSTGRES_DSN='postgres://qurio:qurio-test@127.0.0.1:55432/qurio?sslmode=disable'; export POSTGRES_DSN="$QURIO_TEST_POSTGRES_DSN"; export QURIO_INTEGRATION_DSN="$QURIO_TEST_POSTGRES_DSN"` (`.github/workflows/ci.yml:29-31` 과 동일한 구성).
  - 마이그레이션 적용(README 82~91행 절차): `go test -tags=integration ./cmd/qurio -run '^TestIntegrationDatabaseMigrations$' -count=1`
  - red/green: `go test -race -p=1 -tags=integration ./internal/runtimeapi -count=1` 등 패키지별로, 마지막에 세 패키지 한 번에.
  - 잔존 카운트: `psql "$QURIO_TEST_POSTGRES_DSN" -c "select (select count(*) from users where id>1) u, (select count(*) from api_keys) k, (select count(*) from qurio_ai_providers) p, (select count(*) from qurio_secrets) s, (select count(*) from sql_projects) sp, (select count(*) from meta_word_dict) w, (select count(*) from user_meta_logical_models) lm;"` (컬럼명은 위 파일들의 DELETE 문에서 확인했다. `users` 는 bootstrap 관리자가 id=1 이라 `id>1` 로 센다 — 522f0e5 회차의 경험.)
  - 회귀 없음: `go vet ./...`, `go vet -tags=integration ./internal/platformapi ./internal/intelligenceapi ./internal/runtimeapi`, `gofmt -l cmd internal migrations scripts`(출력 없어야 함 — a432f3f 이후 `make lint` 가 검사한다), `go test ./...`, `go build ./...`.
  - 끝나면 컨테이너 제거.

- 위험과 피할 것:
  - **보호 경로는 이번에 건드리지 말 것**: `internal/agentapi/repository_integration_test.go:33,187,259,387` 와 `internal/httpapi/{api_key_rate:35,167, read_guard:33, key_delegation:39}`, `internal/legacyapi/*`, `internal/store/{api_keys,passwords,auth_oidc,approvals}_integration_test.go`, `internal/jobs`, `internal/backup`, `internal/integrations`, `internal/domain/legacy` 에도 같은 결함이 남아 있다(총 52곳 중 이번 19곳). 인증·세션·키 경로라 별도 회차로 미룬다. 브리프 범위 밖 파일을 diff 에 넣지 말 것.
  - `internal/intelligenceapi/repository_test.go` 는 `//go:build integration` 태그가 없고 `t.Cleanup(func…)` 도 0개다. 범위에서 제외한다(포함하면 태그 없는 파일의 동작을 같이 건드리게 된다).
  - `t.Cleanup` 안의 버려진 오류(`_, _ =`)를 `t.Errorf` 로 바꾸지 말 것 — DB 가 사라진 환경에서 새 실패를 만든다. `t.Logf` 로 남기는 것은 **별도 과제**이며 이번 수용 기준에 없다.
  - `internal/domain/dbexec` 는 `public` 고정명 픽스처와 pg_catalog 캐스트를 CREATE/DROP 하므로 같은 DB 에서 병렬로 돌리지 말 것. `-p=1` 을 반드시 유지.
  - 마이그레이션·`.github/workflows`·프로덕션 코드·릴리즈 경로는 건드리지 않는다.

- 차선 후보: `internal/store` 통합 테스트 6개 파일(`api_keys`, `passwords`, `auth_oidc`, `approvals` 등)의 DSN 을 `QURIO_TEST_POSTGRES_DSN` 전용으로 바꾸고 연결 오류를 Skip 대신 Fatal 로 (가치 3 / 위험 2 / M) — 9a9577b 가 `internal/domain/dbexec` 에 한 것과 같은 모양이고 회귀 테스트는 `internal/domain/dbexec/integration_config_test.go` 를 본뜬다. 1순위가 실제 PostgreSQL 을 띄우지 못해 red 를 만들 수 없을 때 고를 것(이쪽은 자식 프로세스 테스트라 실제 DB 없이도 red 를 만들 수 있다). 다만 저장소 나머지 27개 통합 파일이 `POSTGRES_DSN` 관례를 공유하므로, 후속 통일 과제를 커밋 메시지·회차 노트에 명시적으로 남길 것.

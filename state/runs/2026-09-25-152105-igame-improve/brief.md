- 과제: Migrate 의 context 취소 경로가 부분 적용을 남기지 않고 재시도 가능한지 실제 PostgreSQL 로 검증 (가치 3 / 위험 1 / 작업량 M)
- 왜: `internal/database/database.go:95 Migrate` 는 DDL/이력 INSERT 가 실패하면 **이미 취소된 같은 ctx** 로 `tx.Rollback(ctx)` 를 부르고 에러를 버린다(`database.go:128-131`, `_ = tx.Rollback(ctx)`). 09-21 에 들어간 PG 테스트 3개(`TestMigrateAppliesEmbeddedFilesAndIsIdempotent` / `TestMigrateRejectsChangedChecksumAndRecovers` / `TestMigrateRollsBackDDLAndHistoryThenRetries`)는 모두 **서버 예외**로만 실패를 만들어 취소 경로를 한 줄도 덮지 않는다 — 기동 중 취소(컨테이너 healthcheck 타임아웃, SIGTERM, DSN 타임아웃)가 `schema_migrations` 와 실제 스키마를 어긋난 채로 남기는지 아무도 모른다. 덮고 나면 기동 실패 뒤 재기동이 안전하다는 것이 증거로 남는다.

- 수용 기준:
  1) 010_silent_sso.sql 의 트랜잭션이 열려 있는 동안 ctx 를 취소하면 `Migrate` 가 에러를 돌려주고, 그 에러가 `context.Canceled` 를 감싼다(`errors.Is`). 취소 시점이 010 안이었음을 테스트가 확인한다(아래 "취소 시점 잡기").
  2) 그 직후 **새(취소되지 않은) ctx** 로 관찰했을 때 `schema_migrations` 에 010 행이 없고, 010 이 만드는 산물인 `oidc_flows.silent` 컬럼도 없다. 010 이전 마이그레이션의 이력과 seed(`migrationSeed`)는 그대로다.
  3) 방해 요소를 치운 뒤 **같은 pool** 에 새 ctx 로 `Migrate` 를 다시 부르면 성공하고, 전체 이력이 채워지고 `oidc_flows.silent` 가 존재한다 — 즉 취소로 죽은 커넥션이 pool 을 못 쓰게 만들지 않는다.
  - 만약 1~3 중 하나라도 깨지면: 최소 수정은 `Migrate` 의 롤백을 부모와 분리된 짧은 컨텍스트(`context.WithTimeout(context.WithoutCancel(ctx), …)`)로 수행하는 것. 깨지지 않으면 **소스는 한 줄도 고치지 말고** 테스트만 남길 것(09-21 과 같은 형태).

- 건드릴 파일:
  - `internal/database/database_pg_test.go` — 테스트 1개 추가(예: `TestMigrateCancelledMidMigrationLeavesNothingAndRetries`). 기존 헬퍼 `migrationPool(t)`(24행, UUID 전용 스키마 + pgcrypto 확장 스키마 확인 + CASCADE 정리), `embeddedMigrationNames`, `migrationHistory`, `assertMigrationHistory`, `migrationSeed` 를 그대로 재사용한다. `TestMigrateRollsBackDDLAndHistoryThenRetries`(208행)가 거의 그대로 쓸 수 있는 틀이다 — `schema_migrations` 를 미리 만들고 `BEFORE INSERT` 트리거로 010 을 가로채는 부분까지 동일하고, 트리거 본문만 `RAISE EXCEPTION` 대신 "붙잡아 두기"로 바꾼다.
  - `internal/database/database.go:95 Migrate` — 수용 기준이 깨졌을 때만. 그 외에는 읽기만.

- 취소 시점 잡기(타이밍에 기대지 말 것):
  - 권장: 어드바이저리 락. 테스트가 별도 커넥션에서 `pg_advisory_lock(k)` 를 잡고, 트리거는 `NEW.name='010_silent_sso.sql'` 일 때 `PERFORM pg_advisory_lock(k)` 로 **블록**된다. 테스트는 `pg_locks` 에 `locktype='advisory' AND objid=k AND NOT granted` 인 대기자가 보일 때까지 폴링한 뒤 `cancel()` 을 호출한다 — 이러면 "010 트랜잭션이 열린 채 취소" 가 시간이 아니라 상태로 보장된다. 취소 후 테스트 쪽 락을 풀어 준다.
  - `pg_sleep` + 고정 `time.Sleep` 으로 때려 맞히지 말 것. CI 에서 흔들린다.
  - ctx 는 `migrationPool` 이 돌려준 ctx 의 자식으로 만든다: `mctx, cancel := context.WithCancel(ctx)`. 관찰·재시도는 부모 `ctx`(2분 타임아웃, 살아 있음)로 한다.
  - 취소되면 pgx 가 그 커넥션을 폐기한다. 관찰 쿼리가 "conn closed" 로 실패하면 그건 pool 의 새 커넥션을 쓰라는 뜻이지 계약 위반이 아니다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 일회용 DB 띄우기: 로컬에 `postgres:17-alpine` 이미지가 이미 있다(확인함). README 의 준비 절차대로 **pgcrypto 를 public 이 아닌 전용 확장 스키마에 미리 설치**하고 DSN `search_path` 에 포함시켜야 `migrationPool` 이 skip/Fatal 하지 않는다.
  - `IGAME_TEST_DSN='postgres://…' go test ./internal/database/... -run TestMigrate -count=1 -v`
  - `IGAME_TEST_DSN='postgres://…' go test ./internal/database/... -run TestMigrateCancelled -race -count=3`
  - `make test-db DSN='postgres://…'` (internal/api + internal/database 전체)
  - `gofmt -l .` / `go vet ./...` / `go test ./cmd/... ./internal/... ./migrations/...` / `bash scripts/check-release-contract.sh`

- 위험과 피할 것:
  - `migrations/*.sql` 은 배포된 파일이고 checksum 불변이다 — **절대 수정 금지**. 010 을 실패시키는 수단은 테스트 스키마 안의 트리거뿐이다.
  - `migrationPool` 이 만드는 UUID 전용 스키마 밖으로 나가지 말 것. public 이나 운영 DB 를 쓰면 안 된다. pgcrypto 확장 자체는 지우지 말 것(fixture 가 확장은 보존한다).
  - `_ = tx.Rollback(ctx)` 를 "정리" 하겠다고 Migrate 전반을 리팩터하지 말 것. 수용 기준이 깨졌을 때의 최소 변경만.
  - 새 헬퍼 함수를 여러 개 만들어 기존 세 테스트를 다시 쓰지 말 것 — 기존 테스트는 손대지 않는다.
  - 릴리즈·워크플로·auth/session 경로는 이번 과제와 무관하다. 건드리지 말 것.
  - 과거 교훈: 손으로 만든 대역으로 증명하지 말 것. 반드시 진짜 `Migrate` + 진짜 embedded SQL + 진짜 PostgreSQL 로 확인한다. 소스 문자열 grep 은 증거가 아니다.

- 차선 후보: `startGameSession` 의 `metadata: null` 이 400 으로 걸리지 않고 `game_sessions.client_info` 에 JSON null 로 저장됨 — `internal/api/catalog.go:206-213`. `json.Unmarshal([]byte("null"), &map)` 는 map 대상에 오류 없이 통과한다(`finishGameSession` 은 `*map[string]any` 포인터 대상으로 이미 걸러낸다, catalog.go:496-500). 같은 방식으로 포인터 대상 + nil 검사를 넣고, `internal/api/catalog_pg_test.go` 계열에 진짜 `Router()` 로 `POST /games/{slug}/sessions` 를 보내 400 `invalid_metadata` 와 행의 `client_info` 불변을 확인한다. 기존 `invalid_metadata` 에러 문자열은 바꾸지 말 것(다른 테스트가 본다).

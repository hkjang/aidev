- 과제: API PostgreSQL fixture를 호출별 전용 스키마로 격리 (가치 3 / 위험 2 / 작업량 M)
- 왜: `internal/api/admin_pg_test.go:migratedPool`은 DSN 기본 스키마에서 Migrate를 실행하므로 독립 fixture들이 `system_settings`와 테스트 행을 공유한다. 이미 database 패키지에서 쓰는 UUID 스키마 패턴을 적용하면 다른 fixture 및 기본 스키마의 데이터를 건드리지 않고 API 회귀를 실행할 수 있다.
- 수용 기준: 1) `migratedPool(t)` 호출마다 고유 `api_test_<uuid>` 스키마에 실제 embedded migration을 적용하고, 두 pool에서 같은 setting key에 서로 다른 값을 써도 각 값이 유지된다. 2) 두 연결을 동시에 acquire해 각각 `current_schema()`와 `current_setting('search_path')`를 검증한다. 검색 경로는 자기 스키마와 사전 설치한 pgcrypto 전용 스키마뿐이며 public이나 DSN의 기존 앱 스키마로 폴백하지 않는다. 3) 자식 `t.Run` 종료 후 외부 관리 연결로 해당 스키마가 삭제됐고, 형제 fixture 및 pgcrypto 확장 OID/namespace가 보존됐음을 확인한다. 새 테스트는 기존 helper에서 격리 검증이 실패하고 변경 후 통과해야 한다. DSN 없음만 skip, 연결·권한·확장 준비 오류는 실패한다.
- 건드릴 파일: `internal/api/admin_pg_test.go:migratedPool` — 스키마 생성·startup search_path·cleanup 순서; `internal/api/fixture_pg_test.go`(신규, 제안 테스트명 `TestMigratedPoolIsolation`) — 실제 PG 격리·연결·정리 회귀; `internal/api/settings_put_pg_test.go:settingFixture.preserve` — 기본 스키마 공유라는 낡은 주석만 수정(복원 동작 유지); `README.md:개발과 검증` — API도 fixture별 스키마를 쓴다는 설명. 프로덕션 파일 0개, 총 4개. `internal/database/database_pg_test.go:migrationPool`은 읽기 전용 참고.
- 검증 명령: 아래 순서의 명령을 저장소 루트에서 실행한다. DSN은 일회용 PostgreSQL만 사용한다.
  - `go test ./internal/api/... ./internal/database/... -count=1`
  - `IGAME_TEST_DSN='postgres://igame:igame@127.0.0.1:25432/igame?sslmode=disable&search_path=public,igame_test_extensions' go test ./internal/api/... -run '^TestMigratedPoolIsolation$' -count=1 -v`
  - `make test-db DSN='postgres://igame:igame@127.0.0.1:25432/igame?sslmode=disable&search_path=public,igame_test_extensions'`
  - `IGAME_TEST_DSN='postgres://igame:igame@127.0.0.1:25432/igame?sslmode=disable&search_path=public,igame_test_extensions' go test ./internal/api/... -run '^TestMigratedPoolIsolation$' -race -count=3`
  - `go vet ./internal/api/...`; `git diff --check`; `bash scripts/check-release-contract.sh`
- 위험과 피할 것: auth/session 제품 코드, migrations SQL·체크섬, workflows, 새 런타임 설정, 버전, 프런트, SDK는 범위 밖이다. 테스트 전부의 t.Parallel 도입·수동 cleanup 삭제·패키지 간 fixture 공용화도 하지 않는다. `newJSONBFixture`를 부모에서 한 번 만든 서브테스트들은 여전히 같은 fixture를 공유한다. 이번 격리는 그 내부까지 자동으로 나누지 않으므로 기존 행 수 델타 및 `preserve` 복원을 유지한다. PG 확장은 DB 전역이므로 생성된 테스트 스키마에 설치하면 안 된다. 기본 스키마를 DROP/TRUNCATE하지 않는다. 전체 DB 테스트가 깨지면 제품 코드를 고치지 말고 fixture 가정을 먼저 확인한다.
- 차선 후보: README RealmGuard 서버 재현 설명 정합성 개선 — README의 RealmGuard 마지막 문단은 여전히 `server_received_telemetry_v1` 및 완전한 시뮬레이션이 아니라는 구형 설명이다. `docs/realmguard.md:114~152`와 `internal/api/realmguard_replay.go:replayRealmGuardBattle`의 `server_replay_v1` 계약을 기준으로 그 문단만 수정하고 버전·Defense 설명은 바꾸지 않는다. 검증은 `bash scripts/check-release-contract.sh` 및 `git diff --check`. 1순위가 45분 내 fixture 범위에 머물 수 없을 때만 전환 사유를 기록한다.

범위 및 구현 단계 (모두 구현자 미착수; 사람 승인 체크포인트 없음)
1. 준비·Red: README처럼 disposable PG17에 전용 확장 스키마를 먼저 만든다. 새 `TestMigratedPoolIsolation`에서 두 migratedPool을 생성하고 current_schema가 다름을 요구해 기존 코드 실패를 확인한다. 검증: 위 새 테스트 명령. 체크포인트: 실패가 연결 오류가 아닌 공유 스키마 때문인지 확인한 뒤 진행.
2. helper 수정: 기존 함수 시그니처 유지. 별도 관리 pool로 pg_extension/pg_namespace를 읽고 확장 없음 또는 public 설치를 명확히 거부한다. `pgx.Identifier{schema}.Sanitize()`로 UUID 스키마 식별자를 인용하고 CREATE 직후 DROP cleanup을 등록한다. `pgxpool.ParseConfig` 후 RuntimeParams의 search_path를 자기 스키마와 확장 스키마로 덮어써 모든 신규·교체 연결에 적용한다. NewWithConfig 뒤 pool.Close를 cleanup으로 등록하고 Migrate를 호출한다. 초기화와 DROP은 유한 context를 사용하며 DROP은 이미 취소된 초기화 context를 재사용하지 않는다. LIFO 순서는 기존 행/HTTP fixture cleanup → 앱 pool.Close → DROP SCHEMA CASCADE → 관리 pool.Close가 되어야 한다. MaxConns=2를 database 예제에서 기계적으로 복사할 필요는 없다. 검증: 새 테스트 통과. 체크포인트: pool 연결·트랜잭션을 모두 release한 뒤 자식 t.Run이 끝나는지 확인.
3. 증거 강화·회귀: 새 테스트에서 두 pool의 동일 setting key 독립성, 동시에 잡은 연결들의 search_path, 자식 종료 후 스키마 소멸·형제 및 확장 보존을 검증한다. 기존 helper로 되돌렸을 때 실패함을 확인하고 변경을 복구한다. 검증: make test-db 및 새 테스트 race 반복. 체크포인트: 실패 시 계획을 기록·수정하고 범위를 넓히지 않는다.
4. 주석·README 수정 및 최종 검증: 위 지정 구간만 갱신하고 go vet·diff·release contract를 확인한다. 테스트 상태·실DB 실행 여부를 구현 노트에 남긴다.

대안 비교 및 추정 근거
- 선택: 호출별 스키마. 기존 database migrationPool에서 이미 입증한 패턴이고 제품 코드 변경이 없다. 핵심 가정은 API SQL이 명시적 public 테이블이나 다른 fixture의 seed에 의존하지 않는다는 것; 검색에서는 해당 PG 테스트들의 public 참조가 없었으며 최종 증명은 전체 실DB 회귀다.
- 현상 유지+고유 tag/복원: 변경 비용은 없지만 전역 setting·audit 간섭이 남는다. 최근 jsonb 테스트가 행 수 델타로 우회한 문제도 있어 후순위다.
- 테스트별 DB 생성: 격리는 더 강하나 CREATEDB 권한 및 생성 비용이 추가되고 기존 README/계정 전제가 커진다.
- 트랜잭션 롤백: handler가 자체 pool 연결·트랜잭션을 쓰므로 한 외부 tx로 감쌀 수 없고 제품 인터페이스 변경으로 번진다.
- Bottom-up 추정: 준비·Red 5분 + helper 10분 + 격리/정리 회귀 12분 + 전체 검증·문서 8분 = 기본 35분. 알려진 불확실성(정리 순서·seed 회귀)에 contingency 5~10분을 별도 배정하여 40~45분 예상, 신뢰도 중간(통계적 확률 아님). management reserve는 이번 자율 회차에 배정하지 않으며 새 범위가 발견되면 차선으로 축소한다. 비교한 유사 작업은 09-25 PG migration fixture와 09-26/27 실제 router 회귀지만 작업 시간 기록이 없어 유사 추정으로 숫자를 검증할 수는 없다.

정찰 증거
- 기준 main@7d21df9 / VERSION 0.7.23. 코드·커밋 변경 없음. 최근 git log -30 확인.
- Skill 도구는 노출되지 않았으나 로컬 headcount의 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration` SKILL.md 및 pmo references/sources.md를 직접 읽고 적용했다. 외부 비용 모델을 인용하지 않고 저장소 기반 작업 추정을 사용했다.
- `go test ./internal/api/... ./internal/database/... -count=1` 통과(DSN 미설정 PG skip), `bash scripts/check-release-contract.sh` 통과. 실제 PG baseline 결과는 아래에 추가한다.
- 실제 PG baseline: postgres:17-alpine 일회용 컨테이너, 전용 `igame_test_extensions`에 pgcrypto 준비 후 위 `make test-db DSN=...25432...` 통과(API 9.887s, database 1.886s). 종료 후 `migrate_%` 스키마 0개, pgcrypto namespace 보존, `public.schema_migrations` 10행을 SQL로 확인했다. 이 테스트는 현재 코드 baseline이며 앞으로 작성할 격리 회귀의 통과 증거는 아니다. 정찰 컨테이너는 제거하므로 구현자는 README 절차의 포트를 25432로 맞춰 다시 준비한다.
- 이전 프로필 정정: 현 Makefile 및 `make help`에는 `audit-release`가 없다. 과거 성공 기록과 현재 트리가 다르다는 사실만 확인했으며 원인·원격 PR 상태는 미확인; 이 과제를 재선택하지 않는다.

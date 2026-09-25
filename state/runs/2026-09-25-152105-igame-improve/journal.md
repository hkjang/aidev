# 회차 노트 2026-09-25-152105-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:21] base pinned — main@97368cd
- [러너 15:21] autonomy release — 

## 정찰 노트
- Migrate 취소 경로를 골랐다: 기존 PG 테스트 3개가 모두 서버 예외로만 실패를 만들어 취소 경로가 비어 있고(database_pg_test.go:154/173/208 확인), 테스트 전용이라 릴리즈·보호 경로를 건드리지 않는다. 감사 CSV 인젝션은 이미 방어돼 있었고(csvSafeCell), 랭킹 기간 파서 중복은 "계약 다른 파서 통합 금지" 지침에 걸려 기각했다.
- serviceLocation 이 설정 읽기 오류를 삼키는 건 실재하지만(extended.go:16, 호출처 6곳) 시그니처 변경이 6곳에 번져 45분·저위험 기준에 안 맞아 보류로 남겼다.
- 추측으로 적은 것: 어드바이저리 락 폴링으로 010 트랜잭션 중 취소를 잡는 방법은 **실행해 보지 않았다**. 취소된 ctx 로 롤백해도 커넥션 폐기 덕분에 계약이 지켜질 것이라는 예상도 미확인 — 구현자가 Red 를 먼저 확인할 것.
- 조심할 것: migrations/*.sql 은 checksum 불변이라 절대 수정 금지, 실패 주입은 테스트 스키마 트리거로만. IGAME_TEST_DSN 준비 시 pgcrypto 를 public 아닌 전용 스키마에 미리 넣어야 fixture 가 Fatal 하지 않는다(postgres:17-alpine 로컬 이미지 존재 확인).
- 소스는 수용 기준이 실제로 깨질 때만 고친다. 안 깨지면 테스트만 남기는 것이 정답(09-21 선례).
- [러너 15:25] scout done — Migrate 의 context 취소 경로가 부분 적용을 남기지 않고 재시도 가능한지 실제 PostgreSQL 로 검증 (가치 3 / 위�

## 구현 노트
- 무엇/왜: `internal/database/database_pg_test.go` 에 `TestMigrateCancelledMidMigrationLeavesNothingAndRetries` 하나를 더했다. 010 을 어드바이저리 락으로 붙잡아 트랜잭션이 열린 상태에서 ctx 를 취소하고, 살아 있는 부모 ctx 로 (1) 에러가 `context.Canceled` 를 감싸며 파일명을 담는지 (2) 010 의 이력 행·`oidc_flows.silent` 가 없고 앞선 이력·seed 가 불변인지 (3) **같은 pool** 로 재시도하면 완주하는지를 확인한다. 세 기준이 현재 소스에서 그대로 성립해 `database.go` 는 한 줄도 고치지 않았다(과제서 지시).
- 확신 없는 곳: `errors.Is(err, context.Canceled)` 는 pgx v5.10 의 기본 `DeadlineContextWatcherHandler`(취소 시 즉시 `SetDeadline` → net timeout → `normalizeTimeoutError` 가 `context.Canceled` 로 정규화)에 의존한다. pgx 를 올리면서 `BuildContextWatcherHandler` 기본값이 CancelRequest 방식으로 바뀌면 에러가 57014 `PgError` 가 되어 이 단언이 깨질 수 있다 — 그때는 테스트가 틀린 것이지 배선이 깨진 것이 아니다.
- 확신 없는 곳: 두 폴링 루프(대기자 등장 / 취소된 백엔드 종료)는 각각 1분 상한에 10ms 간격이다. 로컬에서 전체 0.5초 안에 끝났고 `-race -count=3` 도 안정적이었지만 매우 느린 CI 에서의 여유는 측정하지 못했다. 실패하면 hang 이 아니라 명시적 `t.Fatalf` 로 끝난다.
- 일부러 안 한 것: `_ = tx.Rollback(ctx)` 리팩터(`WithoutCancel` 포함) — 수용 기준이 깨지지 않아 필요 없었다. 기존 테스트 3개와 `migrations/*.sql` 은 손대지 않았고, 새 공용 헬퍼도 만들지 않았다(`assertSilentColumn` 은 기존 테스트와 같은 모양의 지역 클로저).
- 다음 역할이 조심할 것: 이 테스트는 `IGAME_TEST_DSN` 이 있어야 돈다(없으면 skip). CI 의 `make test` 는 여전히 전부 skip 하므로 회귀는 로컬 도커로만 보인다. pgcrypto 를 public 아닌 전용 확장 스키마에 미리 설치해야 `migrationPool` 이 Fatal 하지 않는다(README 절차).
- 다음 역할이 조심할 것: 어드바이저리 락 키는 DB 전역이라 매 실행 난수(`rand.Uint32()+1`)를 쓴다. 같은 DB 에 이 패키지를 **동시에** 두 벌 돌려도 충돌하지 않지만, 테스트가 중간에 강제 종료되면 락 보유 커넥션이 끊길 때까지 그 키가 남는다(세션 락이라 연결 종료 시 자동 해제).
- 변이로 확인함: DDL 을 `pool.Exec` 로 트랜잭션 밖에 빼면 `silent column exists=true, want false` 로 실패하고, `_ = tx.Rollback(ctx)` 를 지우면 커넥션이 pool 로 안 돌아와 신규·기존 테스트가 `pool.Close` 에서 멈춘다. 둘 다 원본 복구 후 재검증했다.
- [러너 15:38] brief accepted — 채택 — `database.go:128-131` 의 `_ = tx.Rollback(ctx)` 와 기존 PG 테스트 3개가 취소 경로를 전혀 덮지 않는다는 근거가 지금 코드�
- [러너 15:39] verify passed — 검증 4개 통과 (policy)

## 비평 노트
- 확인함: postgres:17-alpine 을 새로 띄워 4개 Migrate PG 테스트 통과, 새 테스트 단독 `-race -count=15` 안정, DSN 없으면 skip, gofmt/go vet 무소음. 변이(`tx.Exec`→`pool.Exec`)로 원자성 단언이 실제로 잡는 것까지 /tmp 사본에서 재현했고 워크트리는 clean 유지.
- 못 본 것: internal/api 패키지 PG 테스트(기본 스키마 공유라 이번 검토용 DB 에 안 돌림), 느린 CI 에서의 폴링 여유, `make lint`/web 빌드.
- 승인이어도 남는 우려 1: `errors.Is(err, context.Canceled)` 는 pgx v5.10 의 deadline 기반 취소 정규화에 묶여 있다. pgx 범프 때 이 단언만 깨질 수 있고, 그때 고칠 것은 배선이 아니라 테스트다.
- 승인이어도 남는 우려 2: `key := int64(rand.Uint32())+1` 이 2^32 이면 classid 가 1 이 되어 폴링 필터가 안 맞고 60초 뒤 실패한다(2^-32). 그런 실패가 보이면 원인은 database_pg_test.go:299 다.
- 릴리즈 노트: 테스트 전용 + README 1줄, 런타임 동작 변화 없음. CI 는 여전히 이 회귀를 못 본다 — 보증은 로컬 도커에서만 성립한다.
- [러너 15:43] review approved — 리뷰 승인 (risk=low)
- [러너 15:43] pr created — https://github.com/hkjang/igame/pull/27
- [러너 15:47] ci passed — 검사 1개 모두 success
- [러너 15:47] merge done — f2764a6

- 과제: DB 차단기 백오프가 동시 요청에서 누적되지 않는 것 고치기 (가치 4 / 위험 2 / 작업량 S)

- 왜: `server/db.ts:baseTx` 는 호출 **시작 시점**에 `const breaker = breakers.get(key)` 로 읽은 값을 그대로 `trip(key, breaker)` 에 넘긴다. 한 isolate 가 요청을 동시에 여러 개 처리하면(2026-10-14 사고가 정확히 그 상황) 모두 `previous=undefined` 를 본 채로 실패해 각자 `strikes=1` 로 트립하고 서로의 항목을 덮어쓴다. 그래서 대기가 10초 → 20초 → … → 120초 로 늘어나는 백오프가 작동하지 않고 계속 10초에 머문다. 6시간짜리 장애에서 10초마다 프로브가 열려 그 순간의 요청은 매번 5초(연결 3회) 를 기다리고 죽은 DB 를 계속 두드린다. 고치면 장애가 길어질수록 재시도 간격이 실제로 벌어진다.

- 수용 기준:
  1) 같은 키에 대해 **동시에** 실패한 호출들이 strikes 를 누적한다: 끊는 DB 를 향해 `baseTx` 를 동시에 2개 돌리면 `databaseBreaker(db).strikes === 2` 이고 `retryAfterSeconds` 가 20초(두 번째 단계)다. 지금 코드는 strikes=1 / 10초로 실패한다.
  2) 차단기가 닫힌 뒤(DB 가 답한 뒤) 그 전에 시작해 뒤늦게 실패한 호출이 차단기를 다시 열지 않는다. 즉 `mode='pass'` 로 DB 가 돌아온 다음 성공한 호출이 차단기를 닫았으면, 그보다 먼저 시작한 실패 호출의 트립은 무시되고 `databaseBreaker(db).open === false` 로 남는다. (거짓 "잠시 점검 중입니다" 안내가 사용자에게 뜨는 것을 막는 부분이다.)
  3) 기존 순차 동작은 그대로다: `tests/db-breaker.ts` 의 "기다림이 끝나면 한 번만 시도하고, 또 실패하면 두 배로 연다"(10초→20초, strikes 2)와 "DB 가 돌아오면 다음 시도가 차단기를 닫는다"(strikes 0, open false)가 계속 통과한다.
  4) 테스트는 "동시"를 진짜 동시 실행으로 증명해야 한다 — `Promise.allSettled([select1(a), select1(b)])` 처럼 두 호출이 await 로 겹치게 돌려야 하고, `trip()` 을 직접 불러 보는 식의 단위 호출은 배선 결함(baseTx 가 캡처한 값을 넘기는 것)을 증명하지 못한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `server/db.ts:trip` — 인자로 받은 `previous` 를 쓰지 말고 트립하는 순간 `breakers.get(key)` 를 다시 읽어 strikes 를 더한다. `baseTx` 의 `breaker` 지역변수는 열림 검사(`now < breaker.openUntil`)에만 쓰고 `trip` 호출에서 떼어낸다.
  - `server/db.ts` (AC2) — 키별로 "마지막으로 닫힌 시각"을 둔다. 제안: `const closedAt = new Map<string, number>()` 를 두고 `breakers.delete(key)` 하는 두 자리(성공, DB 가 답한 오류)에서 `closedAt.set(key, Date.now())` 를 함께 하고, `trip(key, startedAt)` 가 `(closedAt.get(key) ?? 0) > startedAt` 이면 아무것도 하지 않고 돌아간다. `startedAt` 은 `baseTx` 가 이미 가진 `const now = Date.now()` 를 쓰면 된다. `resetDatabaseBreakers()` 는 `closedAt` 도 비워야 한다(테스트의 `afterEach` 가 이것을 쓴다).
  - `tests/db-breaker.ts` — AC1·AC2 테스트를 2개 추가. 기존 헬퍼를 그대로 쓸 것: `connect(env(DROPPING))`, `select1()`, `failure()`, `databaseBreaker()`, `close()`, `mode='pass'`, `mock.method(Date,'now', …)`. 파일 상단 주석의 설명(차단기 계약)도 동시 요청을 포함하게 한 줄 고치면 좋다.

- 검증 명령 (이 저장소에서 실제로 도는 것. 정찰 세션에서는 승인 제약으로 **실행하지 못했음 — 미확인**):
  - 차단기 묶음만: `node scripts/test.mjs --build-only` 그다음 `node --test artifacts/db-breaker.mjs`
    (DATABASE_URL 미설정이면 테스트 파일이 `postgres://postgres:aptlocal@127.0.0.1:55462/postgres` 로 떨어진다. 테스트는 호스트가 127.0.0.1 이고 포트가 **55462** 가 아니면 즉시 throw 한다 — 로컬 테스트 DB 가 떠 있어야 한다.)
  - 전체: `npm test` (= `node scripts/test.mjs`, 15개 묶음 순차 `--test-concurrency=1`), 그리고 `npm run typecheck`
  - CI 는 `.github/workflows/ci.yml` 에서 postgres:17-alpine 을 55462 로 띄우고 `db:migrate → npm test → test:runtime → build → test:assets → npm audit → test:e2e` 를 돈다. 이 과제는 e2e 와 무관하다.

- 위험과 피할 것:
  - `server/vendor/postgres/**` 는 손대지 말 것. 2026-10-14 에 가져온 패치 사본이고 계약이 `PATCHED.md` 에 적혀 있다(PATCH 5: 세 번 연속 끊기면 CONNECTION_CLOSED — 테스트의 `dropped === 3` 단정이 이것에 걸려 있다).
  - `isDatabaseUnreachable` 의 정규식을 넓히지 말 것. 이번 과제와 무관하고, 넓히면 문장 오류(23505·42501 등)가 503 으로 새어 "아니다" 테스트가 깨진다.
  - `BREAKER_FIRST_WAIT_MS`(10s) · `BREAKER_MAX_WAIT_MS`(120s) · `connect_timeout: 5` 숫자를 바꾸지 말 것. 기존 테스트가 `retryAfterSeconds <= 10`, `=== 20`, `took >= 4_500 && < 6_500` 로 못 박아 두었고 docs/operations.md 의 절차와 묶여 있다.
  - 재시도 로직을 새로 넣지 말 것 — 주석이 명시한 설계는 "apt 는 재시도하지 않는다. 기다림이 끝나면 호출당 한 번 시도"다. 그 계약을 바꾸는 것이 아니라 strikes 누적만 고치는 과제다.
  - 보호 경로 회피: `server/auth.ts`, `server/social-auth.ts`, `supabase/` 마이그레이션, `.github/workflows/` 는 건드리지 않는다. `baseTx` 안의 `set_config(…)` 줄(role·apt.actor·apt.tenant·worker 플래그)은 RLS 계약이므로 한 글자도 바꾸지 말 것.
  - `Date.now()` 를 쓰는 자리만 쓸 것. 기존 테스트가 `Date.now` 를 모킹하고 경과 시간은 `performance.now()` 로 재는 이유가 WSL 벽시계가 초 단위로 튄 적이 있어서다 — 새 코드에서 `performance.now()` 로 차단기 시각을 재면 모킹이 통하지 않아 테스트가 불안정해진다.

- 차선 후보: **차단기가 열렸다 닫히는 순간의 thundering herd 를 단일 프로브로 좁히기** — `openUntil` 이 지나면 그 isolate 의 동시 호출이 *모두* 연결을 시도한다(주석은 "호출당 한 번 시도"를 의도하지만 동시성에서는 N번이 된다). 키별로 진행 중인 프로브 Promise 를 하나 들고 나머지는 그 결과를 공유하게 하면 좁혀진다. 같은 파일·같은 테스트 파일로 끝나지만 Promise 공유가 들어가 위험 3 / 작업량 M. 1순위가 이미 고쳐져 있거나 성립하지 않을 때 이것을 할 것.

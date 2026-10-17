# 과제서 — 2026-10-18 (수정 과제, 우선 배정)

- 과제: CI 가 PR #4·#5 에서 같은 이유로 죽는 자리를 `npm run check` 안에서 확정하고, db-breaker 테스트의 벽시계(wall-clock) 단정을 러너 속도와 무관한 증거로 바꾸기 (가치 4 / 위험 2 / 작업량 M)

- 왜: 마지막 녹색 CI 는 `739ea33`(10-08)이고, 그 뒤 `37ed04f → f3f2836 → 601659d`(v0.1.5~0.1.6, 차단기·빨리 실패)는 **PR 없이 main 으로 바로 올라갔다**. PR #4(`b55ab2d`)와 PR #5(`cb269c5`)는 둘 다 `601659d` 의 자식이므로, 두 PR 이 "같은 이유로" 죽는다는 것은 원인이 **PR 의 변경이 아니라 601659d 자체**에 있다는 뜻이다. 그 창(`739ea33..601659d`, 23개 파일) 안에서 `npm run check` 에 새로 들어온 것은 `packages/core/src/__tests__/db-breaker.test.ts`(207줄, 신규) 하나이고, 이 파일은 실제 TCP 서버를 띄워 **밀리초 벽시계로 단정**한다 — 공유 vCPU 러너에서 반복적으로 깨지는 전형적인 모양이다. 고치면 PR #4·#5 와 이후 모든 회차의 CI 가 다시 신호로 쓸 수 있게 된다.

## 먼저 할 일 (증거 확보 — 분기 판단)
정찰 세션에서는 `gh`(승인 거부)와 `npm ci`(승인 거부, 네트워크) 둘 다 실행할 수 없었다. **그래서 아래 인과는 정적 분석까지만이고 CI 로그로 확인하지 못했다 — 미확인.** 구현자는 먼저 둘 중 되는 쪽으로 실패 단계를 확정하라.

1. `gh` 가 인증돼 있으면: `gh run list --branch auto/2026-10-17-2308 --limit 5` → `gh run view <id> --log-failed | tail -120`. **실패한 step 이름을 기록**하라.
2. `gh` 를 못 쓰면 로컬 재현: `npm ci --no-audit --fund=false` 후 `npm run check`.
   - 한가한 기계에서는 통과할 수 있다. 러너를 모사해 부하를 걸고 돌려라(예: `for i in 1 2 3 4; do (while :; do :; done) & done` 로 코어를 채운 뒤 `npm run test -- packages/core/src/__tests__/db-breaker.test.ts`, 끝나면 `kill %1 %2 %3 %4`). 아래 단정들이 깨지는지 본다.

- **분기 A — 실패 단계가 `Lint, types, unit tests`(= `npm run check`) 이고 db-breaker 테스트다**: 아래 "건드릴 파일" 대로 고친다.
- **분기 B — 실패 단계가 다른 것(e2e / screenshots / Start the server / Migrate)**: 그 단계의 실제 로그를 근거로 그 자리를 고친다. 아래 "내가 배제한 것" 을 다시 조사하느라 시간을 쓰지 말 것. 분기 B 로 가면 과제서의 수용 기준 2~3번은 해당 단계의 것으로 바꿔 적고, **`check.yml` 은 건드리지 말 것**(게이트 약화 금지 — 운영자 명시 금지 사항).
- **분기 C — 저장소 쪽 원인이 전혀 없다고 판단되면**: 10-17 회차가 이미 분기 B(코드 미수정)로 끝냈다. 같은 결론을 두 번 올리지 말고, 이번에는 분기 A 의 벽시계 단정 정리까지는 끝내라 — 그것만으로도 CI 가 다시 재현 가능해진다.

## 내가 이번 세션에 실제로 확인해 배제한 것 (다시 조사하지 말 것)
- 워크플로 파일은 `.github/workflows/check.yml` **하나뿐**이다. 별도의 release 워크플로는 없다(`ls .github/workflows` 로 확인). 회차 지시문의 "릴리즈 워크플로" 는 이 check 를 말한다.
- `package-lock.json` 은 `package.json` 과 동기 상태다. `739ea33..601659d` 의 lock 변경은 **버전 문자열 0.1.4→0.1.6 다섯 군데뿐**이고 의존성 추가·삭제가 없다 → `npm ci` 의 "out of sync" 3초 실패는 **배제**.
- `.env.example` 변경도 `INSIGHT_VERSION=0.1.4→0.1.6` 한 줄뿐이다 → 새 필수 환경변수를 `scripts/ci-env.mjs` 가 못 만들어 서버가 못 뜨는 경로는 **배제**.
- `eslint.config.js` 의 ignores 는 `dist/node_modules/.wrangler/docs/packages/core/vendor` 뿐이라 `eslint .` 는 `functions/`·`workers/`·`scripts/` 까지 린트한다. 다만 typescript-eslint recommended 가 TS 파일의 `no-undef` 를 끄므로, `functions/[[path]].ts`(+176줄)의 CF 바인딩 전역은 린트 에러가 되지 않는다. 또 `functions/` 는 어느 `typecheck` 스크립트에도 들어 있지 않다(`npm run typecheck` = core/server/web 3개 workspace) → `functions/` 타입·전역 때문이라는 가설은 **약함**.
- 10-17 회차가 확인한 것: `git diff 739ea33 601659d -- .github/` 가 빈 출력(워크플로 설정 변경 아님), 같은 계정 다른 저장소 잡 정상(과금·러너 전면 장애 아님), `annotations_count: 2` 는 성공 회차에도 2 라 실패 신호가 아님. 전부 그대로 유효하다.

## 수용 기준
1. 실패한 CI step 이름이 (로그 또는 로컬 재현으로) 과제 기록에 **한 줄로 확정**되어 있다. 추정이면 "미확인" 이라고 적는다.
2. `packages/core/src/__tests__/db-breaker.test.ts` 에서 **러너 속도만 재는 단정이 사라지고**, 같은 의도("빨리 실패한다", "차단기가 열리면 시도조차 않는다")가 속도와 무관한 증거로 바뀌어 있다 — 즉 시도 횟수(`dropped`)·`databaseBreaker()` 상태(`open`/`strikes`/`retryAfterSeconds`)·`AppError` 의 `code`/`status`/`retryAfter` 로 단정한다.
3. 남겨 두는 상한은 **드라이버 예산에 대한 상한**이어야 한다(예: 재시도 총예산 2.5초 + 넉넉한 여유). 500ms 여유 같은 것을 남기지 말 것. 각 상한 옆에 "무엇의 상한인가" 를 한 줄 주석으로 적는다.
4. 부하를 걸고 돌려도(위 재현 절차) `npm run test -- packages/core/src/__tests__/db-breaker.test.ts` 가 통과하고, 수정 전에는 적어도 하나가 실패하는 것을 **같은 명령으로 두 상태 다 보여 준다**. 못 재현하면 "재현 실패" 로 적고 변경 근거를 코드 독해로 밝힌다.
5. `npm run check` 와 `npm run build` 가 통과한다. `.github/workflows/check.yml` 은 한 줄도 바뀌지 않았다(`git diff --stat` 로 확인).

## 건드릴 파일 (프로덕션 0~1개)
- `packages/core/src/__tests__/db-breaker.test.ts` — 벽시계 단정 4곳. 각각 내가 읽은 줄과 이유:
  - **:127 `expect(Date.now() - t).toBeLessThan(50)`** — 가장 위험하고, **증거로서 쓸모가 없다**. `db.ts:250-252` 에서 차단기가 열려 있으면 `retrying` 이 I/O 없이 그 자리에서 `dbUnavailable(...)` 을 던진다. 즉 이 50ms 는 순수 CPU 시간이고, 바로 위 `:126 expect(dropped).toBe(tried)` 가 이미 "연결을 시도하지 않았다" 를 증명한다. **삭제하거나(권장) 의미 있는 상한으로 바꾼다** — 삭제해도 잃는 증거가 없다는 것이 근거이고, 이것은 게이트 약화가 아니다.
  - **:162 `toBeLessThan(3_000)`** (`SILENT`, `connectTimeout: 1`) — 테스트 이름(`:131`)이 말하는 재시도 총예산이 2.5초인데 상한이 3.0초다. 여유 500ms. 두 번째로 위험.
  - **:135 `toBeLessThan(4_000)`**, **:114 `toBeLessThan(5_000)`** — 같은 2.5초 예산에 대한 상한. 여유가 더 크지만 같은 종류다. 상한을 유지하려면 "2.5초 예산이 지켜지는가" 를 재는 것이라고 주석으로 못 박을 것.
- (필요할 때만) `packages/core/src/db.ts` — 시계 이음새가 필요하면. **단, `:379 BREAKER_FIRST_WAIT_MS=10_000`·`:380 BREAKER_MAX_WAIT_MS=120_000`·`:255 maxAttempts = breaker ? 1 : 4`·`:386` 의 지수 증가는 바꾸지 말 것.** 테스트는 이미 `:144` 에서 `vi.spyOn(Date,'now')` 로 쿨다운을 건너뛰므로 새 이음새 없이 끝나는 쪽이 맞다. 프로덕션 파일을 건드리지 않고 끝나면 그게 최선이다.
- 건드릴 파일은 여기까지. `errors.ts`·`http/pipeline.ts`·`http/respond.ts`·`functions/[[path]].ts`·`wrangler.toml` 은 읽기만 할 것.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
npm ci --no-audit --fund=false
npm run test -- packages/core/src/__tests__/db-breaker.test.ts   # 수정 전/후 둘 다, 부하 걸고도 한 번
npm run check        # eslint . + core/server/web typecheck + vitest run
npm run build
git diff --stat      # check.yml 무변경 확인
```
- `node scripts/e2e.mjs` 는 실 Postgres 16·정식 서버·가짜 OAuth `:18999` 가 필요하다. 분기 B 로 가면 필요하지만, 분기 A 만으로 끝나면 돌리지 않아도 된다(돌리지 않았으면 그렇게 적을 것). `scripts/ci-env.mjs`·`scripts/dev-reset.sh` 는 기존 개발 설정을 덮어쓰므로 **기존 환경에서 실행 금지**.

## 위험과 피할 것
- **`check.yml` 을 느슨하게 만들어 통과시키는 것은 금지**(운영자 명시). 단계 삭제·`continue-on-error`·`timeout` 늘리기·테스트 제외 전부 해당. 타임아웃 상한을 "그냥 크게" 올리는 것도 같은 범주로 보고, 왜 그 숫자인지 적지 못하면 하지 말 것.
- 차단기 설정값(10초→120초, 지수 증가, 주소별 키)은 근거 없이 바꾸지 않는다. 포화 오류를 `isDatabaseUnreachable` 에서 빼는 류의 제안은 과거에 기각됐다.
- `vitest.config.ts` 의 `testTimeout: 20_000` 과 `include` 는 건드리지 말 것(전 저장소 영향).
- 이 테스트 파일은 모듈 전역 `dropped` 카운터와 `beforeAll` 의 소켓 서버 2개를 공유하므로 **테스트 순서에 의존**한다. 단정을 고칠 때 `dropped = 0` 초기화 위치를 옮기지 말 것(`:105`, `:143`).
- 10-17 회차가 이미 배제한 것(워크플로 diff·주석 개수·계정 과금·타 저장소)을 다시 파지 말 것. 같은 결론을 세 번째로 올리는 것이 가장 나쁜 결과다.
- 로컬 `main` 은 `5fffd04` 로 낡았고 `origin/main` 이 `601659d` 다. 비교·리베이스는 `origin/main` 기준으로 할 것.

## 차선 후보
**`respond.ts:noContent()` 의 204 계약을 한쪽으로 정리하기** (가치 2 / 위험 1 / S) — `noContent()` 는 정의만 있고 호출처가 0 이며, 10-17 회차에 클라이언트 쪽 204 캐시 무효화는 이미 고쳐졌다. 서버에서 쓰지 않을 것이면 지우고, 쓸 것이면 한 라우트에 적용한다(라우트 응답 형태를 바꾸면 e2e 기대값이 걸리므로 "지우는 쪽" 이 S). 분기 A·B 가 모두 성립하지 않을 때만 고를 것.

- 과제: PR #3 의 CI 실패를 고쳐 한도 429 의 `Retry-After` 수정을 다시 출하한다 (가치 4 / 위험 2 / 작업량 S)
- 왜: 지난 회차가 `quota.ts:retryAfterSeconds` 를 고쳐(429 의 `retry-after` 가 항상 정확히 32400초 과대였다) 로컬 e2e 164 통과까지 확인했는데, GitHub Actions `check` 워크플로가 **같은 이유로 두 번** 실패해 PR #3 이 열린 채 막혔다. 프로덕션 수정 자체는 타당하므로(아래 "내가 확인한 것"), CI 에서만 깨지는 검증 쪽 원인을 찾아 고쳐야 그 수정이 실제로 나간다.
- **먼저 할 일 (추측 금지)**: 이 세션은 `gh`·WebFetch 권한이 없어 **CI 로그를 못 읽었다**. 구현자는 손대기 전에 실패 로그부터 읽을 것:
  - `gh run list --branch auto/2026-10-14-1028 -L 10`
  - `gh run view <id> --log-failed | tail -n 200`  ← 어느 **step** 이 죽었는지(`Lint, types, unit tests` / `Build` / `End-to-end`)와, e2e 면 `FAIL` 한 줄인지 **예외로 중단**인지를 먼저 확정하라. 그 다음 아래 순위를 쓴다.
  - `check.yml` 마지막 단계 `Server log`(`tail -n 300 server.log`)도 같이 읽을 것 — 서버 쪽 예외가 거기 남는다.
- 수용 기준:
  1) PR #3 의 두 커밋 내용(`quota.ts` 수정 + `rules.test.ts` 단위 3검사 + `e2e.mjs` 새 검사)이 이번 브랜치에 들어가고, 원인이 **코드로 지목**된 뒤 고쳐져 있다 (로그의 어느 줄이 왜 그렇게 됐는지 과제 요약에 적을 것).
  2) `check.yml` 과 e2e 러너를 **느슨하게 만들지 않았다** — 검사 삭제·조건 약화·`continue-on-error`·`timeout` 늘리기는 금지. 고치는 대상은 검사가 의존하는 전제(고정 데이터·시각 계산·요청 방식)여야 한다.
  3) `TZ=UTC` 로도 `npm run check` 와 `node scripts/e2e.mjs` 가 전부 통과하고, 수정을 되돌려 넣으면 다시 실패하는 인과를 명령 출력으로 확인했다.
- 의심 순위 (코드를 읽어 확인한 근거만):
  1. **429 가 요청 본문을 다 읽기 전에 돌아간다.** `routes/profile.ts:386-390` 의 `consume(...)` 이 `ctx.rawBody()`(394행)보다 **먼저** 돈다. 즉 서버는 업로드 바디를 한 바이트도 읽지 않고 429 를 쓴다. 로컬 루프백은 소켓 버퍼가 바디를 다 삼켜 주지만, CI 에서는 같은 상황이 클라이언트 쪽 `ECONNRESET`/`EPIPE` 로 나타날 수 있다 — 그러면 `overDaily.status === 429` 가 아니거나 `client.upload` 자체가 던져 e2e 가 중단된다. 로그가 "1 failed" 가 아니라 **예외 중단**이면 1순위가 범인이다. 해결은 검사를 느슨하게 하는 것이 아니라, 429 를 만드는 요청을 **본문이 작은 경로**로 바꾸거나(이미 240x300 으로 작게 잡혀 있다) `upload` 헬퍼가 연결 오류를 그대로 터뜨리지 않게 하는 쪽.
  2. **`window_start` 가 NULL 로 들어간다.** `scripts/e2e.mjs`(새 블록 첫 질의)가 `your_priv.quota_counters` 에서 `jiwoo` 의 `safety.report`/`day` 행을 읽어 그 `window_start` 를 재사용하고, 없으면 `dayWindow?.window_start ?? null` 로 **NULL** 을 넣는다. 그런데 `db/migrations/0001_schema.sql:313` 은 `window_start timestamptz NOT NULL` 이고 PK 의 일부다 → 행이 없으면 `not-null constraint` 위반으로 e2e 가 **예외 중단**한다. 바로 위 신고 검사(`신고 + 차단`)가 200 이면 행은 있어야 정상이지만, `consume` 은 거절 시 `release` 로 되돌리므로(quota.ts) CI 의 설정값에 따라 행이 다를 수 있다. **고치는 방향: NULL 로 흘려보내지 말고, 행이 없으면 명시적으로 실패하거나 `windowStart` 와 같은 규칙으로 직접 만들 것** (`kstDateKey(at)` + `T00:00:00.000Z`).
  3. 초 경계 타이밍 — **이미 커밋 9865101 이 그것을 고치려 한 시도였고 CI 가 또 실패했다.** 따라서 타이밍 계열은 범인이 아닐 가능성이 높다. 같은 방향으로 다시 손대지 말 것(운영자 반복 지시: 반려된 접근을 같은 방식으로 다시 올리지 않는다).
  4. 분 단위 한도가 먼저 걸려 `retry-after` 가 60 이 되는 경우: `consume` 은 `minute(max 10)` → `day(max 40)` 순으로 판정한다. e2e 는 `day` 카운터만 40 으로 올리므로 CI 가 느려 분 창이 달라지면 오히려 안전하다 — 다만 로그의 `retry_after=` 값이 60 이면 이쪽이다.
- 내가 확인한 것 (추측 아님): `retryAfterSeconds` 의 day/month/minute 산식과 `rules.test.ts` 의 기대값 세 쌍을 손으로 계산해 전부 맞았고, `util/time.ts` 의 `kstParts`/`kstDayStart`/`kstDateKey` 는 **고정 오프셋 ±9h** 만 쓰므로 TZ 에 무관하다. 즉 CI 가 UTC 라서 단위 테스트가 깨지는 일은 없다 → 실패는 `End-to-end` 단계일 가능성이 높다. (로그 미확인이므로 **단정은 아니다**.)
- 건드릴 파일 (프로덕션 1개 + 검증 2개, 이 범위를 넘기지 말 것):
  - `packages/core/src/quota.ts:retryAfterSeconds` — PR #3 의 수정을 그대로 가져온다. 키를 만드는 `windowStart()` 는 `your_priv.quota_counters` 의 **PK** 라 절대 옮기지 말 것(옮기면 운영 DB 의 모든 카운터가 한 번에 끊긴다). 회원에게 주는 답만 `kstDayStart`/`kstParts` 로 계산.
  - `packages/core/src/__tests__/rules.test.ts` — `describe('quota windows')` 세 검사
  - `scripts/e2e.mjs` — '차단·신고' 섹션 끝(`차단 목록에 이름이 남는다` 바로 뒤)의 `하루 한도 429의 Retry-After 는 다음 한국 자정까지` 블록
  - 가져오기: `git cherry-pick 2bbf98c 9865101` (origin/auto/2026-10-14-1028) 또는 `git diff main origin/auto/2026-10-14-1028 | git apply`
- 검증 명령 (CI `check.yml` 단계 순서 그대로):
  1. `npm ci --no-audit --fund=false` (설치돼 있으면 생략)
  2. `npm run check`  — lint + typecheck(core→server→web) + vitest
  3. `npm run build`
  4. `bash scripts/dev-reset.sh`  — Docker `postgres:16-alpine` 를 127.0.0.1:55440 에, 마이그레이션, `.env.dev` 로 :8788 서버 기동 (느리다)
  5. `node scripts/e2e.mjs`  — 164검사가 **배선을 증명하는 유일한 수단**이다
  6. `TZ=UTC npm run check && TZ=UTC node scripts/e2e.mjs`  — CI 는 UTC 러너다
- 위험과 피할 것:
  - `scripts/e2e.mjs` 의 가짜 서버 포트 18998·18997 하드코딩 — **세 회차 연속** 같은 기계의 다른 프로젝트 고아 프로세스(`node scripts/fake-toss.mjs 18998`, PPID 1)가 그 포트를 쥐고 e2e 를 막았다. 막히면 `ss -ltnp | grep 1899` 로 확인하고 임시로 18996/18995 + `.env.dev` 의 `YOUR_YEOPJARI_URL` 을 맞춰 돌린 뒤 **되돌릴 것**. 이번 과제가 아니다.
  - `.github/workflows/check.yml` 은 보호 경로다. 단계를 지우거나 조건을 약화시키는 수정 금지.
  - `db/migrations/**`(전진 전용) · `packages/core/src/auth/**` · `packages/core/src/vendor/postgres`(패치 사본) 는 건드리지 말 것.
  - e2e 는 문자열 모양에 의존하는 검사가 있다(`signature LIKE 'CLIENT /chat/:id%'`) — 서명 문자열을 건드리면 운영 DB 의 알림 억제 상태가 한꺼번에 풀린다.
  - 로컬에서 통과한다는 사실만으로 끝내지 말 것 — 이 과제의 실패가 바로 그 상황이다. 수정은 PR 로 올려 **CI 가 초록인 것까지** 확인해야 끝난다.
- 차선 후보: `scripts/e2e.mjs` 의 가짜 서버 포트 18998·18997 을 환경변수(기본값 18998/18997 유지)로 빼고 `scripts/ci-env.mjs` 의 `YOUR_YEOPJARI_URL` 과 같은 값을 쓰게 한다 (2/1/S — 네 회차 연속 같은 벽. 1순위가 로그를 읽어 보니 성립하지 않을 때만).

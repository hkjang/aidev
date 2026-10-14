- 과제: PR #5 의 다섯 커밋을 main 위로 그대로 올려(이미 fast-forward 다) 로컬 전체 게이트로 재검증해 다시 출하한다 — CI 실패 원인은 저장소 밖 (가치 4 / 위험 1 / 작업량 S)
- 왜: PR #5(`origin/auto/2026-10-14-2128`, head `5547bbe`)의 한도 `Retry-After` 수정이 아직 main 에 없다 — main 은 그 사이 v0.1.5(`89f22e3`, DB 차단기)로 따로 전진했고 PR #5 는 열린 채 방치됐다. `5547bbe` 는 **지금 main 의 순수 fast-forward** 라 충돌 없이 올릴 수 있으므로(아래 확인) 이번 회차의 일은 그 가치를 되살리고 CI 가 못 하는 검증을 로컬에서 전부 재현하는 것이다.
- 수용 기준:
  1) 작업 브랜치 트리가 `5547bbe` 와 **바이트 동일**: `git diff 5547bbe HEAD` 가 빈 출력. (`git merge-base --is-ancestor main 5547bbe` 가 이미 통과하므로 cherry-pick 대신 다섯 커밋을 그대로 올리면 된다 — 지난 두 회차와 달리 충돌 가능성 0.)
  2) `.github/workflows/check.yml` 무변경: `git diff main --name-only` 에 `.github/` 0건. 워크플로를 느슨하게 해 통과시키는 것은 **금지**이고, 아래 근거대로 원인이 저장소 밖이므로 어떤 완화도 증상만 가린다.
  3) `check.yml` 과 **같은 순서**로 로컬 전체 게이트가 초록: `npm run check`(단위 53개 이상 — `rules.test.ts` 에 새로 들어오는 Retry-After 검사 2개 포함) → `npm run build` → `node scripts/ci-env.mjs` → 마이그레이션 + :8788 서버 → `node scripts/e2e.mjs` 가 **164 passed / 0 failed, exit 0**.
  4) 수정의 인과를 **되돌려 넣어** 증명: `quota.ts` 의 `retryAfterSeconds` 산술만 옛 코드로 복원하고(`export` 는 남길 것 — import 누락 에러는 결함 증명이 아니다) 단위 테스트가 `expected 36000 to be 3600` 로 2건 실패하고 e2e 가 `163 passed, 1 failed`(`하루 한도 429의 Retry-After 는 다음 한국 자정까지`) 로 떨어지는 것을 보인 뒤 되돌림을 풀어 164/0 복귀. 되돌린 코드는 **커밋하지 말 것**.
- 건드릴 파일 (프로덕션 1개):
  - `packages/core/src/quota.ts:retryAfterSeconds` — 다음 창 시작을 행의 **키**(`windowStart`, 한국 날짜를 UTC 자정에 박은 값)가 아니라 `kstParts`/`kstDayStart` 로 한국 벽시계에서 계산. 이 회차에 새로 쓸 코드가 아니라 `5547bbe` 가 이미 담고 있는 diff 다(직접 확인: `git diff main 5547bbe -- packages/core/src/quota.ts`, `-import { kstDateKey, minuteStart }` → `+import { KST_OFFSET_MINUTES, kstDateKey, kstDayStart, kstParts, minuteStart }`, `retryAfterSeconds` 가 `export` 로 바뀌고 month 는 `Date.UTC(p.y, p.m, 1) - KST_OFFSET_MINUTES * 60_000`, day 는 `kstDayStart(at) + 86_400_000`).
  - `packages/core/src/__tests__/rules.test.ts` — Retry-After 단위 검사 2개(+25행).
  - `scripts/e2e.mjs` — 가짜 서버 주소를 `YOUR_YEOPJARI_URL`/`E2E_PUSH_ORIGIN` 에서 읽고 포트 있는 루프백만 받는 가드(+82행). 기본값은 옛 하드코딩(`http://127.0.0.1:18998` / `:18997`)과 바이트 동일.
  - 합계 3개(`git diff --stat main 5547bbe` 로 확인: 123 insertions / 21 deletions).
- 검증 명령 (이 저장소에서 실제로 도는 것, `check.yml:38-54` 순서):
  - `npm ci --no-audit --fund=false`
  - `TZ=UTC npm run check` (lint + typecheck core→server→web + `vitest run`)
  - `TZ=UTC npm run build`
  - `node scripts/ci-env.mjs`
  - `TZ=UTC bash scripts/dev-reset.sh` (Docker `postgres:16-alpine` 을 127.0.0.1:55440 로 띄워 DB 초기화 + 마이그레이션 10개 + :8788 서버 기동)
  - `TZ=UTC node scripts/e2e.mjs` → 164 passed / 0 failed
  - 로컬 상자가 KST 이므로 **서버까지** `TZ=UTC` 로 띄워 UTC 러너를 흉내 낼 것.
- 위험과 피할 것:
  - **CI 초록은 이번 회차에도 달성 불가이고 원인은 이 저장소에 없다.** `check` 잡이 `2026-10-14T13:48:40Z → 13:48:43Z` **3초** 만에 `failure` 고 러너의 `ci-failure-5547bbe43679.txt` 의 `실패 단계:` 가 **빈 칸** = 어떤 단계에도 진입 못 함(`npm ci` 조차 3초에 안 끝난다). 같은 모양이 `ea6beee`(07:24:11→07:24:13, 2초)·`d195645` 로 세 회차 연속이고, 2026-10-14 01:09Z 이후 같은 계정 다섯 저장소가 전부 2~3초 failure 다. 운영자(hkjang)가 GitHub 과금·스펜딩 한도와 Actions 사용량을 확인해야 풀린다. **CI 로그를 다시 뒤지는 데 시간을 쓰지 말 것.**
  - **`annotations_count: 2` 는 단서가 아니다 (이번 회차에 닫은 거짓 단서).** 98초 만에 **성공**한 `20b1271` 실행(`ci-20b12714a457.json`)도 `annotations_count: 2` 다 — 실패와 성공이 같은 값이므로 실패 원인을 가리키지 않는다. annotations 본문은 네트워크가 막혀 못 읽었다(미확인), 그러나 성공 실행과 값이 같다는 사실만으로 배제된다.
  - `windowStart` 는 `your_priv.quota_counters` 의 **PK** 다 — 옮기면 운영 DB 의 모든 카운터가 한 번에 끊긴다. 회원에게 주는 답(`retryAfterSeconds`)만 고치는 이 수정의 경계를 넓히지 말 것.
  - 보호 경로를 건드리지 말 것: `.github/workflows/**`, `db/migrations/**`(전진 전용), `packages/core/src/auth/**`, `packages/core/src/vendor/postgres`(패치 사본).
  - e2e 가 쓰는 포트 18998·18997 을 다른 프로젝트의 고아 프로세스가 쥐고 있으면 `EADDRINUSE` 로 죽는다(네 회차에서 실제로 막혔다). 그때만 `YOUR_YEOPJARI_URL`·`E2E_PUSH_ORIGIN` 으로 옮기고, 최종 164/0 은 **둘 다 미설정** 상태로 한 번 더 받을 것(기본값이 옛 하드코딩과 동일함을 증명).
  - e2e 는 `signature LIKE 'CLIENT /chat/:id%'` 처럼 문자열 모양에 의존하는 검사가 있다(`scripts/e2e.mjs` 근처 800행) — 오류 알림 서명을 건드리면 운영 DB 의 억제 상태가 한꺼번에 풀린다. 이번 과제 범위 밖이니 손대지 말 것.
- 차선 후보: `v.int` 가 `'0x10'`→16·`'1e3'`→1000 같은 비10진 문자열을 정수로 받아들인다 (3/1/S) — `packages/core/src/util/validate.ts:100` 이 문자열이고 trim 이 비지 않으면 `Number(value)` 를 쓴다. 10진 정수 모양만 받도록 좁히고 기존 거절 메시지를 재사용하며, 고치기 전 실패를 단위 테스트로 먼저 보일 것. 1순위의 fast-forward 가 어떤 이유로 성립하지 않을 때 이것을 고를 것.

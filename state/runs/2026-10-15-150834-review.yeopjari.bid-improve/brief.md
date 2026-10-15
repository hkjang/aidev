# 과제서 — 2026-10-15-150834 (우선 과제: PR #7 CI 실패 대응)

- 과제: PR #7 의 CI 실패 원인을 저장소 안에서 끝까지 배제한 뒤, 배제되면 **CI 의 `perf` 단계가 여유 6.3KB 로 터지기 직전인 것을 경고로 드러내기** (가치 3 / 위험 1 / 작업량 S)
- 왜: PR #7(`origin/auto/2026-10-15-1418`)의 diff 는 `packages/core/src/__tests__/text.test.ts` **한 파일 237+/1-** 뿐이고(`git diff --stat main origin/auto/2026-10-15-1418` 로 이번 회차에 실행 확인), 그 파일을 `git show` 로 전부 읽어 보니 `Date`·`Intl`·`toLocale*`·`process.env`·TZ·로케일·파일시스템·네트워크에 닿는 코드가 **한 줄도 없다**(전부 `isSafeLink`·`koreanParticle`·`maskEmail`·`tidyBody`·`excerpt`·`mailboxKey`·`normalizeEmail`·`emailDomain`·`charCount` 의 고정 입출력 표). 즉 "로컬은 통과, CI 는 실패" 를 만들 환경 의존이 이 PR 안에 없다 — 2026-10-14 회차가 이미 증명한 **계정 수준 Actions startup_failure**(단계 0개 실행·3초 실패·같은 시각부터 계정의 5개 무관한 저장소 13/13 동시 실패)와 같은 그림이다. 그래서 이 회차를 또 "변경없음" 으로 끝내지 않도록, 같은 CI 게이트 안에서 **실제로 터지기 직전인 단계**를 하나 고쳐 둔다: `node scripts/perf.mjs` 는 `223.7 KB / budget 230 KB`(2026-10-14·10-15 두 회차가 실행으로 관측)로 여유가 2.7% 뿐이어서, 다음에 의존성이 하나 늘면 PR 과 무관한 이유로 또 CI 가 붉어진다.

## 0단계 — 원인 배제 (반드시 먼저, 30분 안)
구현자는 `npm` 을 실행할 수 있다(정찰 샌드박스에서는 승인 거부됨 — 이 과제서의 수치는 전부 읽기와 과거 회차 관측이다). 아래를 그대로 돌려 **저장소 쪽 원인이 없음을 실행으로 확정**하라.

1. `git checkout -b <이번브랜치> main` 위에서 PR #7 내용만 올린다: `git checkout origin/auto/2026-10-15-1418 -- packages/core/src/__tests__/text.test.ts`
2. `npm ci --no-audit --fund=false` → `npm run check` → `TZ=UTC npm run check`
   - 기대: 두 번 모두 통과, `Test Files 6 passed (6)` / `Tests 187 passed (187)`. (앞 회차가 관측한 수치 — 파일 수가 6 인 것까지 확인할 것. 기준선은 118 tests / 6 files 였다.)
3. `npm run build` → `node scripts/perf.mjs`
   - 기대: `223.7 KB initial JS (gzip), budget 230 KB` 부근, exit 0.

**분기 A — 2나 3에서 실패하면 그것이 이번 과제다.** 깨진 테스트/단계만 고친다(프로덕션 코드를 건드려야 한다면 그 파일 하나로 끝낼 것). 아래 1단계는 버린다.
**분기 B — 전부 통과하면** 저장소 쪽 원인이 없다는 뜻이다. 워크플로를 느슨하게 만드는 것은 금지이므로(단계 삭제·`continue-on-error`·`timeout-minutes` 완화·`npm ci` → `npm install` 전부 금지) 아래 1단계를 수행하고, 원장에 "PR #7 의 CI 실패는 저장소 밖(계정 Actions) — 로컬 전체 게이트 통과로 배제" 를 실행 출력과 함께 적는다. PR #7 과 같은 내용을 중복 PR 로 다시 올리지 말 것.

## 1단계 — perf 예산 경고 (분기 B 일 때의 본 과제)
- 수용 기준:
  1) `node scripts/perf.mjs` 가 지금처럼 **230 KB 초과에서만 exit≠0** 이다(실패 기준을 숫자 하나도 바꾸지 않는다 — `git diff` 로 임계값 줄이 그대로임을 보일 것).
  2) 현재 크기가 예산의 **95% 이상이면 경고 한 줄**을 표준출력에 찍고 **exit 0 을 유지**한다. 실제로 돌려 그 경고가 오늘 코드에서 출력되는 것을 보인다(223.7/230 = 97.3%).
  3) 경고 문구에 현재 KB·예산 KB·남은 KB 가 들어가 CI 로그만 보고 여유를 알 수 있다.
  4) 임계값을 억지로 넘겨 **실패 경로가 그대로 살아 있음**을 증명한다: 예산 상수를 잠깐 100 KB 로 낮춰 `exit≠0` 과 실패 메시지를 확인하고 즉시 되돌린 뒤(`git diff -- scripts/perf.mjs` 가 의도한 변경만 남는지 확인) 다시 통과시킨다.
- 건드릴 파일: `scripts/perf.mjs` — 예산 비교 지점에 95% 경고 분기 추가. **프로덕션 코드 0개 파일.** (미확인: 이 스크립트를 이번 정찰에서 열어 보지 못했다. 함수·변수 이름과 임계값이 어디에 있는지 먼저 읽고, 출력 문구는 기존 스타일을 그대로 따를 것. 예산 숫자가 `230` 리터럴인지 상수인지도 읽어서 확인하라.)
- 검증 명령:
  - `npm run check` (eslint + 타입 3개 + 단위 테스트)
  - `npm run build` 후 `node scripts/perf.mjs` — 경고가 찍히고 exit 0
  - 예산을 100 KB 로 낮춘 상태에서 `node scripts/perf.mjs` — exit≠0 (되돌릴 것)
  - (DB 가 있으면) `node scripts/e2e.mjs` — 기준선 **218 passed / 0 failed**(`bash scripts/dev-reset.sh` 직후). `scripts/perf.mjs` 는 e2e 와 무관하므로 회귀가 나면 환경 문제다.

## 위험과 피할 것
- **열린 PR 과 파일이 겹치면 안 된다.** 아래 범위 중 **#7 만 이번 회차에 `git diff --stat` 으로 직접 확인했고, #4·#5·#6 은 원장 기록에서 옮긴 것(미확인)** 이다 — 구현자는 시작 전에 `git diff --name-only main origin/<브랜치>` 를 브랜치마다 한 번씩 돌려 겹침 0 을 직접 확인하라(브랜치: `auto/2026-10-14-1413`=#4, `auto/2026-10-15-0938`=#5, `auto/2026-10-15-1108`=#6, `auto/2026-10-15-1418`=#7 — 이 PR↔브랜치 대응도 `gh` 가 막혀 원장 순서로 추정한 것이다). 원장 기준 범위: #4 = `packages/core/src/quota.ts`·`__tests__/quota.test.ts`·`scripts/e2e.mjs`, #5 = `http/pipeline.ts`·`util/text.ts`·`__tests__/client-address.test.ts`, #6 = `scripts/ci-local.sh`·`README.md`·`check.yml`(주석), #7 = `__tests__/text.test.ts`. **`scripts/perf.mjs` 는 어느 PR 에도 없다** — 이 파일만 건드리면 겹침 0.
- `.github/workflows/check.yml` 은 건드리지 말 것. PR #6 이 이 파일(주석 한 줄)을 들고 있고, 워크플로 완화는 금지다.
- `quota.ts`·`pipeline.ts`·`util/text.ts` 는 미머지 PR 과 같은 파일이라 이번 회차에서 금지.
- 보호 경로(`packages/core/src/auth/*`, `db/migrations/`, `routes/sso.ts`, `routes/admin/*`, `media/storage.ts`)는 이 과제와 무관하니 열지도 말 것.
- 과거 교훈: **소스 문자열 검사나 손으로 주입한 대역을 근거로 쓰지 말 것** — `scripts/perf.mjs` 를 실제로 실행한 출력만 증거로 삼는다. 그리고 `npx playwright install --with-deps chromium` 은 이 머신에서 sudo 를 요구해 실패한다(`--with-deps` 없이 설치할 것) — 이번 과제에는 브라우저가 필요 없다.

## 차선 후보
- `domain/products.ts` 의 `slugify` 가 참조 0건의 죽은 코드이고 한글 제품명에서 빈 슬러그를 만든다 — 지우거나 쓰거나 결정하기 (가치 1 / 위험 1 / S). `util/text.ts` 와 달리 어느 열린 PR 과도 겹치지 않는다. 지우는 쪽이면 삭제 전·후로 저장소 전역 grep 0건을 확인할 것(2026-10-15 회차가 `clientIpFrom` 에 쓴 방법 그대로).

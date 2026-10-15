# 과제서 — 2026-10-15-141846-biz.irumx.app-improve

- 과제: 쿼터 429 의 Retry-After 가 하루·한 달 창에서 실제 초기화 시각보다 정확히 9시간 길다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `packages/core/src/quota.ts` 의 카운터 열쇠 `windowStart('day'|'month', at)` 는 **KST 날짜를 UTC 자정으로 라벨링한 값**이라서(`new Date(\`${kstDateKey(at)}T00:00:00.000Z\`)`, :29-30) 카운터가 실제로 바뀌는 순간은 그 라벨보다 9시간 **이른** UTC 15:00 인데, `retryAfterSeconds`(:33-47) 는 그 라벨까지의 초를 세어 `day`·`month` 둘 다 항상 정확히 32,400초(9시간) 과대 응답한다. 이 값은 `rateLimited(message, retryAfter)`(`errors.ts:97`) → `respond.ts:69` 의 `retry-after` 헤더로 그대로 나가고, `helpers.ts:131` 의 `limit(ctx, action, perMinute, perDay)`("Per-account limit, the common case")는 이 오류를 삼키지 않고 그대로 올려보내므로 사용자·API 클라이언트가 실제로 13시간 뒤 풀리는 한도를 "22시간 뒤" 로 안내받는다. 과소가 아니라 과대라 한도 우회는 아니고, 고치면 재시도 안내가 카운터의 실제 초기화 시각과 일치한다.

- 수용 기준:
  1) `retryAfterSeconds`(또는 같은 값을 내는 내보낸 함수)가 `window:'day'` 에서, `at` 이 KST 하루의 어느 시각이든 **다음 KST 하루가 시작되는 UTC 15:00 까지**의 초를 돌려준다. 예: `at = 2026-03-15T00:00:00.000Z`(KST 3/15 09:00) → 현재 `86400`, 기대 `54000`. `at = 2026-03-14T15:00:00.000Z`(KST 3/15 00:00, 창이 막 열린 순간) → 현재 `118800`, 기대 `86400`. `at = 2026-03-15T14:59:59.000Z`(창이 1초 남은 순간) → 현재 `32401`, 기대 `1`.
  2) `window:'month'` 도 같은 방식으로 9시간이 빠진다. 예: `at = 2026-03-31T15:00:00.000Z`(KST 4/1 00:00, 4월 창이 막 열린 순간) → 현재 `2624400`(=30일+9시간), 기대 `2592000`(=30일). `at = 2026-01-31T14:59:59.000Z`(KST 1/31 23:59:59, 1월 창이 1초 남음) → 현재 `32401`, 기대 `1`. 말일이 짧은 달에서 넘어가는 것(`probe` 루프)은 지금 동작을 유지한다 — 예: `at = 2026-02-28T15:00:00.000Z`(KST 3/1 00:00) → 현재 `2710800`, 기대 `2678400`(=31일).
  3) `window:'minute'` 의 반환값은 **한 글자도 바뀌지 않는다** — `minuteStart` 는 진짜 UTC 순간이라 이미 맞다. 시험으로 고정해 두어 수정이 분 창에 번지지 않았음을 보인다.
  4) 내가 손으로 계산한 기대값에만 의존하지 말고, 헤더와 카운터 열쇠를 묶는 **불변식**을 시험으로 증명한다. `r = retryAfterSeconds('day', at)` 에 대해
     - `kstDateKey(new Date(at.getTime() + r*1000)) !== kstDateKey(at)` (그 초가 지나면 창이 정말 바뀌었다)
     - `kstDateKey(new Date(at.getTime() + (r-1)*1000)) === kstDateKey(at)` (필요 이상으로 길지 않다 — **이 줄이 지금 실패한다**)
     `month` 는 같은 두 줄을 `kstDateKey(...).slice(0, 7)` 로, `minute` 은 `minuteStart(...).toISOString()` 비교로 쓴다. 이 불변식은 세 창 모두에 대해 참이어야 한다.
  5) 수정 전 시험이 실패하고(기준 1·2·4), 수정 후 전원 통과하며, 수정을 되돌리면 같은 시험이 다시 실패함을 실행 출력으로 보인다.

- 건드릴 파일 (프로덕션 1개 + 시험 1개):
  - `packages/core/src/quota.ts:33-47` `retryAfterSeconds` — 라벨이 아니라 **창이 실제로 열리는 순간**까지를 센다. 권장 형태(도움 함수 하나 추가, 분기 구조는 그대로):
    ```ts
    /**
     * The instant a day/month window actually opens. The counter key is the
     * Korean date labelled at UTC midnight, and the Korean day starts nine
     * hours before that label.
     */
    function windowOpensAt(window: 'day' | 'month', at: Date): Date {
      return new Date(windowStart(window, at).getTime() - KST_OFFSET_MINUTES * 60_000)
    }
    ```
    - `month` 분기: `probe` 루프는 그대로 두고 마지막 줄만 `windowOpensAt('month', probe)` 로 바꾼다.
    - `day` 분기: `const next = windowOpensAt('day', new Date(at.getTime() + 86_400_000))`.
    - `minute` 분기(:34)는 **건드리지 않는다**.
    - `KST_OFFSET_MINUTES` 는 `./util/time.js` 에서 추가로 import 한다(현재 :11 은 `kstDateKey, minuteStart` 만 가져온다). `kstDayStart` 를 쓰고 싶으면 `day` 분기에는 쓸 수 있지만(`kstDayStart(at + 86_400_000)` 과 동치), `month` 에는 대응 함수가 없으니 한 가지 형태로 통일하는 쪽이 읽기 쉽다. 어느 쪽이든 `time.ts` 는 **고치지 말 것**.
  - `packages/core/src/quota.ts:33` — 시험이 호출할 수 있게 `retryAfterSeconds` 를 `export` 한다(`export function retryAfterSeconds(...)`). 이유를 한 줄 주석으로 남긴다: 429 의 `retry-after` 산술은 카운터 열쇠와 맞아야 하므로 계약으로 고정한다. `windowStart` 도 불변식 시험에 필요하면 함께 내보내도 되지만, 기준 4 는 `kstDateKey`/`minuteStart` 만으로 쓸 수 있으니 **불필요하면 내보내지 말 것**.
  - `packages/core/src/__tests__/quota.test.ts` — **새 파일**. `import { retryAfterSeconds } from '../quota.js'`, `import { kstDateKey, minuteStart } from '../util/time.js'`. 기준 1·2 의 값 표를 `it.each` 류로, 기준 3 의 분 창 표를, 기준 4 의 불변식을 세 창 × 여러 `at` 로 쓴다. 한국어 설명형 `describe`/`it` 이름(예: `describe('쿼터 429 의 Retry-After', …)`).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  npm ci --no-audit --fund=false          # node_modules 가 없으면 먼저. ESLint 지원 종료 경고 1건은 정상
  npm test -- packages/core/src/__tests__/quota.test.ts
  npm run check                           # eslint . + 세 workspace tsc + vitest 전체
  npm run build
  ```
  수정 되돌리기 재실패 확인: `git checkout -- packages/core/src/quota.ts` 로 `windowOpensAt` 변경만 되돌리기는 export 까지 날아가니, **변경 줄만 손으로 임시 복구**(`windowStart(...)` 로 되돌림)해 같은 시험이 실패하는 것을 보이고 다시 고쳐 전원 통과를 재확인한다.
  DB·브라우저가 필요한 `npm run e2e`/`ui`/`a11y` 는 **이번 과제 범위 밖**이다 — `retryAfterSeconds` 는 순수 함수이고 `consume` 의 SQL 경로는 한 줄도 바뀌지 않는다.

- 위험과 피할 것:
  - **`windowStart` 를 고치지 말 것.** 그 반환값은 `biz_priv.quota_counters` 의 기본키 구성요소(`window_start`)로 DB 에 저장돼 있고 `consume`/`release`/`usage` 가 같은 값으로 맞춰 읽는다. 라벨을 9시간 옮기면 기존 카운터가 고아가 되고 한도가 한 번 리셋된다 — 마이그레이션이 필요한 변경이며 이번 회차 범위 밖이다. 고치는 것은 **헤더 산술뿐**이다.
  - `packages/core/src/util/time.ts` 와 `packages/core/src/__tests__/domain.test.ts` 는 **건드리지 말 것**: 미머지 PR #5·#6 이 각각 `domain.test.ts`(+58행, v.day 계약)와 새 `time.test.ts`(15개 시험)를 들고 있다. 새 파일 `quota.test.ts` 로 가면 충돌이 없다(이 체크아웃 `main@714a660` 에 `__tests__/` 는 domain·gateway-role·route-catalogue·shared-db-guard·vendor-postgres 다섯 개뿐이고 `quota.test.ts`·`time.test.ts` 는 없음을 확인했다).
  - 보호 경로를 피할 것: `routes/auth.ts`·`routes/sso.ts`·`auth/*`·`db/migrations`·`.github/workflows` 는 읽기만. 호출부(`helpers.ts`·`agent.ts`·`mcp.ts`·`client-errors.ts`)도 **고치지 않는다** — `agent.ts:97` 이 `rate_limited` 를 `quotaExceeded` 로 바꿔 `retryAfter` 를 버리는 것은 별개 사안이고, 이번 수정의 가치는 `helpers.ts:131` 의 `limit()` 경로에서 이미 나온다.
  - CI 가 2026-10-14 부터 3초·0단계로 죽는 것은 저장소 밖 원인(계정·러너·쿼터)으로 세 회차 연속 확인됐다. `.github/workflows/check.yml` 을 **만지지 말 것** — 로컬 `npm run check` 통과를 증거로 삼는다.
  - 과거 교훈대로 손으로 만든 대역·소스 문자열 검사를 증거로 쓰지 말 것. 이번 증거는 프로덕션 함수의 실제 반환 숫자와 기준 4 의 불변식이다. `consume()` 전체를 돌리려면 실제 `Sql` 이 필요한데 vitest 에는 DB 가 없으니 **가짜 `sql` 을 주입하지 말고** 순수 함수 수준에서 멈춘다.
  - **기준 1·2 의 숫자는 이 세션에서 실행해 얻은 값이 아니라 손으로 계산한 값이다** — 이 세션에서 `node` 실행이 권한에 막혔다(`node -e …` 와 `node <파일>` 모두 "requires approval"). 코드 읽기(`quota.ts:24-47`, `util/time.ts` 의 `kstParts`/`kstDateKey`/`minuteStart`)로 유도한 값이며, 실제로 재계산하면서 처음 적은 월 값(`2678400`)이 틀려 `2624400` 로 고쳤다. **구현자는 기대값을 그대로 믿지 말고 먼저 실행해 확인할 것.** 같은 산술을 현재 동작·수정 후 동작 양쪽으로 재현하는 독립 스크립트를 `/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-15-141846-biz.irumx.app-improve/check-retry.mjs` 에 남겨 뒀다(`node <경로>` — **이 세션에서는 실행하지 못했다**). 숫자가 다르면 **불변식(기준 4)이 정본**이고 표의 숫자를 실제 반환값으로 맞춰라. 9시간(32400초) 차이라는 성질 자체는 `windowStart` 가 KST 날짜를 UTC 자정으로 라벨링한다는 코드에서 바로 따라온다.
  - 미확인: 운영 로그에 `day`/`month` 429 가 실제로 얼마나 나는지는 확인할 수 없었다. `limit()` 이 삼키지 않고 올려보내는 것과 `respond.ts:69` 가 헤더를 쓰는 것은 소스에서 확인했으나, 그 경로를 지나는 실제 HTTP 응답은 이 세션에서 관측하지 않았다.

- 차선 후보: **프로젝트 손익 조회의 from/to 역순·범위 밖 기본값** (`packages/core/src/routes/finance.ts:734-735` 가 from/to 가 비면 `'1900-01-01'`·`'9999-12-31'` 을 `v.day` 를 지나지 않고 쿼리에 넣고, `to < from` 도 거르지 않아 0원 손익을 정상 응답으로 낸다). 1순위가 성립하지 않을 때만 — 기본값 의미를 바꾸면 DB e2e(`scripts/e2e.mjs`)가 필요하고 finance 쓰기 경로에 가깝다. 3순위는 `v.int` 의 안전 정수 상한 누락(`validate.ts`, `Number.isSafeInteger` 가 아니라 `isFinite+isInteger` 만 본다 — `max` 없는 호출부 조사 선행).

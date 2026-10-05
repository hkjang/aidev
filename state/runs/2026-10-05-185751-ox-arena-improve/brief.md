# 과제서 — 2026-10-05 (ox-arena, base main@f68a6fa)

- 과제: 3초 카운트다운 동안 O/X 버튼이 활성처럼 보이지만 눌러도 무반응인 것 고치기 (가치 3 / 위험 1 / 작업량 S)

- 왜: 문제마다 `started_at` 직전 **3초 카운트다운**(0001_init.sql:15 주석, Timer.tsx:10-12 가 3·2·1 을 표시) 동안 `src/pages/Play.tsx:79` 의 `canChoose` 가 `phase === 'countdown'` 을 포함해 버튼의 `disabled` 를 풀어 두지만, `src/pages/Play.tsx:88` 의 `pick()` 첫 줄 `if (phase !== 'choosing') return;` 이 `setLocal` 조차 하기 전에 빠져나간다 → 참가자가 탭해도 선택 표시(`sel`)도, 에러도, 아무 변화가 없다. 행사장에서 문제마다 3초간 "버튼이 먹었나?" 를 전원이 겪는다. 서버도 같은 판단을 한다(`supabase/migrations/0003_elimination.sql:34` `v_now < r.started_at - interval '300 milliseconds'` → `LOCKED`) 이므로 카운트다운 중 제출은 애초에 수락되지 않는다. 즉 클라이언트가 서버 계약과 어긋나 켜 둔 것이고, 끄면 `src/styles.css:6` 의 `button:disabled { opacity:.4; cursor:not-allowed; }` 가 이미 있으므로 **카운트다운엔 흐릿 → "시작" 순간 선명해짐** 이라는 올바른 신호가 공짜로 생긴다.

- 수용 기준:
  1) `phase === 'countdown'` 일 때 O/X 버튼이 `disabled` 이고(브라우저가 클릭을 막으므로 탭이 아무 일도 안 하는 것이 **의도된 상태**로 바뀜), `styles.css:6` 규칙에 의해 흐리게 보인다.
  2) `phase === 'choosing'` 과 `phase === 'locked'` 에서의 동작은 **한 글자도 바뀌지 않는다**: choosing 에서 버튼 활성·`pick()` 정상 제출, locked 에서 비활성. 탈락자(`out === true`) 는 그대로 버튼 블록 자체가 렌더되지 않는다(Play.tsx:143 `!out` 조건).
  3) `canChoose` 와 `pick()` 의 가드가 **같은 조건**이 되어, "버튼은 켜져 있는데 핸들러가 거부" 하는 상태가 코드에 남지 않는다. (즉 `canChoose` 에서 `'countdown'` 을 빼는 것이 정답 방향이고, `pick()` 가드를 느슨하게 푸는 방향은 금지 — 서버가 `LOCKED` 를 던진다.)
  4) `npm run build`(= `tsc --noEmit && vite build`) 가 EXIT 0.
  5) `git diff --stat` 이 **`src/pages/Play.tsx` 한 파일만** 보여 준다(CSS 변경 불필요 — 이미 있다).

- 건드릴 파일:
  - `src/pages/Play.tsx:79` — `const canChoose = !out && (phase === 'choosing' || phase === 'countdown');` 에서 `|| phase === 'countdown'` 를 제거해 `const canChoose = !out && phase === 'choosing';` 로. 이것이 변경의 전부다.
  - (선택, 같은 파일) 카운트다운 중 버튼 영역에 "잠시 후 시작" 같은 힌트를 넣고 싶다면 `Play.tsx:143` 의 `.buttons` 블록 안에서만 하고, 새 CSS 클래스는 만들지 말 것(스타일 파일을 건드리면 수용 기준 5 가 깨진다). 넣지 않아도 수용 기준은 충족된다 — Timer 가 이미 3·2·1 을 크게 보여 준다.

- 검증 명령:
  - 기준선 먼저: `npm ci` → `npm run build` 가 녹색인지 확인하고 시작할 것(2026-10-03 회차에서 `npm ci` = 82 packages, build EXIT 0 으로 실측됨).
  - 변경 후: `npm run build` (EXIT 0), `git diff --stat` (1 파일), `git status --porcelain` (잔여물 없음).
  - **base 에 테스트 러너가 없다**(package.json 에 `test` 스크립트·vitest 없음 — 2026-10-05 직접 확인). 따라서 자동 회귀 증거는 `npm run build` 뿐이다. 다음 두 가지 중 가능한 쪽으로 행동 증거를 보강할 것:
    - (a) 구현 시점에 vitest PR 이 이미 머지됐다면(`grep '"test"' package.json` 로 확인) **`src/lib/phase.ts` 에 `canInteract(phase, eliminated)` 순수 함수를 추가**하고 Play.tsx 가 그것을 쓰게 바꾼 뒤 `npm test` 로 7단계 phase × eliminated 진리표를 고정하라. 이 경우 건드릴 파일은 3개(`phase.ts`, `phase.test.ts`, `Play.tsx`).
    - (b) 머지 전이라면(지금 base 상태) **package.json / package-lock.json / 새 lib 모듈을 절대 만들지 말고** Play.tsx 1줄 수정으로 끝내라. 과제서에 진리표를 적어 리뷰어가 읽고 판정할 수 있게 하라: `countdown → disabled(변경점)`, `choosing → enabled`, `locked → disabled`, `revealed/finished → 블록 자체 없음`, `waiting/ready → disabled`.

- 위험과 피할 것:
  - **`supabase/migrations/**` 를 건드리지 말 것.** 서버는 이미 올바르게 거부하고 있다(0003:34). 로컬 SQL 실행 수단이 없고(`supabase/config.toml` 없음) 되돌리려면 사람이 운영 DB 에서 SQL 을 다시 실행해야 한다.
  - **`package.json` / `package-lock.json` 을 건드리지 말 것** — 2026-10-02 회차의 vitest PR 이 아직 머지 전(review-pending)이라 충돌한다. 같은 이유로 `src/lib/phase.ts` 를 **새로** 만들지 말 것(그 PR 이 같은 경로에 만든다). 위 (a) 조건은 "이미 머지되어 phase.ts 가 존재할 때" 만 해당한다.
  - **`.github/workflows/**` 를 건드리지 말 것** — `pages.yml` 은 main push 즉시 배포되는 유일한 릴리즈 경로이고, `ci.yml` 은 2026-10-03 회차 PR 이 머지 대기 중이다.
  - `pick()` 의 `if (phase !== 'choosing') return;` 를 **완화하지 말 것**. 완화하면 서버가 `LOCKED` 를 던지고, `Play.tsx:93-95` 가 `setLocal(null)` 로 선택을 되돌린 뒤 "마감되어 선택이 반영되지 않았어요" 를 띄운다 — 지금보다 나쁜 체감이 된다.
  - `phase === 'countdown'` 에서 문제 텍스트는 이미 보인다(`Play.tsx:112`). 그것을 숨기는 쪽으로 번지지 말 것(Display/Admin 과 어긋난다).
  - 과거 교훈: 소스 문자열 grep 을 증거로 제출하지 말 것. 실제로 돌린 `npm run build` 출력(EXIT 코드 포함)을 붙일 것.

- 차선 후보: **라우트별 코드 스플리팅으로 참가자 초기 번들 줄이기** — 현재 `dist/assets/index-*.js` 434.14 kB (gzip 127.80 kB) 단일 청크(2026-10-03 실측). `src/main.tsx` 가 Play/Display/Admin 세 페이지를 정적 import 하므로 휴대폰 참가자가 Admin(305줄) + Display + `qrcode.react` 까지 모두 받는다. `React.lazy` + `<Suspense>` 로 바꾸되 세 페이지가 **named export**(`export function Play`)이므로 `lazy(() => import('./pages/Admin').then(m => ({ default: m.Admin })))` 형태가 필요하다. 건드릴 파일은 `src/main.tsx` 1개. 검증은 `npm run build` 출력의 청크 목록으로 객관적으로 보인다(초기 청크가 줄고 Admin/Display 청크가 분리). 위험: Suspense fallback 없으면 전환 중 빈 화면 — `<Suspense fallback={<div className="screen center"><div className="spinner" /></div>}>` 로 기존 스피너 마크업(Play.tsx:75)을 재사용할 것.

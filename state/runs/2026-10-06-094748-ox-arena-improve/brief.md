- 과제: 문제가 진행되지 않는 동안에도 계속 도는 Timer 의 20Hz 리렌더 루프 멈추기 (`useRoom.ts:useNow` + `Timer.tsx`) (가치 3 / 위험 2 / 작업량 S)

- 왜: `src/components/Timer.tsx:6` 이 `useNow(50)` 을 무조건 호출해서 **방 상태와 무관하게 초당 20번** `Timer` 를 리렌더한다. 그런데 `Timer` 는 `countdown`/`choosing`/`locked` 세 단계에서만 DOM 을 그리고 나머지(`waiting`/`ready`/`revealed`/`finished`)에서는 `Timer.tsx:23` 의 `return null` 로 끝난다 — 즉 행사 시간의 대부분(참가자 대기, 정답 공개, 종료 화면) 동안 화면에 아무 변화도 없는데 50ms 타이머가 계속 돈다. 행사장에서 수십 대의 휴대폰이 대기 화면을 켜 두고 있는 것이 이 앱의 정상 사용 패턴이라 배터리·CPU 를 그대로 버리는 중이고, 고치면 `status !== 'playing'` 인 동안 타이머가 0이 된다.

- 수용 기준:
  1) `useNow` 가 `intervalMs` 로 `null` 을 받으면 `setInterval` 을 등록하지 않는다(조기 반환).
  2) `Timer` 가 `room.status === 'playing'` 일 때만 50ms 간격을 요청하고, 그 외에는 `null` 을 넘긴다.
  3) `null → 숫자` 로 바뀌어 타이머가 **재시작하는 순간 `now` 가 즉시 갱신**된다. (이게 이 과제의 유일한 실질 위험이다 — 아래 "위험" 참조. `setInterval` 만 걸고 첫 틱을 기다리면 `now` 가 마운트 시점 값이라 카운트다운 첫 프레임에 엉뚱한 숫자가 스친다.)
  4) 비-playing 상태에서 `now` 가 멈춰도 화면이 같다는 것을 **프로덕션 `derivePhase` 로 실측**: `useRoom.ts` 의 실제 `derivePhase` 에 `status` 를 `waiting`/`ready`/`revealed`/`finished` 로 둔 room row 를 주고 `now` 를 극단값 2개(예: `0` 과 `Date.now()+1e12`)로 넣어 네 상태 모두 반환값이 동일함을 출력으로 보일 것. → `now` 가 멈춰도 `Timer` 와 `Display` 의 표시가 바뀌지 않는다는 근거.
  5) `npm run build`(= `tsc --noEmit && vite build`) EXIT 0. `useNow` 시그니처를 `number | null` 로 넓히는 것이므로 기존 호출부 3곳(`Timer.tsx:6`, `Play.tsx:61`, `Display.tsx:14`)은 타입이 그대로 통과해야 한다.
  6) `git diff --stat` 이 **2개(또는 선택 과제까지 3개) 파일**이고 `src/pages/Play.tsx` · `package.json` · `package-lock.json` · `.github/workflows/**` 는 0줄 diff.

- 건드릴 파일:
  - `src/lib/useRoom.ts:20-27` `useNow(intervalMs = 100)` — 시그니처를 `intervalMs: number | null = 100` 으로 넓히고, effect 안에서 `if (intervalMs == null) return;` 로 조기 반환. 반환 전이 아니라 **인터벌을 걸기 바로 전에 `setNow(serverNow())` 를 한 번 호출**해 재시작 시 즉시 동기화(수용 기준 3). `intervalMs` 는 이미 의존성 배열에 있으므로 값이 바뀌면 effect 가 재실행된다(파일에서 확인함).
  - `src/components/Timer.tsx:6` `const now = useNow(50);` → `useNow(room.status === 'playing' ? 50 : null)`. `room` 은 이미 prop 으로 들어와 있고(`Timer.tsx:5`) 조기 반환보다 앞이라 훅 순서 문제 없음.
  - (선택, 같은 패턴) `src/pages/Display.tsx:14` `useNow(200)` → `useNow(room?.status === 'playing' ? 200 : null)`. 이 호출은 `Display.tsx:16` 의 `if (!room) return <spinner/>` 보다 **위**에 있으므로 `room` 이 `null` 일 수 있다 → 반드시 옵셔널 체이닝. `now` 는 `Display.tsx:17` 의 `derivePhase(room, now)` 한 곳에서만 쓰인다(파일 전체를 읽고 확인함). 시간이 남을 때만 하고, 애매하면 2파일로 끝낼 것.

- 검증 명령:
  - `npm i` → `npm run build` (EXIT 0 확인. 기준선부터: 변경 전에 한 번 돌려 녹색인지 보고 시작할 것 — 이 저장소에 `test` 스크립트는 **없다**, `package.json` 2026-10-06 재확인)
  - `git diff --stat` / `git status --porcelain`
  - 수용 기준 4 실측: 2026-10-05 회차가 성공한 방식을 그대로 쓸 것 — vite 의 `ssrLoadModule` 로 `src/lib/useRoom.ts` 를 불러 실제 `derivePhase` 를 호출하는 임시 프로브(커밋하지 말 것). 그 회차 기록에 따르면 이 방식으로 `derivePhase` 가 Node 에서 실제로 로드·실행된다.

- 위험과 피할 것:
  - **가장 위험한 자리는 타이머 재시작 순간이다.** `status` 가 `revealed` → `playing` 으로 넘어갈 때 effect 가 재실행되는데, `useNow` 의 `now` state 는 그동안 멈춰 있어 몇 분 전 값이다. 인터벌만 새로 걸면 최초 50ms 동안 `Timer.tsx:11` 이 `Math.max(1, Math.ceil((s - now)/1000))` 로 거대한 숫자를 그린다. **반드시 즉시 `setNow(serverNow())`**. 이 한 줄이 빠지면 매 문제 시작마다 눈에 보이는 결함이 생기므로, 과제 전체보다 이 줄을 먼저 확인할 것.
  - `src/pages/Play.tsx` 를 열지 말 것 — 2026-10-05 회차의 카운트다운 수정(`Play.tsx:79`)이 아직 머지 전이다(base `f68a6fa` 에 그 수정 없음을 2026-10-06 재확인). `Play.tsx:61` 도 같은 `useNow(200)` 루프를 갖고 있지만 **이번 회차에서는 손대지 말고** 그 PR 이 정리된 다음 회차로 넘길 것. 과제서가 파일을 2~3개로 묶은 이유가 이것이다.
  - `package.json` / `package-lock.json` 금지 — vitest 도입 PR 이 머지 대기 중(base 에 vitest·`test` 스크립트 없음, 2026-10-06 재확인). 따라서 **이번 과제는 자동 테스트를 추가할 수 없다.** 새 테스트 파일이나 테스트 러너를 들이려 하지 말 것.
  - `.github/workflows/ci.yml` 금지 — CI PR 이 머지 대기 중(base 의 `.github/workflows/` 에 `pages.yml` 하나뿐, 2026-10-06 `ls` 로 재확인). `pages.yml` 은 유일한 릴리즈 경로이므로 절대 건드리지 말 것.
  - `supabase/migrations/**` 는 이번 과제와 무관 — 열 필요 없다.
  - **미확인 / 솔직히 적는 부분**: 수용 기준 2("인터벌이 실제로 등록되지 않는다")를 **실행 증거로 증명할 수단이 이 저장소에 없다.** 렌더러 테스트 수단(vitest/jsdom/react-test-renderer)이 전부 없고 그것을 들이는 것은 위에서 금지했다. 손으로 만든 대역으로 React 훅을 흉내 내지 말 것(운영자 금지 사항). 구현자는 ①수용 기준 4(프로덕션 `derivePhase` 실측, 동작 보존 근거) ②`npm run build` 녹색 ③브라우저에서 수동 확인이 가능하면 그것까지를 증거로 적고, "인터벌 횟수를 측정했다"고 쓰지 말 것. 절감량(실제 리렌더 횟수·배터리)도 **측정하지 않았으므로 수치로 주장하지 말 것** — 이 정찰도 측정하지 못했다.
  - 이 정찰 세션에서는 `npm i` 가 승인 거부로 막혀 **빌드를 한 번도 돌리지 못했다**. 기준선 녹색 여부는 미확인이다. 구현자가 가장 먼저 확인할 것.

- 차선 후보: 라우트별 코드 스플리팅으로 참가자 초기 번들 줄이기 (`src/main.tsx` 1파일) — `main.tsx:4-6` 이 Play/Display/Admin 을 정적 import 해서 단일 청크가 된다. 세 페이지 모두 **named export** 임을 2026-10-06 확인했으므로 `lazy(() => import('./pages/Admin').then(m => ({ default: m.Admin })))` 형태가 필요하고, `<Routes>` 를 `<Suspense fallback={<div className="screen center"><div className="spinner" /></div>}>` 로 감싼다(이 마크업은 `Play.tsx:20` 등에 이미 쓰이는 것이라 새 CSS 불필요 — `styles.css:11,14,15` 에 `.spinner`/`.screen`/`.screen.center` 존재 확인). **다만 1순위로 올리지 않은 이유를 알고 고를 것**: 이 정찰은 `npm i` 가 막혀 절감량을 측정하지 못했고(2026-10-05 기록의 434 kB 는 당시 실측치), 번들의 대부분은 react·react-dom·react-router·supabase-js 공용 의존성이라 Play 진입 청크의 실제 절감은 작을 수 있다. 반대로 청크를 쪼개면 행사장 와이파이에서 **청크 로드 실패라는 새 실패 경로**가 생긴다. 이것을 고른다면 수용 기준에 "진입 청크 크기 전/후 실측치"를 반드시 넣고, 절감이 미미하면 변경을 버릴 것.

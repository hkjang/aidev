- 과제: `web/src/components/AdminAskOriginalLauncher.test.tsx` 의 비결정 실패 제거 — 검색 응답이 도착하기 전에 **disabled 인 Select 를 클릭**해 드롭다운이 열리지 않는 레이스 (가치 5 / 위험 2 / 작업량 S) — **수정 과제 (우선 배정)**

- 왜: 마지막 회차가 `cd web && npm test --silent` exit 1 로 verify-failed 했고 릴리즈 워크플로가 같은 이유로 두 번 실패했다. 이 게이트를 깨는 유일한 알려진 실패는 `AdminAskOriginalLauncher.test.tsx:110` 의 `Unable to find an accessible element with the role "option" and name '홍길동 · hong@example.test · ID 19'` 이고, 이번 회차에 **코드를 읽어 그 비결정성의 구조적 원인을 특정했다**(아래 '원인'). 고치면 릴리즈 게이트(`ci.yml:49`·`release.yml:75` = `npm test --prefix web -- --run`)가 결정적으로 녹색이 된다.

- 원인 (이번 회차에 코드로 확인 — 실행 재현은 아래 '미확인' 참조):
  1. `web/src/components/AdminAskOriginalLauncher.tsx:271` — 대상 사용자 `Select` 는 `disabled={usersLoading || busy || users.length === 0}` 다. 검색이 비행 중이면(`usersLoading===true`) **또는 아직 결과가 없으면**(`users.length===0`) 이 combobox 는 disabled 다.
  2. `searchUsers`(같은 파일 150-173)는 `setUsersLoading(true)` → `await apiClient.get('/admin/users?...')` → `setUsers(found)` → `finally setUsersLoading(false)` 순서다. 즉 **응답이 커밋될 때까지 Select 는 disabled** 다.
  3. 테스트 2(`AdminAskOriginalLauncher.test.tsx:108-110`)는 `사용자 검색` 버튼을 클릭한 뒤 **아무 대기 없이** 곧바로 `await screen.findByRole('combobox', { name: '시험 대상 사용자' })` → `fireEvent.click(...)` → `await screen.findByRole('option', …)` 를 한다. `findByRole` 은 **disabled 인 combobox 도 즉시 찾아 resolve** 하므로(접근성 트리에 그대로 있다) 대기 역할을 하지 못한다. 그 상태에서 클릭하면 react-dom 이 disabled form control 의 마우스 이벤트 핸들러를 호출하지 않아(`shouldPreventMouseEvent`) **클릭이 삼켜지고 드롭다운이 열리지 않는다** → 110행이 기본 1000ms 뒤 실패한다.
  4. 테스트 1(`66-71`)과의 **유일한 구조적 차이**가 이것이다: 테스트 1 은 68행에 `await waitFor(() => … '/admin/users?search=%ED%99%8D%EA%B8%B8&limit=20&offset=0' … )` 가 있어 그 폴링/act 플러시 동안 응답이 커밋되고 Select 가 enabled 가 된 뒤 70행에서 클릭한다. 2026-10-03 회차가 남긴 단서("테스트 2 에는 그 waitFor 가 없다")와 정확히 일치한다.
  - **왜 1/3 확률인가**: 모킹 fetch 는 즉시 resolve 하는 async 함수라 보통은 `findByRole(combobox)` 가 반환되기 전에 마이크로태스크가 소진되어 enabled 가 되지만, 이벤트 루프/GC 타이밍에 따라 `findByRole` 이 먼저 resolve 하면 disabled 상태로 클릭한다. 전체 스위트(27파일, `fileParallelism:false`·`maxWorkers:1` = 한 프로세스에서 순차 실행, `web/vite.config.ts` 의 `test` 블록)에서 타이밍이 흔들리는 것과 맞는다.

- 수용 기준:
  1) **결정적 red 를 먼저 만든다.** 권장 프로브: 테스트 2 의 `/admin/users?search=` 분기 응답에만 지연을 넣는다(`await new Promise((r) => setTimeout(r, 50))` 후 `jsonResponse(...)`). 이 프로브로 110행이 **매번** `Unable to find … role "option" … '홍길동 · hong@example.test · ID 19'` 로 실패해야 한다. (2026-10-03 회차는 "50ms 지연으로 재현 실패" 를 기록했으므로 이 예측이 틀릴 수 있다 — 틀리면 즉시 `screen.debug()`/실패 DOM 덤프로 **클릭 시점의 combobox 가 `disabled`/placeholder `먼저 서버에서 사용자를 검색하세요` 인지** 확인해 원인을 다시 특정하고, 아래 '차선 후보' 로 넘어가지 말고 특정된 원인을 고칠 것.)
  2) 수정 후 프로브(지연)를 **유지한 채** 테스트 2 가 통과하고, 프로브를 제거해도 통과한다. 프로브를 되돌려 수정만 되돌리면 다시 red 가 되는 것(인과 확인)을 보일 것.
  3) 러너와 **똑같은 명령** `cd web && npm test --silent` 가 **연속 5회 exit 0**(27 파일 / 134 테스트). 이번 정찰에서 1회는 이미 exit 0 이었으므로 1회 통과는 증거가 되지 않는다 — 5회를 돌릴 것.
  4) role·접근가능이름 단정을 **글자 하나도 바꾸지 않는다**. `timeout` 확대, `retry` 설정 추가, `vite.config.ts` 의 `test` 블록 완화는 **금지**(게이트를 느슨하게 만드는 것).
  5) 프로덕션 `.tsx`/`.ts` 와 `.github/workflows/` 는 한 줄도 바꾸지 않는다 — `git diff --stat` 로 보일 것.

- 건드릴 파일 (2개):
  - `web/src/components/AdminAskOriginalLauncher.test.tsx:108-110` — `사용자 검색` 클릭 뒤 **Select 가 실제로 조작 가능해지기를** 기다린다. 두 가지 형태 중 하나(둘 다 이 파일의 기존 관용구 범위 안):
    - (권장) `const userSelect = within(dialog).getByRole('combobox', { name: '시험 대상 사용자' });` 후 `await waitFor(() => expect(userSelect).toBeEnabled());` 그 다음 `fireEvent.click(userSelect)`. `toBeEnabled` 는 `@testing-library/jest-dom/vitest`(`web/src/test/setup.ts:1`)로 이미 들어와 있다.
    - (대안) 테스트 1 의 68행과 같은 `await waitFor(() => expect(fetchMock.mock.calls.some(…'/admin/users?search=')).toBe(true))`. 단 이것은 "호출됐다" 만 보장하므로 (권장) 쪽이 더 강하다. 이 경우 90행의 `vi.spyOn` 반환값을 `fetchMock` 으로 받아야 한다(현재는 받지 않는다).
  - `web/src/components/AdminAskOriginalLauncher.test.tsx:70` — 테스트 1 도 같은 레이스에 **원리적으로 노출**돼 있다(68행 waitFor 가 플러시를 해 주는 덕에 지금까지 터지지 않았을 뿐). 같은 `toBeEnabled` 대기를 넣어 두 테스트를 같은 관용구로 맞출 것. 재발 면역이 목적이다.
  - (선택, 범위를 넘기지 말 것) 위 두 곳으로 끝난다. **다른 테스트 파일·프로덕션 컴포넌트는 건드리지 말 것.**

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm test --silent` — 러너가 쓰는 바로 그 명령. **연속 5회.** (이번 정찰에서 1회 실행 = `Test Files 27 passed (27) / Tests 134 passed (134)`, exit 0, 약 2~3분. `pretest`(`web/scripts/ensure-deps.mjs`)가 node_modules 를 자기충족적으로 깐다.)
  - `npm test --prefix web -- --run` — CI(`ci.yml:49`)·release(`release.yml:75`)의 실제 형태.
  - `npm run typecheck --prefix web` (= `tsc -b --pretty false`)
  - 단독 확인용: `cd web && npx vitest run src/components/AdminAskOriginalLauncher.test.tsx`

- 위험과 피할 것:
  - **프로덕션 컴포넌트를 바꾸지 말 것.** `Select` 가 로딩 중 disabled 인 것은 올바른 UX 이고, 결함은 테스트의 대기 누락이다. `disabled` 조건을 건드리면 사용자 동작이 바뀌고 범위가 터진다.
  - **워크플로(`.github/workflows/`)는 보호 경로**이며 이번 과제에서 바꿀 이유가 없다. `-count=1`·`timeout-minutes` 같은 별건(ideas.json 에 있음)을 섞지 말 것 — 12223b0 이 두 번 머지되지 않았다.
  - 단정 완화·timeout 확대·`retry` 추가는 운영자 규칙상 금지된 땜질이다. 증명 없는 수정도 금지 — 기준 1 의 결정적 red 가 선행이다.
  - 1회 통과를 증거로 쓰지 말 것: 이번 정찰의 1회 실행이 exit 0 이었다(= 1/3 확률 flake 와 일치).
  - `grep` 으로 "waitFor 가 있다/없다" 를 증거로 제출하지 말 것 — 실행으로 red→green 을 보일 것.

- 미확인 (정직하게):
  - **실패를 이번 회차에 실행으로 재현하지 못했다.** 게이트 명령 1회 실행은 exit 0(27/134 통과)이었고, 반복 실행은 이 세션의 권한·예산 한도로 돌리지 못했다. 위 '원인' 은 `AdminAskOriginalLauncher.tsx:150-173`·`:263-272` 와 `AdminAskOriginalLauncher.test.tsx:66-71`·`:105-112` 를 직접 읽어 세운 **구조적 가설**이며, react-dom 의 disabled 마우스 이벤트 억제는 코드로 확인하지 않았다(프레임워크 동작 지식).
  - 릴리즈 워크플로가 실제로 이 테스트에서 실패했다는 로그는 보지 못했다 — 회차 노트의 `cd web && npm test --silent (exit 1)` 만 근거다. 다른 테스트가 깨졌을 가능성은 배제하지 못한다. 구현자는 **먼저 5회 반복 실행으로 실제 실패 테스트의 이름을 확정**하고, 그것이 110행이 아니면 이 과제서를 버리고 그 실패를 고칠 것.

- 차선 후보: `AnalyzeDialect` 사유 개수 상한 — 1MiB 질의가 사유 81,512개(약 6.8MB)를 응답에 담는다(3/3/S, ideas.json 2번 항목). 단, 1순위가 "실제 실패가 110행이 아니다" 로 무너진 경우에는 차선으로 가지 말고 **실제로 깨진 테스트**를 고칠 것 — 이번 회차는 verify 실패 수정 회차다.

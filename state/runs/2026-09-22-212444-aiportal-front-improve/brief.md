# 과제서 (정찰, 2026-09-22 21:24 회차)

- 과제: [수정 과제] 릴리즈 실패 원인 확정 — 외부 절차 교착의 근거를 저장소에 남기고, 진입 조건 미충족이면 즉시 차선으로 전환 (가치 5 / 위험 2 / 작업량 S)

- 왜: 릴리즈가 "다음 버전·태그 관례 근거 부족"으로 반복 실패했고, 이번 회차에 원인을 외부 절차 원문에서 직접 확정했다 — `/mnt/c/Users/USER/projects/aidev/release-prompt.md:21,22`(2·3항)은 "최근 릴리즈들의 증가 패턴"과 "이전과 같은 방식"을 요구하는데 이 저장소에는 태그 0개·릴리즈 커밋 0개라 따를 패턴이 없고, 같은 파일 `:24`(5항)의 `skipped`는 "태그도, **버전 파일도**, 릴리즈 노트도 없음"을 요구하는데 `package.json:3`에 `"version": "0.0.0"`이 있어 성립하지 않는다. 즉 `released`도 `skipped`도 불가능한 구조적 교착이며, 해소하려면 관례를 새로 정해야 하는데 그것은 5항 스스로 "사람의 일"로 못 박았고 이 저장소의 `AGENTS.md`·`docs/RELEASE.md:3,83`도 금지한다. 따라서 이번 회차에 저장소 안에서 릴리즈를 통과시키는 합법적 수단은 없고, 7회째 무변경을 반복하는 대신 검증된 실제 결함 하나를 처리해 회차를 성과 있게 끝내는 것이 이득이다.

- 진입 조건(이것부터 판정하라, 30초):
  1) 이 brief.md·journal.md·운영자 지시 어디에도 **승인된 증가 단위(patch/minor)·태그 형식·릴리즈 커밋 메시지 양식·노트 위치**가 출처와 함께 적혀 있지 않다. (정찰이 확인함 — 없다.)
  2) 없으므로 1순위는 **성립하지 않는다**. 관례를 추측해 만들지 말고, 아래 "차선 후보"를 이번 회차의 실제 구현 과제로 삼아라. 1순위에 대해서는 코드·버전 파일·태그를 건드리지 말고, 원장에 위 근거(release-prompt.md 2·3항 vs 5항 교착)를 한 줄로 남기기만 하라.
  3) 만약 위 1)의 정책이 **출처와 함께** 주어진 회차라면, 그때만 그 정책대로 버전 파일 갱신·노트·커밋·태그를 수행하고 차선은 건드리지 마라.

- 수용 기준 (차선 = 이번 회차 실제 구현 대상):
  1) `commonStorage`의 `sidemenuAppList` 값이 배열이 아닌 값(예: `{ unexpected: true }`)일 때 `fetchTopApps()`와 `useAppList().fetch()`가 예외 없이 빈 목록을 돌려준다. 현재는 `getFormattedAppList`가 `list.map`을 호출해 `TypeError: list.map is not a function`으로 터진다.
  2) 두 진입 경로(`fetchTopApps`, `useAppList`의 `updateList`)가 **같은 값을 같게** 읽는다 — 한쪽만 고치지 말 것. 방어는 이미 있는 `toAppArray`(`src/utils/appList.js:18`, `export const toAppArray = (value) => (Array.isArray(value) ? value : [])`)를 재사용하고 새 헬퍼를 만들지 말 것.
  3) 테스트가 증명할 것: 비배열 캐시(`{ unexpected: true }`)와 `null`/미설정 상태에서 `fetchTopApps()`가 `[]`를 반환하고 throw하지 않는다는 것, 그리고 정상 배열에서는 기존 매핑 결과(`id`/`appId`/`label`/`favorite`/`useCnt`/`_raw`)와 `limit` slice가 그대로라는 것(회귀 방지). 가드를 되돌리면 새 테스트가 실패해야 한다(Red 확인).

- 건드릴 파일:
  - `src/composables/useAppList.js:39 fetchTopApps` — `commonStorage.get(...) || []`는 객체가 truthy라 방어가 되지 않는다. `toAppArray(...)`로 교체.
  - `src/composables/useAppList.js:52 updateList`(`useAppList` 내부) — 같은 교체. 두 곳 모두.
  - `src/composables/useAppList.js:12 getFormattedAppList` — `import { toAppArray } from '@/utils/appList'` 추가. (선택) 함수 진입부에서도 `toAppArray(list)`로 감싸면 세 경로가 한 값으로 수렴한다.
  - `tests/unit/` 에 스펙 추가 — 기존 `tests/unit/appList.spec.js:146`의 "캐시 값이 배열이 아니어도 예외 없이 API 로 조회한다" 케이스가 `src/utils/appList.js` 경로만 덮고 있으니, 같은 패턴으로 composable 경로 스펙을 만든다. `fetchTopApps`는 Vue 라이프사이클이 없어 컴포넌트 마운트 없이 직접 호출 가능하다(`useAppList()`는 `onMounted` 사용 — SFC 플러그인이 없는 vitest 설정이므로 컴포넌트 마운트 대신 `fetchTopApps` + `useAppList().fetch()` 직접 호출로 검증할 것).
  - 순환 import 주의: `src/utils/appList.js`가 `useAppList.js`를 import하지 않는지 먼저 확인(정찰 확인 시 하지 않음). 한다면 `toAppArray`만 쓰는 방향을 유지하되 import 방향을 재확인하라.

- 검증 명령:
  - `npm ci` (worktree에 `node_modules` 없음 — 먼저 실행해야 한다. 시간 소요 있음)
  - `npx vitest run tests/unit/appList.spec.js <새 스펙 파일>` (빠른 확인)
  - `npm test` (= `vitest run`, 전체 19개 스펙 회귀 확인. 이것이 통과해야 완료)
  - 가드 한 줄을 되돌린 뒤 같은 명령으로 새 테스트가 **실패**하는지 확인하고 다시 되돌린다(Red/Green 증명).

- 위험과 피할 것:
  - 릴리즈 관례(버전 증가·태그·CHANGELOG)를 **새로 만들지 마라.** `AGENTS.md`와 `docs/RELEASE.md:3,83-84`, release-prompt 5항이 모두 금지한다. `package.json`/`package-lock.json`의 `0.0.0`을 건드리지 마라.
  - 외부 `release-prompt.md`·게이트·러너 스크립트는 이 저장소 밖이며 수정 범위가 아니다. 워크플로를 느슨하게 고쳐 통과시키는 것도 금지다.
  - `docs/RELEASE.md`의 재조회 스크립트는 `package.json`/lock이 `0.0.0`임을 assert한다(`docs/RELEASE.md:45`). 버전을 바꾸면 이 문서 명령이 깨진다 — 또 하나의 "건드리지 말 것" 근거.
  - 차선 작업에서 `getFormattedAppList`의 매핑 규칙(`status`의 `String(...)`, `filter((x) => x.appId && x.label)`)을 바꾸지 마라. 이번 과제는 비배열 입력 방어만이다. 출력이 바뀌지 않는 장식성 변경은 반려 사유다.
  - `Sidemenu.vue:231`(쓰는 쪽)과 `src/utils/appList.js:176`도 같은 키를 쓴다. 읽는 쪽 방어만 하고 쓰는 쪽 계약은 바꾸지 마라.
  - 보호 경로(auth/라우터/스트리밍) 무관 — 건드리지 말 것.

- 차선 후보: **useAppList의 비배열 `sidemenuAppList` 캐시 가드 (가치 3 / 위험 2 / 작업량 S)** — 위 "수용 기준/건드릴 파일/검증 명령"이 곧 이 차선의 명세다. 진입 조건 2)에 따라 **이번 회차의 기본 실행 대상**이다. 그것마저 성립하지 않으면(예: 이미 고쳐져 있으면) 3순위는 `README.md` 문제 해결 절의 `npm run build`를 실제 존재하는 모드별 스크립트(`npm run build:dev` 등)로 고치는 문서 정합성 수정이다.

## 이번 정찰이 실제로 확인한 것 / 미확인
- 확인: `git log --oneline -12`(HEAD `e938e8e`), `git tag | wc -l` = **0**, `package.json:3`·`package-lock.json:3,9` 모두 `0.0.0`, `"private": true`, `package.json:28 "test": "vitest run"`, `node_modules/.bin/vitest` **부재**, `docs/RELEASE.md` 전문, `release-prompt.md` 전문, `useAppList.js` 전문, `toAppArray` 정의와 사용처, `tests/unit/appList.spec.js:146` 비배열 케이스, 소비자 `src/views/Home.vue:35`·`src/views/Chat/Index.vue:40`.
- 미확인: `npm ci`/`npm test`를 이번 회차에 실행하지 않았다(정찰은 저장소를 바꾸지 않으며 설치 시간이 큼) — 테스트가 현재 전부 green인지는 구현자가 먼저 확인할 것. 비배열 캐시가 실제로 저장되는 런타임 경로(어떤 응답이 객체를 넣는지)는 재현하지 않았다. 다만 `src/utils/appList.js`가 이미 같은 값에 `toAppArray` 방어를 두고 테스트까지 있다는 것이 "두 경로 중 하나만 방어됨"의 근거다.
- 미확인: 외부 GitHub Actions run ID·step 로그. "같은 GitHub step 2회 실패"로 단정하지 않는다. 확인한 것은 release 단계가 버전 근거 부족으로 실패했다는 state 기록과 위 교착의 원문 근거다.

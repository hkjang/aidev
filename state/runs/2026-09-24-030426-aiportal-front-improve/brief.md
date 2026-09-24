# 과제서 (2026-09-24, HEAD 0031fde)

## A. 우선 과제(릴리즈 실패) — 이번 회차도 **무변경**이 정답 (19회째)

이번 회차 워크트리(HEAD `0031fde`)에서 직접 재조회한 결과:
- `git tag` **0개**, `package.json:3` `version` = `0.0.0`(`private: true`), `.github`/`CHANGELOG.md`/`VERSION`/`scripts`/`Makefile` **전부 부재**(`ls -a` 로 확인).
- `.gitlab-ci.yml` 은 브랜치 푸시 배포 전용(`CI_COMMIT_TAG` 참조 0건).
- `git log --oneline -8`: 릴리즈 커밋 양식 0건, 전부 `fix:` + merge PR.

→ **고칠 워크플로 파일이 이 저장소에 없다.** 실패 지점은 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이고, 저장소 안에서 통과시키는 유일한 수단은 시작 버전·증가 단위·태그 형식·릴리즈 커밋 양식·노트 위치를 **새로 발명**하는 것인데, 이는 AGENTS.md("확인되지 않은 관례를 새로 만들거나 릴리즈 판정 조건을 완화하지 않는다")와 "워크플로를 느슨하게 만들어 통과시키는 것 금지" 양쪽에 걸린다. `package.json` 에 버전 필드가 있어 외부 절차의 `skipped` 조건도 충족하지 않는다(`docs/RELEASE.md:84` 의 동일 판단).

**구현자 지시**: 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송을 일체 하지 말 것. `docs/RELEASE.md` 도 수정하지 말 것. 판정을 skipped/released 로 낮추지 말 것. 재조사도 불필요(위 근거가 이번 HEAD 기준). 필요한 사람 입력 6가지는 아래 ideas.json 과 동일.

→ 따라서 **이번 회차의 실제 실행 대상은 아래 B** 다.

---

## B. 실행 과제

- 과제: 즐겨찾기 화면의 카드 설정(더보기) 팝업이 **빈 상자로 뜨고 아무 동작도 하지 않는** 문제 수정 (가치 4 / 위험 2 / 작업량 M)

- 왜: `src/views/Favorite/FavoriteList.vue:195` 의 `<ButtonSetting>` 만 형제 화면 4곳과 달리 `:actions` 를 넘기지 않고(`ButtonSetting.vue:7` 의 기본값 `[]` → `visibleActions` 빈 배열 → `<ul>` 이 비어 렌더), 존재하지도 않는 `@toggle-fav` 를 듣고 있다(`ButtonSetting.vue:10` 의 `defineEmits` 는 `['close','action']` 뿐). 그래서 즐겨찾기 화면에서 카드 더보기를 누르면 **빈 팝업만 뜨고**, 이미 구현돼 있는 `onToggleFav`(`:142`)와 `removeCard`(`:134`)는 **한 번도 실행되지 않는 죽은 코드**다 — 사용자는 즐겨찾기 화면에서 즐겨찾기를 해제할 수 없다. 고치면 이 화면의 해제 동작이 살아나고, 이미 존재하지만 아무도 쓰지 않던 정책 분기 `buildSettingActions({ mode: 'favorite' })`(`src/utils/buttonSettingPolicy.js:54-59`)가 제 자리를 찾는다.

- 수용 기준:
  1) 즐겨찾기 화면에서 카드 더보기를 열면 팝업에 액션 항목이 **1개 이상 실제로 렌더된다**(`li` 개수 > 0). 항목 구성은 `buildSettingActions({ mode: 'favorite', item })` 가 돌려주는 것(`fav`, `share`)과 일치한다.
  2) '즐겨찾기' 항목을 클릭하면 `inf.app.appDeleteBookmark` 요청이 실제로 나가고(카드의 `fav` 는 `mapAppToCard:51` 에서 항상 `true` 이므로 `nextYn`은 `'N'`), 성공 시 해당 카드가 목록에서 사라지며(`removeCard`) 설정 팝업이 닫힌다.
  3) '앱 공유' 항목이 렌더된다면 그 키도 처리돼야 한다 — 누르면 `copyAppId` → 성공 시 `toast('app_id를 복사했습니다.')`(`ServiceList.vue:211-219` 와 동일 형태). **처리 핸들러를 붙이지 않은 채 버튼만 렌더하지 말 것**(죽은 버튼을 새로 만드는 것은 이 결함을 다른 모양으로 옮기는 것일 뿐이다).
  4) 업무 실패(HTTP 200 + `isSuccess` false)면 카드가 사라지지 않고 목록이 그대로 유지된다(`addBookmark`/`deleteBookmark` 가 `false` 를 돌려주면 `onToggleFav` 의 `if (!ok) return`).
  5) 테스트는 **`@toggle-fav` 로는 아무 일도 일어나지 않고 `@action` 으로만 동작한다**는 인과를 증명해야 한다: 수정 전 스펙이 Red(팝업 `li` 0개 / 삭제 요청 0건)였다가 수정 후 Green 이 되고, 변이(`:actions` 만 제거 / `@action` 만 제거)로 각각 해당 케이스만 다시 실패함을 확인할 것.

- 건드릴 파일:
  - `src/views/Favorite/FavoriteList.vue:195` — `mode="support"`(선언되지 않은 prop, 폴스루 속성일 뿐)와 `@toggle-fav` 를 제거하고 `:actions="settingActions"` `@action="onSettingAction"` 로 교체. 형제 4곳(`SupportList.vue:560`, `ServiceList.vue:320`, `MyAgentList.vue:635`, `InfoSearchList.vue`)이 전부 이 형태다.
  - `src/views/Favorite/FavoriteList.vue` script — `computed` 와 `buildSettingActions`(`@/utils/buttonSettingPolicy.js`) import 추가, `const settingActions = computed(() => buildSettingActions({ mode: 'favorite', item: selectedItem.value }))`, 그리고 `onSettingAction({ key, item })` 를 추가해 `key === 'fav'` → 기존 `onToggleFav({ item: target, nextYn: target?.fav ? 'N' : 'Y' })`, `key === 'share'` → `copyAppId` 경로로 라우팅. **기존 `onToggleFav` 의 본문은 그대로 둘 것**(이미 옳다).
  - `copyAppId` 의 import 경로는 **미확인** — `ServiceList.vue` 의 import 줄을 그대로 복사해 쓸 것(`grep -n "copyAppId" src/views/Service/ServiceList.vue`).
  - 신규 `tests/unit/favoriteSettingActions.spec.js` — HTTP 전송(axios adapter) 한 겹만 대역으로 바꾸고 실제 `interceptors.js` → 실제 `inf.app.getBookmark/appDeleteBookmark.call` → 실제 `FavoriteList.vue` 마운트를 통과시키는 형태(`sidemenuDeleteCommonError.spec.js` 가 가장 가까운 전례).
  - `src/utils/buttonSettingPolicy.js`, `src/components/common/card/ButtonSetting.vue`, 형제 화면 4곳은 **건드리지 말 것**.

- 검증 명령:
  - `npm ci`(node_modules 가 비어 있으면 선행 필수, 수 분)
  - `npx vitest run tests/unit/favoriteSettingActions.spec.js` (Red → Green 확인용)
  - `npm test` — 기준선 **31파일 509테스트** 통과. 이 숫자보다 줄면 회귀.
  - `npm run build:dev` 후 `dist/` 삭제. (`npm run build`·`npm run lint` 는 존재하지 않는다)

- 위험과 피할 것:
  - `mode: 'favorite'` 분기(`buttonSettingPolicy.js:54-59`)는 `common` = `[fav, share]` 를 돌려준다. `share` 를 렌더하면서 핸들러를 안 붙이면 새 죽은 버튼이 생긴다 — 수용 기준 3을 반드시 지킬 것. 정책 파일 자체를 고쳐 `share` 를 빼는 방향은 금지(다른 mode 와 `common` 을 공유하므로 형제 화면까지 바뀐다).
  - `ButtonSetting` 은 `actions` 항목의 `show === undefined` 를 true 로 본다(`:14`). `hasItem` 이 false 면 전부 숨겨지므로, 테스트는 `openSetting` 이 `selectedItem` 을 채운 뒤의 상태를 봐야 한다.
  - 마운트 하네스 주의: `FavoriteList` 는 `useRouter`(vue-router), `SwiperList`, 전역 로딩(`useGlobalLoading`)을 쓴다. `openSetting`(`:98`)은 `anchorEl.getBoundingClientRect` 가 함수가 아니면 중앙 정렬로 조기 return 하는데 **`settingOpen`/`selectedItem` 은 그 전에 이미 세팅**되므로 jsdom 에서도 팝업은 열린다.
  - 전역 로딩/알림/토스트는 모듈 싱글턴이라 케이스 간 상태가 샌다 — 각 케이스에서 정리할 것.
  - 보호 경로(`api/common/interceptors.js`, `api/auth.js`, `storage/auth*`, `router`)와 릴리즈 관련 파일은 건드리지 말 것.
  - 운영자 지시: 관측 가능한 동작 변화가 없는 정리는 넣지 말 것. 이 과제의 변경은 전부 화면 동작을 바꾸므로 그 선을 넘지 않도록 `mode="support"` 제거 외의 미용 수정은 삼갈 것.

- 차선 후보: `Sidemenu.getHistoryList`(`Sidemenu.vue:416`)의 bare catch 가 설정/전송 오류까지 삼켜 '최근 대화 기록이 없습니다' 와 구분되지 않는 문제 — 단, '조회 실패를 사용자에게 알릴 것인가' 라는 기대 계약이 미확인이므로, 손댄다면 'COM' 가드와 함께 **실패/빈 결과를 구분하는 상태**만 도입하고 문구는 기존 것을 재사용할 것.

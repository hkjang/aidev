# PR 처리기 노트 2026-09-22-214502-aiportal-front-shepherd — aiportal-front PR #20
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-22-212444-aiportal-front-improve)
# 회차 노트 2026-09-22-212444-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:24] base pinned — main@e938e8e
- [러너 21:24] autonomy release — 

## 정찰 노트
- 고정 과제의 원인을 원문에서 확정했다: release-prompt.md:21,22는 이전 증가 패턴을 요구하고 :24의 skipped는 버전 파일 부재를 요구하는데, 태그 0개 + package.json:3 "0.0.0" 이라 released·skipped 모두 불가한 교착이다. 관례 신설은 AGENTS.md·docs/RELEASE.md·5항이 모두 금지하므로 저장소 안 합법 수단은 없다 — 7회 연속 무변경의 진짜 이유다.
- 그래서 과제서는 1순위를 "진입 조건 판정 + 근거 한 줄 기록"으로 좁히고, 미충족(=현재)이면 차선을 **기본 실행**하도록 명시했다. 무변경 반복 대신 검증된 결함 하나를 처리시키려는 판단이다.
- 차선을 고른 이유: useAppList.js:40,53의 `|| []`가 비배열 캐시를 못 막아 list.map이 TypeError를 내는데, 형제 경로 src/utils/appList.js:77은 toAppArray(:18)로 이미 방어되고 tests/unit/appList.spec.js:146이 그 케이스를 덮는다 — 같은 값을 읽는 두 경로 중 하나만 방어된 상태로, 운영자가 반복 지적한 end-to-end 정합 유형이다. Header 폴링·globalLoading·manualChunks는 기대 계약 미확인이라 제쳤다.
- 확신 없는 곳: 비배열 값이 실제로 저장되는 런타임 경로는 재현하지 않았다(근거는 형제 경로의 기존 방어와 테스트). node_modules 부재로 npm ci/test를 이번에 돌리지 않았으니 구현자가 baseline green을 먼저 확인할 것. GitHub run ID/step 로그는 미확인이라 "같은 step 2회 실패"로 단정하지 않았다.
- 조심할 것: 버전 파일·태그·CHANGELOG를 만들지 말 것, 외부 절차를 느슨하게 고치지 말 것, getFormattedAppList의 매핑 규칙은 그대로 둘 것. ideas.json은 도구 제약(다른 run 디렉터리 복사·python 실행 불가)으로 직전 71개 전체를 옮기지 못하고 살아 있는 pending 12개 + 신규 2개만 담았다. 전체 이력은 runs/2026-09-22-194424-aiportal-front-improve/ideas.json 에 있다.
- [러너 21:29] scout done — [수정 과제] 릴리즈 실패 원인 확정 — 외부 절차 교착의 근거를 저장소에 남기고, 진입 조건 미충족이면 �

## 구현 노트
- 1순위(릴리즈)는 진입 조건 미충족이라 코드·버전 파일·태그를 건드리지 않았다: release-prompt.md:21,22는 이전 증가 패턴을 요구하나 태그 0개, :24의 skipped는 버전 파일 부재를 요구하나 package.json:3이 "0.0.0" — released도 skipped도 불가한 교착이고 관례 신설은 금지다. 과제서 지시대로 차선을 구현했다.
- 차선: useAppList.js의 fetchTopApps(:40)·updateList(:53)에서 `|| []`를 기존 toAppArray로 교체. 객체 캐시는 truthy라 방어가 안 돼 getFormattedAppList의 list.map이 TypeError를 냈고, 형제 경로 utils/appList.js:77만 방어돼 두 경로가 갈려 있었다. tests/unit/useAppList.spec.js 9케이스 추가(비배열/null/미설정 + 정상 배열 매핑·limit 회귀).
- 검증: Red 3개 실패(TypeError @ useAppList.js:13) → npm test 20파일 433테스트 통과(기준 19/424) → 가드를 한 곳씩 되돌리면 각각 2개 실패로 인과 증명 → npm run build:dev 통과 후 dist/ 삭제. 커밋 39c6880.
- 확신 없는 곳: **비배열 값이 실제로 저장되는 런타임 경로는 재현하지 못했다.** utils/appList.js:176은 Array.isArray로 거른 list만 쓰므로 이 앱 코드에서 객체를 넣는 경로는 못 찾았다 — 손상/구버전 localStorage 값이 유력한 출처다. 즉 이번 수정은 "형제 경로에 이미 있는 방어를 맞춘 것"이지 관측된 프로덕션 장애의 수정이 아니다.
- 일부러 안 한 것: getFormattedAppList 진입부 wrap(과제서의 선택 항목) — 세 군데 다 넣으면 두 개가 중복이 되어 되돌림 검증으로 증명되지 않는 변경이 된다. 매핑 규칙(String(...), filter)도 손대지 않았다. 쓰는 쪽(Sidemenu.vue:231) 계약 그대로.
- 다음 역할 주의 1: 새 스펙은 `vi.mock('@/api/interface.js')`가 **필수**다. useAppList가 이제 @/utils/appList를 거쳐 @/api → @/router → .vue 를 끌고 오는데 vitest에 SFC 플러그인이 없어 MainLayout.vue:78에서 터진다(appList.spec.js도 같은 이유로 목을 둔다). 목을 빼면 스펙 파일 자체가 로드 실패한다. ideas.json에 별도 항목으로 남겼다.
- 다음 역할 주의 2: 프로덕션 번들에는 새 모듈이 추가되지 않는다 — Home.vue:17,21과 Chat/Index.vue:16이 이미 @/api/interface와 @/utils/appList를 import한다(확인함). DB·네트워크 없이 도는 테스트다.
- [러너 21:34] brief fallback — 차선 — 진입 조건 1)이 정한 대로 승인된 증가 단위·태그 형식·커밋 양식·노트 위치가 어디에도 출처와 함께 없어 1순�
- [러너 21:34] verify passed — 검증 2개 통과 (auto)
- [러너 21:34] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 21:34] pr created — https://github.com/hkjang/aiportal-front/pull/20

## 심사 노트
- 직접 확인: npm ci 후 `npx vitest run` 20파일 433테스트 통과, useAppList.js만 origin/main 으로 되돌리면 새 스펙 3개가 `TypeError: list.map is not a function @ useAppList.js:13` 로 실패 — 테스트가 수정 대상을 실제로 통과한다(무효 테스트 아님). `npm run build:dev` 통과, dist 삭제, 워크트리 clean.
- 직접 확인: SIDEMENU_APP_LIST 독자 3곳(useAppList.js:41,:54, utils/appList.js:77)이 모두 toAppArray 를 거친다 — 운영자가 요구한 "같은 값을 읽는 모든 경로가 같게 읽는지" 가 이번 변경으로 충족된다. 쓰기측(Sidemenu.vue:215,231 / utils/appList.js:171,176)은 둘 다 Array.isArray 로 걸러 배열만 넣는다.
- 순환 import 확인: useAppList → utils/appList → api/interface → api/index → router → Home.vue → useAppList 고리가 새로 생기지만, Home.vue:17,21 이 이미 같은 두 모듈을 useAppList(:27) 보다 먼저 import 하고 toAppArray 는 함수 본문에서만 참조되어 TDZ 위험이 없다. 빌드에 순환 경고 없음.
- 못 본 것: 비배열 값이 실제로 저장되는 런타임 경로(구현자도 재현 못 함) — 저장소 안 writer 로는 도달 불가이고 손상/구버전 localStorage 가 유일한 출처라 실측 장애 수정이 아닌 방어 정합 수정이다. 브라우저 실행·PR CI 로그도 보지 않았다(GitHub 권한 없음).
- 권고 merge 근거: 결함 없음, 범위 이탈 없음(2줄 + import 1줄 + 신규 스펙), 인증·권한·개인정보·의존성 변경 없음, 순수 revert 가능. notes 의 status "undefined" 는 이번 diff 가 건드리지 않은 선재 버그라 차단 사유로 쓰지 않았다.

# 회차 노트 2026-09-27-094216-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:42] base pinned — main@3a5935c
- [러너 09:42] autonomy release — 

## 정찰 노트
- 빈 트리 최상위 추가(3/1/S)를 골랐다. 코드로 직접 확인된 사용자 막힘이고 프로덕션 1파일로 끝난다. 차선인 validateActivityImport 단위 테스트는 프로덕션 결함을 고치지 않아 뒤로 뒀고, propagateStatus 통일(3/2/M)은 09-20 에 고친 자리를 다시 흔들어 단독 회차가 맞다.
- 확인한 것: admin/page.tsx:257-325 의 tree.map, ActivityTree.tsx:111-115 의 isRoot 전용 '같은 레벨 추가', route.ts:62 의 action:'add'+parentId:null 지원, treeUtils.ts:47 addActivity 의 level 1, activityData.ts:139-143 이 빈 activities 배열을 통과시킴, e2e 의 beforeEach/afterEach 복원 패턴.
- 추측으로 적은 것: docs/ADMIN_GUIDE.md 에 작업 추가 절차 절이 있는지 미확인(있을 때만 갱신하라고 적었다). 기존 e2e 가 title='같은 레벨 추가' 버튼을 쓰는지도 미확인 — 그래서 그 버튼 삭제는 선택 사항으로 두고 수용 기준에서 뺐다.
- 구현자가 조심할 것: 새 버튼이 기존 submitActivityChange 를 우회해 새 fetch 를 쓰면 09-26 의 실패 배너 성과가 무너진다. 성공 판정에 SWR 1초 폴링을 쓰지 말 것. lint 전에 playwright-report/·test-results/ 를 지울 것.
- node_modules 가 없어 lint/tsc/test/build 를 하나도 실행하지 못했다(미확인). 코드 변경·커밋 없음.
- [러너 09:46] scout done — 관리자 화면 활동 목록에 항상 보이는 "최상위 작업 추가" 진입점과 빈 상태 안내 추가 (가치 3 / 위험 1 / �

## 구현 노트
- 바꾼 것: `app/admin/page.tsx` 트리 컨테이너에 항상 보이는 `add-root-activity` 버튼과 `tree.length===0` 일 때의 `empty-activity-tree` 안내. activities 가 비면 추가 수단이 0이던 막힘을 없앴다. 핸들러는 기존 `submitActivityChange({action:'add', parentId:null})` 그대로 — 새 fetch 없음.
- 확신 없는 곳: 없음에 가깝지만 두 가지를 적어 둔다. (1) 성공 후 "빈 상태 문구가 사라진다" 단언은 `submitActivityChange` 의 `mutate()` 재검증(200 확인 뒤)에 기대며, 이 한 줄만은 SWR 갱신을 거친다 — 성공 판정 자체는 PUT 응답과 `request.get('/api/activities')` 로 했다. (2) ADMIN_GUIDE 의 PDF·캡처(assets/guide/admin-console.png)는 새 버튼 이전 화면이라 md 와 어긋난다. 다시 굽지 않았다.
- 일부러 안 한 것: `ActivityTree.tsx:111-115` 의 `isRoot` 전용 '같은 레벨 추가' 버튼 제거 — 이제 중복이고 기존 e2e 가 쓰지 않는 것도 확인했지만 수용 기준 밖이라 손대지 않았다(ideas.json 에 후보로 적음). 상태 전파 로직·`route.ts`·`treeUtils.ts` 도 건드리지 않았다.
- 덤으로 고친 문서 1줄: ADMIN_GUIDE 4.2 의 "확인 창이 두 번 뜬다" 설명은 09-26 회차에 이미 한 번으로 바뀐 뒤 남아 있던 옛 서술이라 현재 동작으로 정정했다.
- 다음 역할이 조심할 것: e2e 는 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome` 가 있어야 돈다. `npm run lint` 전에 `rm -rf playwright-report test-results` 를 꼭 할 것(eslint 가 trace 뷰어 번들을 검사한다). 새 spec 은 `beforeEach` 에서 activities 를 빈 배열로 만들고 `afterEach` 에서 되돌리므로 중간에 끊기면 e2e 데이터 파일이 빈 상태로 남을 수 있다(격리본 `test-results/e2e-data/` 라 실데이터 영향 없음).
- [러너 09:51] brief accepted — 채택 — 과제서의 근거가 코드와 정확히 일치했고(257행 `tree.map`, ActivityTree 의 isRoot 전용 추가 버튼, route.ts 의 `parentId:null`
- [러너 09:51] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- 확인한 것(파일 직접 열람): `git diff 3a5935c...HEAD` 는 3파일 110줄뿐. admin/page.tsx 258·333행의 두 testid 가 이번 커밋에서 처음 생기므로 신규 e2e 2건은 수정 전 코드에서 반드시 타임아웃 — 원장의 `- 실패 재현:`(`waiting for getByTestId('add-root-activity')`)과 증상이 일치한다. route.ts:62 의 `parentId === null` 분기, treeUtils.ts:47-58 의 level 1·'새 액티비티'·'대기', ActivityTree.tsx:120 의 확인창 1회(지우는 노드 제목)까지 코드로 확인했다. 신규 spec 은 admin-save-failure.spec.ts 와 같은 `route.fallback()` 관용구·같은 로그인 패턴을 쓴다.
- 구현자가 의심한 두 곳을 시험했다. (1) mutate() 의존은 `expect` 자동 재시도 범위라 문제 없고 성공 판정은 PUT 응답·GET 으로 분리돼 있다. (2) PDF·admin-console.png 스테일은 사실 — md 본문과 어긋난 채 남는다.
- 데이터 유실 경로를 특히 팠다: GET 500 → fetcher 가 res.ok 를 안 봐서 빈 상태로 보이고 버튼이 눌리는 조합. readActivityData 가 non-ENOENT 를 다시 던져 PUT 도 500 으로 끝나므로 덮어쓰기는 없다. 차단 아님, notes 로 남김.
- 못 본 것: lint/tsc/test:unit/test:e2e/build 를 이 세션에서 직접 돌리지 않았다(원장 주장 + verify.gate 의 tsc·build 통과에 의존). ADMIN_GUIDE.pdf 본문과 캡처 이미지도 열지 않았다. 원장의 test:unit 79건/25 suites 는 프로필의 72건/24 와 다르지만 이번 diff 에 단위 테스트 변경이 없어 프로필 드리프트로 본다.
- 승인. 다음 회차에 남는 것: '읽는 중/읽기 실패' 와 '비어 있음' 구분(fetcher 의 res.ok 미검사), 문서 PDF·캡처 재생성, 중복이 된 isRoot '같은 레벨 추가' 버튼 정리.
- [러너 09:55] review approved — 리뷰 승인 (risk=low)
- [러너 09:55] pr created — https://github.com/hkjang/cutover/pull/7
- [러너 09:56] ci passed — 검사 없음 — 정책으로 허용
- [러너 09:56] merge done — 255d9d7
- [러너 09:59] release published — v1.10.0
- [러너 09:59] gh-release created — GitHub Release v1.10.0
- [러너 09:59] manifest ok — cutover-v1.10.0.tar.gz 
- [러너 09:59] assets uploaded — 1개
- [러너 09:59] assets verified — v1.10.0 자산 1개 (이전 v1.9.0: 1)

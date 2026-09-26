- 과제: 관리자 콘솔의 활동 변경 실패를 화면에 표시하고, 삭제 확인창 중복(둘째 창은 엉뚱한 제목)을 제거 (가치 4 / 위험 1 / 작업량 M)

- 왜: `app/admin/page.tsx` 의 활동 PUT 6곳(137·149·267·279·288·296·304행)이 모두 응답을 검사하지 않고 곧바로 `mutate()` 만 부른다. API 는 401(`code: 'UNAUTHORIZED'`)·400(`code: 'INVALID_DATA'`)·500 을 정상적으로 내보내지만(`app/api/activities/route.ts` PUT), 관리자는 실패를 모른 채 1초 SWR 폴링이 되돌린 이전 값을 보고 저장됐다고 오해한다. 또 삭제 확인창이 두 번 뜨는데 둘째 창(`app/admin/page.tsx:287`)의 `node.title` 은 `tree.map(node => …)`(219~220행)의 **최상위 노드**를 가리키므로, 하위 항목을 지울 때 전혀 다른 제목을 보여 준다(실제 오동작).

- 수용 기준:
  1) 활동 상태변경·편집·삭제·이동·하위추가·상황판 제목 저장 중 어느 것이든 응답이 2xx 가 아니면 관리자 화면에 실패 메시지가 보인다. 401 이면 "다시 로그인" 흐름으로 돌아간다(`isAuthed` false 로). 성공하면 메시지는 사라진다.
  2) 삭제 시 `confirm` 은 한 번만 뜨고, 그 문구의 제목은 실제로 지우는 항목의 제목이다(`components/ActivityTree.tsx:122` 의 `node` 는 자기 노드라 맞음 → `app/admin/page.tsx:287` 쪽을 제거).
  3) 제목/시간 인라인 편집 저장이 실패하면 편집 입력이 닫히지 않고 입력한 값이 남는다(현재 `ActivityTree.tsx:70-73 handleSave` 는 결과를 기다리지 않고 `setIsEditing(false)`). 상황판 제목 편집(`app/admin/page.tsx:137·149`)도 실패 시 `setIsEditingTitle(false)` 하지 않는다.
  4) e2e 가 증명할 것: (a) `page.route` 로 PUT 을 500 으로 가로챈 뒤 상태 드롭다운을 바꾸면 실패 메시지가 보인다, (b) 삭제 버튼 클릭 시 `page.on('dialog')` 로 수집한 확인창이 정확히 1건이고 문구에 그 항목 제목이 들어 있다, (c) 가로채기를 풀면 같은 조작이 성공하고 메시지가 사라진다.

- 건드릴 파일 (프로덕션 3개 + 테스트 1개):
  - `app/admin/page.tsx` — 6개 PUT 을 공통 헬퍼(예: `submitActivityChange(body): Promise<boolean>`)로 모아 `res.ok` 검사·응답 JSON 의 `error`/`code` 표시·401 이면 `setIsAuthed(false)`, 실패 메시지 state 와 배너 추가. `onDeleteActivity`(287행)의 중복 `confirm` 제거. 제목 편집 성공 시에만 `setIsEditingTitle(false)`.
  - `components/ActivityTree.tsx` — `onUpdateActivity` 시그니처를 `Promise<boolean> | void` 를 받을 수 있게 하고 `handleSave` 가 결과를 기다려 실패면 `isEditing` 유지. 삭제 버튼(122행)에 e2e 로케이터용 `title="삭제"` 를 붙여도 좋다(형제 버튼들은 모두 `title` 을 가짐).
  - `e2e/admin-save-failure.spec.ts` (신규) — `e2e/tree-ops.spec.ts` 를 형판으로: `request.post('/api/auth/login', {data:{pw:'e2e-admin-password', role:'admin'}})` 로 데이터 준비/복원하고, UI 로그인은 `page.getByPlaceholder('관리자 비밀번호')` + `getByRole('button',{name:'로그인'})`. 트리 상태 드롭다운은 `page.locator('select').filter({ has: page.locator('option[value="완료"]') })` 로 고른다(설정 카드의 select 제외). `beforeEach` 에서 원본 activities 를 저장하고 `afterEach` 에서 되돌릴 것.
  - 문서는 필요하면 `docs/ADMIN_GUIDE.md` 6장 문제 해결 한 줄만. PDF 재생성 도구(md2pdf)는 이 환경에 없으니 굽지 말 것.

- 검증 명령 (이 저장소 실제 명령. 이번 정찰에서는 `node_modules` 부재 + 셸 권한 제한으로 **하나도 실행하지 못했음 — 미확인**):
  - `npm ci --legacy-peer-deps`
  - `npm run lint`
  - `npx tsc --noEmit`
  - `npm run test:unit`  (node --test, 현재 72건/24 suites 기준)
  - `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e`  (현재 10건; webServer 가 127.0.0.1:3100 dev 서버를 띄움)

- 위험과 피할 것:
  - `app/api/auth/**`·`lib/adminSession.ts`·`proxy.ts`·`Dockerfile` 은 건드리지 말 것(보호 경로). 401 처리는 관리자 화면 쪽에서만.
  - `app/api/activities/route.ts` 의 PUT 계약은 바꾸지 말 것. 이 과제는 클라이언트가 이미 오는 응답을 읽게 하는 것이다.
  - 상태 전파 규칙(`onStatusChange` 의 부모 자동 진행/완료 로직, 228~265행)과 `collectDescendantIds` 는 이번에 손대지 말 것 — 09-20 회차에서 막 고친 자리다.
  - SWR 1초 폴링이 실패를 덮으므로 **폴링으로 값이 되돌아오는 것을 성공/실패 증거로 쓰지 말 것**. 증거는 PUT 응답과 화면의 메시지다.
  - 실패 주입은 손으로 만든 대역이 아니라 `page.route` 로 **실제 프로덕션 컴포넌트·실제 fetch 경로**에 대해 할 것.
  - e2e 실패 trace 가 `playwright-report/`·`test-results/` 에 남으면 `npm run lint` 가 그 번들을 검사해 수천 건을 낸다. lint 전에 산출물을 지울 것(eslint globalIgnores 추가는 별도 아이디어로 남겨 둘 것).
  - `confirm` 을 둘 다 없애고 한쪽만 남기는 것이 목표다. 양쪽을 "통합"하려고 삭제 흐름을 admin 쪽으로 옮기지 말 것(ActivityTree 쪽이 올바른 노드를 갖고 있다).

- 차선 후보: `validateActivityImport` 단위 테스트 추가 — `lib/activityData.test.ts` 가 없음(확인됨: lib 의 테스트는 adminSession·treeUtils·mail 4개·tracking 2개뿐). 실제 함수를 동적 import 해 throw 2종(최상위 비객체 / activities 비배열), 필드 검증(id·parentId·level 1~50·time·title 길이), `'진행중'→'진행'` 정규화 카운트, issues 50건 초과 시 `omittedIssueCount` 를 고정. (가치 3 / 위험 1 / S)

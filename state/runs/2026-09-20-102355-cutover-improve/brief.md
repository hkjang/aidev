# 과제서 2026-09-20 — cutover

- 과제: 삭제·상태 전파의 하위 탐색을 ID 접두사 대신 parentId 기반으로 바꾸고 단위·e2e 테스트로 고정 (가치 4 / 위험 2 / 작업량 S)

- 왜: `lib/treeUtils.ts:64-68 deleteActivity` 와 `app/admin/page.tsx:229-234 onStatusChange` 가 하위 노드를 `a.id.startsWith(id + '-')` 로 찾는다. 관리자가 activity.json 을 업로드하면 id 는 아무 문자열이나 될 수 있어(validateActivityImport 는 parentId 관계만 검사하고 id 형식은 강제하지 않음), 예컨대 `{id:"net", parentId:null}` 아래 `{id:"vlan", parentId:"net"}` 인 트리에서 "net" 을 삭제하면 "vlan" 만 남아 `validateActivityImport` 가 `부모 id "net"가 activities에 존재하지 않습니다` 로 400 을 던지고(`app/api/activities/route.ts:81,94-98`), 관리 화면은 이 실패를 표시하지 않아(`admin/page.tsx:284-292` 는 응답을 보지 않음) 삭제가 조용히 안 된다. 같은 트리에서 부모 상태를 바꿔도 자식 상태가 따라 바뀌지 않는다. 두 경로가 `parentId` 로 하위를 모으는 한 함수를 쓰면 업로드 JSON 도 초기 데이터와 똑같이 동작한다.

- 수용 기준:
  1) `lib/treeUtils.ts` 에 `collectDescendantIds(activities, rootId): Set<string>`(또는 배열) 을 추가하고 parentId 관계로 BFS/DFS 한다. `deleteActivity` 는 이 함수를 써서 대상+모든 하위를 지운다. 접두사가 맞아도 parentId 관계가 없으면 지우지 않고, 접두사가 안 맞아도 parentId 관계가 있으면 지운다.
  2) `app/admin/page.tsx onStatusChange` 의 하위 상태 전파(229-234행)도 같은 `collectDescendantIds` 를 쓴다. 부모 자동 진행(237-247)·자식 모두 완료 시 부모 완료(250-263)는 이미 parentId 기반이므로 손대지 않는다.
  3) 단위 테스트 `lib/treeUtils.test.ts`(신규, `node --test`, `'./treeUtils.ts'` 처럼 확장자 붙인 import — 기존 `lib/mail/*.test.ts` 관례): (a) 초기 데이터 `initialActivities` 에서 "2" 삭제 → "2","2-1"~"2-4" 만 빠지고 나머지 15개 유지(기존 동작 회귀 없음); (b) 임의 id 트리(`net`→`vlan`→`vlan-check`, 3단)에서 "net" 삭제 → 셋 모두 빠짐; (c) id 가 `1-` 로 시작하지만 parentId 가 다른 노드(`{id:"1-x", parentId:"2"}`)는 "1" 삭제 시 남음; (d) `collectDescendantIds` 가 순환 데이터에서도 무한루프 없이 끝남(visited 집합). 결과: `npm run test:unit` 이 기존 65건 + 신규 전부 통과.
  4) e2e: `e2e/activity-import.spec.ts` 에 같은 파일 안 새 `test(...)` 를 추가하거나 신규 `e2e/tree-ops.spec.ts` — `loginAsAdmin(request)` 패턴(`e2e/mail.spec.ts:27-31`, `POST /api/auth/login {pw, role:'admin'}`)으로 로그인 → 원본 activities 저장 → 임의 id 트리 PUT `{activities:[...]}` → `PUT {action:'delete', targetId:'net'}` 가 200 이고 GET 결과에 `vlan` 이 없음 → 마지막에 원본 activities 를 PUT 으로 복원(mail.spec 의 afterEach 관례). 실행 뒤 `git status` 가 깨끗해야 한다(e2e 는 `test-results/e2e-data/activity.json` 격리 파일 사용).
  5) `npm run lint`·`npx tsc --noEmit` 통과. 문서(README/ADMIN_GUIDE)는 사용자 관점 동작이 바뀌지 않으므로 갱신 불필요 — 단, ADMIN_GUIDE 에 "id 는 `부모-번호` 형식이어야 한다" 는 문장이 있는지 `grep -n "접두\|부모ID\|<부모" docs/ADMIN_GUIDE.md` 로 확인하고 있으면 그 제약을 지운다(미확인).

- 건드릴 파일:
  - `lib/treeUtils.ts:64-68` `deleteActivity` — `collectDescendantIds` 신설 후 이를 사용. `addActivity` 의 `${parentId}-${Date.now()}` id 생성은 그대로 둔다(표시용 관례일 뿐, 이제 의미를 갖지 않음).
  - `app/admin/page.tsx:229-234` `onStatusChange` — `startsWith` 를 `collectDescendantIds(activities, id)` 로. import 는 파일 상단 기존 `@/lib/treeUtils` import 에 추가(있는지 확인; 없으면 `buildTree` import 줄 참조).
  - `lib/treeUtils.test.ts` — 신규.
  - `e2e/activity-import.spec.ts` 또는 `e2e/tree-ops.spec.ts` — 신규 test 1건.

- 검증 명령:
  - `npm run lint`
  - `npx tsc --noEmit`
  - `npm run test:unit` (node --test, 현재 65건 통과 — 2026-09-19 실측)
  - `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e` (현재 8건 통과, 약 18초; 이 환경엔 ms-playwright 브라우저가 없어 env 필수)
  - `git status --porcelain` 이 비어 있어야 함(e2e 뒤).

- 위험과 피할 것:
  - `app/api/activities/route.ts` 의 PUT 분기 순서·`validateActivityImport`·`writeActivityData`·`after(notifyActivityChanges)` 는 건드리지 않는다(메일 이벤트 감지가 저장 전후 diff 를 보므로 삭제 결과가 달라지면 그 테스트가 영향받을 수 있음 — `lib/mail/events.test.ts` 가 계속 통과하는지 확인).
  - `components/ActivityTree.tsx:122` 와 `admin/page.tsx:285` 의 삭제 확인창 중복(두 번 뜸)은 **이번 범위 밖** — 같이 고치면 e2e 의 `page.once('dialog')` 계열 가정이 바뀔 수 있다. 차선 후보로 남긴다.
  - PUT 실패를 화면에 표시하는 일도 범위 밖(별도 아이디어).
  - auth(`lib/adminSession.ts`, `app/api/auth/**`)·`proxy.ts`·`Dockerfile` 은 손대지 않는다.
  - 순환 parentId 는 validateActivityImport 가 저장 단계에서 거부하지만, `collectDescendantIds` 는 방어적으로 visited 집합을 둔다(무한루프 금지).
  - 하위 찾기를 `buildTree` 결과의 `children` 을 타는 방식으로 하지 말 것 — `onStatusChange` 는 평면 배열 `data.activities` 를 다루므로 평면 배열 + parentId 인덱스(Map<parentId, Activity[]>)로 구현하는 편이 두 경로에 똑같이 맞는다.

- 차선 후보: 관리자 콘솔 PUT 실패(400/401)를 화면에 표시하고 삭제 확인창 중복(ActivityTree.tsx:122 + admin/page.tsx:285) 제거 (가치 3 / 위험 1 / S) — TrackingSettings/MailSettings 의 Feedback 패턴을 재사용. 1순위가 어떤 이유로 성립하지 않을 때(예: 이미 parentId 기반으로 바뀌어 있음) 고를 것.

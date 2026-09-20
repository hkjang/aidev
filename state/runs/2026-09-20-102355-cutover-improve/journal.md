# 회차 노트 2026-09-20-102355-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:23] base pinned — main@fb951ea
- [러너 10:23] autonomy release — 

## 정찰 노트
- 선택 이유: parentId 기반 하위 탐색은 09-11 부터 5회차 연속 보류된 실제 버그(업로드 JSON 트리 삭제가 400 으로 조용히 실패)이며, 오늘 `lib/treeUtils.ts:66`·`app/admin/page.tsx:230` 에서 startsWith 가 그대로임을 확인했다. 지난 회차(09-19)는 "수정 과제" 형식이 no-change 로 끝났으므로 이번엔 코드 변경이 분명한 S 과제를 골랐다. Dockerfile 기본값 제거(4/2/S)는 하위 호환·위험 구역이라 제쳤다.
- 확신 없는 곳: ADMIN_GUIDE 에 "id 는 부모-번호 형식" 류 제약 문장이 있는지는 미확인(과제서에 grep 으로 확인하라고 적음). `lib/mail/events.ts` 가 삭제된 노드를 어떻게 diff 하는지는 읽지 않았다 — 기존 65건 단위 테스트가 회귀를 잡아 줄 것으로 가정.
- 조심할 것: 삭제 확인창 중복·PUT 실패 표시는 범위 밖(차선 후보). e2e 는 시스템 Chrome 경로 env(`PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome`) 필수, 실행 뒤 git status 확인. 프로필은 1일 전 것이 현재 코드와 일치해 다시 쓰지 않았다.
- [러너 10:26] scout done — 삭제·상태 전파의 하위 탐색을 ID 접두사 대신 parentId 기반으로 바꾸고 단위·e2e 테스트로 고정 (가치 4 / �

## 구현 노트
- 무엇/왜: `deleteActivity` 와 admin `onStatusChange` 의 하위 탐색을 `id.startsWith(부모+'-')` 에서 parentId 기반 `collectDescendantIds`(lib/treeUtils.ts, BFS+visited) 로 바꿨다 — 업로드 JSON 의 임의 id 트리에서 삭제가 400 으로 조용히 실패하고 상태가 자식에 안 내려가던 버그. 커밋 a24c58d, 단위 7·e2e 2 추가, ADMIN_GUIDE 4.3·6장 접두사 제약 문장 제거.
- 확신 없는 곳: `treeUtils.ts` 의 import 를 `import type { Activity } from './types.ts'` 로 바꿨다(node --test 로드용, lib/mail 관례). tsc·lint·next dev(e2e)·`npm run build` 모두 통과(build 는 커밋 뒤 같은 트리에서 실행, exit 0). 남은 미검증: 실제 훈련 데이터 규모(수천 건)에서의 성능은 재지 않았으나 BFS 는 O(n).
- 검증 방식: 단위·e2e 모두 옛 로직으로 되돌려 실패(단위 2건, e2e 2건)를 본 뒤 복원. 최종 트리에서 lint 0 / tsc / unit 72 / e2e 10 통과, git status 깨끗함.
- 일부러 안 한 것: 삭제 확인창 중복(ActivityTree.tsx:122 + admin/page.tsx:285), PUT 실패 표시, 부모 자동 진행·자식 전부 완료 규칙(이미 parentId 기반) — 과제서 범위 밖. `addActivity` 의 `${parentId}-${Date.now()}` id 도 그대로.
- 조심할 것: e2e 는 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome` 필수. **e2e 가 하나라도 실패해 trace 가 남으면 그 뒤 `npm run lint` 가 `playwright-report/trace/*.js` 때문에 3천 건 오류를 낸다**(이번에 실측) — 소스 문제가 아니며 다음 성공 실행이 보고서를 덮으면 사라짐. `e2e/tree-ops.spec.ts` 의 UI 테스트는 트리 상태 드롭다운을 `select` 중 `option[value="완료"]` 가 있는 것으로 골라내는데(설정 카드의 select 제외), 트리 렌더 순서(net, vlan, vlan-check, db, net-x)에 기댄다.
- [러너 10:33] brief accepted — 채택 — 과제서의 근거(treeUtils.ts:66·admin/page.tsx:230 의 startsWith, validateActivityImport 의 400)가 코드와 정확히 일치했고 수용 �
- [러너 10:33] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- 확인: diff 5파일 전부 읽음. deleteActivity 를 옛 startsWith 로 되돌려 새 단위 테스트 2건이 실패하는 것을 직접 봤고(복원 후 72 통과), tsc·lint·e2e tree-ops 2건도 이 트리에서 직접 실행해 통과, git status 깨끗함. validateActivityImport 가 중복 id·자기참조·순환을 막아 BFS 의미가 잘 정의됨을 확인.
- 못 본 것: 전체 e2e 10건은 tree-ops 2건만 돌렸고(구현자 기록 신뢰), npm run build 는 재실행하지 않음. 수천 건 규모 성능은 미측정(O(n), 상한 5,000 이라 무시).
- 보안·법무: delete/status 경로는 여전히 isAdminRequest 뒤에만 열리고 새 입력·개인정보 없음 → 차단 없음.
- 릴리즈 노트: "업로드한 JSON 의 임의 id 트리에서 상위 삭제·상태 변경이 하위로 내려간다" 가 사용자 가시 변화. ADMIN_GUIDE PDF 는 md 와 어긋난 채 남음.
- 다음 회차 후보: admin/page.tsx:285 삭제 확인창이 클릭한 항목 대신 루트 node.title 을 보여 주는 기존 결함 + ActivityTree.tsx:122 와 중복 확인창; e2e UI 테스트는 select 개수 5·렌더 순서에 기대 관리 화면에 select 를 더하면 깨짐.
- [러너 10:35] review approved — 리뷰 승인 (risk=low)
- [러너 10:35] pr created — https://github.com/hkjang/cutover/pull/5
- [러너 10:36] ci passed — 검사 없음 — 정책으로 허용
- [러너 10:36] merge done — a24c58d

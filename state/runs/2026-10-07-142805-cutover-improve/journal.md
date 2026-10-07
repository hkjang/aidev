# 회차 노트 2026-10-07-142805-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:28] base pinned — main@51ffd03
- [러너 14:28] autonomy release — 

## 정찰 노트
- 선택: `action:'add'` 분기만 10-06 의 대상 존재 검사에서 빠진 것(route.ts:90). 구현·비평이 이미 세 번 통과시킨 패턴의 마지막 한 조각이라 프로덕션 1파일로 끝나고 빨간 테스트가 결정적이다. 제친 후보 — 경계 테스트 보강(프로덕션 변경 0, 가치 2 → 차선으로 남김), 전파 통합(M, 두 계약 비교 필요), Dockerfile 비밀번호(가치 4지만 보호 경로 + 배포 검증 불가로 소규모 회차 부적합), level 50(정책 판단 필요).
- 확신 있는 것(직접 읽어 확인): route.ts:90 에 존재 검사 없음, activityData.ts:262-264 부모 존재 이슈 → :308 `issues.length > 0` throw → 400 `INVALID_DATA`, 배너 문구는 activityData.ts:43 의 `activity.json 데이터 형식이 올바르지 않습니다.`, UI 경로는 ActivityTree.tsx:108 → admin/page.tsx:357, 숫자 parentId 는 admin-put-contract.spec.ts:78 이 400 을 기대.
- **과장하지 말 것**: 이것은 조용한 200 이 아니라 **이미 400 으로 거절되는** 요청이다. 가치는 데이터 보호가 아니라 오류 분류(400→404)와 관리자가 읽는 문구다. 10-06 처럼 "조용한 저장 성공" 으로 쓰면 비평에서 근거 불일치로 걸린다.
- 추측으로 남긴 것(미확인): 이 워크트리에 `node_modules` 가 없어 이번 회차에 테스트를 **한 번도 실행하지 못했다** — 단위 95 / e2e 38 기준선과 lint·build exit 0 은 모두 과거 회차 기록이다. `e2e/admin-missing-target.spec.ts` 의 `expectNotFoundAndUnchanged` 위치(`:61` 부근)와 ADMIN_GUIDE 303행도 과거 기록 기준이며 이번에 열지 않았다.
- 조심할 것: base 는 main@51ffd03 이라 10-07 의 UUID ID 수정(16c3384)이 **들어 있지 않다** — `addActivity:49` 는 아직 `${parentId}-${Date.now()}` 다. ID 생성과 `lib/treeUtils.ts` 에는 손대지 말 것(그 과제와 충돌 + 단위 기대값 변경 0건 유지). 검사를 `Array.isArray(body.activities)` 분기 위로 올리면 정상 전체 배열 PUT 이 404 가 된다.
- [러너 14:32] scout done — 없는 `parentId` 로 하위를 추가하면 "activity.json 데이터 형식이 올바르지 않습니다" 400 이 나오는 것을 404 `TAR

## 구현 노트
- 무엇/왜: `app/api/activities/route.ts` add 분기에만 빠져 있던 대상 존재 검사를 넣어, 없는 `parentId` 로 하위를 추가하면 400 `INVALID_DATA`("activity.json 데이터 형식이 올바르지 않습니다.") 대신 다른 세 분기와 같은 404 `TARGET_NOT_FOUND` 가 나간다. 조용한 200 이 아니라 **오류 분류와 문구**를 고친 것이다.
- 확신 있는 것: 수정 전 빨간 출력에 400 본문이 그대로 남았고(400 `INVALID_DATA` + `부모 id "..."가 activities에 존재하지 않습니다`) 수정 후 7 passed. tsc/lint/build exit 0, 단위 95 pass, 전체 e2e 40 passed(기준선 38 + 신규 2) 모두 이 세션에서 직접 실행했다.
- 확신 없는 것: UI(➕ 버튼) 경로로 404 배너가 실제로 뜨는 것은 **테스트하지 않았다** — `submitActivityChange` 가 `res.ok` false 에 서버 `error` 를 배너에 넣는 것은 코드로만 확인(10-06 과 동일 경로). 신규 테스트는 모두 API 계약 레벨이다.
- 일부러 하지 않은 것: `lib/treeUtils.ts`·`addActivity`·`validateActivityImport` 무수정(단위 기대값 변경 0건 + 10-07 UUID 과제와 충돌 회피). 숫자 `parentId` 400 테스트는 `e2e/admin-put-contract.spec.ts:78` 에 이미 있어 중복하지 않았다. level 50 부모 아래 추가가 내는 400 은 정책 판단이 필요해 ideas.json 으로 넘겼다.
- 다음 역할 주의: e2e 는 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome` 필요. `npm run test:unit` 은 글롭이 안 풀려 그 자체로 exit 1 — 명시 경로로 돌릴 것. `npm run lint` 는 E2E **전에**, 또는 `rm -rf playwright-report test-results` 뒤에.
- [러너 14:37] brief accepted — 채택 — 근거가 현재 코드와 정확히 일치했고(route.ts:90 에 존재 검사 없음, activityData.ts:262-264 → :308 throw → 400 `INVALID_DATA`
- [러너 14:37] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- 확인: `git diff 51ffd03...HEAD` 3파일(route.ts:91-97, ADMIN_GUIDE 303, admin-missing-target.spec 신규 2개)만. 직접 실행 — tsc exit 0, `node --test lib/*.test.ts` 30 pass, 대상 spec 7 passed, 회귀 admin-put-contract·tree-ops·admin-empty-tree 9 passed, 산출물 삭제 후 lint exit 0.
- 빨간색 재현: `addActivity(base,'no-such-id')` → `validateActivityImport` 가 `400 INVALID_DATA`(`부모 id "no-such-id"가 ...`) 를 throw 하는 것을 직접 실행해 확인 — 신규 테스트는 수정 전 반드시 실패했고 커밋 문구도 과장 없음(수정 전에도 쓰기는 없었다).
- 승인 후 남는 우려(릴리즈 노트): `parentId: ""` 가 200(최상위 생성) → 404 로 바뀌었다. UI 는 `node.id`/`null` 만 보내므로 화면 영향 없음. 또 존재하는 부모라도 id 120자·level 50 초과면 여전히 400 '데이터 형식' 문구 — add 분기 분류는 절반만 정리됨(ideas.json 과 일치).
- 못 본 것: e2e 13 spec 중 4개만 실행(mail·tracking 미실행), `npm run build` 미실행, UI ➕ 버튼의 404 배너는 구현자와 같이 코드만 확인.
- 보안·법무: 차단 없음 — `isAdminRequest` 뒤 기존 관리자 경로 안, 새 권한·비밀값·개인정보·의존성 없음, 404 본문의 parentId 되비춤은 저장 안 되고 React 텍스트 렌더라 공격 경로 없음.
- [러너 14:41] review approved — 리뷰 승인 (risk=low)
- [러너 14:41] pr created — https://github.com/hkjang/cutover/pull/16
- [러너 14:42] ci passed — 검사 없음 — 정책으로 허용
- [러너 14:42] merge done — 563ae4a
- [러너 14:49] release published — v1.16.0
- [러너 14:49] gh-release created — GitHub Release v1.16.0
- [러너 14:49] manifest failed — 누락/불량: cutover-v1.16.0.tar.gz(too-small)

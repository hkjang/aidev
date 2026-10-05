# 회차 노트 2026-10-06-033756-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:38] base pinned — main@c304e91
- [러너 03:38] autonomy release — 

## 정찰 노트
- 없는 `targetId` 의 delete/move/상태변경 404 를 골랐다: 10-01(UNSUPPORTED_ACTION)·10-05(STALE_SNAPSHOT)가 메운 "조용한 200 성공" 구멍의 마지막 칸이고, `route.ts` 1파일에서 끝나며 `treeUtils.test.ts:24,55` 가 고정한 순수 함수 계약을 건드리지 않는다. 10-05 가 차선으로 남긴 항목이라 두 회차 연속 살아남았다.
- 제친 후보: 상태 전파 이전(M·`expectedLastUpdated` 계약 재설계가 붙어 파일이 번진다), 자기참조 중복(테스트 기대값 동반 변경이라 단독 회차가 더 깔끔 — 차선으로 남김), 리비전 도입(파일 6개 상한에 닿는다), `test:unit` 글롭(검증 관문 자체를 바꾸는 일이라 분리), eslint 산출물(09-29 접근이 반려돼 다른 방향이 필요).
- 추측으로 적은 것: ① 404 냐 400 이냐는 내 판단이다(분기는 성립했고 대상만 없으므로 404 권장, 충돌하면 400 으로 내려도 수용 기준은 만족). ② 404 때 클라이언트 `mutate()` 가 필요한지 안 봤다 — 1초 폴링에 맡기고 프로덕션 1파일을 유지하라고 적었다. ③ `docs/ADMIN_GUIDE.md` 5.6 절 표 모양 미열람.
- **검증을 하나도 실행하지 못했다**: 이 정찰 워크트리에 `node_modules` 가 없고(`node_modules/next/package.json` 글롭 0건) `node --test` 실행도 승인에 막혔다. 과제서의 모든 명령은 미실행 — 구현자는 `npm ci --legacy-peer-deps` 부터 하고, 404 기대 e2e 가 **수정 전 200 으로 실패하는 것을 먼저 보고** 저널에 붙여넣기.
- 구현자가 조심할 것: 존재 확인을 `Array.isArray(body.activities)` 분기보다 위로 올리면 `admin/page.tsx:332-347` 의 두 PUT(targetId 를 함께 보낸다)이 404 가 되어 e2e 3개 파일이 깨진다. 그리고 e2e 뒤 `rm -rf playwright-report test-results` 없이는 `npm run lint` 가 실패한다(globalIgnores 에 없음 — 09-29 커밋 `0de2f1e` 는 main 미머지).
- [러너 03:42] scout done — 존재하지 않는 `targetId` 의 delete / move / 상태변경 PUT 이 조용한 200 "저장 성공" 을 돌려주는 것을 404 로 막기

## 구현 노트
- 무엇/왜: `app/api/activities/route.ts` PUT 의 세 분기(`delete`/`move`/`targetId+newStatus`) 안에 대상 존재 확인을 넣고, 없으면 404 `{ error, code:'TARGET_NOT_FOUND' }` 로 조기 반환한다(저장·`after()` 메일까지 가지 않음). 그전에는 아무 변화 없이 같은 배열을 다시 저장하고 새 `lastUpdated` 와 200 이 나가, 이미 지워진 행을 건드린 관리자에게 배너 없이 "저장됐다" 로 보였다. 프로덕션 1파일 + 신규 e2e 1파일 + `ADMIN_GUIDE.md` 5.6 절 한 문장 = 3파일.
- 실패 재현(먼저 봄): `npx playwright test e2e/admin-missing-target.spec.ts` → `3 failed / 2 passed`, `Expected: 404 Received: 200` (at `:61`). 응답 본문에 바뀌지 않은 activities 와 새 lastUpdated 가 그대로 보인다. 수정 후 `5 passed`.
- 확신 없는 곳: ① 404 vs 400 은 판단이다 — 분기는 성립했고 대상만 없으므로 404 를 택했다. 기존 테스트와 충돌은 없었다. ② 오류 문구가 요청자의 `targetId` 를 그대로 끼워 넣는다 — 관리자 전용 JSON 응답이고 React 가 이스케이프하지만, 아주 긴 id 가 배너를 밀 수 있다(ideas.json 에 후보로 남김). ③ UI 에서 404 를 실제로 눌러 본 적은 없다 — e2e 는 `request.put` 으로 API 계약만 증명한다. 배너 경로는 `submitActivityChange` 의 기존 `!res.ok` 분기를 코드로 읽어 확인한 것이다.
- 일부러 안 한 것: 클라이언트 `mutate()` 추가 안 함(1초 폴링이 지운 행을 치운다 → 프로덕션 1파일 유지). `lib/treeUtils.ts` 안 건드림(`treeUtils.test.ts:24,55` 가 고정한 순수 함수 무변화 계약은 옳다 → 단위 기대값 변경 0건). 존재 확인을 `Array.isArray` 분기 위로 올리지 않음(올리면 `admin/page.tsx:332-347` 의 정상 저장이 404 가 된다 — 보호 테스트로 고정했다). 대상이 있는데 이미 맨 위/맨 아래인 `move` 는 그대로 200.
- 다음 역할이 조심할 것: e2e 기준선은 **33** 이다(프로필의 32 는 `9da830b` 가 `admin-stale-overwrite.spec.ts` 에 더한 4번째 테스트 이전 수치). 전체 38 passed. `npm run test:unit` 글롭은 이 환경에서 조용히 0건이므로 `node --test lib/*.test.ts lib/mail/*.test.ts lib/tracking/*.test.ts`(95 pass)로 돌릴 것. e2e 뒤 `rm -rf playwright-report test-results` 없이 `npm run lint` 하면 실패한다.
- [러너 03:51] brief accepted — 채택 — 근거가 현재 코드와 정확히 일치했고(`route.ts:79,81,87` 세 분기, `treeUtils.ts:27,91` 의 무변화 반환, `admin/page.tsx:76` 의 
- [러너 03:52] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- 확인: route.ts 를 base 로 되돌려 신규 spec 이 `3 failed / 2 passed` (`Expected: 404 Received: 200` at `:61`) 로 떨어지는 것을 직접 재현했고 HEAD 에서 5 passed. 전체 e2e 38 passed, 단위 95 pass, `tsc --noEmit` 0, 산출물 삭제 뒤 `lint` 0, 트리 clean. 존재 확인 3곳이 `Array.isArray` 분기 아래에 있어 `admin/page.tsx` 의 정상 저장을 깨지 않는 것도 코드·테스트로 확인.
- 못 본 것: UI 에서 404 배너를 실제로 눌러 보지는 않았다(구현자와 동일 — e2e 는 `request.put` 계약만 증명). `lib/activityData.ts` 는 고아 검사 줄만 읽었다.
- 승인이어도 남는 우려 ①: 커밋 메시지가 '상태 버튼' 도 조용한 200 이었다고 쓰지만, UI 상태 변경은 `admin/page.tsx:331-337` 에서 배열+`expectedLastUpdated` 를 보내 이미 409 로 막힌다. `route.ts:102` 404 는 UI 미도달 — **릴리즈 노트는 '삭제·이동 버튼 + 직접 API 호출자' 로 범위를 좁혀 쓸 것.**
- 우려 ②: `ADMIN_GUIDE.md:303` 에 배열 분기 우선 규칙이 없어, 배열과 `action:'delete'` 를 함께 보내면 404 가 아니라 200 이다(의도된 동작, e2e `:100` 이 고정). 다음 회차 한 줄 보완 후보.
- 보안·법무 차단 없음: 관리자 관문 뒤, 새 수집·의존성·비밀값 없음. 되비추는 `targetId` 는 JSON + React 이스케이프로 공격 경로 없음(길이 상한만 없음).
- [러너 03:56] review approved — 리뷰 승인 (risk=low)
- [러너 03:57] pr created — https://github.com/hkjang/cutover/pull/14
- [러너 03:57] ci passed — 검사 없음 — 정책으로 허용
- [러너 03:57] merge done — c186712
- [러너 04:03] release published — v1.15.0
- [러너 04:03] gh-release created — GitHub Release v1.15.0
- [러너 04:03] manifest ok — cutover-v1.15.0.tar.gz 
- [러너 04:03] assets uploaded — 1개
- [러너 04:03] assets verified — v1.15.0 자산 1개 (이전 v1.14.0: 1)

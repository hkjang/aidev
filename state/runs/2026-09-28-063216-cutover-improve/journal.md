# 회차 노트 2026-09-28-063216-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:32] base pinned — main@105aa9e
- [러너 06:32] autonomy release — 

## 정찰 노트
- 이번 과제: `validateActivityImport` 단위 테스트 신규(`lib/activityData.test.ts`). 지난 3회차가 모두 관리자 UI 였고, 지금 최대 공백은 두 라우트(`activities/route.ts:81`, `activities/import/route.ts:77`)가 공유하는 유일한 검증 관문에 테스트가 0건이라는 점이다. 프로덕션 변경 0 → 위험 1.
- 제친 후보: 상태 전파 `propagateStatus` 통합(3/2/M, 같은 값을 두 경로가 다르게 읽는 진짜 문제지만 클라이언트+API+테스트를 동시에 건드려 M 상한이고 09-20 에 손댄 자리라 단독 회차가 낫다) / eslint globalIgnores(2/1/S, 가치가 낮아 차선으로) / Dockerfile 비밀번호 기본값(4/3/M, 보호 경로·릴리즈 확인 필요).
- 확신 없는 곳: **`node_modules` 가 없어 이번 회차에 lint/tsc/test:unit/e2e/build 를 하나도 실행하지 못했다.** "기존 79건" 은 09-27 기록에서 가져온 숫자이며 직접 확인하지 않았다. `npm ci --legacy-peer-deps` 를 먼저 돌려 기준선을 찍고 시작할 것.
- 구현자 주의: `lib/activityData.ts` 는 절대 수정하지 말 것(같은 파일에 원자 쓰기·백업). 테스트 중 실제 버그를 찾으면 고치지 말고 현재 동작을 고정하고 요약에만 적어라. `readActivityData`/`writeActivityData` 호출 금지(실제 `data/activity.json` 을 덮어쓴다).
- 확인한 것: 기존 lib 테스트 2개가 `from './treeUtils.ts'` 처럼 `.ts` 확장자를 쓴다 / `MAX_VALIDATION_ISSUES`(50)는 export 안 됨 / `normalizedLegacyStatuses` 는 importer UI 가 사용자에게 건수로 보여준다.
- [러너 06:37] scout done — `validateActivityImport` 단위 테스트 신규 추가 (`lib/activityData.test.ts`) (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `lib/activityData.test.ts` 신규 1파일만 추가(프로덕션 변경 0). 두 라우트가 공유하는 유일한 입력 검증 관문 `validateActivityImport` 의 계약(trim·빈 배열·최상위 오류 2종·관계 오류별 path·'진행중' 건수·issues 상한 51)을 16건으로 고정. 커밋 1087213.
- 확신 없는 곳: 없음에 가깝지만 두 가지를 명시한다. (1) 순환 테스트는 a↔b 가 부모-level 규칙을 어떤 조합으로도 만족 못 해 level 오류가 함께 나므로, `issues.length` 대신 '순환' 메시지만 걸러 1건임을 단언했다. (2) e2e 와 `npm run build` 는 돌리지 않았다 — 프로덕션 변경이 0이라 과제서가 선택 사항으로 둔 항목.
- 범위 밖 발견(고치지 않음): `parentId === id` 는 과제서 예상(1건)과 달리 **issues 2건**이다. `continue`(activityData.ts:260)는 부모 관계 루프만 건너뛰고 순환 검사(:273-299)는 별도 패스라 `a → a` 를 순환으로 또 신고한다. 과제서 지시대로 프로덕션은 손대지 않고 현재 동작을 고정했다 — 다음 회차 후보(ideas.json 에 등재).
- 일부러 안 한 것: `lib/activityData.ts` 수정(위험 구역인 원자 쓰기·백업과 같은 파일), `readActivityData`/`writeActivityData` 호출(실제 data/activity.json 을 덮어쓴다), 차선 후보인 eslint globalIgnores 추가(1순위가 성립).
- 다음 역할이 조심할 것: `npm run lint` 전에 `rm -rf playwright-report test-results` 필요(eslint globalIgnores 에 아직 없음). 검증은 전부 실행함 — test:unit **95건/30 suites**(기존 79/25 + 신규 16/5), 단독 `node --test lib/activityData.test.ts` 16건, `npx tsc --noEmit` 통과, lint 오류 0, `git status --porcelain` 은 신규 테스트 파일뿐이고 `data/` diff 없음.
- [러너 06:41] brief accepted — 채택 — 근거(두 라우트가 공유하는 단일 관문, `lib/activityData.test.ts` 부재, `.ts` 확장자 import 관례, 50/51 상한)가 코드와 일�
- [러너 06:42] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- 확인함: 16건 단언을 `lib/activityData.ts:120-315` 와 한 줄씩 대조(중복 id·부모 부재·level±1·자기참조 2건·a↔b 순환 1건·51 상한 모두 실제 동작과 일치) + 직접 실행 `test:unit` 95/30 전부 통과, `tsc --noEmit` 통과, `rm -rf playwright-report test-results` 후 `lint` 0, `git status --porcelain` 비어 있음.
- 원장 `- 실패 재현:` 유효 — 일부러 틀린 기대값으로 `not ok` 를 받았고, 같은 실행의 `2 !== 1` 이 과제서 기준 5) 의 오류를 드러냈다. 테스트가 대상 함수를 실제로 지나간다.
- 못 본 것: e2e 와 `npm run build`(프로덕션 변경 0이라 생략에 동의), `node_modules/next/dist/docs` 가이드(Next API 미사용).
- 승인 후 남는 우려: 이번 테스트는 `activityData.ts:257-260` 자기참조가 순환 검사에서 **두 번** 신고되는 중복을 '정상'으로 고정했다. 다음 회차가 이 중복을 고치면 `lib/activityData.test.ts:109-116` 도 함께 갱신해야 한다(릴리즈 노트에는 동작 변경 없음 — 테스트만 추가).
- 커버리지 공백(결함 아님): `MAX_ACTIVITY_COUNT` 상한·각 필드 길이 상한·`level>50`·원소 비객체, 그리고 프로필이 지적한 **`time`·`title` 미trim 비대칭**이 미고정 — 다음에 이 비대칭을 테스트로 박으면 의도/버그가 문서화된다.
- [러너 06:45] review approved — 리뷰 승인 (risk=low)
- [러너 06:45] pr created — https://github.com/hkjang/cutover/pull/8
- [러너 06:46] ci passed — 검사 없음 — 정책으로 허용
- [러너 06:46] merge done — 1087213
- [러너 06:50] release published — v1.11.0
- [러너 06:50] gh-release created — GitHub Release v1.11.0
- [러너 06:50] manifest ok — cutover-v1.11.0.tar.gz 
- [러너 06:50] assets uploaded — 1개
- [러너 06:50] assets verified — v1.11.0 자산 1개 (이전 v1.10.0: 1)

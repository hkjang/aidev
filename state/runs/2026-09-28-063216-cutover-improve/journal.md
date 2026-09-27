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

# PR 처리기 노트 2026-10-18-141323-workkit.irumx.app-shepherd — workkit.irumx.app PR #8
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-18-122852-workkit.irumx.app-improve)
# 회차 노트 2026-10-18-122852-workkit.irumx.app-improve — workkit.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:29] base pinned — main@aed4f5d
- [러너 12:29] autonomy release — 
- [러너 12:35] scout done — 주간보고 간단 입력이 "진행 70%" 같은 진척률 부분을 읽지 못해 상태도 진척률도 버려지는 것 고치기 (가�

## 구현 노트
- 무엇·왜: `parseSimpleLines`(packages/tools/src/weekly/model.ts)의 상태 단어 분기 **바로 뒤**에 앵커된 퍼센트 분기(`PROGRESS_PART = /^(진척|진행|달성)?\s*(\d{1,3})\s*%$/`, 값 0~100, `task.progress === null` 일 때만)를 넣었다. 접두사 `진행` + `status === null` 이면 `'doing'` 까지 세우고 읽은 방식을 `notes` 에 한 줄 남긴다. 프로덕션 1파일, 테스트 1파일.
- 확신 없는 곳: 노트 문구를 과제서 예시(`'진행 70%' → 진척률 70%`)보다 **한 조각 길게** 냈다 — 상태도 같이 세웠으므로 `'진행 70%' → 진척률 70%, 상태 진행`. 둘을 다 읽었으니 둘을 다 적는 쪽이 정직하다고 봤지만 문구 자체는 취향 문제다. 또 같은 줄에 퍼센트 부분이 **두 개**면 두 번째는 `thisWeek` 로 간다(의도).
- 검증 못 한 것: 브라우저 경로를 돌리지 않았다 — `tools-flows`·`ui-flows`·`a11y`·`e2e` 는 과제서가 돌리지 말라고 했고 외부 의존이 필요하다. `WeeklyTool.tsx` 가 `parseSimpleLines` 를 그대로 쓰는 것은 코드로 확인했을 뿐 화면으로는 못 봤다.
- 일부러 안 한 것: `DEFAULT_TEMPLATE` 의 `progress` 열을 켜지 않았다(기본 `enabled: false`, model.ts:88) — 표 열 기본 구성 변경은 별 과제이고 docx/xlsx 단언에 닿는다. 그래서 **파싱한 진척률은 업무 편집 화면과 사용자가 열을 켤 때만 보인다**(ideas.json 에 후속 후보로 적어 뒀다). `WeeklyTool.tsx`·`table/*`·`compare/*`·`.github/workflows/**` 는 열지 않았다.
- 다음 역할이 조심할 것: `SAMPLE_WEEKLY_LINES`(samples.ts:7)에 `%` 가 **없어서** 기존 단언은 한 줄도 갱신하지 않았다 — `outputs add nothing` 의 `not.toMatch(/%/)` 도 그대로 통과한다. 새 테스트 3개 중 하나(`120%`·`7.5%`·`%`·`70% 이상` → `thisWeek`)는 **고치기 전부터 녹색**인 회귀 가드다(바꾸지 말아야 할 동작을 고정). 단위 시험만으로 돌아가니 DB·Playwright 는 필요 없다.
- [러너 12:39] brief accepted — 채택 — 근거(`STATUS_WORDS` 가 완전 일치만 보고 `progress` 필드·표 열은 이미 있다)가 지금 코드와 정확히 맞았고, 지정한 분
- [러너 12:39] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인: base aed4f5d 기준 변경은 2파일 43줄. `npm run check` 재실행 녹색(11파일 138 passed = 기준선 135 + 새 3). 원장 '실패 재현' 이 증상과 일치하고 새 테스트 2개는 수정 전 반드시 실패하는 경로를 지난다(세 번째는 구현자가 밝힌 회귀 가드, 수정 전부터 녹색).
- 경계 조건 통과: `0%`(`=== null` 가드·cellOf·NumberInput 모두 0 을 제대로 다룸), 범위 밖 값은 분기가 거짓이 되어 `checkReport` 의 0~100 오류를 파서가 만들 수 없음, 앵커 정규식이라 `예정`/`대기`/`요청`/`필요` 판정을 가로채지 못함.
- 판정: approve · risk low · 차단 없음(브라우저 순수 파서, 인증·결제·마이그레이션·개인정보·의존성 무관).
- 남는 우려(릴리즈 노트에 적을 것): `DEFAULT_TEMPLATE` 의 진척률 열이 `enabled: false`(model.ts:88)라 읽어 낸 70 은 기본 요약문·표·docx 에서 사라진다 — 고치기 전엔 `thisWeek` 로 요약문에 보였다. "진척률은 표 열을 켜야 보인다" 를 명시하고, 열 기본값 켜기를 다음 회차 후속으로.
- 다음 회차가 알 것: `업무 / 진행 70% / 완료` 는 이제 doing 이 이기고 `완료` 가 thisWeek 로 간다(모순 입력, 결함 아님). `carryOver`(:250)가 progress 를 복사하므로 가져온 업무가 지난주 숫자를 들고 온다. 브라우저 경로(tools·ui·a11y·e2e)는 외부 의존이라 안 돌렸다.
- [러너 12:43] review approved — 리뷰 승인 (risk=low)
- [러너 12:43] pr created — https://github.com/hkjang/workkit.irumx.app/pull/8
- [러너 12:43] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 틀린 지적: CI `check` 실패는 이 변경 탓이 아니다. 3초 실패 + 로그 본문 없음 = 잡 미시작(Actions 결제 차단), 워크플로는 30분 타임아웃에 첫 단계가 checkout 이다. `gh` 미인증으로 잡 로그는 직접 못 봤다.
- 맞았던 지적: 없음. 비평 노트는 이미 approve 였고 코드에서 결함을 찾지 못했다.
- 고친 방법: 없음 — 절대 규칙대로 커밋하지 않았다. 대신 로컬 게이트로 녹색을 증명: `npm run check` 11파일 138 passed, `npm run build` 성공, `npm run kits` ×2 manifest `cmp` 일치 + `kits:test` 3종 통과.
- 인과 확인: `weekly/model.ts` 만 origin/main 으로 되돌리면 새 테스트 2개가 실패하고 복구하면 통과 — 테스트가 대상을 실제로 실행한다.
- 확신 없는 곳: `e2e`·`tools-flows`·`ui-flows`·`a11y` 는 Docker PostgreSQL·Playwright·가짜 서버(:18997/:18998/:8797)가 필요해 돌리지 않았다. 변경이 브라우저 순수 파서 1파일이라 영향 밖으로 본다.

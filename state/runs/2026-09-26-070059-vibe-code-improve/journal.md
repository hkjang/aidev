# 회차 노트 2026-09-26-070059-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:01] base pinned — main@08b3c09
- [러너 07:01] autonomy release — 

## 정찰 노트
- 차선이던 "plans.ts 테스트 보강"보다, 같은 파일에서 찾은 **실제 데이터 손실**(archiveDonePlans:275 의 unlinkSync — 같은 이름 보관본을 말없이 삭제)을 골랐다. 순수 테스트 추가보다 가치가 높고, 같은 커밋에서 plans.ts 의 첫 테스트 파일도 함께 생긴다.
- 보류 아이디어 중 "goal-metrics 의 exitCode undefined 집계"는 기각했다 — 유일한 기록 경로 verification.ts:115 가 number|null 만 쓰므로 프로덕션 배선에서 도달하지 않는다(손으로 만든 감사 줄로만 재현 가능).
- 추측으로 적은 것(미확인): readme/docs 에 보관 충돌 동작을 설명한 문장이 있는지 확인하지 않았다. 또 `npm ci`/`npm run check` 를 이번 세션에서 실제로 돌리지 않았다(워크트리에 node_modules 없음, 예산 제한) — 기준선 53 tests 는 지난 회차 기록에서 가져온 값이다.
- 구현자가 조심할 것: tests/unit/vscode-stub.ts 의 `commands` 는 빈 객체다. 명령을 테스트에서 호출하려고 스텁을 키우지 말고, checkpoints.ts 선례처럼 로직을 vscode 밖 exported 함수로 빼서 mkdtemp 실제 디렉터리로 검증할 것.
- 프로필은 3일 전 것이 지금 코드와 일치해(구조·명령·검증 함정 모두 확인됨) 새로 쓰지 않았다.
- [러너 07:03] scout done — `archiveDonePlans` 가 같은 이름의 보관 계획을 말없이 지우는 것을 막고, 보관 로직을 테스트 가능한 함수로 �

## 구현 노트
- 무엇/왜: `archiveDonePlans` 가 충돌 시 `unlinkSync` 로 기존 보관본을 지우던 것을 없애고, 보관 루프를 vscode 밖 exported `archiveDonePlans(plansDir, archiveDir, stamp)` 로 분리했다. 충돌 시 `<이름>-<stamp>.md` 로 보존하고, 스탬프는 `conflictStamp()`(KST 14자리)로 만들어 `restoreArchivedPlan` 과 같은 형식을 쓰게 했다.
- 확신 없는 곳: (1) 알림 문구에 새 이름을 나열하도록 바꾼 부분은 실제 VS Code 창에서 본 적이 없다 — 이름이 많으면 길어질 수 있다(리눅스라 smoke:vscode/extension-host 를 돌릴 수 없었다). (2) `restoreArchivedPlan` 의 스탬프 소스만 KST 로 교체했는데 그 경로 자체는 여전히 테스트가 없다(vscode 의존). 형식(숫자 14자리)은 `conflictStamp` 테스트로만 고정했다. (3) `freeName` 의 `-<stamp>-<n>` 재충돌 분기는 실제 시간이 아닌 고정 스탬프를 넘긴 테스트로만 증명했다.
- 일부러 안 한 것: 보관 후 `.active-plan` 포인터 갱신(과제서에서 범위 밖으로 지정, `selectPlanName` 의 `files.includes` 가드가 막음), `restoreArchivedPlan` 의 `-restored-<stamp>` 이름 형식 변경(docs 에 명시돼 있어 그대로 뒀다), plans.ts 의 나머지 순수 함수(planMeta/sortByPriority/linkedPlans) 테스트.
- 다음 역할이 조심할 것: `tests/unit/plans.test.ts` 는 `os.tmpdir()` 아래 실제 디렉터리를 만들고 `afterEach` 에서 지운다 — git 저장소나 vscode 는 필요 없다. `conflictStamp` 테스트는 Asia/Seoul 고정 변환을 단정하므로 로컬 타임존과 무관하게 돈다. `npm run check` 기준선은 53 → 63 tests 로 올라갔다.
- [러너 07:08] brief accepted — 채택 — 과제서의 근거(275줄 unlinkSync, restoreArchivedPlan 과의 정책 모순)를 코드에서 그대로 확인했고, 수용 기준 1~3 을 실패
- [러너 07:08] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인함: plans.ts:179/166/161 을 손으로 추적(unlinkSync 제거, freeName 루프 off-by-one 없음, conflictStamp 14자리 KST 고정) + 이 워크트리에서 `npm run typecheck` 오류 0, `npx vitest run` 63 tests 통과. 회귀 테스트는 구 코드에서 반드시 깨진다(기존 보관본을 지우므로 "지난 회차 작업" 단정 실패) — 통과-무관 테스트 아님.
- 구현자의 의심 3개 모두 검토: (1) 알림 문구 길이 → 결함 아님, 다만 충돌 많으면 잘릴 수 있어 notes 에 남김. (2) restoreArchivedPlan 무테스트 → 이번 변경이 만지는 건 스탬프 소스뿐이라 위험 낮음. (3) `-<stamp>-<n>` 분기 → 루프 추적으로 정확함 확인.
- 못 본 것: VSIX/extension-host/smoke(PowerShell 전용, 리눅스 불가), 실제 VS Code 창의 알림 렌더링.
- 보안·법무 차단 없음: 새 입력·의존성·비밀값·권한 없음, audit 은 여전히 파일명만 기록.
- 다음 회차가 알 것: 보관본을 덮어쓰지 않으므로 `archive/` 가 단조 증가한다(기본 이름 `current-plan.md` 반복 보관 시). retention/정리 명령이 다음 후보.
- [러너 07:10] review approved — 리뷰 승인 (risk=low)
- [러너 07:10] pr created — https://github.com/hkjang/vibe-code/pull/2
- [러너 07:13] ci passed — 검사 2개 모두 success
- [러너 07:13] merge done — fd141bd
- [러너 07:22] release published — v1.4.2
- [러너 07:22] gh-release created — GitHub Release v1.4.2

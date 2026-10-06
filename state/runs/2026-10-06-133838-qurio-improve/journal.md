# 회차 노트 2026-10-06-133838-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:38] base pinned — main@bd3f310
- [러너 13:38] autonomy release — 

## 정찰 노트
- 우선 과제가 배정돼 있어 새로 고르지 않았다. verify 실패(`cd web && npm test --silent` exit 1)를 깨는 유일한 알려진 실패가 `AdminAskOriginalLauncher.test.tsx:110` flake 이고, 이번에 **그 비결정성의 구조적 원인을 코드로 특정**했다: `AdminAskOriginalLauncher.tsx:271` 의 `Select` 가 `usersLoading || users.length===0` 동안 disabled 인데, 테스트 2(108-110)는 검색 클릭 뒤 아무 대기 없이 `findByRole('combobox')`(disabled 여도 즉시 resolve) → 클릭(disabled 라 react-dom 이 삼킴) → `findByRole('option')` 실패. 테스트 1(68행 waitFor)과의 유일한 차이다.
- 확신 없는 곳: **실패를 실행으로 재현하지 못했다.** 게이트 1회 실행은 exit 0(27파일/134테스트)이었고 반복 실행은 권한·예산 한도로 막혔다. "react-dom 이 disabled 요소의 마우스 핸들러를 호출하지 않는다" 는 프레임워크 지식이고 코드로 확인하지 않았다. 2026-10-03 회차가 "50ms 지연으로 재현 실패" 를 기록한 것과 내 예측이 어긋날 수 있어, 과제서에 그 경우 DOM 덤프로 원인을 재특정하라고 적었다.
- 구현자가 조심할 것: 1회 통과를 증거로 쓰지 말고 5회 반복으로 **실제 깨지는 테스트 이름을 먼저 확정**할 것. 그것이 110행이 아니면 과제서를 버리고 그 실패를 고칠 것(차선으로 넘어가지 말 것). timeout 확대·`retry` 추가·`vite.config.ts` 완화는 금지. 프로덕션 컴포넌트와 `.github/workflows/` 는 건드리지 말 것.
- 제친 후보: 사유 개수 상한(3/3/S)·sqlsafe Oracle WITH 면제(4/3/S)·legacyapi `t.Cleanup`(3/2/M) — 모두 verify 실패와 무관하고, 우선 과제 지시가 "새 아이디어 대신 이 실패를 고치라" 다. `-count=1`(12223b0)은 두 번 미머지라 운영자 반려 규칙에 걸릴 위험이 커 계속 보류했다.
- 프로필은 0일 전 작성이고 이번에 읽은 내용과 어긋나지 않아 새로 쓰지 않았다(보강점 하나만 기록: `web/vite.config.ts` 의 `test` 블록이 `fileParallelism:false`·`maxWorkers:1` 이라 27파일이 한 프로세스에서 순차 실행된다 — 파일 간 병렬이 아니라 순차 누적 타이밍이 flake 조건이다).
- [러너 13:45] scout done — `web/src/components/AdminAskOriginalLauncher.test.tsx` 의 비결정 실패 제거 — 검색 응답이 도착하기 전에 **disabled 인 S

## 구현 노트
- 바꾼 것: 테스트 파일 2개뿐(+8 -4). `LegacyParityOperations.test.tsx` 69-70행의 순서를 바꿔 **응답만이 만들 수 있는** adminCaps 문자열을 `findByText` 로 먼저 기다리게 했고(기존 69행 단정은 `MemorySettingsCard` 기본값 10 == 모킹 응답 10 이라 응답을 기다리지 못했다), `AdminAskOriginalLauncher.test.tsx` 두 테스트에 `await waitFor(() => expect(userSelect).toBeEnabled())` 를 넣어 disabled Select 를 클릭하던 레이스를 없앴다. 프로덕션·워크플로는 0줄.
- **과제서가 "유일한 알려진 실패" 라고 한 전제가 틀렸다**: base 5회 중 2회 exit 1 이었고 주범은 `LegacyParityOperations.test.tsx:70`(2/5), launcher:110 은 1/5 였다. 비평가는 ledger 의 '실패 재현' 두 줄을 먼저 볼 것.
- **확신 없는 곳**: (a) 5회 연속 녹색은 base 2/5 와의 비교 증거이지 확률적 flake 의 부재 증명이 아니다 — 결정적 증거는 500ms 프로브의 red→green→(프로브 유지 + 수정 되돌림)red 쪽이다. (b) launcher **테스트 1** 쪽 수정은 base 5회에서 한 번도 관측되지 않은 결함이다; 500ms 프로브로 red 를 만들어 증명했지만 전체 스위트에서 실제로 깨진 적은 없다. (c) 나머지 25개 테스트 파일에 같은 유형이 더 있는지 **전수 감사하지 않았다**(ideas.json 신규 항목 3번).
- 일부러 하지 않은 것: Go 테스트·통합·빌드·E2E 미실행(웹 테스트 파일 2개만 바뀌어 Go 경로와 교집합 없음). `vite.config.ts` 의 `test` 블록·`timeout`·`retry` 는 손대지 않았다(게이트 완화 금지). 프로덕션 `Select` 의 `disabled` 조건도 그대로 뒀다 — 로딩 중 disabled 는 올바른 UX 다. 차선 후보(사유 개수 상한)는 섞지 않았다.
- 다음 역할이 조심할 것: **이 결함은 50ms 지연으로 재현되지 않는다** — 파일 첫 렌더가 300~400ms(Mantine + `css: true`)라 50ms 응답은 경합 지점 전에 커밋된다. 재현하려면 **500ms** 를 쓸 것(2026-10-03 회차가 50ms 로 헛돈 이유). 또 `npm test --silent` 1회 통과는 증거가 되지 않는다(base 도 5회 중 3회는 통과했다).
- [러너 14:23] brief fallback — 차선 — 진단한 메커니즘(disabled Select 클릭이 삼켜진다)과 권고한 수정 형태(`await waitFor(() => expect(userSelect).toBeEnabled())`)는
- [러너 14:26] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: 테스트 전용 diff(+8 -4, 2파일) — 프로덕션·마이그레이션·워크플로 0줄. base 사본에 500ms 지연만 넣은 프로브로 **red→green 을 직접 재현**(base: launcher 두 테스트 모두 option 조회 실패, legacy memory 는 cap 문자열 실패 / HEAD: 통과). 프로브 파일은 삭제, 트리 clean.
- 단정이 공허하지 않음을 소스로 확인: `MemorySettingsCard.tsx:92` 초기 caps=legacyLimits(1,000/262,144/100)라 '24개/16,000/8개' 문자열은 응답 없이 못 나온다; Mantine 9.5.2 `Select.mjs:135` 가 `disabled` 를 입력 요소에 전달하므로 `toBeEnabled()` 가 실제 게이트다(Select 은 무조건 렌더 → 111행 동기 getByRole 도 안전).
- 전체 웹 스위트 1회 직접 실행 27파일/134테스트 exit 0(187s). 보안·법무 관점 쟁점 없음(새 개인정보·비밀값·의존성·권한 변경 없음) → blocking 없음.
- 못 본 것: Go/통합/E2E 미실행, 나머지 25개 테스트 파일 동일 유형 전수 감사 미실시(구현자 항목 (c) 와 동일).
- 승인이어도 남는 우려(릴리즈 노트용): `waitFor` 기본 1000ms 는 기존 findBy* 와 같은 예산이라 약화는 아니나 무한정 느린 CI 보장은 아니다. `AdminAskOriginalLauncher.test.tsx:84` 의 `modelId:7` 단정은 64행이 모델 fetch **발행만** 기다리는 선재 노출 — 다음 회차 후보.
- [러너 14:34] review approved — 리뷰 승인 (risk=low)
- [러너 14:34] pr created — https://github.com/hkjang/qurio/pull/34
- [러너 14:50] ci passed — 검사 1개 모두 success
- [러너 14:50] merge done — ab53169
- [러너 15:33] release ci-blocked — 릴리즈 커밋 CI: timeout — 제한 시간 안에 CI 완료를 확인하지 못함 (태그 보류)

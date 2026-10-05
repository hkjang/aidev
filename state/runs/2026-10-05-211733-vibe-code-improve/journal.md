# 회차 노트 2026-10-05-211733-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:17] base pinned — main@51148a2
- [러너 21:17] autonomy release — 

## 정찰 노트
- 골랐다: `마지막 갱신:` 나이 파서가 `goal-health.ts:24` 와 `goal-loop.ts:149` 에 글자까지 같은 두 벌로 있고 후자는 export 없이 테스트 0줄이라, 운영자가 되풀이한 "같은 값을 읽는 경로가 둘 이상이면 한쪽만 고치지 말 것" 에 정면으로 맞는다. 세 프로덕션 파일(`util/kst.ts`·`goal-health.ts`·`goal-loop.ts`)이 미머지 `origin/auto/*` 열 개 파일과 겹치지 않음을 `git log --name-only` 로 대조해 확인했다 — 다른 후보(`verification.ts`·`markdown.ts`·`goal-metrics.ts`·`command-audit.ts`)는 모두 미머지 중복이라 제쳤다.
- 미확인(과제서에 추측으로 적지 않았지만 구현자가 확인할 것): 이 워크트리에 `node_modules` 가 없어 **테스트를 한 번도 돌리지 못했다** — 기준선 테스트 수는 반드시 실측할 것. 차선 후보의 `latestVsixVersion` 파일명 계약은 `scripts/*.ps1` 을 읽지 못해 미확인이고, 계약이 같으면 그 후보는 성립하지 않는다. 원격 PR/CI 상태도 조회하지 않아 미확인이다.
- 조심할 것: `tests/unit/goal-health.test.ts` 는 미머지 `9b50afc` 가 물고 있으니 수정하지 말고 신규 `goal-age.test.ts` 를 쓸 것. `goal-health.ts` 의 `goalAgeHours` export 와 `(text, now?)` 시그니처를 없애면 그 브랜치가 깨진다. 날짜만 있는 값의 `Infinity` 계약은 이번에 바꾸지 말 것(없던 경고가 새로 생긴다). 빨간 로그는 "심볼 없음" 이 아니라 9시간 값 차이로 받아야 한다 — 회차 #14·#15 가 걸린 자리다.
- [러너 21:22] scout done — `마지막 갱신:` 시각을 읽는 두 벌의 중복 파서를 한 곳으로 모으고, 명시적 UTC/오프셋(`Z`, `±HH:MM`)을 KST로 

## 구현 노트
- 무엇을 왜: `마지막 갱신:` 나이를 읽는 두 벌(`goal-health.ts:24` export, `goal-loop.ts:149` private)을 `src/util/kst.ts` 의 새 순수 export `stampAgeHours(stamp, now)` 한 곳으로 모으고, 시각 뒤의 `Z`/`±HH:MM` 를 별도 정규식으로 확인해 오프셋을 그대로 해석한다(없으면 KST — 기존 동작). `goal-loop` 는 `goal-health` 의 export 를 import 한다. 커밋 `5a59fc7`, 프로덕션 3파일 + 신규 테스트 1파일.
- 확신 없는 곳·검증 못 한 것: (1) **`goal-loop` 가 공용 값을 쓴다는 것은 import(단일 정의 + typecheck)로만 보장했다** — `scheduleAutoResume` 을 실행해 확인하지 않았다. `config()`·`vscode.window`·`workspaceState` 가 필요하고 공유 스텁의 `window` 는 빈 객체여서, 손조립 대역 없이는 돌릴 수 없다. "경고가 뜬다/안 뜬다" 는 증명하지 않았다. (2) `ZONED_STAMP` 가 받는 표기 범위를 `Z`/`z`/`±HH:MM`/`±HHMM` 로 좁혔다 — `±HH`(분 없음)·주(`W`) 표기는 오프셋으로 보지 않고 KST 로 떨어진다. (3) Windows 패키징(`vsix`/`verify`/`smoke:vscode`)과 원격 CI·PR 상태는 이 환경에서 미확인이고 `npm run build` 는 런타임 자산 누락 경고를 그대로 남긴다 — 빌드 통과는 패키징 성공이 아니다.
- 일부러 하지 않은 것: 날짜만 있는 값(`마지막 갱신: 2026-09-01`)과 무스탬프의 `Number.POSITIVE_INFINITY` 계약을 **바꾸지 않았다**(바꾸면 사용자에게 없던 경고가 생긴다 — 신규 테스트가 이 불변을 못박았다). 초(`:SS`)는 instant 계산에 넣지 않아 기존 두 복사본과 같은 분 단위 동작을 보존했다. 기존 `goal-health.test.ts`/`goal-loop.test.ts`·공유 `vscode-stub.ts`·보호 경로는 건드리지 않았다. 차선 후보(`latestVsixVersion`)는 1순위가 막히지 않아 쓰지 않았다.
- 다음 역할이 조심할 것: `goal-health.ts` 의 `goalAgeHours` export 와 `(text, now?)` 시그니처는 미머지 `9b50afc` 가 물고 있으니 유지할 것. `tests/unit/goal-age.test.ts` 는 `node_modules` 만 있으면 돌고 fs·DB·fake timer 가 필요 없다(고정 `now` 숫자를 넘긴다). 기준선 실측값은 `main@51148a2` 에서 11 files / 85 tests 이고 이번 결과는 12 files / 90 tests 다.
- [러너 21:28] brief accepted — 채택 — 근거가 지금 코드와 정확히 맞았다(두 복사본, `goal.md:44`의 ISO 지시, 세 프로덕션 파일이 미머지 브랜치와 겹치지
- [러너 21:28] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: `stampAgeHours` 가 정말 유일한 파서다(src 전역 grep 에서 옛 식의 잔존 복사본 0개, `goalAgeHours(text, now?)` export 유지 → 미머지 `9b50afc` 계약 보존). 실패 재현을 옛 식으로 직접 재계산해 대조했다 — `2026-10-05T03:00:00Z`→18h, `...-04:00`→22h 로 `goal-age.test.ts` 1·2번이 main 에서 실제로 빨갛고, 증상도 커밋이 말하는 9시간 치우침과 같다. 단언은 production `goalTemplate` 의 `마지막 갱신:` 줄을 지난다. 실측 12 files / 90 tests, typecheck 무오류, build 성공.
- 못 본 것: Windows `vsix`/`verify`/`smoke:vscode`, 원격 CI·PR 상태, `scheduleAutoResume` 의 실제 실행(구현자의 미확인 #1 — import 단일화는 grep·typecheck 로만 확인했고 경고 표출은 여전히 미증명). 보안·법무 관점에서는 차단 사유 없음: 새 수집·전송·의존성·비밀값·권한 변화가 없고 regex 는 역추적 폭발 구조가 아니다.
- 승인이어도 남는 우려(릴리즈 노트): 리터럴 `UTC`/`GMT` 접미사와 ISO 기본형 `±HH` 는 **여전히** KST 로 읽힌다(각각 9h/9h 오차) — 커밋 제목 "명시적 UTC" 를 `Z` 지시자로 좁혀 읽어야 맞고, `goal-health.ts:24` 주석은 아직 오프셋 처리를 언급하지 않는다.
- 새 오탐 1개(차단 아님, 다음 회차가 같은 함수를 열면 함께): `마지막 갱신: 2026-10-05 12:00-14:00` 처럼 시각 범위가 들어오면 `-14:00` 을 오프셋으로 받아 나이가 -14h → stale 경고가 꺼지고 자동 재개가 켜진다. 오프셋 범위 clamp(±14:00)가 없어 `+99:99` 도 통과한다. production 쓰기 경로는 이 모양을 만들지 않는다.
- 테스트 품질: `goal-age.test.ts:36` "agrees with the shared parser" 는 위임이라 항상 참(수정 전에도 통과), 3번은 1번의 KST 단언 중복. 변경을 못박는 일은 1·2번이 하므로 거절 사유 아님.
- [러너 21:33] review approved — 리뷰 승인 (risk=low)
- [러너 21:33] pr created — https://github.com/hkjang/vibe-code/pull/18
- [러너 21:35] ci passed — 검사 2개 모두 success
- [러너 21:35] merge done — 5a59fc7
- [러너 21:41] release published — v1.4.7
- [러너 21:41] gh-release created — GitHub Release v1.4.7
- [러너 21:55] assets missing — 이전 v1.4.0 엔 2개, v1.4.7 엔 0개 — 워크플로: null: null/null

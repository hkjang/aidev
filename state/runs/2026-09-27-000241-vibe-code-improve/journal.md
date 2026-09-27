# 회차 노트 2026-09-27-000241-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:02] base pinned — main@b30117e
- [러너 00:02] autonomy release — 

## 정찰 노트
- 고른 이유: v1.4.3 이 섹션 파서를 하나로 모은 김에 그 파서의 **쓰기 쪽** 결함을 짚었다. `advanceCurrentPlan` 은 Now/Next 를 taskLines 로 걸러 읽고 writeSection 으로 섹션을 통째로 덮어써 사용자가 쓴 메모 줄이 사라진다 — 내용 유실이라 차선 후보(들여쓴 체크리스트 판정 불일치, 진행률 숫자만 어긋남)보다 손해가 크다.
- 제친 것: 들여쓴 체크리스트 3갈래 판정(3회 연속 차선, 가치 3/S — 차선으로 그대로 둠), matchLine 의 `\s*` 가 다음 줄을 읽는 건(재현 조건이 '빈 라벨 값' 이라 드묾), vibe-coders-proxy 테스트(스텁 대역 위주가 되어 운영자 지침에 걸림).
- 확신 없는 곳: **실행 검증을 못 했다** — 이 세션에서 `npm ci` 승인이 나지 않아 테스트를 한 번도 돌리지 못했고, 유실 시나리오는 plans.ts:226-243 과 markdown.ts:61-70 독해만으로 판정했다. 구현자는 실패 재현을 먼저 테스트로 찍고 시작할 것.
- 조심할 것: `section()` 이 본문을 trim 하므로 빈 Done 섹션에서 `[""]` 이 끼어드는 것, 원본 텍스트를 직접 split 하면 CRLF 에서 `\r` 이 남는 것, `now.shift()` 가 체크 상태를 안 보는 현행 동작은 범위 밖이라 건드리지도 못 박지도 말 것.
- [러너 00:06] scout done — `advanceCurrentPlan` 이 Now/Next 섹션의 체크리스트가 아닌 줄을 말없이 지우는 것을 막고, advance 로직을 테스트 

## 구현 노트
- 무엇을 왜: advance 가 Now/Next 를 `taskLines()` 로 걸러 읽고 `writeSection` 으로 섹션을 통째로 덮어써 사용자 메모 줄·들여쓰기가 사라지던 것을 고쳤다. `markdown.ts` 에 `sectionLines()`(원본 줄, 빈 섹션은 `[]`)를 더하고, advance 를 `advancePlanText(text, stamp)` 로 뽑아 항목만 splice 하며, `moveTaskToSection` 의 대상 섹션 재조립도 같은 함수로 바꿨다. 프로덕션 2파일 + 테스트 2파일.
- 확신 없는 곳: (1) `writeSection` 특성상 섹션 본문의 **맨 앞/맨 뒤 빈 줄**은 여전히 사라진다(`section()` 이 trim 하므로). 섹션 중간의 빈 줄은 보존된다 — 테스트로 못 박지 않았다. (2) Next 에서 승격되는 줄은 현행처럼 `trim()` 해서 Now 에 넣는다(들여쓴 하위 항목이 첫 항목이면 들여쓰기가 풀린다) — 현행 동작 유지 쪽을 택했고 테스트 픽스처에서는 그 경우가 나지 않는다. (3) 실행 검증은 리눅스 vitest·tsc·esbuild 까지다 — VSIX/확장 호스트 검증(Windows 전용)은 못 돌렸다.
- 일부러 안 한 것: `verification.ts:41` 의 같은 패턴(파일 수 억제 — 다음 회차 후보로 ideas.json 에 남김), 들여쓴 체크리스트 3갈래 판정 통일, `now.shift()` 가 이미 `- [x]` 인 항목도 완료 처리하는 현행 동작(범위 밖 — 새 테스트가 그 동작을 못 박지 않도록 해당 케이스를 아예 넣지 않았다).
- 다음 역할이 조심할 것: `advancePlanText` 는 vscode 를 import 하지 않는 순수 함수라 `tests/unit/plans.test.ts` 에서 문자열만으로 돈다(fs·대역 불필요). 신규 `sectionLines()` 는 `moveTaskToSection` 을 통해 CodeLens 경로도 함께 바꾸므로, 기존 CRLF 테스트(markdown.test.ts 하단)가 그 경로의 회귀 감시다. 검증은 워크트리에서 `npm ci` 후 `npm run check`.
- [러너 00:15] brief accepted — 채택 — 과제서가 미확인으로 남긴 유실 시나리오를 실제 테스트로 먼저 재현했고(메모 줄·들여쓰기 소실), 수용 기준 1~
- [러너 00:15] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인: `npm run check` 통과(78 tests). 프로덕션 2파일만 main 으로 되돌려 돌린 결과 `moveTaskToSection` 테스트가 실제 들여쓰기 소실로 실패 — 회귀 감시가 진짜다. `advancePlanText` 테스트는 함수 미존재로만 실패하므로 옛 인라인 로직을 따로 재현해, Now/Next 메모 줄 소실과 들여쓰기 풀림을 직접 관찰했다(원장의 실패 재현 출력과 일치).
- 구현자가 못 박지 않은 곳 확인: 섹션 **중간** 빈 줄은 보존되고 앞뒤 빈 줄만 사라진다 — 설명대로. Now 가 파일 마지막 섹션인 경우, 첫 체크리스트가 들여쓴 하위 항목인 경우, 이미 `- [x]` 인 경우도 옛 동작과 어긋나지 않는다.
- 승인이어도 남는 것: `## Done` 이 없는 계획에서는 완료 항목이 어디에도 안 남고 사라진다(재현함) — **main 과 같은 기존 동작**이라 이번 결함은 아니지만 구멍은 그대로다. 다음 회차: `advancePlanText` 가 `hasSection(text,"Done")` 거짓이면 null 을 돌려주게 할 것.
- 릴리즈 노트 한 줄: CodeLens '완료로 이동' 이 Done 의 들여쓰기를 더는 평탄화하지 않아, 들여쓴 체크리스트를 쓰던 사용자는 `countChecks`(`^- \[` 만 셈) 때문에 진행률 숫자가 달라져 보일 수 있다. 3갈래 판정 통일이 후속.
- 못 본 것: Windows 전용 경로(확장 호스트 테스트·VSIX·smoke)와 실제 VS Code 수동 확인. 보안·법무 차단 사유 없음(새 진입점·비밀값·외부 요청·의존성 없음, 감사 값은 main 에서도 쓰던 것).
- [러너 00:18] review approved — 리뷰 승인 (risk=low)
- [러너 00:18] pr created — https://github.com/hkjang/vibe-code/pull/4
- [러너 00:20] ci passed — 검사 2개 모두 success
- [러너 00:20] merge done — 4fb1c16
- [러너 00:29] release published — v1.4.4
- [러너 00:29] gh-release created — GitHub Release v1.4.4
- [러너 00:44] assets missing — 이전 v1.4.0 엔 2개, v1.4.4 엔 0개 — 워크플로: null: null/null

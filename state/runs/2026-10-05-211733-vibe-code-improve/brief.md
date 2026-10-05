# 정찰 과제서 — 2026-10-05-211733-vibe-code-improve (vibe-code)

base: `main@51148a2` (v1.4.6). 이 워크트리: `/home/hkjang/.cache/auto-improve-wt/vibe-code`.

- 과제: `마지막 갱신:` 시각을 읽는 두 벌의 중복 파서를 한 곳으로 모으고, 명시적 UTC/오프셋(`Z`, `±HH:MM`)을 KST로 오독하는 9시간 오차를 없애기 (가치 4 / 위험 2 / 작업량 M)

- 왜: 목표 파일의 `마지막 갱신:` 값을 읽는 코드가 **두 벌** 있다 — `src/features/goal-health.ts:24 goalAgeHours`(export)와 `src/features/goal-loop.ts:149 goalAgeHours`(같은 파일 내부 private 복사본). 둘은 정규식·KST 변환식(`+m[4] - 9`)이 글자까지 같고 각각 "정체 경고"(`assessGoalHealth`, 기본 48시간)와 "자동 이어서 진행"(`scheduleAutoResume`, 기본 72시간)을 결정하는데, `goal-loop.ts` 쪽 복사본은 export 되지 않아 테스트가 한 줄도 없다. 그 공통 변환식은 시각 문자열이 **무조건 KST**라고 가정해 `마지막 갱신: 2026-10-05T03:00:00Z` 같은 명시적 UTC 표기를 9시간 과대 계산한다 — 그리고 `assets/demo/.vibe/commands/goal.md:44` 가 AI 에게 이 줄을 `<ISO 또는 로컬 시각>` 으로 쓰라고 **직접 지시**하므로 ISO/`Z` 입력은 실제로 들어오는 경로다. 고치면 두 기능이 같은 입력을 같은 나이로 읽고, 48/72시간 경계에서 9시간 어긋나던 경고·재개 판정이 맞아진다.

- 수용 기준:
  1) `마지막 갱신:` 값에 `Z` 또는 `±HH:MM` 오프셋이 붙어 있으면 그 오프셋을 그대로 해석한다 — `2026-10-05T03:00:00Z` 와 `2026-10-05T12:00:00+09:00` 과 `2026-10-05 12:00:00 KST` 세 표기가 **같은 나이**를 낸다. 오프셋이 없는 기존 표기(`YYYY-MM-DD HH:MM:SS KST`, `kstStamp().human` 이 쓰는 형식)는 지금과 똑같이 KST 로 읽는다.
  2) `goal-loop.ts` 에 `goalAgeHours` 의 **두 번째 정의가 남아 있지 않다** — 같은 공용 함수를 import 해서 쓴다. 두 기능이 같은 목표 텍스트에 대해 같은 수를 말하는 것을 테스트가 **서로 비교해서** 못박는다(같은 입력 → `goal-health` 쪽 export 와 `goal-loop` 쪽이 쓰는 값이 동일).
  3) 기존 동작 보존: 시각이 아예 없거나(`"no stamp"`) 날짜만 있는 값은 지금처럼 `Number.POSITIVE_INFINITY` 를 돌려준다(= 경고도 자동재개도 침묵). 이 회차는 이 계약을 **바꾸지 않는다** — 바꾸면 사용자에게 새 경고가 갑자기 생긴다. 기존 테스트 `tests/unit/goal-health.test.ts:12-13`(`toBe(50)`, `toBe(Number.POSITIVE_INFINITY)`)이 한 줄도 수정되지 않고 통과해야 한다.
  4) 테스트가 증명할 것: (a) `Z`/`+09:00`/KST 세 표기의 나이 일치, (b) 오프셋 없는 표기의 KST 해석이 불변, (c) 중복 제거 후 두 경로의 값 일치, (d) 무시각·무스탬프의 `Infinity` 불변.

- 건드릴 파일 (프로덕션 3 + 테스트 1):
  - `src/util/kst.ts` — 순수 함수 `stampAgeHours(stamp: string, now: number): number` 를 **새로 export**. 본문: 현재 두 복사본의 정규식 `/(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})/` 로 날짜·시·분을 잡고, 같은 문자열의 시각 뒤에 `Z` 또는 `±HH:MM` 오프셋이 있는지 **별도로** 확인한다. 있으면 그 오프셋으로 instant 를 계산하고, 없으면 지금과 같이 `Date.UTC(y, mo-1, d, h - 9, mi)` 로 KST 로 본다. 매칭 실패는 `Number.POSITIVE_INFINITY`. 이 파일은 import 가 전혀 없는 순수 모듈이고(확인함: `Intl.DateTimeFormat` 만 쓴다) 미머지 브랜치가 건드리지 않는다 — 공용 함수의 자리로 안전하다. `\s*` 가 줄바꿈을 먹는 사고를 두 번 낸 저장소이므로 **여러 줄에 걸칠 수 있는 공백 매칭을 쓰지 말 것**(필요하면 `[ ]` 또는 `[^\S\n]`).
  - `src/features/goal-health.ts:24 goalAgeHours` — **export 를 없애지 말고** 본문만 `stampAgeHours(matchLine(text, "마지막 갱신") || matchLine(text, "작성일"), now)` 로 위임. 미머지 브랜치 `9b50afc` 의 `tests/unit/goal-health.test.ts` 가 이 심볼을 import 하므로 시그니처 `(text: string, now?: number)` 를 그대로 유지한다. JSDoc 의 "interpreting the KST timestamp" 는 새 계약(명시 오프셋 우선, 없으면 KST)으로 고쳐 쓴다 — 대체된 옛 설명을 남기지 말 것.
  - `src/features/goal-loop.ts:149-154` — private `goalAgeHours` 를 **삭제**하고 호출부 `:167`(`if (goalAgeHours(goal.text) > withinHours) return;`)이 공용 경로를 쓰게 한다. 가장 깔끔한 형태는 `goal-health.ts` 의 export 된 `goalAgeHours(goal.text)` 를 import 하는 것이다 — 순환 위험을 확인했다: `goals.ts:10` 이 이미 `./goal-health` 를 import 하고 `goal-health.ts:7` 이 `./goals` 의 `parseGoal` 을 import 하는 **기존 순환**이 있으며, `goal-loop.ts:7` 은 이미 `./goals` 를 import 한다. 따라서 `goal-loop → goal-health` 간선은 goal-loop 를 끌어들이는 **새 순환을 만들지 않는다**. 그래도 더 안전한 쪽을 원하면 `goal-loop.ts` 가 `../util/kst` 의 `stampAgeHours` 를 직접 쓰고 `matchLine` 은 이미 import 되어 있으므로 그 자리에서 조립하라 — 그때도 **정규식과 변환식은 `stampAgeHours` 안에만 한 벌** 있어야 한다(수용 기준 2).
  - `tests/unit/goal-age.test.ts` — **신규 파일**. 기존 `tests/unit/goal-health.test.ts` 와 `tests/unit/goal-loop.test.ts` 는 수정하지 말 것: `goal-health.test.ts` 는 미머지 `9b50afc` 가 물고 있어 충돌면이 된다. 고정 `now` 숫자를 넘겨 순수 함수로 검증하고 `vi.setSystemTime` 은 필요 없다(`stampAgeHours`/`goalAgeHours` 둘 다 `now` 를 받는다).

- 검증 명령 (이 워크트리에 `node_modules` 가 **없다** — 직접 확인했다. 먼저 설치해야 한다):
  - `npm ci` — `package.json` 의 `engines.node` 가 정확히 `20.19.2` 로 핀되어 있으므로 Node 20.19.2 에서 돌릴 것(이전 회차들은 `/tmp/node2019` 에 별도 설치해 썼다. 대상 워크트리의 `node_modules` 에 런타임을 설치하면 `npm ci` 가 지운다).
  - 집중: `npx vitest run tests/unit/goal-age.test.ts tests/unit/goal-health.test.ts tests/unit/goal-loop.test.ts`
  - 전체: `npm run check` (= typecheck + vitest + esbuild). **기준선은 추정하지 말고 수정 전에 `npx vitest run` 을 한 번 돌려 실측할 것** — 이전 회차 기록의 테스트 수(82/85/88)는 회차마다 달랐고 이 워크트리에서 다시 재야 한다.
  - 번들: `node --check dist/extension.js` 와 `node --check dist/extension.core.js`.
  - 주의: `npm run build` 는 런타임 자산이 없어도 경고만 내고 성공한다 — 빌드 통과를 패키징 성공으로 보고하지 말 것. `npm run vsix`/`verify`/`smoke:vscode` 는 Windows PowerShell 이 필요해 이 환경에서 미확인이다.

- 실패 재현 순서 (운영자가 되풀이해 요구한 형태):
  1) 먼저 **동작을 바꾸지 말고** 현재 KST-고정 본문을 그대로 `stampAgeHours` 로 추출만 하고 두 호출부를 배선한다. 그 상태에서 신규 테스트를 돌려 `AssertionError` 를 받는다 — `Z` 표기의 나이가 `+09:00`/KST 표기보다 9시간 크다는 **실제 값 불일치**로 빨개져야 한다. "심볼 없음"/`TypeError: not a function` 은 결함 재현이 아니다(회차 #14·#15 에서 같은 함정을 겪었다).
  2) 그다음 오프셋 처리를 넣어 초록으로 만든다.
  3) 인과 확인: export 는 남긴 채 **오프셋 처리만** 되돌려 같은 AssertionError 가 같은 메시지로 재발하는지 본다.

- 위험과 피할 것:
  - **미머지 브랜치가 물고 있는 파일을 건드리지 말 것** — 로컬 `origin/auto/*` 중 HEAD 미포함 커밋이 물고 있는 것: `src/util/markdown.ts`, `src/features/plans.ts`, `goal-lint.ts`, `goals.ts`, `handoff-on-exit.ts`, `journal.ts`, `journal-summary.ts`, `goal-metrics.ts`, `command-audit.ts`, `verification.ts`, 그리고 테스트 `markdown.test.ts`, `plans.test.ts`, `goal-catalog.test.ts`, **`goal-health.test.ts`**, `journal*.test.ts`, `goal-metrics.test.ts`, `command-audit.test.ts`, `verification-git.test.ts`, `goal-handoff.test.ts`. 이번 과제의 세 프로덕션 파일(`util/kst.ts`, `goal-health.ts`, `goal-loop.ts`)은 그 목록에 **없다**(`git log --name-only`로 대조 확인).
  - 공유 스텁 `tests/unit/vscode-stub.ts` 를 수정하지 말 것(미머지 변경과 충돌한다). 이번 과제는 순수 함수만 다루므로 스텁이 필요 없다.
  - 보호 경로 금지: `vendor/`, `src/core/`, `.github/workflows/`, `scripts/*.ps1`, `release/`, `src/features/checkpoints.ts`, `src/features/vibe-coders-proxy.ts`.
  - 손조립 대역(가짜 `CoreHost`·직접 주입한 `globalState`)으로 결함을 증명하지 말 것. `assessGoalHealth`/`scheduleAutoResume` 전체 흐름은 `config()`·`vscode.window`·`workspaceState` 가 필요하고 공유 스텁의 `window` 는 빈 객체다 — **그래서 이 과제는 순수 경계(나이 계산)에서만 증명하고, "경고가 실제로 뜬다/안 뜬다" 를 end-to-end 로 재현했다고 주장하지 말 것.**
  - 날짜만 있는 값(`마지막 갱신: 2026-09-01`)을 자정으로 해석하도록 바꾸지 말 것 — 사용자에게 없던 경고를 새로 만들고 계약 결정이 먼저다. 보류 아이디어로 남겼다.
  - 원격 CI·PR 상태는 이 세션에서 **미확인**이다(이전 10회차가 같은 상태였다). 녹색/빨간 CI 를 근거로 쓰지 말고 로컬 증거만 보고할 것. `ci.yml` 은 읽기만.

- 차선 후보: **`latestVsixVersion` 이 `vibe-code-<ver>.vsix` 만 인식해 실제 패키징 산출물 이름을 놓치는지 확인하고, 못 읽으면 "최신" 판정이 조용히 없음이 되는 것** (`src/features/update-check.ts:87`, 가치 3 / 위험 2 / S) — 단 `scripts/*.ps1` 이 만드는 실제 파일명을 먼저 읽어 확인해야 하고(이번 정찰에서 **미확인**), 파일명 계약이 다르지 않다면 이 후보는 성립하지 않는다. 그 경우 그 다음은 **`compareVersions`(`src/util/semver.ts`)가 프리릴리즈/비수치 조각에서 `NaN` 비교로 조용히 0 을 내는 경계를 테스트로 고정**(가치 2 / 위험 1 / S)이다 — 현재 호출부가 `[0-9.]+` 정규식으로 입력을 걸러 사용자에게 보이는 결함은 아니므로 가치가 낮다.

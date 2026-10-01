# 과제서 (확정 — 2026-10-01 정찰)

- 과제: 수정 과제 — PR #15 head `a1e3c94` 로 CI check job 로컬 재현 판정 + `enforceBudget` 의
  "하루 한 번" 알림 날짜가 KST 가 아니라 **UTC** 라서 한 KST 날에 두 번 발동하는 것 고치기
  (가치 3 / 위험 1 / 작업량 S)

- 왜: `src/features/usage.ts:128` 이 `const today = new Date().toISOString().slice(0, 10)` 으로
  "오늘" 을 **UTC 날짜**로 잡는다. 이 저장소의 다른 모든 날짜 경계는 `kstDate()` 다(감사 파일명
  `workspace.ts:60`, 일지 파일명, 직전 회차들이 `activeDays`·`readRecentAudit`·`readCommandHistory`
  를 모두 KST 로 맞췄다). UTC 날짜는 **09:00 KST 에 넘어가므로** 한 KST 날이 UTC 날짜 두 개에
  걸친다 — 08:00 KST 에 예산 초과 경고를 받은 사용자가 10:00 KST 에 **또** 경고를 받고
  `writeAutonomyPreset(host, "assist", true)` 가 **두 번째로** 자율성을 낮추며, 같은 KST 날의 감사
  파일 하나에 `budgetExceeded` 가 두 줄 쌓인다. 주석과 설정 설명이 약속한 "once per day" 가 오전에
  깨진다. 고치면 알림·자율성 강등·감사가 모두 같은 KST 날 경계를 쓴다.
- 이것은 `src/` 에 남은 **마지막** `toISOString().slice(0, 10)` 이다 — 직접 grep 해 1건뿐임을
  확인했다. 같은 값을 읽는 두 번째 경로가 없으므로 한쪽만 고치는 문제가 생기지 않는다.

## 0단계 — PR #15 CI 재현 판정 (먼저 할 것)

배정: "fix-round: CI failed, PR open https://github.com/hkjang/vibe-code/pull/15".
PR #15 = 로컬 `origin/auto/2026-10-01-0842`, head **`a1e3c948b14a7391fe5194a9b1070bafd1c8d42c`**
(`git rev-parse` 로 확인). 커밋 1개 — `a1e3c94 fix: readCommandHistory 가 "최근 7일" 을 KST 날짜
창으로 자르고 미상 종료 코드를 표시한다`, diff 는 `src/features/command-audit.ts` 14/4 ·
`tests/unit/command-audit.test.ts` 116/0 (`git diff --numstat`). **즉 직전 회차 산출물이다.**
PR 번호 ↔ 브랜치 대응은 **추정**이다(`gh` 미인증). 로컬 head SHA 만 확인했다.

```
# 1) CI 와 같은 Node 를 대상 디렉터리 "밖" 에 설치 — 안에 설치하면 뒤따르는 npm ci 가 지워 exit 127
mkdir -p /tmp/node2019 && cd /tmp/node2019
printf '{"name":"n","private":true}\n' > package.json
npm install node@20.19.2
export PATH=/tmp/node2019/node_modules/node/bin:$PATH
node -v   # v20.19.2

# 2) PR head 를 저장소 밖으로 추출
cd /home/hkjang/.cache/auto-improve-wt/vibe-code
git archive -o /tmp/pr15.tar origin/auto/2026-10-01-0842
mkdir -p /tmp/pr15 && tar -xf /tmp/pr15.tar -C /tmp/pr15

# 3) .github/workflows/ci.yml:13-24 의 check job 네 단계를 같은 순서로
cd /tmp/pr15
npm ci;                              echo "EXIT_npm_ci=$?"
npm run check;                       echo "EXIT_npm_check=$?"
node --check dist/extension.js;      echo "EXIT_node_check_extension=$?"
node --check dist/extension.core.js; echo "EXIT_node_check_core=$?"
```
네 줄 전부 `=0` → **"PR #15 의 원격 CI 실패는 코드 결함이 아니다"** 로 판정하고 로그를
`validation/pr15-node20-check.log` 에 남긴 뒤 **멈추지 말고 1단계로 넘어간다.**
하나라도 0 이 아니면 **그것이 이번 회차의 과제다** — 1단계를 버리고 그 실패를 고친다.

### 0단계에서 원장에 반드시 적을 것
- 배정이 말한 "릴리즈 워크플로" 는 **이 저장소에 없다** — `.github/workflows/` 에 `ci.yml` 하나뿐.
  문자 그대로는 성립하지 않는다(이전 회차가 직접 확인, 이번 정찰은 프로필 기록을 따랐다).
- 원격 CI 결과·annotation 은 이 세션에서 **읽을 수 없다**(`gh` 미인증, push·WebFetch 불가) → 미확인.
- 2026-09-29 보관 annotation: "The job was not started because recent account payments have failed
  or your spending limit needs to be increased.", `steps=[]`, `runner_id=0`. 열 회차 연속
  verify-failed 의 가장 그럴듯한 원인이지만 **이번 회차가 재확인한 것은 아니다.**
- CI `package` job(`ci.yml:26-37`, windows-latest) 은 이번에도 **미확인**(Windows 수단 없음).
- **`ci.yml` 을 완화하지 말 것.** 배정이 명시적으로 금지한다.

## 1단계 — 수용 기준

1) `enforceBudget` 이 쓰는 "오늘" 이 KST 날짜다 — 00:30 KST 와 23:30 KST 가 **같은** 키를 내고,
   09:00 KST 경계를 넘어도(08:00 KST 와 10:00 KST) 같은 키를 낸다. 고치기 전에는 08:00 과 10:00 이
   서로 다른 키라 같은 KST 날에 두 번 통과한다.
2) 그 키가 **감사 파일명과 같은 날짜 규칙**이다 — `budgetNoticeDay(d) === kstDate(d)` 가 위 경계
   순간들에서 성립한다(`writeAudit` 이 `kstDate()` 로 파일명을 만든다. 운영자의 "두 경로가 같은
   입력을 같은 값으로 읽는지 확인" 지시를 테스트가 직접 못박는다).
3) 테스트는 고치기 전 **빨갛고** 고친 뒤 통과한다. 빨간 로그는 아래 "실패 재현 순서" 대로
   **현재 식을 그대로 추출만 한 상태**에서 받아 "심볼 없음" 이 아니라 "프로덕션 동작이 틀림" 으로
   빨개지게 한다(회차 #14 가 `commandMark` 로 검증한 순서다).
4) `npm run check` 전체 통과, 기존 기대값 한 줄도 바뀌지 않는다.

## 건드릴 파일 (프로덕션 1개 · 테스트 1개)

- `src/features/usage.ts`
  - 순수 export 를 하나 더한다 — 이름은 `budgetNoticeDay` 로 할 것:
    ```ts
    /** KST day key for the once-per-day budget notice — audit filenames use the same KST day. */
    export const budgetNoticeDay = (now: Date = new Date()): string => kstDate(now);
    ```
  - `import { kstDate } from "../util/kst";` 를 더한다. **`kst.ts` 는 `vscode` 를 import 하지 않으므로**
    (직접 읽어 확인 — `Intl.DateTimeFormat` 만 쓴다) 테스트 그래프가 더러워지지 않고 순환도 없다.
  - `:128` `const today = new Date().toISOString().slice(0, 10);` → `const today = budgetNoticeDay();`
  - 그 외 `enforceBudget` 의 흐름·문구·`writeAudit` 필드·`globalState` 키
    (`"vibeCode.budgetNotifiedDate"`)는 **그대로 둔다.**
- `tests/unit/usage.test.ts` — 파일 **끝에 새 `describe`** 를 붙인다(기존 `describe "usage formatting"`
  두 블록과 `const report` 는 손대지 않는다). `budgetNoticeDay` 와 `kstDate` 를 import 해
  수용 기준 1·2 를 고정 `Date` 인스턴스로 단정한다. 권장 경계 순간:
  `new Date("2026-10-01T00:30:00+09:00")`(UTC 로는 09-30), `new Date("2026-10-01T08:00:00+09:00")`
  (UTC 로는 09-30 23:00), `new Date("2026-10-01T10:00:00+09:00")`, `new Date("2026-10-01T23:30:00+09:00")`
  — 네 개 모두 `"2026-10-01"` 이어야 하고 네 개 모두 `kstDate()` 와 같아야 한다.
  `vi.setSystemTime` 은 **필요 없다** — 인자로 `Date` 를 넘기면 된다.

### 실패 재현 순서 (이대로 할 것)
1. 먼저 `budgetNoticeDay` 를 **현재 동작 그대로** 추출만 한다:
   `export const budgetNoticeDay = (now: Date = new Date()): string => now.toISOString().slice(0, 10);`
   그리고 `:128` 을 `budgetNoticeDay()` 로 배선한다(동작 변경 없음).
2. 신규 테스트를 돌려 빨간 로그를 받는다 — `validation/usage-kst-red.log`.
   00:30 KST 와 08:00 KST 가 `"2026-09-30"` 으로 나와 `"2026-10-01"` 기대값과 어긋나고,
   `kstDate()` 와도 어긋나는 것이 `AssertionError` 로 보여야 한다.
3. 본문을 `kstDate(now)` 로 고치고 전부 통과시킨다 — `validation/usage-kst-green.log`.
4. 인과 확인: `git checkout HEAD -- src/features/usage.ts` 로 **프로덕션 1파일만** 되돌린다.
   이때는 `budgetNoticeDay` export 가 사라져 import 오류로 오염되므로, **깨끗한 결함 증거는
   2단계의 빨간 로그**임을 로그 머리에 적을 것(회차 #14 와 같은 상황이고 그렇게 처리해 채택됐다).

## 검증 명령

```
cd /home/hkjang/.cache/auto-improve-wt/vibe-code
npm ci                                    # 이 워크트리에 node_modules 가 없다. 반드시 먼저.
npx vitest run tests/unit/usage.test.ts    # 집중 검사
npm run check                              # typecheck + 전체 테스트 + esbuild (약 15초)
node --check dist/extension.js
node --check dist/extension.core.js
```
테스트 기준선은 **추정하지 말고 실측할 것** — 이 워크트리 `main@43fd7a1` 에서 `npx vitest run` 이
직전 회차 실측으로 **11 files / 82 tests** 였다. 신규 N개를 더한 수가 나와야 한다.

## 위험과 피할 것

- **`npm ci` 를 먼저.** 그리고 Node 는 `/tmp/node2019` 처럼 **대상 디렉터리 밖에** 설치할 것 —
  안에 설치하면 `npm ci` 가 `node_modules` 를 지워 `node --check` 가 exit 127 이 된다(실제로 겪었다).
- **`enforceBudget` 을 테스트에서 호출하려 하지 말 것.** `config()`·`host.context.globalState`·
  `vscode.window.showWarningMessage`·`writeAutonomyPreset` 네 협력자가 필요하고 공유
  `tests/unit/vscode-stub.ts` 는 `window = {}` 이고 `workspace.getConfiguration` 도 없다(직접 확인).
  그것을 채우려면 **손으로 만든 `CoreHost`·`globalState` 대역**이 필요한데 운영자가 되풀이해 금지한
  것이다. 그래서 이 과제는 날짜 키 계약만 순수 경계에서 증명한다 — **과제서는 이 한계를 숨기지 말고
  원장에 "알림 전체 흐름이 아니라 날짜 키 계약을 증명했다" 고 적을 것.** 과장 금지.
- **공유 `tests/unit/vscode-stub.ts` 를 고치지 말 것** — 미머지 PR 열 개와 충돌면이다.
- **`readRecentAudit`(goal-metrics.ts:21)을 이번에 고치지 말 것.** 보류 아이디어 1순위였지만 정찰이
  확인한 결과 **미머지 PR #13(`origin/auto/2026-10-01-0632`, head `1dd2151`)이 이미 정확히 그 수정을
  담고 있다**(`git show` 로 본문 대조 — `from = kstDate(...)` + 파일명 사전순 비교까지 동일). 다시
  하면 순수 중복이다. PR #13 머지 후 main 에서 확인할 것.
- **충돌 회피 — 아래 파일은 열지 말 것(미머지 PR 이 물고 있다):** `markdown.ts`(#6·#7·#8) ·
  `goal-lint.ts`(#7) · `plans.ts`(#9) · `journal.ts`(#10) · `goals.ts`·`handoff-on-exit.ts`(#11,
  이번에 `git diff --name-only` 로 재확인) · `goal-metrics.ts`(#12·#13·#14, 재확인) ·
  `journal-summary.ts` · `command-audit.ts`(#14·#15). `usage.ts`/`usage.test.ts` 는 **충돌면 0** 이다.
- **보호 경로를 건드리지 말 것:** `.github/workflows/`(완화 금지, 읽기만) · `vendor/` · `src/core/` ·
  `scripts/*.ps1` · `release/` · `checkpoints.ts` · `vibe-coders-proxy.ts`(전역 설정 덮어쓰기 금지).
- 감사에 **새 필드·원문을 넣지 말 것** — `writeAudit` 호출은 그대로 둔다(식별자만 쓰는 현 상태 유지).
- **정규식을 새로 쓰지 말 것** — 이 과제엔 필요 없다. 이 저장소는 `\s*` 가 줄바꿈을 먹어 두 번 데였다.
- 워크트리이므로 **`git stash` 금지**(스택 공유). 되돌리기는 `git checkout HEAD -- <경로>`.
- 문서: `docs/`·README·CHANGELOG·`package.nls*.json` 에 이 "하루 한 번" 날짜 규칙을 단언하는 줄이
  있는지 **확인하고**, 있으면 함께 갱신할 것(운영자 지시: 대체된 옛 가이드를 남기지 말 것).
  정찰은 이 grep 을 **하지 못했다 — 미확인**이다. `package.json` 의 `weeklyBudgetKrw` 설명도 볼 것.

## 차선 후보

**`trackNowItem`(`src/features/goal-health.ts:57`)의 `since` 가 UTC ISO 라 KST 기준 지표·표시와
경계가 어긋난다** — `since: new Date().toISOString()`. `goal-health.ts` 는 미머지 PR 충돌면이 0 이고
`tests/unit/goal-health.test.ts` 가 이미 있다. 단 `since` 를 **읽어서 표시하는 경로가 있는지 정찰이
확인하지 못했다(미확인)** — 쓰기만 하고 아무도 읽지 않으면 사용자에게 보이는 결함이 아니므로 가치가
낮다. 1순위가 성립하지 않으면 **먼저 `since` 의 독자를 grep 으로 찾고**, 독자가 없으면 이 후보도
버리고 0단계 판정만 원장에 남길 것.

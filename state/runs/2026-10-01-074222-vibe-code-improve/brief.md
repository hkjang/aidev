# 과제서 — 2026-10-01-074222-vibe-code-improve

- 과제: 수정 과제 — PR #13 head `1dd2151` 로 CI check job 로컬 재현 판정 + 감사 `exitCode` **누락**을 세 읽기 경로가 서로 반대로 읽는 것을 한 술어로 통일 (가치 4 / 위험 1 / 작업량 S)

## 0단계 — 먼저 이것부터 (배정된 오류 대응)

배정: "fix-round: CI failed, PR open https://github.com/hkjang/vibe-code/pull/13".

PR #13 ↔ 브랜치 대응은 **정찰의 추정**이다. 로컬에서 확인한 사실만 적는다:
`git for-each-ref refs/remotes/origin` 에서 가장 최신 자동 브랜치가 `origin/auto/2026-10-01-0632` head **`1dd2151`**(2026-10-01)이고,
`git diff --stat main 1dd2151` = `src/features/goal-metrics.ts | 20 ++++--`, `tests/unit/goal-metrics.test.ts | 108 ++++-` — 직전 회차(`readRecentAudit` KST 날짜 창)의 산출물과 일치한다. 따라서 `1dd2151` 을 PR #13 head 로 본다.

`.github/workflows/ci.yml:13-24` 의 check job 을 **같은 순서로** 재현하라. Node 는 `engines.node` 정확 핀 **20.19.2**.

```
mkdir -p /tmp/node2019 && cd /tmp/node2019
printf '{"name":"n","private":true}\n' > package.json
npm install node@20.19.2
export PATH=/tmp/node2019/node_modules/node/bin:$PATH
node -v   # v20.19.2 확인
```
**`/tmp/node2019` 는 반드시 대상 디렉터리 밖이어야 한다** — 안에 설치하면 뒤따르는 `npm ci` 가 `node_modules` 를 지워 `node --check` 가 exit 127 이 된다(2026-10-01 회차가 실제로 당했다).

```
cd /home/hkjang/.cache/auto-improve-wt/vibe-code
git archive -o /tmp/pr13.tar 1dd2151 && mkdir -p /tmp/pr13 && tar -xf /tmp/pr13.tar -C /tmp/pr13
cd /tmp/pr13
npm ci                          ; echo EXIT_npm_ci=$?
npm run check                   ; echo EXIT_npm_check=$?
node --check dist/extension.js  ; echo EXIT_node_check_extension=$?
node --check dist/extension.core.js ; echo EXIT_node_check_core=$?
```
전부 `validation/pr13-node20-check.log` 에 남겨라.

- **네 단계가 모두 exit 0 이면** → "PR #13 의 원격 CI 실패는 코드 결함이 아니다" 로 판정하고 **멈추지 말고 1단계로 넘어가라**. 직전 7회차(PR #6~#12)가 전부 같은 판정이었다.
- **하나라도 실패하면** → 그것이 이번 회차의 전부다. 1단계를 버리고 그 실패를 고쳐라.
- 원격 CI 자체는 이 세션에서 **확인 불가**다(`gh` 미인증, push 불가, WebFetch 권한 거부). 보관된 2026-09-29 annotation 은 `"The job was not started because recent account payments have failed or your spending limit needs to be increased."`, `steps=[]`, `runner_id=0` — 즉 job 이 시작조차 안 됐다. **`ci.yml` 은 읽기만 하고 절대 고치지 마라**(워크플로 완화 금지).

## 1단계 — 본 과제

- 왜: `src/features/command-audit.ts:20` 이 `if (event.exitCode !== undefined) details.exitCode = event.exitCode;` 로 쓰고(`CommandEvent.exitCode?: number` — `src/core/host.ts:65`, 옵셔널), 같은 파일 `:26` 이 `event.exitCode !== undefined` 를 따로 가드하는 것이 **exitCode 없는 `exited` 이벤트가 실제로 온다는 프로덕션의 자백**이다. 그렇게 쓰인 감사 줄 하나를 읽기 경로 셋이 정반대로 해석한다: `journal-summary.ts:66` 은 `Number(undefined)=NaN !== 0` 이라 **실패로 세고** 일지에 `npm test (exit undefined)` 를 찍는데, `goal-metrics.ts:69`·`:110` 은 `num(undefined)=0` 이라 **실패가 아니라고** 세어 주간 지표·회고에서 사라진다.
- 같은 `num()` 결함이 검증 통과율에도 있다: `goal-metrics.ts:51` 의 `num(e.details?.exitCode) === 0 && e.details?.exitCode !== null` 은 `exitCode` 가 **없는** `runVerification` 항목을 통과로 세고, `:52` 의 `failedVerifications` 에도 안 넣는다 — 그 항목은 통과율 분자와 분모에 모두 들어가 **통과율을 실제보다 높인다**. 고치면 일지·지표·회고가 같은 감사 파일에서 같은 숫자를 말한다.

### 계약 결정 (구현자는 이대로 — 재논의하지 말 것)

`exitCode` 누락 = **미상(unknown)**. 미상은 **성공이 아니다**. 근거: `verification.ts:34` 가 이미 `exitCode === null` 을 `TIMEOUT` 으로 다루고 `goal-metrics.ts:52` 가 그것을 `failedVerifications` 에 넣는다 — "종료 코드를 모른다" 를 통과로 세지 않는 계약이 이 저장소에 이미 있다. 누락을 `null` 과 같은 칸에 놓는 것이 그 계약의 연장이다. 그리고 두 경로 중 `journal-summary` 쪽(실패로 셈)이 이미 그렇게 하고 있으므로, 통일 방향은 `goal-metrics` 를 `journal-summary` 에 맞추는 것이다.

### 수용 기준

1. `exitCode` 필드가 **없는** `{kind:"command", action:"exited"}` 감사 줄 하나에 대해 `computeGoalMetrics(...).commandsFailed` 와 `buildSessionSummary(...)` 의 "실패 N개" 가 **같은 수**를 말한다(둘 다 1). 고치기 전에는 0 vs 1 이다.
2. `exitCode` 필드가 없는 `{kind:"plan", action:"runVerification"}` 항목이 `verificationPassRate` 의 **분자에서 빠지고** `failedVerifications` 에 **들어간다**. 항목 2개(하나는 `exitCode: 0`, 하나는 누락)면 통과율이 `1.0` 이 아니라 `0.5` 다.
3. `buildRetro` 의 `## 병목과 실패` 에 exitCode 누락 명령이 나타난다(현재는 조용히 빠진다).
4. 일지 요약이 `(exit undefined)` 라는 문자열을 더는 출력하지 않는다 — 미상임을 사람이 읽을 수 있게 쓴다.
5. **기존 기대값을 한 줄도 바꾸지 않고** 통과한다. 특히 `tests/unit/goal-metrics.test.ts:26` `verificationPassRate` `toBeCloseTo(1/3)`, `:28` `failedVerifications` 2건, `:31` `commandsFailed` 1, `:41` `` `go test ./...` — 실패 2회 ``, `tests/unit/journal-summary.test.ts:29` `"실행한 명령: 1개 승인, 1개 거부 · 실패 1개: npm test (exit 1)"`. 정찰이 두 픽스처를 열어 확인했다 — 모든 항목이 `exitCode` 를 0/1/2/null 로 **갖고 있어** 이 변경의 영향을 받지 않는다.
6. 인과 확인: 프로덕션 두 파일만 `git checkout HEAD -- <경로>` 로 되돌리면 새 테스트가 같은 메시지로 다시 빨개진다.

### 건드릴 파일 (프로덕션 2개)

- `src/features/journal-summary.ts` — 순수 export 헬퍼 하나 추가:
  `export function auditExitCode(details: Record<string, unknown> | undefined): number | null` — `const v = details?.exitCode; return typeof v === "number" && Number.isFinite(v) ? v : null;` (누락·`null`·NaN 을 전부 `null`= 미상으로 접는다). 여기에 두는 이유: `AuditEntry`/`parseAuditLines` 가 이미 이 파일에 있고 `goal-metrics.ts:9` 가 **이미 이 파일에서 import 한다** — 새 import 간선도 순환도 없다. 이 파일은 `vscode` 를 import 하지 않아 테스트 그래프가 깨끗하다(정찰이 `:1-4` 에서 확인: `node:fs`/`node:path`/`../util/kst`/`../util/markdown` 뿐).
  그리고 `:66 commandsFailed` 를 `auditExitCode(e.details) !== 0` 로 바꾸고, 렌더 문자열의 `(exit ${e.details?.exitCode})` 를 미상일 때 `undefined` 가 찍히지 않게 고친다(예: `(exit ${code ?? "미상"})`). **`exitCode: 1` 인 기존 기대값은 `(exit 1)` 그대로 나와야 한다.**
- `src/features/goal-metrics.ts` — 네 곳을 같은 헬퍼로 바꾼다:
  `:51 passed` → `auditExitCode(e.details) === 0` (기존 `num(...)===0 && ... !== null` 두 조건을 대체),
  `:52 failedVerifications` → `auditExitCode(e.details) !== 0`,
  `:69 commandsFailed` → `auditExitCode(e.details) !== 0`,
  `:110 buildRetro` 의 병목 집계 → `auditExitCode(e.details) !== 0`.
  import 는 기존 `:9` 줄에 이름만 추가한다. **`num()`(`:14`)은 tokens/cost/archived 등 다른 곳이 쓰므로 지우지 마라.**
- 테스트: `tests/unit/goal-metrics.test.ts` 에 **새 `describe` 블록**만 추가(기존 `AUDIT` 상수와 기대값은 손대지 말 것), `tests/unit/journal-summary.test.ts` 에 새 `it` 추가. 픽스처는 두 파일의 기존 방식 그대로 — JSON 문자열 줄을 `parseAuditLines` 에 넣는다(이건 손조립 대역이 아니라 **감사 파일의 실제 온-디스크 형식**이고 두 기존 테스트가 이미 쓰는 프로덕션 경로다). 더 강하게 하고 싶으면 `writeAudit` 로 실제 파일을 만들어도 되지만 필수는 아니다 — `handleCommandEvent` 는 `activeHost` 전역과 `checkpointBeforeCommand` 를 타므로 이번 회차에 부르지 마라.

### 검증 명령 (이 저장소에서 실제로 도는 것)

```
npx vitest run tests/unit/goal-metrics.test.ts tests/unit/journal-summary.test.ts   # 실패 재현 → green
npm run check          # typecheck + vitest 전체 + esbuild (~15초)
node --check dist/extension.js
node --check dist/extension.core.js
```
테스트 기준선은 **추정하지 말고 실측하라**. 직전 두 회차가 이 워크트리의 `main@43fd7a1` 에서 `npx vitest run` → **11 files / 82 tests** 를 실측했고 base 가 같은 커밋에 핀돼 있지만, 브랜치마다 수가 달랐던 전례가 있으니 직접 한 번 돌려 확인할 것. 정찰 세션에서는 `npx vitest run` 실행이 **권한 거부**돼 이번에 직접 재확인하지 못했다(미확인).

### 위험과 피할 것

- **`ci.yml` 을 고치지 마라.** 워크플로 완화는 이 배정 계열이 명시적으로 금지한다. 읽기만.
- 보호 경로 금지: `vendor/`, `scripts/*.ps1`, `release/`, `checkpoints.ts`, `vibe-coders-proxy.ts`. 감사(`workspace.ts:60`)에 새 필드·원문을 넣지 마라 — **쓰는 쪽은 건드리지 않는다. 이번 과제는 읽는 쪽만이다.**
- `command-audit.ts` 도 고치지 마라. `:20` 의 조건부 쓰기는 결함이 아니라 계약이고, 거기를 고치면 이미 쌓인 감사 파일은 그대로라 읽기 쪽 불일치가 남는다.
- 미머지 PR 충돌: `goal-metrics.ts` 는 PR #13(`readRecentAudit:17-25`, `activeDays:59`)이, `journal-summary.ts` 는 PR #11(`appendToJournal`, `:85·:101` 부근)이 물고 있다. 둘 다 **이번 변경과 다른 hunk** 라 머지 가능하지만, `goal-metrics.test.ts` 는 PR #13 이 108줄을 더했으니 **기존 `AUDIT` 상수를 수정하지 말고 새 `describe` 를 파일 끝에 붙여** 충돌면을 줄여라.
- 공백 정규식 `\s*` 를 쓰지 마라 — 이 저장소가 줄바꿈을 먹혀 두 번 데인 자리다. 이번 과제엔 정규식이 필요 없다.
- 공유 `tests/unit/vscode-stub.ts` 를 고치지 마라(미머지 테스트와 충돌).
- 두 경로를 **한쪽만** 고치지 마라 — 운영자가 되풀이해 금지한 항목이고, 이 과제의 존재 이유 자체가 그 불일치다.
- 문서: `docs/`·README·CHANGELOG 에 이 통과율/실패 집계 규칙을 단언하는 줄이 있는지 확인하고, 있으면 함께 갱신하라(정찰은 확인하지 못했다 — 미확인).

## 차선 후보

**`readCommandHistory`(`command-audit.ts:38-58`)가 "최근 7일" 을 "최근 7개 파일" 로 읽는다** (가치 3 / 위험 1 / 작업량 S) — `:45` 의 `files.slice(-7)` 이 직전 회차가 `readRecentAudit` 에서 고친 것과 **정확히 같은 결함**이고, 출력 헤더 `:70` 은 "명령 실행 이력 (최근 7일, 최대 30건)" 이라고 단언한다. 감사 파일이 7개 미만이면 두 달 전 명령이 "최근 7일" 로 표시된다. 고치는 방법도 이미 검증됐다 — 파일명이 `YYYY-MM-DD.jsonl` 이므로 `kstDate(new Date(Date.now() - 6*864e5))` 와 사전순 비교. 단 `:43` 필터가 `endsWith(".jsonl")` 라 `readRecentAudit` 의 날짜 정규식보다 느슨하니 날짜 형식이 아닌 파일 이름을 먼저 배제해야 한다. 프로덕션 1파일. 1순위가 0단계에서 막히거나 이미 해결돼 있으면 이것을 집어라.

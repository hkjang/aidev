- 과제: 수정 과제 — PR #14 head `bf12461` 로 CI check job 로컬 재현 판정 + `readCommandHistory` 가 "최근 7일" 을 "최근 7개 파일" 로 읽고 `exitCode` 누락을 `! 종료 undefined` 로 찍는 것 고치기 (가치 3 / 위험 1 / 작업량 S)

- 왜: 배정이 "릴리즈 워크플로가 같은 이유로 두 번 실패" 라고 했지만 이 저장소의 워크플로는 `.github/workflows/ci.yml` **하나뿐**이고(확인: `ls .github/workflows/` → `ci.yml`), 직전 여덟 회차(PR #6~#13)가 전부 check job 을 로컬에서 재현해 exit 0 을 받았다. 그러므로 0단계는 **같은 재현을 PR #14 head 에서 한 번 더 해 판정을 남기는 것**이고, 거기서 막히지 않으면(막히지 않을 것으로 예상) 1단계로 넘어가 실사용 결함을 하나 고친다. 1단계 결함은 `src/features/command-audit.ts` 에 있다: `readCommandHistory`(:45)가 감사 디렉터리를 `files.slice(-7)` 로 골라 "최근 7일" 이 아니라 "최근 7개 파일" 을 읽는데 바로 아래 출력 헤더(:70)가 `=== 명령 실행 이력 (최근 7일, 최대 30건) ===` 라고 단언하므로, 감사 파일이 7개 미만인 실제 워크스페이스에서는 두 달 전 명령이 "최근 7일" 로 사용자에게 표시된다. 같은 함수가 읽은 행을 찍는 :73 은 `row.exitCode === 0 ? "✓ 종료 0" : \`! 종료 ${row.exitCode}\`` 이므로 `exitCode` 없는 `exited` 행이 `! 종료 undefined` 로 나간다 — 이 행은 실제로 쌓이는 경로다(같은 파일 `handleCommandEvent:20` 이 `event.exitCode !== undefined` 일 때만 `details.exitCode` 를 넣는다). 고치면 이 패널이 자기 헤더와 일치하는 범위를 보여주고 미상 종료 코드를 사람이 읽을 수 있게 표시한다. 프로덕션 파일 **1개**로 끝난다.

- 수용 기준:
  1. **0단계 판정 기록.** PR #14 head(`bf12461`, 로컬 `origin/auto/2026-10-01-0742` — 확인됨)를 저장소 밖에 추출해 `ci.yml:13-24` check job 네 단계(`npm ci` → `npm run check` → `node --check dist/extension.js` → `node --check dist/extension.core.js`)를 Node 20.19.2 에서 같은 순서로 돌린 exit code 를 로그 파일에 남긴다. 전부 0 이면 "PR #14 의 원격 CI 실패는 코드 결함이 아니다" 로 판정하고 **멈추지 말고** 2번으로 간다. 하나라도 실패하면 거기가 이번 과제이고 그 실패를 고친다(워크플로 조건 완화는 금지 — `ci.yml` 은 읽기만).
  2. 감사 디렉터리에 창 밖 날짜 파일(예: 60일 전, 7일 전)과 창 안 파일(6일 전 = 경계, 오늘)이 있을 때, `readCommandHistory` 가 **창 안 파일의 행만** 돌려준다. 창 밖 두 파일의 명령 문자열이 결과에 없다. 파일이 7개 미만이어도 그렇다(현재 동작은 5개면 전부 반환한다).
  3. 경계는 KST 로 자른다 — `writeAudit`(`workspace.ts:60`)이 파일명을 `kstDate()` 로 만들므로 읽는 쪽도 같은 규칙이어야 한다. 6일 전 파일은 **포함**, 7일 전 파일은 **제외**.
  4. `exitCode` 가 없는 `exited` 행의 출력 줄에 `undefined` 가 들어가지 않고 `미상` 으로 표시된다. `exitCode` 가 0 인 행은 `✓ 종료 0`, 0 이 아닌 수인 행은 `! 종료 1` 처럼 **기존 문구 그대로**다. `approved`/`denied` 문구도 그대로다.
  5. 테스트가 증명하는 것: 손으로 조립한 JSONL·가짜 `CommandEvent` 없이, 프로덕션 `writeAudit(host, "command", <phase>, {...})` 으로 실제 감사 파일을 만들고 프로덕션 `readCommandHistory(paths.audit)` 로 읽어 2·3·4를 확인한다. 과거 날짜 파일은 `fs.copyFileSync` 손조립 대신 `vi.useFakeTimers({ toFake: ["Date"] })` + `vi.setSystemTime` 으로 **시계만 옮겨** 프로덕션 `writeAudit` 이 스스로 KST 날짜 파일명과 일관된 `ts` 를 만들게 한다(직전 회차에서 이 패턴이 실제로 동작했다).
  6. `npm run check`(typecheck + vitest + esbuild) 와 `node --check` 두 번들이 Node 20.19.2 에서 전부 exit 0. 기존 기대값은 한 줄도 바꾸지 않는다. 테스트 수 기준선은 **추정하지 말고 실측**한다(이 워크트리 main@43fd7a1 에서 `npx vitest run` → 11 files / 82 tests 로 기록돼 있으나 직접 확인할 것).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `src/features/command-audit.ts:38 readCommandHistory` — 두 곳만 바꾼다. (a) `:43` 의 `.filter((name) => name.endsWith(".jsonl"))` 를 `readRecentAudit`(`goal-metrics.ts:21`)이 이미 쓰는 것과 같은 `/^\d{4}-\d{2}-\d{2}\.jsonl$/` 로 좁힌다 — **이것이 먼저다.** 현재 필터는 `notes.jsonl` 같은 비날짜 이름을 통과시키고 `"notes.jsonl".slice(0,10)` = `"notes.json"` 은 `"2026-…"` 보다 사전순으로 크므로 날짜 비교만 넣으면 오히려 항상 포함된다. (b) `.sort().slice(-7)` 을 날짜 창 필터로 바꾼다: `import { kstDate } from "../util/kst";` 를 더하고 `const from = kstDate(new Date(Date.now() - 6 * 864e5));` 를 잡아 `files.filter((n) => n.slice(0, 10) >= from)` 로 고른다. 파일명이 `YYYY-MM-DD.jsonl` 로 고정되므로 사전순 비교 = 날짜 비교다 — **새 정규식을 쓰지 말고 문자열 비교로 끝낸다**(이 저장소는 `\s*` 정규식으로 두 번 데였다). `.sort()` 와 끝의 `rows.slice(-limit)` 는 그대로 둔다.
  - `src/features/command-audit.ts:73` 의 mark 식 — 그 삼항식을 **순수 export 함수로 빼낸다**: `export function commandMark(row: CommandAuditRow): string`. 본문은 현재 문구를 그대로 유지하고 마지막 분기만 `` `! 종료 ${row.exitCode ?? "미상"}` `` 으로 바꾼다. `:73` 은 `const mark = commandMark(row);` 가 된다. 출력 형식(`  ${row.ts} :: ${mark} :: ${row.command}…`)과 헤더 문구는 손대지 않는다 — 헤더는 이제 사실이 된다.
  - 함수 JSDoc `:37` "Recent command events across all audit files" 도 KST 날짜 창으로 고친다(운영자 지시: 대체된 옛 설명을 남기지 않는다).
  - `tests/unit/command-audit.test.ts` (**신규**) — 이 이름의 테스트 파일은 없다(확인: `ls tests/unit/` 11개 + `vscode-stub.ts`). `mkdtempSync` 임시 디렉터리를 `vscode-stub` 의 `workspace.workspaceFolders` 에 **국소 캐스트로** 대입해 프로덕션 `ensureWorkspacePaths(true)`/`workspaceRoot()` 배선을 그대로 지난다. **공유 `tests/unit/vscode-stub.ts` 는 고치지 말 것** — 미머지 PR 들과 충돌한다. `host` 는 `writeAudit` 이 실패 시 `log` 에만 쓰므로 `output.appendLine` 만 통과하는 최소 객체로 충분하다(결함 증명에 쓰는 대역이 아니다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 이 워크트리에는 `node_modules` 가 **없다**(확인). 먼저 `npm ci`.
  - CI 와 같은 Node 확보 — **대상 디렉터리 안에 설치하지 말 것**(뒤따르는 `npm ci` 가 `node_modules` 를 지워 `node --check` 가 exit 127 이 된다):
    ```
    mkdir -p /tmp/node2019 && cd /tmp/node2019
    printf '{"name":"n","private":true}\n' > package.json
    npm install node@20.19.2
    export PATH=/tmp/node2019/node_modules/node/bin:$PATH
    ```
  - 0단계: `git archive --format=tar -o /tmp/pr14.tar bf12461 && mkdir -p /tmp/pr14 && tar -xf /tmp/pr14.tar -C /tmp/pr14`, 그 안에서 `npm ci` → `npm run check` → `node --check dist/extension.js` → `node --check dist/extension.core.js`, 각 exit code 를 `validation/pr14-node20-check.log` 에 기록.
  - 1단계 집중: `npx vitest run tests/unit/command-audit.test.ts`
  - 1단계 전체: `npm run check` 그리고 `node --check dist/extension.js && node --check dist/extension.core.js`
  - 실패 재현 순서: **먼저 현재 동작을 그대로 `commandMark` 로 추출만 하고**(동작 변경 없음) 신규 테스트를 돌려 빨간 것을 받는다 — 그러면 결함이 "심볼 없음" 이 아니라 "프로덕션 동작이 틀림" 으로 빨개진다(PR #9 회차에서 검증된 순서). 로그는 `validation/command-audit-red.log`. 고친 뒤 인과 확인: `git checkout HEAD -- src/features/command-audit.ts` 로 프로덕션 1파일만 되돌려 같은 테스트가 같은 메시지로 다시 빨개지는지 확인(`validation/causation-red.log`). 이때 새 export `commandMark` 를 테스트가 직접 import 하면 import 오류가 섞여 증거가 더러워진다 — **날짜 창 테스트는 기존 export `readCommandHistory` 만 부르게 쓰고**, `commandMark` 를 부르는 테스트는 import 오류가 섞이는 것을 로그에 명시하거나 위의 "추출만 한 상태" 빨간 로그를 그 항목의 증거로 쓸 것.

- 위험과 피할 것:
  - **`ci.yml` 은 읽기만.** 워크플로 조건 완화는 이 배정이 명시적으로 금지한다. `package` job(`ci.yml:26-37`, windows-latest)은 이 세션에 Windows 실행 수단이 없어 **미확인**이고, 그 뒤 Package/Verify/SHA256/Upload 는 `ci.yml:41-57` 의 `continue-on-error: true` + `steps.assets.outputs.found == 'true'` 게이트로 조용히 skip 될 수 있다 — 녹색 CI 가 패키징 성공을 증명하지 않는다. 이번 회차에서 고치려 하지 말 것(보호 경로 + Windows 필요).
  - **`goal-metrics.ts:17 readRecentAudit` 는 건드리지 말 것.** main@43fd7a1 에서 여전히 `.slice(-days)` 이고 JSDoc 도 "the last `days` daily files" 다(직접 확인) — 즉 이번 과제와 **똑같은 결함**이지만 그 수정은 미머지 PR #13(`origin/auto/2026-10-01-0632`)이 물고 있고 PR #14 도 같은 파일을 고쳤다. 두 기능(주간 지표 vs 명령 이력 패널)은 각자 자기 창을 문서화하는 별개 출력이므로 한쪽만 고쳐도 어떤 단일 출력이 어긋나지 않는다. 같은 커밋에 얹으면 3중 충돌 + 인과 귀속이 흐려진다(shotgun 금지). 원장에 "같은 결함이 `readRecentAudit` 에도 있고 PR #13 이 물고 있다" 를 남길 것.
  - **`handleCommandEvent`(:16-27) 과 `writeAudit`(`workspace.ts:60`) 은 건드리지 말 것.** `exitCode` 를 옵셔널로 두는 것은 `CommandEvent`(`core/host.ts`)의 계약이고 이미 쌓인 감사 파일이 그대로다 — 읽는·찍는 쪽을 맞추는 것이 맞다. 감사에 새 필드나 출력 원문을 넣지 말 것(식별자만).
  - 미머지 PR 이 물고 있어 피해야 하는 파일: `markdown.ts`(#6·#7·#8), `plans.ts`(#9), `journal.ts`(#10), `goals.ts`·`handoff-on-exit.ts`(#11), `goal-metrics.ts`·`journal-summary.ts`(#12·#13·#14), `goal-lint.ts`(#7), 공유 `tests/unit/vscode-stub.ts`. `command-audit.ts` 는 **어느 PR 도 건드리지 않았다**(확인: PR #14 diff 4파일에 없음) — 그래서 이것을 골랐다.
  - 보호 경로: `vendor/`, `scripts/*.ps1`, `release/`, `checkpoints.ts`, `vibe-coders-proxy.ts` — 건드리지 말 것.
  - 임시 디렉터리가 상위 Git 저장소 안이면 Git 관련 테스트가 잘못된 저장소를 찾는다. `TMPDIR` 을 쓸 때 `GIT_CEILING_DIRECTORIES=$TMPDIR` 도 명령 범위에 지정.
  - `git stash` 금지(워크트리가 스택을 공유한다). 되돌리기는 `git checkout HEAD -- <경로>`.
  - 문서는 고칠 것이 없다 — `docs/`·README·CHANGELOG·`package.nls*.json` 을 grep 해 이 7일 창을 단언하는 줄이 **없음**을 확인했다(`docs/improvement-roadmap.md:42` 는 기능 존재만 말하고 창을 단언하지 않는다).
  - **원격 CI 는 이 환경에서 확인할 수 없다** — `gh` 미인증, push·WebFetch 불가. PR 번호 ↔ 브랜치 대응(`#14` = `auto/2026-10-01-0742`)은 **추정**이다(head `bf12461` 이 로컬에 있는 것만 확인). 이것을 코드 변경으로 우회하지 말고 원장에 운영 항목으로 남길 것: (1) GitHub 계정 결제/사용 한도 해소, (2) 세션에 `gh` 토큰 제공.

- 차선 후보: **`readCommandHistory` 의 날짜 창만** 하고 `commandMark` 추출은 다음 회차로 넘긴다(수용 기준 4 제외, 프로덕션 변경 3줄). 0단계가 예상과 달리 실제 실패를 내면 그 실패가 1순위이고 이 1단계는 전부 버린다.

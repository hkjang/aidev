# 과제서 — 2026-10-01 (정찰)

- 과제: 수정 과제 — PR #11(`origin/auto/2026-10-01-0332`, head `af9b04bfa95b74252ddd82bb864a65426651b9c4`) CI 실패를 Node 20.19.2 로 로컬 재현해 판정하고, 그다음 **오늘 일지 파일에 쓰는 두 경로**(`journal.ts:35`, `journal-summary.ts:101`)가 CRLF 일지에 LF-only 줄을 섞는 것을 하나의 공용 헬퍼로 함께 고치기 (가치 3 / 위험 1 / 작업량 S)

- 왜: PR #6~#11 여섯 회차가 전부 "원격 CI 실패 → 로컬 재현 전부 통과 → 코드 결함 아님" 으로 끝났고, 2026-09-29 회차가 보관한 GitHub annotation 은 `"The job was not started because recent account payments have failed or your spending limit needs to be increased."`(`steps=[]`, `runner_id=0`)였다. 즉 고칠 코드 결함이 원격에 있다는 증거가 없으므로 0단계에서 그것을 증거 로그로 못 박고, 멈추지 말고 1단계의 실제 결함을 고친다. 1단계 결함은 이 확장이 이미 두 번(v1.4.5 `appendVerificationLog`, PR #11 `appendGoalNote`) 같은 이유로 고친 것과 같은 계열인데, **같은 파일 `.vibe-code/journal/<오늘>.md` 에 쓰는 writer 두 개가 아직 둘 다 LF-only** 라서 Windows(이 확장의 1차 대상 — 패키징·검증 스크립트가 전부 PowerShell) 사용자의 CRLF 일지에 LF 줄이 섞인다.

## 0단계 (게이트) — PR #11 head 로 CI check job 재현

`.github/workflows/ci.yml:13-24`(실제로 읽음) 의 check job 과 **같은 순서**로 돌린다. `ci.yml` 은 읽기만 하고 **절대 수정하지 않는다**(배정이 워크플로 완화를 금지).

```
# 1) Node 20.19.2 를 대상 디렉터리 "밖" 에 설치한다 — 안에서 하면 뒤따르는 npm ci 가
#    node_modules 를 통째로 지워 pinned node 가 사라지고 node --check 가 exit 127 이 된다
#    (2026-10-01 구현 회차에서 실제로 발생한 함정).
mkdir -p /tmp/node2019 && cd /tmp/node2019
printf '{"name":"n","private":true}\n' > package.json
npm install node@20.19.2
export PATH=/tmp/node2019/node_modules/node/bin:$PATH
node -v          # v20.19.2 여야 한다

# 2) PR head 를 저장소 밖으로 추출 (작업 트리를 건드리지 않는다)
cd <worktree>
git archive -o /tmp/pr11.tar af9b04bfa95b74252ddd82bb864a65426651b9c4
mkdir -p /tmp/pr11 && tar -xf /tmp/pr11.tar -C /tmp/pr11

# 3) ci.yml 의 4단계
cd /tmp/pr11
npm ci                       # EBADENGINE 경고는 실패가 아니다
npm run check                # = npm run typecheck && npm run test && npm run build
node --check dist/extension.js
node --check dist/extension.core.js
```

- 네 단계가 모두 exit 0 이면 **"PR #11 의 원격 CI 실패는 코드 결함이 아니다"** 로 판정하고 종료 코드를 로그 파일에 남긴 뒤(`validation/pr11-node20-check.log`) **멈추지 말고 1단계로 간다.**
- 어느 하나라도 실패하면 그것이 이번 회차의 과제다. 1단계는 버리고 그 실패를 고친다.
- **원격 상태는 이 세션에서 확인할 수 없다** — 이번 정찰에서 `gh auth status` 실행이 권한 거부됐고 앞선 다섯 회차도 `gh` 미인증·push 불가·WebFetch 거부였다. PR 번호 ↔ 브랜치 대응(`#11` = `auto/2026-10-01-0332`)은 **정찰의 추정**이다. 확인한 것은 `origin/auto/2026-10-01-0332` head 가 `af9b04b`(= 직전 회차가 만든 goals/handoff EOL 수정)이라는 것뿐이다. 추정을 사실처럼 쓰지 말 것.
- **아직 아무도 재현하지 않은 경로 하나**: 여섯 회차 모두 `check` job(Linux)만 재현했고 `package` job(`ci.yml:26-37`, windows-latest, `needs: check`)의 `npm ci` + `npm run build` 는 재현한 적이 없다. 정찰이 `scripts/build.mjs` 전문을 읽어 확인한 바로는 `path.join`/`node:fs` 만 쓰고 셸을 타지 않아 플랫폼 의존이 없고, `vendor/extension.core.js` 는 `git archive` 추출본으로 `npm run check` 가 통과한 사실로 이미 tracked 임이 증명된다. 따라서 여기서 실패할 근거는 **없지만 미확인**이다. Windows 가 없으면 "미확인" 으로 적을 것.

## 1단계 — 일지 파일에 쓰는 두 writer 의 EOL 보존

정찰이 `grep -rn "appendFileSync" src/` 로 확인한 main@43fd7a1 의 전체 목록:

| 위치 | 대상 파일 | 상태 |
|---|---|---|
| `src/features/journal.ts:35` | `.vibe-code/journal/<오늘>.md` | **LF-only (이번 과제)** |
| `src/features/journal-summary.ts:101` | `.vibe-code/journal/<오늘>.md` | **LF-only (이번 과제)** |
| `src/features/goals.ts:123` | `.vibe-code/goals/current.md` | PR #11 이 고치는 중 — **건드리지 말 것** |
| `src/features/handoff-on-exit.ts:48` | `.vibe-code/goals/current.md` | PR #11 이 고치는 중 — **건드리지 말 것** |
| `src/features/workspace.ts:66` | `.vibe-code/audit/<오늘>.jsonl` | JSONL 은 LF 가 맞다 — **고치지 말 것** |

정찰이 두 대상 경로를 직접 읽어 확인한 것:
- `journal.ts:26 initJournal` — 오늘 일지가 없으면 `fs.writeFileSync(file, journalTemplate(ymd, clock))`(journal.ts:9-23, LF 템플릿), 있으면 `fs.appendFileSync(file, \`- ${clock} — 확장 재활성화\n\`)`. 인코딩 인자도 없다.
- `journal-summary.ts:101 appendSessionSummary` — `fs.appendFileSync(journalFile, (journal.endsWith("\n") ? "" : "\n") + "\n" + summary, "utf8")`. `summary` 는 `buildSessionSummary`(같은 파일, export 됨)가 `lines.join("\n")` 으로 만든다. `journal` 원문을 이미 `readFileSync` 로 손에 들고 있으므로 추가 읽기가 필요 없다.
- 즉 사용자가 Windows 에서 일지를 편집해 CRLF 로 저장하면, 재활성화 줄과 세션 요약 블록만 LF 단독으로 섞인다.

### 하는 일

1. **순수 export 헬퍼 하나**를 `src/features/journal-summary.ts` 에 둔다(여기에 두는 이유: 이 파일은 `vscode` 를 import 하지 않아 테스트 그래프가 깨끗하고, `journal.ts` → `journal-summary.ts` 방향은 순환이 아니다 — 정찰이 `grep` 으로 확인했다. `journal-summary.ts` 를 import 하는 것은 `extension.ts:3`·`goal-metrics.ts:9` 뿐이고 둘 다 `journal.ts` 를 import 하지 않는다):

```ts
/** Append a block to a journal keeping the file's own line endings. */
export function appendToJournal(text: string, block: string): string {
	return restoreEol(normalizeEol(text) + normalizeEol(block), detectEol(text));
}
```

`detectEol`/`normalizeEol`/`restoreEol` 는 `../util/markdown` 에서 가져온다 — `verification.ts:45 appendVerificationLog`(v1.4.5)와 **같은 계약**이다. `markdown.ts` 는 **읽기만** 하고 수정하지 말 것(PR #6·#7·#8 이 물고 있다). 세 함수가 실제로 export 되는지는 착수 시 확인할 것(정찰은 프로필 기록과 `verification.ts` 사용을 근거로 했고 export 줄을 직접 열지는 않았다 — **미확인**).

2. 두 호출부를 `appendFileSync` → `readFileSync`(또는 이미 읽어 둔 텍스트) + `appendToJournal` + `writeFileSync` 로 바꾼다.
   - `journal.ts:35`: 지금은 원문을 읽지 않으므로 `fs.readFileSync(file, "utf8")` 를 한 번 추가해야 한다. 파일 존재는 바로 위 `fs.existsSync(file)` 분기로 이미 보장된다.
   - `journal-summary.ts:101`: `journal` 변수를 그대로 쓴다. 기존의 `(journal.endsWith("\n") ? "" : "\n") + "\n"` 빈 줄 규칙은 **그대로 보존**할 것 — 블록 사이 빈 줄이 없어지면 `lastSummaryTs`/`## ` 섹션 파싱 기대값이 흔들린다.
3. **정규식으로 공백을 매칭하지 말 것.** 이 저장소는 `\s*` 가 줄바꿈을 먹어 두 번 데였다(`headingTitle`, 메타데이터 줄 경계). 문자열 연결과 `normalizeEol`/`restoreEol` 만 쓴다.
4. `journal.ts:35` 의 **다른 결함**(재활성화 줄이 `## 세션` 이 아니라 파일 끝 = `## 다음 세션을 위한 노트` 에 쌓인다)은 PR #10(`auto/2026-10-01-0103`, 미머지)이 `appendSessionLine` 으로 고치는 중이다. **이번 회차는 EOL 만 고치고 삽입 위치는 건드리지 말 것** — 범위가 커지고 PR #10 과 정면 충돌한다.
   - **착수 시 분기 판단**: `git log --oneline main -5` 와 `src/features/journal.ts` 를 열어 `appendSessionLine` 이 main 에 이미 있는지 확인한다.
     - 있으면(PR #10 머지됨) `journal.ts` 는 이미 EOL 을 보존하므로 **`journal-summary.ts` 한 파일만** 고친다. 그러면 프로덕션 1파일로 끝난다.
     - 없으면(정찰이 확인한 현재 상태) 위 표대로 **두 파일**을 함께 고친다. 한쪽만 고치면 같은 일지 파일에 EOL 보존 writer 와 LF-only writer 가 공존하는 비대칭이 남고, 그것이 운영자가 되풀이해 금지한 "같은 값을 읽는·쓰는 경로가 둘 이상인데 한쪽만 고치기" 다.

- 수용 기준:
  1) 0단계 4개 명령의 종료 코드가 로그 파일에 남고, 전부 0 이면 "PR #11 의 CI 실패는 코드 결함이 아니다" 라는 판정이 증거와 함께 기록된다. 원격 확인 불가는 "미확인" 으로 명시된다.
  2) CRLF 일지(프로덕션 `journalTemplate` 로 만든 문서를 CRLF 로 변환한 것)에 `initJournal` 이 재활성화 줄을 세 번 붙여도 파일 안에 LF 단독 줄이 **0개**다 — `text.split("\r\n").join("")` 에 `\n` 이 남지 않는 것으로 증명한다.
  3) 같은 CRLF 일지에 `appendSessionSummary` 가 요약 블록을 붙여도 LF 단독 줄이 **0개**다. 요약 블록 안의 여러 줄(`- 감사 이벤트: …`, `<!-- vibe-code:summary-until … -->`)까지 CRLF 여야 한다.
  4) LF 일지는 LF 로 남는다(`\r` 이 0개). 기존 동작 회귀 없음.
  5) 요약 블록 앞의 빈 줄 규칙이 보존되어 `lastSummaryTs(journal)` 가 붙인 타임스탬프를 그대로 돌려주고, `hasSection`/`section(text, "세션")` 이 CRLF·LF 양쪽에서 같은 값을 읽는다.
  6) `npm run check` 전체 통과 — typecheck + vitest + esbuild. 테스트 수는 **기준선을 직접 실측해 기록할 것**(main@43fd7a1 기준선은 82 로 보고돼 있으나 회차마다 87/88 로 달라졌다 — 브랜치 베이스를 확인하고 실측치를 쓸 것).

- 건드릴 파일:
  - `src/features/journal-summary.ts` — `appendToJournal`(신규 순수 export) 추가, `appendSessionSummary`(:101)의 `appendFileSync` 를 `appendToJournal` + `writeFileSync` 로 교체. `../util/markdown` 에서 `detectEol`/`normalizeEol`/`restoreEol` import 추가.
  - `src/features/journal.ts` — `initJournal`(:26) 의 else 분기(:35)를 `readFileSync` + `appendToJournal` + `writeFileSync` 로 교체. **PR #10 이 이미 main 에 머지돼 있으면 이 파일은 건드리지 않는다.**
  - `tests/unit/journal-summary.test.ts` — 기존 파일. `appendSessionSummary` 의 CRLF/LF 경계 테스트 추가.
  - `tests/unit/journal-eol.test.ts` — 신규. `initJournal` 경로용. **파일명을 `journal.test.ts` 로 하지 말 것** — PR #10 이 그 이름을 쓰고 있어 충돌한다.
  - 프로덕션 2파일 · 테스트 2파일. 그 이상으로 넓히지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 실패 재현(고치기 전): `npx vitest run tests/unit/journal-summary.test.ts tests/unit/journal-eol.test.ts`
  - 인과 확인: 두 프로덕션 파일만 `git checkout HEAD -- src/features/journal.ts src/features/journal-summary.ts` 로 되돌려 같은 테스트가 다시 빨개지는 것 확인. **새 export 를 테스트에서 직접 import 하면 되돌린 상태에서 import 오류가 섞여 결함 증거가 더러워진다** — 2026-10-01 PR #9 회차가 실제로 이 문제를 겪었다. 테스트는 프로덕션 진입점 `initJournal`/`appendSessionSummary` 만 호출하고 `appendToJournal` 을 직접 import 하지 말 것.
  - 전체: `npm run check` (Node 20.19.2, 위 0단계 절차로 확보)
  - 번들: `node --check dist/extension.js`, `node --check dist/extension.core.js`
  - `initJournal` 은 `vscode` 를 import 하고 `workspaceRoot()` 로 경로를 정한다. 테스트는 `tests/unit/vscode-stub.ts`(vitest alias) 경로를 쓰되 **가짜 `CoreHost` 로 결함을 증명하지 말 것** — `initJournal(host)` 는 `log(host, …)` 에만 host 를 쓰므로 실제 임시 fs + 실제 `journalTemplate` 로 파일 경계를 증명하고, host 는 최소 통과용으로만 쓴다. 기존 테스트 파일들이 `workspaceRoot` 를 어떻게 다루는지 `tests/unit/goal-catalog.test.ts`·`verification.test.ts` 를 먼저 볼 것(정찰은 이 두 파일을 열지 않았다 — **미확인**).
  - 임시 디렉터리를 쓸 때 `TMPDIR` 와 `GIT_CEILING_DIRECTORIES` 를 **명령 범위에만** 지정할 것(상위 Git 저장소를 잘못 발견하는 함정).

- 위험과 피할 것:
  - `.github/workflows/ci.yml` — 읽기만. 조건 완화·`continue-on-error` 제거·엔진 핀 변경 금지(배정이 명시적으로 금지).
  - `src/util/markdown.ts` — 읽기만(PR #6·#7·#8 미머지). `src/features/plans.ts`(PR #9), `src/features/goals.ts`·`src/features/handoff-on-exit.ts`(PR #11), `src/features/goal-lint.ts`(PR #7) — 건드리지 말 것.
  - `src/features/workspace.ts:66` — JSONL 감사 로그. LF 가 정답이다. 고치지 말 것. 감사에 비밀·출력 원문을 추가하지 말 것.
  - `src/features/checkpoints.ts`(임시 Git index/commit-tree/update-ref), `src/features/vibe-coders-proxy.ts`(전역 provider 설정 저장/복원), `vendor/`, `scripts/*.ps1`, `release/` — 보호 경로. 이번 과제와 무관하므로 열지 말 것.
  - 원격 push·CI 재시도·결제 설정 변경은 하지 말 것. 전역 설정을 백업 없이 덮어쓰는 스크립트를 만들지 말 것.
  - `git stash` 를 쓰지 말 것(워크트리 간 stash 스택 공유). 되돌릴 때는 `git checkout HEAD -- <경로>` 만 쓸 것.
  - 소스 문자열 grep 을 증거로 제출하지 말 것. 판정은 실행 로그와 실패→통과 테스트로 한다.
  - 예상되는 회차 결과: **이 회차도 verify-failed 로 끝날 가능성이 높다.** 원격 CI 가 계정 결제/한도로 시작조차 하지 않으면 어떤 코드 변경도 녹색을 만들 수 없다. 여섯 회차 연속 같은 결말의 공통 원인은 코드가 아니라 **원격 판정 수단 부재**이므로, 보고서에 이것을 운영자 조치 항목(결제/사용 한도 해소 + 세션에 `gh` 토큰 제공)으로 분명히 적을 것. 이것을 코드 변경으로 우회하려 하지 말 것.

- 차선 후보: **`src/features/journal-summary.ts` 한 파일만** — 착수 시 PR #10 이 main 에 머지돼 있어 `journal.ts` 가 이미 EOL 을 보존한다면, 또는 `journal.ts` 를 건드리는 것이 PR #10 과 충돌해 위험하다고 판단되면, `appendSessionSummary`(:101) 의 EOL 보존만 고치고 헬퍼를 같은 파일 안에 두어 프로덕션 1파일 · 테스트 1파일로 끝낸다. 이때는 "`journal.ts:35` 가 아직 LF-only 로 남아 있고 PR #10 이 그것을 고치는 중" 임을 보고서에 명시할 것.

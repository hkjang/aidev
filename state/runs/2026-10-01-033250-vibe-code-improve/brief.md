# 과제서 — 2026-10-01-033250-vibe-code-improve

- 과제: 수정 과제 — PR #10(`origin/auto/2026-10-01-0103`, head `c3d1ddb43c08ab8ce1a3a48eb971745d45ccf0e5`) CI 실패를 Node 20 로컬 재현으로 판정하고, 코드 결함이 아니면 목표 파일에 LF 전용으로 덧붙이는 두 경로(`createHandoffFile` / `handoffOnExit`)를 EOL 보존으로 고치기 (가치 3 / 위험 1 / 작업량 S)

## 왜
배정이 "릴리즈 워크플로가 같은 이유로 두 번 실패" 라고 했고 PR #10 이 열려 있다. PR #10 의 코드 변경은 `src/features/journal.ts`(+41)·`tests/unit/journal.test.ts`(+61) 두 파일뿐이며(`git diff --stat main...origin/auto/2026-10-01-0103` 로 직접 확인), 직전 4회차(PR #6·#7·#8·#9)가 모두 같은 0단계에서 "로컬 전부 통과 = 코드 결함 아님" 으로 판정했다. 그러니 0단계는 **시간을 박아서 빠르게 끝내고**, 남은 시간을 실제 결함 하나에 쓴다. 그 결함은 목표 파일 `.vibe-code/goals/current.md` 에 쓰는 세 경로 중 두 개가 줄바꿈을 깨뜨리는 것이다 — `verification.ts:45-46 appendVerificationLog` 는 `detectEol`/`restoreEol` 로 CRLF 를 보존하는데(v1.4.5 가 이 이유로 버전을 올려 릴리즈했다), **같은 파일에 쓰는** `goals.ts:123`(`createHandoffFile`)와 `handoff-on-exit.ts:48`(세션 종료 경로)은 `fs.appendFileSync(current, "\n- …\n")` 로 LF 만 쓴다. CRLF 목표 파일에 LF 줄이 섞여 들어가고, 이 확장의 1차 대상이 Windows(패키징·검증 스크립트가 전부 PowerShell)라 실사용에서 도달한다.

## 0단계 (먼저, 그리고 반드시 로그를 남길 것)
`.github/workflows/ci.yml:13-24` 의 check job 을 같은 순서로 재현한다. 저장소 작업 트리를 더럽히지 말고 PR #10 head 를 밖으로 뽑아서 한다:

```
git archive c3d1ddb43c08ab8ce1a3a48eb971745d45ccf0e5 -o /tmp/pr10.tar && mkdir -p /tmp/pr10 && tar -xf /tmp/pr10.tar -C /tmp/pr10
```
(이 세션의 정찰은 `git archive | tar -x` 파이프가 샌드박스에 막혀 추출 자체를 못 했다. `-o` 로 tar 파일을 만든 뒤 따로 `tar -xf` 하면 된다 — 미검증이니 막히면 `git worktree add` 로 별도 경로에 PR head 를 펴서 하라.)

Node 는 `engines.node` 정확 핀인 **20.19.2** 로. 이 저장소의 기본 node 는 v22.23.1 이라 그대로 쓰면 CI 와 다르다:
```
npm install --no-save node@20.19.2   # 그 뒤 <dir>/node_modules/node/bin 을 PATH 앞에
```
(직전 회차가 이 절차로 동작을 확인했다.)

네 단계를 순서대로, 각 exit code 를 로그에 남긴다:
1. `npm ci`
2. `npm run check` (= `tsc --noEmit` → `vitest run` → `node scripts/build.mjs`)
3. `node --check dist/extension.js`
4. `node --check dist/extension.core.js`

- **하나라도 실패하면 그것이 이번 과제다.** 아래 1단계는 버리고 그 실패를 고쳐라.
- **전부 exit 0 이면** "PR #10 의 원격 CI 실패는 코드 결함이 아니다" 를 로그 경로와 함께 기록하고 **멈추지 말고** 1단계로 간다. `ci.yml` 은 절대 손대지 마라(워크플로 완화 금지).
- 정찰은 원격 상태를 **확인하지 못했다** — 이 세션에 GitHub 접근이 없다(`WebFetch` 권한 거부, `gh` 미인증). PR 번호 ↔ 브랜치 대응(`#10` = `auto/2026-10-01-0103`)은 브랜치 시간순 + 변경 내용에서 온 **추정**이고, 로컬 `origin/auto/2026-10-01-0103` head 가 `c3d1ddb` 인 것만 확인했다. 2026-09-29 회차가 보관한 annotation("The job was not started because recent account payments have failed or your spending limit needs to be increased.", `steps=[]`, `runner_id=0`)이 유일한 원격 증거이고 **이번에 재확인되지 않았다.** 배정이 말한 "같은 이유로 두 번 실패" 도 여전히 미확인이다 — 사실처럼 쓰지 마라.

## 1단계 — 고칠 것
`.vibe-code/goals/current.md` 에 덧붙이는 두 경로를 `appendVerificationLog` 와 같은 계약(입력 파일의 줄바꿈 보존)으로 맞춘다.

- `src/features/goals.ts:123` — `createHandoffFile` 안의
  `fs.appendFileSync(paths.current, "\n- ${stamp.human} - handoff 생성: .vibe-code/sessions/${basename} (${note})\n", "utf8")`
- `src/features/handoff-on-exit.ts:48` — 세션 종료 경로의
  `fs.appendFileSync(current, "\n- ${stamp.human} - handoff 생성: .vibe-code/sessions/${basename} (세션 종료)\n", "utf8")`

두 줄은 **문구가 사실상 같은 한 줄 포맷**이다. 한쪽만 고치면 같은 파일에 쓰는 두 경로가 또 어긋난다(운영자 지시). `src/features/goals.ts` 에 순수 export 함수 하나를 더해 두 곳이 같이 쓰게 하라. 예:

```ts
/** 목표 파일에 한 줄 메모를 덧붙인다. 입력의 줄바꿈을 보존한다. */
export function appendGoalNote(text: string, note: string): string
```
- `detectEol(text)` 로 원본 줄바꿈을 잡고, `normalizeEol(text)` 위에서 붙인 뒤 `restoreEol(…, eol)` 로 되돌린다 — `verification.ts:45-46` 과 같은 패턴.
- 호출부는 `fs.appendFileSync` 대신 `readUtf8` → `appendGoalNote` → `fs.writeFileSync` 로 바꾼다(빈 파일/파일 없음도 다뤄야 한다).
- 기존 출력 문자열(앞 빈 줄 하나 + `- ` 줄 + 끝 개행)은 LF 파일에서 **한 글자도 달라지면 안 된다.** 기존 동작을 바꾸는 게 아니라 CRLF 에서만 달라져야 한다.
- 정규식으로 공백을 먹지 마라 — 이 저장소는 `\s*` 가 줄바꿈을 먹어 두 번 데였다(v1.4.5, `headingTitle`). 필요하면 `[ \t]*`.
- `handoff-on-exit.ts` 가 `goals.ts` 를 import 할 때 **순환 import 가 생기는지 확인**하라(미확인). 생기면 헬퍼를 `handoff.ts`/`templates` 쪽이 아니라 두 파일이 모두 이미 의존하는 곳에 두되, **`src/util/markdown.ts` 에는 두지 마라** — PR #7·#8 이 그 파일을 물고 있어 충돌한다.

## 수용 기준
1. CRLF 로 된 실제 목표 파일(프로덕션 `goalTemplate` 로 만든 문자열을 `\r\n` 으로 바꾼 것)에 handoff 메모를 덧붙였을 때 결과 파일에 `\n` 단독(= `\r` 이 앞에 없는 개행)이 하나도 없다.
2. LF 목표 파일에 대한 결과 바이트가 수정 전과 완전히 동일하다(기존 동작 무회귀).
3. `createHandoffFile`(goals.ts)과 세션 종료 경로(handoff-on-exit.ts) **둘 다** 같은 결과를 낸다 — 한 경로만 검증한 테스트는 불충분하다.
4. 덧붙인 뒤 프로덕션 파서로 읽어 계약이 유지된다: `hasSection`/`section`/`matchLine` 이 기존 섹션·값을 그대로 돌려주고, 이어서 `appendVerificationLog` 를 부르면 `## 검증 로그` 가 **중복 생성되지 않는다**(v1.4.5 가 고친 자리).
5. 실패 재현: 고치기 전에 새 테스트를 돌려 빨간 것을 로그로 남기고(`validation/goal-eol-red.log`), 고친 뒤 통과. 인과 확인으로 프로덕션 파일만 `git checkout HEAD --` 로 되돌려 같은 개수가 다시 빨개지는지 본다.
6. 대역 금지 — 가짜 `CoreHost`/손으로 조립한 목표 문자열로 증명하지 말고, 프로덕션 `goalTemplate` 이 만든 실제 텍스트와 실제 임시 fs 파일로 하라. 임시 경로가 상위 Git 저장소 안에 들어가지 않게 `TMPDIR` 과 `GIT_CEILING_DIRECTORIES` 를 명령 범위에 같이 지정하라(과거에 이 함정에 걸렸다).

## 건드릴 파일 (프로덕션 2 · 테스트 1)
- `src/features/goals.ts:123 createHandoffFile` — `appendGoalNote` 로 교체 + 순수 헬퍼 export 추가
- `src/features/handoff-on-exit.ts:48` — 같은 헬퍼 사용
- `tests/unit/goal-catalog.test.ts` — 이 파일이 이미 `goals.ts`(`saveCurrentGoal` 등)를 덮는다. 신규 테스트를 여기에 붙이거나 새 `tests/unit/goal-handoff.test.ts` 를 만들어도 된다.
- 0단계 결과가 실패면 위 세 파일 대신 그 실패 지점.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
npx vitest run tests/unit/goal-catalog.test.ts tests/unit/verification.test.ts   # 집중 (실패 재현용)
npm run check                                                                    # typecheck + vitest run + esbuild
node --check dist/extension.js
node --check dist/extension.core.js
```
- 기준선: main@43fd7a1 전체 **82 tests**(2026-09-30 회차가 실측). 신규 테스트 수만큼 늘어야 한다.
- `npm run check` 의 build 단계는 런타임 자산 5종이 없어도 경고만 내고 성공한다 — 이건 정상이다.
- `npm run vsix` / `npm run verify` / `npm run smoke:vscode` 는 Windows PowerShell 이 필요하다. 돌리지 마라.

## 위험과 피할 것
- **`.github/workflows/ci.yml` 금지.** 조건 완화·`continue-on-error` 손질·Node 핀 변경 전부 금지. 배정 자체가 "워크플로를 느슨하게 만들어 통과시키는 것 금지" 라고 했다.
- **미머지 PR 이 물고 있는 파일 금지**: `src/util/markdown.ts`(PR #7·#8), `src/features/goal-lint.ts`(#8), `src/features/plans.ts`(#9), `src/features/journal.ts`(#10). 읽기만 하라. 이번 과제 파일(`goals.ts`, `handoff-on-exit.ts`, `journal-summary.ts`, `goal-catalog.test.ts`)은 다섯 브랜치 중 **어느 것도 건드리지 않는다** — `git diff --name-only main...origin/<branch>` 로 이번 정찰이 다섯 브랜치 전부 확인했다.
- `src/features/journal-summary.ts:101` 의 같은 계열 LF-only append 는 **이번엔 건드리지 마라.** 같은 파일(오늘 일지)에 쓰는 다른 경로가 `journal.ts:35` 인데 그건 PR #10 이 이미 고쳐 열려 있어, 지금 손대면 충돌하거나 한쪽만 고치는 꼴이 된다. PR #10 머지 뒤가 맞다.
- `checkpoints.ts`(임시 Git index/commit-tree/update-ref), `vibe-coders-proxy.ts`(전역 설정 저장·복원), `vendor/`, `scripts/*.ps1`, `release/` 는 보호 경로다. 전부 피하라.
- 감사(`workspace.ts:66`)에 원문이나 비밀을 새로 넣지 마라.
- 0단계가 전부 통과했다고 해서 "PR #10 의 원격 CI 가 괜찮다" 고 쓰지 마라. 원격은 이번에도 미확인이다.

## 차선 후보
`buildSessionSummary`/`appendSessionSummary` 의 CRLF 안전화(`src/features/journal-summary.ts:85·101`) — 순수 함수 `buildSessionSummary` 는 이미 export 되어 있고 `tests/unit/journal-summary.test.ts` 가 있어 붙이기 쉽다. **단, PR #10 머지 뒤에** 하라(위 "피할 것" 참조). 그 전에 해야 한다면 `journal.ts` 는 절대 건드리지 말고 `journal-summary.ts` 한 파일로만 끝내고, 과제서에 "일지의 다른 writer 는 PR #10 이 고치는 중" 이라고 명시하라.

- 과제: 수정 과제 — PR #12 head `2a264c8` 로 CI check job 로컬 재현 판정 + `readRecentAudit` 이 "최근 N일" 을 "최근 N개 파일" 로 읽어 지표·회고가 창 밖 데이터를 창 안이라고 보고하는 것 고치기 (가치 4 / 위험 1 / 작업량 S)
- 왜: 배정이 "릴리즈 워크플로가 같은 이유로 두 번 실패" 라고 했으므로 먼저 PR #12 head 에서 CI check job 4단계를 그대로 재현해 코드 결함인지 증거 로그로 판정해야 한다(이 계열은 PR #6~#11 여섯 회차가 모두 '로컬 전부 통과 → 코드 결함 아님' 이었다). 재현되지 않으면 멈추지 말고, `goal-metrics.ts` 의 감사 읽기 경로가 파일 개수로 창을 잡아 두 달 전 감사도 "최근 7일" 로 집계·회고 제목에 올리는 것을 고친다 — 지표와 회고가 사용자에게 거짓 기간을 말하는 정확성 결함이고, 미머지 PR 여섯 건이 물고 있는 파일을 전혀 건드리지 않는다.

## 0단계 — 게이트 (반드시 먼저)

PR #12 = `origin/auto/2026-10-01-0535`, head `2a264c83c52c05b303235c6825f5373457c57c1b`("fix: 오늘 일지에 쓰는 두 writer 가 CRLF 일지의 줄바꿈을 보존한다", 즉 직전 회차 산출물). `git log -1` 과 `git diff --stat main...` 으로 이번 회차가 직접 확인했다(변경: `src/features/journal-summary.ts`, `src/features/journal.ts`, `tests/unit/journal-eol.test.ts`, `tests/unit/journal-summary.test.ts` — 4파일 212+/6-). **PR 번호 ↔ 브랜치 대응은 추정이다**(이 세션에 GitHub 접근 없음).

`.github/workflows/ci.yml:13-24`(이번 회차가 전문을 다시 읽었다) 의 check job 과 같은 순서로, 저장소 밖에 추출해 돌린다:

```
mkdir -p /tmp/node2019 && cd /tmp/node2019
printf '{"name":"n","private":true}\n' > package.json
npm install node@20.19.2
export PATH=/tmp/node2019/node_modules/node/bin:$PATH   # node -v → v20.19.2 확인
cd <이 워크트리> && git archive -o /tmp/pr12.tar 2a264c83c52c05b303235c6825f5373457c57c1b
mkdir -p /tmp/pr12 && tar -xf /tmp/pr12.tar -C /tmp/pr12
cd /tmp/pr12 && npm ci; echo EXIT_npm_ci=$?
npm run check; echo EXIT_npm_check=$?
node --check dist/extension.js; echo EXIT_node_check_extension=$?
node --check dist/extension.core.js; echo EXIT_node_check_core=$?
```

**Node 를 `/tmp/pr12` 안에 설치하지 말 것** — 뒤따르는 `npm ci` 가 `node_modules` 를 지워 exit 127 이 된다(2026-10-01 회차가 실제로 걸렸다). 네 exit 가 전부 0 이면 "PR #12 의 원격 실패는 코드 결함이 아니다" 를 로그(`validation/pr12-node20-check.log`)로 남기고 **멈추지 말고 1단계로** 간다. 하나라도 실패하면 그 실패가 이번 과제다(1단계는 버린다). `ci.yml` 은 읽기만 — 조건 완화 금지.

이 세션은 `npx vitest run` 조차 권한 거부돼 **테스트 기준선을 실측하지 못했다**. 이전 회차 실측값은 main@43fd7a1 = 11 files / 82 tests 이지만 구현자가 직접 실측할 것(추정 금지).

## 1단계 — 감사 읽기 창 (0단계가 통과하면)

- 수용 기준:
  1) `readRecentAudit(auditDir, 7)` 이 **KST 날짜 창**(오늘 포함 7일: `kstDate(new Date(Date.now() - 6*864e5))` 이상)의 파일만 읽는다. 오늘 파일 하나와 8일 전·60일 전 파일이 함께 있는 감사 디렉터리에서 8일/60일 전 항목이 결과에 없다.
  2) 창 안에 파일이 7개 미만이어도 창 안 파일은 전부 읽는다(빈 디렉터리·디렉터리 없음은 지금처럼 `[]`).
  3) `computeGoalMetrics(...).activeDays` 가 **KST 날짜**로 센다(`kstDate(new Date(e.ts))`). 기존 기대값 `expect(m.activeDays).toBe(3)`(goal-metrics.test.ts:36)은 그대로 통과해야 한다 — 픽스처 ts 가 01:00~05:00Z(=KST 10~14시)여서 UTC/KST 날짜가 같다. 이번 회차가 확인했다.
  4) `buildRetro` 의 `# 주간 회고 — <from> ~ <to>` 제목과 집계 대상이 같은 창을 가리킨다(제목 형식은 바꾸지 말 것 — goal-metrics.test.ts:40 이 고정).
  5) 테스트가 증명할 것: 파일 개수가 아니라 **날짜**로 자른다는 것과, 파일명을 만드는 쪽(`writeAudit`)과 읽는 쪽(`readRecentAudit`)이 같은 KST 날짜 규칙을 쓴다는 것.
  6) `npm run check` 전체 통과(typecheck + 기존 테스트 전부 + esbuild) + `node --check` 두 번들 exit 0.

- 건드릴 파일 (프로덕션 1개):
  - `src/features/goal-metrics.ts:17 readRecentAudit` — `.sort().slice(-days)` 를 파일명 문자열 비교(`n.slice(0,10) >= from`)로 바꾼다. 파일명이 `YYYY-MM-DD.jsonl` 이므로 사전순 = 날짜순이고 이미 정규식으로 그 형식만 통과시킨다. `kstDate` 는 이 파일이 이미 import 중(`../util/kst`).
  - `src/features/goal-metrics.ts:59 activeDays` — `e.ts.slice(0,10)`(UTC) → `kstDate(new Date(e.ts))`. 불량 ts 는 `Number.isNaN(d.getTime())` 으로 건너뛸 것(`parseAuditLines` 는 ts 가 string 인지만 본다).
  - `tests/unit/goal-metrics.test.ts` — 신규 `describe`. 기존 4개 기대값(24~46행)은 바꾸지 말 것.

- 검증 명령:
  - `npx vitest run tests/unit/goal-metrics.test.ts` (고치기 전 빨강 확인 → 고친 뒤 초록)
  - `npm run check` (= typecheck + 전체 vitest + esbuild, 약 15초)
  - `node --check dist/extension.js && node --check dist/extension.core.js`
  - 인과 확인: `git checkout HEAD -- src/features/goal-metrics.ts` 로 프로덕션만 되돌려 같은 테스트가 같은 메시지로 다시 빨개지는 것 확인(`git stash` 금지 — 워크트리 공유).

- 테스트 경계 (대역 금지 규칙 준수):
  - 감사 파일은 **프로덕션 `writeAudit(host, kind, action, details)`**(`src/features/workspace.ts:60`)로 쓴다. `workspaceRoot()` 는 `vscode.workspace.workspaceFolders?.[0]?.uri.fsPath` 를 읽으므로, `mkdtempSync` 임시 디렉터리를 `vscode-stub` 의 `workspace.workspaceFolders` 에 넣어 프로덕션 배선을 그대로 지난다(직전 회차 `journal-eol.test.ts` 가 세운 패턴). **주의: main 의 `tests/unit/vscode-stub.ts:4` 는 `workspaceFolders: undefined as undefined` 로 타입이 좁아 대입이 typecheck 에서 막힌다.** 공유 스텁을 고치면 미머지 PR #12 의 테스트와 충돌할 수 있으니, 새 테스트 안에서 국소 캐스트로 대입할 것(스텁 파일 수정은 최후 수단).
  - `writeAudit` 은 항상 **오늘** KST 날짜 파일에 쓴다. 과거 날짜 파일은 그 프로덕션 산출물을 `fs.copyFileSync` 로 과거 날짜 파일명(`2026-XX-XX.jsonl`)에 복사해 만든다 — 손으로 JSONL 을 조립하지 않는다. 읽기는 프로덕션 `readRecentAudit`/`computeGoalMetrics`/`buildRetro` 만 쓴다.
  - 임시 디렉터리를 쓸 때 `TMPDIR` 지정 시 `GIT_CEILING_DIRECTORIES=$TMPDIR` 도 함께. `afterEach` 에서 `rmSync(..., {recursive:true, force:true})`.
  - 새 export 를 만들 필요가 없다(기존 export 만 부른다) — 되돌린 상태에서 import 오류가 섞이지 않아 결함 증거가 깨끗하다.

- 위험과 피할 것:
  - **미머지 PR 여섯 건이 물고 있는 파일 금지**: `src/util/markdown.ts`(#6·#7·#8), `src/features/plans.ts`(#9), `src/features/goals.ts`·`handoff-on-exit.ts`(#11), `src/features/journal.ts`·`journal-summary.ts`(#12), `goal-lint.ts`(#7). `journal-summary.ts` 의 `parseAuditLines`/`AuditEntry` 는 **import 해서 쓰기만** 할 것 — 파서를 통합하거나 옮기지 말 것.
  - 보호 경로 금지: `.github/workflows/ci.yml`(읽기만, 완화 금지), `vendor/`, `scripts/*.ps1`, `release/`, `src/features/checkpoints.ts`, `src/features/vibe-coders-proxy.ts`.
  - `src/features/workspace.ts:66`(감사 JSONL append)은 고치지 말 것 — JSONL 은 LF 가 정답이고, 감사에 원문·비밀을 추가하지 않는다.
  - 공백 매칭에 `\s*` 를 쓰지 말 것(`[ \t]*`) — 이 저장소가 두 번 데인 자리. 이번 과제는 정규식을 새로 쓸 필요가 없다(파일명 비교는 문자열 비교).
  - `formatMetrics`/`buildRetro` 의 **출력 문구·형식은 바꾸지 말 것** — 기존 테스트 6개가 문자열을 고정한다. 창을 고치는 것이지 표시를 바꾸는 것이 아니다.
  - `days` 파라미터의 의미를 "일" 로 확정하므로 함수 주석("the last `days` daily files")도 같이 고칠 것 — 대체된 설명을 남기지 말 것.
  - 프로덕션 1파일·테스트 1파일로 끝내고, 지표 표시 개선·코드블록 집계 등으로 범위를 넓히지 말 것.

- 차선 후보: `computeGoalMetrics` 의 검증 통과율이 `exitCode` 누락 항목을 통과로 센다(`goal-metrics.ts:51-52` — `num(undefined)===0` 이 true 이고 `!== null` 도 true 이므로 `details.exitCode` 가 없는 `runVerification` 항목이 100% 통과에 들어간다). 같은 파일 한 곳 수정 + 테스트 1~2개로 끝나며 위험이 더 낮다. 1순위와 겹치는 파일이라 **둘 중 하나만** 하거나, 1순위를 끝낸 뒤 시간이 남으면 같은 커밋에 얹을 수 있다.

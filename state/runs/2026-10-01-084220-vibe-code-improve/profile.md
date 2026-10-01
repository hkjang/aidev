# vibe-code 프로필 (2026-10-01)
- 목적: 한국어 기본 VS Code AI 코딩 확장. 목표 자율 루프·계획 보드·검증·감사·사용량 대시보드를 `.vibe-code/` 아래 markdown/JSONL 로 관리한다.
- 스택: TypeScript ES2022/commonjs/strict, esbuild(번들), vitest 3, CI Node 20. DB 없음.
- 현재: main@43fd7a1, 버전 1.4.5. **미머지 PR 이 아홉 개 겹쳐 있다**(브랜치는 `git branch -r` 로 확인, PR 번호 대응은 추정):
  - #6 `auto/2026-09-29-1232` · #7 `auto/2026-09-30-1821` · #8 `auto/2026-09-30-2132` · #9 `auto/2026-09-30-2217`
  - #10 `auto/2026-10-01-0103` · #11 `auto/2026-10-01-0332` · #12 `auto/2026-10-01-0535` · #13 `auto/2026-10-01-0632`
  - #14 `auto/2026-10-01-0742` head `bf12461`(2커밋, main 직계: goal-metrics.ts 5/5, journal-summary.ts 18/1, 테스트 2파일)
  - 충돌 구역: `markdown.ts`(#6·#7·#8) · `goal-lint.ts`(#7) · `plans.ts`(#9) · `journal.ts`(#10) · `goals.ts`·`handoff-on-exit.ts`(#11) · `goal-metrics.ts`·`journal-summary.ts`(#12·#13·#14) · 공유 `tests/unit/vscode-stub.ts`.
  - **브랜치가 main 직계이므로 PR #13 의 수정은 main 에 없다** — main 의 `readRecentAudit`(goal-metrics.ts:17)은 여전히 `.slice(-days)` 다. 이전 프로필이 "고쳐졌다" 고 한 것은 오류였다. 미머지 diff 는 매 회차 실측할 것.
  - 충돌면 0 인 파일(2026-10-01 확인): `command-audit.ts`, `usage.ts`, `semver.ts`, `checkpoints.ts`, `goal-catalog.ts`, `verification.ts`, `workspace.ts`.
- 구조:
  - `src/extension.ts` → `src/activation.ts`: 기능 등록. `src/features/`: goals/plans/verification, 트리·상태바, goal-loop/lint/health/catalog/metrics, checkpoints, command-audit, journal/journal-summary, proxy.
  - `src/util/markdown.ts`: 섹션·메타데이터·체크리스트 공통 읽기/편집. `detectEol`/`normalizeEol`/`restoreEol` 로 LF/CRLF 처리(세 개 모두 export 확인).
  - `src/util/kst.ts`: `kstParts`/`kstDate`(`YYYY-MM-DD`)/`kstClock`/`kstStamp`. 감사·일지 파일명이 모두 `kstDate()` 다.
  - `src/features/workspace.ts`: 상태 경로·읽기·감사 공통 관문. `writeAudit`(:60, `AuditKind = goal|plan|proxy|command`)이 `.vibe-code/audit/<kstDate>.jsonl` 에 JSON 한 줄 append.
  - `src/core/`, `vendor/extension.core.js`: 벤더 코어. **임의 수정 금지.** `scripts/build.mjs` 가 dist 로 복사만 한다.
  - `tests/unit/`: 11개 파일(checkpoints, goal-catalog, goal-health, goal-loop, goal-metrics, journal-summary, markdown, plans, semver, usage, verification) + `vscode-stub.ts`. `command-audit.test.ts` 는 **없다**.
  - `vitest.config.ts`: `resolve.alias.vscode` → `tests/unit/vscode-stub.ts`, `include: tests/unit/**/*.test.ts`.
  - `scripts/`: `build.mjs`, `restore-dist-assets.mjs`, `run-extension-host-test.js`, PowerShell 5종. `docs/`: maintenance-guide, source-analysis, 자율 목표, 프록시, improvement-roadmap.
- 빌드·테스트:
  - **이 워크트리에 `node_modules` 가 없다** — 먼저 `npm ci`.
  - `npm run check` = `npm run typecheck && npm run test && npm run build` (약 15초). `npx vitest run tests/unit/<파일>` 로 집중 검사.
  - `node --check dist/extension.js`, `node --check dist/extension.core.js`.
  - **테스트 수는 브랜치마다 다르다**(main 82 로 기록, PR 별 87/88/90). 기준선은 추정하지 말고 실측할 것.
  - `npm run vsix` / `verify` / `smoke:vscode` / `test:extension-host`: Windows PowerShell + 릴리즈 런타임 자산 필요. Linux 회차에서 실행 불가.
- CI 와 같은 Node 확보 (여러 회차가 여기서 헤맸다 — 이대로 할 것):
  ```
  mkdir -p /tmp/node2019 && cd /tmp/node2019
  printf '{"name":"n","private":true}\n' > package.json
  npm install node@20.19.2
  export PATH=/tmp/node2019/node_modules/node/bin:$PATH
  ```
  **대상 디렉터리 안에 설치하지 말 것** — 뒤따르는 `npm ci` 가 `node_modules` 를 지워 `node --check` 가 exit 127 이 된다.
- 워크플로: `.github/workflows/` 에 **`ci.yml` 하나뿐**이다(별도 릴리즈 워크플로 없음). `check`(ubuntu, :13-24) = npm ci / npm run check / node --check 두 번들. `package`(windows-latest, `needs: check`, :26-37) = npm ci / npm run build, 그 뒤 :41-57 이 자산 fetch(`continue-on-error: true`) + `found == 'true'` 게이트. **워크플로 조건 완화 금지.**
- 관례: 영어 conventional prefix + 한국어/영어 요약. 설정은 `package.json` + `package.nls.json`/`package.nls.ko.json` 동시 관리. 마이그레이션 체계 없이 상태 파일 편집. 순수 export + 실제 템플릿·fs·Git 테스트 선호. **가짜 Task/CoreHost/provider/CommandEvent 로 실제 배선을 입증하지 않는다.**
- 위험 구역: `checkpoints.ts`(임시 Git index + commit-tree/update-ref — 작업 트리·index·stash 보존 필수) · `vibe-coders-proxy.ts`(전역 설정 덮어쓰기 금지) · `vendor/`·`.github/workflows/`·`scripts/*.ps1`·`release/` · 감사(`workspace.ts:60`)에 비밀·출력 원문 추가 금지(식별자만).
- 자주 깨지는 곳:
  - `\s*` 정규식이 줄바꿈을 먹는다 — `headingTitle`·메타데이터 줄 경계에서 두 번 데였다. 공백은 `[ \t]*` 로. 가능하면 정규식 대신 문자열 비교로.
  - **"최근 N일" 을 `slice(-N)` 으로 구현한 파일 창 결함이 계열로 있다** — `readRecentAudit`(goal-metrics.ts:21, main 미수정), `readCommandHistory`(command-audit.ts:45, 미수정). 파일명 `YYYY-MM-DD.jsonl` 사전순 비교 + `kstDate` 창이 검증된 수정법이고, 비날짜 파일명을 날짜 정규식으로 **먼저** 배제해야 한다(`"notes.json" > "2026-…"`).
  - `writeSection` 은 본문 전체 교체. `taskLines`/`lines` 로 걸러 쓰면 메모·들여쓰기 유실. `sectionLines` 는 `section().trim()` 영향으로 섹션 첫 줄 들여쓰기를 지운다(기존 기대값 2곳이 고정).
  - **LF-only append.** main@43fd7a1 의 `appendFileSync`: `journal.ts:35`, `journal-summary.ts:101`(PR #11 수정 중), `goals.ts:123`·`handoff-on-exit.ts:48`(PR #11), `workspace.ts:60`(JSONL — LF 가 정답). `verification.ts:45` 만 EOL 보존(v1.4.5).
  - `countChecks`/`isTaskLine` vs `taskLines`/`checkLine` 들여쓰기 계약 불일치 잔존.
  - `exitCode` 누락(`CommandEvent.exitCode` 옵셔널 → `command-audit.ts:20` 이 조건부로만 넣는다)을 읽는 경로가 여럿이고 main 에서 서로 반대로 읽는다(PR #14 가 `auditExitCode` 로 통일 중). 표시 경로 `command-audit.ts:73` 은 미수정 — `! 종료 undefined`.
- 검증 함정:
  - **원격 CI 를 이 환경에서 확인할 수 없다.** `gh` 미인증, push·WebFetch 불가. PR 번호 ↔ 브랜치 대응은 전부 추정이다. PR #6~#14 아홉 회차가 전부 로컬 4단계 exit 0 이었고 머지된 것은 없다.
  - 2026-09-29 run 36518589343 annotation: "The job was not started because recent account payments have failed or your spending limit needs to be increased.", `steps=[]`, `runner_id=0`. 코드/테스트 실패가 아니다 — 운영자 조치(결제/한도 + gh 토큰)가 필요하다.
  - CI `package` job 의 Windows `npm ci`/`npm run build` 는 아홉 회차 동안 **미확인**. 그 뒤 단계는 조용히 skip 될 수 있다 — **녹색 CI 가 패키징 성공을 증명하지 않는다.**
  - `node-version: 20`(20.20.x) vs `engines.node` 핀 20.19.2 드리프트는 `npm warn EBADENGINE` 뿐 exit 0 — 실패 원인이 아니다.
  - `npm run build` 는 dist 런타임 자산이 없어도 경고 후 성공한다.
  - 임시 테스트 경로가 상위 Git 저장소 안이면 "저장소 아님" 테스트가 잘못된 저장소를 발견한다. `TMPDIR` 지정 시 `GIT_CEILING_DIRECTORIES=$TMPDIR` 도 지정.
  - `vscode-stub` 는 import 경계용. `workspace.workspaceFolders` 를 임시 디렉터리로 국소 캐스트 대입하면 프로덕션 `workspaceRoot()` 를 그대로 지난다(대역 아님). 공유 스텁 파일 자체는 고치지 말 것.
  - 과거 날짜 감사 파일은 `copyFileSync` 손조립 대신 `vi.useFakeTimers({toFake:["Date"]})` + `vi.setSystemTime` 으로 프로덕션 `writeAudit` 이 파일명과 `ts` 를 일관되게 만들게 할 것.
  - 워크트리 환경이므로 `git stash` 금지(스택 공유). 되돌리기는 `git checkout HEAD -- <경로>`.
  - 새 export 를 테스트에서 직접 import 하면 되돌린 인과 확인에서 import 오류가 섞여 증거가 더러워진다. 가능하면 기존 진입점만 호출하고, 안 되면 "추출만 한 상태" 의 빨간 로그를 증거로 쓸 것.

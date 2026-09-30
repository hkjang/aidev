# vibe-code 프로필 (2026-10-01)
- 목적: 한국어 기본 VS Code AI 코딩 확장. 목표 자율 루프·계획 보드·검증·감사·사용량 대시보드를 `.vibe-code/` 아래 markdown/JSONL 로 관리한다.
- 스택: TypeScript ES2022/commonjs/strict, esbuild(번들), vitest 3, CI Node 20. DB 없음.
- 현재: main@43fd7a1, 버전 1.4.5. **미머지 PR 이 여섯 개 겹쳐 있다**(브랜치 head 는 로컬 origin 기준으로만 확인, PR 번호 대응은 추정):
  - #6 `auto/2026-09-29-1232` markdown 메타데이터 줄 경계 · #7 `auto/2026-09-30-1821` headingTitle/goal-lint
  - #8 `auto/2026-09-30-2132` advancePlanText 열린 항목 · #9 `auto/2026-09-30-2217` (plans.ts)
  - #10 `auto/2026-10-01-0103` journal.ts appendSessionLine · #11 `auto/2026-10-01-0332` head `af9b04b` goals/handoff EOL
  - 따라서 `markdown.ts`·`plans.ts`·`goals.ts`·`handoff-on-exit.ts`·`goal-lint.ts`·`journal.ts` 는 모두 충돌 위험 구역이다.
- 구조:
  - `src/extension.ts` → `src/activation.ts`: 기능 등록. `src/features/`: goals/plans/verification, 트리·상태바, goal-loop/lint/health/catalog/metrics, checkpoints, journal/journal-summary, proxy.
  - `src/util/markdown.ts`: 섹션·메타데이터·체크리스트 공통 읽기/편집. `normalizeEol`/`detectEol`/`restoreEol` 로 LF/CRLF 처리.
  - `src/features/workspace.ts`: 상태 경로·읽기·감사 공통 관문(`:66` 이 audit JSONL append).
  - `src/core/`, `vendor/extension.core.js`: 벤더 코어. **임의 수정 금지.** `scripts/build.mjs` 가 dist 로 복사만 한다.
  - `tests/unit/`: 11개 파일(checkpoints, goal-catalog, goal-health, goal-loop, goal-metrics, journal-summary, markdown, plans, semver, usage, verification) + `vscode-stub.ts`(vitest alias).
  - `scripts/`: `build.mjs`, `restore-dist-assets.mjs`, `run-extension-host-test.js`, PowerShell 5종(package/verify/smoke/host/icon).
  - `docs/`: maintenance-guide, source-analysis, 자율 목표, 프록시, improvement-roadmap.
- 빌드·테스트:
  - `npm ci` → `npm run check` = `npm run typecheck && npm run test && npm run build` (약 15초).
  - `npx vitest run tests/unit/<파일>` 로 집중 검사.
  - `node --check dist/extension.js`, `node --check dist/extension.core.js`.
  - **테스트 수는 브랜치마다 다르다**(main 82, PR 별로 87/88/146). 기준선은 추정하지 말고 실측할 것.
  - `npm run vsix` / `npm run verify` / `npm run smoke:vscode` / `npm run test:extension-host`: Windows PowerShell + 릴리즈 런타임 자산 필요. Linux 회차에서 실행 불가.
  - `node scripts/restore-dist-assets.mjs --from <vsix>`: dist 런타임 자산 복원.
- CI 와 같은 Node 확보 절차 (여러 회차가 여기서 헤맸다 — 이대로 할 것):
  ```
  mkdir -p /tmp/node2019 && cd /tmp/node2019
  printf '{"name":"n","private":true}\n' > package.json
  npm install node@20.19.2
  export PATH=/tmp/node2019/node_modules/node/bin:$PATH
  ```
  **대상 디렉터리 안에 설치하지 말 것** — 뒤따르는 `npm ci` 가 `node_modules` 를 지워 pinned node 가 사라지고 `node --check` 가 exit 127 이 된다. 빈 디렉터리에서 `npm install --no-save` 는 package.json 이 없으면 아무것도 설치하지 않고 exit 0 을 낸다.
- 관례: 영어 conventional prefix + 한국어/영어 요약. 설정은 `package.json` + `package.nls.json`/`package.nls.ko.json` 동시 관리. 마이그레이션 체계 없이 상태 파일 편집. 순수 export + 실제 템플릿·fs·Git 테스트 선호. **가짜 Task/CoreHost/provider 객체로 실제 배선을 입증하지 않는다.**
- 위험 구역:
  - `checkpoints.ts`: 임시 Git index + commit-tree/update-ref. 작업 트리·index·stash 보존 필수.
  - `vibe-coders-proxy.ts`: provider 설정 저장/복원. 전역 설정 덮어쓰기 금지.
  - `vendor/`, `.github/workflows/`, `scripts/*.ps1`, `release/`: 활성화·릴리즈 영향. **워크플로 조건 완화 금지.**
  - 감사(`workspace.ts:66`)에는 비밀·출력 원문을 추가하지 않는다. 식별자만 넘긴다.
- 자주 깨지는 곳:
  - `\s*` 정규식이 줄바꿈을 먹는다 — `headingTitle`·메타데이터 줄 경계에서 두 번 데였다. 공백은 `[ \t]*` 로.
  - `writeSection` 은 본문 전체 교체. `taskLines`/`lines` 로 걸러 쓰면 메모·들여쓰기 유실.
  - `sectionLines` 는 `section().trim()` 영향으로 섹션 첫 줄 들여쓰기를 지운다. 기존 기대값 2곳이 이 동작을 고정.
  - **LF-only append.** main@43fd7a1 의 `appendFileSync` 전체: `journal.ts:35`, `journal-summary.ts:101`(둘 다 같은 일지 파일, 미수정), `goals.ts:123`·`handoff-on-exit.ts:48`(PR #11 수정 중), `workspace.ts:66`(JSONL — LF 가 정답). `verification.ts:45` 만 EOL 보존(v1.4.5).
  - `countChecks`/`isTaskLine` vs `taskLines`/`checkLine` 들여쓰기 계약 불일치 잔존.
- 검증 함정:
  - **원격 CI 를 이 환경에서 확인할 수 없다.** `gh` 미인증이고 2026-10-01 정찰에서는 `gh auth status` 실행 자체가 권한 거부됐다. push·WebFetch 도 불가. PR 번호 ↔ 브랜치 대응은 전부 추정이다.
  - 2026-09-29 run 36518589343 annotation: "The job was not started because recent account payments have failed or your spending limit needs to be increased.", `steps=[]`, `runner_id=0`. 코드/테스트 실패가 아니다. PR #6~#11 여섯 회차 모두 로컬 4단계 exit 0.
  - CI `package` job(`ci.yml:26-37`, windows-latest, `needs: check`)의 `npm ci`/`npm run build` 는 여섯 회차 동안 재현된 적이 없다 — 미확인. 그 뒤 Package/Verify/SHA256/Upload 는 `ci.yml:41-57` 의 `continue-on-error: true` + `found == 'true'` 게이트로 조용히 skip 될 수 있다. **녹색 CI 가 패키징 성공을 증명하지 않는다.**
  - `node-version: 20`(20.20.x) vs `engines.node` 정확 핀 20.19.2 드리프트는 `npm warn EBADENGINE` 뿐이고 exit 0 — 실패 원인이 아니다(2026-09-30 로그 확인).
  - `npm run build` 는 dist 런타임 자산(node_modules, i18n, workers, tree-sitter.wasm, tiktoken_bg.wasm)이 없어도 경고 후 성공한다.
  - 임시 테스트 경로가 상위 Git 저장소 안이면 "저장소 아님" 테스트가 잘못된 저장소를 발견한다. `TMPDIR` 지정 시 `GIT_CEILING_DIRECTORIES=$TMPDIR` 도 지정.
  - `vscode-stub` 는 import 경계용. Linux check 는 실제 Extension Host·VSIX 실행을 검증하지 않는다.
  - 워크트리 환경이므로 `git stash` 금지(스택 공유). 되돌리기는 `git checkout HEAD -- <경로>`.
  - 새 export 를 테스트에서 직접 import 하면 프로덕션 파일을 되돌린 인과 확인에서 import 오류가 섞여 결함 증거가 더러워진다. 테스트는 프로덕션 진입점만 호출할 것.

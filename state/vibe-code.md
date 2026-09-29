## 2026-09-23
- 선택: 파괴적 명령 전 체크포인트가 untracked 파일까지 담도록 고치고 첫 테스트를 붙이기 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `createCheckpoint`의 `git stash create`(추적 변경만 담김)를 `os.tmpdir()` 아래 임시 index 기반 스냅샷(`read-tree`→`add -A`→`write-tree`→`commit-tree`)으로 교체해, 아직 `git add` 하지 않은 새 파일도 `rm -rf` 직전에 보호되게 했다. 스냅샷 커밋은 stash 형식이 아니므로 노트의 복원 블록을 `git diff`/`git checkout <commit> -- .` 로 바꾸고, 커밋이 없는 저장소는 부모 없는 커밋으로 처리했으며, 설정 설명(ko/en)과 `docs/autonomous-goal-workflow.md` 문구를 정정했다. 검증은 `fs.mkdtempSync`로 만든 **진짜 git 저장소**를 통과하는 신규 `tests/unit/checkpoints.test.ts` 9개(가짜 git 대역 없음 — 복원 블록은 노트에서 명령을 파싱해 실제로 실행해 파일 복구를 확인)로 했고, 고치기 전 4개가 예상대로 실패(untracked 누락·`git stash apply` 잔존·빈 저장소)하는 것을 확인한 뒤 고쳤다. `npm run check` 전체 통과(53 tests, typecheck, esbuild 빌드).
- 보류 아이디어: plans.ts(523줄, 테스트 0개) 순수 함수 테스트 보강 — selectPlanName/planMeta/sortByPriority/linkedPlans/openPlanItems (3/1/M) · computeGoalMetrics가 `details.exitCode` 없는(undefined) 검증 항목을 통과로 집계 — 실제 감사 로그에 그런 줄이 나오는지 먼저 확인 필요 (2/1/S) · vibe-coders-proxy.ts provider 저장/복원 경로 테스트 — vscode 설정 API 스텁 확장이 필요해 대역 위주가 되기 쉬움 (3/2/M) · `createCheckpoint`의 catch가 `update-ref` 실패 시에도 "git 저장소가 아님" 노트를 남겨 commit/note가 어긋남 (2/1/S) · 거대한 untracked 디렉터리(.gitignore 미등록 node_modules 등)가 있을 때 동기 `git add -A` 지연 측정과 가드 (2/2/M)
- 과제서: 채택 — 과제서의 근거(stash create가 untracked를 제외, stash apply가 비-stash 커밋 거부)를 실제 git 2.43으로 재현해 확인한 뒤 그대로 구현했다.

- 릴리즈: v1.4.1 (2026-09-23, run 2026-09-23-123422-vibe-code-improve)
## 2026-09-26
- 선택: archiveDonePlans 가 같은 이름의 보관 계획을 말없이 지우는 것을 막고 보관 로직을 테스트 가능한 함수로 분리 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `vibe-code.archiveDonePlans` 는 `archive/<이름>` 이 이미 있으면 `fs.unlinkSync(dest)` 로 기존 보관본을 지운 뒤 덮어써서, "계획 → done → 보관" 루프만 반복해도 과거 계획 기록이 휴지통 없이 사라졌다. 보관 루프를 vscode 에 의존하지 않는 `archiveDonePlans(plansDir, archiveDir, stamp)` 로 분리하고 충돌 시 `restoreArchivedPlan` 과 같은 방식(`path.parse` + 14자리 숫자 스탬프)으로 `<이름>-<stamp>.md` 에 보관하도록 바꿨으며, 스탬프 생성을 `conflictStamp()`(KST) 로 통일해 두 경로가 같은 형식을 쓰게 했다. 검증은 `fs.mkdtempSync` 로 만든 실제 디렉터리 트리에서 도는 신규 `tests/unit/plans.test.ts` 10개로 했고, 고치기 전에 "같은 이름 두 번 보관" 테스트 2개가 기존 보관본 내용이 사라지는 형태로 실패하는 것을 먼저 확인한 뒤 고쳤다(가짜 fs·가짜 vscode 명령 대역 없음). `npm run check` 전체 통과 — typecheck + 63 tests(기준선 53 + 신규 10) + esbuild 빌드.
- 보류 아이디어: vibe-coders-proxy.ts provider 저장/복원 경로 테스트 — 스텁 확장 없이 순수 함수로 뺄 수 있는지 먼저 확인 (3/2/M) · plans.ts 나머지 순수 함수(planMeta, sortByPriority, linkedPlans) 테스트 보강 (2/1/S) · createCheckpoint 의 catch 가 update-ref 실패를 '저장소 아님' 으로 뭉개 audit 의 commit/note 가 어긋남 (2/1/S) · 거대한 untracked 디렉터리에서 동기 `git add -A` 지연 측정 후 필요할 때만 가드 (2/2/M) · 보관 후 `.active-plan` 포인터가 옮겨진 이름을 가리킨 채 남음 — 이제 보관 함수가 분리돼 싸게 얹을 수 있음 (2/1/S)
- 과제서: 채택 — 과제서의 근거(275줄 unlinkSync, restoreArchivedPlan 과의 정책 모순)를 코드에서 그대로 확인했고, 수용 기준 1~3 을 실패 재현 후 구현으로 모두 충족했다.

- 릴리즈: v1.4.2 (2026-09-26, run 2026-09-26-070059-vibe-code-improve)
## 2026-09-27
- 선택: CRLF `.vibe-code/*.md` 에서 섹션 파서·에디터(section/subsection/writeSection/moveTaskToSection)가 전부 실패하는 것 고치기 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `src/util/markdown.ts` 의 섹션 정규식이 제목 뒤 `\n` 하나만 허용해 CRLF 파일에서 `section`/`subsection` 은 빈 문자열, `writeSection` 은 입력 그대로, `moveTaskToSection` 은 `null` 을 돌려주고(CodeLens "완료로 이동" 이 조용히 무동작) 린트는 섹션마다 "없습니다" 를 쏟아냈다. 읽기 계열은 진입부에서 `normalizeEol` 로 CRLF→LF 정규화 후 기존 정규식을 그대로 쓰고, 편집 계열(`writeSection`/`moveTaskToSection`/`toggleCheckbox`/`touchPlan`/`setLine`)은 `detectEol`(혼합 파일은 첫 줄바꿈 기준, VS Code 규칙) 로 원본 줄바꿈을 기억해 편집 후 복원하도록 바꿨다 — 체크 한 번에 파일 전체가 LF 로 바뀌던 문제도 같이 사라졌다. 섹션 존재 검사는 `hasSection` 하나로 모아 `goal-lint.ts` 의 별도 정규식을 없애 파서를 하나로 만들었다. 검증은 프로덕션 템플릿(`planTemplate`/`goalTemplate`)과 실제 `lintPlan`/`lintGoal`/`parseGoal`/`openPlanItems` 를 그대로 통과하는 신규 테스트 7개(가짜 계획 문자열·대역 없음)로 했고, `npm run check` 전체 통과 — typecheck + 70 tests(기준선 63 + 신규 7) + esbuild 빌드.
- 실패 재현: 고치기 전 `npx vitest run tests/unit/markdown.test.ts` → `Tests 6 failed | 13 passed` — `AssertionError: expected '' to be '- [ ] A\n- [ ] B'`(writeSection), `AssertionError: expected null not to be null`(moveTaskToSection), `expected '# 계획: 테스트 목표\n\n상태: draft…' to be '# 계획: 테스트 목표\r\n\r\n상태: draft…'`(toggleCheckbox 가 CRLF 를 LF 로 바꿈), `expected [ {…}, …(3) ] to deeply equal []`(린트 섹션 누락 오경고). 정찰이 남긴 `crlf-check.js` 도 먼저 돌려 CRLF 열에서 section 빈 값·moveTaskToSection null·lint 미검출을 눈으로 확인했다(다만 이 스크립트는 정규식 복제본이라 수정 후 검증에는 쓰지 않았다).
- 보류 아이디어: countChecks 와 taskLines 와 plan-codelens 의 isTaskLine 이 들여쓴 체크리스트 항목을 세 갈래로 다르게 판정 (3/1/S) · moveTaskToSection 이 대상 섹션을 lines() 로 재조립해 들여쓰기·빈 줄을 날림 (2/1/S) · vibe-coders-proxy.ts provider 저장/복원 경로 테스트 — 스텁 확장 없이 순수 함수로 뺄 수 있는지 먼저 확인 (3/2/M) · createCheckpoint 의 catch 가 update-ref 실패를 '저장소 아님' 으로 뭉개 audit 의 commit/note 가 어긋남 (2/1/S) · 보관 후 `.active-plan` 포인터가 옮겨진 이름을 가리킨 채 남음 (2/1/S)
- 과제서: 채택 — 재현 스크립트 출력이 과제서의 판정표와 정확히 일치했고, 수용 기준 1~5 를 실패 재현 후 프로덕션 파일 2개로 모두 충족했다.

- 릴리즈: v1.4.3 (2026-09-27, run 2026-09-27-041201-vibe-code-improve)
## 2026-09-27
- 선택: advanceCurrentPlan 이 Now/Next 의 체크리스트 아닌 줄을 말없이 지우는 것 막고 advance 를 순수 함수로 분리 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `vibe-code.advanceCurrentPlan` 은 Now/Next 를 `taskLines(section(...))` 로 — 체크리스트 줄만, 그것도 trim 된 형태로 — 읽은 뒤 `writeSection` 으로 섹션 본문을 통째로 덮어써, 사용자가 손으로 적어 둔 메모 줄("참고: …")과 들여쓴 하위 항목이 "계획 한 단계 진행" 한 번에 사라졌다(이 명령은 체크포인트 대상이 아니라 되돌릴 수단도 없다). `markdown.ts` 에 섹션을 원본 줄 그대로 읽는 `sectionLines()`(빈 섹션은 `[""]` 이 아니라 `[]`)를 더하고, advance 의 텍스트 변환을 vscode 에 의존하지 않는 `advancePlanText(text, stamp)` 로 뽑아내 섹션 재조립 대신 항목만 splice 하도록 고쳤으며, 같은 뿌리인 `moveTaskToSection` 의 대상 섹션 재조립도 같은 방식으로 바꿨다(CodeLens "완료로 이동" 이 Done 의 들여쓴 줄을 지우던 것). 검증은 프로덕션 `planTemplate(...)` 이 만든 실제 계획 문자열과 거기에 `writeSection` 으로 메모/들여쓴 줄을 더한 문자열을 순수 함수에 그대로 넣는 신규 테스트 8개(가짜 fs·가짜 vscode 대역 없음, CRLF 케이스 포함)로 했고 `npm run check` 전체 통과 — typecheck + 78 tests(기준선 70 + 신규 8) + esbuild 빌드.
- 실패 재현: 고치기 전 `npx vitest run tests/unit/plans.test.ts tests/unit/markdown.test.ts` → `Tests 3 failed | 35 passed (38)` — `AssertionError: expected '- [ ] 하위 항목\n- [ ] 뒤이어 실행할 작업 1개 정의' to be '참고: Now 메모\n  - [ ] 하위 항목\n  참고: 들여쓴 …'`(advance 가 Now 의 메모 줄을 지우고 들여쓰기를 날림), `AssertionError: expected '- [x] 지난 작업\n참고: 들여쓴 완료 메모\n…' to be '- [x] 지난 작업\n  참고: 들여쓴 완료 메모\n…'`(Done 재조립), 그리고 `moveTaskToSection` 의 대상 섹션 들여쓰기 소실.
- 보류 아이디어: verification.ts:41 appendVerificationLog 도 검증 로그 섹션을 lines() 로 재조립해 들여쓴 출력·코드블록을 깬다 — 이제 sectionLines() 가 있어 한 줄로 바뀐다 (2/1/S) · countChecks·taskLines·plan-codelens 의 isTaskLine 이 들여쓴 체크리스트 항목을 세 갈래로 판정해 진행률과 트리 항목 수가 어긋난다 (3/1/S) · matchLine 의 `\s*` 가 빈 라벨 줄에서 다음 줄 값을 읽는다 (2/1/S) · advanceCurrentPlan 이 이미 `- [x]` 인 Now 첫 항목도 다시 완료 처리한다 — 기대 동작 판단이 먼저 필요 (2/2/S) · vibe-coders-proxy.ts provider 저장/복원 경로 테스트 — 스텁 확장 없이 순수 함수로 뺄 수 있는 부분부터 (3/2/M)
- 과제서: 채택 — 과제서가 미확인으로 남긴 유실 시나리오를 실제 테스트로 먼저 재현했고(메모 줄·들여쓰기 소실), 수용 기준 1~4 를 프로덕션 파일 2개로 모두 충족했다.

- 릴리즈: v1.4.4 (2026-09-27, run 2026-09-27-000241-vibe-code-improve)
## 2026-09-28
- 선택: `appendVerificationLog` 가 CRLF 파일에서 `## 검증 로그` 를 못 찾아 섹션을 중복 생성하고 `checkLine` 이 파일 전체를 LF 로 바꾸는 것 고치기 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `src/features/verification.ts` 가 v1.4.3 의 CRLF 통합에서 빠진 자체 정규식 `/(?:^|\n)## 검증 로그\n/` 을 들고 있어, CRLF 워크스페이스에서 검증을 돌릴 때마다 `## 검증 로그` 섹션이 파일 끝에 하나씩 더 붙고(goals.ts:104 는 첫 섹션만 읽으므로 "최근 검증" 이 영원히 갱신되지 않는다) `checkLine` 은 체크 한 번에 파일 전체를 LF 로 바꿨다. 섹션 판정을 `hasSection` 하나로 모으고, 기존 기록 줄은 `lines()` 대신 `sectionLines()` 로 원본 그대로 읽어 들여쓰기·빈 줄을 지키며, 신규 섹션 생성 경로와 `checkLine` 은 `detectEol`/`restoreEol`(markdown.ts 에서 export 만 추가)로 입력의 줄바꿈을 복원하도록 고쳤다. 검증은 프로덕션 템플릿 `goalTemplate`/`planTemplate` 을 `.replace(/\n/g,"\r\n")` 한 실제 문자열에 `appendVerificationLog` 를 3회 연속 적용하고 프로덕션 파서 `sectionLines()`/`normalizeEol()` 로 확인하는 신규 테스트 4개(대역·직접 조립 문자열 없음)로 했고, `npm run check` 전체 통과 — typecheck + 82 tests(기준선 78 + 신규 4) + esbuild 빌드.
- 실패 재현: 고치기 전 `npx vitest run tests/unit/verification.test.ts` → `Tests 4 failed | 6 passed (10)` — `AssertionError: expected 3 to be 1`(CRLF 목표 파일에 3회 호출 → 검증 로그 섹션 3개), `AssertionError: expected 2 to be 1`(CRLF 계획 파일 → 2개. 정찰이 미확인으로 남긴 시나리오대로 1회차가 LF 꼬리 섹션을 만들고 2회차가 그것을 매칭해 전체를 CRLF 로 되돌린 뒤 3회차에서 다시 중복), `expected false to be true`(출력에 CRLF 아닌 줄이 있음), `expected [ …(4) ] to deeply equal [ …(5) ]` — 받은 값이 `"출력: 78 tests passed"`(들여쓰기 소실) 이고 빈 줄이 사라짐.
- 보류 아이디어: countChecks·taskLines·plan-codelens 의 isTaskLine·checkLine 이 들여쓴 체크리스트 항목을 네 갈래로 판정 (3/1/S) · matchLine 의 `\s*` 가 빈 라벨 줄에서 다음 줄 값을 읽는다 — 이번 차선 후보였다 (2/1/S) · advanceCurrentPlan 이 이미 `- [x]` 인 Now 첫 항목도 다시 완료 처리한다 (2/2/S) · vibe-coders-proxy.ts provider 저장/복원 경로 테스트 — 스텁 확장 없이 순수 함수로 뺄 수 있는 부분부터 (3/2/M) · createCheckpoint 의 catch 가 update-ref 실패를 '저장소 아님' 으로 뭉갠다 (2/1/S)
- 과제서: 채택 — 과제서가 미확인으로 남긴 "CRLF 계획 파일 2·3회차 중복" 시나리오를 3회 연속 호출 테스트로 먼저 재현(섹션 2개)했고, 수용 기준 1~5 를 모두 충족했다. 다만 `restoreEol` 을 verification.ts 에서 쓰려면 markdown.ts 의 `function` 을 `export function` 으로 바꿔야 해 프로덕션 파일이 1개가 아니라 2개가 됐다(둘째는 한 단어 변경).

- 릴리즈: v1.4.5 (2026-09-28, run 2026-09-28-211209-vibe-code-improve)
## 2026-09-29
- 선택: 빈 메타데이터의 줄 경계를 지켜 인접 정보 삭제와 잘못된 완료 보관을 막기 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: matchLine/setLine/touchPlan의 값·앵커 매칭을 한 물리적 줄로 제한하고 빈 값은 trim 후 호출자 fallback을 반환하도록 했다. archiveDonePlans도 matchLine의 상태를 사용하며 실제 템플릿·파서·린트·임시 fs에서 LF/CRLF, 공백·탭, EOF, 원문 보존을 검증했다. npm ci 완료, 집중 102 tests와 npm run check(typecheck + 전체 146 tests + esbuild) 통과; 프로덕션 2파일과 테스트 2파일만 변경했으며 원본 코드 복원 시 신규 64개가 다시 실패하는 인과 검증도 수행했다.
- 실패 재현: `Tests 62 failed | 22 passed (84)` (metadata-red.log); `AssertionError: expected [ 'plan-2.md', 'plan-3.md', …(1) ] to deeply equal [ 'plan-4.md', 'plan-5.md' ]` (archive-red.log).
- 보류 아이디어:
  - countChecks/taskLines/isTaskLine/checkLine 들여쓰기 계약 통일 (3/2/M) — 별도 소비자 계약 결정 필요.
  - 빈 제목의 다음 줄 흡수와 제목 린트 동시 수정 (3/1/S) — 이번 메타데이터 계약과 분리.
  - sectionLines 첫 줄 들여쓰기 보존 (3/2/S) — 섹션 경계 계약과 기존 편집 경로 검증 필요.
  - provider 저장/복원 실제 배선 테스트 (3/2/M) — 가짜 객체 없이 검증할 경계 필요.
- 과제서: 채택 — 기존 집중 38 tests 기준선에서 빈 메타데이터의 인접 줄 흡수·삭제 및 실제 파일 오보관이 모두 재현되어 지정된 범위를 구현했다.


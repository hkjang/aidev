- 과제: `appendVerificationLog` 가 CRLF 파일에서 `## 검증 로그` 를 못 찾아 섹션을 중복 생성하고, `checkLine` 이 파일 전체를 LF 로 바꾸는 것 고치기 (가치 4 / 위험 2 / 작업량 M)

- 왜: v1.4.3 이 CRLF 대응을 `src/util/markdown.ts` 한 곳에 넣었지만 `src/features/verification.ts:40` 은 그 파서를 쓰지 않고 자기 정규식 `/(?:^|\n)## 검증 로그\n/` 을 그대로 들고 있다 — CRLF 파일의 `## 검증 로그\r\n` 에는 `\n` 뒤가 `\r` 라 절대 매칭되지 않으므로, 검증을 돌릴 때마다 파일 끝에 `## 검증 로그` 섹션이 하나씩 더 붙는다. 그런데 `src/features/goals.ts:104` 의 `lines(section(text, "검증 로그")).slice(-5)` 는 정규화 후 **첫 번째** 섹션만 읽으므로, 사용자는 검증을 아무리 돌려도 목표 뷰의 "최근 검증" 이 갱신되지 않고 파일만 지저분해지는 것을 본다. 고치면 CRLF 워크스페이스(Windows 체크아웃)에서 검증 기록이 제자리에 쌓이고, 섹션 판정 경로가 `hasSection` 하나로 모인다.

- 수용 기준:
  1) CRLF 계획/목표 파일에 `appendVerificationLog` 를 두 번 이상 적용해도 `## 검증 로그` 섹션은 **하나**이고 기록 줄이 그 섹션 안에 순서대로 쌓인다.
  2) `appendVerificationLog` / `checkLine` 을 거친 결과가 **입력의 줄바꿈을 그대로 유지**한다 (CRLF 입력 → CRLF 출력, LF 입력 → LF 출력). 현재 `checkLine` 은 `split(/\r?\n/)` 후 `join("\n")` 이라 CRLF 파일을 통째로 LF 로 바꾼다.
  3) 검증 로그 섹션에 들여쓴 줄이나 빈 줄이 있어도 새 줄을 덧붙이는 과정에서 사라지지 않는다 (현재 41줄의 `lines(section(...))` 가 trim + 빈 줄 제거로 날린다 — `sectionLines()` 가 이미 있으니 한 줄 교체).
  4) 위 1~3 을 증명하는 테스트가 **고치기 전에 실패**하는 것을 먼저 확인하고(실패 출력을 회차 노트에 남길 것), 고친 뒤 통과한다.
  5) 기존 LF 동작은 그대로 — `tests/unit/verification.test.ts` 의 기존 6개 테스트가 수정 없이 통과한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `src/features/verification.ts:38-44 appendVerificationLog` — ① 40줄의 자체 정규식을 `hasSection(text, "검증 로그")` 로 교체(`../util/markdown` 에서 import 추가) ② 41줄의 `lines(section(text, "검증 로그"))` 를 `sectionLines(text, "검증 로그")` 로 교체 ③ 43줄의 섹션 신규 생성 경로가 `detectEol(text)` 를 써서 원본 줄바꿈으로 붙이도록(현재 `\n\n## 검증 로그\n...` 하드코딩이라 CRLF 파일에 LF 꼬리가 생긴다).
  - `src/features/verification.ts:46-51 checkLine` — `normalizeEol` 로 읽고 `detectEol` 로 기억한 줄바꿈으로 복원. `markdown.ts` 의 `toggleCheckbox`(175줄)가 이미 정확히 이 형태이니 그대로 따를 것. 판정은 지금처럼 `trim()` 후 `/^- \[ \] /` 를 유지(들여쓴 항목 지원) — 들여쓰기 판정 통일은 이번 범위 밖.
  - `tests/unit/verification.test.ts` — CRLF 케이스 추가. **프로덕션 템플릿을 그대로 쓸 것**: 이미 import 된 `goalTemplate("s")`(50줄에 `## 검증 로그` 가 있음)와 `planTemplate("t","p.md","s")`(없음)를 `.replace(/\n/g,"\r\n")` 해 두 경로를 모두 덮고, 검증은 `section()`/`sectionLines()` 같은 프로덕션 파서로 확인할 것(문자열 직접 조립·대역 금지 — 이 저장소의 v1.4.1~1.4.4 가 모두 이 형태다).

- 검증 명령:
  1) `npm ci` (워크트리에 `node_modules` 가 없다 — 반드시 먼저)
  2) 고치기 전: `npx vitest run tests/unit/verification.test.ts` → 새 테스트가 실패하는 것을 확인하고 출력을 기록
  3) 고친 뒤: `npx vitest run tests/unit/verification.test.ts`
  4) 마지막: `npm run check` (typecheck + vitest 78 tests 기준선 + esbuild)

- 위험과 피할 것:
  - `runAndRecord`(97줄)·CodeLens·`registerVerification` 은 vscode 의존이므로 **건드리지 말 것**. 순수 함수 2개만 고친다.
  - `writeSection` 은 섹션 본문을 통째로 교체한다 — 걸러진 배열을 넘기면 그 줄이 파일에서 사라진다(v1.4.3/v1.4.4 가 세 번 걸린 자리). 반드시 `sectionLines()` 로 원본 줄을 받아 뒤에 붙일 것.
  - `section()` 은 본문을 trim 해 돌려주므로 빈 섹션은 `""` 이고 `"".split("\n")` 은 `[""]` — 그래서 `sectionLines()` 가 있다. 직접 split 하지 말 것.
  - `writeSection`/`toggleCheckbox` 는 이미 EOL 을 복원한다. `appendVerificationLog` 안에서 `restoreEol` 을 **이중으로** 적용하면 `\r\r\n` 이 된다(`markdown.ts:19` 주석 참고) — 정규화한 텍스트에만 복원할 것.
  - 보호 경로(`.github/workflows/`, `scripts/*.ps1`, `release/`, `vendor/`, `src/core/hooks.ts`)는 이 과제와 무관하니 손대지 말 것.
  - **미확인**: CRLF 계획 파일에서 섹션이 새로 생긴 뒤(LF 꼬리) 두 번째 호출의 동작은 `npm ci` 를 돌릴 수 없어 실행으로 확인하지 못했다 — LF 꼬리 덕에 가드가 통과했다가 `writeSection` 의 `restoreEol` 이 전체를 CRLF 로 되돌려 세 번째 호출에서 다시 중복될 것으로 읽었다. 구현자는 **이 시나리오를 3회 연속 호출 테스트로 먼저 재현**한 뒤 고칠 것. 반면 "CRLF 섹션 헤더에 40줄 정규식이 매칭되지 않는다" 는 정규식 리터럴만으로 확정적이다(`\n` 뒤에 `\r` 이 오므로).
  - `goals.ts:104` 는 **읽지만 고치지 말 것** — 중복 섹션이 사라지면 저절로 맞는다. 파일 수를 늘리지 않는다.

- 차선 후보: `matchLine` 의 `\s*` 가 빈 라벨 줄에서 다음 줄 값을 읽는 것 고치기 (`src/util/markdown.ts:35` — `'^라벨:\s*(.+)$'` 의 `\s` 는 `\n` 을 포함하므로 `상태:` 가 비어 있으면 다음 줄이 값이 된다. `[^\S\n]*` 로 좁히고 빈 값은 `fallback` 으로. `setLine`/`touchPlan` 의 같은 패턴도 함께.)

# 과제서 — 2026-09-30-213215-vibe-code-improve

- 과제: 수정 과제 — PR #7(`origin/auto/2026-09-30-1821`, head `2ac2a0d`) CI 실패가 코드 결함인지 로컬 재현으로 판정하고, 결함이 아니면 회차를 빈손으로 끝내지 말고 지정된 후속 수정(`headingTitle`)까지 마치기 (가치 4 / 위험 2 / 작업량 M)
- 왜: 러너가 이번 회차를 'verify-failed — CI failed, PR open #7' 로 배정했지만, 이 저장소에서 2026-09-29 에 똑같은 배정이 이미 한 번 '결제/사용 한도로 job 미시작(코드 실패 아님)' 으로 판명되어 소스 변경 0개(no-change)로 끝났다. 판정은 반드시 하되, 같은 결론이 나올 때 회차가 또 빈손이 되지 않도록 소스에서 확인된 실제 결함 하나를 같은 회차에 붙인다.

## 0단계(필수 게이트) — CI `check` job 을 PR #7 head 에서 그대로 재현
`.github/workflows/ci.yml:13-24` 의 check job 과 같은 순서로 돌린다. **이 정찰 세션은 `npm ci` 실행 권한이 없어 재현하지 못했다 — 아래 판정은 전부 미확인이다.**

```
git fetch origin auto/2026-09-30-1821
git worktree add ../pr7-check 2ac2a0d      # 또는 git archive 로 별도 디렉터리 추출
cd ../pr7-check
npm ci
npm run check                               # = tsc --noEmit && vitest run && node scripts/build.mjs
node --check dist/extension.js
node --check dist/extension.core.js
```

- **실패하면** 그 실패가 이번 회차의 과제다. 실패 출력을 그대로 붙이고 원인을 고친다. `ci.yml` 의 조건·`continue-on-error`·`if:` 를 완화해 통과시키는 것은 금지.
- **전부 exit 0 이면** 코드 결함이 아니다. 원장에 "PR #7 head 로컬 재현 전부 통과 / 원격 실패는 외부 blocker" 를 증거 로그와 함께 적고 **1단계로 넘어간다.** 여기서 멈추지 말 것.

### 0단계에서 이미 확인된 사실 (정찰이 읽어서 확인한 것 — 다시 조사하지 말 것)
- PR #7 = `origin/auto/2026-09-30-1821`, head `2ac2a0d`, main@43fd7a1 위 커밋 **1개**. 변경은 `src/util/markdown.ts`(+19/-6)와 `tests/unit/markdown.test.ts`(+69)뿐.
- 그 diff 전문을 읽었다: `isDoneTask`/`countChecks`/`isTaskLine` 의 앵커를 `^` → `^[ \t]*` 로 넓히고 `toggleCheckbox` 의 replace 두 줄을 들여쓰기 캡처(`/^([ \t]*)- \[[xX]\]/` → `"$1- [ ]"`)로 바꾼 것. 타입 오류로 보이는 것은 없고 주석만 추가됐다. 즉 typecheck·build 를 깨뜨릴 구석이 눈에 보이지 않는다.
- `.npmrc` 가 없다(`ls -a` 로 확인). 따라서 `package.json:14` 의 `"node": "20.19.2"` 정확 핀은 CI 의 `node-version: 20`(최신 20.x)과 어긋나도 `engine-strict` 가 꺼져 있어 **EBADENGINE 경고만** 나고 `npm ci` 를 실패시키지 않는다 — 이것은 실패 원인이 아니다.
- PR #6(`origin/auto/2026-09-29-1232`)도 여전히 열려 있고 `src/util/markdown.ts`·`tests/unit/markdown.test.ts`·`src/features/plans.ts`·`tests/unit/plans.test.ts` 를 바꾼다. **두 PR 의 hunk 는 겹치지 않는다**(`git diff -U0` hunk 헤더로 확인: PR #6 는 markdown.ts 33/35/126/130/141/144 와 test 21·240 뒤 삽입, PR #7 는 markdown.ts 94/96/109/111/169-180 와 test 6·233 뒤 삽입). 즉 "두 PR 이 서로 충돌해서 막혔다" 는 가설은 **틀렸다** — 그 방향으로 시간 쓰지 말 것.
- 2026-09-29 회차가 보관한 CI annotation 원문: "The job was not started because recent account payments have failed or your spending limit needs to be increased." 당시 job 은 `steps=[]`, `runner_id=0` — 실행된 단계가 없었다.
- 이 정찰 세션은 `gh`·WebFetch·`npm ci` 전부 권한 거부를 받았다. **PR #7 의 지금 원격 상태(check 결과, annotation)는 미확인이다.** 구현자가 `gh pr checks 7` / `gh run view --log-failed` 를 쓸 수 있으면 먼저 그것으로 실패 원문을 확보할 것 — 계정 차단 문구가 또 나오면 그것이 답이고, 코드를 고칠 것이 없다.

## 1단계 — `headingTitle` 이 빈 제목에서 다음 줄을 제목으로 읽는 것 고치기
`src/util/markdown.ts:41`(실제로 열어 확인):
```ts
const match = normalizeEol(text).match(new RegExp("^#\\s*" + escapeRegExp(prefix) + ":\\s*(.+)$", "m"));
```
`m` 플래그만 있고 `s` 는 없으므로 `.` 는 `\n` 을 안 먹지만 **`\s*` 는 `\n` 을 먹는다.** 그래서 `# 목표:` 다음 줄이 `상태: draft` 면 제목이 `"상태: draft"` 로 읽히고, `fallback` 은 절대 쓰이지 않는다.

- 수용 기준:
  1) `headingTitle("# 목표:\n\n상태: draft\n", "목표", "현재 목표")` 가 `"현재 목표"`(fallback)를 돌려준다. `"상태: draft"` 가 아니다.
  2) 제목 뒤 공백·탭이 있는 정상 입력(`# 목표: 제목  `)과 CRLF 입력의 기존 동작이 그대로다 — 기존 `tests/unit/markdown.test.ts:59`, `:190` 의 기대값이 수정 없이 통과한다.
  3) 테스트는 손으로 조립한 문자열이 아니라 프로덕션 템플릿(`goalTemplate`/`planTemplate`)이 만든 실제 문서와, 거기서 제목만 비운 변형으로 증명한다. 실패 재현(고치기 전 빨간 것)을 먼저 보여 줄 것.
  4) 파일 이름 경로도 같이 증명한다: `src/features/goal-catalog.ts:100` 이 `slugify(headingTitle(text, "목표", "goal"))` 로 파일명을 만들므로, 빈 제목 문서는 다음 줄이 아니라 `goal` 에서 온 이름을 얻어야 한다. 이 함수를 순수하게 호출할 수 있으면 그것으로, 아니면 최소한 `headingTitle` 반환값이 `"goal"` 임을 보인다.
- 건드릴 파일:
  - `src/util/markdown.ts:40-43 headingTitle` — 정규식의 `\s*` 두 곳을 `[ \t]*` 로 좁힌다(`^#[ \t]*` 와 `:[ \t]*`). `\s*` 를 쓰지 말 것 — 이 저장소가 2026-09-30 에 같은 이유로 `[ \t]*` 를 골랐다.
  - `tests/unit/markdown.test.ts` — 신규 테스트 3~4개. **파일 끝이 아니라 기존 `describe("headingTitle"…)`/제목 관련 블록 근처에 넣지 말고, 40~60 행대 기존 제목 테스트 뒤에 붙이는 것이 자연스럽다**(아래 위험 참조).
- 검증 명령:
  - `npx vitest run tests/unit/markdown.test.ts` (집중, 실패 재현용)
  - `npm run check` (typecheck + vitest run 전체 + esbuild 빌드)
  - `node --check dist/extension.js`, `node --check dist/extension.core.js`
- 위험과 피할 것:
  - `goal-lint.ts` 의 제목 검사가 같은 뿌리를 쓰는지는 **미확인**이다. 이번 수정으로 빈 제목이 fallback 을 돌려주면 린트가 "제목 없음" 경고를 내야 하는지 여부는 제품 계약 판단이 필요하다 — 린트 쪽까지 넓히지 말고 `headingTitle` 만 고치고, 린트 동작이 바뀌면 그 사실만 원장에 적을 것.
  - `matchLine`(markdown.ts:35 부근)은 PR #6 가 이미 같은 계열로 고친 자리다. **`matchLine`/`setLine`/`touchPlan` 은 건드리지 말 것** — PR #6 와 같은 줄을 고치면 머지 충돌이 난다. `headingTitle`(41행)은 PR #6 의 35행 hunk 와 컨텍스트가 가깝다. 수정 범위를 41행 한 줄로 최소화하고 주변 줄을 재정렬하지 말 것.
  - `tests/unit/markdown.test.ts` 의 import 블록(1-25행)과 파일 끝(233·240행 뒤)은 PR #6·#7 가 각각 이미 쓰는 자리다. import 추가가 필요 없다면(`headingTitle` 은 이미 6행에서 import 됨) 손대지 말 것.
  - `.github/workflows/`, `scripts/*.ps1`, `release/`, `vendor/`, `webview-ui/build/` 는 손대지 않는다. 워크플로 조건 완화는 금지.
  - grep 결과를 증거로 내지 말 것. 실패 재현 출력과 통과 출력을 붙일 것.
- 차선 후보: `moveTaskToSection`/`advancePlanText` 가 `line.trim()` 으로 항목을 옮겨 들여쓴 하위 항목의 들여쓰기를 잃는 문제 (3/2/S). 단 PR #7 가 들여쓴 항목에 CodeLens 를 붙인 뒤라 계약 판단("하위 항목을 옮길 때 들여쓰기를 지켜야 하는가")이 먼저 필요하고, `markdown.ts:198` 은 PR #7 hunk(169-180)와 가깝다.

## 프로필 델타 (별도 profile.md 는 쓰지 않았다 — 1일 전 프로필이 대부분 맞다)
- main 은 여전히 43fd7a1(v1.4.5). **열린 PR 이 둘이다**: #6 = `auto/2026-09-29-1232`(metadata 줄 경계), #7 = `auto/2026-09-30-1821` head `2ac2a0d`(들여쓴 체크리스트 인정). 둘 다 main 에 없고 서로 충돌하지 않는다.
- 검증 함정 추가: 정찰/구현 세션의 도구 권한이 회차마다 다르다. 이번 정찰은 `npm ci`·`gh`·WebFetch 가 모두 거부돼 로컬 검증과 원격 조회를 하나도 못 했다. 권한이 없으면 "미확인" 으로 적고 추측을 사실로 쓰지 말 것.

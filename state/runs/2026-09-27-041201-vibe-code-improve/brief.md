# 과제서 — 2026-09-27

- 과제: CRLF 줄바꿈 `.vibe-code/*.md` 에서 섹션 파서·에디터(`section`/`subsection`/`writeSection`/`moveTaskToSection`)가 전부 실패하는 것을 고치기 (가치 5 / 위험 2 / 작업량 M)

- 왜: `src/util/markdown.ts` 의 섹션 정규식은 제목 뒤에 `\n` 하나만 허용하는데(`"(?:^|\\n)## " + name + "\\n"`), 같은 파일의 `lines`/`sectionOfLine`/`toggleCheckbox` 는 `/\r?\n/` 로 CRLF 를 이미 받아들인다. 그래서 CRLF 파일에서는 `matchLine`·`headingTitle`·`countChecks`(모두 `m` 플래그 + `.trim()` 이라 `\r` 를 흘려보냄)는 정상 동작하는데 섹션 계열만 전부 빈 값을 돌려준다 — 제목과 "3/4" 진행률은 맞게 보이는데 단계·Now·Next·Done 이 전부 비어 보이고, 린트는 "`## 단계` 섹션이 없습니다" 를 섹션마다 쏟아내며, CodeLens 의 "완료로 이동"·"Now로 승격" 은 **오류도 안내도 없이 아무 일도 안 한다**(`moveTaskToSection` 이 `null` 을 돌려주면 `plan-codelens.ts:editDocument` 가 조용히 반환하고 감사 로그도 안 남는다). 고치면 Windows·zip 공유 경로에서 계획 보드와 인라인 동작이 제대로 산다.

## 근거 (실제로 확인한 것 / 추측인 것)

확인한 것 — 아래는 코드를 열어 정규식을 그대로 읽고 판정한 결과다:
- `src/util/markdown.ts:19` `section` — `(?:^|\n)## <name>\n([\s\S]*?)(?=\n## |$)`. CRLF 본문의 실제 바이트는 `## Now\r\n` 이라 이름 뒤 `\n` 요구가 `\r` 에서 깨진다 → `""`.
- `src/util/markdown.ts:25` `subsection` — `### ` 판본으로 동일하게 깨진다.
- `src/util/markdown.ts:31` `writeSection` — 같은 정규식으로 `re.test(text)` 가 false → **입력을 그대로 반환**(조용한 무동작).
- `src/util/markdown.ts:139` `moveTaskToSection` — 대상 섹션 확인 정규식 `(?:^|\n)## <target>\n` 이 false → `null` 반환.
- `src/features/goal-lint.ts:30`, `:52` — `new RegExp("(?:^|\\n)## " + name + "\\n")` 로 섹션 존재를 검사하므로 CRLF 에서 모든 섹션을 "없음" 으로 신고한다.
- 반대쪽(깨지지 않는 쪽): `matchLine`(`:9`)·`headingTitle`(`:14`) 는 `m` 플래그라 `$` 가 `\n` 앞에서 맞고 `(.+)` 가 `\r` 를 먹은 뒤 `.trim()` 이 떼어낸다. `countChecks`(`:64`) 는 `^- \[( |x|X)\] ` 로 줄 끝을 보지 않아 영향 없다. → **같은 파일을 두 파서가 다르게 읽는 상태**다.

미확인 (구현자가 먼저 재현할 것):
- 위 판정을 **실행으로** 확인하지는 못했다(이 정찰 세션에서 `node` 실행이 승인되지 않았다). 재현 스크립트를 `/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-27-041201-vibe-code-improve/crlf-check.js` 에 남겨 뒀다 — `node crlf-check.js` 로 LF/CRLF 출력을 나란히 볼 수 있다. **구현 전에 반드시 먼저 돌려 위 표를 눈으로 확인하고, 어긋나면 과제를 재판단할 것.**
- CRLF 가 실제로 생기는 경로: `.gitattributes` 가 없고 `.gitignore` 에 `.vibe-code` 가 없어 상태 파일이 커밋 대상이며, `docs/autonomous-goal-workflow.md:11` 이 "Windows와 zip 공유" 를 명시적 설계 목표로 둔다. Windows 의 `core.autocrlf=true` 체크아웃과 zip 왕복이 유력한 경로지만, 실제 사용자 파일이 CRLF 가 된 사례를 본 것은 아니다(미확인).

## 수용 기준

1) CRLF 텍스트를 넣었을 때 `section`·`subsection`·`headingTitle`·`matchLine`·`countChecks`·`taskLines` 가 **같은 내용의 LF 텍스트와 완전히 같은 값**을 돌려준다(섹션 본문 문자열에 `\r` 가 남지 않는다).
2) CRLF 계획 텍스트에서 `writeSection` 과 `moveTaskToSection` 이 LF 일 때와 같은 편집 결과를 낸다 — 특히 `moveTaskToSection` 이 `null` 이 아니라 옮겨진 텍스트를 돌려준다.
3) 편집 함수(`writeSection`/`moveTaskToSection`/`toggleCheckbox`/`touchPlan`/`setLine`)가 **입력 파일의 줄바꿈을 보존**한다 — CRLF 를 넣으면 CRLF 가 나온다. (지금 `toggleCheckbox:129`/`moveTaskToSection:141` 은 `join("\n")` 으로 파일 전체를 말없이 LF 로 바꿔 버려, 체크 하나 눌렀는데 git diff 가 전 줄로 뜬다. 같이 잡을 것.)
4) `lintPlan`/`lintGoal` 이 CRLF 정상 계획에 대해 "섹션이 없습니다" 경고를 내지 않는다.
5) 테스트가 증명할 것: 고치기 **전에** CRLF 케이스가 실패하는 것을 먼저 확인한 뒤 고친다. 기존 `tests/unit/markdown.test.ts` 의 LF 단정은 하나도 바뀌지 않아야 한다(회귀 없음의 증거).

## 건드릴 파일 (프로덕션 2개)

- `src/util/markdown.ts` — 읽기 계열(`section`, `subsection`, `headingTitle`, `matchLine`, `countChecks`)은 진입부에서 `\r\n`→`\n` 정규화 후 기존 정규식을 그대로 쓴다. 쓰기 계열(`writeSection`, `moveTaskToSection`, `toggleCheckbox`, `touchPlan`, `setLine`)은 진입부에서 원본 EOL 을 감지→정규화→편집→원본 EOL 로 복원한다. 파일 안에 `detectEol(text)`/`normalizeEol(text)` 작은 헬퍼 두 개를 export 해 테스트로 직접 찌를 수 있게 할 것.
- `src/features/goal-lint.ts:30`, `:52` — 섹션 존재 검사 정규식. `markdown.ts` 에 `hasSection(text, name)` 를 하나 export 해서 양쪽이 **같은 판정 함수**를 쓰게 하는 편이 좋다(지금은 린트가 자기 정규식을 따로 들고 있어 파서가 둘이다).
- `tests/unit/markdown.test.ts` — 신규 `describe("CRLF")`. 기존 LF 케이스를 `describe.each([["LF","\n"],["CRLF","\r\n"]])` 로 감싸 두 EOL 에서 같은 값이 나오는지 한 번에 보는 것이 가장 강한 증거다. `planTemplate`/`goalTemplate` 을 `.replace(/\n/g, eol)` 해서 입력을 만들 것 — 손으로 만든 가짜 계획 문자열 말고 **프로덕션 템플릿 함수**를 쓸 것(운영자 지침: 대역 금지).

건드리지 말 것: `src/features/plans.ts`, `goals.ts`, `plan-codelens.ts` 는 이번에 손대지 않는다 — 전부 `markdown.ts` 헬퍼를 경유하므로 헬퍼만 고치면 따라온다.

## 검증 명령

```
cd /home/hkjang/.cache/auto-improve-wt/vibe-code
node /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-27-041201-vibe-code-improve/crlf-check.js   # 먼저 재현
npm ci            # 워크트리에 node_modules 가 없다. 느림(수 분)
npx vitest run tests/unit/markdown.test.ts   # 반복 루프용
npm run check     # 최종: typecheck + vitest run(기준선 63개) + esbuild build
```

## 위험과 피할 것

- **두 파서를 한쪽만 고치지 말 것.** `matchLine`/`countChecks` 는 지금도 CRLF 에서 "맞게" 동작한다. 정규화를 넣은 뒤 이 둘의 반환값이 **LF 입력에서 한 글자도 안 바뀌는지** 기존 테스트로 확인할 것. 린트의 별도 정규식(`goal-lint.ts:30,52`)을 그대로 두면 파서가 둘로 남는다 — 같은 입력을 같은 값으로 읽는지 end-to-end 로 볼 것.
- `section()` 의 `(?=\n## |$)` 에서 `$` 는 `m` 플래그가 **없어** 문자열 끝만 뜻한다. 정규화 헬퍼를 넣다가 실수로 `m` 을 붙이면 섹션 본문이 첫 줄에서 잘린다 — 기존 "reads whole multi-line sections" 테스트가 이걸 잡는다.
- 쓰기 경로의 EOL 복원은 `replace(/\n/g, eol)` 같은 단순 치환으로 하면 이미 `\r\n` 인 곳이 `\r\r\n` 이 될 수 있다. 정규화된 텍스트에만 적용할 것.
- 혼합 EOL 파일(일부 LF, 일부 CRLF)에서 `detectEol` 이 무엇을 고를지 정하고 테스트로 못 박을 것(다수결 또는 첫 줄 기준 — 하나만 고르고 문서화).
- 보호 경로 회피: `vendor/`(수정 금지), `.github/workflows/`, `scripts/*.ps1`, `release/`, `src/features/checkpoints.ts`, `src/features/vibe-coders-proxy.ts` 는 이번 과제와 무관하다. 손대지 말 것.
- `npm run vsix`/`verify`/`smoke:vscode`/`test:extension-host` 는 PowerShell 전용이라 리눅스에서 안 돈다 — 통과했다고 쓰지 말 것.
- 사용자에게 보이는 문자열·주석은 한국어, 커밋 메시지는 영어 conventional commit.

## 차선 후보

**`countChecks` 와 `taskLines` 가 들여쓴 체크리스트 항목을 다르게 센다** (가치 3 / 위험 1 / 작업량 S) — `countChecks`(`markdown.ts:64`)는 `^- \[` 로 줄 시작에 고정돼 `  - [ ] 하위 작업` 을 안 세는데, `taskLines`(`:45`)는 `lines()` 가 먼저 `trim()` 하므로 센다. 그래서 하위 작업을 들여쓴 계획은 린트가 "Now에 미완료 4개" 라고 경고하는 동시에 진행률은 그 항목들을 빼고 계산한다. 추가로 `plan-codelens.ts:29` 는 `isTaskLine(line)` 에 **트림하지 않은** 원문을 넘겨 들여쓴 항목에 렌즈를 안 달지만, `moveTaskToSection:138` 은 `line.trim()` 으로 받아들인다 — 판정이 세 갈래다. 1순위가 성립하지 않으면(= `crlf-check.js` 재현이 위 설명과 다르면) 이것을 하되, **세 경로를 한꺼번에** 같은 판정 함수로 모을 것.

- 과제: ATX 제목의 닫는 `#` 시퀀스를 제목에서 떼기 — `## 분기 요약 ##` 이 덱 이름까지 `분기 요약 ##` 으로 만드는 것 (가치 2 / 위험 1 / 작업량 S)
- 왜: `atxHeading`(server/internal/docs/prose.go:422)은 선두 `#` 개수만 세고 줄 끝의 `#` 런을 제목 글자로 그대로 둔다. 그래서 `## 분기 요약 ##` 은 슬라이드 제목이 `분기 요약 ##` 이 되고, 그것이 첫 제목이면 `deckWriter.cover`(writer.go:52)를 거쳐 **덱 이름**과 모든 `!source` 인용 locator 까지 `분기 요약 ##` 이 된다. CommonMark 의 닫는 시퀀스 규칙을 같은 헬퍼 안에서 적용하면 프로덕션 한 자리로 끝나고, 사용자가 손으로 지울 일이 없어진다.

- 실측 재현(이번 정찰이 임시 테스트로 실제 실행해 확인, 2026-10-02):
  - `"## 분기 요약 ##\n매출이 늘었습니다.\n"` → `# 분기 요약 ##` / `@cover` / `> 월간 보고서.md` … `!source 월간 보고서.md | 분기 요약 ##`
  - `"# 제목 #########\n매출이 늘었습니다.\n"` → 제목이 `제목 #########`
  - `"# 제목 ## \n…"` → 제목이 `제목 ##` (줄 끝 공백은 이미 TrimSpace 됨)
  - `"## ###\n…"` → 제목이 `###` 이고 제목 줄은 `# \###` 로 escapeLine 이 지킨다(CommonMark 은 이것을 **빈 제목**으로 읽는다)
  - 건드리면 안 되는 것 — 현 동작이 이미 맞다: `"# C# 도입\n…"` → `C# 도입` 유지, `"# 제목#\n…"` → `제목#` 유지(앞에 공백이 없으면 닫는 시퀀스가 아니다)

- 수용 기준:
  1) 끝의 `#` 런 **앞에 공백/탭이 있을 때만** 떼어 낸다: `## 분기 요약 ##` → 제목 `분기 요약`, `# 제목 #########` → `제목`. 첫 제목이면 덱 이름(`@cover` 줄)과 `!source … | …` locator 에도 `##` 이 남지 않는다.
  2) 닫는 시퀀스가 아닌 `#` 는 한 글자도 건드리지 않는다: `# C# 도입` → `C# 도입`, `# 제목#` → `제목#`. 선두 `#` 개수 판정(1~6)과 "`#` 뒤 공백 또는 줄 끝" 규칙(70f65fe)은 그대로. `####### 일곱개` 는 여전히 제목이 아니다.
  3) 글자가 전부 `#` 인 본문(`## ###`)은 CommonMark 대로 **빈 제목**으로 읽는다 — 빈 제목의 출력 모양은 이미 `TestAnEmptyHeadingReadsExactlyAsItDidBefore`(markdownhash_test.go:102)에 고정돼 있으니, 새 테스트는 그 모양을 실행으로 확인해 문자열을 그대로 박을 것(이 정찰은 `## ###` 의 **고친 뒤** 출력을 실행으로 확인하지 않았다 — 미확인).
  4) 테스트가 증명할 것: 고치기 전 코드에서 1) 이 red(제목에 `##` 이 남고 덱 이름까지 그렇다)이고, 2) 의 세 입력은 전후 모두 green(계약 고정)이라는 것.

- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `server/internal/docs/prose.go:atxHeading` — `rest` 를 돌려주기 전 닫는 시퀀스를 뗀다. 권장 형태(헬퍼 추가 없이 끝남):
    ```go
    text := strings.TrimSpace(rest)
    if closing := strings.TrimRight(text, "#"); closing != text &&
        (closing == "" || closing[len(closing)-1] == ' ' || closing[len(closing)-1] == '\t') {
        text = strings.TrimSpace(closing)
    }
    return text, true
    ```
    `closing == ""` 가 수용 기준 3(글자가 전부 `#`), 마지막 글자가 공백인지 보는 쪽이 수용 기준 1·2 를 함께 만족한다. 함수 앞 주석의 밀도(현재 400~421행의 산문)를 맞춰 **왜** 공백을 요구하는지 한 단락 더할 것 — 이 파일의 관례다.
  - `server/internal/docs/markdownhash_test.go` — 새 테스트 함수 2~3개를 **이 파일에 덧붙인다**(닫는 `#` 은 70f65fe 가 만든 같은 헬퍼의 같은 규칙이므로 새 파일을 만들 이유가 없다). 기존 6개 테스트는 한 글자도 바꾸지 말 것. 단언은 이 저장소 관례대로 `Read(...)` 의 `document.Source` 전체를 문자열로 비교하고, 테스트 이름은 영어 문장형("A heading's closing hashes are not part of its name").

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd server && go test -count=1 ./internal/docs` — 이 정찰이 실행함, **0.003s~2.8s, ok**
  - `cd server && go test -race ./...` — 전 패키지(`golden` 포함)
  - `cd server && go vet ./...`
  - `cd server && gofmt -l internal/docs` — 출력 없어야 함
  - `git diff --check`
  - 웹·API·배포 매니페스트 변경이 없으면 `make test` 의 웹 단계는 건너뛴다(최근 회차 관례). PTIUM_TEST_DSN 이 없으면 DB 테스트는 Skip 된다.

- 위험과 피할 것:
  - `internal/deck/source.go`(덱 DSL 의 `#` 제목)는 docs 의 마크다운 리더와 **다른 계약**이다. 이 규칙을 거기로 옮기지 말 것 — `golden` 회귀가 그 위에 있다.
  - `writer.go`(`deckWriter` 는 docx·pdf·markdown 세 리더가 공용), `tables.go`, 숫자 파서(`amountOf`·`allNumeric`·`bareFigure`)와 `deck` 의 `parseNumber`·`parseBareNumber`·`chartFields` 는 한 글자도 건드리지 말 것(2026-09-09 에 두 번 반려된 자리).
  - `underlinesHeading`(setext)과 `fenceOf`/`closesFence`(펜스)는 손대지 말 것. 미닫힘 펜스 재생 루프(prose.go:302)도 같은 `handle` 을 부르므로 새 규칙이 자동으로 적용된다 — 거기에도 테스트 한 개를 두면 좋지만 선택이다.
  - 백슬래시로 지킨 닫는 시퀀스(`# 제목 \###`)는 CommonMark 이 `제목 ###` 로 읽지만, 위 권장 형태는 끝 글자가 `\` 라서 떼지 않고 **현 동작을 유지**한다. 이번 범위 밖이라고 커밋 메시지에 적을 것.
  - 보호 경로(auth·httpapi 인증·db/migrations.go·scripts/release.sh)는 전혀 닿지 않는다. VERSION(현재 **1.69.53**, base main@70a0c4e)·릴리즈 노트·`deploy/kubernetes.yaml` 은 바꾸지 말 것.

- 차선 후보: **마크다운 수평선(`---` / `***` / `___`)이 `\---` 라는 요점 글머리로 남는 것** (가치 3 / 위험 2 / 작업량 S). 이번 정찰이 실행으로 확인: `"# 제목\n---\n매출이 늘었습니다.\n"` → `- \---` 가 요점으로 들어가고, YAML front matter 가 있는 문서(`"---\ntitle: 보고서\n---\n# 제목\n…"`)는 파일명으로 이름 붙은 슬라이드에 `- \---` / `- title: 보고서` / `- \---` 세 줄이 쌓인다. `readMarkdown` 의 `handle`(prose.go:218)에서 `isListLine` 보다 **앞**에, 글자가 전부 같은 `-`·`*`·`_` 이고 공백이 없으며 3개 이상인 줄만 버리는 분기를 두면 끝난다(`- - -` 형태는 일부러 제외해 목록 판정과 부딪히지 않게 할 것). 표 머리 밑의 `|---|---|` 를 버리는 `isRule`(prose.go:465)과 같은 논리이고, setext h2(`---`)는 애초에 읽지 않으므로 제목 판정은 바뀌지 않는다 — 다만 front matter 의 `title: 보고서` 는 여전히 요점으로 남으므로 "front matter 를 다 건너뛴다" 고 주장하지 말 것.

- 과제: 마크다운 setext 제목(제목 다음 줄의 `===`)을 제목으로 읽기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `readMarkdown`(server/internal/docs/prose.go:205)은 `#` 제목만 안다. 그래서 제목을 `제목` + 다음 줄 `=====` 로 쓴 문서는 제목이 사라지고 밑줄이 불릿으로 한 줄 더 붙는다 — 문서 첫 줄이 그렇게 쓰여 있으면 덱 이름이 문서의 제목 대신 파일 이름이 되고, 슬라이드가 갈리지 않아 본문이 한 장에 쌓이다 `maximumPoints`(5)에서 `# (계속)` 으로 넘어간다. 고치면 같은 문서를 `#` 로 쓴 것과 같은 덱이 나온다.
- 실패 재현(정찰이 실제로 실행해 확인. 임시 테스트는 지웠음):
  `docs.Read("월간 보고서.md", []byte("# 분기 요약\n\n매출이 늘었습니다.\n"))` 의 `Source`

  ```
  # 분기 요약
  @cover
  > 월간 보고서.md

  # 분기 요약
  - 매출이 늘었습니다.
  !source 월간 보고서.md | 분기 요약
  ```

  같은 문서를 setext 로 쓴 `[]byte("분기 요약\n=========\n\n매출이 늘었습니다.\n")` 의 `Source` (지금)

  ```
  # 월간 보고서
  @cover

  # 월간 보고서
  - 분기 요약
  - =========
  - 매출이 늘었습니다.
  !source 월간 보고서.md
  ```

  더 긴 문서에서는 제목이 둘 다 불릿으로 남아 한 장에 쌓이고 5개를 넘으면 `# (계속)` 장이 생긴다
  (정찰이 4절 문서로 확인). `-----` 밑줄은 `escapeLine` 때문에 `- \---------` 으로 나온다.
- 수용 기준:
  1) `docs.Read` 로 읽은 `분기 요약\n=========\n\n매출이 늘었습니다.\n` 의 `Source` 가 같은 문서를 `# 분기 요약` 로 쓴 것의 `Source` 와 **문자열로 같다**(위 첫 블록). 테스트는 두 입력을 각각 `Read` 해서 `!=` 로 비교할 것 — 기대 문자열을 손으로 베껴 두 번 적지 말 것.
  2) `=====` 밑줄 줄이 불릿(`- =====`)으로 남지 않고, 문서 중간의 setext 제목도 새 슬라이드를 시작한다(2절 문서에서 `# …` 본문 슬라이드가 2장, `!source … | <제목>` 인용이 각 장의 제목을 가리킨다).
  3) 회귀 없음을 테스트가 증명한다: (a) 앞에 문장이 없는 홀로 있는 `===`(빈 줄 뒤 또는 파일 첫 줄)은 고치기 전과 똑같이 불릿으로 남는다, (b) 문장과 `===` 사이에 빈 줄이 있으면 제목이 되지 않는다, (c) 펜스 코드 블록 안의 `===` 는 여전히 코드로 세어지고(경고 줄 유지) 제목이 되지 않는다.
- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `server/internal/docs/prose.go:readMarkdown` — 지금 `handle := func(line string)` 클로저와 그 바깥 루프 `for _, raw := range strings.Split(...)` 를 한 줄 앞보기가 되게 바꾼다. 권장 형태:
    - `lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")` 를 변수로 잡고 `for index, raw := range lines` 로 돌리며 `skip bool` 을 둔다(밑줄 줄을 한 번 건너뛰기 위한 것).
    - `handle` 의 시그니처를 `func(line, next string) bool` 로 바꾸고, **`default:` 분기에서만** `flush()` 뒤에 `if underlinesHeading(next) { writer.slide(line); return true }` 를 넣고 그 밖에는 지금처럼 `writer.point(line)`. 나머지 분기(``, `#`, `|`, `isListLine`)는 한 글자도 바꾸지 말 것.
    - 펜스 처리(`fence`/`held`/`blocks`/`skipped`)는 지금 구조 그대로 두고, `handle` 호출 자리에만 다음 줄(`lines[index+1]` 의 `TrimSpace`, 없으면 `""`)을 넘긴다. 닫히지 않은 펜스의 재생 루프(prose.go:279)도 인덱스 루프로 바꿔 `held[i+1]` 을 넘기고 같은 `skip` 규칙을 쓸 것 — 재생된 줄에서도 setext 가 같게 읽히도록.
  - `server/internal/docs/prose.go` — 새 헬퍼 하나. `fenceOf`/`closesFence` 옆에 두는 것이 자연스럽다:
    `// underlinesHeading … func underlinesHeading(line string) bool { return line != "" && strings.Trim(line, "=") == "" }`
    (`=` 는 줄 첫머리에서 다른 뜻이 없으므로 1개 이상이면 밑줄로 본다 — CommonMark 도 그렇다. 주변 주석 밀도에 맞춰 "왜 앞 줄을 제목으로 되돌리지 않고 앞보기로 하는가" 를 산문으로 적을 것.)
  - `server/internal/docs/markdownsetext_test.go` (새 파일) — `markdownfence_test.go` 의 관례(영어 문장형 테스트 이름, 실패 시 실제 `Source` 를 찍기)를 따를 것.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd server && go test ./internal/docs` (약 3~6초)
  - `cd server && go test -race ./...` (25개 패키지)
  - `cd server && go vet ./...` · `gofmt -l internal/docs` (출력 없어야 함) · `git diff --check`
  - 웹·API·문법 문서 변경이 없으므로 `make test` 의 웹 단계는 건너뛸 수 있다(최근 회차들의 관례).
- 위험과 피할 것:
  - **`---` 밑줄(setext h2)은 이번 범위 밖.** `-` 는 `listMarkers` 의 marker 이고 thematic break 이며 YAML front matter 의 구분선이기도 하다. 정찰이 확인한 것: `---\ntitle: 보고서\n---` 는 지금 `- \---` / `- title: 보고서` / `- \---` 세 불릿으로 나오는데, `-` 밑줄까지 접으면 `title: 보고서` 가 슬라이드 제목이 된다. CommonMark 는 실제로 그렇게 읽지만 사용자에게는 이득이 아니다. 커밋 메시지에 범위 밖이라고 남기고 다음 회차로 넘길 것.
  - `escapeLine`·`writer.point`·`writer.slide`·`cover()`·`deckWriter` 는 건드리지 말 것 — writer.go 는 docx·pdf·markdown 세 리더가 함께 쓴다.
  - 숫자 파서(`amountOf`/`allNumeric`/`bareFigure`)와 `deck` 쪽 파서, `tables.go`, `isRule`, `withoutListMarker` 는 이 과제와 무관하다. 2026-09-09 에 분류기를 넓혀 두 번 반려된 자리다.
  - 앞 줄을 이미 쓴 뒤에 되돌리는(retract) 방식은 쓰지 말 것 — `writer.point` 는 `maximumPoints` 에서 `(계속)` 슬라이드를 만들 수 있어 되돌릴 수 없다. 그래서 앞보기다.
  - 기존 마크다운 테스트(`docs_test.go`·`markdownfence_test.go`·`wordbreaks_test.go`·`listmarker_test.go`·`told_test.go`)가 한 글자도 안 바뀌는지 확인할 것. 바뀌면 그 입력에 `===` 가 있었다는 뜻이니 먼저 왜 바뀌는지 적을 것.
  - 보호 경로(auth·httpapi·migrations·workflows·release 스크립트)는 건드리지 않는다. 버전(`VERSION` 현재 1.69.50)·릴리즈 노트·배포 매니페스트도 손대지 말 것.
- 차선 후보: tables.go:186 이 `writer.go` 의 `continued()` 대신 `' (계속)'` 을 직접 붙이는 것을 `continued()` 로 통일(가치 2 / 위험 1 / 작업량 S) — 시트 이름이 이미 '(계속)' 으로 끝나는 실제 xlsx 가 있는지는 미확인이라 값이 낮다. 그 다음은 deck 의 네 숫자 파서 계약을 한 표 테스트로 묶기(파서는 손대지 않음).

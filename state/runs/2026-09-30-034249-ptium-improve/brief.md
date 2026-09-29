- 과제: 마크다운/텍스트 업로드에서 `#` 제목은 `#` 뒤에 공백이 있을 때만 — `#출시` 해시태그와 `#1 우선순위`가 슬라이드를 열지 않게 (가치 3 / 위험 2 / 작업량 S)

- 왜: `readMarkdown`(server/internal/docs/prose.go:222)이 `case strings.HasPrefix(line, "#")` 만으로 제목을 판정해, 문단 한가운데의 해시태그 줄과 `#1` 로 시작하는 문장이 슬라이드를 연다. 이 리더는 `.md`·`.markdown`·`.txt` 를 전부 받으므로(docs.go:81) 해시태그를 쓰는 평범한 메모가 그대로 걸린다. 특히 `#1 우선순위는 출시입니다.` 는 **문장 자체가 본문에서 사라진다** — 제목이 되어 버려 요점이 하나도 없는 슬라이드가 남는다. CommonMark 는 ATX 제목에 `#` 뒤 공백(또는 줄 끝)을 요구하고 `#` 6개까지만 인정한다. 그 규칙만 맞추면 해시태그 줄은 요점으로 남고(`escapeLine`(docs.go:125)이 선두 `#` 를 이미 `\#` 로 지켜 준다) 문서가 쪼개지지 않는다.

- 실패 재현 (이번 정찰이 임시 테스트를 넣고 `go test ./internal/docs` 로 **실제로 실행해 확인**했고, 임시 파일은 지웠다):
  - 입력 `"# 분기 요약\n\n매출이 늘었습니다.\n#출시 #마케팅\n비용은 줄었습니다.\n"`
    → 슬라이드 3장. 셋째가 `# 출시 #마케팅` / `- 비용은 줄었습니다.` / `!source 월간 보고서.md | 출시 #마케팅`.
    기대: 슬라이드 2장, 둘째 슬라이드에 `- \#출시 #마케팅` 과 `- 비용은 줄었습니다.` 두 요점.
  - 입력 `"# 분기 요약\n\n매출이 늘었습니다.\n#1 우선순위는 출시입니다.\n"`
    → 둘째 슬라이드가 `# 1 우선순위는 출시입니다.` 에 요점 0개(`!source … | 1 우선순위는 출시입니다.`). 문장이 본문에서 사라진다.
  - 입력 `"# 분기 요약\n\n####### 일곱개\n매출이 늘었습니다.\n"` → `# 일곱개` 슬라이드. CommonMark 는 `#` 7개를 제목으로 보지 않는다.
  - (범위 밖, 차선 후보로 남김) `"## 분기 요약 ##\n\n매출이 늘었습니다.\n"` → 제목이 `분기 요약 ##` 이고 **덱 이름까지** `분기 요약 ##` 이 된다.

- 수용 기준:
  1) `#` 바로 뒤에 공백이 없는 줄(`#출시 #마케팅`, `#1 우선순위는 출시입니다.`)은 슬라이드를 열지 않고 그 자리의 요점이 된다 — 위 첫째 입력이 슬라이드 2장이 되고 `#1 …` 문장이 `Source` 안에 요점으로 남는다.
  2) `#` 이 7개 이상인 줄도 제목이 아니다(`####### 일곱개` → 요점).
  3) 지금 제목인 것은 전부 그대로 제목이다: `# 제목`, `## 제목`, `###### 제목`, 그리고 `#` 만 있는 줄과 `#` 뒤 공백만 있는 줄(둘 다 CommonMark 의 빈 제목 — 현재 동작은 `writer.slide("")` 라 `flush` 가 덱 제목으로 대체한다. 이 두 경우는 **고치기 전 출력을 그대로 테스트에 박아** 한 글자도 안 바뀌는 것을 증명할 것).
  4) 새 테스트가 고치기 전 코드에서 실패하고(위 재현 문자열로) 고친 뒤 통과한다. 기존 `docs_test.go`·`markdownfence_test.go`·`markdownsetext_test.go`·`listmarker_test.go`·`wordbreaks_test.go`·`told_test.go` 는 한 글자도 바뀌지 않는다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `server/internal/docs/prose.go:222` — `case strings.HasPrefix(line, "#")` 를 새 헬퍼로 바꾼다. 권장: `atxHeading(line string) (string, bool)` 를 `underlinesHeading`(prose.go:419) 옆에 두고, 선두 `#` 를 세어 1~6개이고 그 다음이 줄 끝이거나 공백/탭일 때만 `(strings.TrimSpace(line[n:]), true)` 를 돌린다. `handle` 의 분기는 `case`→`if` 로 바꾸거나 `switch` 앞에서 `if text, ok := atxHeading(line); ok { flush(); writer.slide(text); return false }` 로 꺼내면 된다(기존 `flush()` + `writer.slide(...)` 두 줄의 순서·인자는 그대로).
  - 제목이 아니게 된 줄은 `switch` 의 `default:` 로 떨어져야 한다 — 그래야 setext 앞보기(`underlinesHeading(next)`)와 `writer.point` 를 지금 그대로 탄다. `isListLine` 보다 **뒤**로 가지 않게 순서를 지킬 것(`#` 는 listmarker 가 아니므로 실제 충돌은 없지만, 분기 순서를 바꾸면 펜스 재생 루프까지 같이 움직인다).
  - 새 테스트 파일 `server/internal/docs/markdownhash_test.go` — 위 재현 입력 4개 + 기준 3)의 "한 글자도 안 바뀜" 고정 문자열. 이 저장소 관례대로 테스트 이름은 영어 문장형("A hashtag does not start a slide"), 주석은 왜 그렇게 읽는지를 길게 산문으로.

- 검증 명령 (전부 이 저장소에서 실제로 도는 것. 정찰이 baseline `go test ./internal/docs` = ok 2.741s 를 확인했다):
  - `cd server && go test -count=1 ./internal/docs`
  - `cd server && go test -race ./...` (25개 패키지)
  - `cd server && go vet ./...`
  - `cd server && gofmt -l internal/docs` (출력 없어야 함)
  - `git diff --check`
  - 웹·API·문법 문서 변경이 없으므로 `make test` 의 웹 단계는 건너뛰어도 된다(최근 회차 관례).

- 위험과 피할 것:
  - **`server/internal/deck/source.go:261` 은 건드리지 말 것.** 거기도 `strings.HasPrefix(trimmed, "#")` 이지만 그것은 ptium 의 **덱 소스 DSL** 이지 CommonMark 가 아니다. 모델·사용자가 쓴 덱 소스와 `golden` 회귀가 전부 그 계약 위에 있다. 이번 과제는 **업로드 리더(docs)만** 이다.
  - `writer.go`(deckWriter — docx·pdf·markdown 셋이 같이 쓴다), `tables.go`, 숫자 파서(`amountOf`·`allNumeric`·`bareFigure`·deck 의 `parseNumber`)는 한 글자도 건드리지 말 것. 과거에 통합 시도가 두 번 반려됐다.
  - `escapeLine`(docs.go:125)도 건드리지 말 것 — 선두 `#` 를 `\#` 로 지키는 것은 이미 맞게 돌고, 기대 문자열의 `- \#출시` 는 그 결과다.
  - 펜스(`fenceOf`/`closesFence`)와 setext(`underlinesHeading`/`lineAfter`/`skip`)는 그대로 둘 것. 다만 미닫힘 펜스 재생 루프(prose.go:300)도 같은 `handle` 을 부르므로 새 규칙이 거기에도 자동으로 적용되는 것이 맞다 — 테스트로 한 번 확인만.
  - 보호 경로(auth·httpapi·db/migrations·workflows)와 VERSION·릴리즈 노트·배포 매니페스트는 손대지 않는다.
  - 문서: `docs/` 의 한국어 가이드가 "업로드한 마크다운의 `#` 는 제목" 이라고 적고 있는지 **미확인**. 구현자가 `grep` 한 번 해 보고, 공백 규칙을 적을 자리가 있으면 한 줄만 더할 것(없으면 그냥 두기).

- 차선 후보: **ATX 제목의 닫는 `#` 시퀀스를 제목에서 떼기** — `## 분기 요약 ##` 이 지금 `분기 요약 ##` 이라는 제목이 되고 그것이 그대로 **덱 이름**이 된다(위 재현으로 확인). 같은 `atxHeading` 헬퍼 안에서 끝의 `#` 런을 앞에 공백이 있을 때만 떼면 끝나므로 1순위가 성립하지 않으면 이것을 고를 것. 3순위: `tables.go` 의 마크다운 표에서 이스케이프된 `\|` 가 셀을 가르는 것(위험 3/M, 계약 확인 필요).

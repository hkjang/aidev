- 과제: 여러 줄로 쓴 생각이 문서 차례(outline)에서는 한 단락으로 합쳐진다 — 덱 경로와 같은 계약으로 쓰기 (가치 3 / 위험 1 / 작업량 S)

- 왜: 같은 `Storyline` 을 쓰는 두 작성기 중 `WriteSource`(`internal/presentation/source.go:103-118`)는 여러 줄 생각을 `splitLines` 로 나눠 "한 단락으로 합치지 않는다" 는 계약을 지키고 그 계약을 시험으로 못 박았는데(`source_test.go:63-69` `TestAMultiLineThoughtKeepsItsLines`, 주석 "A thought written over several lines is several lines. Joining them into a paragraph would be rewriting it."), `WriteOutline`(`internal/presentation/outline.go:74`)은 `point.Text` 를 `- ` 한 항목에 **날것으로** 써서 둘째 줄이 0열에 떨어집니다. 이 패키지가 "사람의 문장을 바꾸지 않는다" 고 선언한 것이 바로 이 지점이고, `outline_test.go` 에는 여러 줄 본문 시험이 **한 건도 없습니다**(`\n` 검색 결과 단언 문구뿐).

- 확인된 재현 (정찰이 실제로 돌린 것): 임시 시험 파일을 `internal/presentation` 에 두고 손으로 만든 대역 없이 실제 `Compile(thoughts, links, Options{})` → 실제 `Thought`(기존 `thought(2, "첫 줄\n둘째 줄", 0, 300)` 헬퍼) 로 돌려 두 작성기의 출력을 같은 `Storyline` 에서 나란히 찍었습니다(`go test ./internal/presentation -run … -v`, 0.003s, 확인 뒤 파일 삭제 — 작업 트리 `git status` 깨끗):
  - `POINT depth=0 text="첫 줄\n둘째 줄"` — 줄바꿈이 `Compile` 을 지나 `Point.Text` 까지 그대로 옵니다(`body()` 가 `storyline.go:610` 에서 `TrimSpace(t.Content)` 만 함).
  - 덱: `- 첫 줄` / `  - 둘째 줄` ← 계약대로.
  - 차례: `- 첫 줄` / `둘째 줄` ← **0열**. 두 줄이 한 항목의 한 단락으로 붙습니다.
  - 둘째 줄이 `---` 인 생각(`"결론\n---"`)은 차례에 `- 결론` / `---` 로 나옵니다. 0열의 `---` 는 CommonMark 에서 블록 수준 마커(수평선 또는 바로 위 단락의 setext 머리글)이므로 사람이 쓰지 않은 구조가 생깁니다 — 다만 정찰은 **출력 문자열까지만 확인했고 실제 렌더러에 넣어 보지는 않았습니다(미확인)**. 구현자는 수용 기준 2를 "0열 마커가 출력에 없다" 로 단언하면 되고 렌더러를 가져올 필요는 없습니다.

- 수용 기준:
  1) `WriteOutline` 이 여러 줄짜리 `Point.Text` 를 쓸 때 둘째 줄 이후가 0열에 나오지 않는다(덱 경로와 같이 한 단 들여써서 같은 항목에 속함이 드러난다).
  2) 둘째 줄이 `---`·`#`·`> ` 로 시작해도 문서에 수평선·머리글·인용이 새로 생기지 않는다.
  3) 시험이 **같은 입력**(여러 줄 본문 하나를 담은 생각)으로 `WriteSource` 와 `WriteOutline` 두 경로를 함께 단언해, 두 작성기가 같은 값을 같은 계약으로 읽는 것을 보인다(운영자 규칙 3번). 기존 `outline_test.go` 의 깊이/머리글 단언(`:128`, `:131`)과 `source_test.go` 의 여러 줄 단언은 그대로 통과한다.

- 건드릴 파일:
  - `internal/presentation/outline.go:writeOutlineSection` — 점 루프에서 `splitLines(text)`(같은 패키지 `source.go:170` 에 이미 있음, 빈 줄은 거기서 이미 떨어진다)를 써서 첫 줄은 `indent + "- "`, 둘째 줄부터는 `indent + "  - "` 로 쓰기. **덱 경로(`source.go:112-118`)와 같은 모양을 권합니다** — 두 작성기가 같은 입력을 같은 shape 으로 읽는 것이 이번 과제의 요지이고, 2칸 들여쓴 이어쓰기(`indent + "  "`)만으로는 마커는 막히지만 Markdown 에서는 여전히 한 단락으로 합쳐집니다. 들여쓰기 폭 계산 `strings.Repeat("  ", max(point.Depth,0))` 는 그대로 둘 것(`Depth` 는 0 또는 1 뿐이므로 이어쓰기 줄의 선행 공백은 최대 4칸이다 — 시험에서 나오는 줄을 글자 그대로 단언할 것).
    - `heading`·`text == heading` 비교와 "점이 하나라도 있으면 끝에 빈 줄" 규칙은 건드리지 말 것(기존 시험 `outline_test.go:101`·`:128`·`:131` 이 그것을 본다).
  - `internal/presentation/outline_test.go` — 새 시험 1~2개(여러 줄 생각 / 둘째 줄이 `---` 인 생각). 가능하면 실제 `Compile` 을 지나 `WriteOutline` 과 `WriteSource` 를 같은 `Storyline` 에서 함께 단언(기존 시험이 쓰는 `thought(id, content, x, y)` 헬퍼 재사용 — `source_test.go:66` 이 `thought(2, "첫 줄\n둘째 줄", 0, 300)` 꼴로 씀).
  - 프로덕션 파일 1개. 그 이상 늘리지 말 것.

- 검증 명령:
  - `go test ./internal/presentation -count=1 -v` (DB 불필요, 0.1초 미만. `-v` 로 새 시험이 RUN 되는지 확인 — SKIP 을 PASS 로 읽지 말 것)
  - `gofmt -l internal/presentation` (무출력) · `go vet ./...` · `go build ./cmd/...` (산출물 `umm` 은 커밋 전 삭제)
  - 고친 뒤 `git show HEAD:internal/presentation/outline.go` 로 **프로덕션 파일만** 되돌려 같은 실패가 다시 나는 것 확인.
  - 여유가 있으면 격리 DB 로 `POSTGRES_DSN=… go test -p 1 ./... -count=1` (DSN 없으면 HTTP 통합은 SKIP 되니 "돌았다" 고 쓰지 말 것).

- 위험과 피할 것:
  - `point.Depth` 는 이미 상한이 있다 — `storyline.go:518` 이 `Depth: min(depth, 1)`. 그러니 "깊이가 무한히 들여써진다" 는 보류 아이디어는 이 트리에서 성립하지 않는다(그 아이디어는 `rejected`). 들여쓰기 상한을 새로 넣지 말 것.
  - `usableHeading`(`headings.go:157`)·`usableSections`(`sections.go`)는 건드리지 말 것. 이 베이스(main@6aaf940)에는 2026-10-04 의 2패스 수정도, 2026-10-02 의 중복 부 제목 거절도 **없고**(직접 읽어 확인), 그 두 회차는 verify-failed 로 끝났다. 같은 자리를 다시 고르는 것이 이번 과제가 아니다.
  - `escapeLine` 을 outline 으로 끌어오지 말 것 — 덱 소스 마커 이스케이프는 Ptium 언어의 계약이고 Markdown 문서의 계약이 아니다. 두 작성기를 **통합하지 말 것**(운영자 규칙 3번 후반).
  - `slide.Lead` 처리(`outline.go:63`)는 산문이라 이번 범위 밖. 점(bullet)만 고칠 것.
  - 보호 경로(`internal/auth/`, `migrations/`, `.github/workflows/`, Dockerfile, `web/`)는 건드리지 않는다 — 이번 과제에 필요 없다.

- 차선 후보: 캔버스 공간 이름 대체값이 한국어 UI 에 영어 'My Space' 로 뜬다 — `web/src/pages/CanvasPage.tsx` 의 `spaces.find(...)?.name || 'My Space'`. 증명이 Playwright 라 S 치고 무겁고, 같은 자리에서 공백 한 칸 이름이 truthy 로 통과하는 문제도 함께 봐야 한다.

- 과제: 마크다운 업로드에서 펜스 코드 블록(``` / ~~~)이 덱을 쪼개지 않게 하기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `readMarkdown`(server/internal/docs/prose.go:205)은 줄 하나씩만 보고 코드 펜스라는 것을 모른다. 그래서 블록 **안**의 `# 1단계` 가 새 슬라이드가 되고 ``` 줄 자체가 불릿으로 남아, 런북 한 장이 엉뚱한 슬라이드 여러 장으로 흩어지고 블록 뒤의 문장이 마지막 코드 줄의 슬라이드에 붙는다. 고치면 .md 업로드가 사람이 문서에서 본 구조 그대로 나오고, 덱이 담을 수 없는 코드는 조용히 사라지는 대신 경고로 말해진다(패키지 주석 docs.go:11 "읽을 수 없는 것은 추측하지 말고 말한다" 그대로).

- 실패 재현 (이번 정찰에서 임시 테스트로 실제 실행해 확인, 그 파일은 지웠다):
  입력 `런북.md`
  ```
  # 배포 절차
  (빈 줄)
  배포는 아래 순서로 합니다.
  (빈 줄)
  ```bash
  # 1단계: 이미지 빌드
  make build
  # 2단계: 배포
  kubectl apply -f deploy/kubernetes.yaml
  ```
  (빈 줄)
  끝입니다.
  ```
  지금 나오는 `Document.Source` (경고는 빈 배열):
  ```
  # 배포 절차 / @cover / > 런북.md
  # 배포 절차 / - 배포는 아래 순서로 합니다. / - ```bash / !source 런북.md | 배포 절차
  # 1단계: 이미지 빌드 / - make build / !source 런북.md | 1단계: 이미지 빌드
  # 2단계: 배포 / - kubectl apply -f deploy/kubernetes.yaml / - ``` / - 끝입니다. / !source 런북.md | 2단계: 배포
  ```
  즉 본문 슬라이드가 1장이어야 할 것이 3장이 되고, "끝입니다." 가 "2단계: 배포" 슬라이드에 얹힌다.

- 수용 기준:
  1) 위 입력이 본문 슬라이드 **한 장**("배포 절차")만 내고, 그 슬라이드의 점은 "배포는 아래 순서로 합니다." 와 "끝입니다." 둘뿐이다. 출력 어디에도 "1단계"·"2단계" 로 시작하는 `# ` 줄과 ``` 를 담은 `- ` 줄이 없다.
  2) `Document.Warnings` 에 가져오지 않은 코드 블록의 **개수**를 말하는 줄이 한 줄 생긴다(docx 그림 경고 `prose.go:94` 와 같은 자리·같은 어투: `document, err := writer.document()` 뒤에 붙이고, 문구는 "코드 블록 %d개는 가져오지 않았습니다" + 무엇을 하라는 짧은 한 마디). 코드 블록이 없는 파일에는 이 경고가 붙지 않는다.
  3) **닫히지 않은 펜스는 아무것도 삼키지 않는다.** ``` 한 줄만 있고 끝까지 닫히지 않는 문서에서 그 뒤의 `# 둘째 장` 슬라이드와 그 문장이 지금과 똑같이 나오고, 코드 블록 경고도 붙지 않는다. (테스트가 증명할 것: 펜스 처리를 넣은 뒤에도 이 문서의 `Document.Source` 가 고치기 전과 한 글자도 다르지 않다.)
  4) `~~~` 펜스도 같게 다루고, 여는 줄의 정보 문자열(```bash)은 여는 줄로 인정하되 닫는 줄로는 인정하지 않는다(닫는 줄은 같은 문자만으로 이루어진 줄).
  5) 회귀: `go test ./internal/docs` 의 기존 마크다운 테스트(`docs_test.go`, `wordbreaks_test.go`, `listmarker_test.go`, `told_test.go`)가 한 글자도 바뀌지 않고 통과한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `server/internal/docs/prose.go:readMarkdown`(205행) — 지금의 `for range strings.Split(...)` 본문 switch 를 `handle := func(line string)` 클로저로 **그대로** 빼내고(안의 `flush()`·`writer.slide`·`writer.point`·`table` 처리는 한 글자도 바꾸지 말 것), 바깥 루프에 펜스 상태를 둔다:
    - 펜스 밖에서 여는 줄을 만나면 `flush()` 한 뒤 `fence` 문자와 길이를 기억하고, **여는 줄 자체를 포함해** 원본 줄을 버퍼에 쌓기 시작한다.
    - 펜스 안에서 닫는 줄을 만나면 블록 수 +1, 버퍼는 버리고 펜스를 닫는다.
    - 루프가 끝났는데 펜스가 열린 채면 — 문장 속 ``` 한 개짜리 오타다 — 블록 수를 세지 말고 버퍼의 줄들을 `handle()` 로 **재생**한다. 이것이 수용 기준 3을 만든다.
    - 마지막에 기존 `flush()`.
  - `server/internal/docs/prose.go:readMarkdown` 반환 직전 — `writer.document()` 의 err 를 먼저 돌려보낸 뒤(그림 경고와 같은 순서), 블록 수가 0보다 크면 경고 한 줄 append.
  - 새 테스트 파일 `server/internal/docs/markdownfence_test.go` — 위 재현 입력으로 red 를 먼저 확인하고, 닫히지 않은 펜스·`~~~`·```bash 정보 문자열·코드 블록만 있는 파일(=기존 `"이 문서에서 슬라이드로 만들 내용을 찾지 못했습니다"` 에러가 그대로 나오고 패닉하지 않는지) 각각 한 케이스. 파일 이름은 이웃 관례(`wordbreaks_test.go`·`sheetparts_test.go`)를 따랐고, 테스트 이름은 영어 문장형("A fenced code block does not start a slide").

- 검증 명령:
  - `cd server && go test ./internal/docs` (약 3초)
  - `cd server && go test -race ./...` (25개 패키지)
  - `cd server && go vet ./...` 와 `gofmt -l internal/docs` (출력 없어야 함)
  - `git diff --check`
  - 웹·API·문법 문서를 건드리지 않으므로 `make test` 의 웹 단계는 건너뛴다(최근 회차들의 관례).

- 위험과 피할 것:
  - **숫자 파서에 손대지 말 것.** `amountOf`·`allNumeric`·`bareFigure`(tables.go)와 `deck` 의 `parseNumber`·`parseBareNumber`·`chartFields` 는 계약이 다른 별개의 파서다. 2026-09-09 에 두 번 반려된 자리이므로 이번 과제와 한 줄도 섞지 말 것.
  - **`writer.point`·`writer.slide`·`flush`·`splitTables`(writer.go)를 바꾸지 말 것.** 이 과제는 `readMarkdown` 이 어떤 줄을 `handle` 에 넘기느냐만 바꾼다. docx·pdf 경로는 같은 `deckWriter` 를 쓰므로 writer.go 를 건드리면 세 리더가 한꺼번에 움직인다.
  - 들여쓰기 4칸 코드 블록(펜스 없는 것)은 **범위 밖**이다. 지금도 그냥 점이 되고, 넣으려면 목록 이어짐과 구별해야 해서 한 회차에 안 끝난다. 과제서에 범위 밖이라고 남기고 끝낼 것.
  - `line := strings.TrimSpace(raw)` 가 이미 최대 3칸 들여쓴 펜스를 흡수한다(마크다운 규격보다 관대하지만 해롭지 않다). 재생 버퍼에는 **원본 raw** 가 아니라 지금 루프가 쓰는 것과 같은 형태를 넣어 재생 결과가 고치기 전과 같게 할 것 — 수용 기준 3 의 "한 글자도 다르지 않다" 가 여기를 문다.
  - 경고 문구는 한국어다. `korean` 패키지의 조사 검사는 덱 렌더링 쪽이라 여기에 걸리지 않지만, 옆의 그림 경고와 같은 "%d개는 …" 꼴을 그대로 쓰면 안전하다.
  - 보호 경로(auth·httpapi·db/migrations·scripts/release.sh)와는 무관하다. 버전·릴리즈 노트·배포 매니페스트는 손대지 말 것.

- 차선 후보: `tables.go:186` 이 `writer.go:132` 의 `continued()` 대신 `" (계속)"` 을 직접 붙이는 것을 `continued()` 로 바꾸기 (가치 2 / 위험 1 / 작업량 S). 시트 이름이 이미 "(계속)" 으로 끝나는 실제 xlsx 가 있는지는 여전히 **미확인**이라 값이 낮다 — 1순위가 성립하지 않을 때만.

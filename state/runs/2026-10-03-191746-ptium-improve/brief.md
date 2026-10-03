# 과제서 2026-10-03 (run 2026-10-03-191746-ptium-improve, base main@e10d356 = Release 1.69.54)

- 과제: 마크다운/텍스트 업로드에서 수평선(`---` / `***` / `___` / `- - -`)이 요점으로 남는 것 — 구분선은 글자가 아니라 구두점이다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `readMarkdown`(server/internal/docs/prose.go:205)의 `handle` 에 수평선 분기가 없어 구분선이 `default:` 로 떨어져 `writer.point` 가 되고, `escapeLine`(docs.go:125)이 선두 `-`·`*` 를 지키느라 **사용자 슬라이드에 `\---`·`\***` 라는 백슬래시까지 보인다**(아래 실행 확인). 구분선은 문서를 나누는 기호라 슬라이드에 옮길 내용이 아예 없으므로, 떼어내면 손실 없이 요점 자리가 깨끗해지고 YAML front matter 로 시작하는 문서의 첫 장도 `\---` 두 줄을 잃는다.

- 실행 확인(2026-10-03, 임시 테스트로 `Read("월간 보고서.md", …)` 의 `Source` 를 찍어 본 결과. 임시 파일은 지웠다):
  - `"# 월간 보고서\n매출이 늘었습니다.\n\n---\n\n비용은 줄었습니다.\n"` → 본문 슬라이드가
    `- 매출이 늘었습니다.` / `- \---` / `- 비용은 줄었습니다.` 세 요점.
  - `"---\ntitle: 보고서\n---\n\n# 분기 요약\n문장입니다.\n"` → 파일명 슬라이드에
    `- \---` / `- title: 보고서` / `- \---` 세 줄이 쌓인다(그 뒤에 `# 분기 요약` 장이 따로 생긴다).
  - `***` → `- \***`,  `___` → `- ___`(이건 escapeLine 이 안 지켜서 백슬래시는 없다),
    `- - -` → `- \- -`,  `* * *` → `- \* *`.
  - **지켜야 하는 현 동작**: `--` → `- \--`(두 개는 수평선이 아니다), `-5% 감소` → `- \-5% 감소`,
    `# C# 도입`·`# 제목#`·setext `===` 제목·펜스 경고는 그대로.

- 수용 기준:
  1) 위 재현 두 입력의 `Source` 에서 `\---` 요점이 사라진다 — 첫 입력은 `- 매출이 늘었습니다.` / `- 비용은 줄었습니다.` 두 요점만 남고(슬라이드 수는 **늘지 않는다**: 수평선은 새 장을 열지 않는다), 두 번째 입력의 파일명 슬라이드는 `- title: 보고서` 한 줄만 남는다. front matter 자체를 건너뛰는 것은 **이번 과제가 아니다**(별도 보류 아이디어, 아래 차선 참고).
  2) `---`·`***`·`___`·`- - -`·`* * *` 가 모두 떨어지고, `--`·`-5% 감소`·`***중요***`·`*** 중요 ***`·`# 제목` 은 한 글자도 바뀌지 않는다(테스트가 두 방향을 모두 박는다).
  3) 수평선은 **경고를 만들지 않는다** — 그림(prose.go:94)·코드 블록(prose.go:327)과 달리 가져오지 못한 내용이 없다. "구분선 n개를 뺐습니다" 같은 줄을 붙이지 말 것. 테스트가 `document.Warnings` 가 비어 있음을 단언한다.
  4) 닫히지 않은 펜스의 재생 루프(prose.go:302~308)도 같은 `handle` 을 부르므로 거기서도 수평선이 떨어지고, 펜스 **안**의 `---` 은 지금처럼 코드 블록으로 세어진다(경고 유지).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개 + 문서 1줄):
  - `server/internal/docs/prose.go`
    - 새 헬퍼 `isThematicBreak(line string) bool` — `isRule`(prose.go:483) 옆에 두는 것이 자리다. CommonMark 의 thematic break: 같은 문자(`-`·`_`·`*`) 3개 이상 + 사이에 공백/탭만, 다른 글자가 하나라도 있으면 false. 호출부가 이미 `strings.TrimSpace` 한 줄을 넘기므로(prose.go:266) 선두 공백은 신경 쓸 필요가 없다. 권장 형태:
      ```go
      const markdownBreaks = "-_*"
      // 첫 글자를 mark 로 잡고, mark·공백·탭 외의 바이트가 나오면 false,
      // 끝까지 갔으면 mark 개수가 3 이상인지.
      ```
    - `handle`(prose.go:218)의 `switch` 에 `case isThematicBreak(line):` 를 **`case isListLine(line):` 앞, `case strings.HasPrefix(line, "|")` 뒤**에 넣는다. `isListLine` 앞이어야 하는 이유: `- - -` 은 `withoutListMarker` 가 선두 `- ` 를 떼어 `- -` 라는 요점으로 만들기 때문이다(실행 확인).
    - 그 case 의 몸은 `flush()` 뒤 `return false` — 아무것도 쓰지 않는다. `flush()` 를 부르는 이유는 수평선이 빈 줄(`case line == ""`)과 최소한 같은 세기의 블록 구분이고, 표 두 개가 수평선으로 갈린 문서에서 두 표가 한 표로 붙지 않게 하기 때문이다. 주석에 그 이유를 적을 것(이 패키지는 "왜 그렇게 읽는지" 를 산문으로 적는다 — prose.go:454~478, 474~478 이 바로 `-` 를 setext 로 안 읽는 이유를 적어 둔 표본이고, 이번 변경은 그 문단과 **모순되지 않는다**: 여기서는 `-` 를 제목으로 읽는 것이 아니라 버리는 것이다).
    - `atxHeading`·`underlinesHeading`·`fenceOf`·`closesFence`·`isRule`·`withoutListMarker`·`listMarkers` 는 **한 글자도 건드리지 말 것**.
  - `server/internal/docs/markdownrule_test.go` (신규) — 실제 `Read("월간 보고서.md", []byte(...))` → `Document.Source`/`Warnings` 로 검증하고, 필요하면 `deck.ParseSource` 로 슬라이드 수·제목까지 본다(기존 `markdownhash_test.go`·`markdownsetext_test.go` 가 그 본이다). 위 재현 입력 두 개 + 떨어지는 다섯 모양 + 지켜야 하는 네 모양 + 펜스 안/미닫힘 펜스 재생. **고치기 전에 먼저 넣어 red 를 확인**하고(위 재현 두 개와 다섯 모양이 red, 지켜야 하는 것들은 전후 green 인 계약 고정용), 고쳐 green.
  - `docs/USER_GUIDE.md:276` — `.md`·`.txt` 행에 한 문장("`---`·`***` 같은 구분선은 슬라이드에 남지 않습니다" 정도). 이 칸은 이미 세 규칙이 쌓여 매우 길다 — 길이를 더 늘리지 말고 짧게 한 절만 붙이고, 칸을 쪼개는 것은 별도 아이디어로 둘 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd server && go test -count=1 ./internal/docs` ← 1순위. 2026-10-02 실측 3.160s ok.
  - `cd server && go test -race ./...` (26개 패키지, `golden`·`korean` 포함)
  - `cd server && go vet ./...` / `cd server && gofmt -l internal/docs` (둘 다 출력 없어야 함)
  - `git diff --check`
  - 웹·API·OpenAPI 변경이 없으므로 `make test` 의 웹 단계는 건너뛴다(최근 회차 관례). `USER_GUIDE.md` 는 Go 테스트가 읽지 않는다.

- 위험과 피할 것:
  - `internal/deck/source.go`(덱 DSL 파서)를 건드리지 말 것 — docs 의 마크다운 리더와 **계약이 다르고** golden 회귀가 그 위에 있다. 덱 DSL 에서 `-` 는 요점 마커이고 `>` 는 표지 부제다. `escapeLine`(docs.go:125)의 선두 문자 보호도 그대로 둘 것 — 이번 고침은 "escape 를 줄이는" 것이 아니라 "그 줄을 아예 쓰지 않는" 것이다.
  - `writer.go` 의 `deckWriter` 는 docx·pdf·markdown 세 리더가 공유한다. 거기엔 손대지 말 것(point 는 `maximumPoints` 에서 `(계속)` 을 만들어 되돌릴 수 없다 — 그래서 판정은 `handle` 안에서 **쓰기 전에** 해야 한다).
  - 숫자 파서(`docs.amountOf`/`allNumeric` 과 deck 의 `parseNumber`/`parseBareNumber`/`chartFields`)는 서로 다른 계약이다 — 2026-09-09 에 두 번 반려된 자리이고 이번 과제와 무관하다. 근처도 가지 말 것.
  - 보호 경로(auth·httpapi 인증·db/migrations.go·scripts/release.sh·deploy/) 와 VERSION·릴리즈 노트·배포 매니페스트는 건드리지 않는다(`server/internal/config/stamped_test.go` 가 버전 어긋남을 잡는다).
  - 범위를 넓히지 말 것: YAML front matter 건너뛰기, setext h2(`---`) 를 제목으로 읽기, 인용 `>` 떼기는 **각각 별도 과제**다. 수평선만 버리는 이번 변경이 들어가야 front matter 방침을 정할 자리가 생긴다.
  - grep 으로 "문자열이 있다" 를 증거로 내지 말 것 — 실제 `Read()` 출력으로 red→green 을 보일 것.

- 차선 후보: 마크다운 인용 `> 문장` 의 `>` 가 사용자에게 보이는 글자로 남는 것(`- \> 인용된 문장입니다.`, 2026-10-02 실행 확인) — 가치 2 / 위험 2 / S. 여러 줄·중첩 인용을 어떻게 다룰지(한 요점으로 합칠지, 줄마다 요점으로 둘지) 먼저 한 줄로 정하고 들어갈 것. 1순위가 성립하지 않을 때만 고르고, 두 과제를 한 회차에 같이 넣지 말 것 — 같은 `handle` 의 같은 `switch` 를 건드린다.

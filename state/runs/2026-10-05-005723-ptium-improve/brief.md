- 과제: 마크다운 YAML front matter(`---` … `---`)를 슬라이드로 읽지 않기 (가치 3 / 위험 2 / 작업량 S)
- 왜: 지난 회차에 수평선이 요점에서 떨어진 뒤(9517fbc), front matter 가 있는 `.md` 를 올리면 파일명 슬라이드에 `- title: 보고서` 같은 메타데이터 한 줄만 남는다 — 실제 동작이 `server/internal/docs/markdownrule_test.go:35 TestTheRulesAroundFrontMatterAreNotPoints` 에 문자열로 박혀 있다. Jekyll·Hugo·Obsidian 이 붙이는 머리말은 슬라이드에 올릴 내용이 아니므로 버리면 사용자가 손으로 지워야 할 슬라이드가 사라진다.
- 수용 기준:
  1) `Read("월간 보고서.md", "---\ntitle: 보고서\n---\n\n# 분기 요약\n문장입니다.\n")` 의 `Source` 가 front matter 세 줄이 없는 같은 문서(`"# 분기 요약\n문장입니다.\n"`)의 `Source` 와 같다 — 문자열을 직접 박지 말고 두 `Read` 를 비교할 것(`markdownsetext_test.go` 의 선례).
  2) `Warnings` 는 비어 있다 — 잃은 내용이 없는 구두점이므로 수평선(9517fbc)과 같고 그림·코드 블록 경고와 다르다.
  3) 삼키지 않는 것을 테스트가 증명한다: 첫 줄이 `---` 이어도 (a) 닫는 `---` 이 없으면, (b) 머리말 첫 내용 줄이 `key: value` 모양이 아니면(`---\n\n# 제목\n문장\n---\n` 처럼 본문이 `---` 로 끝나는 문서) 한 글자도 바뀌지 않는다 — 고치기 전 `Read` 결과와 같다.
  4) 첫 줄이 아닌 자리의 `---` 은 여전히 수평선이다(기존 `TestEveryShapeOfHorizontalRuleDropsOut` 8개 모양 그대로 green).
- 건드릴 파일:
  - `server/internal/docs/prose.go:readMarkdown`(205) — `lines := strings.Split(...)`(272) 바로 뒤, `for index, raw := range lines` 루프 앞에서 `lines = lines[frontMatterLines(lines):]` 로 잘라낸다. 슬라이스를 잘라야 `lineAfter(lines, index)` 의 앞보기와 `skip` 플래그가 그대로 맞는다(인덱스 오프셋을 따로 들고 다니지 말 것).
  - `server/internal/docs/prose.go` — 새 헬퍼 `frontMatterLines(lines []string) int` 를 `isThematicBreak`(519~534) 바로 뒤, `isRule`(536) 앞에 둔다(파일은 543행에서 끝난다). 규칙: `lines[0]` 의 뒤쪽 공백·탭만 떼어 정확히 `---` 일 때만 시작(선두 공백은 front matter 가 아니다), 그 뒤 같은 모양의 `---` 을 닫는 줄로 찾고, 열고 닫는 사이의 첫 비어 있지 않은 줄이 YAML 매핑 키 모양(`:` 앞이 비어 있지 않고 공백·탭을 포함하지 않음)일 때만 "닫는 줄 다음" 인덱스를 돌려준다. 그 밖에는 0(= 아무것도 건너뛰지 않음).
  - `server/internal/docs/markdownrule_test.go:35` — `TestTheRulesAroundFrontMatterAreNotPoints` 가 고정한 기대값이 곧 이 과제의 현 동작이다. 이름과 주석을 새 계약으로 고치고, 기대값은 수용 기준 1 의 "두 `Read` 비교" 로 바꿀 것. 다른 테스트 파일은 한 글자도 건드리지 말 것.
  - 새 `server/internal/docs/markdownfrontmatter_test.go` — 재현 1건 + 삼키지 않는 두 경우 + 빈 머리말(`---\n---\n`) + 경고 없음.
  - `docs/USER_GUIDE.md:276` 의 `.md`·`.txt` 칸 — 한 문장. (이 칸은 이미 네 규칙·350자다. 절로 빼는 일은 별도 아이디어이니 이번에는 한 문장만 더할 것.)
- 검증 명령:
  - `cd server && go test -count=1 ./internal/docs` (실측 ~3s)
  - `cd server && go test -race ./...` (26개 패키지, ~25s), `cd server && go vet ./...`, `cd server && gofmt -l internal/docs`(출력 없어야 함)
  - 웹·API·OpenAPI 변경이 없으므로 `make test` 의 웹 단계는 건너뛴다(최근 회차 관례).
- 위험과 피할 것:
  - **문서를 삼키는 것이 이 과제의 유일한 진짜 위험이다.** `---` 로 시작하고 아래에 또 `---` 이 있는 평범한 문서(맨 위 구분선 + 본문 끝 구분선)를 front matter 로 읽으면 제목·본문이 전부 사라진다. 그래서 "첫 내용 줄이 YAML 키 모양" 가드를 계약에 넣었다 — 이걸 빼고 단순화하지 말 것.
  - `isThematicBreak`·`atxHeading`·`underlinesHeading`·`fenceOf`/`closesFence`·`isRule`·`withoutListMarker`·`escapeLine`·`writer.go`·`internal/deck/source.go`·숫자 파서(`amountOf`/`allNumeric`)는 건드리지 말 것. 특히 `internal/deck/source.go` 는 덱 DSL 로 계약이 다르다(golden 회귀가 그 위에 있다).
  - `title:` 을 덱 이름으로 쓰는 것은 **범위 밖**이다(그걸 넣으면 M 이 되고 `titleOf`·`@cover`·`!source` locator 까지 번진다). 커밋 메시지에 범위 밖이라고 적을 것.
  - 미닫힘 펜스 재생 루프(prose.go:313)는 `held` 를 돌리므로 front matter 와 무관하다 — front matter 잘라내기는 **원본 `lines` 에 한 번만** 적용하고 재생 루프에는 넣지 말 것.
  - 빈 파일명 슬라이드는 생기지 않는다(읽어서 확인): `newDeckWriter`(writer.go:40)는 `heading` 을 비워 두고 `title` 만 잡으므로, front matter 를 건너뛴 뒤 첫 줄이 `# 분기 요약` 이면 `atxHeading` 분기의 `flush()`(writer.go:149~152)가 heading·points·tables·notes 전부 비어 바로 돌아가고 아무것도 쓰지 않는다. 그 뒤 `cover()`(writer.go:52~75)가 heading 을 덱 이름으로 올려 `# 분기 요약 / @cover / > 월간 보고서.md` + `# 분기 요약 / - 문장입니다. / !source 월간 보고서.md | 분기 요약` 을 쓴다 — 이 모양은 `markdownhash_test.go` 가 이미 같은 입력으로 박아 둔 출력과 같다. 그래도 기대값은 문자열로 박지 말고 수용 기준 1 의 두 `Read` 비교로 둘 것.
  - `underlinesHeading` 의 주석(prose.go:485~489)은 `-` 를 setext 로 읽지 않는 이유로 "front matter 의 울타리" 를 든다. front matter 를 건너뛰어도 **이번 회차에 `-` 를 setext 로 읽기 시작하지 말 것**(별도 아이디어다). 주석이 어긋나면 한 구절만 고쳐 사실을 맞추고 동작은 그대로 둘 것.
- 실행 순서(각 단계 끝에 빌드가 서 있다):
  1) `markdownfrontmatter_test.go` 를 먼저 쓰고 `cd server && go test -count=1 ./internal/docs` 로 red 를 확인한다(재현 1건이 red, 삼키지 않는 두 경우와 빈 머리말은 계약 고정용이라 green 일 수 있다).
  2) `frontMatterLines` 헬퍼를 더하고 `readMarkdown` 에서 `lines` 를 자른다 → 같은 명령으로 green. 이 시점에 `TestTheRulesAroundFrontMatterAreNotPoints` 가 red 가 되는 것이 정상이다.
  3) `markdownrule_test.go:35` 의 이름·주석·기대값을 새 계약으로 고친다 → green.
  4) `go test -race ./...` + `go vet ./...` + `gofmt -l internal/docs` + `git diff --check`.
  5) `docs/USER_GUIDE.md:276` 한 문장. 사람 검토 체크포인트 없음(자율 회차).
- 추정 근거(범위): 프로덕션 1파일 + 테스트 2파일 + 문서 1줄 = S. 지난 다섯 회차(58ec05a·80b1be5·70f65fe·5bad44a·9517fbc)가 모두 같은 모양으로 한 세션에 끝난 유추 추정이고, 8할 신뢰로 30~50분. 예비는 `frontMatterLines` 의 가드 설계(YAML 키 모양)에만 두었다 — 거기서 넘치면 "닫는 `---` 이 없으면 아무것도 건너뛰지 않는다" 까지만 구현하고 가드는 다음 회차로 미루는 것이 아니라, **가드 없이 머지하지 말고 과제를 멈출 것**(문서를 삼키는 위험이 그 가드에 걸려 있다).
- 범위 밖: `title:` 을 덱 이름으로 쓰기, setext h2(`---`), 인용 `>`, 들여쓰기 코드 블록, `USER_GUIDE` 표를 절로 빼기, 버전·릴리즈 노트·배포 매니페스트.
- 차선 후보: 마크다운 인용(`> 문장`)의 `>` 가 `- \> 문장` 으로 사용자에게 보이는 것을 리더에서 떼기 (가치 2 / 위험 2 / S). 먼저 "여러 줄 인용을 한 요점으로 합칠지, 줄마다 요점으로 둘지" 를 한 줄로 정하고 그대로 구현할 것. `escapeLine`(docs.go:125)의 선두 `>` 보호는 덱 DSL 의 표지 부제 때문에 유지해야 한다.

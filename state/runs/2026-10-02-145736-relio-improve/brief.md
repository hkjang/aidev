# 과제서 — 2026-10-02-145736-relio-improve

- 과제: ESCAPE 선언 검사를 **줄 단위**에서 **비교 단위**로 옮기기 — 19개 비교 중 18개가 "같은 줄의 다른 ESCAPE" 에 가려져 있는 것을 닫기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/crm/search_test.go:48 TestEveryFreeTextSearchDeclaresItsEscapeCharacter` 는 파일을 `strings.Split(..., "\n")` 으로 줄로 쪼갠 뒤 `strings.Contains(line, "LIKE $") && !strings.Contains(line, "ESCAPE")` 만 본다(`:58-63`). 그런데 `internal/` 의 비-테스트 `.go` 에서 LIKE 비교는 **8줄에 19개**가 몰려 있고(아래 셈), 한 줄에 하나뿐인 자리는 `internal/voice/knowledge.go:167` **하나**다 — 나머지 7줄에서는 어느 비교의 `ESCAPE '\'` 를 지워도 같은 줄의 다른 `ESCAPE` 가 `Contains` 를 참으로 만들어 테스트가 green 이다. 즉 이 불변식은 19자리 중 1자리에서만 실제로 작동한다. 같은 파일의 다른 테스트(`:101 TestEveryFreeTextSearchComparesAgainstFoldedText`)는 이미 **비교 단위 위치**(`likeComparison.FindAllStringIndex` 로 리터럴 안의 오프셋)를 뽑고 있으므로, 같은 자리에서 그 비교의 **뒤쪽**을 보면 닫힌다. 닫히면 `crm.SearchPattern` 의 와일드카드 이스케이프 계약(`%`·`_`·`\` 를 리터럴로 쓰는 것)이 19자리 전부에서 지켜진다 — 지금은 한 자리에서만 지켜진다.

- 수용 기준:
  1) `TestEveryFreeTextSearchDeclaresItsEscapeCharacter` 가 줄이 아니라 **비교 하나하나**를 보고, 어느 한 비교에서 `ESCAPE` 가 빠지면 그 비교의 `path:line` 을 찍고 실패한다. 같은 줄에 ESCAPE 를 선언한 다른 비교가 있어도 실패해야 한다.
  2) 테스트 **이름**과 기존 실패 메시지 문구(`uses %s without ESCAPE; build the pattern with crm.SearchPattern`)를 유지한다 — 2026-09-30 회차의 섭동 ⑤ 가 이 메시지를 기대값으로 썼다. 비교 단위가 됐으므로 `%s` 에 들어가는 연산자 표기(`LIKE $` 등)는 정규식이 잡은 실제 텍스트로 맞춰도 된다.
  3) 스캔이 표에 **닿고 있음**을 개수 가드로 증명한다 — 현재 ESCAPE 테스트에는 가드가 전혀 없어, 걷기 루트가 어긋나거나 정규식이 아무것도 못 잡아도 영원히 green 이다. `TestEveryFreeTextSearchComparesAgainstFoldedText` 의 `knownComparisons = 19` / `t.Fatalf("only %d LIKE comparisons were read out of %s …")` 관용을 그대로 써서 두 테스트 **모두**가 19 미만이면 `t.Fatalf` 로 죽게 한다.
  4) 섭동으로 red 를 눈으로 본다(최소 3종, 전부 `git checkout --` 로 되돌릴 것):
     - `internal/crm/resources.go:82` 의 **첫 번째** `ESCAPE '\'` 만 제거 → 고치기 **전**에는 두 테스트 다 green 인 것을 먼저 확인하고(이것이 이 과제의 red), 고친 **뒤**에는 `resources.go:82` 를 짚으며 실패해야 한다.
     - `internal/server/admin_operations.go:99` 의 5개 `ILIKE … ESCAPE '\'` 중 가운데 하나의 `ESCAPE '\'` 만 제거 → 그 자리만 red.
     - `internal/voice/knowledge.go:167` 의 `ESCAPE '\\'` 제거 → 기존에도 잡혔던 자리가 **계속** 잡힘(회귀 없음).
  5) 기존 green 3개(`TestSearchPatternKeepsWildcardsLiteral`, `TestSearchPatternLeavesABlankQueryEmpty`, `TestEveryFreeTextSearchComparesAgainstFoldedText`)가 그대로 통과하고, 삭제된 동작 줄은 0이다.

- 건드릴 파일: **`internal/crm/search_test.go` 한 파일만. 프로덕션 0파일.**
  - `:48 TestEveryFreeTextSearchDeclaresItsEscapeCharacter` — `os.ReadFile` + 줄 쪼개기 + 연산자 4종(`LIKE '`/`LIKE $`/`ILIKE '`/`ILIKE $`) 루프를 걷어내고, 아래 권장 모양의 비교 단위 검사로 바꾼다.
  - `:89 likeComparison`(`\bI?LIKE\s+[$']`) — 그대로 재사용. 이 정규식이 `LIKE '` 와 `LIKE $` 를 모두 덮으므로 기존 연산자 4종 목록과 덮는 범위가 같다(오늘 19개 전부 `$` 형태).
  - `:101 TestEveryFreeTextSearchComparesAgainstFoldedText` — 걷기 부분만 공용 헬퍼 호출로 바뀌고, 판정 본문(`leftOperand`/`checkFormattedOperand` 호출)은 **그대로** 둔다.
  - `:102 knownComparisons` — 두 테스트가 함께 읽도록 파일 수준 상수로 올린다.
  - `leftOperand`/`isOperandByte`/`verbsBefore`/`checkFormattedOperand`/`enclosing`/`operandValues`/`literalsAssignedTo`/`stringLiteral`(`:159-321`) — 건드리지 말 것.

- 권장 모양(한 가지로 수렴): 두 테스트가 **같은 걷기**를 쓰게 공용 헬퍼를 하나 뽑는다. 두 테스트가 각자 걷으면(한쪽은 줄, 한쪽은 AST) 지금처럼 다시 어긋날 수 있고, 어차피 ESCAPE 쪽도 AST 로 가야 한다(문자열 리터럴만 봐야 주석·산문이 오탐으로 걸리지 않는다).

  ```go
  // likeSite is one free-text comparison found in a string literal.
  type likeSite struct {
      path  string
      line  int
      lit   *ast.BasicLit
      stack []ast.Node // ancestors, live only during the callback
      at    []int      // operator bounds inside lit.Value
      end   int        // start of the next comparison in this literal, else len(lit.Value)
  }

  // forEachLikeComparison parses every non-test .go file under internal/ and
  // calls fn once per comparison, returning how many it found.
  func forEachLikeComparison(t *testing.T, fn func(likeSite)) int
  ```
  - 걷기 본문은 `:105-149` 를 그대로 옮긴다(`filepath.Walk` + `parser.ParseFile(..., parser.SkipObjectResolution)` + `ast.Inspect` + `stack` 누적/절단 + `lit.Value` 오프셋으로 `line` 계산).
  - `end` 는 `FindAllStringIndex` 결과에서 **다음** 매치의 시작(없으면 `len(lit.Value)`).
  - ESCAPE 검사는 `strings.Contains(s.lit.Value[s.at[1]:s.end], "ESCAPE")`. 확인해 둔 것: `admin_operations.go:99` 는 매치가 `$` 직후에 끝나므로 창이 `4 ESCAPE '\' OR action ` 이 되어 각 비교가 자기 ESCAPE 를 본다(마지막은 `4 ESCAPE '\')`). `knowledge.go:167` 은 `%s LIKE $%d ESCAPE '\\'` 한 개라 창이 `%d ESCAPE '\\'`.
  - 접기 검사는 `s.stack`·`s.lit.Value[:s.at[0]]` 만 쓰므로 현재 코드와 입력이 같다. `stack` 은 `ast.Inspect` 중에 잘려 나가니 **콜백 안에서만** 쓰고 보관하지 말 것(`enclosing(s.stack)` 도 콜백 안에서 호출).
  - 비교가 두 리터럴에 걸쳐 쪼개져 있으면 창이 리터럴 끝에서 멈춰 오탐이 난다 — 오늘 그런 자리는 없고(19개 전부 한 리터럴 안), 조용히 통과하는 것보다 시끄럽게 실패하는 쪽이 맞으니 그대로 둔다. 주석으로 한 줄 적어 둘 것.

- 검증 명령 (이 저장소에서 실제로 도는 것, 순서대로):
  - `go test ./internal/crm/ -run 'TestSearchPattern|TestEveryFreeTextSearch' -v` — 4건 PASS 가 기준선(이번 회차 정찰이 직접 돌려 확인: 4 PASS, `ok … 0.032s`)
  - `go test ./...` / `go test -race ./...` / `go vet ./...` / `go build ./...`
  - `gofmt -l internal/crm/search_test.go` (무출력) / `git diff --check` (무출력)
  - `./scripts/check-env-contract.sh` / `./scripts/check-static-assets.sh`
  - `make test` 는 이제 `npm --prefix web test` 까지 돌린다(ed0352e). 이번 변경은 프런트와 무관하고 정찰 환경에서 `npm ci` 가 ETIMEDOUT 이었으므로, Go 쪽만 돌려도 충분하다. 돌릴 수 있으면 돌리되 실패가 네트워크면 그렇게 적을 것.

- 위험과 피할 것:
  - **프로덕션 코드를 고치지 말 것.** 이번 회차에 고칠 프로덕션 결함은 없다 — 오늘 19자리 전부 `ESCAPE` 를 올바로 선언하고 있다(정찰이 세어 확인). 고치는 것은 "그 계약을 지키는 그물" 이다. 섭동은 반드시 되돌리고 최종 `git diff` 에 `search_test.go` 외의 파일이 없는지 확인할 것.
  - 2026-09-30 회차가 세운 접기 불변식(`TestEveryFreeTextSearchComparesAgainstFoldedText`)의 **판정 로직을 재작성하지 말 것**. 걷기만 공용화하고 본문은 옮기기만 한다. 그 테스트가 빨개지면 공용화가 입력을 바꿨다는 뜻이다.
  - `admin_operations.go:99` 의 5개는 `ILIKE` 다. 접기 불변식은 `ILIKE` 를 건너뛰지만(`:131-133`) **ESCAPE 검사는 `ILIKE` 도 봐야 한다** — 이스케이프는 대소문자와 무관하다. 건너뛰는 분기를 ESCAPE 쪽에 복사하지 말 것.
  - 보호 경로(`internal/auth`·`internal/oidc`·`migrations/`·`.github/workflows/`)와 감사 경로(`admin_operations.go` 의 `ILIKE`)는 **읽기만**. 감사 질의의 `ILIKE` 를 `lower(...) LIKE` 로 통일하려 하지 말 것(인덱스 계획이 걸린다 — 보류 아이디어에 이미 적혀 있다).
  - `AdminPages.tsx`·`internal/mcp/server.go` 대규모 포매팅 금지(프로필의 위험 구역).
  - 이것은 **소스 불변식이지 런타임 증명이 아니다**. 실제 PostgreSQL 이 `%`·`_` 를 리터럴로 맞추는지는 이 테스트가 답하지 않는다 — 요약에 그렇게 적을 것(2026-09-30 회차와 같은 표현).

- 셈(정찰이 직접 센 것, 구현자가 다시 확인할 기준):
  | 자리 | 비교 수 | 연산자 |
  |---|---|---|
  | `internal/server/admin_operations.go:99` | 5 | `ILIKE $` |
  | `internal/crm/service.go:209` | 3 | `LIKE $` |
  | `internal/crm/service.go:559` | 2 | `LIKE $` |
  | `internal/crm/resources.go:30` | 2 | `LIKE $` |
  | `internal/crm/resources.go:82` | 2 | `LIKE $` |
  | `internal/crm/advanced.go:21` | 2 | `LIKE $` |
  | `internal/relationship/team.go:166` | 2 | `LIKE $` |
  | `internal/voice/knowledge.go:167` | 1 | `LIKE $` (fmt.Sprintf 조립) |
  | 합계 | **19** | `knownComparisons` 와 일치 |
  `cmd/` 와 `migrations/` 에는 LIKE 비교가 없다(정찰 확인) — 걷기 루트 `internal/` 은 그대로 둘 것.

- 차선 후보: **감사 화면 Frame 부제를 실제 채널값으로 맞추기** (가치 1 / 위험 1 / S) — `web/src/pages/AdminPages.tsx:619` 의 부제는 `Login과 Key` 인데 바로 아래 `:620` 드롭다운은 `['WEB','API','MCP','ADMIN','LOGIN','SSO']` 로, `Key` 는 존재하지 않는 채널이고 `SSO` 가 빠져 있다. 드롭다운 배열을 정본으로 삼아 부제 문자열 하나만 고친다. 주의: 이 파일은 한 줄이 매우 길어 대규모 재포매팅 금지, 그리고 정찰 환경에서 `npm ci` 가 ETIMEDOUT 이었으므로 착수 전 `npm --prefix web ci` 가 되는지 먼저 확인할 것. (정찰은 이번 회차에 `:619-620` 을 다시 열어 보지 못했다 — **미확인**, 2026-09-30 회차 기록에 근거.)

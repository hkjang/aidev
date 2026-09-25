- 과제: 토크나이저의 도달 불가능한 한국어 접미사 `별로` 를 살려 `…별로` 와 `…별` 이 같은 토큰이 되게 하고, 지금 테스트가 0 개인 `tokenize` 에 표 테스트를 깐다 (가치 4 / 위험 2 / 작업량 S)
- 왜: `internal/catalog/search.go:832` 의 접미사 목록이 `"로"` 를 `"별로"` 보다 먼저 놓아서 `별로` 분기는 절대 실행되지 않는다 — 루프는 첫 일치에서 `return` 하고, 어떤 문자열이 `별로` 로 끝나면 반드시 `로` 로도 끝나며 `로` 쪽 길이 가드(`>len(suf)+1` = 3 룬 초과 아님, `로` 는 2 초과)가 더 느슨해서 항상 `로` 가 먼저 먹는다. 결과로 같은 뜻의 두 표기가 다른 토큰이 된다: `회원사별` → `회원사`(목록의 `"별"` 이 처리), 그런데 `회원사별로` → `회원사별`. `tokenize` 는 search.go:91/325/664, graph.go:156/234, metrics.go:177, analyze.go:269, clarify.go:235, feedback.go:94, validate.go:496 등 8 개 이상 호출부의 공용 1차 토큰이라 이 불일치가 테이블·조인·지표 검색 리콜을 조용히 깎는다. 게다가 `internal/catalog/*_test.go` 에 `tokenize`/`stripKoreanSuffix` 를 부르는 테스트가 하나도 없어(이번 회차 grep 확인) 이 공용 프리미티브는 회귀 감지 장치가 없다.
- 수용 기준:
  1) `tokenize("회원사별로")` 의 결과가 `tokenize("회원사별")` 과 같다(둘 다 `회원사`). 같은 성질이 `지점별로`/`지점별`, `상품별로`/`상품별` 에서도 성립한다.
  2) 짧은 토큰은 변하지 않는다 — `tokenize("월별로")` 는 수정 전후 모두 `월별` 이다(길이 가드가 `별로`(3 룬 초과 요구)에서 실패해 `로` 로 흘러내려가야 한다). `tokenize("으로")`·`tokenize("바로")` 류 2~3 룬 입력의 기존 출력이 그대로여야 한다.
  3) 새 표 테스트가 **수정 전에 red** 인 것을 실제로 보여야 한다(기대: `회원사별로` 가 `회원사별` 로 나와 실패). 수정 후 green. 테스트는 `internal/catalog` 패키지 내부 테스트 파일에 두어 미공개 `tokenize` 를 직접 부르고, 손으로 만든 대역 없이 프로덕션 함수를 쓴다.
  4) 골든 평가 점수가 내려가지 않는다 — `go run ./cmd/jasql-eval` 을 **수정 전에 한 번 돌려 숫자를 적어 두고**, 수정 후 같은 명령의 숫자와 비교해 커밋 메시지/보고에 두 값을 같이 남긴다.
- 건드릴 파일:
  - `internal/catalog/search.go:831 stripKoreanSuffix` — 접미사 슬라이스에서 `"별로"` 를 `"로"` **앞으로** 옮기는 것만. (`"으로"` 와 `"별로"` 는 서로 부분집합이 아니므로 `"으로"` 와의 상대 순서는 무관하다.) 길이 가드 식·`return` 구조·나머지 접미사 순서는 건드리지 말 것.
  - `internal/catalog/tokenize_test.go` (신규) — `tokenize` 와 `stripKoreanSuffix` 의 표 테스트. 위 수용 기준 1·2 를 덮고, 덧붙여 현재 동작을 고정하는 특성화 케이스 몇 개(예: `"고객_ID, 이용금액"` 같은 구분자 분해, 대문자 소문자화, 중복 제거)를 같이 넣어 이후 회차가 토크나이저를 깰 때 잡히게 한다.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/catalog -run Tokenize -count=1 -v` (수정 전 red 확인 → 수정 후 green)
  - `go run ./cmd/jasql-eval` (수정 전/후 각 1 회, 점수 비교. 정찰에서는 Bash 승인이 막혀 이번 회차 실측 못 함 — **미확인**, 베이스라인 숫자는 구현자가 직접 찍을 것)
  - `go build ./...` · `go vet ./...` · `go test ./...` (catalog 약 55~57 초, mcp 약 10 초 — 이전 회차 실측치)
  - `gofmt -l ./internal ./cmd` — 신규 파일이 깨끗한지만 확인. `internal/oracle/profile.go`·`internal/oracle/oracle_test.go` 의 기존 드리프트는 이번 과제 범위가 아니니 **고치지 말 것**(diff 오염).
- 위험과 피할 것:
  - **`expandTokens`·동의어·점수 가중치·`unique`·`tokenize` 의 구분자 `strings.NewReplacer` 목록은 건드리지 말 것.** 과제는 접미사 한 개의 순서다.
  - **원문 문자열을 읽는 다른 경로와 통합하려 하지 말 것.** `internal/catalog/analyze.go:47/131/259` 와 `patterns.go:24/32` 는 `hasAny(q, "별", "별로", …)` 로 **질문 원문**을 본다 — 토큰이 아니다. 계약이 다른 파서이므로 함께 고치거나 합치면 안 된다. 다만 수정 후에도 `analyze_question` 의 `aggregation_level`·차원 추출 출력이 그대로인지 `go test ./internal/mcp -count=1` 로 확인할 것.
  - 골든 평가 임계값이 흔들리면(수용 기준 4 실패) **순서 변경은 되돌리고 표 테스트만 남겨** 특성화 테스트로 제출하고, 관측한 점수 차이를 보고에 적을 것. 이것이 정직한 축소이지 실패가 아니다.
  - `data/kcb/` 의 실데이터·골든셋을 출력 대상으로 쓰지 말 것. `go run ./cmd/jasql-goldgen` 은 기본 `-out` 이 `data/kcb/golden_queries.json` 을 덮어쓴다 — 이번 과제에서는 goldgen 을 돌릴 이유가 없다.
  - 보호 경로(`internal/mcp/auth.go`·`oauth.go`·`authapi.go`, `internal/oracle/sqlguard.go`·`execguard.go`, `internal/meta/pg.go`, `.github/workflows/`)는 전혀 손대지 않는다.
  - **이미 미머지 PR 이 떠 있는 과제 3 개는 절대 다시 하지 말 것** — 이번 회차 확인: `TrimStatement` 심볼이 base 에 없음(1507e00 = SQL trailing 정리 미머지), `caf7a00` 은 `auto/2026-09-22-1244` 브랜치에만 있음(eval `-verbose` MISS 미머지), `timeparse.go` 의 보고 단위 순서 결정성(c712da1)도 미머지. 이 세 영역은 피한다.
- 차선 후보: `internal/oracle` 의 gofmt 드리프트 2 파일(`profile.go`, `oracle_test.go`) 을 포맷만 정리하고, 그 뒤 `.github/workflows/ci.yml` 에 `gofmt -l` 게이트 단계를 추가 (가치 3 / 위험 1 / 작업량 S). 드리프트 존재는 2026-09-25 이전 회차가 실측했으나 이번 정찰에서는 `gofmt` 실행 승인이 막혀 **재확인 못 함(미확인)** — 구현자가 먼저 `gofmt -l ./internal ./cmd` 로 확인하고, 출력이 비어 있으면 이 차선은 성립하지 않으니 `docs/README.md` 의 데이터셋 개수를 `DatasetRegistry` 실제 개수와 맞추는 과제로 내려갈 것.

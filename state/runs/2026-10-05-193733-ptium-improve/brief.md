- 과제: 마크다운/텍스트 업로드에서 인용(`> 문장`)의 `>` 를 슬라이드 글자에서 떼기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `readMarkdown`(server/internal/docs/prose.go:205)의 `handle` 클로저(prose.go:218)에 인용 분기가 없어 `> 인용입니다.` 가 `default:`(prose.go:252)로 떨어져 `writer.point("> 인용입니다.")` 가 되고, `writeSlide`(writer.go:210)가 그 요점을 `escapeLine`(docs.go:125)에 통과시키면 선두 `>` 가 덱 DSL 의 표지 부제로 읽히지 않도록 `\` 로 보호되어 사용자 슬라이드에 **`- \> 인용입니다.`** 가 보인다. 수평선(9517fbc)과 YAML 머리말(f5988f4)이 떨어져 나간 지금, 마크다운 리더에 남은 가장 눈에 띄는 쓰레기 글자다. 고치면 인용이 그냥 요점이 되고, 인용 안의 제목·목록·표도 그 블록으로 읽힌다.
  - 재현은 코드를 읽어 도출한 것이고 **실행으로 확인하지 않았다(미확인)** — `>` 는 `atxHeading` 도, `""` 도, `|` 선두도, `isThematicBreak` 도 아니고 `listMarkers`(listmarker.go:13)에 없어 `default:` 밖에 갈 곳이 없다. 구현자는 먼저 red 테스트로 실제 출력 문자열을 확인할 것.
  - 이 동작을 박아 둔 기존 테스트는 없다(`internal/docs` 전체에서 `\>`·인용 관련 단언 0건 — Grep 확인). 기존 테스트 파일은 한 글자도 고칠 필요가 없을 것으로 본다.

- 수용 기준:
  1) `> 인용입니다.` 한 줄짜리 문서가 `- 인용입니다.` 요점이 된다 — 백슬래시도 `>` 도 없다. 여러 줄 인용은 **줄마다 한 요점**이다(합치지 않는다, 아래 "위험" 참고).
  2) 중첩 인용 `>> 깊은 인용`·`> > 깊은 인용` 도 마커를 전부 떼고 `- 깊은 인용` 이 된다.
  3) 인용 안의 블록이 그 블록으로 읽힌다: `> # 분기 요약` → 슬라이드 제목(덱 이름·`!source … | 분기 요약` locator 까지), `> - 항목` → `- 항목`(백슬래시 없음), `> | 분기 | 매출 |` → 표 행.
  4) 지켜야 하는 것(전후 출력이 한 글자도 같아야 한다): `>` 하나만 있는 줄은 요점을 만들지 않는다 / 줄 중간의 `>`(`매출 > 목표`)는 글자로 남는다 / 펜스 코드 블록 안의 `> 인용` 은 코드로 건너뛰어지고 경고의 줄 수에 포함된다 / `escapeLine` 은 손대지 않으므로 다른 리더(docx·pdf·표)가 만든 `>` 로 시작하는 요점은 여전히 `\>` 로 보호된다.
  5) 미닫힘 펜스 재생 루프(prose.go:317~323)도 같은 `handle` 을 부르므로 거기서도 인용 규칙이 적용되는 것을 테스트가 증명한다.
  6) `document.Warnings` 는 비어 있다 — 가져오지 못한 내용이 없는 구두점이다(수평선 9517fbc·머리말 f5988f4 와 같은 정책). 코드 블록·그림처럼 개수를 말하지 않는다.
  7) 기대값은 가능한 곳에서 **두 `Read` 비교**(인용 마커를 손으로 지운 같은 문서)로 쓴다 — f5988f4 회차가 쓴 방식이고 문자열을 박지 않는다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개 + 문서 1줄):
  - `server/internal/docs/prose.go`
    - 새 헬퍼 `withoutQuoteMarker(line string) (string, bool)` — `isThematicBreak`(prose.go:519 부근)·`frontMatterLines` 옆, `isRule` 앞에 둔다. 선두 `>` 를 하나 떼고 바로 뒤의 공백/탭 하나를 떼며, 남은 것이 또 `>` 로 시작하면 반복한다(중첩 인용). `>` 가 없으면 `(line, false)`.
    - `handle` 을 자기 자신을 부를 수 있게 선언만 바꾼다: `handle := func(...)` → `var handle func(line, next string) bool` + `handle = func(line, next string) bool {…}`. 로직은 그대로.
    - `switch` 에 `case strings.HasPrefix(line, ">"):` 를 **`case line == "":` 바로 뒤**에 넣는다. 인용은 마크다운에서 다른 블록을 감싸는 가장 바깥 블록이므로 다른 case 보다 먼저 벗기는 것이 원칙이고, `>` 는 `|`·수평선·목록 마커 어느 것과도 겹치지 않아 순서가 동작을 바꾸지는 않는다. 몸은: 마커를 떼고 — 남은 것이 빈 문자열이면 `flush()` 만(빈 인용 줄은 빈 줄이다) — 아니면 `return handle(rest, next)` 로 그 줄을 **벗겨진 모습으로 다시 읽는다**. `>` 를 한 개 이상 실제로 떼었을 때만 재귀하므로 끝난다.
    - 주석은 이 저장소 밀도(prose.go:262~268, 485~489 가 표본)로, **거절한 선택지의 이유까지** 적는다: ① 이어지는 인용 줄을 한 요점으로 합치지 않는 이유 — 이 리더가 줄을 넘겨 들고 있는 블록은 표 하나뿐이고, 합치려면 새 블록 상태와 `maximumPoints` 와의 상호작용이 생긴다 ② `next` 를 벗기지 않는 이유 — `> 제목` / `> ===` 같은 인용 안 setext 는 범위 밖이고, 벗긴 `next` 를 넘기면 앞보기와 `skip` 의 뜻이 어긋난다.
  - `server/internal/docs/markdownquote_test.go` (신규) — 위 수용 기준 1~6을 덮는 테스트. 영어 문장형 이름(`A quoted sentence is a point without its marker` 투). 먼저 넣어 red 를 확인하고 실패 출력을 회차 노트에 적을 것.
  - `docs/USER_GUIDE.md:276` 의 `.md` · `.txt` 칸에 **한 문장**만 더한다(예: "`> 인용` 의 `>` 는 떼고 그 안의 내용만 남습니다"). 이 칸은 이미 다섯 규칙·450자라 — 절로 빼는 것은 별도 아이디어(차선 후보)이니 이번 회차에 같이 하지 말 것.

- 검증 명령:
  - `cd server && go test -count=1 ./internal/docs` (기준선 2.919s ok — 이번 세션에 실행 확인)
  - `cd server && go test -race ./...` (26개 패키지, ~25s) / `cd server && go vet ./...` / `cd server && gofmt -l internal/docs`(출력 없음) / `git diff --check`
  - 웹·API·OpenAPI 변경이 없으므로 `make test` 의 웹 단계는 건너뛴다(회차 관례).

- 위험과 피할 것:
  - **`escapeLine`(docs.go:125)의 `>` 보호를 지우지 말 것.** 덱 DSL 은 표지 부제를 `> 월간 보고서.md` 로 쓰고 커버 슬라이드가 실제로 그 모양이다 — 보호를 지우면 docx·pdf·표 리더가 만든 요점이 부제 지시어로 읽힌다. 고치는 자리는 리더(마커 제거)이지 이스케이프가 아니다.
  - `internal/deck/source.go`(덱 DSL, golden 회귀가 그 위에 있다)·`writer.go`·`tables.go`·숫자 파서(`amountOf`/`allNumeric` ↔ deck 의 `parseNumber`/`bareNumber`)는 한 글자도 건드리지 말 것 — 숫자 파서 통합은 2026-09-09 에 두 번 반려됐다.
  - 재귀는 **`>` 를 실제로 떼었을 때만**. 떼지 못한 채 `handle(line, …)` 를 다시 부르면 무한 루프다.
  - `writer.point` 는 `maximumPoints` 에서 `(계속)` 슬라이드를 만들어 되돌릴 수 없다 — 쓰기 전에 판단할 것(앞보기 구조가 그 이유다).
  - `- > 인용`(목록 안 인용)은 `isListLine` 분기로 가 `> 인용` 요점이 되어 여전히 `\>` 가 남는다. **이번 범위 밖**으로 두고 커밋 메시지·주석에 적을 것(목록 분기에서 재귀를 추가하면 범위가 커지고 `withoutListMarker` 계약을 건드린다).
  - 보호 경로(auth·httpapi·db/migrations.go·scripts/release.sh)는 이 과제와 무관하다 — 열지 말 것. 버전·릴리즈 노트·배포 매니페스트(VERSION·api/openapi.yaml·deploy/kubernetes.yaml·docs/offline-deployment.md)도 손대지 않는다.
  - 샌드박스: `python3 -c`·히어독·복합 명령(`A && B`)은 막혀 있고 `cd server && go test …` 는 통과한다. PTIUM_TEST_DSN 이 없어 DB 테스트는 Skip 된다(정상).

- 차선 후보: `docs/USER_GUIDE.md:276` 의 `.md` · `.txt` 한 칸(다섯 규칙·450자)을 표에서는 한 문장으로 줄이고 규칙은 별도 절로 옮겨 링크만 걸기 (가치 3 / 위험 1 / 작업량 S). 정본이 둘이 되지 않게 표에는 규칙을 남기지 말 것.

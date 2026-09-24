# 과제서 — 2026-09-22

- 과제: XLSX 셀 참조의 열 번호를 시트의 실제 한계(XFD=16384)로 묶고, 세 글자 열 이름(AAA~XFD)을 출처에 바르게 쓰기 (가치 4 / 위험 2 / 작업량 S)

- 왜: `server/internal/docs/workbook.go:444 columnOf` 는 참조의 글자 수에 상한이 없어 `<c r="AAAAA1">` 같은 셀 하나가 열 번호 12,356,629 를 만들고, 바로 뒤 `gridOf`(workbook.go:419)의 `line := make([]string, widest+1)` 가 **행마다** 그 크기를 잡는다 — 셀 여섯 개짜리 업로드 파일로 서버 메모리를 몇 GB 먹일 수 있다(이 저장소가 이미 같은 이유로 숨긴 열 범위를 전개하지 않기로 한 것과 정확히 같은 종류의 구멍이다; workbook.go:255 주석 참고). 또 `tables.go:227 columnLetter` 는 두 글자까지만 만들어서 index 702 이상이면 `'A'+index/26-1` 이 'Z' 를 넘어가 `"[A"` 같은 글자가 출처 인용에 들어간다(열이 703개 이상인 시트는 Excel 에서 정상). 열 번호를 XFD 로 묶으면 할당이 시트 한 줄당 16,384칸으로 유한해지고, 그때 비로소 필요한 세 글자 표기를 같이 맞출 수 있다.

- 수용 기준:
  1) 열 참조가 `XFD`(0 기준 16383)를 넘는 셀이 있는 워크북을 `Read` 해도 그 행의 grid 길이가 16,384칸을 넘지 않는다 — 테스트가 큰 참조를 담은 실제 XLSX ZIP 을 만들어 `Read` 가 시간·메모리 폭주 없이 돌아오고 결과 표가 온전함을 보인다.
  2) 한계를 넘는 참조를 가진 셀은 **버려지지 않고** 지금의 "참조 없는 셀" 규칙대로 바로 앞 셀 다음 자리(`gridOf` 의 `next`)에 놓인다 — 같은 행의 정상 셀들의 자리는 바뀌지 않는다.
  3) 출처 인용이 세 글자 열을 바르게 쓴다: `columnLetter(701)=="ZZ"`, `columnLetter(702)=="AAA"`, `columnLetter(16383)=="XFD"` — 그리고 열이 703개인 시트를 `Read`→`deck.ParseSource` 한 끝에서 인용 문자열이 `AAA` 이상을 담고 `[` 같은 글자를 담지 않는다.
  4) 기존 동작 회귀 없음: 숨긴 행·열 경고(`sheetconcealed_test.go`), 생략 행의 r 좌표 출처, 인라인 rich text, 숫자·날짜 서식 테스트가 그대로 통과한다.

- 건드릴 파일:
  - `server/internal/docs/workbook.go:444 columnOf` — 글자를 누적하는 루프 안에서 `column` 이 16384 를 넘으면 그 자리에서 "열을 못 읽었다"(-1)로 빠진다. 루프 **안에서** 검사해야 글자가 많은 참조(`AAAAAAAAAA1`)에서 int 가 넘치지 않는다. 함수 주석(참조가 열을 안 가리키는 경우)에 "시트에 없는 열 번호" 도 같은 경우임을 한 문장으로 적을 것.
  - `server/internal/docs/tables.go:227 columnLetter` — 26진 자리올림을 되풀이하는 일반 변환으로 바꿔 세 글자까지(XFD) 낸다. 음수는 지금처럼 0(`A`)으로.
  - `server/internal/docs/` 에 새 테스트 파일(예: `sheetwidth_test.go`) — 기존 `sheetconcealed_test.go`/`sheetrichtext_test.go` 가 쓰는 방식(`archive/zip` 으로 진짜 xlsx 바이트를 만들고 `Read`→`deck.ParseSource` 까지 통과)을 그대로 따를 것. 손으로 만든 구조체에 직접 `gridOf` 를 먹이는 단위 검사만으로 증거를 삼지 말 것(운영자 지시).
  - 문서는 손대지 않아도 된다(사용자에게 보이는 문법·가이드 변화 없음). `VERSION`·릴리스 노트는 건드리지 않는다.

- 검증 명령:
  - `cd server && go test ./internal/docs`  (집중 검증, 약 2초 — 이번 정찰에서 통과 확인)
  - `cd server && go test -race ./... && go vet ./...`  (전체)
  - `git diff --check`
  - 웹 변경이 없으므로 `make test` 의 웹 단계(`npm ci`·`tsc`·`vite build`)는 건너뛴다. 이 워크트리에 `web/node_modules` 는 없다.

- 위험과 피할 것:
  - **숫자 파서를 건드리지 말 것.** `docs.amountOf`·`allNumeric`·`deck` 의 `parseNumber`/`parseBareNumber`/`chartFields` 는 서로 다른 계약을 지키고 있고, 과거 두 번 반려된 자리다. 이번 과제는 열 *좌표* 만 다룬다.
  - `onScreen`(workbook.go:263)의 숨김 제거·`placement` 매핑, `gridOf` 의 시그니처와 셀 타입 분기(`s`/`inlineStr`/`b`/`d`)는 그대로 둘 것. 최근 세 회차가 모두 이 함수들을 고쳤으므로 회귀가 나기 쉽다.
  - 한계 초과 셀을 **조용히 버리는** 선택은 하지 말 것 — 값이 사라지면 사용자에게 보이지 않는 손실이 된다. 경고 한 줄을 새로 만드는 것도 이번 범위 밖이다(문구·한국어 조사 검사 `korean` 패키지까지 번지고 45분을 넘긴다).
  - 열 번호 상한은 **16384(XFD)** 로, Excel 의 실제 시트 한계에 맞출 것. 임의로 작은 값(예: 256)을 고르면 정상 파일의 열을 잘라먹는다.
  - 보호 경로(auth·httpapi 인증·`db/migrations.go`·handoff·mail)는 건드리지 않는다. 이 과제는 `internal/docs` 두 파일 + 새 테스트로 끝난다.
  - 남는 인접 위험(이번 범위 밖, 기록만): 행 수 자체에는 여전히 상한이 없어 16,384칸 × 많은 행이면 메모리는 계속 늘 수 있다. ZIP 전체 해제 크기 제한은 별도 보류 항목이다.

- 차선 후보: e2e 후속 — `scripts/e2e/api.py` 의 `call()` 이 `headers={}` 를 개발 신원으로 바꿔 버리는 함정을 없애고, 빈 dict 를 "자격 없음" 으로 읽게 하기 (가치 2 / 위험 1 / 작업량 S). 1순위가 성립하지 않을 때만 고를 것 — 기존 `headers={}` 호출(공유 링크·로그인 시도)이 뜻한 대로였는지 전수 확인이 필요하고, 실서버·DB 가 있어야 돌려 볼 수 있다.

# 과제서 (2026-10-14 정찰)

- 과제: CSV 구분자 자동 감지가 "제목 줄"에 끌려가지 않게 (가치 4 / 위험 2 / 작업량 S)

- 왜: `detectDelimiter`(packages/tools/src/table/csv.ts:35)는 점수를 **첫 줄의 구분자 개수**(`const first = counts[0] ?? 0`)에 고정해서 매긴다. 그런데 이 저장소는 직전 커밋 74458bf "실제 파일 호환: 제목 줄 위 표"에서 표 위에 제목 줄이 있는 파일을 정식으로 지원하기로 했고(`guessHeaderLine`, packages/tools/src/table/types.ts:82), 그런 파일의 첫 줄은 데이터 줄과 구분자 개수가 다르다. 제목 줄에 쉼표가 하나라도 들어가고(예: `2026년 10월, 자산 목록`) 본문이 세미콜론·탭으로 나뉜 파일이면, 쉼표가 `consistent=1 → score 1001`, 세미콜론이 `first=0 → score 0` 이 되어 **쉼표가 이긴다**. 그 결과 표 전체가 1열로 읽혀 대조기가 말이 안 되는 결과(전부 추가/삭제)를 내놓는다. 고치면 제목 줄이 있는 실제 업무 파일에서 대조기가 처음부터 제대로 열린다.

- 수용 기준:
  1) 제목 줄에 쉼표가 들어 있고 본문이 세미콜론(또는 탭)으로 나뉜 CSV 텍스트에서 `detectDelimiter` 가 본문의 구분자를 돌려준다.
  2) 기존 동작이 그대로다 — `detectDelimiter('a,b\n1,2') === ','`, `detectDelimiter('a\tb\tc\n1\t2\t3') === '\t'`, `detectDelimiter('a;b\n1;2') === ';'` (packages/tools/src/__tests__/compare.test.ts:46-47 에 이미 있는 단언), 구분자가 하나도 없는 한 열짜리 파일은 여전히 `','`.
  3) 새 시험이 "제목 줄이 있는 파일" 을 `parseCsv(text, detectDelimiter(text))` → `toTable` 까지 끝까지 돌려서 **열 개수가 1 이 아니라 본문 열 개수**가 나오는 것을 증명한다(감지 함수만 단위로 보지 말고 표가 제대로 서는 것까지). robustness.test.ts:25 의 무작위 알파벳 시험은 그대로 통과해야 한다.

- 건드릴 파일 (2개, 생산 코드 1개):
  - `packages/tools/src/table/csv.ts:35 detectDelimiter` — 점수 기준을 "첫 줄의 개수" 에서 "줄들 사이에서 가장 많이 반복되는 0 이 아닌 개수(최빈값)" 로 바꾼다. 권장 구현: 후보마다 `counts` 에서 0 을 뺀 값들의 빈도표를 만들고, 가장 많이 나온 개수 `modal` 과 그 줄 수 `hits` 를 구해 `score = hits * 1000 + modal`. 전부 0 이면 지금처럼 `continue`. 동점이면 `CANDIDATES` 순서(`,` `\t` `;` `|`)가 이기도록 `>` 비교를 유지할 것.
  - `packages/tools/src/__tests__/compare.test.ts` (43행대 'quotes, doubled quotes, line breaks inside quotes, delimiters' 블록 또는 바로 뒤) — 위 수용 기준 1)·2)·3) 시험 추가. 파일 상단 헬퍼 `table(csv)`(35행)를 그대로 쓸 수 있다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  npm ci                 # 워크트리에 node_modules 가 없어 vitest 가 없다. 정찰 때 확인함
  npx vitest run packages/tools/src/__tests__/compare.test.ts packages/tools/src/__tests__/robustness.test.ts
  npm run check          # lint + typecheck + 단위 시험 전체 (CI 의 1단계와 같다)
  ```
  `npm run tools`(도구 화면 흐름)·`npm run e2e` 는 Playwright·PostgreSQL·가짜 서버가 필요하므로 이 변경에는 돌리지 않아도 된다. 다만 `npm run check` 는 반드시 통과시킬 것.

- 위험과 피할 것:
  - **`parseCsv` 는 건드리지 말 것.** 구분자 감지만 고친다. 파서를 같이 손대면 robustness.test.ts 의 무작위 입력 시험과 `npm run tools` 의 5만 행 시험이 함께 흔들린다.
  - **같은 종류의 파서가 여러 곳에 있다**(`table/csv.ts`, `table/xlsx-read.ts`, `apps/web/src/tools/compare/compare.worker.ts`). 통합하지 말고 `detectDelimiter` 한 함수만 좁게 고칠 것. worker(`compare.worker.ts:88`)는 `req.delimiter ?? detectDelimiter(text)` 로 부르기만 하므로 손댈 필요가 없다.
  - 기존 시험 두 줄(`compare.test.ts:46-47`)의 기대값을 바꾸지 말 것. 바꿔야 한다면 그 구현은 틀린 것이다.
  - 보호 경로(`packages/core/src/auth/*`, `db/` 마이그레이션, `.github/workflows/`)는 이 과제와 무관하다 — 열지 말 것.
  - `text.split(/\r?\n/)` 가 10 MB 전체를 쪼개는 성능 문제도 같은 함수 안에 있지만 **이번 회차에 끼워 넣지 말 것**(별건, ideas.json 에 pending 으로 남겨 둠).
  - 커밋 메시지는 이 저장소 관례대로 한국어 한 줄(예: `파일 대조: 구분자 감지가 제목 줄에 끌려가지 않게`).

- 차선 후보: **빈 줄만 있는 CSV 가 행 한도를 우회한다** — `parseCsv`(csv.ts:76 `end()`)에서 `LIMITS.rows`·`LIMITS.cells` 검사가 "내용이 있는 레코드" 분기 안에만 있어서, 개행만 가득한 10 MB 파일은 수백만 개의 빈 행 객체를 만든 뒤에야 `empty` 로 끝난다(브라우저가 멈춘다). 빈 줄 분기(csv.ts:86)에도 행 한도 검사를 넣고, 한도를 넘으면 `too_many_rows` 로 끊는다. 파일 1개 + 시험 1개.

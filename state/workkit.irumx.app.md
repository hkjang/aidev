## 2026-10-14
- 선택: CSV 구분자 자동 감지가 "제목 줄"에 끌려가지 않게 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `detectDelimiter`(packages/tools/src/table/csv.ts)가 점수를 첫 줄의 구분자 개수(`counts[0]`)에 고정해 매겨서, 표 위에 제목 줄이 있고 그 줄에 쉼표가 하나라도 들어간 파일에서는 쉼표가 본문의 세미콜론·탭을 이겼다. 점수 기준을 "줄들 사이에서 가장 많이 반복되는 0 이 아닌 개수(최빈값)" 로 바꿨다 — `score = hits * 1000 + modal`, 빈도가 동점이면 더 큰 개수를 쓰고 후보 순서(`,` `\t` `;` `|`)는 `>` 비교로 그대로 유지. `parseCsv` 와 다른 경로의 파서(xlsx-read, compare.worker)는 손대지 않았다. 검증: `npx vitest run compare.test.ts robustness.test.ts` (31 passed), `npm run check` (lint + typecheck 4개 패키지 + vitest 10 파일 122 passed).
- 실패 재현: `AssertionError: expected ',' to be ';'` (compare.test.ts:53) 와 `AssertionError: expected [ '2026년 10월', '자산 목록' ] to deeply equal [ '자산번호', '이름', '사용 부서' ]` (compare.test.ts:64) — 고치기 전 2 failed | 25 passed. end-to-end 쪽은 제목 줄이 헤더로 잡히고 열이 2개로 서는 것까지 그대로 드러났다.
- 보류 아이디어: ① 빈 줄만 있는 CSV 가 행 한도를 우회해 브라우저를 멈춘다(csv.ts end() 의 빈 줄 분기에 LIMITS.rows 검사 없음, 3/2/S — 차선 후보였고 다음 회차 1순위) ② detectDelimiter 가 10MB 텍스트 전체를 split 한 뒤 30줄만 쓴다(앞 64KB 만 잘라 쪼개기, 2/1/S) ③ serialToIso 가 Excel 1900 윤년 버그를 반영하지 않는다(xlsx-read.ts:228, 2/2/S) ④ writeXlsx/writeDocx 의 `Math.max(0, ...rows.map(...))` spread 가 5만 행 내보내기에서 인자 한도에 근접한다(미확인, 2/2/S) ⑤ docx runs() 가 '\n' 만 쪼개 CRLF 텍스트에 \r 가 남는다(미확인, 1/1/S)
- 과제서: 채택 — 근거(`counts[0]` 고정, 직전 커밋 74458bf 의 제목 줄 지원, 권장 구현·검증 명령)가 지금 코드와 정확히 맞았고 제안한 최빈값 점수가 기존 단언 세 줄을 모두 유지했다.

- 릴리즈: 0.3.1 (2026-10-14, run 2026-10-14-090819-workkit.irumx.app-improve)
## 2026-10-14
- 선택: 빈 줄만 가득한 CSV 가 행 한도 검사를 우회해 브라우저를 멈추는 것 막기 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `parseCsv`(packages/tools/src/table/csv.ts) 의 `end()` 안에서 `LIMITS.rows` 검사가 "내용 있는 레코드" 분기 안에만 있어, 개행만 든 파일은 한도(50,000)를 넘겨도 멈추지 않고 최대 약 1,000만 개의 빈 행 객체를 쌓은 뒤에야 `empty` 로 끝났다(10MB 한도 기준). 행 한도 검사를 `push()` 직후로 한 줄 올려 두 분기가 공유하게 했고, `cells > LIMITS.cells` 검사는 빈 행이 셀을 0개 더하므로 내용 분기에 그대로 남겼다. 행 번호(`line: rows.length + 1`)와 빈 행 push 는 손대지 않아 사용자에게 보이는 행 번호가 그대로다. 검증: `npx vitest run robustness.test.ts compare.test.ts` (32 passed), `npm run check` (lint + typecheck 4개 패키지 + vitest 10 파일 123 passed).
- 실패 재현: `AssertionError: expected error to match asymmetric matcher` — `- ObjectContaining { "code": "too_many_rows" }` / `+ TableError { "message": "내용이 있는 행이 없습니다.", "code": "empty" }` (robustness.test.ts:73, 고치기 전 1 failed | 4 passed)
- 보류 아이디어: ① detectDelimiter 가 10MB 텍스트 전체를 split 한 뒤 30줄만 쓴다 — 앞 64KB 만 잘라 쪼개기(2/1/S, 다음 회차 1순위) ② serialToIso 가 Excel 1900 윤년 버그를 반영하지 않는다(xlsx-read.ts:228, 2/2/S) ③ writeXlsx/writeDocx 의 `Math.max(0, ...rows.map(...))` spread 가 5만 행 내보내기에서 인자 한도에 근접한다(미확인, 2/2/S) ④ 병합 셀 제목 줄에서 guessHeaderLine 이 제목 줄을 헤더로 잡는지 확인(types.ts:82~89, 2/2/S) ⑤ docx runs() 가 '\n' 만 쪼개 CRLF 텍스트에 \r 가 남는다(미확인, 1/1/S)
- 과제서: 채택 — 근거(csv.ts:93 의 행 한도 검사가 내용 분기 안에만 있음, 빈 줄 분기 csv.ts:99)가 지금 코드와 정확히 맞았고, 권장한 "검사를 push() 직후로 올려 공유" 가 기존 단언(compare.test.ts:85 행 한도, 43~74 CSV 읽기, empty 에러)을 전부 유지했다.


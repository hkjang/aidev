- 과제: 빈 줄만 가득한 CSV 가 행 한도 검사를 우회해 브라우저를 멈추는 것 막기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `parseCsv`(`packages/tools/src/table/csv.ts:89` `end()`)의 `LIMITS.rows` 검사가 "내용 있는 레코드" 분기(csv.ts:91~96) 안에만 있어서, 개행만 들어 있는 파일은 한도(50,000)를 한참 넘겨도 멈추지 않고 `rows.push({ line, cells: [] })`(csv.ts:99)를 계속 쌓는다. `LIMITS.fileBytes` 가 10MB 이므로 최대 약 1,000만 개의 빈 행 객체를 만든 뒤에야 마지막(csv.ts:124)에 `empty` 로 끝난다 — 이 파일의 시험 설명(`robustness.test.ts:1~5`)이 약속한 "crash, hang 없이 TableError" 를 깨는 자리다. 빈 줄 분기에도 같은 행 한도를 적용하면 50,001번째 줄에서 즉시 `too_many_rows` 로 끝나 워커 탭이 메모리를 먹고 멈추는 일이 없어진다.

- 수용 기준:
  1) `parseCsv('\n'.repeat(LIMITS.rows + 2), ',')` 가 `TableError` 를 던지고 `code === 'too_many_rows'` 다 (지금은 한참 뒤에 `empty` 가 난다).
  2) 기존 동작이 그대로다: 짧은 입력에서 빈 줄이 행 번호를 차지하는 것(`end()` 의 `line: rows.length + 1` 계산), 내용 없는 파일의 `empty` 에러(csv.ts:124), `compare.test.ts:85` 의 행 한도 시험, `compare.test.ts:43~74` 의 CSV 읽기 단언 전부.
  3) 시험이 "한도를 넘는 빈 줄 입력은 전부 쌓은 뒤가 아니라 한도에서 멈춘다" 를 증명한다 — 에러 코드 단언으로 충분하다. 시간 단언(`toBeLessThan`)을 쓰려면 `robustness.test.ts:68` 처럼 넉넉한 값으로.

- 건드릴 파일 (프로덕션 1개 + 시험 1개):
  - `packages/tools/src/table/csv.ts` — `parseCsv` 안의 `end()` 빈 줄 분기(현재 csv.ts:97~100)에 `if (rows.length + 1 > LIMITS.rows) throw new TableError('too_many_rows', …)` 를 추가. 내용 있는 분기(csv.ts:93)와 메시지 문자열을 똑같이 쓰거나 검사를 `push()` 직후 한 줄로 올려 두 분기가 공유하게 하는 것 중 하나 — 후자가 더 짧다(`cells` 검사는 내용 분기에만 남겨야 한다. 빈 행은 셀을 0개 더하므로 `cells > LIMITS.cells` 는 빈 줄에서 의미가 없다).
  - `packages/tools/src/__tests__/robustness.test.ts` — `describe('large files')` 에 시험 1개 추가(예: `it('빈 줄만 가득한 CSV 는 행 한도에서 멈춘다')`). `LIMITS` 는 아직 이 파일에서 import 하지 않으므로 `../index.js` import 목록에 추가해야 한다(`compare.test.ts` 가 이미 `LIMITS` 를 쓰므로 index 에서 나온다 — 확인함).

- 검증 명령 (이 워크트리에 `node_modules` 가 없다 — 안 하면 `vitest: not found`):
  1) `npm ci`
  2) `npx vitest run robustness.test.ts compare.test.ts` — 고치기 전에 먼저 새 시험만 돌려 실패를 확인할 것(기대 실패: `expected [Error: 내용이 있는 행이 없습니다.] to have property code 'too_many_rows'` 또는 시간 초과). **미확인: 나는 node_modules 가 없어 시험을 실제로 돌리지 못했다. 실패 메시지 문구는 추정이다.**
  3) `npm run check` (lint + typecheck + vitest 전체) — 1차 관문.

- 위험과 피할 것:
  - 같은 종류의 파서가 `packages/tools/src/table/xlsx-read.ts` 와 `apps/web/src/tools/compare/compare.worker.ts`(parseCsv 호출 지점은 compare.worker.ts:89) 에도 있다 — **통합하지 말고 csv.ts 한쪽만 좁게 고칠 것**(운영자 지시).
  - `detectDelimiter` 는 직전 회차(53ea324)에서 고쳤다. 이번에 건드리지 말 것 — 보류 아이디어의 "앞 64KB 만 split" 은 별건이다.
  - **행 번호 계산을 바꾸지 말 것.** `line: rows.length + 1` 이 빈 행이 `rows` 에 들어 있다는 전제에 의존하므로, "빈 행을 아예 push 하지 않기" 로 메모리를 줄이려 들면 사용자에게 보이는 행 번호가 전부 밀린다(대조 보고서가 이 번호를 쓴다). 한도 검사만 추가하는 것으로 끝낼 것.
  - 보호 경로(`packages/core/src/auth/*`, `domain/payments.ts`, `db/` 마이그레이션, `.github/workflows/`)와 무관하다 — 열지 말 것.
  - `npm run e2e`·`tools`·`ui`·`a11y` 는 Docker PostgreSQL·Playwright·가짜 서버가 필요하다. 이 과제는 단위 시험만으로 끝난다 — 돌리지 말 것.

- 차선 후보: `detectDelimiter` 가 10MB 텍스트 전체를 `split(/\r?\n/)` 한 뒤 앞 30줄만 쓰는 것(csv.ts:41)을 앞 64KB 만 잘라 쪼개도록 좁히기 (가치 2 / 위험 1 / 작업량 S). 마지막 잘린 줄을 버리는 처리가 필요하고, `compare.test.ts:46~60` 의 구분자 단언이 그대로 통과해야 한다.

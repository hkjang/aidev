# 과제서 (2026-10-15 정찰)

- 과제: 저장한 비교 조건이 **이름이 겹치는 열**에 조용히 잘못 붙는 것 막기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `applyPreset`(packages/tools/src/compare/preset.ts:35~)이 저장한 열을 `t.headers.findIndex((h) => h.trim() === name.trim())` 로 **처음 일치하는 하나**만 찾는다. 실무 엑셀에는 `비고`·`금액`·`담당` 같은 열 이름이 한 시트에 두 번 나오는 일이 흔한데, 사용자가 두 번째 `비고`(index 7)를 골라 조건을 저장해도 다음 달 재적용 때는 첫 번째 `비고`(index 3)에 붙는다 — 경고도 메모도 없이 **다른 열을 비교한 결과**가 나오고, 이 도구의 존재 이유(바뀐 값을 믿을 수 있게 보여 주기)가 거기서 무너진다. 이 모듈의 파일 상단 주석이 이미 계약을 적어 두었다: "파일들의 구조가 달라졌으면(기준 열이나 비교 열이 없어졌으면, 시트 이름이 바뀌었으면) **사용자가 확인하기 전까지 아무것도 돌지 않는다**." 이름이 두 곳을 가리키는 것은 "없어진 것" 과 똑같이 그대로 적용하면 안 되는 모호함인데 지금은 걸러지지 않는다.

- 수용 기준:
  1) 저장한 **기준 열**(`preset.keys`) 이름이 이전/새 파일 중 어느 쪽에서든 두 개 이상의 헤더와 일치하면 `applyPreset` 이 `problems` 에 한국어 한 줄을 넣고 `settings: null` 로 돌려준다(= `apps/web/src/tools/compare/CompareTool.tsx:326` 의 `applySaved` 가 이미 `if (!fit.settings)` 분기에서 적용을 멈추고 주황 알림을 띄운다 — UI 는 건드릴 필요 없다).
  2) 저장한 **비교 열**(`preset.pairs`) 이름이 겹칠 때도 같은 기준으로 `problems` 에 들어간다. 메시지는 기존 문장들과 같은 꼴(`이전 파일에 기준 열 'X'이(가) 없습니다.` 와 나란히 읽히게, 예: `이전 파일에 '비고' 열이 2개 있어 어느 것인지 알 수 없습니다.`)로 쓴다.
  3) 이름이 겹치지 않는 기존 동작은 한 글자도 바뀌지 않는다 — `compare.test.ts:302~316` 의 '저장한 조건' 테스트(열 순서가 바뀐 파일에 재적용 → `problems` 빈 배열, `newKeys === [1]`, 열이 없어진 파일 → `settings === null`)가 그대로 통과한다.
  4) 테스트가 증명할 것: 헤더에 같은 이름이 두 번 있는 새 파일로 재적용하면 (a) `settings` 가 `null` 이고 (b) `problems` 에 그 열 이름이 들어간다. 고치기 전에는 이 테스트가 **빨간불**이어야 한다(지금은 `problems` 가 비고 `settings` 가 첫 번째 열을 가리킨 채 돌아온다) — 먼저 빨간불을 확인하고 나서 고칠 것.

- 건드릴 파일 (프로덕션 1개):
  - `packages/tools/src/compare/preset.ts:applyPreset` — `find` 가 "처음 일치" 대신 일치 개수를 세게 바꾼다. 예: `const hits = (t: Table, name: string) => t.headers.map((h, i) => [h.trim(), i] as const).filter(([h]) => h === name.trim()).map(([, i]) => i)` 를 두고, 길이 0 이면 지금의 "없습니다" problem, 길이 ≥2 면 새 모호함 problem, 1 이면 그 인덱스를 쓴다. `oldKeys`/`newKeys` 가 `preset.keys` 와 **길이·순서를 유지**해야 하므로(`presetFrom` 이 `oldKeys[i]` ↔ `newKeys[i]` 로 짝지어 저장하고, `CompareTool.tsx:336` 의 `newKeys.forEach((n, i) => map[...] = oldKeys[i]!)` 가 그 짝을 믿는다) 모호할 때도 지금처럼 `push(-1)` 해서 배열 길이는 그대로 두고 `problems` 로만 막는다 — `problems.length` 가 있으면 `settings: null` 로 빠지므로 `-1` 은 밖으로 나가지 않는다.
  - `packages/tools/src/__tests__/compare.test.ts` — `describe('saved conditions')`(302행) 안에 `it(...)` 하나 추가. 그 블록이 쓰는 헬퍼 `table(...)`·`settingsFor(...)`·`SAMPLE_COMPARE_OLD/NEW`(정의: `packages/tools/src/samples.ts:34`, 헤더는 `자산번호,자산명,사용 부서,상태,도입일` — **중복 이름이 없다**, 그래서 기존 단언은 새 검사에 걸리지 않는다) 를 그대로 쓰고, 새 파일 쪽에 같은 헤더 이름을 한 번 더 붙인 CSV 문자열을 만들어 넣는다.

- 검증 명령 (이 워크트리에서 실제로 도는 것):
  1) `npm ci` — 워크트리에는 node_modules 가 없다. 빼면 `vitest: not found`.
  2) `npx vitest run compare.test.ts` — 고치기 전 1 failed 를 먼저 눈으로 확인하고, 고친 뒤 전부 통과.
  3) `npm run check` — eslint + 4개 패키지 typecheck + vitest 전체(직전 회차 기준 10 파일 122 passed, 이번에 테스트 1개 늘어 123이 되어야 한다. 숫자는 그대로 믿지 말고 출력대로 적을 것).
  4) 선택: `npm run build` 까지. `npm run e2e`·`tools`·`ui`·`a11y` 는 **돌리지 말 것** — Docker PostgreSQL·Playwright·가짜 서버가 필요하고, 이 워크트리는 공유 dev DB(:55447)를 집어 e2e 7개가 환경 탓으로 깨진다(2026-10-14 회차에서 확인). 이 과제는 단위 시험만으로 끝난다.

- 위험과 피할 것:
  - **`packages/tools/src/table/csv.ts` 는 이번 회차에 열지 말 것.** 열린 PR #2·#3·#4 가 모두 `parseCsv` 의 `end()`/`push()` 같은 줄을 각각 고쳐 머지 충돌이 예정돼 있다. 이 워크트리의 `csv.ts:89~100` 에는 그 수정들이 아직 **없다**(행 한도 검사가 여전히 내용 분기 안, `cells` 누산기도 그대로 — 직접 확인했다). 같은 곳을 또 고치면 네 번째 갈래가 된다.
  - `.github/workflows/**` 는 한 줄도 건드리지 말 것 — CI `check` 잡의 2~3초 실패는 저장소 밖(GitHub Actions 결제·지출 한도 차단)이고, 10/14 세 회차가 이미 이 길에서 각각 한 회차를 썼다. 로컬에서 `npm run check` 로 green 을 증명하는 것이 이번 회차의 관문이다.
  - `packages/core/src/auth/*`·`domain/payments.ts`·`db/` 마이그레이션·`packages/core/vendor/postgres/**` 는 이 과제와 무관하다 — 열지 말 것(vendor 사본은 `vendor-postgres.test.ts` 가 upstream 과 줄 단위로 비교한다).
  - 같은 종류의 파서를 통합하려 하지 말 것: 열 이름을 찾는 코드가 `preset.ts` 외에 `CompareTool.tsx`(headers 로 keyMap 을 만든다)에도 있지만, **이번에는 `preset.ts` 한 곳만** 좁게 고친다.
  - 메시지를 영어로 쓰지 말 것. 커밋 메시지는 한국어 한 줄 `파일 대조: …` 꼴.
  - 수용 기준 2)가 기존 테스트를 깨뜨리는 것으로 드러나면(예: 샘플 파일에 이미 중복 헤더가 있어 비교 열 쪽이 막힌다면) **비교 열은 `notes` 로 내리고 기준 열만 `problems` 로** 남겨라. 기준 열이 어긋나면 행 짝짓기 자체가 틀어지므로 거기가 막아야 할 최소선이다.

- 차선 후보: **`sanitizePreset` 이 범위를 벗어난 헤더 줄 번호를 그대로 통과시킨다** (가치 2 / 위험 1 / 작업량 S) — `packages/tools/src/project/sanitize.ts:136~137` 이 `Number.isInteger(o.oldHeaderLine) ? ... : 1` 로만 걸러서 `0`·`-5`·`1e9` 같은 값이 저장 파일에서 그대로 살아 들어온다. 같은 파일의 다른 필드들은 모두 길이·후보군으로 좁히고 있는데(`str(…, 200)`, `oneOf(…)`) 이 둘만 비어 있다. `Math.min(Math.max(1, n), LIMITS.rows)` 꼴로 좁히고 `project/sanitize` 를 다루는 테스트를 새로 만들면 된다(현재 `packages/tools/src/__tests__/` 에 sanitize 전용 테스트가 **없다** — 테스트 공백 보강도 같이 된다). 1순위보다 사용자 영향이 작아 2순위.

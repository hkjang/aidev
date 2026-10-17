- 과제: 대사 범위 파서 `splitSheetRange` 가 `'Bob''s Data'!A1:B2` 의 이중 작은따옴표를 풀지 않는다 — 저장소의 나머지 파서 세 곳과 어긋난다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `web/src/lib/compareLists.ts:130` 의 `splitSheetRange` 는 바깥 따옴표만 `.replace(/^'(.*)'$/,'$1')` 로 벗기고 안쪽 `''` 이스케이프를 풀지 않아, 작은따옴표가 든 시트 이름이 `Bob''s Data` 로 나오고 `EditorPage.parseSheetRange`(web/src/pages/EditorPage.tsx:962-968)가 `sheets.find(...name.toLowerCase()===name.toLowerCase())` 로 실제 시트를 못 찾아 "시트!A1:C100 꼴로 적어 주세요" 로 거절한다. 이 저장소는 이미 `''`→`'` 를 확정된 계약으로 쓰고 있다 — 쓰는 쪽 `internal/formula/reference.go:58,161`(`strings.ReplaceAll(name,"'","''")`), 읽는 쪽 `internal/formula/structure.go:180-183`(`quotedQualifier`), `internal/formula/evaluator.go:250-252`, `internal/formula/reference.go:116-117` 이 모두 같은 규칙이고 `internal/formula/reference_test.go:31` 이 `'Bob''s Data'!A1` 을 픽스처로 쓴다. 즉 수식 바가 사용자에게 가르치는 바로 그 표기를 대사 범위 입력만 거절한다.

- 수용 기준:
  1) `splitSheetRange("'Bob''s Data'!A1:B2")` 가 `sheetName === "Bob's Data"` 를 낸다(현재는 `Bob''s Data`).
  2) 바깥 따옴표가 **있을 때만** 푼다 — `splitSheetRange("A''B!A1:B2")?.sheetName` 는 그대로 `A''B` 다(따옴표 없는 이름은 Go `quotedQualifier` 와 마찬가지로 이스케이프 해석을 하지 않는다).
  3) `web/src/lib/compareLists.test.ts:70-89` 의 기존 `splitSheetRange` 6사례(`장부!A1:C100`, `1분기!실적!A1:B2`, `'상반기 장부'!A1:B2`, `A1:B2`, `장부!`/`장부!A1`/``/`!A1:B2` → undefined)가 **한 줄도 바뀌지 않고** 그대로 통과한다.
  4) 새 테스트가 ①②를 못 박고, 경계인 `splitSheetRange("''!A1:B2")`(빈 이름 → undefined)와 `splitSheetRange("'O''Brien'!A1:B2")` 도 함께 고정한다.

- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `web/src/lib/compareLists.ts:splitSheetRange` — 지금의 한 줄 `const sheetName=text.slice(0,split).trim().replace(/^'(.*)'$/,'$1')` 을, 바깥 따옴표가 벗겨진 경우에만 `''`→`'` 를 치환하도록 두 단계로 나눈다. 예: `const raw=text.slice(0,split).trim()`; `const quoted=/^'(.*)'$/.exec(raw)`; `const sheetName=quoted?quoted[1].replaceAll("''","'"):raw`. 바로 위 주석(129행)에 "안쪽 `''` 는 작은따옴표 한 개다 — `internal/formula/structure.go` 의 `quotedQualifier` 와 같은 규칙" 한 줄을 한국어로 덧붙인다.
  - `web/src/lib/compareLists.test.ts` — 기존 `describe('splitSheetRange',…)` 블록 안에 `it` 을 더한다. 기존 `it` 을 고치지 말고 추가만 할 것.

- 검증 명령 (이 체크아웃에 `web/node_modules` 가 **없다** — 확인함):
  - `cd web && npm ci` (몇 분 걸린다)
  - `cd web && npx vitest run compareLists` — 고치기 전 새 테스트가 빨강인 것을 먼저 보고, 고친 뒤 초록.
  - `cd web && npm run lint` (= `tsc -b`), `cd web && npm test`, `cd web && npm run build`
  - 되돌리기 확인: 고친 한 자리만 원래 `.replace(/^'(.*)'$/,'$1')` 로 되돌려 **정확히 새 테스트만** 다시 빨강인지 볼 것.
  - Go 는 이번 변경과 무관하지만 관례대로: `go build ./...`, `go vet ./...`, `gofmt -l ./cmd ./internal ./pkg`, `./scripts/check-release-docs.sh`, `./scripts/check-commit-identities.sh HEAD`

- 위험과 피할 것:
  - **Go 파서와 통합하지 말 것.** 운영자가 되풀이해 말한 규칙이다 — 같은 종류의 파서가 여러 경로에 있으면 좁히는 방향으로만 고친다. `internal/formula/*.go` 는 한 글자도 건드리지 않는다. 이번 변경은 TS 한 함수를 Go 계약에 맞추는 것이지 공통 파서를 만드는 것이 아니다.
  - `compareKey` / `looksLikeIdentifier` / `significantDigits` / `textToNumber` / `parseFilterRange` / `compareLists` 는 손대지 말 것 — 보류 아이디어에 `0x10` 식별자 합침과 boolean/TRUE 키 대소문자가 남아 있지만 **별개 과제**이고 계약 결정이 먼저다.
  - `EditorPage.tsx` 는 읽기만 하고 고치지 말 것. 특히 `openCompare` 의 프리필 `` `${other.name}!A1:…` ``(986행 부근)은 시트 이름을 따옴표로 감싸지 않는데, 따옴표 없는 경로는 지금도 바르게 동작하므로 이번 범위가 아니다. 건드리면 파일 수가 늘고 수용 기준 3)의 "기존 동작 불변" 증명이 흐려진다.
  - `replaceAll` 은 ES2021 이다. `tsconfig` 타깃이 낮아 `npm run lint`(tsc -b)가 걸리면 `.split("''").join("'")` 로 바꿀 것 — **미확인**(tsconfig 의 lib/target 을 열어 보지 않았다).
  - 보호 경로(auth·apikey·handoff·migrations·.github/workflows)는 닿지 않는다.
  - docs/PDF 재생성 없음.

- 차선 후보: `web/node_modules` 설치가 끝내 실패하면 1순위는 증명할 수 없다. 그때는 **관리자 가이드 `docs/ADMIN_GUIDE.md:785` 의 외부 호출 오류 코드 안내를 실제 코드와 맞추기**(가치 2 / 위험 1 / 작업량 S, 스물두 회차 연속 보류)로 간다 — 다만 먼저 `internal/external/fetcher.go` 에서 각 실패 갈래가 실제로 내는 코드를 읽어 확인할 것. 2026-10-05 회차가 읽기 실패와 UTF-8 거절은 `#VALUE!` 임을 코드로 확인했으므로, 어긋난 자리는 `#N/A` 를 쓰는 fetch·허용목록 정책 오류 쪽으로 좁혀진다. 가이드를 고치면 PDF 재생성이 동반된다(USER_GUIDE 가 아니라 ADMIN_GUIDE 하나만).

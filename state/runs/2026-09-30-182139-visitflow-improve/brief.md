# 과제서 (2026-09-30, base main@9b66607 / v2.8.9)

- 과제: 반복 예약 「총 예약 횟수」 칸이 키 입력마다 값을 강제 보정해 10~19회를 입력할 수 없고 소수를 조용히 통과시킨다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/VisitFormPage.tsx:137` 의 입력칸 `onChange` 가 `setRepeatCount(Math.max(2, Math.min(52, Number(e.target.value))))` 로 **키 입력마다** 숫자 state 를 덮어써서, 칸을 비우면 `Number("") === 0` → `2` 로 되돌아가고(비울 수 없다) 첫 글자로 `1` 을 누르면 그 자리에서 `2` 가 되어 이어서 `0` 을 눌러도 `20` 이 된다 — 10~19회 반복 예약은 스피너 화살표를 여러 번 누르지 않고는 입력할 방법이 없다. 반대로 `Math.max`/`Math.min` 은 소수를 그대로 통과시켜 `2.5` 가 `submit()` 본문의 `occurrences` 로 전송되고 서버 `internal/app/visits.go:506-514` 의 `case float64: occurrences = int(value)` 가 **말없이 2회로 잘라** 사용자가 의도한 횟수와 다른 예약이 만들어진다. 입력칸이 지금처럼 사용자와 싸우지 않고, 잘못된 값은 이 저장소의 기존 관례대로 화면에서 한국어로 설명하고 제출을 잠그면 둘 다 없어진다.

- 수용 기준:
  1) 실제 dist 를 임베드한 실제 서버 + 실제 Chromium 에서 「매주 같은 시간으로 반복 예약」을 켜고 `총 예약 횟수` 칸을 전체 선택한 뒤 `1` → `0` 을 차례로 입력하면 칸에 `10` 이 남는다(수정 전 같은 조작은 `20` 이 되는 것을 먼저 재현할 것). `10` 으로 제출하면 지금처럼 방문이 등록된다.
  2) 칸을 비우면 빈 칸으로 남고(`2` 로 되돌아가지 않는다) 칸에 한국어 안내가 `error` 로 붙고 제출 버튼이 잠기며, 잠긴 동안 `/api/v1/visits` 요청이 0건이다(`disabled` 를 지운 변이 번들에서도 `submit()` 가드가 요청을 막고 같은 문구를 띄운다).
  3) `2.5` 를 입력하면 정수 안내가 뜨고 제출이 잠긴다 — 수정 전에는 제출이 통과해 서버가 조용히 2회로 만드는 것을 먼저 재현할 것(브라우저 네트워크에서 `occurrences: 2.5` 가 실제로 전송되고 응답이 201 인 것). `53`·`1` 도 범위 안내로 잠긴다.
  4) 방문자 수와 겹치는 상한은 기존 문구가 그대로 유지된다: 방문자 10명 × 51회에서 `방문자 10명이면 반복 예약은 최대 50회까지 가능합니다 (전체 방문 일정 500건 상한)`. 방문자 10명 × 50회는 통과한다(`web/src/visitors.test.ts:109` 의 기존 `recurrenceError` 테스트 전부 유지).
  5) 실패를 먼저 본 vitest 신규 테스트가 새 순수 함수의 경계를 고정한다: `""`, `"  "`, `"2.5"`, `"1"`, `"53"`, `"10"`, `"52"`, `"abc"`(number 입력이 `""` 를 주는 경로와 별개로 함수 계약으로). 기존 60개는 계속 통과한다.

- 건드릴 파일 (프로덕션 2개):
  - `web/src/visitors.ts` — 새 순수 함수 `repeatCountError(raw: string): string` 와 상수 `minRecurringOccurrences = 2` / `maxRecurringOccurrences = 52` 추가. 계약: `raw.trim()` 이 `""` 면 입력 요청 문구, `Number.isInteger(Number(text))` 가 false 면 정수 문구(NaN·`2.5` 모두 여기), 범위 밖이면 `2~52` 문구, 그 외 `""`. 기존 `recurrenceError(visitorCount, occurrences)` 는 **그대로 두고** 시그니처를 바꾸지 말 것 — 서버 `visits.go:516` 의 `occurrences*len(Visitors) > 500` 경계를 지키는 함수이고 테스트가 이미 고정하고 있다.
  - `web/src/pages/VisitFormPage.tsx` —
    - `:40` `useState(2)` → 문자열 state(`useState("2")`)로 바꾼다.
    - `:99` 를 `repeatCountError(raw) || recurrenceError(visitors.length, Number(raw.trim()))` 한 값(예: `recurrenceMessage`)으로 만들어, `repeatWeekly && !walkIn` 일 때만 계산하는 지금 조건을 유지한다. `recurrenceError` 에는 `repeatCountError` 가 `""` 일 때의 정수만 들어가게 순서를 지킬 것.
    - `:137` `onChange` 는 `setRepeatCount(e.target.value)` 로만 두고(보정 없음), `error`/`helperText` 는 그 한 값을 읽는다. `type="number"`, `slotProps={{ htmlInput: { min: 2, max: 52 } }}`, 기본 helperText `현재 일정을 포함해 최대 52회`, label `총 예약 횟수` 는 유지.
    - `:112` 의 `recurrence: ... { frequency: "weekly", occurrences: repeatCount }` 는 문자열이 아니라 파싱한 정수를 보내야 한다(`occurrences: Number(...)`). 같은 값을 읽는 네 경로 — 칸의 `error`/`helperText`, 제출 버튼 `disabled`(`:145` 의 `recurrenceMessage !== ""` 항), `submit()` 가드(`:108`), 전송 본문 — 가 **한 파싱 결과만** 보게 배선할 것(이 저장소의 `scheduleMessage`·`blockReason` 과 같은 방식).
  - 선택: `docs/USER_GUIDE.md` 의 방문 신청 반복 예약 문장 한 줄(PDF 재생성 불필요). 미머지 `origin/auto/2026-09-29-1232` 가 같은 파일의 3.7/3.8 을 건드리므로 그 절은 피하고, 자신 없으면 문서는 손대지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run lint && npm test && npm run build` (node_modules 없음 — `npm ci` 필요. `lint` 는 `tsc -b --pretty false`)
  - 서버는 건드리지 않지만 회귀 확인: `go build ./... && go vet ./... && gofmt -l . && git diff --check`
  - 브라우저 확인(수용 기준 1~4): `npm run build` 산출물을 `cmd/visitflow/webdist` 에 넣어 실제 서버를 띄우고 실제 Chromium(`/usr/bin/google-chrome`)으로 확인. 끝나면 `cmd/visitflow/webdist/index.html` 스텁을 **임시 복사본으로** 되돌릴 것(`git checkout --` 금지 — 과거에 미커밋 테스트를 잃었다).
  - DB 통합까지 돌릴 경우: `VISITFLOW_TEST_DSN='postgres://visitflow:visitflow@127.0.0.1:5432/visitflow?sslmode=disable' go test ./... -count=1` (internal/app 약 55초, CREATE DATABASE 권한 필요).

- 위험과 피할 것:
  - **서버를 고치지 말 것.** `internal/app/visits.go` 의 `occurrences < 2 || > 52 || occurrences*len(Visitors) > 500` 과 `int(float64)` 절단은 이번 과제의 기준선이다. 화면이 먼저 막는 것으로 끝낸다(`int()` 절단 자체를 서버에서 거절하도록 바꾸는 것은 별도 과제).
  - `scheduleError`·`submitBlockReason`·`visitorCountError`·`visitorsError`·`importVisitors`·`loadReference` 배선은 건드리지 말 것. 특히 `submitBlockReason` 은 v2.8.9 에서 한 줄 안내 정책으로 고정됐다.
  - `web/src/pages/ScannerPage.tsx`·`LobbyPage.tsx`·`web/e2e/visit-flow.spec.ts` 는 미머지 `origin/auto/2026-09-29-1232` 의 변경 대상이다 — 손대지 말 것.
  - 손대지 않은 빈 칸을 빨갛게 칠하지 않는 기존 정책과 충돌하지 않는지 확인할 것: 이 칸은 `repeatWeekly` 를 켠 사용자가 명시적으로 연 칸이고 기본값 `"2"` 가 이미 들어 있으므로, 사용자가 직접 비웠을 때만 안내가 뜬다(첫 렌더에서는 `""` 다).
  - 문자열 state 로 바꾼 뒤 `repeatCount` 를 숫자로 쓰는 자리가 남지 않았는지 확인할 것 — 지금 사용처는 `:40`, `:99`, `:112`, `:137` 네 곳뿐이다(grep 으로 확인했다).
  - 미확인: 「`1` 을 누르면 2 가 되고 이어서 `0` 을 누르면 20 이 된다」는 위 코드에서 따라 나오는 추론이며 **이번 정찰에서 브라우저로 재현하지 않았다**(node_modules 없음, 정찰은 코드 변경 금지). 구현자는 수정 전 번들로 이 증상을 먼저 실제 Chromium 에서 재현하고, 커서 위치 때문에 결과가 `20` 이 아니라 다른 값이면 재현한 실제 값을 기준으로 수용 기준 1 을 고쳐 쓸 것. 「비울 수 없다」와 「`2.5` 가 그대로 전송돼 서버가 2로 자른다」는 코드에서 직접 읽히는 부분이다.

- 차선 후보: `web/src/pages/KeysPage.tsx:57-58` 의 `Promise.all([...]).then(...)` 에 `.catch` 가 없어 API 키·정책 로드 실패가 unhandled rejection 으로 사라지고 빈 목록만 남는다 — v2.8.8 의 `loadReference`/`refError`/`refLoading` + 닫기 없는 Alert + 다시 불러오기 버튼 선례를 그대로 적용(파일 1개). 단, 같은 성격의 수정이 `origin/auto/2026-09-29-1232`(Scanner·Lobby)로 이미 미머지 상태라 중복처럼 보일 수 있다는 점을 감안할 것.

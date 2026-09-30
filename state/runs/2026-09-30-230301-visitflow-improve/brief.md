- 과제: 한 방문 안에 같은 휴대전화를 두 번 넣으면 나는 이유 없는 500 을 화면 사전 검사 + 서버 400 으로 닫기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/app/visits.go:682 upsertVisitor` 는 `SELECT id FROM visitors WHERE phone_hash=$1` 로 **휴대전화 해시 하나만** 보고 방문자를 찾는다. 그래서 한 신청에 같은 번호를 두 번 넣으면 두 번째 방문자가 첫 번째와 같은 `visitor_id` 를 받고, 바로 뒤 `INSERT INTO visitor_visits`(visits.go:574)가 `0005_baseline.sql:198` 의 `UNIQUE(visit_id, visitor_id)` 에 걸린다. 이 pgx 오류는 `visitError` 가 아니라서 `writeVisitError`(visits.go:406-413)의 fallback 으로 떨어져 **500 `visit_failed` "방문 요청을 처리하지 못했습니다"** 만 나간다 — 어느 방문자가 문제인지도, 무엇을 고쳐야 하는지도 알 수 없고 다시 눌러도 항상 실패한다(지난 회차 2026-09-30 실측: 동행 9명에게 같은 번호 → 반복 예약 여부와 무관하게 500, 번호를 모두 다르게 하면 10명 201). 화면이 먼저 번호를 붙여 알려 주고 서버가 400 으로 답하면, 이 저장소가 이름·휴대전화·인원수·반복 횟수에 이미 적용한 "같은 경계를 화면과 서버가 같은 값으로 본다" 방식이 마지막 남은 구멍까지 덮는다.

- 수용 기준:
  1) 방문 신청 화면에서 방문자 2의 휴대전화를 방문자 1과 같게 입력하면(정규화 후 같은 숫자열이면 `010-1234-5678` 과 `01012345678` 도 같은 것으로 본다) 그 **휴대전화 칸**에 `error` + 한국어 helperText 가 붙고, 제출 버튼이 잠기며, 잠긴 이유가 기존 안내 자리에 한 줄로 보인다. 번호를 고치면 안내가 사라지고 등록이 201 로 성공한다.
  2) 잠긴 상태에서 `POST /api/v1/visits` 요청이 **0건** 이다. `disabled` 항만 지운 변이 번들에서도 `submit()` 가드가 요청을 막고 같은 문구를 띄운다(브라우저로 확인).
  3) 서버가 같은 입력에 **400** 과 새 코드(`duplicate_visitor` 권장)로 답한다. 500 도 `visit_failed` 도 더 이상 나오지 않는다. 화면을 거치지 않는 경로(MCP `internal/app/mcp.go:166`, 현장 등록 `visits.go:1289`)도 같은 400 을 받는다 — 세 진입점이 모두 `createVisitRecord` 하나를 지나므로 검사는 한 곳에만 둔다.
  4) Go 통합 테스트가 "같은 번호 2명 → 400 + `duplicate_visitor`" 와 "번호가 다른 2명 → 201" 을 함께 고정한다. 서버 검사를 지운 변이에서 그 테스트가 실제로 실패하는 것(500 이 돌아오는 것)을 확인할 것.
  5) vitest 가 새 순수 함수를 고정한다: 정규화가 다른 같은 번호(하이픈/공백)는 중복, 7자리 미만이라 아직 `visitorFieldErrors.phone` 이 잡는 값은 중복으로 **이중 표시하지 않음**, 아무것도 입력하지 않은 빈 칸은 조용함(이 저장소의 "첫 화면을 빨갛게 칠하지 않는다" 정책 유지). 기존 69개는 그대로 통과.

- 건드릴 파일 (프로덕션 3개 + 테스트 2개):
  - `internal/app/visits.go:451-470` — `createVisitRecord` 의 기존 방문자 검증 루프(`for _, visitor := range in.Visitors`, 이름·전화·동의·회사명·watchlist 를 보는 그 루프). 인덱스를 받도록 `for index, visitor := range` 로 바꾸고 `seen := map[string]int{}` 로 `normalizePhone(visitor.Phone)` 를 추적해, 이미 본 번호면 `visitError{400, "duplicate_visitor", "방문자 N 과 방문자 M 의 휴대전화가 같습니다. 방문자마다 다른 번호를 입력하세요"}` 를 돌려준다. 이 루프는 `tx` 시작(visits.go:521) **이전**이므로 트랜잭션·QR·consent 경로를 건드리지 않는다. `upsertVisitor` 와 스키마는 손대지 말 것.
  - `web/src/visitors.ts` — 파일 맨 아래 `visitorsError` 옆에 추가. 서버와 같은 정규화를 쓰는 기존 `phoneDigits`(파일 12행, `value.replace(/[^0-9]/g, "")` — 서버 `normalizePhone`(visits.go:25) 과 같은 규칙)을 **재사용**하고 새로 만들지 말 것. 두 함수를 제안한다: 인덱스 정렬 배열을 돌려주는 `duplicatePhoneErrors(visitors: VisitorCheck[]): string[]`(칸에 붙일 문구) 와, 그중 첫 항목에 `방문자 N: ` 을 붙여 한 줄로 돌려주는 `duplicatePhoneError(visitors): string`(버튼·가드용). 뒤엣것은 반드시 앞엣것을 호출해 **한 번만 계산**한다 — 이 저장소의 `visitorFieldErrors`/`visitorsError` 쌍과 같은 구조다. 기존 `visitorFieldErrors`·`visitorsError`·`recurrenceError` 의 시그니처는 바꾸지 말 것(각각 기존 테스트가 고정하고 있다).
  - `web/src/pages/VisitFormPage.tsx` — 세 자리만 배선한다. (a) 96행 근처에 `const duplicateMessages = duplicatePhoneErrors(visitors);` 와 `const duplicateMessage = duplicatePhoneError(visitors);` 를 두고, (b) 148행 휴대전화 `TextField` 의 `error={fieldErrors.phone !== "" || duplicateMessages[index] !== ""}` / `helperText={fieldErrors.phone || duplicateMessages[index] || undefined}`, (c) 151행 제출 버튼 `disabled` 식에 `|| duplicateMessage !== ""` 를 더하고 109행 `submit()` 가드에 `if (duplicateMessage) { setError(duplicateMessage); return; }` 를 `visitorMessage` 계열과 같은 자리에 넣는다. 13행 import 에 두 이름을 추가한다. 잠긴 이유 한 줄 안내는 기존 `blockReason` Alert 을 재사용해도 되고 별도로 두어도 되지만, **문구의 출처는 위 두 함수 하나뿐**이어야 한다.
  - `web/src/visitors.test.ts` — 수용 기준 5.
  - `internal/app/integration_test.go` — 수용 기준 4. `TestVisitorCountAndRecurrenceLimits`(864행 부근)의 `env.json(...)`/`env.do(...)` + `visitBody(siteID, map[string]any{"visitors": ...})` 패턴을 그대로 쓰면 된다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run lint && npm test && npm run build` — `npm test` 는 vitest, 현재 69개. `lint` 는 `tsc -b`. 이 정찰 세션에서는 `node_modules` 가 없어 프런트를 실행하지 않았다(미확인).
  - `VISITFLOW_TEST_DSN='postgres://visitflow:visitflow@127.0.0.1:5432/visitflow?sslmode=disable' go test ./... -count=1` — **DSN 없이 돌리면 DB 통합이 통째로 SKIP 되어 위 수용 기준 4 를 전혀 검증하지 못한다.** CREATE DATABASE 권한 필요, 과거 기록상 `internal/app` 55~60초. 이 환경에서 5432 는 다른 프로젝트 컨테이너가 쓰고 있어 전용 도커 네트워크 + 다른 포트로 띄워야 했다는 기록이 있다.
  - `go build ./... && go vet ./... && gofmt -l . && git diff --check`
  - 브라우저 확인(수용 기준 1·2): `npm run build` 결과를 `cmd/visitflow/webdist/` 에 임베드해 실제 서버를 띄우고 실제 Chromium(`/usr/bin/google-chrome`, Playwright `channel: "chrome"`)으로 확인한다. **끝나고 `webdist/index.html` 스텁을 임시 복사본으로 되돌릴 것 — `git checkout --` 은 쓰지 말 것**(미커밋 테스트를 잃은 기록이 있다).

- 위험과 피할 것:
  - **`upsertVisitor` 의 조회 키를 바꾸지 말 것.** "이름+전화로 찾자" 는 재방문 방문자 식별·watchlist·개인정보 마스킹(`masked_at`)·`frequent_visitors` 까지 의미가 번지는 데이터 변경이고, 이번 과제(S)가 아니다. 스키마·마이그레이션도 건드리지 않는다.
  - `normalizePhone` 은 watchlist 전화 해시(`s.keys.Digest("phone:"+...)`, visits.go:462)의 입력이다. **읽기만 하고 수정하지 말 것.**
  - 화면과 서버 두 경로가 같은 입력을 같은 값으로 읽는지가 이 과제의 핵심이다. 하이픈·공백·국가번호가 섞인 값을 두 쪽이 같게 보는지 브라우저에서 end-to-end 로 확인하고(화면이 막는 값과 서버가 400 을 주는 값이 일치해야 한다), 정규화 규칙을 프런트에 새로 쓰지 말고 기존 `phoneDigits` 를 쓸 것.
  - 미머지 브랜치 `origin/auto/2026-09-16-1212`(메일)·`2026-09-18-0533`(MCP OAuth)·`2026-09-21-0654`(import 중복 헤더)·`2026-09-29-1232`(Scanner/Lobby) 와 겹치는 파일은 피한다. 이번 세 파일은 겹치지 않는다(`mcp.go` 는 읽기만 하고 수정 대상 아님).
  - 보호 경로(auth/session/OIDC·keys·migrations·settings·release 워크플로)는 건드리지 않는다.
  - 문자열 grep 을 증거로 제출하지 말 것. 손으로 만든 가짜 방문자 객체가 아니라 실제 라우터·실제 PostgreSQL·실제 번들로 검증한다.
  - `TestSelfRegistrationRecordsVisitorConsent` 는 `consented_at DESC LIMIT 1` 타이브레이크 때문에 드물게 흔들린 기록이 있다. 실패하면 이번 변경 탓으로 단정하기 전에 단독 재실행할 것.

- 차선 후보: **`KeysPage.tsx:57-58` 의 `Promise.all(...).then(...)` 에 `.catch` 가 없어 API 키·정책 로드 실패가 unhandled rejection 으로 사라진다** — v2.8.8/v2.8.10 의 `loadReference`/`refError`/`refLoading` + 닫기 없는 Alert + `다시 불러오기` 버튼 선례를 그대로 옮기면 프로덕션 파일 1개로 끝난다(가치 2 / 위험 1 / S). 단, `origin/auto/2026-09-29-1232`(Scanner·Lobby 의 같은 수정)가 아직 미머지라 같은 성격의 변경이 둘 겹칠 수 있으니 diff 를 먼저 확인할 것.

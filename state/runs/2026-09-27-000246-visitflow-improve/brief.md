# 과제서 (정찰, 2026-09-28)

- 과제: 방문 신청 화면에서 기준 정보(reference-data) 로드 실패를 복구 가능하게 만들기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/VisitFormPage.tsx:45` 의 `useEffect(..., [])` 는 `api<ReferenceData>("/api/v1/reference-data")` 가 실패하면 `.catch((e) => setError(e.message))` 만 하고 끝난다 — `ref` 는 `null` 로 남고 재시도 수단이 없다. 그 결과 사업장·로비·부서·방문 유형 select 가 전부 빈 채로 렌더되고 `siteId` 가 `""` 이라 제출 버튼은 영구히 `disabled` 인데, 상단 오류 Alert 은 `onClose={() => setError("")}` 로 닫히므로 한 번 닫으면 **이유도 복구 수단도 없는 빈 양식**만 남는다(전체 새로고침 외 탈출구 없음). 일시적 네트워크 오류·세션 갱신 중 5xx 하나로 신청 자체가 막히는데 "다시 불러오기" 버튼 하나면 복구된다.

- 수용 기준:
  1) `/api/v1/reference-data` 가 실패하면 화면에 실패 사실과 함께 **"다시 불러오기"** 동작(버튼)이 남고, 그 오류 표시는 사용자가 닫아서 없앨 수 있는 상단 `error` Alert 과 별개여야 한다(닫아도 복구 수단이 사라지지 않을 것).
  2) 다시 불러오기를 눌러 이번에는 성공하면 사업장·로비·부서·방문 유형 select 가 정상적으로 채워지고, 최초 로드와 똑같이 첫 번째 허용 사업장이 `siteId` 로 자동 선택되며(현행 `x.sites.find((site) => siteScope.length === 0 || siteScope.includes(site.id))` 동작 유지), 그대로 방문 등록까지 성공한다.
  3) 로드 중에는 다시 불러오기 버튼이 중복 요청을 내지 않도록 잠기고(진행 중 표시), 실패 상태에서 사용자가 이미 입력해 둔 값(방문 목적·방문자 이름/휴대전화 등)은 재시도 성공 후에도 지워지지 않는다.
  4) 성공 경로는 기존과 동일하다 — 정상 로드 시 새 안내·버튼이 보이지 않아야 한다.
  5) 증명: 실제 서버 + 실제 `npm run build` 번들 + 실제 Chromium 에서 **수정 전 증상을 먼저 재현**(reference-data 가 5xx → Alert 닫기 → 빈 양식·잠긴 제출 버튼·복구 불가)하고, 수정 후 같은 조건에서 재시도로 복구되는 것을 확인할 것.

- 건드릴 파일 (프로덕션 2개 이내):
  - `web/src/pages/VisitFormPage.tsx` — `VisitFormPage`: (a) 45행의 reference-data `useEffect` 본문을 `loadReference` 같은 이름 붙은 함수로 빼고 `refError`(또는 `refLoading`) state 를 추가, (b) 실패 시 `<Card>` 안 또는 `PageHeader` 바로 아래에 닫히지 않는 `Alert severity="error"` + `Button`(다시 불러오기) 표시, (c) 재시도 중 버튼 `disabled`. `submit()`·`importVisitors()`·`scheduleError`/`visitorsError`/`visitorCountError`/`recurrenceError` 배선은 건드리지 말 것.
  - 참고 선례(새 관례를 만들지 말고 이것을 따를 것): `web/src/pages/AdminPage.tsx:68` 이 이미 `const load = async () => { try { … } catch (e) { setError(…) } }; useEffect(() => { void load(); }, [])` 형태로 이름 붙은 로더를 쓴다. 같은 모양으로 빼고 재시도 버튼이 `void load()` 를 부르게 하면 된다.
  - (선택) `docs/USER_GUIDE.md` — 기준 정보 로드 실패 시 다시 불러오기로 복구한다는 한 문장. PDF 재생성은 하지 않아도 된다(최근 회차들도 미재생성).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run lint && npm test && npm run build`  (`lint`=`tsc -b`, `test`=`vitest run`, 현재 단위 테스트 47개 통과. **`web/node_modules` 는 워크트리에 없으므로 `npm ci` 부터 필요**)
  - `go build ./... && go vet ./... && gofmt -l .`  (서버는 건드리지 않지만 워크트리가 깨끗한지 확인)
  - `git diff --check`
  - 화면 확인(필수, 최근 4회차가 쓴 방식): `web/dist` → `cmd/visitflow/webdist` 복사 → `go build` → PostgreSQL + 서버 기동 → `playwright-core` 의 Chromium(`executablePath=/usr/bin/google-chrome`). reference-data 실패는 프록시/네트워크 차단 또는 응답 가로채기로 만들 것. **끝나고 `cmd/visitflow/webdist/index.html` 스텁을 반드시 되돌릴 것**(빌드 산출물 커밋 사고 방지).
  - 참고: `go test ./... -count=1` 은 `VISITFLOW_TEST_DSN` 이 없으면 PostgreSQL 통합이 SKIP 된다 — PASS 를 통합 증거로 쓰지 말 것. 이번 과제는 서버를 바꾸지 않으므로 Go 테스트 추가는 불필요하다.

- 위험과 피할 것:
  - **`internal/app/visits.go` 와 `createVisitRecord` 는 절대 건드리지 말 것.** REST·현장·MCP 공용 계약이다. 이번 과제는 순수 프런트 복구 UX 다.
  - **vitest 는 `src/**/*.test.ts` 만 수집한다 — `.tsx` 테스트는 조용히 0개가 된다.** 이 과제의 핵심은 React state/effect 라 순수 `.ts` 모듈로 뽑기 어렵다. 억지로 순수 함수를 만들지 말고 **브라우저 확인을 1급 증거로 쓰라**. (손으로 만든 fetch 대역으로 결함을 증명하지 말 것 — 운영자 규칙.)
  - `ref` 가 `null` 인 동안 이미 `ref?.…` 로 전부 옵셔널 접근하고 있으므로 렌더 크래시 위험은 낮다. `sites`·`siteLobbies` 파생값과 `useEffect([siteLobbies, lobbyId])` 의 로비 자동 선택이 재시도 성공 뒤에도 한 번 도는지 확인할 것.
  - 기존 `error` state 를 재사용해 오류를 표시하면 수용 기준 1)을 못 지킨다(닫으면 복구 수단이 사라짐) — **별도 state 로 둘 것**.
  - 되돌리기에 `git checkout -- <파일>` 을 쓰지 말 것(이 저장소에서 미커밋 테스트를 두 번 잃었다). 임시 복사본이나 `sed` 를 쓸 것.
  - 미머지 브랜치 3개(`origin/auto/2026-09-16-1212` 메일, `origin/auto/2026-09-18-0533` MCP OAuth, `origin/auto/2026-09-21-0654` 가져오기)와 파일이 겹치지 않는다 — `VisitFormPage.tsx` 는 이들 중 어느 것도 건드리지 않는다.

- 차선 후보: **제출 버튼이 말없이 잠기는 이유를 안내** — `VisitFormPage.tsx:127` 의 `disabled` 식은 `!checklistSatisfied`(보안서약·안전교육 체크), `!declarationsSatisfied`(유형이 요구하는 차량번호·반입장비), `visitors.some((x) => !x.name || !x.phone || !x.consent)`(개인정보 동의 해제)를 포함하는데, `submit()` 가드에도 이 셋은 없고 화면 어디에도 "왜 잠겼는지" 문구가 없다. 특히 개인정보 동의 체크를 끄면 아무 안내 없이 제출이 막힌다(서버 `visits.go:455` 는 `!visitor.Consent` 를 `invalid_visitor` 로 거절한다). 최근 4회차가 성립시킨 관례 그대로 순수 모듈(`web/src/visitors.ts` 에 `submitBlockReason` 추가)로 만들고 vitest `.ts` 로 검증할 수 있어 테스트 증거가 강하다 — 1순위가 성립하지 않으면 이것을 고를 것. (가치 2 / 위험 1 / 작업량 S)

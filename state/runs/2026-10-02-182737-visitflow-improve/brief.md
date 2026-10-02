- 과제: 방문 목록에서 늦게 도착한 이전 질의 응답이 목록을 오염시키는 문제 닫기 (가치 3 / 위험 1 / 작업량 M)

- 왜: `web/src/pages/VisitsPage.tsx` 의 `load`(35행)와 `loadMore`(43행)는 순번도 취소도 없이 응답이 도착하는 **순서대로** state 를 덮는다. 특히 `loadMore` 는 `setItems((current) => [...current, ...data.items])` 로 **덧붙이므로**, 「더 보기」가 비행 중일 때 검색창을 건드리면(검색창 `onChange` → `q` → `buildQuery` → `load` 신원 변경 → `useEffect([load])` 재조회) 새 질의의 목록 **아래에 이전 질의의 행이 그대로 붙고**, 같은 응답이 `setNextCursor(옛 커서)` 까지 해서 다음 「더 보기」가 이전 질의의 다음 페이지를 계속 이어 붙인다. 화면에 남는 오염이고(로비 현황의 깜빡임과 달리 사라지지 않는다) 사용자는 검색 결과가 아닌 행을 보고 조치하게 된다.

- 수용 기준:
  1) **수정 전 재현**: `/visits` 에서 `cursor=` 가 들어간 요청만 지연시킨 뒤(Playwright `page.route` 로 해당 요청만 `await new Promise(r => setTimeout(r, 3000))` 후 `route.continue()`; 나머지는 실서버 응답 그대로) 「더 보기」를 누르고 곧바로 검색창에 질의를 입력하면, 지연 응답 도착 시 **새 질의 결과 아래에 이전 질의 행이 붙는** 것을 행 수와 행 내용으로 관찰한다. 이어서 「더 보기」를 한 번 더 누르면 이전 질의의 커서가 쓰인 것도 요청 URL 로 관찰한다.
  2) **수정 후 같은 조작**: 지연 응답이 버려진다 — 목록은 새 질의 결과만 남고 행 수가 늘지 않으며, 다음 「더 보기」 요청 URL 의 `cursor` 가 새 질의의 `nextCursor` 다.
  3) **최신 질의 우선**: 검색어를 빠르게 바꿔 조회 두 개가 겹칠 때(먼저 보낸 것을 더 늦게 응답하도록 지연) 화면에 남는 것은 **마지막으로 보낸** 질의의 결과다. 검색창 값과 목록이 어긋나지 않는다.
  4) **변이 확인(인과)**: 가드 비교문만 제거한 번들을 같은 실서버에 임베드해 돌리면 1) 의 증상이 그대로 재발한다. (되돌리기는 임시 복사본으로; `git checkout --` 쓰지 말 것)
  5) **회귀 보존**: 기간·상태 필터(`useSearchParams`), Enter 검색, 취소/반복 취소/방문자 취소/QR 재발급 뒤의 `await load()` 갱신, 상세 다이얼로그, `loading`/`loadingMore` 스피너와 「더 보기」 버튼 `disabled` 가 모두 전과 같이 동작한다. 늦은(버려진) 응답이 **살아 있는 스피너를 미리 끄지 않는다**.

- 건드릴 파일:
  - `web/src/pages/VisitsPage.tsx` — 컴포넌트 안에 요청 티켓 `useRef<number>(0)` 하나를 두고, `load`·`loadMore` 가 **같은** 카운터를 쓰게 한다. 각 함수 진입에서 `const ticket = ++seq.current`, 응답 뒤 `if (seq.current !== ticket) return;` 로 `setItems`/`setNextCursor`/`setError` 커밋을 막고, `finally` 의 `setLoading(false)`/`setLoadingMore(false)` 도 같은 비교로 감싼다(그래야 늦은 응답이 새 요청의 스피너를 끄지 않는다). `load` 와 `loadMore` 가 카운터를 공유하므로 새 `load` 가 비행 중 `loadMore` 를 자동으로 무효화한다 — 이것이 1) 을 닫는 핵심이다.
  - 그 외 파일 없음(프로덕션 1개). 가이드 문구가 필요하면 `docs/USER_GUIDE.md` 한 줄까지만, PDF 재생성은 하지 말 것.

- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` — `lint` 는 `tsc -b`, `test` 는 `vitest run`. 이번 정찰에서 확인: `web/vite.config.ts:10` 의 `include: ["src/**/*.test.ts"]` 라 **`.tsx` 단위 테스트는 조용히 0개로 수집된다**. `web/package.json` 에 testing-library 계열 의존성이 없으므로 컴포넌트 단위 테스트는 애초에 불가능하다 → 이 결함의 1급 증거는 **실제 서버 + 실제 Chromium** 이다(이 저장소의 2026-09-27·28·29·30 선례와 같다).
  - 브라우저 검증: `npm run build` 결과를 `cmd/visitflow/webdist` 에 임베드 → Go 서버 기동 → Playwright `channel: "chrome"`(`/usr/bin/google-chrome`; 번들 chromium 없음). 끝난 뒤 `cmd/visitflow/webdist/index.html` 스텁을 **임시 복사본으로** 되돌리고 `git status` 가 의도한 파일만 보이는지 확인.
  - `go build ./...`, `go vet ./...`, `gofmt -l .`(빈 출력), `git diff --check` — 서버는 안 건드리지만 CI test 잡과 맞추기 위해.
  - 서버 변경이 없으므로 `VISITFLOW_TEST_DSN` 지정 `go test ./... -count=1` 은 선택(과거 기록상 internal/app 약 55~60초).

- 위험과 피할 것:
  - **`web/src/api.ts` 의 `api()` 시그니처를 바꾸지 말 것.** AbortController 를 공용 fetch 헬퍼에 밀어 넣으면 18개 페이지가 전부 영향권에 들어간다. 티켓은 `VisitsPage` 안의 `useRef` 로만 둔다(프로덕션 파일 1개 유지).
  - **디바운스를 넣지 말 것.** 지금 검색은 키 입력마다 `limit=100` 조회를 보내고(`onChange` → `q` → `buildQuery` → `load` 재생성 → `useEffect([load])`) `onKeyDown` 의 Enter 핸들러는 사실상 중복이다. 이것은 별도 과제(아래 아이디어 파일 `visits-search-debounce`)이고, 함께 고치면 수용 기준 3 의 재현 조건 자체가 바뀌어 증명이 흐려진다.
  - `period`/`status` 의 `useSearchParams` 배선, 키셋 페이징 전제("newly created visits never shift a later page" 주석)를 건드리지 말 것.
  - `setItems((current) => [...current, ...])` 의 함수형 갱신 형태를 유지할 것 — 티켓 가드는 그 **앞**에서 return 하는 방식으로.
  - 보호 경로(auth/session/OIDC, migrations, internal/app 서버, 릴리즈·Docker 워크플로)는 이번 과제에서 전혀 건드릴 이유가 없다.
  - **미확인**: 지연 주입 없이 실사용 타이밍만으로 1) 이 재현되는지는 확인하지 않았다(코드 경로로만 확정). 재현은 `page.route` 지연으로 하고, 그것이 실제 응답 역전과 같은 경로임을 수정 전/후 짝 실행으로 보일 것.
  - **미확인**: 늦은 `setError` 를 버리는 것이 사용자에게 더 나은지(네트워크 실패를 삼키게 되는지)는 판단이 갈릴 수 있다. 버리는 쪽을 권하되, 버린다면 새 요청이 곧 자기 결과나 자기 오류를 쓴다는 점을 코드 주석 한 줄로 남길 것.

- 차선 후보: **로비 현황에서 늦은 이전 조회 응답 무시** (`web/src/pages/LobbyPage.tsx:30` 의 `load` — `useCallback([tab, query, lobbyFilter])` 이고 모든 완료가 무조건 `setData` 한다). 같은 패턴·프로덕션 1개 파일이고, 검색창 키 입력과 탭 전환이 겹칠 때 카운트 카드와 목록이 옛 결과로 되돌아간다. 다만 덧붙이기가 없어 오염이 **일시적**이라 1순위보다 증상이 약하다. 고른다면 `useEffect(() => { const stream = new EventSource(...) }, [])` 의 한 번만 구독, `loadRef`, 빈 `lobbyId` 계약, 기준 정보 복구 Alert(v2.8.8 선례)을 그대로 보존할 것.

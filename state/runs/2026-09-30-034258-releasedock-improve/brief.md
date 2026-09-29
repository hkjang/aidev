- 과제: 다른 릴리즈로 이동할 때 이전 릴리즈의 실시간 로그가 화면에 남는 결함 수정 (가치 3 / 위험 1 / 작업량 S)
- 왜: `useReleaseLogs`(web/src/pages/releases/ReleaseDetailPage.tsx:62-64)는 `releaseId` 가 바뀌면 `cursor.current = 0` 만 되돌리고 `logs` 상태는 그대로 두는데, `useAsync` 가 새 릴리즈를 불러오는 동안 이전 `data` 를 유지하므로 LogPanel 이 언마운트되지 않고 `releaseId` prop 만 갈아치워진다. 결과적으로 릴리즈 A 의 배포 로그가 릴리즈 B 상세 화면에 남고 B 의 새 줄이 그 아래에 이어 붙어, 승인·감사 도구에서 어느 릴리즈의 출력인지 구분할 수 없는 화면이 만들어진다(복사 버튼도 두 릴리즈를 섞어 내보낸다).
- 수용 기준:
  1) 릴리즈 A 상세의 "실시간 로그" 탭에서 줄을 받은 뒤 `/releases/release-2` 로 이동하면, 이전 릴리즈의 줄이 화면에서 사라지고 빈 로그 안내("실행 로그가 도착하면…")로 돌아간다.
  2) 서버 30분 상한 재연결(`end {"reason":"max_duration"}` → `streamAttempt` 증가)에서는 기존 줄이 **그대로 유지**된다. 즉 초기화는 `releaseId`/`enabled` 변경에만 걸리고 `streamAttempt` 재연결에는 걸리지 않는다.
  3) 기존 스트림 테스트 6개 묶음(연결 1개 유지, 커서 전진, 지우기, 탭 이탈·언마운트)이 그대로 통과한다 — 특히 `'resumes timeout streams with server IDs, retaining lines and one connection across renders'` 가 계속 초록이어야 기준 2)가 증명된다.
  4) (같이 넣으면 좋음) 릴리즈가 바뀐 직후에는 `connected` 가 false 로 떨어져 "로그 연결 대기" 가 표시되고, 새 스트림의 `open` 이벤트에서 다시 "실시간 로그 연결됨" 이 된다. 이 항목을 넣을 때는 반드시 테스트로 고정할 것(넣지 않아도 1~3만 만족하면 채택 가능).
- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `web/src/pages/releases/ReleaseDetailPage.tsx:62-64` — `useReleaseLogs` 안의 리셋 effect(현재 `cursor.current = 0` 한 줄, 의존성 `[releaseId, enabled]`)에 표시 로그 초기화를 추가. **스트림 effect(66-116행, 의존성 `[releaseId, enabled, streamAttempt]`)에는 넣지 말 것** — 거기 넣으면 max_duration 재연결마다 화면이 비워져 v0.5.24 의 수정을 되돌린다. `sequence.current` 는 초기화하지 말 것(단조 증가라야 React key 가 충돌하지 않는다 — 2026-09-28 회차에서 같은 이유로 `Date.now()+sequence` 를 ref 로 바꿨다).
  - `web/src/pages/releases/ReleaseDetailPage.stream.test.tsx:145-156` — 기존 `it('does not carry the cursor to another release route')` 가 이미 `line(57)` 을 띄운 뒤 `/releases/release-2` 로 pushState+popstate 로 이동한다. 이동 직후에 `expect(screen.queryByText('server line 57')).not.toBeInTheDocument()` 를 추가하면 **지금은 실패하고 수정 후 통과**한다(수용 기준 1). 기준 2 는 70-84행 테스트가 이미 `for (const id of [41, 57, 58]) expect(screen.getByText(...)).toBeVisible()` 로 고정하고 있으니 새로 만들 필요 없다. 테스트 유틸(`TestEventSource`, `emit`, `reconnected`, `renderPage`)은 그대로 재사용하고 새 대역을 만들지 말 것 — 이 파일은 실제 `App` 라우트와 프로덕션 컴포넌트를 지난다.
- 검증 명령 (모두 `web/` 디렉터리에서 — 루트에는 `npm test` 스크립트가 없다. **이 워크트리에는 `web/node_modules` 가 없다**(정찰에서 `node_modules/.bin/vitest` 부재 확인) → 먼저 `cd web && npm ci`. 정찰은 이 때문에 웹 스위트를 실제로 돌려 보지 못했다: 아래 빨강/초록은 구현자가 직접 확인해야 하는 예측이다):
  - `cd web && npx vitest run src/pages/releases/ReleaseDetailPage.stream.test.tsx` — 수정 전 새 assertion 1건 실패(빨강)를 먼저 확인한 뒤 구현.
  - `cd web && npm test -- --run` — 전체(직전 기준 122건)가 통과해야 한다.
  - `cd web && npx tsc -b --noEmit`
  - 반증 실험: 추가한 초기화를 스트림 effect(`[releaseId, enabled, streamAttempt]`)로 옮겨 보면 70행 테스트(`retaining lines…`)가 실패해야 한다. 이것을 확인하고 원복할 것.
  - 백엔드는 이 과제와 무관하다. 그래도 회차 관례상 `cd backend && go test ./... -count=1` 을 한 번 돌리되, `TEST_POSTGRES_DSN` 이 없으면 DB 테스트는 SKIP 되므로 그것을 통합 검증 성공으로 보고하지 말 것.
- 위험과 피할 것:
  - 서버·SSE 계약(`backend/internal/server/releases.go:streamReleaseLogs`, `Last-Event-ID` 대 `?after` 중 큰 값 선택)은 건드리지 말 것. 이번 변경은 프런트 표시 상태만이다.
  - 단순 모드(`web/src/pages/simple/SimpleRunDetailPage.tsx`, `SimpleDeployPage.tsx`)를 같이 고치지 말 것 — 같은 결함이 없다. `SimpleRunDetailPage` 의 `loadStoredLogs` 는 의존성이 `[id]` 이고 성공 시 `setLogs(stored.lines)` 로 **전체를 교체**하므로 런이 바뀌면 이전 런의 줄이 남지 않는다(207-211행에서 확인). 전체 모드만 저장 로그 재수집 경로가 없어 append 만 하는 구조다.
  - `LogPanel` 의 `enabled` prop 은 JSX 에서 항상 리터럴 `true` 이고 탭을 벗어나면 컴포넌트가 언마운트된다(`tab === 1 && …`). 따라서 `enabled` 변경 경로는 실제로 돌지 않는다 — 그것을 근거로 쓴 테스트는 만들지 말 것.
  - 자동 스크롤 effect(`[logs, autoScroll]`)와 `clear` 버튼 동작은 그대로 둘 것.
  - `web/dist` 를 만들었다면 커밋 전에 지울 것. `VERSION` 은 건드리지 말 것.
- 차선 후보: 로그 스트림 한도(사용자당 3 / 전역 64) 거절과 해제의 실제 HTTP 통합 테스트 — `backend/internal/server/server.go` 의 `acquireLogStream`/`releaseLogStream` 을 `newSimpleStreamFixture`(프로덕션 `New()` 기반) 위에서 네 번째 스트림이 429 `stream_limit` 을 받고 하나를 닫으면 다시 열리는지 고정한다. 동시 스트림의 비동기 해제 동기화가 남은 위험이며 실제 PostgreSQL(`TEST_POSTGRES_DSN`)이 필요하다.

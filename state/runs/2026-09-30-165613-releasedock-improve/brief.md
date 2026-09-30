- 과제: 전체 모드 릴리즈 실시간 로그의 화면 상한 초과(오래된 줄 버림)를 화면과 복사 결과에 알리기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `useReleaseLogs` 의 `setLogs((current) => [...current.slice(-4998), …])`(web/src/pages/releases/ReleaseDetailPage.tsx:97)가 오래된 줄을 **조용히** 버리는데, 전체 모드에는 저장 로그 조회도 내려받기도 없다 — `api` 에 있는 릴리즈 로그 관련 함수는 `releaseLogStreamUrl` 하나뿐(client.ts:470)이라 버려진 줄은 UI 어디에서도 되찾을 수 없다. 그런데도 복사 버튼은 `aria-label="전체 로그 복사"`(ReleaseDetailPage.tsx:303)로 "전체"를 약속하므로, 긴 배포의 실패를 조사하려 복사한 운영자는 앞부분이 빠진 로그를 전체 로그로 착각한다. 단순 모드 실행 상세에는 이미 `logTruncated` + `LOG_TRUNCATED_NOTICE`(SimpleRunDetailPage.tsx:50,189,297,484)로 같은 개념이 있으니, 전체 모드에도 같은 계약을 주되 **버려지는 쪽이 반대(단순 모드는 뒤쪽, 전체 모드는 앞쪽)** 임을 문구에 반영한다.

- 수용 기준:
  1) 실시간 로그 탭에서 표시 상한을 넘는 줄이 도착하면 로그 영역 위에 `severity="warning"` Alert 이 나타나 **오래된 줄이 빠졌다**는 사실과 현재 표시 줄 수를 알린다. 상한 이하에서는 Alert 이 없다.
  2) 잘린 상태에서 복사 버튼을 누르면 `navigator.clipboard.writeText` 에 넘어가는 문자열에 잘림 안내 문장이 포함되고, 그 안내가 **본문 앞**에 온다(버려진 쪽이 앞이므로 뒤에 붙이면 거짓말이 된다). 잘리지 않은 상태의 복사 문자열은 지금과 **한 글자도 다르지 않다**.
  3) `end {"reason":"max_duration"}` 재연결 뒤에도 잘림 표시가 유지된다. 즉 잘림 상태 초기화는 리셋 effect(`[releaseId, enabled]`, ReleaseDetailPage.tsx:62-67)에만 들어가고 스트림 effect(`[releaseId, enabled, streamAttempt]`)에는 들어가지 않는다. — v0.5.24 가 고친 결함(재연결이 표시 상태를 날림)의 재발 방지.
  4) 다른 릴리즈로 이동, 탭 이탈 후 재진입, `지우기` 버튼 뒤에는 잘림 표시가 사라진다(표시 버퍼가 비었으므로 상한 초과 안내는 더 이상 현재 버퍼를 설명하지 않는다).
  5) 테스트는 (a) 상한 **경계**에서 정확히 한 줄 넘칠 때 처음 표시가 켜지고 그 뒤로 꺼지지 않음을 증명하고, (b) 실제 `App` 라우트 렌더로 Alert 문구와 클립보드 문자열이 사용자에게 실제로 닿음을 증명한다. `logs.length` 가 아니라 "버림이 일어났는가" 로 판정해야 한다(상한에 정확히 닿기만 한 경우는 잘림이 아니다).

- 건드릴 파일 (프로덕션 1 + 테스트 1~2):
  - `web/src/pages/releases/ReleaseDetailPage.tsx`
    - 모듈 상수 두 개를 export 로 추가: `LOG_DISPLAY_LIMIT`(현재 동작과 같은 4999 — `slice(-4998)` + 새 줄 1), `RELEASE_LOG_TRUNCATED_NOTICE`(한국어 한 문장, 존재하지 않는 "내려받기" 를 안내하지 말 것).
    - `useReleaseLogs`: `logs` 상태를 `{ lines: LogEntry[]; truncated: boolean }` **한 객체**로 합치고, 순수 export 헬퍼 `appendReleaseLogLine(state, entry, limit = LOG_DISPLAY_LIMIT)` 가 자르기와 `truncated` 를 **한 번에** 계산하게 한다(`truncated: state.truncated || 버림이 일어남`). 별도 ref 로 개수를 세는 방식은 피할 것 — updater 안에서 다른 setState 를 부르는 것도, 개수를 두 곳에서 세는 것도 어긋난다.
    - 리셋 effect(62-67)에 잘림 초기화를 더한다. 스트림 effect 에는 넣지 않는다.
    - 훅 반환에 `truncated` 를 더하고 `clear` 는 lines 와 truncated 를 함께 되돌린다.
    - `LogPanel`: `truncated` 일 때 로그 Box 위에 warning Alert, `copy` 는 `truncated ? `${NOTICE}\n${text}` : text`. `aria-label="전체 로그 복사"` 는 **그대로 둔다**(기존 스냅샷/접근성 이름 변경은 이번 범위 밖).
  - `web/src/pages/releases/ReleaseDetailPage.stream.test.tsx`
    - 헬퍼 경계 테스트(작은 `limit` 인자로 `appendReleaseLogLine` 직접 호출 — 손으로 만든 대역이 아니라 프로덕션 함수 그 자체. 저장소 선례: `streamDisconnected`·`streamEndedRun`).
    - 실제 렌더 테스트: 기존 `renderPage()` / `TestEventSource` / `emit` 을 재사용. **주의** — 기존 `emit` 은 프레임마다 `await act()` 라 `LOG_DISPLAY_LIMIT + 1` 회 반복하면 매우 느리다. 프레임 전체를 **하나의 `act()`** 안에서 dispatch 하는 대량 emit 헬퍼를 새로 만들어 React 가 한 번의 커밋으로 합치게 하라. 그래도 느리면(대략 30초 초과) 시간을 측정해 보고하고 렌더 테스트를 조용히 빼지는 말 것.
    - jsdom 에 `navigator.clipboard` 가 없다. 이 저장소에는 클립보드 스텁 선례가 없고 `src/test/setup.ts` 에도 없으므로(확인함), 새 테스트 안에서 `Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText: vi.fn() } })` 로 세우고 끝나면 원복하라. `vi.stubGlobal('navigator', …)` 는 navigator 전체를 갈아치우니 쓰지 말 것(파일의 `vi.stubGlobal('EventSource', …)` 와 혼동 금지). `vi.restoreAllMocks()` 는 defineProperty 를 되돌리지 않는다.

- 검증 명령 (web 디렉터리에서 — 루트에는 test 스크립트가 없다):
  - `cd web && npm ci` — **node_modules 가 없다**(이번 정찰에서 `web/node_modules/.bin/vitest` 부재 확인). 이것 없이는 아래가 전부 127 로 죽는다.
  - `cd web && npm test -- --run` (package.json 의 `test` 는 `vitest run`) — 기준선은 122건 통과(이전 회차 기록; 이번 정찰에서는 미실행 = 미확인).
  - `cd web && npx tsc -b --noEmit`
  - 백엔드는 이 과제와 무관하다. 무관함을 보이려 `cd backend && go test ./... -count=1` 을 돌린다면 `TEST_POSTGRES_DSN` 이 없어 DB 테스트가 SKIP 되므로 **통합 검증으로 보고하지 말 것**.
  - `web/dist` 를 만들었다면 커밋 전에 지우고, `VERSION` 은 건드리지 말 것.

- 위험과 피할 것:
  - **잘림 초기화를 스트림 effect 에 넣지 말 것.** `streamAttempt` 가 의존성에 있어 max_duration 재연결마다 초기화되고, v0.5.24 가 고친 결함을 그대로 되살린다(v0.5.25 회차의 반증 실험이 이미 그 방향을 잡아냈다).
  - `sequence.current` 는 화면용 React key 라 단조 증가로 남겨야 한다(리셋하지 말 것 — 재연결 시 key 충돌). 이것을 잘림 판정 카운터로 재사용하지 말 것: 릴리즈 이동에도 리셋되지 않으므로 "이 릴리즈의 버퍼가 넘쳤는가" 를 답할 수 없다.
  - `cursor.current` 와 `advanceCursor` 는 손대지 말 것. v0.5.23 의 `backend/internal/server/simple_stream_cursor_test.go` 가 서버 쪽 커서 계약을 고정하고 있고, 이번 과제는 서버·백엔드를 전혀 건드리지 않는다.
  - 잘리지 않은 경우의 복사 문자열 형식(`${timestamp} ${level} ${message}` 조합, ReleaseDetailPage.tsx:293)을 바꾸지 말 것 — 수용 기준 2 의 절반이다.
  - 안내 문구에 "내려받기" 를 쓰지 말 것. 단순 모드 문구를 복사하면 존재하지 않는 기능을 안내하게 된다(전체 모드 로그 API 는 스트림 하나뿐임을 확인함).
  - 보호 경로(auth/OIDC/RBAC, `backend/internal/store/migrations/`, `.github/workflows/`)는 이번 과제와 무관하니 열지 말 것.
  - 상한 값 4999 는 이번 정찰에서 `slice(-4998)` 로부터 **계산한 값**이다(실행으로 확인한 것은 아님). 구현자는 헬퍼 테스트로 실제 경계를 고정하고, 현재 동작과 표시 줄 수가 달라지지 않는지 확인하라.

- 차선 후보: 단순 모드 배포 화면(`SimpleDeployPage.tsx:232,234`)도 같은 `slice(-4998)` 를 조용히 쓰고 잘림 표시가 없다 — 1순위가 성립하지 않으면(예: 대량 emit 렌더 테스트가 현실적으로 너무 느리면) 같은 패턴을 그 화면에 적용하라. 그 다음은 전체 모드 `source.onerror`(ReleaseDetailPage.tsx:114)가 `readyState` 를 보지 않아 CONNECTING(브라우저 자동 재시도)과 CLOSED(포기)를 구분하지 못하는 문제로, 단순 모드의 export 된 `streamDisconnected` 헬퍼를 재사용할 수 있다.

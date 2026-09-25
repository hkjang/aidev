# 과제서 (2026-09-26 정찰, base main@d539386 v0.5.20)

- 과제: 단순 모드 실행 상세의 라이브 로그 연결이 끊긴 것을 화면에 알리고 다시 시도할 수 있게 하기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `web/src/pages/simple/SimpleRunDetailPage.tsx:215-242` 의 스트림 effect 는 `log`·`end` 만 듣고 `error` 는 듣지 않는다. 서버가 429 `stream_limit`(`backend/internal/server/simple.go:1165`, 한도는 `server.go:121 acquireLogStream` 의 사용자당 3·전역 64) 로 거절하거나 세션 만료·프록시 절단으로 연결이 죽으면 브라우저는 스펙대로 `readyState=CLOSED` 로 두고 `error` 만 한 번 발생시키며 재연결하지 않는다 — `end` 프레임이 없으므로 effect 의 재연결 경로(`streamAttempt`)도 돌지 않고, 실행 상태도 `end` 때만 다시 읽으므로 화면은 "진행 중" 인 채 로그가 영구히 멈춘 것처럼 보인다. 같은 앱의 전체 모드(`web/src/pages/releases/ReleaseDetailPage.tsx:62-90`)는 이미 `onerror`/`onopen` 으로 연결 상태를 표시하므로, 단순 모드만 끊김을 숨기고 있다.

- 수용 기준:
  1) 스트림이 끊기고 `readyState` 가 CLOSED(2) 면 로그 영역에 안내(예: "실시간 로그 연결이 끊겼습니다")와 "다시 연결" 조작이 보인다. 기존 `logError`/`logTruncated` Alert 슬롯(`SimpleRunDetailPage.tsx:423-428`) 옆에 같은 방식으로 넣는다.
  2) 브라우저가 스스로 재연결하는 중(`readyState` 가 CONNECTING(0))에 발생한 `error` 로는 안내를 띄우지 않는다. 안내가 떠 있다가 스트림이 다시 열리면(`open`) 안내가 사라진다.
  3) "다시 연결" 을 누르면 저장된 로그를 다시 수집한 뒤(끊긴 동안 쌓인 줄이 보이도록) 새 EventSource 가 한 개만 열리고, URL 이 마지막으로 든 줄의 id 에서 이어진다(`?after=<lastId>`). 자동 무한 재시도는 넣지 않는다(429 를 되레 때린다).
  4) 테스트가 세 가지를 구분해 증명한다: CLOSED 오류 → 안내 표시, CONNECTING 오류 → 안내 없음, 다시 연결 → 스트림 1개·`after=<lastId>`.

- 건드릴 파일:
  - `web/src/pages/simple/SimpleRunDetailPage.tsx` — 스트림 effect 에 `source.addEventListener('error', ...)` 와 `'open'` 추가, 끊김 상태 `useState` 와 Alert 렌더, "다시 연결" 핸들러(`loadStoredLogs` 재호출 + `setStreamAttempt` 증가 + 끊김 상태 해제). 판정은 순수 헬퍼로 빼서 내보내면(`export function streamDisconnected(readyState: number): boolean`) 단위로도 고정할 수 있다.
  - `web/src/pages/simple/SimpleRunDetailPage.stream.test.tsx` — 기존 `TestEventSource` 에 `readyState`(0/1/2, `close()` 는 2)와 `emitError(readyState)`·`emitOpen()` 를 더하고 테스트 3건 추가. 이 파일은 실제 라우트로 `App` 을 렌더해 프로덕션 페이지·effect·useAsync 를 그대로 지나며 transport 만 대역이다 — 그 구조를 유지할 것.

- 검증 명령 (이 저장소에서 실제로 도는 것. `web/node_modules` 가 없으므로 `npm ci` 가 선행되어야 한다):
  - `cd web && npm ci`
  - `cd web && npm test -- --run` (전체. 직전 회차 기준 94~98건)
  - `cd web && npx vitest run src/pages/simple/SimpleRunDetailPage.stream.test.tsx` (좁혀 볼 때)
  - `cd web && npx tsc -b --noEmit` (타입 검사는 `npm test` 에 포함되지 않는다)
  - 백엔드를 건드리지 않았음을 보이려면 `cd backend && go test ./...` 한 번 (TEST_POSTGRES_DSN 없으면 통합 테스트는 Skip 되고 `make test` 가 WARN 을 낸다)

- 위험과 피할 것:
  - **`source.onerror = fn` / `source.onmessage = fn` 로 쓰지 말고 반드시 `addEventListener('error'|'open', ...)` 를 쓸 것.** 테스트 대역 `TestEventSource` 는 `EventTarget` 상속이라 `dispatchEvent` 가 `onerror` 프로퍼티를 호출하지 않는다 — 프로퍼티로 쓰면 테스트가 그 경로를 전혀 지나지 않는다(전체 모드 페이지가 `onerror =` 를 쓰는 것을 그대로 베끼지 말 것).
  - `EventSource.CLOSED` 같은 정적 상수를 읽지 말 것. 테스트에서 전역 `EventSource` 가 대역으로 교체되어 `undefined` 가 된다. 스펙 숫자(0/1/2)를 이름 붙인 모듈 상수로 둘 것.
  - `appendStreamedLine`·`streamEndedRun`·`lastIdRef` 갱신·`end` 처리·자동 스크롤·effect 의 의존성 배열(특히 `run` 객체를 다시 넣지 말 것 — 1d3e27c 가 줄마다 재연결되던 것을 고친 자리다)은 건드리지 말 것.
  - 자동 재시도 루프·폴링 추가 금지(서버 스트림 예산을 태운다). 재시도는 사용자 조작으로만.
  - `web/src/pages/simple/SimpleDeployPage.tsx:205` 와 `web/src/pages/releases/ReleaseDetailPage.tsx:62` 도 같은 종류의 빈틈이 있으나 **이번 범위 밖**이다(배포 화면의 스트림은 업로드 직후에만 살아 있어 렌더 테스트로 도달하는 비용이 크고, 전체 모드는 이미 `connected` 표시가 있다). 범위를 넓히지 말고, 남는 빈틈은 ideas.json 에 항목으로 남겨 둘 것.
  - 보호 경로(`backend/internal/server/auth.go`·`oidc.go`·`rbac_policy.go`, `store/migrations/`, `.github/workflows/`), `VERSION`, `web/dist` 는 손대지 말 것.

- 차선 후보: 통합 테스트 픽스처를 `New()` 기반 Server 로 옮기기 (가치 2 / 위험 1 / S) — `backend/internal/server/simple_batch_test.go:71` 이 `&Server{store: st, log: ...}` 리터럴을 돌려주어 `New()` 가 채우는 `streams`/`aiActive` 맵이 nil 이다. 그 서버로 스트림 라우트를 타면 `acquireLogStream`(`server.go:121`) 이 nil map 대입으로 패닉하므로 통합 테스트가 프로덕션 배선을 지나지 못한다(프로덕션 `New()` 는 안전). 이 과제는 `TEST_POSTGRES_DSN` 이 필요하다(도커 PostgreSQL 16 을 띄우고 끝나면 지울 것).

- 미확인(이번 정찰에서 실행하지 못한 것): `npm ci`/vitest 를 이번 회차에 돌려 보지 않았다(worktree 에 `web/node_modules` 가 없음). 테스트 개수와 통과 여부는 직전 회차 기록에 근거한 추정이다. 브라우저에서 429 를 실제로 재현해 `readyState` 를 관찰한 것도 아니며, CLOSED/CONNECTING 구분은 EventSource 스펙(비-200 응답 → fail the connection, 네트워크 단절 → reestablish)에 근거한 판단이다.

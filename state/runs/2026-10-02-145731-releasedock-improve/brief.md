- 과제: 단순 모드 배포 화면의 라이브 로그가 오래된 줄을 조용히 버리는 것을 읽는 사람에게 알리기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `SimpleDeployPage.tsx:232,234` 의 `setLogs((current) => [...current.slice(-4998), …])` 는 상한(새 줄 1개 포함 4999줄)을 넘는 **가장 오래된** 줄을 아무 표시 없이 버린다. 그런데 이 화면의 `setLogs([])` 는 패키지마다가 아니라 **배치 시작 시 한 번만**(`:338`) 돌고 각 패키지 사이에는 `── 파일명 ──` 구분선만 끼워 넣으므로(`:361`), 상한은 한 런이 아니라 **업로드 배치 전체**에 걸린다 — 패키지 여러 개를 한 번에 올리는 평소 사용에서 가장 먼저 올린 패키지의 출력이 화면에서 사라지고, 사라졌다는 사실조차 알 수 없다. 이 화면에는 복사 버튼도, 실행 상세로 가는 링크도 없어서(`grep` 으로 `복사`/`clipboard`/`RouterLink` 모두 없음을 확인) 읽는 사람은 앞부분이 빠진 로그를 그 배치의 전체 출력으로 착각한 채 배포 성공/실패를 판단한다. 전체 모드는 v0.5.26 에서 같은 결함을 고쳤고(`appendReleaseLogLine` + 경고 Alert), 단순 모드 실행 상세도 이미 잘림 안내를 갖고 있다(`SimpleRunDetailPage.tsx:484`) — 배포 화면만 남았다.

- 수용 기준:
  1) 배치 로그가 상한을 넘겨 오래된 줄이 버려지면 `실행 로그` 카드의 로그 영역 **위에** `severity="warning"` Alert 이 뜨고, 그 문구가 **오래된(앞쪽) 줄이 버려졌다**는 것과 각 런의 전체 로그는 실행 기록(`/simple/runs` → 런 상세의 `로그 내려받기`)에서 받을 수 있다는 것을 말한다. (실행 상세의 `LOG_TRUNCATED_NOTICE` 문구를 **그대로 재사용하면 거짓말이다** — 그쪽은 `처음 N줄만 표시했습니다`, 즉 **뒤쪽**을 버리는 반대 방향이다. 문구는 새로 쓰고, 확인하지 않은 기능은 적지 말 것.)
  2) 상한에 **정확히 닿기만** 한 경우(4999줄)는 잘림이 아니다 — Alert 이 뜨지 않는다. 4999줄째를 넘기는 순간 뜬다.
  3) 잘림 플래그와 그 플래그가 설명하는 버퍼가 어긋날 수 없다 — 자르기와 플래그 계산을 한 순수 함수가 함께 수행하고(전체 모드의 `appendReleaseLogLine` 과 같은 모양), 그 함수를 export 해 경계를 작은 `limit` 으로 단위 테스트한다.
  4) 현재 상한 없이 추가하는 세 경로(`:361` 구분선, `:378`·`:387` 오류 줄)도 같은 함수를 지나 상한과 플래그 계산이 한 군데로 모인다. (지금은 이 세 곳이 `[...current, …]` 라 배열이 4999줄을 넘어 자랄 수 있다.)
  5) 잘림 표시는 **다음 배치가 시작될 때**(`:338` 의 `setLogs([])` 자리) 함께 사라지고, 스트림 재연결(`reconnectStream`, `max_duration` 재연결)에서는 **유지된다**. — v0.5.24/0.5.26 이 고정한 계약이며, 스트림 effect(의존성 `[activeRunId, streamAttempt]`, `:265`)에 초기화를 넣으면 재연결마다 안내가 사라진다.
  6) 테스트가 증명할 것: ① 실제 App 라우트로 렌더한 **프로덕션** 배포 화면에서 상한을 넘는 로그를 스트림으로 흘려 Alert 이 실제로 나타남 ② 상한에 닿기만 한 경우 나타나지 않음(순수 함수 단위 테스트의 작은 `limit` 으로) ③ 재연결 뒤 유지 / 새 배치에서 사라짐.

- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `web/src/pages/simple/SimpleDeployPage.tsx` — ① 새 순수 export 헬퍼(이름 예: `appendDeployLogLine(state, entry, limit = DEPLOY_LOG_DISPLAY_LIMIT)`)와 상한 상수(`4999`), 잘림 안내 문구 상수를 추가 ② 로그 상태를 `LogLine[]` 에서 `{ lines: LogLine[]; truncated: boolean }` 한 객체로 합치고 `:232,234,361,378,387` 다섯 군데의 추가를 모두 그 헬퍼로 바꿈 ③ `:568` 의 `Boolean(logs.length)` 조건과 `:601` 의 `logs.map` 을 새 모양에 맞춤 ④ `streamLost` Alert(`:576`) 아래/위에 잘림 Alert 추가 ⑤ `:338` 의 초기화를 빈 상태 상수로.
  - `web/src/pages/simple/SimpleDeployPage.stream.test.tsx` — 기존 7건(`:136`~`:216`)을 깨지 않고 새 테스트 추가. 기존 `TestEventSource`(`:12`, 실제 App 라우트 + 프로덕션 페이지)를 그대로 쓸 것.
  - **헬퍼를 `ReleaseDetailPage.tsx` 에서 import 하지 말 것.** 단순 모드는 `releases/` 를 import 하지 않는 경계를 지키고 있고(현재 import 는 `./SimpleRunDetailPage` 뿐), `LogLine`(로컬 타입, 표시용 `id`)과 전체 모드 `LogEntry` 는 타입이 다르다. 공통화하려 두 모듈을 묶지 말고 배포 화면 안에 두 번째 구현을 두는 것이 이번 범위다.

- 검증 명령 (이 저장소에서 실제로 도는 것. `web/node_modules` 가 **없으므로 `npm ci` 가 선행 필수** — 이번 정찰에서 확인했다):
  - `cd web && npm ci`
  - `cd web && npm test -- --run` (기준선은 v0.5.26 기록의 131건. 이번 정찰은 `npm ci` 를 돌리지 않아 기준선을 **실측하지 않았다** — 구현자가 먼저 기준선을 찍을 것)
  - `cd web && npx tsc -b --noEmit` (`npm test` 만으로는 타입 오류를 못 잡는다 — 과거 회차에서 실증됨)
  - `cd web && npm run build`
  - `git diff --check`
  - 백엔드 0 파일 변경이면 Go 테스트는 생략 가능. 돌릴 경우 `TEST_POSTGRES_DSN` 이 없으면 DB 테스트가 SKIP 되므로 종료 0 을 통합 검증 성공으로 보고하지 말 것.

- 위험과 피할 것:
  - **대량 emit 렌더 테스트 비용**: v0.5.26 이 실측한 바로 jsdom 에서 5000 프레임을 한 `act()` 로 흘리면 약 2초다. 수용 기준 1 의 렌더 테스트 1건만 그 비용을 쓰고, 경계(기준 2)는 작은 `limit` 을 넘긴 순수 함수 단위 테스트로 증명할 것. 렌더 테스트를 여러 건으로 늘리지 말 것.
  - **React key 함정**(v0.5.26 에서 실제로 걸린 자리): 행 번호를 `setLogs` updater **안에서** 읽으면 한 배치에 들어온 프레임이 모두 flush 시점의 같은 값을 읽어 key 가 겹친다. `nextLineId()`(`:186`)는 지금 updater **밖**에서 호출되고 있다 — 리팩터 중에 updater 안으로 끌어들이지 말 것.
  - **커서/재연결 계약을 건드리지 말 것**: `logCursorRef`(`:231` 전진, `:372` 패키지마다 0 으로 초기화), `streamEndedRun` 가드(`:255` 근처), `streamDisconnected`(`:251`), 의존성 배열 `[activeRunId, streamAttempt]`(`:265`). v0.5.22/0.5.24 가 테스트로 고정한 지점이고 기존 7건이 지키고 있다.
  - 보호 경로를 건드리지 않는다: `backend/internal/server/auth.go`·`oidc.go`·RBAC, `backend/internal/store/migrations/`, `.github/workflows/release.yml`. 이번 과제는 `web/` 두 파일 밖으로 나갈 이유가 없다.
  - `VERSION` 과 생성된 `web/dist` 를 커밋하지 말 것(저장소 관례). `npm run build` 뒤 `web/dist` 를 지울 것.
  - 문구에 확인되지 않은 기능을 적지 말 것. 확인된 것은 `api.simpleRunLogDownloadUrl`(`SimpleRunDetailPage.tsx:327`)과 라우트 `/simple/runs/:id`(`web/src/app/App.tsx:41`)·`/simple/runs`(`:40`)뿐이다. 배포 화면에서 그 경로로 가는 **링크를 새로 추가하는 것은 이번 범위 밖**이다(원하면 다음 회차 조각). 문구로만 안내할 것.

- 차선 후보: 로그 스트림 한도(사용자당 3 / 전역 64) 거절과 해제의 HTTP 통합 테스트 — `backend/internal/server/server.go` 의 `acquireLogStream`/`releaseLogStream` 을 `simple_stream_test.go:newSimpleStreamFixture`(프로덕션 `New()` 기반) 위에서 네 번째 스트림 `429 stream_limit` → 하나 닫으면 재개로 고정한다. `TEST_POSTGRES_DSN` 과 실제 PostgreSQL 16 이 필요하고, 동시 스트림의 비동기 해제 동기화가 남은 위험이다. 1순위가 이미 고쳐져 있거나 범위가 두 배로 불면 이쪽으로. (이번 정찰에서 `server.go` 의 해당 함수 본문은 **재확인하지 않았다 — 미확인**.)

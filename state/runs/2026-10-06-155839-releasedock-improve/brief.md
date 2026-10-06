- 과제: 전체 모드 릴리즈 실시간 로그에 저장 로그 내려받기 버튼 붙이기 (가치 3 / 위험 1 / 작업량 S)
- 왜: v0.5.30 이 `GET /api/v1/releases/{id}/logs`(JSON 커서 + `?format=text` 다운로드)를 백엔드에 넣었는데(`server.go:301` 에서 `s.withPermission("releases.read", s.listReleaseLogs)` 로 등록됨을 확인) 웹에는 그 엔드포인트를 가리키는 코드가 한 줄도 없다 — `web/src/api/client.ts:470` 에 `releaseLogStreamUrl` 만 있고 다운로드 URL 헬퍼가 없다. 그래서 v0.5.26 의 잘림 경고("오래된 로그 줄이 화면에서 빠졌습니다")와 v0.5.29 의 끊김 경고("아래 로그는 끊긴 시점까지입니다")가 빠진 부분이 있다고 알려 주면서도 되찾을 수단을 주지 못한다 — 서버에는 이미 있는데 화면에서 닿을 수 없다.

- 수용 기준:
  1) `api.releaseLogDownloadUrl(id)` 가 `/api/v1/releases/<encodeURIComponent(id)>/logs?format=text` 를 반환한다. 단순 모드의 `simpleRunLogDownloadUrl`(client.ts:585)과 **같은 모양**으로 쓴다(`encodeURIComponent` 포함).
  2) 릴리즈 상세의 실시간 로그 패널에 `로그 내려받기` 버튼이 보이고, 그 `href` 가 1)의 URL 이며 **표시 중인 줄이 0개여도 활성**이다 — 저장 로그는 화면 버퍼와 무관하고, 끊김·잘림이야말로 버퍼가 비거나 모자란 상황이다(`지우기`·`전체 로그 복사` 는 `disabled={!logs.length}` 지만 이 버튼은 그러면 안 된다).
  3) 잘림 경고와 끊김 경고의 문구가 그 버튼을 가리킨다("`로그 내려받기` 로 전체 로그를 받을 수 있습니다" 류). 테스트가 두 Alert 각각의 안내가 바뀐 것을 고정한다.
  4) `ReleaseDetailPage.tsx:55-58` 의 `LOG_DISPLAY_LIMIT` 주석("Full mode streams and nothing else - there is no stored-log query and no download")은 v0.5.30 으로 **거짓이 됐다**. 지금 사실로 고친다(주석만, 상수 4999 는 그대로).
  5) 테스트가 증명할 것: (a) 실시간 로그 탭에 내려받기 링크가 있고 그 href 가 릴리즈 id 를 담는다 (b) 로그가 한 줄도 없을 때도 링크가 살아 있다 (c) 잘림 Alert 과 끊김 Alert 이 각각 내려받기를 가리키는 문구를 담는다. 기존 24건은 전부 그대로 통과해야 한다.
  6) 선택(하면 좋음): `RELEASE_LOG_TRUNCATED_NOTICE`(ReleaseDetailPage.tsx:66)는 v0.5.26 이 "내려받기가 없으니 쓰지 않는다" 는 이유로 일부러 그 단어를 뺀 문구다. 이제 성립하므로 복사 안내에도 내려받기를 언급할 수 있다 — 단 이 상수를 바꾸면 클립보드 테스트가 문자열을 정확히 비교하므로 테스트도 같이 고쳐야 한다. 3)까지만 하고 6)은 건너뛰어도 수용한다.

- 건드릴 파일 (프로덕션 2개):
  - `web/src/api/client.ts` — `releaseLogStreamUrl`(:470) 바로 옆에 `releaseLogDownloadUrl` 추가. `simpleRunLogDownloadUrl`(:585)을 그대로 베낀다. API 는 쿠키 인증이다(`API_BASE = '/api/v1'`, `request` 가 `credentials: 'include'`, Authorization 헤더 없음 — :16, :82 확인) 그러므로 평범한 `<a href>` 다운로드가 동작한다. 단순 모드가 이미 같은 방식으로 돈다.
  - `web/src/pages/releases/ReleaseDetailPage.tsx` — `LogPanel`(:347) 안. 버튼은 `지우기` 옆 Stack(:366-370)에 `component="a" href={api.releaseLogDownloadUrl(releaseId)}` 로 넣는다(`SimpleRunDetailPage.tsx:325-332` 모양: `component="a"`, `startIcon={<DownloadRoundedIcon />}`, `variant="outlined"`). `LogPanel` 은 `releaseId` 를 prop 으로 이미 받고 `api` 는 이 파일에 이미 import 돼 있다(:36) — 새 import 는 `DownloadRoundedIcon` 하나뿐. 문구 변경은 `streamLost` Alert(:375-383)과 `truncated` Alert(:384-388), 주석은 :55-58.
  - `web/src/pages/releases/ReleaseDetailPage.stream.test.tsx` — 테스트 추가. 이 파일은 실제 `App` 라우트로 `/releases/release-1` 를 렌더하고 `TestEventSource` 대역으로 프레임을 넣는다(대역에 `readyState`·`emitError(readyState)` 가 **있다** — v0.5.29 가 추가했다. 프로필의 "릴리즈 대역에 readyState 없음" 은 낡은 기록이다). 잘림 재현은 기존 테스트가 쓰는 `LOG_DISPLAY_LIMIT` 기반 대량 emit 패턴을, 끊김 재현은 `emitError(2)` 를 그대로 쓸 것 — 새 대역을 만들지 말고 이 파일의 프로덕션 배선을 쓴다.
  - (선택) `web/src/api/client.test.ts` — 다른 URL 헬퍼 단위 테스트가 거기 있으면 1)을 그 관례대로 고정한다. 그 파일에 URL 헬퍼 테스트가 실제로 있는지는 **미확인**이다.

- 검증 명령 (전부 `web/` 안에서):
  - `cd web && npm ci` (node_modules 가 없을 때가 많다. 먼저 **고치기 전** 기준선을 실측할 것 — 프로필 기준 145건/13파일이지만 v0.5.30 은 백엔드만 바꿨으므로 그대로일 가능성이 크다. 실측치를 보고에 적는다.)
  - `npm test -- --run` (또는 좁혀서 `npm test -- --run src/pages/releases/ReleaseDetailPage.stream.test.tsx`)
  - `npx tsc -b --noEmit` — `npm test` 는 타입을 보지 않는다. 생략하면 CI 에서 깨진다.
  - `npm run build`
  - `git diff --check`
  - 백엔드는 0 파일 변경이므로 Go 테스트는 불필요하다. 돌렸다면 `TEST_POSTGRES_DSN` 이 없어 DB 테스트가 SKIP 되니 **SKIP 을 통과로 보고하지 말 것**.

- 위험과 피할 것:
  - **엔드포인트를 다시 만들지 말 것.** 백엔드는 이미 있고 이번 회차는 0 파일 변경이다. `releases.go`·`server.go` 를 건드리면 릴리즈 경로에 들어가고 방금 머지된 계약을 흔든다.
  - **`useReleaseLogs` 의 두 effect 를 건드리지 말 것** — 이 과제는 훅에 손댈 필요가 전혀 없다(버튼은 `releaseId` 만 쓴다). 리셋 effect(`[releaseId, enabled]`)와 스트림 effect(`[releaseId, enabled, streamAttempt]`)의 역할을 섞으면 v0.5.24/v0.5.26/v0.5.29 가 고친 결함이 재발하고 기존 테스트가 잡는다. 로그 effect 에 매 렌더 새 객체를 넣으면 재연결 폭주로 사용자당 3스트림 한도 429 가 난다.
  - **`disabled={!logs.length}` 를 내려받기 버튼에 복사해 붙이지 말 것** — 수용 기준 2 의 정반대다. 끊김·잘림 상황이 바로 버퍼를 믿을 수 없는 상황이다.
  - 권한 게이트는 **불필요하다**: `/releases/:id` 라우트 자체가 `RequirePermission permission="releases.read"`(App.tsx:45)이고 엔드포인트도 같은 `releases.read` 다(server.go:301). 단순 모드처럼 화면 간 권한이 갈리는 상황이 아니므로 v0.5.28 식 `canRead` 가드를 흉내내지 말 것.
  - 문구에 "실행 기록" 같은 단순 모드 표현을 쓰지 말 것 — 전체 모드에는 그 화면이 없다.
  - 보호 경로(auth/OIDC/RBAC, store/migrations, .github/workflows)는 전혀 닿지 않는다. 그대로 두라.
  - `web/dist` 는 커밋하지 말고(`npm run build` 산출물은 지울 것), `VERSION` 은 릴리즈 담당 몫이니 손대지 말 것.

- 차선 후보: `writeSimpleRunLog`·`runLogDisposition`·`logRowScanner`·`simpleRunLogTruncatedNotice` 를 두 모드 공용 이름으로 옮기기 (가치 2 / 위험 1 / S) — v0.5.30 에서 `releases.go` 가 이 네 조각을 그대로 호출해 이름의 'simple' 이 실제로 거짓이 됐다(`releases.go:1640` 의 `writeReleaseLogDownload` 가 `logRowScanner` 를 받는 것을 확인). 동작 변화 0 인 개명·이동이지만 호출부 전부를 건드려 diff 가 커진다. 1순위가 성립하지 않을 때만 고를 것.

- 과제: 로그인 실패 응답 처리 보강 — 429 Retry-After 카운트다운·재제출 차단, 비-JSON 오류 본문에서 파서 예외 노출 방지 (가치 3 / 위험 1 / 작업량 S)
- 왜: 서버는 `internal/server/http.go:1713-1717` 에서 429 에 `Retry-After` 헤더와 "…N초 후 다시 시도하세요" 문구를 주는데, `internal/webui/js/login.js:73-82` 는 그 문구만 한 번 찍고 `button.disabled = false` 로 단추를 즉시 되살린다 — 사용자가 계속 두드려도 매번 같은 429 만 받고, 잠금이 언제 풀리는지 알 수 없다. 같은 자리의 `const body = await response.json()` 은 본문이 JSON 이 아닐 때(프록시·게이트웨이가 낸 HTML 502/504, 빈 본문) 예외를 던지고, 그 예외가 `catch (cause)` 로 흘러 `Unexpected token '<' …` 같은 파서 오류가 그대로 로그인 화면 `#login-error` 에 찍힌다.
- 수용 기준:
  1) 429 응답이면 제출 단추가 대기 시간 동안 비활성으로 남고, 남은 초가 1초마다 줄어들며 안내 문구(또는 단추 라벨)에 보인다. 0이 되면 단추가 다시 눌리고 원래 라벨(`로그인`)로 돌아온다. 대기 시간은 `response.headers.get('Retry-After')` 의 정수 초를 우선 쓰고, 없거나 파싱 불가면 서버 문구에서 뽑지 말고 고정 기본값(예: 30초)으로 떨어진다.
  2) 본문이 JSON 이 아니거나 비어 있어도 파서 예외 문구가 화면에 나오지 않는다 — 대신 우리 문구(예: `로그인하지 못했습니다. (HTTP 502)`)가 나온다. 401 처럼 정상 problem JSON 이 오는 경우의 문구(`body.title`)는 지금 그대로 유지된다.
  3) `test/js/login.test.mjs` 가 (a) 429 → 단추가 비활성으로 남고 타이머 진행 후 재활성, (b) `response.json()` 이 throw 하는 502 → `#login-error` 에 `Unexpected token` 이 없고 우리 문구가 있음, (c) 정상 401 problem → 기존 `body.title` 문구 유지, (d) 성공 → `clearSilentSsoState()` 호출 + `window.location.replace(returnTo)` 를 증명한다. 구현 전에 (a)(b)가 red 인 것을 먼저 확인할 것.
- 건드릴 파일:
  - `internal/webui/js/login.js` — `form.addEventListener('submit', …)` 핸들러(59-83행)만. 실패 분기에서 (i) 본문을 안전하게 읽는 헬퍼(예: `problemMessage(response)`: `await response.json()` 을 try/catch 로 감싸고 실패하면 상태코드 기반 문구 반환), (ii) `response.status === 429` 전용 분기(대기 초 계산 → 단추 비활성 유지 → `setInterval`/`setTimeout` 카운트다운 → 종료 시 라벨·disabled 복구)를 추가. 성공 경로(`clearSilentSsoState()` → `location.replace(returnTo)`)와 상단 IIFE(`system/info`·silent SSO)는 건드리지 말 것.
  - `test/js/login.test.mjs` — 신규. `internal/webui/jstest_test.go:TestBrowserModuleTests` 가 `test/js/*.test.mjs` 를 glob 으로 잡아 `go test ./internal/webui/...` 에서 같이 돌므로 파일만 두면 배선 끝(등록 목록 없음).
- 검증 명령:
  - `go test ./internal/webui/...` (새 JS 테스트 포함. node 없으면 Skip 되므로 출력에 `login.test.mjs` 가 실제로 돌았는지 확인할 것)
  - `node test/js/login.test.mjs` (단독 실행)
  - `go test -race ./...` · `go vet ./...` · `gofmt -l .`
  - 화면 동작 변화가 있으므로 여유가 되면 `DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh` (수분). 실제 429 를 만들려면 `DARTFLY_LOGIN_MAX_FAILURES=1` 로 서버를 띄우고 틀린 비밀번호로 2회 제출.
- 테스트 하네스 쓰는 법(확인한 것 / 미확인):
  - **확인**: `test/js/load.mjs` 의 `repoFile()` + `vm.runInContext(src.replace(/^import .*;\n/gm, ''), context)` 패턴이 `test/js/saved.test.mjs:47` 에서 실제로 쓰이고 통과 중. `login.js` 의 import 는 1행 한 줄짜리라 같은 정규식에 걸린다.
  - **확인**: 따라서 context 에 `shouldAttemptSilentSso`·`beginSilentSso`·`clearSilentSsoState`·`safeReturnTo` 를 직접 넣어야 한다(스텁 또는 `loadModule('silentsso.js')` 의 실물).
  - **확인**: `login.js` 는 `layout.js` 의 `api()` 를 쓰지 않고 전역 `fetch` 를 직접 쓴다 → context 에 `fetch` 스텁 필요. 또 모듈 최상단에서 `document.querySelector('#login-form')`·`#login-error`·`#version-info` 를 읽고 `new URLSearchParams(window.location.search)` 를 부르므로 `document`·`window.location.search`·`URLSearchParams`·`window.location.replace` 가 필요하다. 상단 IIFE 가 즉시 `fetch('/api/v1/system/info')` 를 호출하므로 그 URL 은 `{ ok: false }` 로 흘려보내면 된다.
  - **미확인**: 이 정찰 세션은 샌드박스 권한 때문에 `node` 를 실행하지 못했다. 위 배선을 그대로 담은 초안 스크립트를 `/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-26-121641-DartFly-improve/login_probe.mjs` 에 남겨 두었으니 **구현 전에 먼저 돌려 현재 동작(429 에서 단추가 바로 살아나는지, 502 에서 `Unexpected token` 이 찍히는지)을 눈으로 확인**하고 그것을 씨앗으로 테스트를 쓸 것. 돌려 보기 전에는 "현재 502 에서 파서 문구가 나온다" 를 사실로 단정하지 말 것(코드 독해상으로는 그렇게 보인다).
  - 카운트다운 테스트는 실제 1초를 기다리지 말 것 — context 에 넣는 `setTimeout`/`setInterval` 을 가짜 타이머로 바꿔 수동으로 진행시키는 편이 빠르고 flaky 하지 않다(`saved.test.mjs` 는 디바운스 때문에 실시간 260ms 를 기다리는데, 이번엔 초 단위라 그 방식은 못 쓴다).
- 위험과 피할 것:
  - `internal/server/http.go` 의 `logins` 리미터·`loginHandler`·`writeProblem` 은 **건드리지 말 것**. 서버는 이미 `Retry-After` 를 정확히 보내고 있고(1714행), auth 는 위험 구역이다. 이번 과제는 순수 클라이언트 변경이다.
  - `silentsso.js`·상단 IIFE·`safeReturnTo` 도 건드리지 말 것. 2026-09-20 회차에서 서버 302(`d40c325`)가 이 흐름과 얽혔다.
  - CSP 가 `style-src 'self'` 라 **HTML 인라인 `style` 속성이 막힌다**(2026-09-23 회차에서 걸림). 카운트다운 표시는 기존 `#login-error`/단추 라벨의 textContent 로 하고, 새 스타일이 필요하면 `internal/webui/css/app.css` 에 클래스로 넣을 것.
  - 인라인 `display` 가 `hidden` 속성을 이기는 함정(같은 회차) — 요소를 새로 숨기고 보이는 설계는 피하고 textContent·disabled 만 쓰는 쪽이 안전하다.
  - 타이머를 걸어 둔 채 성공 제출이 일어나는 경합: 카운트다운 중에는 단추가 비활성이라 제출이 안 되지만, 그래도 새 제출 시작 시 이전 타이머를 `clearInterval` 로 반드시 끊을 것(2026-09-20 `saved.js` 회차와 같은 종류의 늦은 콜백 문제).
  - 서버 문구 `"…N초 후 다시 시도하세요"` 를 정규식으로 파싱해 초를 뽑지 말 것 — 문구가 바뀌면 조용히 깨진다. 헤더를 쓰고, 없으면 고정 기본값으로.
- 차선 후보: `internal/server/history.go:55` 와 `internal/server/http.go:1659` 의 '모든 실패 404' 를 2026-09-24 회차가 `internal/resultsave` 에 세운 센티널(`ErrNotFound`/`ErrInvalidID`) + `errors.Is` 분류 방식과 같은 모양으로 따라가기 (가치 2 / 위험 2 / 작업량 M). 이미 성립이 증명된 패턴의 복제라 위험은 낮지만 사용자가 보는 변화는 거의 없다.

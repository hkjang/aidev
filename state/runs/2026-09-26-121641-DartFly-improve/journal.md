# 회차 노트 2026-09-26-121641-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:16] base pinned — main@dd4dd3b
- [러너 12:16] autonomy release —

## 정찰 노트
- 고른 이유: 로그인 429 는 오래 보류돼 있었지만, 코드를 열어 보니 서버가 이미 `Retry-After` 를 정확히 보내고 있어(http.go:1714) **순수 클라이언트 변경**으로 끝난다는 점과, 같은 몇 줄 안에서 `await response.json()` 이 비-JSON 본문에 예외를 던져 파서 문구가 화면에 새는 진짜 결함을 함께 발견해 가치가 2→3 으로 올랐습니다. 차선(history.go/http.go 404 분류)은 증명된 패턴의 복제라 안전하지만 사용자가 보는 변화가 없어 제쳤고, 서버 쪽 검색(q)·end_session 은 권한/IdP 위험 구역이라 제외했습니다.
- 확신 없는 곳: 502 에서 파서 문구가 실제로 화면에 찍히는지는 **코드 독해까지만** 했습니다 — 이 세션은 샌드박스 권한 때문에 `node` 를 한 번도 실행하지 못했습니다(`go test` 포함 전부 미실행, baseline green 미확인). 그래서 과제서에 배선을 그대로 담은 `login_probe.mjs` 를 남겼습니다.
- 구현자가 조심할 것: **먼저 `login_probe.mjs` 를 돌려 red 를 눈으로 확인한 뒤** 시작하세요. 과제서의 전제가 틀렸으면 거기서 드러납니다. 서버 문구에서 정규식으로 초를 뽑지 말고 `Retry-After` 헤더를 쓰고, CSP(`style-src 'self'`) 때문에 인라인 style 은 금지이며, 새 제출 시 이전 카운트다운 타이머를 반드시 끊으세요.
 
- [러너 12:21] scout done — 로그인 실패 응답 처리 보강 — 429 Retry-After 카운트다운·재제출 차단, 비-JSON 오류 본문에서 파서 예외 노�

## 구현 노트
- 무엇을/왜: `login.js` 의 submit 실패 분기만 고쳤습니다(커밋 6daa89e, 서버 무변경). 429 면 `Retry-After` 초만큼 단추를 비활성으로 두고 라벨에 남은 초를 세다 0 에서 `로그인` 으로 복구하고, 오류 본문 읽기를 감싸 비-JSON 본문일 때 `Unexpected token '<' …` 대신 `로그인하지 못했습니다. (HTTP nnn)` 을 냅니다.
- 확신 없는 곳: 사실상 없습니다 — 유닛(6개, 구현 전 4 red)뿐 아니라 실제 바이너리+MariaDB 에 진짜 429(`Retry-After: 12`)를 만들어 Chromium 으로 카운트다운·재제출 0건·502·401·성공 이동·CSP 0건을 모두 봤습니다. 다만 (1) `Retry-After` 가 HTTP-date 형식으로 오는 경우는 코드상 고정 30초로 떨어지게만 두고 테스트하지 않았습니다(이 서버는 항상 정수 초를 보냄), (2) 탭을 백그라운드로 두면 브라우저가 `setInterval` 을 1초보다 느리게 돌려 카운트다운이 실제 잠금보다 늦게 끝날 수 있습니다 — 그 경우 단추가 조금 더 오래 잠길 뿐이라 안전 방향입니다.
- 일부러 안 한 것: 서버(`logins` 리미터·`loginHandler`·`writeProblem`)·`silentsso.js`·상단 IIFE·`safeReturnTo` 는 손대지 않았습니다. 보류 아이디어의 '제출 중 라벨 로그인 중…' 도 범위를 좁히려고 넣지 않았습니다(라벨 상수만 세워 뒀으니 다음에 한 줄).
- 다음 역할이 조심할 것: `test/js/login.test.mjs` 는 node 만 있으면 돌고 DB 가 필요 없습니다(`go test ./internal/webui/...` 에 glob 으로 자동 포함 — Skip 여부를 `-v` 로 확인하세요). 실제 429 를 다시 보려면 `DARTFLY_LOGIN_MAX_FAILURES=1 DARTFLY_LOGIN_FAIL_WINDOW=12s` 로 띄우세요. 함정 둘: `test/smoke/run.sh` 의 `pkill -f "$BIN"` 은 같은 문자열을 담은 내 셸 명령줄까지 죽입니다(exit 144 — 런처를 파일로 빼세요), 그리고 CSP `script-src 'self'` 가 Playwright `wait_for_function` 의 문자열 평가를 막으므로 폴링으로 기다려야 합니다.
- [러너 12:29] brief accepted — 채택 — 지목한 두 결함과 배선 전제(정규식 import 제거, 전역 fetch, 필요한 context 스텁, glob 자동 배선, `Retry-After` 존재)가 �
- [러너 12:29] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인함: 테스트가 진짜 검증하는지를 직접 재현 — main 의 login.js 로 되돌리면 4개 red(429 카운트다운·Retry-After 폴백·비-JSON 본문·타이머 취소), HEAD 에서 6개 green. `go test ./internal/webui/... -v` 에서 `login.test.mjs` 가 Skip 없이 실제로 도는 것도 확인(glob 자동 배선 OK). gofmt·go vet·go build 통과. LOGIN_LABEL 이 login.html:23 의 '로그인' 과 일치함, textContent 경로라 XSS 없음, 스모크의 429 프로브는 다른 이메일이라 브라우저 로그인을 오염시키지 않음(IP 뒷단 한도 100)도 대조함.
- 못 봄: 브라우저 스모크(`DF_SMOKE_REQUIRE_BROWSER=1`)와 livedb 는 이 세션에서 돌리지 않았습니다 — 구현자가 실제 바이너리+Chromium 으로 봤다는 보고에 의존합니다. 탭 백그라운드 스로틀링과 HTTP-date 형식 Retry-After 도 미검증(구현자가 밝힌 두 자리 그대로).
- 승인이어도 남는 우려: 서버는 남은 창 **전체**를 Retry-After 로 보내므로(http.go:1714) 기본 설정에서 단추가 최대 약 5분 잠깁니다. 구현자는 12초 창으로만 실사용 검증했습니다 — 릴리스 노트에 "기본값에서 최대 5분" 을 적어 두세요.
- 다음 회차 거리: 클라이언트 잠금이 서버보다 넓습니다(서버는 (IP,이메일), 클라이언트는 폼 전체). 이메일을 고쳐도 잠금이 남으므로 '입력 변경 시 카운트다운 해제' 와 setInterval 대신 Date.now() 마감시각 앵커가 자연스러운 후속입니다.
- 보안·법무: 차단 없음. 신규 경로·인가 변경·비밀값·의존성·암호 비교 없음, 개인정보 신규 수집 없음. 429 문구가 계정 존재를 드러내지 않는 점도 확인했습니다.
- [러너 12:32] review approved — 리뷰 승인 (risk=low)
- [러너 12:32] pr created — https://github.com/hkjang/DartFly/pull/14
- [러너 12:37] ci passed — 검사 3개 모두 success
- [러너 12:37] merge done — 6daa89e

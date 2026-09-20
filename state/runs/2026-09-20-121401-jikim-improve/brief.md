# 과제서 2026-09-20 — jikim

- 과제: loginRateLimiter가 10,000개 초과 시 잠긴 계정의 항목을 임의로 지워 잠금이 밀려나는 문제 정리 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/httpapi/login_rate_limit.go:66-80`의 `failed()`는 항목이 10,000개를 넘으면 만료된 것을 지운 뒤에도 남으면 **Go map 순회 순서(무작위)** 로 방금 넣은 key만 빼고 아무 항목이나 지웁니다. 잠긴(count ≥ limit) 항목도 대상이라, 한 IP에서 서로 다른 username 10,000개로 실패를 흘리면 같은 IP가 잠가 둔 표적 계정의 잠금 항목이 확률적으로 밀려나 5분 창이 사라지고 다시 5번을 시도할 수 있습니다(잠금이 "지연"으로 격하됨). 잠긴 항목은 만료 전에는 지우지 않고, 아직 잠기지 않은 항목부터 지우며, 그래도 넘치면 **만료가 가장 이른 것**부터 결정적으로 지우게 바꾸면 용량 상한은 유지하면서 잠금 보장이 살아납니다.

- 수용 기준:
  1) 항목이 10,000개를 넘어도 `count >= limit`이고 만료 전인 항목은 아직 잠기지 않은 항목이 하나라도 남아 있는 한 지워지지 않는다. 방금 실패한 key도 지워지지 않는다(기존 동작 유지).
  2) 잠긴 항목만으로 상한을 넘을 때는 만료 시각(`expiresAt`)이 가장 이른 항목부터 지운다(무작위 아님). 상한(10,000)과 만료 항목 우선 삭제는 그대로.
  3) 테스트가 증명할 것 — (a) **프로덕션 배선**으로: `quietServer()` + `server.loginLimiter = newLoginRateLimiter()` + `authenticator`가 `store.ErrUnauthorized`를 돌려주는 상태에서 `server.login`을 실제로 호출해 표적 username을 5번 실패시켜 잠근 뒤(429 확인), 같은 RemoteAddr에서 서로 다른 username으로 상한을 훌쩍 넘게 `server.login` 실패를 흘리고, 다시 표적으로 로그인하면 여전히 429(`Retry-After` 헤더 있음)이고 `len(server.loginLimiter.failures) <= 상한`이다. 상한은 `limit`·`window`처럼 limiter 구조체 필드(`capacity`, 기본 10,000)로 빼서 테스트에서 100으로 낮추고 2,000번 흘린다 — 현재 코드는 초과 뒤 매 호출마다 무작위 항목 하나를 지우므로 표적 항목의 생존 확률이 (1-1/101)^1900 ≈ e^-19로 사실상 0이라 수정 전에는 반드시 실패하고, 수정 뒤에는 결정적으로 통과한다(정찰 계산: 상한 10,000 그대로 두고 20,000번 흘리면 생존 확률 e^-1 ≈ 37%라 현재 코드에서도 통과할 수 있고, 초과 시 매 호출 O(n) 순회라 느리다). 구현자는 수정 전 한 번 실행해 실패를 확인한 뒤 고친다. (b) limiter 단위: `limit=2`, `now` 고정, 잠긴 항목만 10,001개(만료 시각을 서로 다르게)를 채운 뒤 `failed("new")`를 부르면 만료가 가장 이른 항목이 지워지고 `"new"`와 나머지는 남는다. (c) 기존 `TestLoginRateLimiterBlocksResetsAndExpires`·`auth_outage_test.go`의 4개 limiter 테스트는 그대로 통과.

- 건드릴 파일:
  - `internal/httpapi/login_rate_limit.go:failed` — 초과 시 정리 순서를 ① 만료 항목 ② `count < l.limit`인 항목(방금 넣은 key 제외) ③ 그래도 넘치면 `expiresAt` 오름차순으로 key 제외하고 삭제, 로 바꾼다. ③은 슬라이스로 모아 `sort.Slice` 후 앞에서부터 지우면 된다(10k 정렬은 초과 상황에서만 발생, O(n log n) 허용). 상수 `10_000`은 `loginFailureCapacity` 상수 + 구조체 필드 `capacity`(`newLoginRateLimiter`에서 기본값 대입, `limit`·`window`와 같은 방식)로 빼서 테스트가 낮출 수 있게 한다.
  - `internal/httpapi/login_rate_limit_test.go` — 수용 기준 3(b) 단위 테스트 추가.
  - `internal/httpapi/auth_outage_test.go` 또는 새 `login_rate_limit_flood_test.go` — 수용 기준 3(a) 배선 테스트 추가(`TestRejectedCredentialStillCountsAsFailedAttempt` 패턴 그대로: `httptest.NewRequest("POST", "/api/v1/auth/login", …)`, `server.login(response, request)`; `httptest`의 기본 RemoteAddr `192.0.2.1:1234`가 모든 요청에 같아 `loginRateKey`의 IP 부분이 일치함).
  - 문서: 이 정찰에서 README·docs·SECURITY.md에 로그인 잠금 상한(10,000)을 설명한 곳을 찾지 못했음(`grep -rn "rate limit\|잠금" docs README.md SECURITY.md` 결과 없음) — 문서 수정 불필요. CHANGELOG는 릴리즈 세션 관례대로 두되, 있으면 Unreleased 항목에 한 줄.

- 검증 명령:
  - `go test ./internal/httpapi/ -run 'TestLoginRateLimiter|TestLoginOutage|TestRejectedCredential|TestRefusedLocalLogin|Flood' -count=1 -v` (정찰에서 기존 limiter 테스트 0.9초 확인)
  - `gofmt -l internal/ && go vet ./...`
  - `./scripts/verify.sh` (전체, 2~5분; Go test·vet·gofmt, React test·lint·build, docs, compose — React 쪽은 변경 없으니 통과해야 함)

- 위험과 피할 것:
  - 미머지 PR 두 개와 겹치지 말 것: 메일 PR(auto/2026-09-16-0332, `4f729aa`)은 `resource_handlers.go` settings·`store/settings.go`·`SettingsPage.tsx`를, 출력 축소 PR(auto/2026-09-19-1527, `2f42b18`)은 `scripts/verify.sh`·`Makefile`·`scripts/e2e-docker.sh`·`web/e2e`를 만진다. 이 과제는 `login_rate_limit*.go`와 테스트 파일만 건드리므로 그 파일들은 열지도 말 것.
  - `auth_handlers.go:login`·`openbao.go:baoUserpassLogin`의 `failed/succeeded` 호출 위치는 2026-09-17에 정리됐고(`d6f5f5b`) 바꿀 이유가 없다 — 인증 경로(auth) 로직은 손대지 않는다.
  - `blocked()`·`succeeded()`의 동작은 그대로. 상한 값 10,000, 5회/5분도 바꾸지 않는다(문서화되지 않은 값이지만 운영 기대치).
  - 배선 테스트가 20,000번 `server.login`을 부르므로 실행 시간이 늘 수 있다 — `quietServer()`가 로거를 버리는지 확인하고(auth_outage_test.go 참고), 1초 이상 걸리면 `testing.Short()`로 건너뛰지 말고 흘리는 양을 상한+limit 수준으로 줄이되 "수정 전 실패"를 유지할 수 있는지 다시 확인할 것. 테스트에 FakeLimiter·소스 문자열 검사·주입 대역을 쓰지 말 것(운영자 지시).
  - 효과 없는 변경 금지: 실제로 무작위 삭제가 결정적 삭제로 바뀌고 잠금이 보존되는 것을 테스트 실패→성공으로 보여야 한다.

- 차선 후보: `requestedOpenBaoVersion`(`internal/httpapi/openbao.go:514`)이 `?version=-1`·`?version=0`을 오류 대신 latest로 처리하는 동작을 400 `invalid version`으로 바꾸고 `docs/api-guide` 호환 절에 명시 (2/2/S). 단 OpenBao 자체 동작을 확인하지 못했으므로 구현자가 OpenBao 문서(`version=0`이 latest인지)를 먼저 확인하고, 0은 latest 유지·음수만 거부하는 쪽이 안전하다.

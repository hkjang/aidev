- 과제: 추적이 꺼져 있어도 인증 없는 CSP 리포트가 관리자 진단 목록을 채우는 문제 막기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `/api/v1/tracking/csp-report`(`internal/httpapi/tracking.go:23` `cspReportPath`)는 인증이 없고(브라우저가 자격증명 없이 보내므로 의도된 것), `receiveCSPReport`는 추적 설정을 보지 않고 무조건 `s.violations.Record(...)` 를 호출한다. 그런데 추적이 꺼져 있으면 `pagePolicy`가 `defaultPagePolicy()`(=`report-uri` 없음)를 내보내므로 정상 브라우저는 이 주소로 아무것도 보내지 않는다 — 즉 기본 설치에서 이 엔드포인트에 도달하는 리포트는 전부 외부에서 직접 넣은 것이다. 같은 파일의 `momentoProxy`는 `ProxyActive()`가 아니면 404로 "기본 설치는 아무것도 노출하지 않는다"를 이미 지키는데(tracking.go:205-212) 리포트 기록만 이 원칙에서 빠져 있다.
- 왜(계속): 결과적으로 추적을 켜 본 적 없는 서버에서도 누구든 서로 다른 `blocked-uri` 100개(`internal/tracking/violations.go:14` `MaxViolations = 100`, 초과 시 `evictOldest`로 오래된 것부터 밀어냄)를 넣어 관리자 화면의 "차단된 출처" 목록을 임의 값으로 채울 수 있고, 그 목록에는 `allowTrackingOrigin`(tracking.go:179) 로 CSP 허용 목록(`script-src`·`connect-src`)에 한 번에 넣는 버튼이 붙어 있다. 고치면 추적이 꺼진 서버에서 이 목록은 항상 비어 있고, 관리자가 보는 항목은 실제로 켜 둔 추적이 만든 것만 남는다.

- 수용 기준:
  1) 추적이 꺼진(기본) 설정에서 `POST /api/v1/tracking/csp-report` 에 형식이 올바른 리포트를 여러 번 보내도 응답은 그대로 `204 No Content` 이고, 관리자(`GET /api/v1/tracking/violations`)가 보는 목록은 `"data":[]` 다.
  2) 추적이 켜진 설정에서는 기존 동작이 한 글자도 바뀌지 않는다 — 기존 `TestPolicyReportsAreRecordedOnceAndListedForAdministrators`(internal/httpapi/tracking_test.go:198)가 수정 없이 통과하고, 같은 리포트 3회 → `"count":3` 한 항목이 그대로 나온다.
  3) 응답 코드로 추적 on/off 를 구별할 수 없다 — 꺼진 상태에서도 204이며 본문이 없다(현재 주석 "Reports are always answered with 204 so a misbehaving page never sees an error from us." 를 지킨다).
  4) 새 테스트가 증명할 것: (a) 추적 off + 올바른 리포트 N회 → 목록 0건·204, (b) 추적 on + 같은 리포트 → 목록 1건(`count` 누적), (c) 설정 읽기 실패(`trackingServer(t, cfg, loadErr)` 의 `loadErr`) → 목록 0건·204 (저장소 장애가 500이나 패닉이 되지 않는다).
  5) 수정을 되돌리면 (a) 테스트가 다시 실패하는 것을 확인한다.

- 건드릴 파일 (프로덕션 2개, 테스트 1개):
  - `internal/tracking/tracking.go` — `ProxyActive()`(102행) 바로 옆에 같은 꼴의 작은 메서드 하나 추가: `func (c Config) ReportingActive() bool`. 판정은 `c.Enabled && c.Provider != ProviderNone && c.Provider != "" && strings.TrimSpace(c.Snippet("")) != ""` — 즉 `Active(path)`(84행)에서 경로별 관리자 제외 규칙만 뺀 것. (경로 규칙을 쓰면 안 되는 이유: 리포트의 `document-uri` 는 공격자가 정하는 값이라 게이트로 쓸 수 없다. 관리자 화면만 추적하는 `IncludeAdmin` 구성에서도 리포트는 받아야 한다.)
  - `internal/httpapi/tracking.go:receiveCSPReport`(138행) — `if s.violations == nil { return }` 바로 뒤에 `if !s.trackingConfig(r.Context()).ReportingActive() { return }` 를 넣는다. `defer w.WriteHeader(http.StatusNoContent)` 는 함수 첫 줄에 그대로 두어 어느 분기로 빠져도 204가 나가게 한다. 본문 읽기(`io.ReadAll(io.LimitReader(...))`)·JSON 파싱·`Record` 호출 순서와 문구는 건드리지 않는다.
  - `internal/httpapi/tracking_test.go` — 기존 하네스 `trackingServer(t, config, loadErr)`(29행)와 `momentoConfig()`(48행), `tokenRequest`, `server.sessionResolver` 를 그대로 써서 테스트 1개(또는 서브테스트 3개) 추가. 꺼진 설정은 `tracking.ReadConfig(nil)` 이면 되고(= `TestFreshInstallServesTheShippedPolicyAndNoSnippet`(61행)이 쓰는 것과 같은 기본), `Content-Type: application/csp-report` 를 붙인 `httptest.NewRequest(http.MethodPost, cspReportPath, ...)` 로 왕복한다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/httpapi/ -run Tracking -count=1 -v`  ← 새 테스트와 기존 리포트 테스트를 같이 본다
  - `go test ./internal/tracking/ ./internal/httpapi/ -count=1`
  - `go test ./... -count=1` · `go vet ./...` · `gofmt -l .`
  - 마지막에 `./scripts/verify.sh` (프런트 `npm ci` 부터 돌기 때문에 시간이 걸린다. 이 과제는 `web/` 을 건드리지 않으니 프런트 실패가 나면 자기 변경 탓이 아닌지 먼저 확인할 것)

- 위험과 피할 것:
  - **`Active(path)` 를 게이트로 쓰지 말 것.** `Active` 는 `IncludeAdmin` 이 꺼져 있으면 관리자 경로에서 false 이고, 리포트 게이트에 경로를 넣으려면 공격자가 정하는 `document-uri` 를 믿어야 한다. 새 메서드를 만들고 `Active` 의 본문·시그니처는 그대로 둘 것.
  - **`MaxViolations` 를 바꾸거나 `evictOldest` 를 건드리지 말 것.** 추적이 켜진 동안의 리포트 폭주는 이 과제의 범위가 아니다(그쪽은 속도 제한·동일 출처 검증이라는 별도 계약이 필요하다). 이번 회차는 "꺼져 있으면 아무것도 기록하지 않는다" 한 가지만 담는다.
  - **204 와 본문 없음을 유지할 것.** 꺼진 상태에서 403/404를 주면 외부에서 추적 설정 여부를 알아낼 수 있고 기존 주석의 계약을 깬다.
  - 인증 없는 경로에서 설정을 읽게 되는 점은 이 파일의 기존 관례와 같다 — `decorateIndex`(113행)가 이미 로그인 화면을 포함한 모든 SPA 페이지 요청마다 `s.trackingConfig` 를 부른다. 비평 단계에서 "인증 없는 DB 읽기가 새로 생겼다"는 지적이 나오면 이 선례를 근거로 답하고, 그래도 걸리면 `trackingConfig` 의 기존 실패 폴백(경고 로그 + 추적 off)이 그대로 적용되어 장애 시에는 기록하지 않는 쪽으로 닫힌다는 점을 함께 적을 것.
  - `internal/httpapi/server.go:routes` 의 라우트 등록·`securityHeaders`·`auditablePath` 는 건드리지 말 것(리포트가 감사 로그를 채우지 않는다는 기존 단정이 tracking_test.go:216에 있다).
  - 보호 경로(`auth_handlers.go`·`oidc*.go`·`mcp_oauth.go`·`store/users.go`·`migrations/`·`.github/workflows/`)와 `web/` 은 이번 과제에서 전혀 필요 없다.
  - 작업 커밋에서 `CHANGELOG.md`·`scripts/version.sh` 를 건드리지 말 것(릴리즈는 별도 커밋 관례).
  - `scripts/e2e-docker.sh`·Playwright 는 `docs/screenshots/*.png` 를 덮어쓰므로 돌리지 말 것.

- 차선 후보: **aiRequestLimiter 의 사용자 표에 결정적 상한을 세우기** (가치 2 / 위험 1 / 작업량 S) — `internal/httpapi/ai_rate_limit.go:45-51`. `len(l.users) > 4096` 일 때만, 그것도 `active == 0 && windowStart` 가 2분 넘은 항목만 지우므로 최근 활동 사용자가 많으면 한 건도 지워지지 않고 표가 계속 자란다. 2026-09-20 회차가 `login_rate_limit.go` 에 넣은 방식(상수 + capacity 필드 + 분리된 결정적 evict 함수 + 흘림 테스트)을 그대로 옮길 수 있고 기존 `ai_rate_limit_test.go` 가 하네스를 준다. 키가 인증된 사용자 ID라 외부에서 임의로 늘릴 수 없어 가치는 1순위보다 낮다.

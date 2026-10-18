- 과제: 로그인 경로의 속도 제한·감사 IP 를 collector 와 같은 신뢰 프록시 인식 `s.clientIP(r)` 로 맞춘다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `internal/httpapi/server.go` 에는 이름이 거의 같은 두 함수가 있다 — 패키지 함수 `clientIP(r)`(463행, `r.RemoteAddr` 만 본다)와 메서드 `(*Server).clientIP(r)`(535행, `trusted_proxy_cidrs` 로 신뢰한 프록시에서 온 요청에만 `X-Forwarded-For` 를 오른쪽→왼쪽으로 걸어 실제 클라이언트를 찾는다). collector 는 후자를 쓰는데(417행 `ip := s.clientIP(r)` → 418행 `s.Limiter.Allow(req.SiteID+"|"+ip)`) **로그인은 전자를 쓴다**(584행 `s.LoginLimiter.Allow(clientIP(r))`). 이 제품이 문서로 안내하는 배치가 리버스 프록시 뒤(`admin.go:497` 의 설치 안내 nginx 블록에 `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;` 가 있다)라서, 그 배치에서 로그인 제한 키는 모든 요청에 대해 **프록시 IP 하나**가 된다 — `newRateLimiter(10)`(59행) 의 분당 10회가 IP 별이 아니라 설치 전체의 합계가 되어, 한 사람(또는 한 공격자)의 실패 10번이 전 직원의 로그인을 429 로 막고, 동시에 개별 출처에 대한 brute-force 제한으로는 동작하지 않는다. 같은 자리의 `login`·`login.failed` 감사 기록(599·604행)도 프록시 IP 만 남겨, 로그인 실패 추적이라는 그 기록의 용도를 잃는다.

- 수용 기준:
  1) 신뢰 프록시 대역이 설정된 상태에서 **같은** RemoteAddr(프록시)로 들어온 서로 다른 `X-Forwarded-For` 클라이언트들이 로그인 속도 제한을 **공유하지 않는다** — 한쪽이 한도를 다 써도 다른 쪽은 429 를 받지 않는다.
  2) RemoteAddr 이 신뢰 대역 **밖**이면 `X-Forwarded-For` 를 무시하고 지금과 똑같이 RemoteAddr 로 제한한다 — 헤더를 바꿔 가며 보내도 한도를 벗어나지 못한다. (신뢰 프록시가 아예 설정되지 않은 설치에서는 동작이 글자 그대로 지금과 같다.)
  3) `login`·`login.failed` 감사 기록의 IP 가 제한 키와 **같은 값**에서 나온다(두 자리가 다시 어긋나지 않게 핸들러 안에서 한 번만 계산한 지역 변수를 쓴다).
  4) 테스트는 손으로 만든 핸들러가 아니라 **`s.Handler()` 가 등록한 실제 라우트**(`POST /api/v1/auth/login`, 86행)로 요청을 보내 증명한다. 1)·2) 는 DB 없이 증명된다 — 제한 거절은 `decodeJSON`·`s.Auth.Login` 보다 **앞**(584행)이라 429 경로는 DB 를 건드리지 않는다.

- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `internal/httpapi/server.go:583 func (s *Server) login` — 584·599·604행의 `clientIP(r)` 세 자리를 핸들러 첫 줄의 `ip := s.clientIP(r)` 한 번으로 바꾸고 그 변수를 쓴다. collector(417-418행)와 같은 모양이 되도록. 왜 두 함수가 갈라져 있었는지와 왜 로그인이 메서드 쪽이어야 하는지를 이 저장소의 관례대로 긴 한국어 주석으로 남길 것.
  - `internal/httpapi/server_test.go` (신규 테스트 함수 추가; 기존 테스트는 손대지 말 것) — `&Server{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Limiter: newRateLimiter(1), LoginLimiter: newRateLimiter(N)}` 로 서버를 짜고(기존 160행 `TestHandlerRegistersSecretAndDiagnosticRoutes` 가 같은 방식으로 `&Server{...}` 를 쓴다 — 관례), `trustedProxies` 를 `reloadSecurity`(485-490행)와 **같은 방법**으로(`net.ParseCIDR` 결과의 `*net.IPNet`) 채운 뒤 `server.Handler()` 에 `httptest.NewRequest` 로 보낸다. `httptest.NewRequest` 의 기본 RemoteAddr 은 `192.0.2.1:1234` 이므로 신뢰 대역을 `192.0.2.0/24` 로 두면 "프록시 뒤" 를, RemoteAddr 을 직접 다른 값으로 덮으면 "신뢰 대역 밖" 을 만들 수 있다.
  - `Logger` 가 nil 이면 `s.requestLog`(369행 `s.Logger.Info`)에서 패닉한다 — `Handler()` 경로를 쓰는 테스트는 Logger 를 반드시 채울 것.

- 검증 명령:
  - `go test ./internal/httpapi/` — 정찰 실측 `ok … 0.055s`(실제 소요 약 9초, 대부분 컴파일). DSN 이 없어 DB 통합 테스트는 skip 된다.
  - `go vet ./cmd/... ./internal/...`
  - 여유가 있으면 `make verify`(964f507 이 추가함, ci.yml 의 여섯 게이트를 같은 순서로 돌린다). `govulncheck`·`npm audit` 은 네트워크가 필요하다.

- 위험과 피할 것:
  - **정직하게 적는 트레이드오프(과제서가 숨기지 않는다)**: 운영자가 `trusted_proxy_cidrs` 를 지나치게 넓게(예: `0.0.0.0/0`) 잘못 설정하면, 이 변경 뒤에는 누구나 `X-Forwarded-For` 를 돌려 가며 로그인 제한을 우회할 수 있다. 지금은 그 설정이 로그인 제한에 영향을 주지 않는다. **그래도 고치는 것이 맞다** — collector 의 속도 제한(418행)과 쿠키 Secure 판정(`requestSecure`, 552행)이 **이미** 같은 신뢰 계약 위에 서 있어서, 넓은 신뢰 대역은 이미 그 두 곳을 무력화하는 설정 오류다. 로그인만 다른 계약을 쓰는 것이 결함이다. 새 정책(예: 신뢰 대역이 너무 넓으면 거절)을 **발명하지 말 것** — 이번 과제 범위 밖이고, 그것은 `validateAdminSetting`(admin.go:721) 의 계약을 바꾸는 별도 회차다.
  - `internal/auth`, `internal/database/migrations`, `.github/workflows`, `docs/openapi.yaml` 은 건드리지 않는다.
  - `clientIP`·`s.clientIP` 두 함수를 **통합하거나 이름을 바꾸지 말 것**. 운영자 지침("같은 종류의 파서가 여러 경로에 있으면 통합하지 말고 좁히는 방향으로만 고칠 것")이 그대로 적용된다 — 패키지 함수 `clientIP` 는 `s.clientIP`·`requestSecure` 자신이 안쪽에서 쓰는 원시 RemoteAddr 판독기(536·538·556행)라서 이름을 바꾸면 이 회차가 세 파일 이상으로 번진다.
  - 나머지 **약 55곳의 감사 호출**(`enterprise_governance.go`·`segments.go`·`admin.go`·`automation.go` … 전부 `clientIP(r)`)도 같은 결함이지만 **이번에 함께 고치지 말 것** — 파일 12개가 되어 과거 추적치(파일 10개 이상 변경은 47%가 사람 손을 다시 탔다)로 들어간다. 로그인만 먼저 하는 이유는 거기만 **기록이 아니라 동작**(제한 키)이 틀리기 때문이다. 다음 회차용으로 ideas.json 에 남겨 두었다.
  - 테스트의 `LoginLimiter` 한도는 `newRateLimiter(N)` 로 직접 정할 것(프로덕션 기본 10 에 의존해 10번을 돌리지 말 것 — 느리고 기본값 변경에 깨진다). `rateLimiter` 가 분 단위 창을 쓰는지 슬라이딩인지는 **미확인** — 구현자가 `newRateLimiter`/`Allow` 를 먼저 열어 보고 한도 소진 방식을 정할 것.
  - 기존 테스트 문장을 글자 그대로 단언하는 web 테스트와 무관한 변경이다(web·sdk 무변경).

- 차선 후보: `cidrRule.ts` 의 호스트 비트 hint 를 서버의 정규화 저장 결과로 고친다 (가치 2 / 위험 1 / S). 지금 `web/src/pages/cidrRule.ts` 의 마지막 블록 주석은 "Postgres `cidr` 컬럼이 이것을 어떻게 다루는지는 확인하지 못했으니 아무 주장도 하지 않고" 라고 적고 hint 를 `대역의 시작 주소는 10.20.30.0/24 입니다.` 로만 둔다. 그런데 서버는 이제 확인된 동작을 한다 — `admin.go:820 normalizeCIDR` 이 `net.ParseCIDR` 이 계산한 대역을 그대로 저장하므로(802-833행 주석이 이유를 설명한다) 사용자가 적은 `10.20.30.1/24` 는 `10.20.30.0/24` 로 **저장된다**. hint 를 "저장하면 …로 바뀝니다" 로 바꾸고 주석의 '확인하지 못했다' 를 걷어내면 된다. `web/test/cidrRule.test.mjs` 가 이 문장을 단언하는지 **열어서 먼저 확인할 것**(이전 회차 기록은 "주소 정규식만 본다" 고 적었으나 이번 회차에 재확인하지 않았다 — 미확인). 검증: `cd web && npm test`.

# 회차 노트 2026-10-18-153940-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:39] base pinned — main@f8fd507
- [러너 15:39] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 1순위였던 cidrRule.ts hint(가치 2)보다 큰 것이 코드를 읽다 나왔다 — `server.go` 에 이름이 거의 같은 `clientIP(r)`(raw RemoteAddr)와 `(*Server).clientIP(r)`(신뢰 프록시 인식)가 있고, collector 는 후자로 속도 제한을 키잉하는데(417-418행) **로그인만 전자**다(584행). 문서화된 nginx 배치에서는 로그인 제한이 설치 전체 한 덩어리가 되어 한 사람의 실패가 전원을 429 로 막는다 — 안내 문구 교체보다 분명히 가치가 크고, 프로덕션 1파일로 끝난다.
- 추측으로 적은 것(구현자가 확인할 것): `rateLimiter.Allow` 의 창 방식(분 고정 창인지 슬라이딩인지)을 열어 보지 않았다 — 테스트에서 한도를 소진하는 방법은 구현자가 정할 것. `httptest.NewRequest` 의 기본 RemoteAddr 이 `192.0.2.1:1234` 라는 것은 Go 표준 라이브러리 문서 기준이고 이번에 실행으로 확인하지는 않았다. 차선 후보의 `web/test/cidrRule.test.mjs` 단언 범위도 미확인이라고 과제서에 적었다.
- 조심할 것: 두 `clientIP` 를 통합·개명하지 말 것(운영자 지침의 "파서 통합 금지" 가 그대로 적용되고, 번지면 세 파일을 넘는다). 나머지 ~55곳 감사 호출은 같은 결함이지만 12파일이 되므로 이번 회차에서 배제했다 — ideas.json 에 별 항목으로 남겼다. 과제서에 트레이드오프(지나치게 넓은 trusted_proxy_cidrs 설정에서는 이 변경이 제한 우회를 열어 준다)를 숨기지 않고 적었고, collector·requestSecure 가 이미 같은 계약 위에 있다는 근거로 그대로 고치라고 지시했다 — 새 정책을 발명하지 말 것.
- 기각·차선 반복 유형 회피: 최근 두 번은 "실행해 보니 전제가 거짓" 으로 1순위가 기각됐다. 그래서 이번 과제서의 모든 인용은 실제로 파일을 열어 행 번호를 확인했고(server.go 417·463·535·552·583-604·59·86·369, admin.go 497·721·802-863), 검증 명령은 직접 돌려 `ok … 0.055s` 를 찍었다.
- [러너 15:45] scout done — 로그인 경로의 속도 제한·감사 IP 를 collector 와 같은 신뢰 프록시 인식 `s.clientIP(r)` 로 맞춘다 (가치 4 / 위�

## 구현 노트
- 무엇을 왜: `login` 핸들러 첫 줄에서 `ip := s.clientIP(r)` 를 한 번 계산해 `LoginLimiter` 키와 `login`·`login.failed` 감사 IP 세 자리가 같은 값을 쓰게 했다(프로덕션 1파일). 프록시 뒤에서 분당 한도가 설치 전체 합계가 되던 것이 IP 별로 돌아온다. 두 `clientIP` 는 통합·개명하지 않았다.
- 확신 없는 곳: 성공 `login` 감사 행은 **실행으로 보지 못했다** — 픽스처의 `password_hash` 가 리터럴 `'hash'` 라서 테스트에서 성공 로그인을 만들 수 없다. 실패 행(`login.failed`)만 DB 로 확인했고, 성공 행은 한 줄 아래에서 같은 지역 변수를 쓴다는 것이 근거다. 또 제한 테스트는 `rateLimiter` 의 **분 고정 창**에 기대므로 요청 사이에 분이 넘어가면 이론상 흔들린다(기존 `TestRateLimiter`·`TestLoginIsRateLimited` 와 같은 성질, 요청은 마이크로초 간격).
- 일부러 안 한 것: 나머지 ~55곳의 감사 `clientIP(r)`(12파일), 넓은 `trusted_proxy_cidrs` 에 대한 새 정책, 429 분기의 감사 기록 — 전부 ideas.json 에 남겼다. 과제서가 범위 밖으로 둔 `internal/auth`·migrations·워크플로·openapi.yaml 은 손대지 않았다.
- 다음 역할이 조심할 것: `integration_test.go` 에 더한 `TestLoginAuditRecordsTheTrustedProxyClientIP` 는 **DB 가 있어야 돈다**(DSN 없으면 skip). 검증은 로컬 docker `postgres:17-alpine`(포트 55433)에 DSN 을 걸고 `make verify` 를 돌렸다 — 컨테이너는 세션 끝에 지웠다. `server_test.go` 쪽 두 subtest 는 DB 없이 돈다.
- [러너 15:54] brief accepted — 채택 — 인용한 근거가 전부 현재 코드와 일치했다(`server.go:417-418` collector, `463`·`535` 두 `clientIP`, `552` `requestSecure`, `583-604`
- [러너 15:54] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 두 새 테스트를 **깨끗한 pre-fix 워크트리**(main + 테스트 파일만 cherry-pick)에서 직접 돌려 red 를 재현했다 — `server_test.go:253`, `integration_test.go:2680` 으로 원장의 실패 출력과 글자까지 일치했다. 로컬 `postgres:17-alpine`(55434)로 DB 테스트도 red→green 을 양쪽 다 확인했고, `go test -race ./internal/httpapi/` +DSN (76s), `go test ./...`, `go vet ./...` 모두 녹색. 컨테이너·임시 워크트리는 지웠다.
- 구현자가 확신 없다고 적은 성공 `login` 감사 행은 나도 실행으로 보지 못했다 — 같은 지역 변수를 한 줄 아래에서 쓰는 것이 분기 없이 자명해 추가 검증값이 낮다고 판단했다. npm/sdk/web·govulncheck·Docker·릴리즈 smoke 는 미실행(diff 가 JS/TS·의존성·Dockerfile·워크플로를 건드리지 않는다).
- 보안 잔여 위험(차단 아님): `trusted_proxy_cidrs` 안의 행위자가 nginx 를 우회해 `/api/v1/auth/login` 에 직접 닿으면 요청마다 XFF 를 지어내 로그인 brute-force 제한을 완전히 무력화할 수 있다. 기본값이 `[]`(001_initial.sql:200)이고, 같은 행위자가 이미 collector 제한(server.go:417)과 `requestSecure`(:552)를 무력화하며, 안내된 `$proxy_add_x_forwarded_for`(admin.go:497)에서는 오른쪽 끝이 실제 peer 라 우회가 성립하지 않는다. `0.0.0.0/0` 처럼 지나치게 넓어도 루프가 untrusted 를 못 찾아 `remote.String()` 으로 **변경 전 동작으로 퇴화**할 뿐 우회가 열리지는 않는다 — 정찰이 적어 둔 트레이드오프보다 작다.
- 릴리즈 노트가 알아야 할 것: ADMIN_GUIDE.md:358 이 이미 "프록시 뒤에서 모든 사용자가 한 IP로 보이면 `trusted_proxy_cidrs`를 설정합니다" 라고 약속해 왔다 — 이 변경 전까지 그 문장은 거짓이었다. 즉 문서 수정이 필요한 변경이 아니라 **공표된 약속을 코드가 따라잡은** 변경이다.
- 개인정보(차단 아님): `audit_logs.client_ip` 가 프록시가 아닌 실제 직원 주소를 담게 된다. 새 수집이 아니라(같은 열·같은 행·같은 쓰기) 값이 목적에 맞아진 것이고 읽기는 admin 전용(server.go:121)이지만, ADMIN_GUIDE:324 가 보존을 DB 운영자에게 위임해 둔 그 정책이 이제 실질적 의미를 가진다 — 다음 회차가 감사 로그 보존을 볼 가치가 있다. 429 분기 무기록과 나머지 ~55곳 `clientIP(r)` 는 ideas.json 에 남아 있고 모두 기존 성질이다.
- [러너 16:00] review approved — 리뷰 승인 (risk=low)
- [러너 16:00] pr created — https://github.com/hkjang/Momento/pull/35
- [러너 16:06] ci passed — 검사 1개 모두 success
- [러너 16:06] merge done — 9d47ad1
- [러너 16:21] release published — v0.34.65
- [러너 16:23] assets verified — v0.34.65 자산 2개 (이전 v0.34.64: 2)

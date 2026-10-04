- 과제: API 키 발급의 `allowed_cidrs` 검증이 PostgreSQL `cidr` 보다 느슨해 잘못된 주소를 500 "API 키를 저장하지 못했습니다." 로 알리는 것을 400 으로 가르기 (가치 4 / 위험 2 / 작업량 S)

- 왜: `validateKeyInput`(internal/httpapi/keys.go:81-85)이 `net.ParseCIDR` 하나로만 검사한다. Go 는 호스트 비트가 켜진 `10.0.0.5/8` 을 **받아들이고**(마스크를 적용한 네트워크를 따로 돌려준다) PostgreSQL `cidr` 은 "마스크 오른쪽에 비트가 켜졌다" 며 **거절한다**. 그래서 `createMyAPIKey`(keys.go:117)의 `$7::cidr[]` 캐스트가 터지고, 사용자는 자기 입력이 틀렸다는 안내 대신 500 `key_save_failed` 를 받는다 — 바로 위 주석(keys.go:79-80, "Checking the syntax here keeps a bad address a client error, so that a failure at the insert can be reported as what it is.")이 선언한 계약이 깨져 있다. 더해 검증은 `strings.TrimSpace(entry)` 를 보지만 저장은 **트림하지 않은 원문**을 넣어, 같은 값을 읽는 두 경로(검증기 / DB·`ipAllowed`)가 어긋난다.

- 수용 기준:
  1) `POST /api/v1/me/api-keys` 에 `{"name":"...","allowed_cidrs":["10.0.0.5/8"]}` 를 보내면 400 `invalid_key_policy` 가 나온다(지금은 500 `key_save_failed`). **먼저 500 을 실제 HTTP→실제 DB 로 재현해 테스트를 실패시킨 뒤** 고쳐라.
  2) 그 실패 요청이 `api_keys` 행을 남기지 않는다(같은 클라이언트로 `GET /api/v1/me/api-keys` 가 빈 목록).
  3) 정상 입력(`["10.0.0.0/8","192.168.1.0/24"]`, `["2001:db8::/32"]`, 빈 배열, 필드 생략)은 여전히 201 이고 `GET /api/v1/me/api-keys` 의 `allowed_cidrs` 가 그대로 돌아온다 — 기존 키 테스트가 깨지지 않는다.
  4) 테스트가 증명해야 하는 것: 검증기가 DB 의 `cidr` 계약과 **같은 집합**을 받아들인다는 것. 즉 "Go 가 받고 PG 가 거절하는 값" 이 400 으로 나오고, 통과한 값은 실제 INSERT 가 성공하는 왕복. 분기를 두 방향으로 변이시켜(새 판정 무력화 → 500, 항상 거절 → 정상 입력이 400) 각각 다시 실패하는지 확인하라.
  5) (여력이 있으면) 공백이 섞인 `[" 10.0.0.0/8 "]` 의 저장 값이 트림된 값이다 — 목록 응답과 `ipAllowed` 가 읽는 값이 검증기가 본 값과 같아진다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/httpapi/keys.go:47` `validateKeyInput` — `AllowedCIDRs` 루프에서 ① `strings.TrimSpace(entry)` 로 **정규화한 값을 `in.AllowedCIDRs[i]` 에 되써 넣고** ② 호스트 비트가 켜진 입력을 거절하라. `ip, ipnet, err := net.ParseCIDR(v)` 뒤 `if err != nil || !ip.Equal(ipnet.IP) { return false }` 가 가장 짧은 판정이다(루프를 `for i, entry := range` 로 바꿔야 한다). `createMyAPIKey`·`rotateMyAPIKey` 둘 다 이 함수만 거치므로 **다른 함수는 고치지 마라**.
  - `internal/httpapi/integration_test.go` — 새 테스트 하나(`TestIntegrationAPIKeyRejectsCIDRPostgresRejects` 류). 관례는 `newClient(t, server.URL)` → `register(name)` → `do(method, path, body, expectedStatus)`; 본보기는 integration_test.go:225-233. `register` 로 만든 계정은 `buyer` 역할이고 `buyer` 에 `keys.manage.self` 가 있으므로(001_initial.sql:526) 라우트(`router.go:59`, `s.require("keys.manage.self", s.createMyAPIKey)`)를 그대로 통과한다.
  - `docs/openapi.yaml:318-322` — `POST /me/api-keys` 응답에 `'201'` **하나만** 있다(확인). 핸들러가 실제로 내보내는 `'400'`(`invalid_key_policy`)·`'500'`(`key_generation_failed`/`key_save_failed`) 두 줄을 더하라. 없는 코드는 적지 마라.

- 검증 명령:
  - 버릴 PostgreSQL 16 을 띄우고(이번 회차에는 `docker version` 이 **승인 거부로 막혀 가용성 재확인 실패** — 지난 5회차에서는 29.7.2 였다):
    `docker run --rm -d -e POSTGRES_PASSWORD=p -p 5433:5432 postgres:16`
  - 단독: `cd /home/hkjang/.cache/auto-improve-wt/Kkiit && KKIIT_TEST_DSN='postgres://postgres:p@127.0.0.1:5433/postgres?sslmode=disable' go test ./internal/httpapi/ -run TestIntegrationAPIKey -v -timeout 300s` — `--- PASS` 를 확인하라. **DSN 없이 통과하는 것은 SKIP 이고 검증이 아니다.**
  - 전체: `KKIIT_TEST_DSN=... go test ./cmd/... ./internal/...`(httpapi 만 80~100초) · `go vet ./cmd/... ./internal/...` · `gofmt -l cmd internal`(무출력).
  - 프런트를 건드리지 않으므로 `npm --prefix web test` 는 불필요. 돌릴 거라면 PATH 의 node 가 24 이상인지 보라 — 이 기계의 `/home/hkjang/node_modules/node/bin/node`(v20)가 끼어들면 글롭이 0건이 된다(2026-10-03 확인).

- 위험과 피할 것:
  - **`internal/httpapi/middleware.go` 를 고치지 마라.** `ipAllowed`(:248-262)와 `apiKeyPrincipal`(:227)은 인증 경로다. 이번 과제는 "들어올 때 정규화" 로 두 경로를 일치시키는 것이고, 읽기 쪽을 손대면 위험 구역에 들어간다.
  - 검증을 **더 조이기만** 하라. `net.ParseCIDR` 을 다른 파서로 바꾸거나 IPv6 처리를 새로 쓰지 마라 — `2001:db8::/32` 가 계속 201 인 것을 테스트로 고정하라(`2001:db8::1/32` 는 호스트 비트가 켜진 쪽이라 400 이어야 한다).
  - `rotateMyAPIKey`(keys.go:140)는 DB 가 이미 정규화해 돌려준 `allowed_cidrs::text[]` 를 다시 검증·삽입한다. **회전 경로가 새 판정에 걸리지 않는 것을 확인하라** — 여기서 400 이 나면 정규화 판정이 PG 의 `cidr` 출력 표기와 어긋난 것이다. 기존 회전 테스트가 있으면 그것으로 족하고, 없으면 `allowed_cidrs` 가 설정된 키를 회전시키는 단계를 새 테스트에 한 줄 더해라.
  - 500 → 400 은 `writeError` 한 곳으로 나가고 프런트는 `error.message` 를 그대로 띄우므로 `web/`·`internal/ui/dist` 를 **건드리지 마라**(추적되는 dist 커밋 금지). 새 문구를 만들지 말고 기존 `invalid_key_policy` "키 이름, 권한, IP 제한 또는 만료일을 확인해 주세요." 를 그대로 써라.
  - grep 결과를 증거로 내지 마라. 실제 HTTP → 실제 DB 왕복으로 500 을 먼저 재현한 뒤 고쳐라.
  - 프로덕션 파일은 1개(keys.go)로 끝난다. 늘어나면 과제가 샌 것이다.

- 미확인 (이 정찰이 실행으로 확인하지 못한 것):
  1) **`net.ParseCIDR("10.0.0.5/8")` 이 err=nil 로 통과한다**는 것과 **PostgreSQL 이 그 값을 `cidr` 로 거절한다**는 것 — 둘 다 각 문서에 명시된 계약이지만 이 회차에서는 `go run`·`docker` 가 모두 승인 거부로 막혀 **실행으로 보지 못했다**. 구현자는 수용 기준 1 의 "먼저 500 재현" 으로 이것을 증명하라. 만약 `10.0.0.5/8` 이 실제로 201 이 나오면(= PG 가 받아들이면) **1순위는 성립하지 않는다** → 차선 후보로 가라.
  2) PostgreSQL `cidr_in` 이 선행·후행 공백을 허용하는지 — 수용 기준 5 를 "지금 몇 번이 나오는지" 테스트로 먼저 고정한 뒤 판단하라.
  3) `integration_test.go` 에 기존 API 키 테스트가 어디까지 있는지 — 열어 보지 않았다. 중복되는 이름·헬퍼가 있으면 그것을 따르라.

- 차선 후보: `setTalentStatus`(internal/httpapi/reports.go:325)의 `SELECT status,title FROM talents WHERE id=$1 FOR UPDATE` 실패를 전부 404 `talent_not_found` 로 묶는 것을 500 과 가르기 — 같은 함수의 :336·:340 이 이미 500 `update_failed` 를 쓰므로 비대칭이 한 함수 안에 있다. 다만 파라미터가 uuid 뿐이라 HTTP 로 DB 오류를 만들 경로가 없어 증거는 404 회귀 + 분기 단위 검증까지다.

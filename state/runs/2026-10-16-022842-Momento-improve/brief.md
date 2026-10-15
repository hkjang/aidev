# 과제서 — run 2026-10-16-022842-Momento-improve (base: main@0beff4c, v0.34.61)

- 과제: `createNetwork` 가 `net.ParseCIDR` 로 파싱한 대역을 버려서, 호스트 비트가 켜진 CIDR 이 Postgres `cidr` 컬럼에 막혀 500 NETWORK_CREATE_FAILED + pgx 원문으로 돌아온다 — 파싱 결과(정규화된 대역)를 INSERT 에 쓴다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/httpapi/admin.go:createNetwork`(800번대)는 `if _, _, err := net.ParseCIDR(in.CIDR); err != nil` 로 **반환값 두 개를 버리고** 원문 `in.CIDR` 를 그대로 `INSERT INTO network_ranges(...)` 에 넘긴다. 그런데 컬럼은 `internal/database/migrations/001_initial.sql:67` 의 `cidr cidr NOT NULL` 이고, PostgreSQL `cidr` 타입은 넷마스크 오른쪽에 비트가 켜진 값을 **거절한다**(문서화된 동작, SQLSTATE 22P02 `invalid cidr value ... Value has bits set to right of mask`). `net.ParseCIDR` 은 `10.20.30.1/24` 를 통과시키므로 400 INVALID_CIDR 도 아니고, 화면(`web/src/pages/cidrRule.ts:104-116`)도 호스트 비트를 **막지 않고 hint 로만** 「대역의 시작 주소는 10.20.30.0/24 입니다.」 를 보여 준다 — 즉 사용자는 화면이 괜찮다고 한 값을 넣고 500 을 받는다. 고치면 400 도 500 도 아닌 **정상 저장**이 되고(사용자가 말한 뜻 그대로 — "10.20.30.1 이 속한 /24"), 화면의 기존 hint 문장과 서버의 실제 저장값이 처음으로 일치한다.

- 수용 기준:
  1) `POST /api/v1/networks` 에 `{"cidr":"10.20.30.1/24"}` 를 보내면 **201** 이 오고, 저장·목록(`listNetworks`)의 `cidr` 이 `10.20.30.0/24` 다. 500 `NETWORK_CREATE_FAILED` 와 pgx 원문(`SQLSTATE`, `invalid cidr value`)이 응답에 나오지 않는다.
  2) 기존 동작이 줄지 않는다: 이미 정규화된 값(`10.20.30.0/24`, `0.0.0.0/0`, `10.20.30.1/32`)은 글자 그대로 저장되고, IPv6 (`2001:db8::1/64` → `2001:db8::/64`)도 같은 규칙으로 저장되며, `net.ParseCIDR` 이 거절하는 값(`10.20.30.0`, `10.20.30.0/33`, `""`)은 **여전히 400 `INVALID_CIDR`** 이다(코드·메시지 문자열 변경 금지).
  3) 테스트가 증명할 것 — 두 층으로 나눠서:
     - **DB 없이 도는 순수 단위 테스트**: 정규화 함수가 입력 표에 대해 `net.ParseCIDR` 의 두 번째 반환값 `(*net.IPNet).String()` 과 **같은 결과**를 낸다(위 2)의 값 전부 포함). 거절 입력에서는 오류를 돌려준다.
     - **진짜 Postgres 를 쓰는 테스트**(= 결함이 실재했다는 증거): `testPool(t)`(`internal/httpapi/integration_test.go:35`) 로 받은 풀에 **원문** `10.20.30.1/24` 를 `INSERT INTO network_ranges(...)` 하면 **실패**하고, 정규화한 `10.20.30.0/24` 는 **성공**한다. 전례는 `internal/httpapi/integration_traffic_class_test.go:41,45`(같은 테이블에 직접 INSERT·DELETE 로 정리)를 그대로 따를 것.
     - 가능하면 한 걸음 더: `&Server{DB: pool}` 로 `s.createNetwork` 를 `httptest` 요청으로 **핸들러째** 호출해 201 과 저장값을 읽는다. `auth.FromContext` 결과를 `p, _ :=` 로 받으므로 principal 없이도 핸들러는 돈다 — 다만 끝에서 `s.audit(...)` 가 `audit_log` 에 쓰므로 **principal 이 비었을 때 FK/NOT NULL 로 깨질 수 있다(미확인)**. 깨지면 `mcp_oauth_test.go` 나 `seed(t, pool)` 의 사용자 전례를 따라 실제 사용자를 하나 두거나, 이 한 걸음만 포기하고 위 두 층으로 남길 것(지어내지 말고 못 한 것은 못 했다고 적을 것).

- 건드릴 파일 (프로덕션 **1파일**):
  - `internal/httpapi/admin.go:createNetwork` — `net.ParseCIDR` 의 두 번째 반환값(`*net.IPNet`)을 버리지 말고 받아, `INSERT INTO network_ranges(name,cidr,description,internal)` 의 `$2` 와 끝줄 `s.audit(..., map[string]any{"cidr": ...})` 에 **정규화된 문자열**을 쓴다. 판정은 같은 파일 안의 아주 작은 헬퍼(예: `func normalizeCIDR(text string) (string, error)`)로 빼서 DB 없이 단위 테스트할 수 있게 한다. `net` 은 이미 import 되어 있다. 400 분기의 코드·메시지(`"INVALID_CIDR"`, `"CIDR is invalid"`)는 그대로.
  - `internal/httpapi/network_cidr_test.go` (신규) — 위 3)의 단위 테스트 + DSN 없으면 `testPool` 이 스스로 skip 하는 DB 테스트. 같은 패키지(`package httpapi`)로.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go build ./...`
  - `go vet ./cmd/... ./internal/...`
  - `go test ./internal/httpapi/ -run 'CIDR|Network' -v` — DSN 이 없으면 DB 부분은 `MOMENTO_TEST_POSTGRES_DSN is not set` 으로 skip 된다. **skip 을 통과로 적지 말 것.**
  - DB 가 올라간다면(compose.yml 에 postgres 가 있다): `MOMENTO_TEST_POSTGRES_DSN='postgres://...?sslmode=disable' go test ./internal/httpapi/ -run 'CIDR|Network' -v` — 고치기 전 **빨간불**(22P02)을 먼저 찍고, 고친 뒤 초록불을 찍을 것.
  - 회귀: `go test ./cmd/... ./internal/...` (DB 통합은 DSN 없으면 skip)
  - **이 과제는 CI 에서 진짜 end-to-end 로 증명된다**: `.github/workflows/ci.yml:17-30` 에 `postgres:17-alpine` 서비스와 `MOMENTO_TEST_POSTGRES_DSN: postgres://postgres:ci@127.0.0.1:5432/momento?sslmode=disable` 가 **이미 있다**. 즉 새 DB 테스트는 CI 에서 실제로 실행된다(이전 회차 프로필이 "DB 통합은 skip" 만 적어 둔 자리 — 그건 로컬 이야기다). 로컬에서 DB 를 못 띄워도 과제는 성립하지만, **PR 본문에 "로컬 skip / CI 에서 실행됨" 을 명시**할 것.
  - web·sdk 는 이번에 손대지 않으므로 돌리지 않아도 된다. 돌렸다면 결과를 그대로 적을 것.

- 위험과 피할 것:
  - **접근 선택을 바꾸지 말 것.** 호스트 비트를 400 으로 **거절**하는 쪽으로 돌아서지 말 것 — 화면(`cidrRule.ts`)이 이미 막지 않는 hint 로 두기로 **의도적으로** 결정했고(그 파일 104-107 주석이 그 이유를 적고 있다), 400 으로 바꾸면 화면이 서버보다 넓어져 이 저장소가 네 회차에 걸쳐 지켜 온 "화면은 서버의 거울" 계약이 반대로 깨진다.
  - `web/src/pages/cidrRule.ts` 를 **이번에 고치지 말 것.** hint 문장은 고친 뒤의 서버 동작과 이미 일치한다(시작 주소를 알려 준다). 문장을 "저장하면 …로 바뀝니다" 로 다듬는 것은 별 회차(차선 참고).
  - `internal/database/migrations/**` 를 건드리지 말 것 — `cidr` 컬럼 타입을 `text`/`inet` 으로 바꾸는 것은 이 과제가 아니다(보호 경로).
  - `internal/auth`, `.github/workflows/**`, `docs/openapi.yaml` 은 손대지 않는다. 다른 핸들러(`deleteNetwork`, `updateNetwork` 류, `trusted_proxy_cidrs` 설정 경로, `segment`/`insight`)로 같은 정규화를 **번지게 하지 말 것** — 운영자 반복 지시: 같은 종류의 파서가 여러 경로에 있으면 통합하지 말고 좁히는 방향으로만 고친다.
  - 감사 기록(`s.audit`)의 `details` 에 **새 원문을 더하지 말 것**. 지금 들어가는 `{"cidr": ...}` 하나를 정규화된 값으로 바꾸는 것까지만. 오류 원문(`err.Error()`)을 `details` 로 옮기지 말 것.
  - 과거 교훈: 중간 계층이 변경을 삼키는 일이 있었다 — 저장만 보고 끝내지 말고 **`listNetworks` 가 돌려주는 값**까지 읽어 확인할 것.
  - 전제를 실행으로 반증할 여지를 열어 둘 것: 혹시 로컬/CI Postgres 에서 `10.20.30.1/24` 원문 INSERT 가 **성공한다면** 이 과제의 전제가 거짓이다. 그때는 억지로 밀지 말고 그 실행 결과를 그대로 적고 차선으로 갈 것.

- 차선 후보: 「보존 정책」 의 비울 수 없는 네 칸을 지우면 `Number("")` 가 0 을 만들어 controlled value 가 "0" 으로 되돌아오는 것을 고친다 (가치 2 / 위험 2 / 작업량 M) — `web/src/pages/AdminPage.tsx` 의 `RetentionAdmin` 네 `Number(event.target.value)`. 2026-10-07 회차가 실제 브라우저에서 이 되돌아옴을 **관측해 기록**해 두었고(`retentionRule.ts` 가 이미 저장을 막는다), 남은 일은 빈 문자열 상태와 PUT payload 를 분리하는 것뿐이다. 검증은 `cd web && npm ci && npm run lint && npm test && npm run build` + `retentionRule.test.mjs` 전례.

## 피할 영역 (머지 대기 중인 변경과 겹친다 — 재구현 금지)
- **`DimensionsAdmin` / `describeDimensionError`**: 2026-10-08 회차(review-pending)가 `adminErrors.ts` 에 `describeDimensionError` 를 넣고 `AdminPage.tsx:3122` 의 원문 Alert 을 바꿨다. pinned main 에는 **없다**(확인: `adminErrors.ts` 의 `export function describe*` 는 User·Site·Network·EventDefinition·Retention 다섯 개뿐, 3122 는 아직 `{save.error.message}`). 같은 줄을 다시 건드리면 충돌한다. Custom Dimension helperText 아이디어도 같은 이유로 이번 회차 제외.
- **`README.md` 개발 명령 블록**: 2026-10-09 회차(review-pending)가 두 줄을 서브셸로 감쌌다. 건드리지 말 것.

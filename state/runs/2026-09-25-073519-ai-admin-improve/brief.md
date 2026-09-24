- 과제: `loadGrants`가 SQL의 `ORDER BY`를 map 순회로 버려 `roles`·`permissions` 배열 순서가 요청마다 뒤바뀌는 문제 수정 (가치 2 / 위험 1 / 작업량 S)
- 왜: `internal/auth/auth.go:297` `loadGrants`의 조회는 `ORDER BY r.code,p.code`로 정렬해 받아오지만, 곧바로 `roles`/`permissions` map에 넣고 `for role := range roles` map 순회로 슬라이스를 만들어 Go의 무작위 map 순회 순서가 그대로 응답 배열 순서가 된다. 그래서 같은 사용자가 `/api/v1/auth/me`를 두 번 불러도 `roles`·`permissions` 배열 순서가 달라져 응답이 불안정하고(캐시·스냅샷 비교·문서화된 계약 모두 불가), 이미 SQL이 정렬 비용을 치르고 있는데 그 결과만 버리고 있다.
- 수용 기준: 1) `/api/v1/auth/me`가 같은 사용자에 대해 여러 번 호출돼도 `roles`·`permissions`가 항상 코드 오름차순(사전순)으로 같은 순서를 낸다. 2) 세션 로그인·API 키 인증 등 `loadGrants`를 쓰는 모든 경로에서 권한 판정 동작(`principalHasRole`·권한 검사)이 이전과 동일하다 — 순서만 바뀌고 집합은 바뀌지 않는다. 3) 테스트가, 여러 역할·여러 권한을 가진 사용자로 `/api/v1/auth/me`를 반복 호출(최소 10회)해 매번 정렬된 동일 배열이 나오는 것과, 중복 권한(두 역할이 같은 permission을 가진 경우)이 한 번만 나오는 것을 증명한다.
- 건드릴 파일:
  - `internal/auth/auth.go:297 loadGrants` — `roles`/`permissions` map을 "이미 본 값" 집합으로만 쓰고, `rows.Next()` 루프 안에서 처음 본 값일 때 바로 `p.Roles`/`p.Permissions`에 append 한다(SQL이 이미 `ORDER BY r.code,p.code`이므로 정렬이 보존됨). 루프 뒤의 두 `for … range map` 블록은 삭제. 슬라이스가 nil로 남는 기존 동작(역할이 없는 사용자)은 그대로 둘 것 — JSON에서 `null`이었다면 `[]`로 바꾸지 말 것(응답 계약 변경이 됨).
  - `docs/api.md` — `/api/v1/auth/me` 응답 절에 "`roles`·`permissions`는 코드 오름차순으로 정렬된다" 한 줄 계약 추가. (이 저장소 관례상 계약 변경은 api.md에 적는다.)
  - 새 테스트 파일 `internal/server/auth_me_order_integration_test.go`(이름은 자유). 셋업은 `internal/server/request_id_integration_test.go:25-45`를 그대로 베낄 것: `TEST_POSTGRES_DSN` 없으면 `t.Skip` → `database.Open` → `DROP SCHEMA IF EXISTS ai_admin CASCADE` → `db.Migrate` → `db.Seed(ctx, "admin@example.com", "integration-password")` → `secrets.New(bytes.Repeat([]byte{7}, 32))` → `New(...).Handler()`로 실제 서버를 세워 로그인 후 `GET /api/v1/auth/me`를 친다(운영자 규칙: 수동 주입·대역이 아니라 프로덕션 배선을 지나는 테스트).
    테스트 재료: `Seed`는 `super_admin` 역할에 `ai_admin.permission`의 **모든** 코드를 넣으므로(`internal/database/database.go:213-218`) 시드 관리자 한 명만으로도 permission이 여러 개여서 정렬 검증이 의미 있다. 역할 순서와 중복 제거까지 보려면 그 관리자에게 `user_role` 행을 하나 더(예: `team_lead`) 직접 INSERT 해 역할 2개 + 겹치는 permission 상황을 만들 것.
- 검증 명령:
  - 전용 폐기 PostgreSQL을 띄운다(이 머신은 55432·55433·55439·15434가 이미 점유돼 있으니 **다른 포트**를 쓸 것):
    `docker run -d --rm -e POSTGRES_PASSWORD=pass -p 55451:5432 --name aiadmin-scout postgres:16-alpine`
    `export TEST_POSTGRES_DSN='postgres://postgres:pass@127.0.0.1:55451/postgres?sslmode=disable'`
  - `go test -run TestAuthMe -v -count=1 ./internal/server/` — **SKIP이 아니라 PASS**인지 눈으로 확인할 것(DSN 없으면 통합 테스트는 조용히 SKIP된다).
  - `go test -race -count=1 ./...` (internal/server 약 70~90초)
  - `make lint` (`gofmt -l`·`go vet`·`scripts/verify-version.sh`), `go build ./...`
  - 웹 변경이 없으므로 `npm test`·`npm run build`는 불필요.
- 위험과 피할 것:
  - `internal/auth`는 위험 구역이다. **`loadGrants` 안의 슬라이스 조립 부분만** 고치고 세션 생성(`CreateSession`)·`Authenticate`·API 키 경로·SQL 문자열 자체는 건드리지 말 것.
  - SQL의 `ORDER BY r.code,p.code`는 `p.code`가 NULL일 수 있다(LEFT JOIN). PostgreSQL 기본 `ORDER BY ASC`는 NULL을 마지막에 둔다 — permission이 없는 역할 행이 섞여도 role 순서는 유지되지만, 구현자는 `permission != nil` 검사를 그대로 유지할 것.
  - 순서를 "아무 순서나 고정"이 아니라 **SQL이 이미 내는 정렬을 보존**하는 방식으로 고칠 것. Go 쪽에서 `sort.Strings`를 따로 부르는 방법도 동작하지만, 그러면 SQL 정렬과 Go 정렬 두 계약이 생긴다(운영자가 되풀이한 "같은 값을 읽는 두 경로의 계약 불일치" 교훈). 하나만 남길 것.
  - 권한 판정 로직(`principalHasRole`, `Principal.HasPermission` — `auth.go:39`의 `for _, value := range p.Permissions`)은 집합 포함 검사라 순서에 의존하지 않는다. 기존 Go 테스트 중 `roles`/`permissions` 배열의 인덱스나 순서를 단언하는 것은 확인한 범위에서 없었고(`internal/server/*_test.go` grep), 웹(`web/src/App.test.tsx` 등)도 순서에 의존하지 않는 mock이다 — 그래도 전체 `go test -race`로 회귀를 확인할 것.
  - `/api/v1/auth/me`는 `auth_handlers.go:80 me`가 `Principal`을 그대로 직렬화한다(`writeData(w, 200, map[string]any{"user": p})`). 즉 `loadGrants`의 출력이 곧 응답 순서다 — 핸들러 쪽에 정렬을 넣지 말 것.
  - 통합 테스트는 `DROP SCHEMA … CASCADE`를 하므로 운영·공유 DB 금지, 같은 DB로 병렬 실행 금지.
  - VERSION·CHANGELOG·`.github/workflows`·`internal/ui/dist`·migrations는 건드리지 말 것.
- 차선 후보: `decideApproval`이 `decision_comment`를 trim·길이 제한 없이 저장 — `internal/server/workflow.go:283`은 `approval_action.comment`에 `nullString(strings.TrimSpace(request.Comment))`를 넣는데, 같은 트랜잭션의 `:290`·`:292`는 `approval_request.decision_comment`에 **원문 그대로** 넣는다. 같은 결정의 두 기록이 공백 처리에서 달라지고, 두 컬럼 모두 `text`라 길이 상한이 전혀 없다(migrations/001_ai_admin.sql:169,186). 고칠 방향: trim한 값을 한 번 계산해 두 곳에 같이 쓰고, `len([]rune(x)) > 2000` 같은 rune 기준 상한을 요청 검증에 추가해 400 `comment_too_long`으로 거부(이 저장소의 길이 상한은 바이트가 아니라 rune 기준이다). 단 `workflow.go`는 승인 트랜잭션이라 위험 구역이므로, 위 1순위가 성립하지 않을 때만 고를 것.

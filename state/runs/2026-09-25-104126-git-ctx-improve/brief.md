- 과제: `mcp.cacheKey` 가 호출자의 ACL 주체 슬라이스를 제자리 정렬하지 않게 복사 (가치 2 / 위험 1 / 작업량 S)
- 왜: `internal/mcp/cache.go:32-33` 이 `principals := principalACLs(p); sort.Strings(principals)` 로 호출자의 슬라이스를 **제자리** 정렬하는데, 바로 다음 줄(`cache.go:34`)은 같은 목적의 `p.AllowedRepositories` 를 `append([]string(nil), ...)` 로 복사한 뒤 정렬한다 — 같은 함수 안의 이 비대칭이 누락의 증거다. 아래 "확인한 별칭 사슬" 대로 `principalACLs` 는 흔한 경우(제한 있는 일반 사용자) 에 `p.ACLPrincipals` 의 백킹 배열을 그대로 돌려주므로, 캐시 키를 만드는 부수효과로 그 요청의 나머지 구간(도구 핸들러가 `principalACLs(p)` 로 다시 읽는 값, `freshnessNote` 의 `IndexAges`)이 auth 가 만든 순서와 다른 순서를 보게 된다. 키 계산 함수가 입력을 바꾸지 않게 하면 이 무성 부수효과가 없어진다.

## 확인한 별칭 사슬 (모두 이번에 직접 읽음)
- `internal/mcp/cache.go:30-38` `cacheKey` — `principals := principalACLs(p)` → `sort.Strings(principals)` (복사 없음). 34번 줄 `repositories := append([]string(nil), p.AllowedRepositories...)` 는 복사함.
- `internal/mcp/args.go:167-176` `principalACLs` — `len(p.ACLPrincipals) > 0` 이면 `principals = p.ACLPrincipals` (그 슬라이스 자체), 마지막에 `return search.WithUnrestricted(principals, p.Roles)`.
- `internal/search/service.go:230-235` `WithUnrestricted` — `GrantsUnrestrictedSearch(roles)` 가 false 면 **`principals` 를 그대로 반환**(복사 안 함). true 면 새 슬라이스를 만들어 반환하므로 그때는 무해하다. 즉 제한 있는 일반 사용자 경로에서만 별칭이 생긴다.
- `internal/mcp/dispatch.go:98-123` 순서 — 예산 계산 → `cacheKey`/`cached` → 도구 핸들러 → `finishCall`. 따라서 정렬은 핸들러가 `principalACLs(p)`(`internal/mcp/tools.go` 40여 곳)를 호출하기 **전에** 일어난다.
- 주체 원본: `internal/app/auth.go:121`(API 키 경로)·`:160`(Keycloak 토큰 경로)이 `sourceACLPrincipals(...)`(`auth.go:937-951`) 결과를 `auth.Principal.ACLPrincipals` 에 넣어 컨텍스트에 싣는다. **확인함**: `sourceACLPrincipals` 는 `var out []string` 에서 시작해 append 로만 채우므로 요청마다 새 백킹 배열이다 — 따라서 요청 간 공유나 경합은 **없고**, 영향은 한 요청 안에서의 순서 변경으로 한정된다. 이 과제를 데이터 경합으로 포장하지 말 것.
- **미확인(과장 금지)**: 현재 소비자 중 주체 **순서**에 의존하는 곳은 찾지 못했다(ACL 은 SQL `IN (...)` 로 쓰여 순서 무관). 따라서 "사용자에게 보이는 잘못된 출력"을 증명하는 테스트는 기대하지 말 것. 이 과제의 증명 대상은 **함수가 입력을 변경한다는 사실 그 자체**다. 없는 사용자 피해를 만들어 쓰지 말 것.

- 수용 기준:
  1) 실제 `store.Open("sqlite", …)` 로 만든 실제 `mcp.Server` 와 실제 `auth.Principal`(손으로 만든 대역 타입이 아니라 프로덕션 타입)로 `s.cacheKey(ctx, p, "search-code", args)` 를 호출한 뒤, 호출자가 넘긴 `p.ACLPrincipals` 의 **순서와 내용이 그대로**임을 단언한다. 수정 전 이 단언이 실패하는 것을 직접 확인하고 실패 출력을 기록한다.
  2) 캐시 키 자체의 동작은 불변: 같은 주체 집합을 순서만 다르게 준 두 호출이 **같은 키**를 만든다(정렬의 본래 목적). 이 단언도 함께 둔다.
  3) `GrantsUnrestrictedSearch` 가 true 인 역할(복사 경로)에서도 1)·2)가 성립한다 — 두 분기 모두 테이블 테스트로 덮는다.
  4) `internal/mcp` 의 기존 테스트가 모두 그대로 통과한다(캐시 히트/미스 동작 회귀 없음).

- 건드릴 파일:
  - `internal/mcp/cache.go:32-33` `cacheKey` — `principals := append([]string(nil), principalACLs(p)...)` 로 복사한 뒤 정렬(34번 줄과 같은 관용구로 맞출 것). 이것이 최소 수정이며 권장안.
  - `internal/mcp/cache_test.go` — 위 수용 기준의 테이블 테스트 추가.
  - 선택(구현자 판단): `internal/search/service.go:230-235` `WithUnrestricted` 가 항상 복사해 돌려주게 하는 쪽이 모든 호출자를 한 번에 막는다. 다만 이 함수는 `internal/app/health.go:616`·`internal/app/auth.go:923` 도 쓰므로 **바꾸려면 세 호출자 모두 확인**하고, 확신이 없으면 `cache.go` 한 줄만 고치고 끝낼 것. 두 곳을 동시에 고쳐 한쪽만 맞는 상태를 만들지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것, 이번에 돌려 확인함):
  - `go test -tags sqlite_fts5 -count=1 ./internal/mcp`
  - `go test -tags sqlite_fts5 -race -count=1 ./internal/mcp`
  - `gofmt -l ./cmd ./internal` (출력이 비어야 성공)
  - `go vet ./...`
  - `go build -tags sqlite_fts5 ./...`
  - `go test -tags sqlite_fts5 ./...` (수 분; `./internal/app` 혼자 ~100초)

- 위험과 피할 것:
  - `internal/auth`·`internal/app` 의 인증/세션/ACL 판정, `internal/store` migration, `.github/workflows`, `internal/version` 은 건드리지 말 것. 이 과제는 `internal/mcp` 안에서 끝난다.
  - `sort.Strings` 자체를 없애지 말 것 — 정렬은 순서가 달라도 같은 캐시 키를 만들기 위한 것이다(수용 기준 2 가 이것을 고정한다).
  - 과거 교훈: 손으로 만든 대역으로 결함을 증명하지 말 것. `s.search` 는 구체 타입이라 스파이를 끼울 수 없다 — 억지로 대역을 넣으려 하지 말고, 호출자 슬라이스를 직접 단언하는 방식으로 끝낼 것.
  - 과거 교훈: 같은 값을 읽는 경로가 둘 이상이면 한쪽만 고치지 말 것 — 그래서 `WithUnrestricted` 수정은 "전부 확인하든 아예 손대지 말든" 둘 중 하나로만 할 것.
  - `mcp` 테스트 fixture 는 공유 in-memory SQLite 이름을 쓴다(프로필 기록). 새 테스트를 `t.Parallel()` 로 만들지 말 것.
  - 마스킹 규칙(`contentsecurity.Revision`)과 무관한 변경이므로 재색인 지문은 건드리지 않는다 — Revision 을 바꾸지 말 것.

- 차선 후보: `responseNoticeBytes = 320`(`internal/mcp/budget.go:70`)이 실제 절단 공지보다 짧을 수 있어 `clampResponse` 결과가 예산을 넘는 것 — 공지 문자열 3줄을 손으로 세어 보니 현실적 숫자(budget 24,576 / produced 1,048,576 / 123 of 4567)에서 약 319B 로 320 에 거의 붙어 있고, 예산 상한(262,144)·더 큰 produced 자릿수에서 320 을 넘긴다. 다만 초과분이 몇 바이트 규모라 가치가 낮고, `internal/mcp/truncation_test.go:31` 이 이미 `budget+responseNoticeBytes` 를 허용하고 있어 느슨한 계약이 의도인지 먼저 확인해야 한다. 물리적으로 큰 초과는 `finishCall`(`dispatch.go:376-381`)이 `clampResponse` **뒤에** `argumentNotes + freshnessNote` 를 덧붙이는 쪽인데, 거기에는 "예산이 노트를 잘라선 안 된다"는 의도가 주석으로 명시돼 있으므로 "버그"로 다루지 말고 "노트 길이를 먼저 빼서 예산 안에 넣기"로만 접근할 것.

- 과제: 승인 레코드 인자 스크럽을 중첩 map/slice 까지 내리고, 2000바이트 절단을 UTF-8 경계로 맞춘다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/approval/approval.go:redactArgs` (271행) 는 **최상위 키와 최상위 string 만** 검사한다 — 중첩 값은 `out[k] = v` 로 원문 그대로 통과하므로, `{"body":{"password":"…"}}` 같은 인자가 `approval_requests.arguments_redacted` 에 평문 저장되고 `GET /api/admin/approvals` 로 관리 콘솔에 그대로 나간다. 같은 함수의 `s[:2000]` 는 바이트 오프셋 절단이라 한국어 인자의 마지막 글자를 반쪽으로 잘라 `json.Marshal` 이 U+FFFD 로 치환한다(2026-10-06 에 `internal/mcp/server.go:trimToRune` 로 닫은 것과 같은 결함이 여기에 남아 있다).

- 수용 기준:
  1) 중첩 map 안의 민감 키(`sensitiveKeyParts` 목록)가 `[redacted]` 로 저장된다 — `{"body":{"password":"p"},"list":[{"token":"t"}]}` 가 DB 왕복 후 둘 다 `[redacted]`.
  2) 2000바이트를 넘는 한국어 string 인자가 절단될 때 결과에 U+FFFD 가 생기지 않는다(절단 지점이 rune 경계). **2026-10-06 교훈: 한국어는 3바이트이므로 1바이트씩 밀린 ASCII 접두사 3종(`""`/`"A"`/`"AA"`)으로 오프셋 2000 의 mod 3 잉여류를 모두 덮을 것** — 하나만 쓰면 운 좋게 경계에 맞아 수정 전에도 통과한다(그때 3종 중 1종이 그렇게 통과했다).
  3) 기존 동작 회귀 없음: 최상위 민감 키 스크럽(2026-10-04 회차의 8개 키 테스트)과 `Hash(toolName, args)` 값이 바뀌지 않는다 — **해시는 원본 `args` 로 계산되므로 `redactArgs` 는 입력 map 을 변형(mutate)하면 안 된다.**
  4) 재귀 깊이 상한(권장 8)과 요소 수 상한을 두어, 깊게 중첩된 악의적 인자가 스택을 태우지 않는다. 상한 초과 시 그 하위 값은 `"[redacted]"` 로 대체한다(통과시키지 말 것).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/approval/approval.go:redactArgs` — `map[string]any` / `[]any` 를 재귀 순회하는 헬퍼(예: `redactValue(key string, v any, depth int) any`)로 바꾼다. 규칙: ① 맵 키가 `containsAny(lower(k), sensitiveKeyParts)` 면 값 전체를 `"[redacted]"`, ② string 이고 2000바이트 초과면 rune 경계로 트림 후 `…`, ③ map/slice 면 깊이+1 로 재귀, ④ 그 외는 그대로. 입력 map 은 수정하지 말고 새 map/slice 를 만들 것.
  - rune 경계 트림은 `internal/mcp/server.go:trimToRune` 과 같은 방식(`utf8.DecodeLastRuneInString` 역방향)으로 **approval 패키지 안에 따로 작성**한다 — mcp 패키지를 import 하거나 공용 util 패키지를 새로 만들지 말 것(운영자 지시: 같은 종류의 파서를 통합하려 하지 말고 좁히는 방향으로만 고칠 것).
  - `internal/tools/integration_test.go` — 2026-10-04 회차가 만든 최상위 스크럽 테스트(`Arguments["apiKey"]` 를 단정하는 그 테스트, 460행 부근) 옆에 중첩 케이스와 한국어 절단 케이스를 추가한다. 실제 프로덕션 배선 하네스 `newFixture` 로 `Invoke` → `Approvals.Create` → PostgreSQL → `ByID`/`scan` 왕복을 거친 `Request.Arguments` 를 검사할 것. 새 가짜 Engine/Executor 를 만들지 말 것.

- 검증 명령:
  ```
  docker start bbmcp-test-pg
  export TEST_DATABASE_URL='postgres://bbmcp:bbmcp@127.0.0.1:15532/bbmcp_test?sslmode=disable'
  go build ./... && go vet ./...
  go test ./internal/tools -count=1 -p 1 -v     # red 를 먼저 만들 것
  go test ./... -count=1 -p 1 -v                # 9개 패키지 ok, --- SKIP 0건을 눈으로 확인
  ```
  TEST_DATABASE_URL 이 없으면 통합 테스트가 조용히 `t.Skip` 된다 — exit 0 을 통과로 읽지 말 것.
  **이번 정찰 세션에서는 `go build` 실행이 샌드박스 승인에 막혀 명령을 돌려 보지 못했다(2026-10-16 회차와 동일).** 아래 근거는 전부 코드 판독이며, 구현자는 수정 전에 red 로그를 먼저 만들어 결함을 런타임으로 확정할 것.

- 정찰이 코드로 직접 확인한 것(행 번호는 main@6532321 기준):
  - `internal/approval/approval.go:270-285` `redactArgs` — `out[k] = v` 로 중첩 값 무검사 통과. 최상위 string 만 `len(s) > 2000` 에서 `s[:2000]` 바이트 절단.
  - `internal/approval/approval.go:118,125` — `redacted := redactArgs(args)` 와 `Hash(toolName, args)` 가 **같은 map** 을 본다(해시는 원본).
  - `internal/approval/approval.go:98-108` `scan()` — `arguments_redacted` 를 `r.Arguments` 로 역직렬화.
  - `internal/api/admin.go:60, 1242` — `GET /approvals` → `s.Approvals.List(...)` → 그 `Arguments` 가 JSON 으로 그대로 응답에 실린다.
  - `internal/tools/args.go:13` `type Args map[string]any`, `internal/tools/executor.go:341` — 호출자 인자가 필터 없이 `Approvals.Create` 로 간다(기존 테스트 `TestApprovalRecordRedactsSensitiveArgumentNames`, integration_test.go:427 이 이 사실을 이미 증명한다).
  - `internal/mcp/server.go:335` `trimToRune` — 베껴 쓸 참조 구현(`utf8.DecodeLastRune` 역방향 트림).
- 미확인(추측): `json.Marshal` 이 깨진 바이트를 U+FFFD 로 치환해 저장 값이 손상된다는 부분은 2026-10-06 회차에서 같은 패턴으로 런타임 확인된 사실에서 유추했을 뿐, 이번에 `approval` 경로로 재현하지는 않았다.

- 위험과 피할 것:
  - **`Hash` 와 `redactArgs` 를 같이 건드리지 말 것.** `Create` 는 `Hash(toolName, args)` 에 **원본** args 를 넘긴다(approval.go:125). 해시가 바뀌면 `Check` 의 인자 해시 비교가 깨져 기존 승인이 전부 무효가 된다. `redactArgs` 가 입력 map 을 in-place 로 고치면 같은 사고가 난다.
  - `internal/audit` 의 `redactKeys` 는 베끼지도 통합하지도 말 것 — 2026-10-04 회차가 의도적으로 분리해 두었다(승인자는 diff/content/text 본문을 봐야 한다).
  - `approval.go:Check` 의 만료·인자 해시·PR 버전 nil 거부·조건부 UPDATE 단일 소비는 회귀 금지선이다. 이번 변경은 `Create` 경로의 표시용 필드만 건드린다.
  - `internal/auth/*`, `internal/api/oauth.go`, migrations, `.github/workflows` 는 건드리지 말 것.
  - 테스트는 `t.Parallel()` 금지, `-p 1` 필수(fixture 가 공유 테이블을 TRUNCATE 한다).

- 차선 후보: `internal/permission/resolver.go` 캐시·fail-closed 경로 통합 테스트 (가치 4 / 위험 1 / 작업량 S) — `internal/tools/integration_test.go:newFixture` 가 `CacheTTLSec=0` 으로 돌아 Resolver 의 TTL 캐시 경로가 한 번도 실행되지 않는다. fixture 전역 TTL 을 바꾸면 다른 테스트의 권한 변경 시나리오가 깨지므로, TTL 을 받는 전용 fixture 옵션을 추가해 그 테스트에서만 `CacheTTLSec=60` 으로 띄우고 (캐시 적중 → 권한 회수 후에도 TTL 내 적중 → `Reset()` 후 재조회 거부) 흐름을 고정할 것.

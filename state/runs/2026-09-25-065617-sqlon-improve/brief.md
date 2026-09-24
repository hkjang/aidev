- 과제: 비동기 쿼리 제출 `POST /api/query/submit` 이 `binds` 를 받지 않아 동기 경로와 실행 계약이 어긋나는 결함 수정 (가치 4 / 위험 2 / 작업량 S)

- 왜: `internal/mcp/dbapi.go` 의 동기 실행 핸들러(`execute` 클로저, 197~245행)는 요청의 `Binds []any \`json:"binds"\`` 를 받아 `dbconn.ExecOptions{... Binds: req.Binds ...}` 로 넘기고 `internal/dbconn/manager.go:397` 이 `db.QueryContext(qctx, limited, opts.Binds...)` 로 드라이버에 그대로 전달한다. 그런데 비동기 제출 핸들러(dbapi.go:802~840)의 요청 구조체에는 `binds` 필드 자체가 없고 `dbconn.ExecOptions{MaxRows, TimeoutSeconds, User}` 만 만들어 `submitAsyncQuery` 에 넘긴다 — 즉 같은 플레이스홀더 SQL 을 `/api/query` 로 보내면 바인드되어 실행되고 `/api/query/submit` 으로 보내면 인자 0개로 실행되어(드라이버가 거절하거나, 최악의 경우 의도와 다른 SQL 이) 돈다. 고치면 두 실행 경로가 같은 요청 스키마·같은 실행 계약을 갖게 되고, 긴 분석 쿼리를 비동기로 돌릴 때만 파라미터를 문자열로 이어 붙이는(SQL 인젝션을 부르는) 우회가 필요 없어진다.

- 수용 기준:
  1) `POST /api/query/submit` 이 `binds` 배열을 받아들이고, 그 값이 그대로 잡 실행에 쓰이는 `dbconn.ExecOptions.Binds` 에 들어간다. `binds` 를 생략한 기존 요청은 지금과 똑같이 동작한다(nil Binds, 202 응답 본문 무변화).
  2) 동기 `POST /api/query` 와 비동기 `POST /api/query/submit` 이 같은 JSON 본문(`profile_id`/`sql`/`max_rows`/`timeout_seconds`/`binds`)에 대해 같은 `ExecOptions.Binds` 를 만든다. `User` 는 지금처럼 양쪽 모두 인증 사용자 이름으로 강제되어야 하고(비동기는 `actorName(actor)`), 요청 본문의 `user` 를 비동기가 새로 신뢰하게 만들지 말 것.
  3) 테스트가 **수정 전에 실패**해야 한다: 실제 로그인 쿠키 → 실제 `Register` mux → `POST /api/query/submit {"binds":[...]}` 를 통과시킨 뒤, 그 잡이 받은 `ExecOptions.Binds` 가 요청의 binds 와 같음을 단언한다. 현 HEAD 에서는 nil 이라 RED 가 난다.
     - 관측 방법(권장): `internal/mcp/asyncquery.go` 의 `asyncJob` 에 **직렬화되지 않는 비공개 필드** `opts dbconn.ExecOptions` 를 추가하고 `submitAsyncQuery` 가 그것을 채우게 한 뒤, 테스트가 `s.asyncJobs` 잠금 안에서 그 값을 읽는다. 이 값은 `executeGuarded` 에 실제로 넘어가는 바로 그 구조체여야 한다(따로 복사본을 만들어 테스트용으로만 저장하면 안 됨). `asyncJob` 의 JSON 응답 필드는 절대 늘리지 말 것 — `jobView` 응답 계약은 그대로여야 한다.
     - 주의: 이 저장소에는 테스트용 `database/sql` 드라이버가 등록되어 있지 않고(`grep -rn "sql.Register" .` 결과 없음) `internal/dbconn/driver.go` 는 `pgx`/`mysql` 이름만 연다. 그래서 일반 `go test` 로는 "바인드가 드라이버까지 갔다" 를 끝까지 볼 수 없다. **소스 문자열 검사로 대신하지 말고** 위의 프로덕션 배선 경로(실제 HTTP → 실제 핸들러 → 실제 잡 저장소)로 증명할 것. 도커가 있으면 `test/integration` 에 `m.Execute(ctx, "pg-meta", "SELECT $1::int", dbconn.ExecOptions{Binds: []any{7}})` 류의 실 DB 검증을 **추가로** 붙이면 더 좋다(필수 아님).

- 건드릴 파일:
  - `internal/mcp/dbapi.go` — `POST /api/query/submit` 핸들러(802행부터). 요청 구조체에 `Binds []any \`json:"binds"\`` 추가, `dbconn.ExecOptions{MaxRows: req.MaxRows, TimeoutSeconds: req.TimeoutSeconds, User: actorName(actor), Binds: req.Binds}` 로 전달. 그 위의 `s.cat().ValidateSQL(...)` 호출·403/400 검사·`adminAudit` 문자열은 손대지 말 것(감사 details 에 SQL 이나 binds 원문을 절대 넣지 말 것 — 지금처럼 job ID·profile ID·사용자만).
  - `internal/mcp/asyncquery.go` — `asyncJob` 에 비공개 `opts` 필드 + `submitAsyncQuery` 에서 대입(수용 기준 3의 관측점). 기존 `jobResultTTL`/`jobMaxPerUser`/`jobMaxTotal`/`prune`/`cancelJob` 로직은 건드리지 말 것.
  - `internal/mcp/openapi.go:570` — `/api/query/submit` requestBody 스키마에 `"binds":{"type":"array","items":{}}` 추가(동기 `/api/query` 스키마와 같은 표기를 따를 것).
  - `internal/mcp/execguard_test.go` 또는 `internal/mcp/dbapi_test.go` — 회귀 테스트. 기존 `TestAsyncJobLifecycleWithStubDriver`(execguard_test.go:80)가 `newFixtureServer`(datasets_test.go:15)로 잡 저장소를 다루는 방식이 출발점이고, 실제 로그인→mux 패턴은 기존 인증 테스트(`internal/mcp/authhttp_test.go`, `dbapi_test.go`)의 헬퍼를 재사용할 것.
  - `CHANGELOG.md` — Unreleased 에 한 줄. **파일 전체 줄바꿈을 바꾸지 말 것**(이 파일은 LF/CRLF 혼합이라 편집 도구가 전체를 뒤집으면 100줄짜리 diff 가 난다). 저장 후 `git diff --stat CHANGELOG.md` 로 1~2줄만 바뀌었는지 확인.

- 검증 명령:
  - `go test ./internal/mcp -count=1` (기준선 확인함: 이번 회차 HEAD 57f99b7 에서 `ok sqlon/internal/mcp 3.642s`)
  - `go test ./internal/mcp -run 'TestAsync' -count=1 -race`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go build ./...`
  - `gofmt -l internal/mcp/dbapi.go internal/mcp/asyncquery.go internal/mcp/openapi.go` (손댄 파일만 — 저장소의 다른 ~90개 Go 파일은 CRLF 라 원래 걸린다)
  - `git diff --check`
  - (선택, 도커 있을 때만) `docker compose -f deploy/test/docker-compose.yml up -d --wait` 후 `go test -tags integration ./test/integration -v`

- 위험과 피할 것:
  - **결과 캐시와의 상호작용이 이번 회차의 가장 큰 함정이다.** 현 HEAD 의 `internal/mcp/execguard.go:88` `cacheKey(profile, sql string, maxRows int)` 는 binds 를 키에 넣지 않는다. 비동기는 `executeGuarded(ctx, profile, sql, opts, true)` 로 캐시 **읽기만** 건너뛰고 `put` 은 그대로 하므로(execguard.go:145), binds 를 붙이면 "binds=[1] 로 돌린 비동기 결과" 가 binds 를 무시한 키로 캐시에 올라가 동기 경로가 다른 binds 로 그걸 읽을 수 있다. 다만 이 캐시 키 결함 자체는 **2026-09-24 회차에서 이미 성공적으로 고쳐진 과제**(커밋 57563cf, 이 브랜치에 미통합)이므로 **다시 구현하지 말 것**. 이번 과제는 제출 핸들러의 binds 누락만 고치고, 캐시 키 문제는 별건으로 남긴다. 테스트를 쓸 때 캐시가 결과를 가로채 RED/GREEN 판정을 흐리지 않도록, 관측은 캐시 이전 지점(`ExecOptions`)에서 할 것.
  - `requireQueryActor`(dbapi.go:875 부근)·`canUseProfileID`·`ValidateSQL` 게이트 순서를 바꾸지 말 것. 특히 `canUseProfileID(ctx, nil, ...)` 의 로컬 신뢰 규칙을 전역으로 손대지 말 것(과거 회차의 상습 지뢰).
  - 보호 경로 회피: `internal/mcp/auth.go`·`authapi.go`·`internal/meta/pg.go`(마이그레이션 포함)·`.github/workflows` 는 이번 과제에서 열 필요가 없다.
  - 저장소 전체 `gofmt`/줄바꿈 일괄 변환 금지.
  - 미확인으로 남긴 것: 웹 UI(`internal/mcp` embed HTML)가 비동기 제출 폼에서 binds 를 보내는지 확인하지 않았다. UI 를 바꿀 필요는 없다(서버가 생략을 그대로 허용하므로 호환된다).

- 차선 후보: **빈 `docs/README.md` 를 실제 문서 색인으로 채우기** (가치 3 / 위험 1 / 작업량 S) — `docs/README.md` 는 0바이트이고 루트 `README.md` 가 가리키는 대상이다. 비어 있지 않은 가이드만 링크하고, 내용이 v0.1.2 기준임을 명시할 것. 그 다음 후보는 **관리자 가이드 §3.2 의 DB 프로파일 생성 curl 을 실제 `POST /api/db-profiles` 계약으로 교정**(가치 3 / 위험 1 / 작업량 S) — 가이드는 `POST /api/fleet/instances` 를 안내하지만 그 라우트는 GET 만 존재한다.

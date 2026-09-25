- 과제: `GET /api/metrics` 가 인증 없이 DB 프로파일 인벤토리(pools·breakers 키)를 노출하는 결함 수정 (가치 4 / 위험 2 / 작업량 S)
- 왜: `internal/mcp/dbapi.go:870` 의 `GET /api/metrics` 핸들러는 요청을 아예 무시(`func(w http.ResponseWriter, _ *http.Request)`)하고 `s.DB.Snapshot()` 을 그대로 돌려준다 — 같은 파일의 다른 모든 운영 조회 엔드포인트(`GET /api/db/alerts` dbapi.go:752-753, `GET /api/query/history` dbapi.go:759-760)가 `requireActor` 로 막혀 있는데 이 하나만 게이트가 없다. `Snapshot()`(internal/dbconn/manager.go:741-780)의 `pools` 와 `breakers` 는 **프로파일 ID 를 맵 키로** 담으므로, 메타 DB 모드 배포에서 미인증 요청자가 비공개 프로파일까지 포함한 설정된 DB 프로파일 목록과 풀 사용량·서킷브레이커 상태·쿼리 실패 카운터를 받아 간다.
- 수용 기준:
  1) 메타 DB 모드(`authEnabled()==true`)에서 쿠키·키 없는 `GET /api/metrics` 가 401 이고 본문에 `pools`/`driver_available` 이 없다. 유효하지 않은 세션 쿠키도 401.
  2) 로그인한 일반(비관리자) 사용자의 `GET /api/metrics` 는 여전히 200 이고 `driver_available` 을 담는다 — 즉 이 수정은 인증만 요구하고 역할을 새로 요구하지 않는다(관리자 전용으로 바꾸지 말 것).
  3) 단독(마스터 토큰) 모드의 기존 동작이 보존된다: `internal/mcp/dbapi_test.go:104` 의 무헤더 `GET /api/metrics` 200 단언이 수정 없이 통과한다(같은 테스트가 이미 `requireActor` 로 막힌 `POST /api/query/validate` 를 무헤더로 통과시키므로 이 모드에서 게이트는 통과해야 한다). `AdminToken` 이 설정된 단독 모드에서는 토큰 없는 요청이 거절된다.
  4) 회귀 테스트가 수정 전 코드에서 실패(RED)하고 수정 후 통과(GREEN)함을 커밋 메시지나 회차 노트에 적는다.
- 건드릴 파일:
  - `internal/mcp/dbapi.go:870` — `GET /api/metrics` 핸들러에 `if _, ok := s.requireActor(w, r); !ok { return }` 를 앞세운다. 클로저 시그니처의 `_ *http.Request` 를 `r *http.Request` 로 바꿔야 한다. `requireActor` 가 아니라 `requireAdmin`/`requireQueryActor` 를 쓰지 말 것 — `GET /api/db/alerts`(752) 가 쓰는 것과 같아야 하고, 이 데이터는 웹 UI 의 일반 사용자 화면 성격이다.
  - `internal/mcp/dbapi_test.go` 또는 `internal/mcp` 의 인증 테스트 파일 — 수용 기준 1)·2)·3) 을 덮는 테스트 추가. 기존 `newAuthServer` → 실제 로그인 → `Register` mux → `doReq`/`withCookie` 관례를 따를 것(과거 회차에서 반복 검증된 배선).
  - `internal/mcp/openapi.go:593` (`"/api/metrics"` 항목) — 이 문서가 보안 요구사항을 적고 있다면 실제와 맞춰 갱신한다. **미확인**: 해당 항목에 security 필드가 있는지 열어 보지 않았다. 없으면 건드리지 말 것.
  - `CHANGELOG.md` Unreleased — 한 줄. **줄바꿈 주의**: 이 저장소의 CHANGELOG 는 CRLF/LF 혼합이라 편집 도구가 파일 전체 줄바꿈을 뒤집은 전례가 있다(2026-09-25 회차). 추가 후 `git diff --numstat` 로 추가 줄 수가 실제 추가분과 같은지 확인할 것.
- 검증 명령:
  - `go test ./internal/mcp -count=1` (직전 회차 기준 약 3.7초)
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go build ./...`
  - `gofmt -l internal/mcp/dbapi.go internal/mcp/dbapi_test.go` (무출력이어야 함)
  - `git diff --check`
  - RED 확인: 수정 한 줄을 임시로 되돌려 새 테스트가 실패하는지 본다.
- 위험과 피할 것:
  - **`GET /metrics`(dbapi.go:873, `s.serveMetrics`)는 건드리지 말 것.** 같은 프로파일 ID 를 Prometheus 텍스트로 노출하지만 스크레이퍼는 세션 쿠키를 보낼 수 없어 게이트를 걸면 관측이 끊긴다. 별건으로 남긴다(ideas.json 에 기록됨).
  - 보호 경로 회피: `internal/mcp/auth.go`·`authapi.go`·`internal/meta/pg.go`·`.github/` 를 수정하지 말 것. 이 과제는 `requireActor` 의 **호출부 한 곳**만 추가하며 `requireActor` 자체의 동작은 바꾸지 않는다.
  - `canUseProfileID(nil)` 의 로컬 신뢰 규칙을 전역으로 바꾸지 말 것(과거 교훈).
  - 프로파일 ID 를 스냅샷에서 제거하거나 마스킹하려 들지 말 것 — 응답 스키마를 바꾸면 웹 UI·openapi 계약이 함께 깨진다. 인증만 추가하는 최소 수정이다.
  - **이 브랜치(main@57f99b7)에 미통합인 과거 성공은 재구현 금지**: 이번 회차에 직접 확인한 것 — `internal/mcp/execguard.go:88 cacheKey(profile, sql string, maxRows int)` 는 여전히 binds 를 무시하고(2026-09-24 성공), `POST /api/query/submit`(dbapi.go:807-812)의 요청 구조체에 `binds` 가 없고(2026-09-25 성공), `asyncquery.go` 의 `jobView`·`cancelJob` 은 `prune()` 을 호출하지 않는다(2026-09-21 성공). 세 개 모두 손대지 말 것.
- 차선 후보: 이미 끝난(done/failed) 비동기 잡의 취소가 `200 {"canceled":true}` 를 돌려주는 결함 수정 (가치 2 / 위험 1 / 작업량 S) — `internal/mcp/asyncquery.go:cancelJob` 이 `Status != "running"` 이면 아무것도 하지 않고 `true` 를 반환하고 `dbapi.go:862-866` 이 그대로 `canceled:true` 로 응답한다(소스 확인). 응답에 실제 취소 여부를 담도록 고친다. 주의: `prune()` 을 같이 넣으면 2026-09-21 성공의 재구현이 되므로 넣지 말 것.

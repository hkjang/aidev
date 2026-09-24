# 과제서 — 2026-09-23 (relio, base main@38c88ce)

- 과제: 행 반복이 끝난 뒤 `rows.Err()` 를 검사해 스트림 오류를 조용한 부분 응답으로 삼키지 않기 (가치 4 / 위험 2 / 작업량 M)

- 왜: pgx 의 `rows.Next()` 는 스트림 오류가 나도 `false` 를 돌려주므로, 검사 없이 루프를 빠져나가는 곳은 **끊긴 결과를 200 과 함께 정상인 것처럼** 돌려준다 — `mySessions`(내 세션 목록), `adminAudit`(감사 로그), `adminOrganizations`, 키·VOC 목록, CRM 목록 등 운영자가 "다 봤다" 고 믿는 화면들이다. 이 저장소는 같은 루프의 절반 이상에서 이미 `rows.Err()` 를 검사하므로(예: `internal/crm/service.go:188`, `internal/intelligence/health.go:119`) 고치는 것은 새 관례 도입이 아니라 **빠진 자리를 관례에 맞추는 일**이고, 끝나면 목록 API 가 "짧게 왔다" 와 "여기까지가 전부다" 를 구분해 준다.

- 수용 기준:
  1) `internal/` 의 모든 비테스트 Go 파일에서 `for rows.Next()` 루프가 끝난 직후 그 루프가 쓴 rows 값의 `rows.Err()`(또는 `return out, rows.Err()`) 를 검사한다. 스트림 오류는 기존 Scan 오류와 **같은 경로**로 나간다 — 핸들러는 `s.serviceError(w, r, err)` + return, 서비스 함수는 기존 시그니처 그대로 `err` 반환. 응답 형태·상태 코드 매핑을 새로 만들지 않는다.
  2) 부분 결과를 200 으로 내보내지 않는다: 검사에 걸리면 `items` 를 버리고 오류로 끝낸다(이미 모은 항목을 함께 내보내는 타협 금지).
  3) 새 테스트가 이 불변을 **소스 스캔**으로 증명한다. 이 저장소에는 같은 방식의 선례가 이미 있다 — `internal/crm/search_test.go:43` `TestEveryFreeTextSearchDeclaresItsEscapeCharacter` 가 `filepath.Walk(filepath.Join("..","..","internal"))` 로 `internal/` 전체를 훑어 `LIKE $` 뒤에 `ESCAPE` 가 없으면 `t.Errorf(path:line)` 한다. 새 테스트도 같은 모양으로 쓰고(파일·줄번호를 찍어 어디가 빠졌는지 바로 보이게), **먼저 지금 코드에서 실패(red)하는 것을 확인한 뒤** 각 자리를 고쳐 green 으로 만든다. 고친 자리 하나를 되돌리면 다시 실패해야 한다.
  4) 기존 동작은 그대로다: `go test ./...` 가 변경 없이 통과하고 OpenAPI 계약 테스트도 손대지 않는다.

- 건드릴 파일 (직접 열어 확인한 자리. 줄번호는 base main@38c88ce 기준):
  - `internal/server/sessions.go:33` `mySessions` — 루프 직후 바로 `httpx.JSON(w,200,…)`(42행). 검사 추가.
  - `internal/server/voice.go:118` — 루프 직후 129행이 `httpx.JSON(w,200,…)`. 검사 추가.
  - `internal/server/keys_approval.go:72` — 루프 직후 81행이 `httpx.JSON(w,200,…)`. 검사 추가.
  - `internal/server/admin_operations.go:36` `adminOrganizations`, `:103`, `:411` — 셋 다 루프 직후 `httpx.JSON(w,200,…)`. (`:336` 은 344행에 이미 검사가 있으니 건드리지 말 것.)
  - `internal/server/admin_operations.go:103` 이 `adminAudit` 다 — 여기 `auditRow`/`auditItem`(115~131행)은 **손대지 말 것**. 루프 뒤 한 군데만 더한다.
  - `internal/server/admin.go:274`, `:405` — 루프 직후 `httpx.JSON(w,200,…)`.
  - `internal/server/admin.go:122` — **가장 눈에 띄는 자리.** 한 함수가 `rows` 변수를 재사용한다: 122행 roles 루프 → 130행 `rows.Close()` → 132행 `rows, err = s.DB.Query(...)` 로 groups 를 다시 담고, 145행 `return out, rows.Err()` 는 **두 번째(groups) rows 만** 검사한다. roles 루프의 스트림 오류는 통째로 사라진다. roles 루프 뒤(130행 `rows.Close()` 앞뒤)에 별도 검사를 넣는다.
  - `internal/apikey/service.go:395` — `:189`·`:209` 는 이미 `return out, rows.Err()` 이니 그대로 두고 395 만.
  - `internal/crm/service.go:394`, `:419`, `:824`, `:952` — `:179`·`:481`·`:907` 은 이미 검사가 있다.
  - `internal/intelligence/health.go:255`, `:278` — 둘 다 `return out` 으로 끝난다. 이 두 함수가 오류를 못 돌려주는 시그니처면 **시그니처를 바꾸기보다** 호출부를 보고 판단하라(둘 중 하나라도 시그니처 변경이 번지면 그 자리는 이번 범위에서 빼고 새 테스트에 예외를 두지 말고 과제서에 남은 자리로 적을 것 — 아래 "위험과 피할 것" 참고).
  - 새 파일: `internal/platform/database/rows_err_test.go`(package `database`, 이 패키지에는 현재 `database.go` 하나뿐이고 테스트 파일이 없다). `filepath.Walk(filepath.Join("..","..","..","internal"))` 로 훑는다 — `internal/crm/search_test.go` 는 두 단계였지만 이 패키지는 세 단계 깊이다. 경로 깊이를 반드시 실행해 확인할 것.
  - 위 목록은 파일별 `rows.Next()` 개수와 `rows.Err()` 개수를 세고(총 85 대 70) 의심 자리를 눈으로 연 결과다. 셈이 맞는 파일 안에서도 `admin.go:122` 처럼 Err 가 **다른 루프** 것일 수 있으므로, 최종 목록은 새 테스트가 red 일 때 찍는 `path:line` 을 정답으로 삼아라 — 위 목록에 없는 자리가 더 나올 수 있다(미확인).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/platform/database/` — 새 테스트만 먼저 red → green
  - `go test ./internal/server/ ./internal/crm/ ./internal/apikey/ ./internal/intelligence/` — 이번 정찰에서 base 가 전부 `ok` 임을 확인했다
  - `go test ./...` → `go test -race ./...`
  - `go vet ./...`
  - `go build ./...`
  - `gofmt -l <변경한 Go 파일들>` (출력이 없어야 함)
  - 프런트는 이번 변경과 무관하므로 `npm --prefix web` 계열은 돌릴 필요 없다. 돌린다면 build 뒤 `internal/webui/dist/README`·`web/public/README` 앵커가 복원되어 `git status` 가 깨끗한지 확인할 것.

- 위험과 피할 것:
  - **동작 변화가 있는 변경이다.** 지금은 스트림 오류가 나도 200 + 부분 목록이고, 고친 뒤에는 500 이다. 이것이 의도다(수용 기준 2). 다만 "행이 0개" 와 혼동하지 말 것 — 정상적으로 결과가 비면 `rows.Err()` 는 nil 이고 기존대로 빈 배열 200 이다.
  - **보호 경로 회피**: `internal/auth`, `internal/oidc`, `internal/server/public.go`, `migrations/`, `.github/workflows/` 는 이번 과제에서 손대지 않는다. `internal/auth/service.go` 의 루프는 이미 `rows.Err()` 가 있어 건드릴 것이 없고, `internal/oidc/service.go` 도 1:1 로 맞는다 — 새 테스트가 이 둘을 red 로 찍으면 그때만 최소 3줄을 더하고, 그 외 어떤 줄도 바꾸지 말 것.
  - **`auditItem`/`auditRow` 재포매팅 금지.** 2026-09-20 두 회차가 이 자리를 좁게 고쳤다. 루프 뒤 검사 한 덩어리만 더한다.
  - `internal/intelligence/health.go:255`·`:278` 처럼 **오류를 돌려줄 수 없는 시그니처**를 만나면 호출부까지 번져 M 이 L 이 된다. 그런 자리는 시그니처를 억지로 바꾸지 말고 이번 범위에서 빼되, 새 테스트에 그 파일/줄을 화이트리스트로 넣지 말고 **테스트 대상 범위를 정직하게 좁히거나**(예: 지금은 `internal/server`·`internal/crm`·`internal/apikey` 만 훑고 주석으로 "나머지는 다음 회차" 라고 적는다) 회차 노트에 남은 자리를 명시하라. 조용히 예외 목록을 늘리는 것이 최악이다.
  - 파일 전체 gofmt/포매터를 돌리지 말 것 — diff 가 수백 줄이 되면 비평 단계에서 되돌아온다. 한 줄 스타일이 많은 파일(`AdminPages.tsx` 등 프런트)은 이번에 아예 열지 않는다.
  - 실제 DB 없이 스트림 오류를 단위 테스트로 재현할 수 없다(`Server.DB` 가 인터페이스가 아니라 `*pgxpool.Pool` 이다 — `internal/server/server.go:41` 확인). 그래서 증명은 **소스 스캔 테스트**다. `pgxpool.Pool` 을 인터페이스로 바꾸는 리팩터는 이번 회차 금지.

- 차선 후보: **`security.allowed_origins` 시드 행 제거 (가치 2 / 위험 1 / S)** — `migrations/001_initial.sql:639` 의 시드 행 하나를 아무 Go 코드도 읽지 않는다(이번에 다시 grep 해 `internal`·`web` 에서 소비자 0건 확인; 단 같은 파일 645행의 `mcp.allowed_origins` 는 `internal/mcp/server.go:198` 와 `web/src/pages/AdminPages.tsx:325` 가 실제로 쓰는 **별개 설정**이므로 절대 같이 지우지 말 것). main 의 마지막 마이그레이션은 `014_momento_provider.sql` 이므로 새 번호는 `015_`. 다만 마이그레이션은 보호 경로이고 미머지 브랜치가 015 를 쓸 수 있어 번호 충돌을 먼저 확인할 것. `docs` 의 관리자 가이드 설정 표에서도 그 행을 빼야 한다.

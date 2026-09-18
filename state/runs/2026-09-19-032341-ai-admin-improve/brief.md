# 과제서 (2026-09-19, ai-admin)

## 우선 과제 판정 — "hold: budget"은 저장소 결함이 아니다
- 직전 회차 `2026-09-18-161356-ai-admin-improve`의 `stages.json`을 열어 확인: `{"improve": {"state":"hold", "reason":"회차 예산($22)이 오늘 남은 상한을 넘음"}}`. 즉 러너가 **일일 비용 상한** 때문에 회차를 시작하지 않고 멈춘 것이며, 코드·테스트·워크플로 실패가 아니다.
- 저장소 상태: `main`은 `76e5167 chore: release ai-admin v1.2.21`로 깨끗하고(작업 트리 clean), `.github/workflows/release.yml`(verify-version → package-offline → verify-offline → checksums → gh-release)·`ci.yml`(lint + test)·`Makefile`·`scripts/verify-version.sh`를 읽었으나 "같은 이유로 두 번 실패한" 단계에 해당하는 흔적이 없다. GitHub Actions 실행 이력은 이 세션에서 `gh run list`가 승인 없이 실행되지 않아 **미확인**이다.
- 따라서 이 회차에 "고칠 워크플로 결함"은 없다. 원장에는 다음처럼 **수정 과제**로 기록한다: *"직전 error는 러너 예산 hold(일일 상한 초과)였고 저장소 쪽 원인 없음 — 워크플로 변경 없음"*. 워크플로를 느슨하게 만들지 말 것.
- 구현자는 아래의 실제 과제를 수행한다(작고 확실한 S 과제로 골랐다 — 이번 회차도 예산이 빠듯하므로 범위를 넓히지 말 것).

---

- 과제: `listUsers`의 `q` 검색어에 다른 조회 경로와 같은 200자 상한 적용 (가치 2 / 위험 1 / 작업량 S)
- 왜: `/api/v1/users?q=`만 검색어 길이 상한이 없다 — `internal/server/users.go:21`은 `TrimSpace`만 하고 곧바로 세 컬럼(`username`·`email`·`display_name`) `ILIKE '%…%'` 두 번(count + 목록)에 넣는다. 감사(`audit_events.go:47`)·레거시(`legacy.go:317-318`, `query_too_long`)·MCP(`mcp.go:225`)는 모두 `len([]rune(q)) > 200`을 거부하므로, 같은 계약을 적용해 매우 긴 검색어가 2MiB URL 한도까지 그대로 스캔에 들어가는 예외를 없애고 API 계약을 일관되게 만든다.
- 수용 기준:
  1) `GET /api/v1/users?q=<201자 이상>` → 400, 오류 코드 `query_too_long`, 메시지 `검색어는 200자 이하여야 합니다.`(legacy.go:318과 동일 문구), DB 조회 없이 반환.
  2) 정확히 200자(rune 기준 — 한글 200자는 byte로 600이지만 허용)는 그대로 검색되고, 기존 `page`·`pageSize`·빈 `q` 동작은 불변.
  3) 테스트가 증명할 것: (a) 201 rune(한글로 만들어 byte≠rune임을 드러낼 것)에서 400 `query_too_long`, (b) 200 rune에서 200 응답과 정상 목록(기존 PostgreSQL 통합 테스트 헬퍼 `doJSONWithCookies`·`service` 패턴을 그대로 사용, `legacy_http_integration_test.go:18` 참고). 손으로 주입한 대역 없이 실제 서버·실제 pool을 통과할 것.
- 건드릴 파일:
  - `internal/server/users.go:listUsers` — `q := strings.TrimSpace(...)` 직후, count 조회 전에 `if len([]rune(q)) > 200 { writeError(w, http.StatusBadRequest, "query_too_long", "검색어는 200자 이하여야 합니다."); return }` 추가(users.go에 `net/http` import가 이미 있는지 확인; 현재 `writeError(w, 500, ...)`처럼 숫자 리터럴을 쓰고 있으니 파일 관례에 맞춰도 됨).
  - `internal/server/*_integration_test.go` — 기존 users 통합 테스트가 있는 파일(예: `legacy_http_integration_test.go`) 옆에 `TestListUsersRejectsOverlongQuery` 추가. `TEST_POSTGRES_DSN` 없으면 skip하는 기존 패턴을 따를 것.
  - `docs/api.md` — 사용자 목록 절에 `q` 200자 상한과 `query_too_long` 한 줄 추가(감사·레거시 절이 이미 같은 계약을 적는지 grep해서 같은 표현을 쓸 것).
- 검증 명령:
  - `docker run -d --name ai-admin-pg -e POSTGRES_USER=ai_admin -e POSTGRES_PASSWORD=test-password -e POSTGRES_DB=ai_admin_test -p 5432:5432 postgres:16-alpine`
  - `TEST_POSTGRES_DSN='postgres://ai_admin:test-password@localhost:5432/ai_admin_test?sslmode=disable' go test -race -count=1 ./internal/server/ -run 'TestListUsers|TestLegacy'` 그다음 `go test -race -count=1 ./...`(internal/server 약 65s)
  - `make lint` (= `gofmt -l .`·`go vet ./...`·`scripts/verify-version.sh`) · `go build ./...`
  - 되돌림 검증: 추가한 검사 블록만 잠시 제거하고 새 테스트가 실제로 실패(200 응답)하는 것을 확인한 뒤 복구.
- 위험과 피할 것:
  - VERSION·CHANGELOG·`.github/workflows/*`·`scripts/verify-version.sh`는 건드리지 않는다(릴리즈 메타데이터는 승인 단계가 올린다; 워크플로 완화 금지).
  - `escapeLike`·ILIKE 절·`legacy.sensitive.read` 분기(users.go:38-42)는 손대지 않는다.
  - 상한을 byte(`len(q)`)로 두지 말 것 — 다른 세 경로가 rune 기준이며, 프런트가 한글 검색어를 보낸다.
  - 웹(`web/`)은 변경 불필요. 예산이 빠듯하므로 `npm test`는 Go 변경만이라면 생략 가능(원장에 생략을 명시).
- 차선 후보: `safeCSVCell`(`internal/server/audit_events.go:220`)이 선행 tab(0x09)·CR(0x0D)를 중화하지 않음 — `"=+-@"` 검사에 `\t`·`\r`을 더하고 표 테스트로 덮기 (가치 2 / 위험 1 / S). 1순위와 독립이며 같은 회차에 둘 다 하지는 말 것.

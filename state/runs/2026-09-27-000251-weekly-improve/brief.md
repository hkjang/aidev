- 과제: Confluence 자동 수집 카드가 실패 수를 보여 주고, 진단 표가 20건에서 잘렸다고 말한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `docs/ADMIN_GUIDE.md:150` 은 이 탭이 "조회/변경 Page 수, 생성 후보, **실패 수**와 최근 단계별 오류" 를 보여 준다고 적었지만, `frontend/src/pages/AdminPage.tsx:149` 의 `sync-overview` 에는 최근 성공·조회/변경·새 초안·매핑/미매핑 네 칸만 있고 `pagesFailed` 는 응답(`confluence_handlers.go:300`)과 타입(`types.ts:229`)에만 있으며 화면에서 한 번도 쓰이지 않는다(`grep -c pagesFailed AdminPage.tsx` == 0). 그리고 진단 표의 원천인 `confluence_sync_errors` 는 최신 500건까지 남는데(`confluence_sync.go:729`) 질의는 `LIMIT 20` 이고 응답에 잘렸다는 표시가 없어, 가득 찬 표가 "진단이 이것뿐" 으로 읽힌다 — 실패가 40건 난 동기화에서 관리자는 20건만 보고 나머지가 있다는 것을 알 길이 없다.
- 수용 기준:
  1) `GET /api/v1/admin/confluence/sync/status` 응답이 `confluence_sync_errors` 행이 21건 이상일 때 잘림을 말하는 필드(예: `recentErrorsTruncated: true`)를 싣고, `recentErrors` 는 여전히 20건이다.
  2) 행이 20건 이하이면 그 필드가 응답에 없다(`omitempty`) — 상시 문구가 되지 않아야 한다.
  3) 관리자 `Confluence 자동화` 탭의 상태 칸에 실패 수가 보이고, 진단 표에는 잘렸을 때만 "최신 20건만 표시" 라는 뜻의 한 줄이 붙는다.
  4) 새 시험이 **고치기 전 코드에서 먼저 실패**하는 것을 확인한다(기대 실패: `recentErrorsTruncated=<nil>, want true`).
- 건드릴 파일 (제품 3 + 시험 1):
  - `internal/app/confluence_handlers.go:adminConfluenceStatus` (265~307행) — 280행의 질의를 `LIMIT 21`(상수 예: `confluenceRecentErrorLimit = 20` 을 두고 `limit+1`)로 바꾸고, 스캔 루프 머리에서 `len(errorsView) == confluenceRecentErrorLimit` 이면 `truncated = true` 후 `break`. 21번째 행은 `errorsView` 에 넣지 않는다(2026-09-26 회차가 `mail.go` 에서 쓴 것과 같은 한 질의 방식 — COUNT 질의를 더하지 말 것). `ORDER BY created_at DESC` 에 `, id DESC` 를 더해 동시각 동률에서 어느 행이 빠지는지가 흔들리지 않게 한다. `writeData` 맵에 `omitempty` 성격의 필드를 넣어야 하므로 맵이면 `if truncated { m["recentErrorsTruncated"] = true }` 로 조건부 삽입.
  - `frontend/src/types.ts:230` 근처 — `recentErrorsTruncated?: boolean` 추가.
  - `frontend/src/pages/AdminPage.tsx:149` (`ConfluenceTab` 의 매우 긴 한 줄 JSX) — `sync-overview` 에 `<div><small>실패</small><strong>{status.pagesFailed}</strong></div>` 칸을 더하고, `최근 동기화 진단` Card 안 표 뒤(또는 앞)에 `{status.recentErrorsTruncated&&<p className="muted">최신 20건만 표시합니다. 이전 진단은 더 남아 있습니다.</p>}`. 기존 `{status.recentErrors.length>0&&<Card …>}` 조건을 바꾸지 말 것.
  - 시험: `internal/app/confluence_test.go` 에 추가(파일이 이미 Confluence 경로 시험을 담고 있다). DB 하네스는 `newTestServer`(`httpharness_test.go`). 진단 행은 **프로덕션 쓰기 경로**로 만들 것 — `a.recordConfluenceNotice(ctx, phase, message)`(`confluence_sync.go:781`) 또는 `a.recordConfluenceError(...)`(같은 파일 722행)를 21회 호출한다(직접 INSERT 하지 말 것: 손으로 만든 대역·우회 배선은 반려 사유였다). `confluence_sync_state` 에 `system_type='CONFLUENCE'` 행이 없으면 핸들러가 500 `QUERY_FAILED` 를 반환하므로, 그 행이 마이그레이션에서 생기는지 확인하고 없으면 `finishConfluenceSync`(715행)로 채운 뒤 ADMIN 으로 GET 한다. 같은 시험이 20건 이하에서 필드가 **없음**까지 양방향으로 증명하면 수용 기준 2를 덮는다.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l internal/app` (빈 출력) · `go build ./...` · `go vet ./...`
  - `go test ./internal/app/ -run <새 시험 이름> -count=1` → 그다음 전체 `go test ./... -count=1` (실제 DB 로 약 150초)
  - **DB 는 이미 떠 있다**: `weekly-test-pg` = `pgvector/pgvector:pg16`, `0.0.0.0:15434->5432`, Up 4 days (2026-09-28 확인). 워크트리의 `WEEKLY_TEST_POSTGRES_DSN` 이 그 포트를 가리킨다.
  - `python3 scripts/openapi-check.py` (119개) — 이 경로는 `docs/openapi.yaml:2273` 에 요약만 있고 스키마가 없어 필드 추가로 깨지지 않는다(확인). 요약 문구를 손대는 것은 선택.
  - `python3 scripts/paging-check.py` — 2026-09-26 회차가 `truncated?: boolean` 스칼라 필드는 offset 요구를 만들지 않음을 실행으로 확인했다(같은 모양).
  - `python3 scripts/guard-check.py --changed main` — `adminConfluenceStatus` 가 새 시험으로 도달되어야 한다. 이 저장소 관례대로 제품 함수를 짚는 `// guards` 주석을 시험에 달 것.
  - `npm --prefix frontend ci` → `npm --prefix frontend run lint` (tsc -b) → `npm --prefix frontend test` (vitest 160개) → `npm --prefix frontend run build`
  - 문서: ADMIN_GUIDE.md:150 은 이미 실패 수를 약속하므로 새 문장이 필요 없다. 잘림 문구를 가이드에 한 줄 더한다면 `python3 scripts/render-docs.py ADMIN_GUIDE` 로 HTML 을 다시 만들 것(다른 문서 재생성 금지).
- 위험과 피할 것:
  - `total`·건수를 응답에 더하지 말 것 — 불리언만. 총건수를 실으면 `paging-check.py` 가 그 화면에 offset 경로나 허용 목록 사유를 요구한다.
  - 질의 오류가 `errorRows, _ :=` 로 버려지는 것(280행)은 **이번 과제 범위 밖**이다. 계약(새 필드·상태 코드)을 바꾸지 말고, 고칠 생각이면 로그 한 줄(`a.logger.Error`)까지만 — 시험으로 증명할 수 없는 경로를 수용 기준에 올리지 말 것.
  - `confluence_sync_errors` 는 오류만 담는 표가 아니다 — `recordConfluenceNotice`(781행)가 오류가 아닌 진단도 같은 표에 넣는다. 화면 문구는 "오류" 가 아니라 "진단" 으로 쓸 것(Card 제목이 이미 `최근 동기화 진단`).
  - `migrations/`·`auth.go`·`.github/workflows/` 는 건드리지 않는다. 마이그레이션 없이 끝나는 과제다.
  - `pages_failed` 는 마지막 동기화 1회의 카운터이고 진단 표는 최신 500건의 누적이다 — 두 숫자가 일치해야 한다고 쓰지 말 것(화면 문구가 둘을 같은 것처럼 말하면 거짓이 된다).
  - AdminPage.tsx 의 해당 JSX 는 한 줄이 매우 길다. 줄 전체를 재포맷하지 말고 필요한 조각만 끼워 넣어 diff 를 작게 유지할 것.
- 차선 후보: README:47 · docs/MCP.md:3 · AdminPage 의 MCP '읽기 전용' 서술을 `internal/app/mcpwrite.go` 의 `mcpMayWrite`·`mcp:write` 기준으로 정정 (2/1/S) — 문서만 고치는 변경이며 대체된 옛 서술은 남기지 말 것.

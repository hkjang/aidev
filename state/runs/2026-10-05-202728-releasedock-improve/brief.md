- 과제: 전체 모드 릴리즈의 저장 로그 조회·내려받기 백엔드 엔드포인트 (가치 3 / 위험 2 / 작업량 M)
- 왜: 전체 모드에는 릴리즈 로그를 읽는 경로가 SSE 스트림 하나뿐이다 — `server.go:301` 에 `GET /api/v1/releases/{id}/logs/stream` → `releases.go:1464 streamReleaseLogs` 만 있고, `/logs` 는 없다(라우트 등록표 287-301 전부 확인). 그래서 화면 표시 한도(4999줄)로 밀려난 줄과 v0.5.29 가 알려 주는 끊김("아래 로그는 끊긴 시점까지입니다") 이후의 줄을 되찾을 수단이 서버에 아예 없다 — 릴리즈 실패 조사와 감사가 불완전한 로그 위에서 이뤄진다. 단순 모드는 같은 문제를 이미 풀어 두고 재사용 가능한 조각까지 만들어 뒀으므로(아래), 그 계약을 릴리즈에 옮기는 것은 복제가 아니라 배선이다.

- 수용 기준:
  1) `GET /api/v1/releases/{id}/logs` 가 존재하고 `releases.read` 로 보호된다(`server.go` 의 다른 릴리즈 라우트와 같은 `s.withPermission("releases.read", …)` 형태). 새 권한을 만들지 않는다.
  2) 기본(JSON) 응답이 `release_job_logs` 저장 행을 id 오름차순으로 돌려주고, 단순 모드와 **같은 필드 이름과 커서 계약**을 쓴다 — `{"items":[{"id","stream","message","createdAt"}],"lastId","hasMore"}`, `?after=`·`?limit=` 지원, 행이 없으면 `lastId` 를 호출자가 준 `after` 로 유지(simple.go:1291-1327 이 그 이유를 주석으로 적어 둔다).
  3) `?format=text` 가 전체 로그를 평문으로 내려주고, `Content-Type: text/plain; charset=utf-8`·`X-Content-Type-Options: nosniff`·`Content-Disposition` 을 단순 모드와 같게 보낸다. 파일 이름은 릴리즈 id 만이 아니라 사람이 구분할 수 있는 릴리즈 식별 정보(이름/버전 — `releaseSelect` 가 무엇을 주는지 먼저 확인할 것, **미확인**)를 담는다.
  4) 없는 릴리즈 id → 404 `not_found`, DB 오류 → 500 `database_error`(기존 `writeError` 관례; `getReleaseByID` 가 `pgx.ErrNoRows` 를 404 로 바꾸는 방식을 그대로).
  5) 테스트가 증명할 것 — **DB 없이** 도는 단위 테스트로: ① 릴리즈 로그 행(stream=stderr 포함)이 기대한 평문으로 렌더되고 ② 읽기 도중 Scan/Err 실패가 본문에 잘림 공지를 남긴다(= 받아 둔 파일이 완전한 것처럼 보이지 않는다) ③ 내려받기 파일 이름이 릴리즈 식별 정보를 담는다. 라우트·권한 배선은 `go build` 와 (가능하면) DB 픽스처 테스트로 확인하되, DSN 없으면 SKIP 되므로 **SKIP 을 통과로 보고하지 말 것**.
  6) `streamReleaseLogs` 의 동작·SQL·커서 계약은 바꾸지 않는다.

- 건드릴 파일 (프로덕션 2개):
  - `backend/internal/server/releases.go` — `listReleaseLogs(w,r)` 와 `downloadReleaseLog(w,r,releaseID)` 추가. SQL 은 `streamReleaseLogs` 가 쓰는 식(releases.go:1507 `SELECT l.id,l.stream,l.payload,l.created_at FROM release_job_logs l JOIN release_jobs j ON j.id=l.job_id WHERE j.release_id=$1 AND l.id>$2 ORDER BY l.id LIMIT $3`)과 **같은 JOIN·같은 정렬**을 쓸 것. 새 테이블·새 뷰를 만들지 말 것.
  - `backend/internal/server/server.go` — `:301` 바로 위/아래에 `mux.HandleFunc("GET /api/v1/releases/{id}/logs", s.withPermission("releases.read", s.listReleaseLogs))` 한 줄. 두 패턴은 끝 세그먼트가 다르므로(`logs` vs `logs/stream`) Go 1.22 mux 에서 충돌하지 않는다.
  - `backend/internal/server/release_log_test.go` (신규 테스트) — `simple_log_test.go` 가 순수 단위 테스트(DB 불필요)라는 선례를 따를 것.
  - **재사용할 것(복제 금지)** — `simple.go` 가 이미 일반화해 뒀다: `logRowScanner`(Next/Scan/Err 세 메서드만 요구하는 인터페이스 — 바로 이 "DB 없이 잘림 경로를 시험한다" 는 목적으로 만들어졌다고 주석에 적혀 있다), `writeSimpleRunLog(w io.Writer, rows logRowScanner)`(stream=="stderr" 에 `[stderr] ` 접두, Err 비어 있지 않으면 공지), `simpleRunLogTruncatedNotice`, `runLogDisposition(filename, status, id string)`. 이름에 simple 이 들어가도 동작은 두 모드에 공통이다 — 릴리즈용으로 같은 로직을 두 번째로 쓰면 두 경로가 갈라진다. 이름이 걸리면 함수를 옮기지 말고 그대로 호출하라(파일 이동·개명은 이번 범위 밖).
  - 프런트(`web/`, `api.ts`, `ReleaseDetailPage.tsx`)는 **이번 조각 밖**이다. 다음 회차가 내려받기 버튼을 붙인다.

- 검증 명령:
  - `cd backend && go build ./... && go test ./internal/server/ -count=1`
  - 새 테스트만: `cd backend && go test ./internal/server/ -run 'ReleaseLog' -count=1 -v`
  - `cd backend && gofmt -l internal/server` 가 아무것도 출력하지 않을 것, `cd backend && go vet ./internal/server/`.
  - 웹 0 파일 변경이면 `web` 테스트는 돌리지 않아도 된다(최근 회차의 반대 관례 그대로).

- 위험과 피할 것:
  - 보호 경로 금지: `rbac_policy.go`, `auth.go`/`oidc.go`, `internal/store/migrations/`, `.github/workflows/`. 마이그레이션은 필요 없다 — `release_job_logs` 는 이미 있다.
  - 같은 값을 두 경로가 읽게 된다(스트림과 조회). 같은 JOIN·같은 정렬로 두 경로가 같은 행을 같은 순서로 보게 하고, 스트림 쪽을 조회에 맞춰 고치려 하지 말 것 — 계약이 다르다(스트림은 Last-Event-ID 와 `?after` 중 큰 값을 쓰고 `simple_stream_cursor_test.go` 가 그것을 고정한다).
  - 평문 내려받기는 로그 payload 원문을 그대로 내보낸다. 새 스크럽을 발명하지 말고 단순 모드 내려받기와 **정확히 같은** 처리 수준을 유지할 것.
  - 응답을 메모리에 다 모으지 말 것 — 단순 모드는 헤더·200 을 먼저 보내고 행을 스트리밍하며, 그래서 중간 실패를 본문 공지로 알린다(simple.go:1330-1388). 같은 순서를 지킬 것.
  - `releases_*_test.go` 가 이 저장소에 **하나도 없다**(simple_* 만 11개) — 릴리즈용 DB 픽스처는 새로 짜야 하고 `newSimpleBatchFixture` 처럼 `store.Pool` 을 요구해 DSN 없으면 SKIP 된다. 픽스처 작성이 세션을 먹으면 수용 기준 5의 ①②③(DB 불필요)만으로 마치고, 라우트 테스트를 뺐다는 사실을 보고에 명시할 것 — 없는 PASS 를 주장하지 말 것.
  - 세션이 모자라면 `?format=text`(내려받기)를 먼저 끝낼 것. 한도로 밀려난 로그를 되찾는다는 목적은 내려받기 하나로 달성되고, JSON 페이지네이션은 화면 작업이 붙는 다음 조각에서 더 필요해진다.

- 차선 후보: 단순 모드 두 화면(`SimpleDeployPage.tsx:287`, `SimpleRunDetailPage.tsx:261`)의 error 리스너에 `stopped` 가드 추가 — 두 화면 모두 `end` 핸들러에서 `source.close()` 를 부르므로(:296, :270) 그 뒤 늦게 도착하는 error 는 readyState 2 로 보여 거짓 끊김 경고를 띄울 수 있고, 전체 모드는 v0.5.29 에 같은 가드를 받았다. **단, EventSource 스펙상 close() 이후에는 error 가 발사되지 않으므로 실제 브라우저에서 도달 가능한지 미확인이다 — 프로덕션 배선(실제 App 렌더 + 각 화면의 EventSource 대역)으로 재현하지 못하면 손대지 말고 1순위로 돌아갈 것.**

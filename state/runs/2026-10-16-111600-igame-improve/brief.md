- 과제: 본인 플레이 기록(GET /api/v1/me/history)의 페이지 정렬에 유일한 tiebreak 추가 (가치 3 / 위험 2 / 작업량 S)

- 왜: `playHistory`(internal/api/catalog.go:987, 질의는 :990)는 `ORDER BY gs.started_at DESC LIMIT $2 OFFSET $3` 로 유일하지 않은 컬럼만으로 정렬한다 — OFFSET 은 정렬이 전순서일 때만 결과를 페이지로 가르므로 `started_at` 이 같은 세션 행들의 페이지 배치가 요청마다 달라질 수 있고, 같은 행이 두 페이지에 나오거나 어느 페이지에도 나오지 않을 수 있다. 같은 결함을 관리자 목록 3개(admin.go:580 `created_at DESC,id DESC` / :718 / :755)와 공개 카탈로그(catalog.go:130 `ORDER BY g.name,g.id`)에서 이미 고쳤고 이 사용자 경로만 보증 밖에 남아 있어, 한 줄로 계약을 맞출 수 있다.
- 수용 기준:
  1) `GET /api/v1/me/history?limit=&offset=` 가 `started_at` 이 동일한 세션들을 **`started_at` 내림차순, 동점이면 `id` 내림차순**의 명시적 순서로 돌려준다(관리자 목록과 같은 방향).
  2) `limit=2` 로 `offset=0/2/4` 를 순회하면 6개 행이 각각 정확히 한 번 나오고, 서로 다른 `started_at` 을 가진 대조군 행은 여전히 시간 우선으로 앞/뒤에 온다.
  3) 테스트는 실제 PostgreSQL 17 + 실제 `New(pool,…).Router()` + 실제 세션 쿠키 왕복으로, 수정 전 **명시적 ID 순서 불일치 Red**, 수정 후 Green, 보조 키(`,gs.id DESC`)만 되돌리면 다시 Red 임을 보인다(소스 문자열 검사·대역 금지).
- 건드릴 파일 (프로덕션 1 + 테스트 1 + 문서 1):
  - `internal/api/catalog.go:990 playHistory` — `ORDER BY gs.started_at DESC` → `ORDER BY gs.started_at DESC,gs.id DESC`. 다른 질의·응답 필드·`writeJSON(w,200,map[string]any{"items": items})` 모양은 그대로 둔다(이 endpoint 는 `total`/`limit`/`offset` 을 반환하지 않는다 — 확인함).
  - 신규 `internal/api/history_order_pg_test.go` — 아래 레시피대로 PG 회귀 1파일.
  - `docs/api.md:10` — 결정론 보증 문장(현재 "관리자 목록 … 과 로그인한 사용자가 조회하는 `/api/v1/games` … 결정론입니다")에 `/api/v1/me/history`(시작시각 내림차순, 동점이면 ID 내림차순)를 한 구절 추가. 같은 줄의 "다른 목록은 아직 이 보증이 없어 …" 와 "페이지 사이 행 추가·삭제 시 창이 밀린다" 경고는 유지한다.
- 테스트 레시피 (이 저장소에서 실제로 쓰이는 헬퍼만 사용):
  - `migratedPool(t)`(admin_pg_test.go:29 — DSN 없으면 skip), `insertTestUser(t,pool,"user")`(:81), `insertTestSession(t,pool,user)`(:94, raw 쿠키 반환), `insertTestGame(t,pool,slug)`(catalog_pg_test.go:27)를 재사용하고 서버는 `httptest.NewServer(New(pool,nil,logger).Router())` 로 띄운다(session_finish_pg_test.go:33 `newFinishFixture` 가 이 조합의 선례다).
  - 동점 행은 **명시적 UUID 로 직접 INSERT** 한다 — 공유 헬퍼 `insertTestPlaySession`(catalog_pg_test.go:38)은 id 를 돌려주지 않으므로 그대로 쓰지 말고, 새 파일 안에서 `INSERT INTO game_sessions(id,user_id,game_id,session_token_hash,status,started_at,duration_ms)` 를 쓴다. 스키마 확인: `id uuid PRIMARY KEY DEFAULT gen_random_uuid()`, `session_token_hash bytea NOT NULL UNIQUE`, `started_at timestamptz NOT NULL`(migrations/001_initial.sql:126-138). 토큰 해시는 행마다 `sha256.Sum256([]byte(uuid.NewString()))` 로 유일하게 만든다.
  - id 는 `30000000-0000-0000-0000-00000000000{1..6}` 처럼 고정하고 **오름차순(1→6)으로 삽입**한다. 그러면 보조 키가 없는 현재 질의는 삽입/인덱스 순서인 `[1,2,3,4,5,6]` 을 주고 새 계약 기대값 `[6,5,4,3,2,1]` 과 달라 Red 가 안정적으로 재현된다(catalog_order_pg_test.go:36 의 "UUID order is deliberately the reverse of insertion order" 와 같은 수법). 모든 동점 행의 `started_at` 은 하나의 고정 시각(예: `now() - interval '1 hour'`를 Go 로 한 번 읽어 재사용하거나 `time.Date(...)` 고정값)으로 같게 준다.
  - 대조군: 더 최근 `started_at` 1행과 더 오래된 `started_at` 1행을 넣어, 시간 우선 정렬이 보조 키 때문에 깨지지 않음을 같은 테스트에서 단정한다. 사용자 격리를 위해 **다른 사용자**의 세션 1행도 넣어 응답에 섞이지 않음을 확인한다(`WHERE gs.user_id=$1`).
  - 정리: 새 user/game 행은 각 헬퍼가 `t.Cleanup` 으로 지우고 `game_sessions` 는 FK `ON DELETE CASCADE` 로 따라 지워진다(001_initial.sql:128-129). 다른 테스트의 fixture(`newAdminListFixture`, `settingFixture`)는 수정하지 않는다.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - PG 준비: README 의 `postgres:17-alpine` 컨테이너 + `CREATE SCHEMA igame_test_extensions; CREATE EXTENSION pgcrypto WITH SCHEMA igame_test_extensions;` (public 확장은 `migratedPool` 이 거부한다).
  - `IGAME_TEST_DSN='postgres://igame:igame@127.0.0.1:15432/igame?sslmode=disable&search_path=public,igame_test_extensions' go test ./internal/api -run '^TestHistory' -count=1 -v` (수정 전 Red → 수정 후 Green → 보조 키 제거 변이에서 다시 Red)
  - `IGAME_TEST_DSN=… go test ./internal/api -run '^TestHistory' -race -count=5`
  - `make test-db DSN='postgres://…'` (api 약 31–34s / database 약 2s)
  - `go test ./cmd/... ./internal/... ./migrations/... -count=1`, `go vet ./cmd/... ./internal/... ./migrations/...`, `go build ./cmd/... ./internal/... ./migrations/...`, `gofmt -l internal/api/catalog.go internal/api/history_order_pg_test.go`, `git diff --check`, `bash scripts/check-release-contract.sh`
  - 실행 후 잔여 `api_test_`/`migrate_` 스키마 0개와 `igame_test_extensions` 보존을 확인하고, 직접 만든 컨테이너만 제거한다.
- 위험과 피할 것:
  - 이미 착지한 정렬(admin.go:580/718/755, catalog.go:130)과 `gameSelect`·`usersFilter`·`auditLogFilter`·`pagedTotal`·감사 CSV 내보내기는 건드리지 말 것. `extended.go:581 adminRankings` 는 이번 범위 밖(차선 후보).
  - 플래키 금지: `limit=1` 단독 순회나 "아직 보지 못한 행을 UPDATE" 류 재현은 과거 회차에서 Red 가 안 났거나 다른 테스트와 함께 돌릴 때만 깨졌다. 실행 계획·힙 순서를 계약으로 단정하지 말고, **명시적 ID 순서 기대값**을 주 단정으로 쓸 것.
  - 도달성 과장 금지: 운영 장애 보고가 아니라 페이지 경계의 결정론을 세우는 수정이다. `INSERT INTO game_sessions` 프로덕션 경로는 catalog.go:303 한 곳뿐이고 한 요청이 한 행을 쓰므로 `started_at` 동점은 같은 트랜잭션 다중 쓰기·데이터 이관·시계 해상도에서 생긴다(현장 동점 사례는 **미확인**). 커밋 메시지·문서에서 "장애 수정" 으로 쓰지 말 것.
  - 보호 경로(auth/session 발급, migrations 파일, .github/workflows)는 손대지 않는다. 기존 인덱스 `game_sessions_user_idx(user_id, started_at DESC)` 가 있으므로 새 인덱스를 추가하지 말 것(limit ≤ 200 이라 정렬 비용은 무해).
  - 감사 로그 detail 에 조회 문자열 같은 원문을 넣지 않는다(이 과제에는 감사 기록 변경이 없다).
- 차선 후보: 관리자 점수 심사 목록 `extended.go:581 adminRankings` 의 `ORDER BY s.created_at DESC` 에 `,s.id DESC` 추가 — 1순위가 성립하지 않을 때(예: 이미 보조 키가 있음) 이 쪽을 같은 방식(실제 PG + 관리자 세션 + 명시적 `scores.id` 순서)으로 고정할 것. `scores.id` 는 `uuid PRIMARY KEY`(001_initial.sql:154)이고 `scores.session_id` 는 UNIQUE 이므로 fixture 는 세션 1:1 로 만들어야 한다.

## 작업 분해와 추정 (bottom-up, 범위로)
| 조각 | 산출물 | 추정 |
| --- | --- | --- |
| PG 환경 준비·기준 실행 | 컨테이너 + `make test-db` 1회 | 10–15분 |
| 신규 테스트로 Red 재현 | history_order_pg_test.go | 15–25분 |
| 한 줄 수정 + Green + 변이 재Red | catalog.go:990 | 5–10분 |
| 문서 1줄 + 전체 검증 | docs/api.md:10, vet/build/gofmt/release-contract | 10–15분 |
- 합계 40–65분(한 세션 상한 45분을 넘길 수 있는 꼬리는 PG 준비와 race 5회). 가장 많이 틀릴 가정: "삽입 순서대로 반환되어 Red 가 난다" — 실행 계획이 인덱스 스캔이 아니라 다른 경로를 타면 기대 Red 가 다른 순서로 나올 수 있다. 그때는 기대값을 계약(`[6,5,4,3,2,1]`)에 고정한 채 **관측된 현재 순서**를 로그로 남기고 진행할 것(계약을 관측에 맞추지 말 것).
- 컨틴전시(이 과제 안의 known unknown): +15분 — 플래키 발견 시 테스트 설계 교체(과거 games 목록에서 실제로 1회 발생).
- 관리 예비(범위 밖 발견용, 러너 재량): +1 회차 — 예: adminRankings 까지 확장하거나 동점 재현이 구조적으로 불가능하다고 판명되는 경우. 이번 회차에서 당겨쓰지 말 것.

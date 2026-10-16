# 과제서 — igame 2026-10-16-132818

## 0. 우선 과제(자동 배정) 판정: **이미 해결됨 — 고칠 실패가 남아 있지 않다**

배정된 내용("마지막 회차가 verify-failed, CI failed, PR open #40, 릴리즈 워크플로 2회 실패")은 **낡은 적재**다.
실제로 열어 본 증거:

- `state/runs/2026-10-16-120827-igame-shepherd/fix-summary.md` — 실패 단계는 release/ci 의
  `Scan reachable Go vulnerabilities`(govulncheck exit 3). 원인은 PR #40 의 ORDER BY 변경과 무관하며,
  저장소가 핀한 **Go 1.26.6 의 stdlib 취약점 10건**(GO-2026-6603…6617, net/http·net/textproto·crypto/tls,
  전부 `Fixed in go1.26.9`)이 reachable 이었다. 수리 회차가 `GOTOOLCHAIN=go1.26.6/1.26.7 → EXIT=3`,
  `go1.26.9 → EXIT=0` 으로 양방향 확인한 뒤 핀을 1.26.9 로 올렸다(커밋 `a1341ce`).
- `state/runs/2026-10-16-120827-igame-shepherd/verify.gate.json` — `{"ok":true,"state":"verified"}`.
- `state/runs/2026-10-16-122033-igame-approve/stages.json` — merge done / release **published v0.7.33** /
  assets **verified(1개)**. `release.json` = `{"status":"released","version":"0.7.33"}`.
- `state/runs/2026-10-16-122033-igame-approve/ci-b718a25c9a7e.json` — main@b718a25 의 check-run 4개
  (test/build/deploy/report-build-status) 전부 `"conclusion":"success"`.
- 현 worktree 의 Go 핀 **8곳 전부 `1.26.9` 로 일치**(직접 grep):
  `go.mod:3`, `Dockerfile:21`·`:59`, `.github/workflows/ci.yml:109`·`:120`,
  `.github/workflows/release.yml:122`·`:133`, `scripts/check-release-contract.sh:14`·`:74`·`:76`·`:110`.
  핀이 어긋난 곳은 없다 — 반쪽 수정으로 남은 흔적이 없다.
- PR #40 은 `284e70a` 로 **머지됨**(`git log`). VERSION = `0.7.33`.

따라서 워크플로를 느슨하게 만드는 변경도, 재수정도 하지 않는다. 이번 회차는 일반 회차로 진행한다.
**미확인:** GitHub 의 현재 워크플로 런 목록은 이 세션에서 `gh` 실행 권한이 없어 직접 조회하지 못했다
(위 판정은 저장소 상태 + 러너가 남긴 run 기록에 근거한다). 구현자가 `gh run list` 를 쓸 수 있다면
v0.7.33 태그 런이 success 인지 한 번 확인하고 시작할 것. 만약 **아직 붉다면** 그 로그의 실패 단계가
이 과제서보다 우선이다.

---

- 과제: 관리자 점수 심사 목록(`adminRankings`)의 페이지 정렬에 유일한 tiebreak 추가 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/api/extended.go:581` 의 질의는 `ORDER BY s.created_at DESC LIMIT $3 OFFSET $4` 로
  **유일하지 않은 컬럼 하나**만으로 정렬한다. OFFSET 이 결과를 겹치지 않는 창으로 가르는 것은 정렬이
  전순서일 때뿐이므로, `created_at` 이 같은 심사 대기 점수들은 요청마다 상대 위치가 달라져 어떤 행은 두
  페이지에 나오고 어떤 행은 어느 페이지에도 나오지 않을 수 있다 — 관리자가 점수를 승인·반려하는 목록에서
  행이 사라지는 것은 심사 누락이다. 이 저장소는 같은 결함을 이미 네 곳에서 같은 수법으로 고쳤고
  (`admin.go:580/718/755` 의 `created_at DESC,id DESC`, `catalog.go:130` 의 `g.name,g.id`,
  `realmguard.go:1879` 의 `score DESC,created_at,id`, `catalog.go` playHistory 의 `gs.started_at DESC,gs.id DESC`),
  **보조 키가 없는 페이지 목록은 이제 이 한 곳만 남았다**. 고치면 페이지 계약이 저장소 전체에서 일관해진다.

- 수용 기준:
  1) `GET /api/v1/admin/rankings?game_id=<테스트 게임>&limit=2&offset=…` 를 끝까지 넘기면 모든 행이
     **정확히 한 번씩** 나오고, 순서가 `created_at DESC`, 동점이면 `id DESC` 다.
  2) `created_at` 이 **서로 다른** 대조군(한 시간 뒤/한 시간 전 행)은 시각이 먼저 정렬을 결정한다 —
     id 보조 키가 1차 키를 가로채지 않는다.
  3) 테스트는 보조 키를 되돌리면 **Red** 가 되고 복원하면 Green 이 되어야 한다(인과 확인). 즉
     `,s.id DESC` 만 제거한 변이에서 실패하는 것을 로그로 남긴다. 보조 키가 없는 현재 코드에서
     Red 가 먼저 재현되는 것을 확인한 뒤 수정한다.
  4) `status` / `game_id` 필터와 응답 필드(`items` 의 15개 키)는 그대로다 — 회귀 없음.

- 건드릴 파일 (프로덕션 1 + 테스트 1 + 문서 1 = **3개**):
  - `internal/api/extended.go:581` — `adminRankings` 의 `ORDER BY s.created_at DESC` →
    `ORDER BY s.created_at DESC, s.id DESC`. **이 한 줄만.** 같은 함수의 SELECT 목록·WHERE·스캔 순서는
    손대지 않는다.
  - `internal/api/rankings_order_pg_test.go` (**신규**) — 아래 레시피.
  - `docs/api.md:10` — 보증 목록에 `/api/v1/admin/rankings`(생성시각 내림차순, 동점이면 UUID ID 내림차순)를
    추가하고, 바로 뒤 "다른 목록은 아직 이 보증이 없어…" 문장을 남은 범위에 맞게 좁힌다.
    (지금 그 줄은 `/api/v1/admin/{users,audit,games}`·`/api/v1/games`·`/api/v1/me/history` 만 열거한다.)

- 테스트 레시피 (검증된 blueprint 를 그대로 베낄 것):
  `internal/api/history_order_pg_test.go` 가 지난 회차에 **착지해 통과한** 같은 모양의 테스트다. 그 파일의
  구조(픽스처 struct → `migratedPool(t)` → `insertTestUser`/`insertTestSession`/`insertTestGame` →
  명시적 id 로 행 삽입 → `httptest.NewServer(New(pool,nil,logger).Router())` → 쿠키로 page 헬퍼 →
  `assert…Order` 중복/누락 단정)를 복사해 scores 로 바꾼다. 확인한 사실:
  - `insertTestUser(t, pool, "admin")` — 역할 문자열을 받는다(`admin_pg_test.go:81`).
    `/api/v1/admin/rankings` 는 그룹 미들웨어 `requireRole("admin","operator")` 만 요구한다
    (`api.go:178`, 라우트는 `api.go:230`) — `With(requireRole("admin"))` 가 아니다.
  - `migrations/001_initial.sql:153` 의 `scores`: `id uuid PRIMARY KEY DEFAULT gen_random_uuid()`,
    **`session_id uuid NOT NULL UNIQUE REFERENCES game_sessions(id) ON DELETE CASCADE`**,
    `created_at timestamptz NOT NULL DEFAULT now()`. → **점수 1개마다 전용 game_session 1개**가 필요하다.
    이것이 history 보다 설치 비용이 큰 유일한 지점이다.
  - `game_sessions` 삽입은 `session_token_hash` 가 필요하다 —
    `history_order_pg_test.go:44 insertHistorySession` 이 `sha256.Sum256([]byte(uuid.NewString()))` 로 만드는
    방식을 그대로 쓴다(공용 `insertTestPlaySession`(catalog_pg_test.go:38)은 **id 를 돌려주지 않아** 못 쓴다).
  - `scores` 에 명시적 `id` 와 `created_at` 을 넣는 선례가 이미 있다: `rankings_pg_test.go:108`
    (`INSERT INTO scores(user_id,game_id,session_id,score,created_at)`). **그 파일의 `newRankingFixture` 는
    재사용하지 말 것** — `tiedPlayers = 3000` 명을 만들고 `ANALYZE` 를 돌리는 무거운 픽스처다.
  - **마이그레이션은 `scores` 를 seed 하지 않는다**(`migrations/` 에 `INSERT INTO scores` 0건 확인).
    그래도 `adminRankings` 는 사용자 필터가 없으니, 테스트는 **`?game_id=<전용 slug 또는 uuid>` 로 범위를
    좁혀** 다른 행의 간섭을 구조적으로 배제할 것(핸들러가 `s.game_id::text=$2 OR g.slug=$2` 를 지원한다).
  - 권장 fixture: 전용 게임 1개 + admin 1명 + 세션 쿠키. 동점 블록 6행은
    `40000000-0000-0000-0000-%012d` (i=1..6) 로 명시적 id, 전부 같은 `created_at`
    (예: `time.Date(2026,3,4,5,6,7,0,time.UTC)` 같은 고정값 — 시계 해상도에 의존하지 말 것).
    대조군 2행: `…009`(한 시간 뒤), `…000`(한 시간 전). 기대 순서 = `[009, 006..001, 000]`.
  - 테스트 2개면 충분하다: ① 전체를 `limit=2` 로 순회해 순서·중복·누락 단정(+ 끝을 지난 offset 이 0건)
    ② 동점 블록 **안쪽**에서 창이 끊기도록 `offset ∈ {1,3,5}` 를 걸어 경계가 동점 구간에 떨어져도 각 행이
    정확히 한 번 나오는 것을 단정.
  - 응답은 `{"items":[…]}` 뿐이다(`total` 없음) — `items[i]["id"]` 가 **score 의 id** 이고
    `items[i]["session_id"]` 가 세션 id 다. 혼동하지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  # PG 준비(README): postgres:17-alpine 컨테이너 +
  #   CREATE SCHEMA igame_test_extensions; CREATE EXTENSION pgcrypto WITH SCHEMA igame_test_extensions;
  # public 에 확장이 있으면 migratedPool 이 거부한다.
  export IGAME_TEST_DSN='postgres://…/…?search_path=public,igame_test_extensions'

  # 1) 수정 전 Red
  go test ./internal/api -run '^TestAdminRankings' -count=1 -v
  # 2) extended.go:581 한 줄 수정 후 Green, 3) `,s.id DESC` 만 제거한 변이에서 다시 Red, 4) 복원
  go test ./internal/api -run '^TestAdminRankings' -race -count=5
  make test-db DSN=…                      # api 약 31–37s / database 약 2s
  go test ./cmd/... ./internal/... ./migrations/... -count=1   # DSN 없이 — PG 테스트는 skip
  go vet ./cmd/... ./internal/... ./migrations/...
  go build ./cmd/... ./internal/... ./migrations/...
  gofmt -l internal/api/extended.go internal/api/rankings_order_pg_test.go
  bash scripts/check-release-contract.sh
  git diff --check
  ```
  실행 뒤 잔여 `api_test_`/`migrate_` 스키마 **0개**와 `igame_test_extensions` 의 pgcrypto 보존을 확인할 것.

- 작업분해·추정 (bottom-up, 기준: 같은 수법이 이 저장소에서 4회 착지한 실측 소요):
  | 조각 | 추정 |
  |---|---|
  | PG 컨테이너·확장 스키마 준비 | 5분 |
  | 신규 테스트 파일 작성(history blueprint 복사 + scores 어댑트) | 15분 |
  | Red 재현 → 한 줄 수정 → Green → 변이 Red → 복원 | 10분 |
  | `docs/api.md:10` 문장 갱신 | 3분 |
  | 전체 검증(`make test-db` + vet/build/fmt/contract) | 10분 |
  | **합계(기대)** | **43분** |
  - 범위: 35–55분, 10회 중 8회. 포함: 위 5조각. **제외**: web/SDK 변경, 새 인덱스 추가, CI 워크플로 변경.
  - contingency(프로젝트 몫, 10분): scores 의 `session_id UNIQUE` 때문에 점수당 세션을 만들어야 해
    픽스처가 history 보다 길어지는 것 — 이것이 유일한 알려진 변동 요인이다.
  - management reserve(러너 몫, 15분): PG 컨테이너·확장 준비가 실패하는 경우. 이때는 PG 회귀를 포기하지 말고
    준비를 고칠 것 — DSN 없이 돌린 PASS 는 **증거가 아니다**(api 54개가 skip 된다).

- 위험과 피할 것:
  - **보호 경로 전부 회피**: `auth`·`migrations`·`.github/workflows`·`scripts/check-release-contract.sh` 를
    건드리지 않는다. 새 인덱스를 추가하지 말 것 — `scores_rank_idx(game_id, score DESC, created_at ASC) WHERE verified`
    는 이 질의용이 아니고, 이번 수정은 정렬 **계약**만 좁힌다(심사 목록은 관리자용 저빈도 경로다).
  - `extended.go` 는 토너먼트·공지·배너·점수 심사가 한 파일에 있다. `moderateRanking`·`excludeRanking`·
    `moderationInput` 은 **손대지 말 것**. 바꾸는 것은 `adminRankings` 의 ORDER BY 한 줄이다.
  - 과거 교훈에서 실제로 걸린 자리: ① `limit=1` 단독 순회 + seed 된 동점 행 방식의 정렬 테스트는
    **플래키**했다(2026-10-07 games 목록, 단독 5/5 녹색이 합동 실행에서 Red — top-N heapsort bound/통계 타이밍
    의존). 그래서 **명시적 id 를 직접 삽입하는 방식**(history blueprint)만 쓸 것. ② "아직 못 본 행을 UPDATE"
    하는 Red 레시피는 Red 가 나지 않는다(힙 튜플이 테이블 끝으로 가므로). 이번 레시피는 UPDATE 를 전혀 쓰지
    않으니 그 함정에 들어가지 않는다.
  - 운영자 지침 준수: 프로덕션 배선을 통과하는 실행 테스트로 증명할 것 — `New(pool,nil,logger).Router()` 를
    httptest 로 띄우고 실제 DB 세션 쿠키로 왕복한다. **소스 문자열 검사나 손으로 주입한 의존성은 근거로 쓰지 말 것.**
  - 도달성을 과장하지 말 것: 현장에서 `created_at` 동점 사례는 **미확인**이다. 프로덕션의 `INSERT INTO scores` 는
    `catalog.go:616`·`realmguard.go:1549` 두 곳이고 한 요청이 한 행을 쓰므로 동점은 같은 트랜잭션 다중 쓰기·
    데이터 이관·시계 해상도에서 생긴다. 장애 수정이 아니라 **페이지 경계의 결정론을 세우는 수정**이라고 적을 것.
  - `docs/api.md:10` 문장은 "다른 목록은 아직 이 보증이 없어…" 로 범위를 명시한다. 코드만 고치고 문서를 두면
    정본이 어긋난다 — 반드시 같이 갱신하고, **보증을 실제보다 넓게 쓰지 말 것**.

- 차선 후보: **공개 카탈로그 `q` 의 리터럴 특수문자(`\ % _`) 검색을 실제 PG/Router 로 고정**
  (가치 2 / 위험 1 / M). `likeEscaper`(`api.go:842`)는 `\ % _` 를 escape 하지만 같은 핸들러의 `tags=ANY` 는
  원문 `$2` 를 쓰므로 계약이 둘이다. 기존 `api_test.go` 테스트는 헬퍼 출력만 보고 endpoint 회귀가 없다.
  운영자 지침("같은 종류의 파서가 여러 경로에 있으면 통합하지 말고 **좁히는 방향으로만**")에 따라
  두 경로를 합치려 하지 말고 **현재 동작을 테스트로 고정**하는 것까지만 할 것.
  1순위로 넘어갈 조건: `extended.go:581` 에 이미 `s.id` 가 들어가 있거나, 동점 Red 가 재현되지 않을 때.

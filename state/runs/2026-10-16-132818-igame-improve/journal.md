# 회차 노트 2026-10-16-132818-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:28] base pinned — main@b718a25
- [러너 13:28] autonomy release — 

## 정찰 노트
- 배정된 우선 과제(verify-failed / 릴리즈 2회 실패)는 **낡은 적재**다: shepherd 의 fix-summary·verify.gate.json, approve 의 stages.json/release.json(v0.7.33 published, 자산 1개 verified), ci-b718a25c9a7e.json(check-run 4개 전부 success), 그리고 worktree 의 Go 핀 8곳이 모두 1.26.9 로 일치하는 것을 직접 확인했다. 고칠 실패가 없으므로 일반 회차로 돌렸고, 워크플로는 손대지 않는다.
- 그래서 1순위는 `extended.go:581 adminRankings` 의 `s.id DESC` tiebreak — 같은 수법이 이 저장소에서 4회 연속 채택·릴리즈됐고, 보조 키 없는 페이지 목록이 **이제 이 한 곳만** 남았다. 프로덕션 1파일 한 줄 + 테스트 1 + 문서 1 = 3파일로 "6파일 안쪽" 규칙에 넉넉히 들어간다. favorite 격리·q escaping 은 결함 근거가 없어 제쳤고, CI 에 make test-db 배선은 보호 경로라 제외했다.
- 추측으로 적은 것(구현자가 확인할 것): GitHub 의 v0.7.33 태그 런 상태를 `gh` 권한이 없어 직접 조회하지 못했다 — 쓸 수 있으면 먼저 확인하고, 아직 붉으면 그 로그가 이 과제서보다 우선이다. 또 `?game_id=` 로 범위를 좁히라고 쓴 것은 핸들러 SQL(`s.game_id::text=$2 OR g.slug=$2`)을 읽은 근거이고 실행해 본 것은 아니다.
- 조심할 것: `rankings_pg_test.go` 의 `newRankingFixture`(3000명 + ANALYZE)를 재사용하지 말고 `history_order_pg_test.go` blueprint 를 복사할 것. scores 의 `session_id NOT NULL UNIQUE` 때문에 점수당 세션 1개가 필요한 것이 유일한 추가 비용이다. `limit=1` 단독 순회나 "못 본 행 UPDATE" Red 레시피는 과거에 각각 플래키/Red 미재현으로 실패했으니 쓰지 말 것.
- 못 한 것(정직하게): `technology:implementation-planning` 스킬은 예산 한도가 가까워 불러오지 않았다. 과제서에는 solution-exploration 의 수렴(옵션 비교 후 1개 추천 + 가장 무거운 가정 명시)과 estimating-and-contingency 의 작업분해·범위·contingency/관리예비 분리를 반영했다.
- [러너 13:34] scout done — 관리자 점수 심사 목록(`adminRankings`)의 페이지 정렬에 유일한 tiebreak 추가 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- `adminRankings`(extended.go:581) 의 `ORDER BY s.created_at DESC` 에 `, s.id DESC` 를 덧붙였다 — OFFSET 은 정렬이 전순서일 때만 페이지를 겹치지 않게 가르므로 `created_at` 동점 점수가 두 페이지에 나오거나 어느 페이지에도 안 나올 수 있었다(심사 누락). 프로덕션 1파일 한 줄 + 신규 PG 테스트 1 + docs/api.md 한 구절.
- Red 는 예상보다 강했다: 보조 키 없는 질의가 **같은 순회 안에서도** 창마다 동점 블록 순서를 바꿨다(`009,002,003,004,005,006,001` — `002` 가 앞으로, `001` 이 끝으로). 변이(`,s.id DESC` 제거)로 인과를 확인하고 복원했다.
- 확신 없는 곳: ① 우선 과제가 낡은 적재라는 판정은 Go 핀 8곳 전부 1.26.9 일치 + PR #40 머지 + VERSION 0.7.33 으로 확인했지만, 이 세션에도 `gh` 권한이 없어 **GitHub 의 v0.7.33 태그 런 상태는 직접 조회하지 못했다**. ② docs/api.md 에 "`offset` 을 받는 목록은 이제 모두 이 보증 범위" 라고 적은 근거는 `pageParams(r)` 호출 9곳의 ORDER BY 를 읽은 것이다 — 6곳은 offset 을 쓰고 전부 유일 보조 키가 있고, 3곳(catalog.go:912, defense.go:1750, realmguard.go:1792)은 `limit` 만 쓰는 상위 N 이라 페이지 경계가 없다. 테스트로 덮은 것은 이 중 rankings 한 곳뿐이다.
- 일부러 하지 않은 것: 새 인덱스 추가(관리자 저빈도 경로이고 `scores_rank_idx` 는 이 질의용이 아니다), `moderateRanking`/`excludeRanking` 수정, 워크플로·auth·migrations 접근, `q` escaping 차선 후보(1순위가 성립했다).
- 다음 역할이 조심할 것: `rankings_order_pg_test.go` 는 **`IGAME_TEST_DSN` 이 있어야 돈다** — 없으면 skip 되고 그 PASS 는 증거가 아니다. CI 에는 PG 배선이 없으니 회귀 확인은 `make test-db DSN=…` 로 해야 한다. 점수마다 `session_id NOT NULL UNIQUE` 때문에 전용 `game_sessions` 행이 하나씩 필요하다.
- [러너 13:40] brief accepted — 채택 — 근거가 지금 코드와 정확히 일치했고(extended.go:581 의 보조 키 없는 ORDER BY, `items` 만 돌려주는 응답, `scores.session_id
- [러너 13:40] verify passed — 검증 4개 통과 (policy)

## 비평 노트
- 직접 돌렸다: 일회용 postgres:17-alpine(전용 확장 스키마) 로 `-run '^TestAdminRankings' -count=3` 3개 전부 PASS, `,s.id DESC` 제거 변이에서 구현 노트와 **같은 출력**으로 158/176 두 줄 Red(`009,002,…,001`), 복원 후 `internal/api`+`internal/database` 전체 PASS(35.9s/1.9s), `go vet` 0, 잔여 `api_test_`/`migrate_` 스키마 0개·pgcrypto 보존, 컨테이너 제거·worktree clean. 구현 노트의 "확신 없는 곳 ②"(offset 목록 전부 보조 키)도 `pageParams` 호출 9곳 ORDER BY 를 직접 읽어 사실로 확인했다.
- 못 본 것: GitHub 런 상태(`gh` 권한 없음 — 이 세션도 동일), web/SDK·릴리즈 경로(이번 diff 무관).
- 승인이어도 남는 우려 ①: 새 테스트의 대조군 ID 배치(`newer`=…009 최대, `older`=…000 최소)가 created_at 순서와 ID 순서를 동시에 만족해서, `ORDER BY s.id DESC` **단독**이어도 세 테스트가 모두 통과한다 — 1차 키가 created_at 이라는 보증은 이 파일이 고정하지 못한다. 다음 회차가 대조군 ID 를 동점 블록 사이 값으로 바꾸면 한 줄로 메워진다.
- 우려 ②: CI 에 PG 배선이 없어 이 회귀는 `make test-db DSN=…` 를 돌리는 사람에게만 존재한다(저장소 기존 속성). 우려 ③: docs/api.md 의 "offset 목록 전부" 는 오늘 참이지만 새 목록이 추가되면 조용히 거짓이 되는 선언이고 이를 강제하는 테스트·린트가 없다.
- 프로필 정비: 프로필의 `migrations/001..004` 는 낡았다(실제 001..010), `extended.go:577`→실제 581. 릴리즈 노트는 "관리자 점수 심사 목록 페이지 경계 결정론" 한 줄로 충분하다.
- [러너 13:45] review approved — 리뷰 승인 (risk=low)
- [러너 13:45] pr created — https://github.com/hkjang/igame/pull/41
- [러너 13:50] ci passed — 검사 1개 모두 success
- [러너 13:50] merge done — 539c244

## 릴리즈 노트
- `v0.7.34` 로 패치 릴리즈했다. 이전 3회 태그(v0.7.31~33)의 방식을 그대로 따랐다: 0.7.33→0.7.34 를 15개 파일 26줄에 치환(VERSION, docker-compose 이미지 태그, web/SDK package·package-lock, README 2곳, docs 7개 파일), README 58행 릴리즈 노트 단락을 이번 수정 내용으로 재작성, `make docs-pdf` 로 두 엔터프라이즈 매뉴얼 재생성 → 17파일 26 insertions/26 deletions 으로 직전 릴리즈 커밋과 같은 footprint. 커밋 `c44772f` (`feat: release igame v0.7.34`, 4단락 양식 동일), 주석 태그 `v0.7.34` (메시지 `igame v0.7.34`, 이전과 같이 서명 없음).
- 이전 릴리즈가 밟던 검증을 전부 재실행해 통과했다: `make check-contract`("Release contract verified for igame:v0.7.34"), `make deps`, `make lint`, `make test`(Go 전부 ok / SDK / web 41파일 246테스트), `make test-race`, `make build`(offline bundle 검증 "igame 0.7.34, Phaser 3.90.0"), 일회용 postgres:17-alpine 로 `make test-db`(api 37.179s / database 2.048s), `IGAME_TEST_DSN` 으로 `-run '^TestAdminRankings' -race -count=5`(8.409s, skip 없음), npm audit 2개 모두 0 vulnerabilities, `govulncheck@v1.6.0` reachable 0·uncalled 3, `git diff --check` 무출력. 실행 후 잔여 `api_test_`/`migrate_` 스키마 0개·pgcrypto 는 `igame_test_extensions` 에 보존 확인하고 컨테이너 제거.
- `github_release` 는 **false**, `assets` 는 **빈 배열**이다 — `.github/workflows/release.yml` 이 `v*` 태그 push 에서 `softprops/action-gh-release`(`generate_release_notes: true` + 고정 본문)로 Release 를 만들고 `igame-v<ver>.tar.gz` 하나만 올린 뒤 "asset 정확히 1개" 를 스스로 검증한다. 본문의 archive·SBOM SHA-256 은 그 런이 빌드한 이미지에서만 나오므로 로컬에서 자산을 만들거나 노트를 미리 써 두지 않았다. 절대 규칙대로 push·태그 push·release 생성은 하지 않았다.
- 로컬에서 하지 못한 것(정직하게): 태그 런이 돌리는 배포 서비스 기반 게이트 — clean-daemon load, PostgreSQL readiness, Bootstrap 로그인·관리자 API smoke, RealmGuard/Defense Series smoke, Playwright 브라우저·axe 게이트, Grype 이미지 스캔, SBOM 의 Phaser·gamehub-js 식별 — 는 이 기계에서 수행하지 않았다. `make release`(docker save archive)도 만들지 않았다: 워크플로가 같은 것을 빌드·게시하고 자산 경로에 올릴 것이 없다. 이전 세 회차와 동일하게 `gh` 권한이 없어 직전 태그 런의 GitHub 상태는 이번에도 직접 조회하지 못했다.
- 다음 회차로 넘기는 것: 비평 노트의 우려 ①(새 테스트의 대조군 ID 가 created_at 순서와 ID 순서를 동시에 만족해 `ORDER BY s.id DESC` 단독으로도 통과 — 1차 키가 created_at 이라는 보증이 고정되지 않았다)은 릴리즈가 손댈 범위가 아니라 그대로 남겼다. 대조군 ID 를 동점 블록 사이 값으로 바꾸는 한 줄 과제이고, 보류 아이디어 목록보다 가치가 높다.
- [러너 14:02] release published — v0.7.34
- [러너 14:19] assets verified — v0.7.34 자산 1개 (이전 v0.7.33: 1)

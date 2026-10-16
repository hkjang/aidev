# 회차 노트 2026-10-16-111600-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:16] base pinned — main@5969bec
- [러너 11:16] autonomy release — 

## 정찰 노트
- 고른 이유: 페이지 정렬 캠페인에서 사용자가 직접 보는 마지막 경로가 playHistory(catalog.go:990)다 — 관리자 3개와 공개 카탈로그는 이미 착지했고 같은 수법이 세 회차 연속 release-ready였다. 특수문자 검색·favorite 격리 테스트(가치 2)와 README 정합성은 제쳤고, serviceLocation·playWindows·api_keys 가설은 반복 실패라 rejected로 내렸다. 차선은 extended.go:581 adminRankings(fixture 비용이 더 커서 2순위).
- 추측으로 적은 것: Red가 "삽입 순서 [1..6] vs 계약 [6..1]"로 나온다는 것은 실행 계획 가정이다 — 이번 회차에는 PG를 띄우지 않았고 테스트·make test-db를 실행하지 않았다. 현장에서 started_at 동점이 실제로 생긴 사례도 미확인이므로 "장애 수정"으로 쓰지 말 것.
- 조심할 것: insertTestPlaySession은 id를 돌려주지 않으니 동점 행은 새 테스트 파일에서 명시적 UUID로 직접 INSERT하고 session_token_hash를 행마다 유일하게 줄 것. 기존 fixture와 이미 착지한 ORDER BY는 건드리지 말고 docs/api.md:10 보증 문장만 한 구절 늘릴 것.
- [러너 11:21] scout done — 본인 플레이 기록(GET /api/v1/me/history)의 페이지 정렬에 유일한 tiebreak 추가 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 무엇/왜: `catalog.go:990 playHistory` 의 `ORDER BY gs.started_at DESC` 에 `,gs.id DESC` 한 조각을 더해, `started_at` 동점 행의 OFFSET 페이지 배치를 결정론으로 만들었다(관리자 목록 `created_at DESC,id DESC` 와 같은 방향). 프로덕션 1파일·1줄 + 신규 PG 테스트 1파일 + `docs/api.md` 한 구절. 커밋 73dfb09.
- 확신 없는 곳: (1) 수정 전 Red 가 "삽입 순서 [1..6]" 로 난 것은 이 PG17 인스턴스의 실행 계획에 의존한다 — 다른 환경에서 Red 의 **구체적 순서**는 달라질 수 있다(단정은 계약 `[6..1]` 에 고정했으므로 Red/Green 자체는 안정적이다). (2) 현장에서 `started_at` 동점이 실제로 생긴 사례는 **미확인**이다 — 프로덕션 INSERT 경로(catalog.go:303)는 한 요청이 한 행만 쓴다. 장애 수정으로 읽지 말 것. (3) `-race -count=5` 는 통과했지만 플래키 여부를 그 이상 반복해 보지는 않았다.
- 검증 못 한 것: CI 에는 PG 서비스·`IGAME_TEST_DSN` 배선이 없어 이 테스트는 **CI 에서 skip 된다** — PG 회귀는 로컬 컨테이너에서만 돌았다. `go test ./...`(DSN 없이)의 PASS 는 이 경로의 증거가 아니다. web/SDK 는 이번 변경과 무관해 빌드하지 않았다.
- 일부러 하지 않은 것: 차선 후보 `extended.go:581 adminRankings` 의 `,s.id DESC` — 과제서가 범위 밖으로 지정했고 fixture(scores 1:1 session)가 더 커서 파일 수를 늘리지 않으려 뺐다. 신규 인덱스도 추가하지 않았다(`game_sessions_user_idx(user_id, started_at DESC)` 가 이미 있고 limit ≤ 200). 기존 fixture·이미 착지한 ORDER BY·감사 경로는 손대지 않았다.
- 다음 역할이 조심할 것: `internal/api/history_order_pg_test.go` 는 **실제 PostgreSQL 17 이 있어야 돈다**(`IGAME_TEST_DSN` 없으면 skip). 준비는 README 의 `postgres:17-alpine` + `igame_test_extensions` 스키마에 pgcrypto(public 확장은 `migratedPool` 이 거부). 실행 뒤 잔여 `api_test_`/`migrate_` 스키마 0개·확장 스키마 보존을 확인했고 직접 만든 컨테이너만 제거했다. 세션 ID 는 `30000000-…-00000000000{0,1..6,9,a}` 고정값이라 같은 prefix 를 쓰는 테스트를 새로 만들면 충돌한다.
- [러너 11:25] brief accepted — 채택 — 근거가 지금 코드와 정확히 일치했고(catalog.go:990 의 보조 키 없는 ORDER BY, `items` 만 돌려주는 응답, `insertTestPlaySess
- [러너 11:26] verify passed — 검증 4개 통과 (policy)

## 비평 노트
- 확인한 것: 일회용 postgres:17-alpine 컨테이너를 직접 띄워 Red/Green 을 독립 재현했다 — 3개 PASS, `catalog.go:990` 의 `,gs.id DESC` 만 되돌리면 `-count=3` 세 번 모두 앞의 2개가 FAIL(ledger-entry.md:5 출력과 바이트 일치), 복원 후 `git status --porcelain` 무출력. PG 배선 전체 `go test ./...` 전부 ok, `go vet`·gofmt 무출력, 잔여 `api_test_`/`migrate_` 스키마 0개·pgcrypto 보존 확인 후 내 컨테이너만 제거.
- 판정 근거: 보조 키 1개 추가뿐이고 `WHERE gs.user_id=$1`·응답 필드·권한(profile:read)은 불변. 신규 수집·전송·의존성·암호·마이그레이션이 없어 security/legal 차단 사유 없음. docs/api.md:10 문구는 코드와 일치.
- 못 본 것: 현장에서 `started_at` 동점이 실제로 발생한 사례(구현자와 동일하게 미확인) — 릴리즈 노트에 장애 수정으로 쓰지 말 것. web/SDK 빌드는 소비자가 `client.ts:214` 하나이고 집합이 불변이라 생략했다.
- 승인이어도 남는 우려: `TestHistoryKeepsOtherPlayersSessionsOut` 은 수정 전에도 통과하므로 이번 수정의 증거가 아니다(같은 엔드포인트의 격리 가드). CI 에는 PG 배선이 없어 이 파일은 skip 되니 CI 녹색을 이 경로의 회귀 증거로 세지 말 것.
- 다음 회차: 보조 키 없는 페이지 목록은 `extended.go:581 adminRankings` 한 곳만 남았다.
- [러너 11:30] review approved — 리뷰 승인 (risk=low)
- [러너 11:30] pr created — https://github.com/hkjang/igame/pull/40
- [러너 11:32] ci failed — 성공이 아닌 검사: test=failure · 실패한 검사: ? 잡: test 

# igame 프로필 (2026-10-16)
- 기준: main@b718a25, VERSION 0.7.33. 이전 프로필(2026-10-16, 0.7.32)에서 **Go 핀 1.26.6→1.26.9**,
  playHistory 착지, 남은 정렬 경로 한 곳만 갱신했다. 이번 회차는 읽기만 했고 테스트는 실행하지 않았다
  (아래 소요 시간은 이전 회차 실측치).
- 목적: 사내 폐쇄망의 게임 카탈로그·세션·랭킹·업적 및 RealmGuard/Defense를 단일 오프라인 이미지로 제공한다.
- 스택: Go 1.26.9(핀), chi v5, pgx v5, go-oidc, PostgreSQL 17; React 19/TypeScript/Vite 7/MUI 7/Phaser 3.90,
  vitest, JS SDK.
- 구조:
  - cmd/igame — config.Load → DB 연결·migration·bootstrap → secretbox.New → HTTP :8080.
  - internal/config — 네 필수 환경변수만 로드, ENCRYPTION_KEY는 plain/hex:/base64:로 정확히 32바이트 검증.
  - internal/api — REST/SSE/MCP. catalog.go(공개 카탈로그·세션·랭킹·기록), admin.go,
    extended.go(토너먼트·공지·배너·**점수 심사 adminRankings:577**), auth.go, apikeys.go,
    realmguard*.go, defense*.go, auditexport.go. 라우팅은 api.go:178~(admin 그룹 미들웨어
    `requireRole("admin","operator")`, 일부만 `With(requireRole("admin"))`).
  - internal/api/*_pg_test.go — migratedPool(admin_pg_test.go:29)이 DSN마다 UUID 전용 스키마를 만들고
    자기 스키마만 제거한다. 공용 헬퍼: insertTestUser(:81, 역할 인수), insertTestSession(:94),
    insertTestGame(catalog_pg_test.go:27), insertTestPlaySession(:38, **id 미반환**),
    newAdminListFixture, newFinishFixture(session_finish_pg_test.go:33).
    정렬 테스트 blueprint: **history_order_pg_test.go**(명시적 id 삽입 + 실제 Router + 쿠키 왕복).
    rankings_pg_test.go 의 newRankingFixture 는 3000명 + ANALYZE 로 **무겁다 — 재사용 금지**.
  - migrations/001..004_*.sql — embedded, 체크섬·트랜잭션 적용. 적용된 SQL은 불변으로 취급한다.
    scores(001:153): id uuid PK, **session_id NOT NULL UNIQUE FK game_sessions**, created_at default now().
    마이그레이션은 scores/game_sessions 를 seed하지 않는다(games 카탈로그는 한 문장으로 seed → created_at 동점).
  - internal/secretbox — AES-GCM(v1: 접두사 + AAD "igame:v1"), secretbox_reject_test.go 로 거부 경로 고정.
  - internal/battle/realmguard · web/src/games — 결정론 전투의 Go/TS 쌍.
  - web · sdk/gamehub-js · docs · scripts · .github/workflows.
- 빌드·테스트: make deps(네트워크); make lint; make test; make test-race; make web-build; make build;
  make check-contract; make test-db DSN=…(PG 필요, api 약 31–37s / database 약 2s). docker-build/release는 느리다.
  - PG 준비: postgres:17-alpine 컨테이너(README) + `CREATE SCHEMA igame_test_extensions;
    CREATE EXTENSION pgcrypto WITH SCHEMA igame_test_extensions;`. public 확장은 migratedPool이 거부한다.
    DSN 없이 돌리면 api 54개·database 4개가 skip된다 — 그 PASS는 PG 회귀의 증거가 아니다.
- 관례: 영어 `feat:/fix:/test:/docs:` 커밋. VERSION/web/SDK 버전 정렬. 설정은 네 환경변수 + DB system_settings
  (새 환경변수 금지). 오류 코드·메시지는 영어, 한국어 표시는 web/src/api/errorMessages.ts.
  감사 detail에 보호 대상 원문을 넣지 않는다. 문서는 docs/(api.md·security.md·release.md),
  목록 계약은 **docs/api.md:10**.
- 페이지네이션 계약(현 상태): pageParams(api.go:824)는 limit 기본 50·최대 200, offset 음수는 0.
  유일 키 보조 정렬이 **있는** 곳 — admin.go:580 users / :718 audit / :755 games(모두 `created_at DESC,id DESC`),
  catalog.go:130 공개 목록(`g.name,g.id`), catalog.go playHistory(`gs.started_at DESC,gs.id DESC`),
  realmguard.go:1879 랭킹(`score DESC,created_at,id`).
  **없는 곳은 단 한 곳 — extended.go:581 adminRankings(`ORDER BY s.created_at DESC`)**.
  docs/api.md:10이 이 보증 범위를 글로 적어 두므로 코드와 함께 갱신한다.
- Go 핀: 1.26.9 가 **8곳**에 흩어져 있고 그 중복이 곧 게이트다 —
  go.mod:3, Dockerfile:21/59, ci.yml:109/120, release.yml:122/133, scripts/check-release-contract.sh:14/74/76/110.
  상향할 때 전부 함께 올린다(단일화하지 말 것 — 계약 검사를 약화시킨다).
- 위험 구역: auth/session 발급·개인 키, migrations 체크섬, .github/workflows 게이트,
  RealmGuard Go/TS 동치성, secretbox 포맷/AAD, 결제 성격의 보상·점수 심사.
- 자주 깨지는 곳: ① release/ci 의 `Scan reachable Go vulnerabilities`(govulncheck v1.6.0) — 2026-10-16 에
  Go 1.26.6 stdlib 취약점 10건으로 2회 실패, a1341ce 의 1.26.9 핀 상향으로 해결. Go stdlib CVE 가 나오면
  **머지와 무관한 PR 에서도** 같은 자리가 붉어진다. ② release 의 dev 포함 npm audit vs CI production-only audit
  차이(2026-09-10 2회 실패, f508458). **둘 다 게이트 완화로 해결하지 않는다 — 핀/의존성을 올려 고친다.**
  ③ 페이지 정렬 테스트는 `limit=1` 단독 순회·미조회 행 UPDATE 방식에서 플래키했다(명시적 id 삽입 방식만 쓸 것).
- 검증 함정: CI에는 PG 서비스·IGAME_TEST_DSN·make test-db 배선이 없다(release.yml의 PG는 smoke용).
  새 테스트가 0개 매칭인데 PASS 하는 것은 증거가 아니므로 `-run` 패턴과 -v 출력을 확인한다.
  실행 뒤 잔여 `api_test_`/`migrate_` 스키마 0개와 전용 확장 스키마 보존을 확인한다.
  govulncheck 은 go.mod 의 Go 버전을 GOTOOLCHAIN 으로 쓰므로 로컬 재현 시 버전을 맞춰야 한다.
- 기각된 가설(재선정 금지): serviceLocation Asia/Seoul fallback(호출처 6곳, 쪼갤 모양 없음),
  playWindowsAllow fail-open(가설 오류), loadAPIKeyPolicyContext ErrNoRows(도달성 5회 미증명),
  settings PG fixture updated_by 복원(증상 없음, 7회 보류 후 기각), Go 핀 단일화(게이트 약화).
- 로컬 지침: git tracked CLAUDE.md/AGENTS.md/로드맵 없음. internal/cmd/web/sdk/docs에 TODO/FIXME 없음
  (이전 회차 확인, 이번 미재확인). 이 세션에서는 `gh` 실행 권한이 없었다 — GitHub 런 상태는
  러너가 남긴 state/runs/*/ci-*.json·stages.json 으로 확인한다.
- 조직 스킬: 이 세션에는 Skill 도구로 pmo:estimating-and-contingency, technology:implementation-planning,
  technology:solution-exploration 이 실렸다. brief에 작업분해·범위 추정·contingency/관리 예비 분리를 반영한다.

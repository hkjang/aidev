- 과제: 공개 게임 목록의 동명 게임을 ID로 보조 정렬하여 페이지 순서를 고정하기 (가치 3 / 위험 2 / 작업량 M)
- 왜: `internal/api/catalog.go:listGames`는 `ORDER BY g.name`만 사용하고 games.name은 유일하지 않아, 같은 이름의 게임을 페이지로 나누는 순서가 정해지지 않는다. 실제 서버에서 이미 반환된 게임의 description만 바꾼 뒤 중복·누락을 관측했으며, `g.id` 보조 정렬은 이름과 조회 집합을 유지하는 수정 사이에도 같은 페이지 순서를 제공한다.
- 수용 기준: 1) 로그인한 `GET /api/v1/games`는 기존 이름 오름차순을 유지하면서 동명이면 UUID ID 오름차순으로 반환하고 `items/limit/offset`, active·q·category·favorite 필터 계약을 유지한다. 2) 동일 이름의 active 게임 6개를 ID 역순으로 넣고 `limit=2`, `offset=0,2,4`로 순회하면 예상 ID 오름차순과 정확히 일치하며 각 ID는 한 번씩만 나온다. 이름이 다른 대조군도 두어 ID가 이름보다 앞선 정렬 키가 되는 회귀를 막는다. 3) 실제 PostgreSQL과 `New(...).Router()` 및 실제 세션 쿠키로 조회하고, 첫 페이지의 이미 본 게임을 실제 관리자 PUT으로 수정해 description만 바뀌고 name/id/status/필터 집합은 유지됨을 확인한 뒤에도 예상 순서와 유일성이 유지된다. 수정 전에는 명시적 ID 정렬 단정이 실패하고 수정 후에는 통과해야 한다. 중복 자체의 발생 여부만으로 Red를 판정하지 않는다.
- 건드릴 파일: `internal/api/catalog.go:listGames` — 질의의 `ORDER BY g.name`을 `ORDER BY g.name,g.id`로 변경(프로덕션 1파일); `internal/api/catalog_order_pg_test.go`(신규, 제안 테스트명 `TestCatalogListOrdersTiedNamesByID`, `TestCatalogListKeepsOrderAcrossDescriptionUpdate`) — 실제 HTTP/PG 회귀; `docs/api.md:공통 규칙` — 기존 관리자 목록 보증에 `/api/v1/games`의 name/id 보조 정렬을 명시하고 다른 목록·동시 삽입/삭제에 관한 한계를 유지. 총 3파일로 제한한다.
- 검증 명령: 아래 DB 준비 후 `make test-db DSN='postgres://igame:igame@127.0.0.1:15432/igame?sslmode=disable&search_path=public,igame_test_extensions'`; 신규 테스트 작성 후 `IGAME_TEST_DSN='postgres://igame:igame@127.0.0.1:15432/igame?sslmode=disable&search_path=public,igame_test_extensions' go test ./internal/api -run '^TestCatalogList' -race -count=5 -v`; `go test ./cmd/... ./internal/... ./migrations/... -count=1`; `go vet ./cmd/... ./internal/... ./migrations/...`; `gofmt -l internal/api/catalog.go internal/api/catalog_order_pg_test.go`; `git diff --check`.
- 위험과 피할 것: auth/session, migrations, workflows, 설치키, RealmGuard 전투 코드와 기존 관리자 목록은 변경하지 않는다. 공유 `gameSelect`·`scanGame`, API 필터/응답 모양, 새 인덱스, 페이지 방식도 그대로 둔다. 정렬 키·필터 값 변경이나 행 추가/삭제까지 OFFSET snapshot 일관성을 보증하지 않는다. 정찰의 description UPDATE 중복 재현은 1/5였으므로 “힙 이동으로 항상 실패”라고 가정하지 않는다. `limit=1`의 불안정한 정렬 결과, 미조회 행 수정, SQL 문자열 검사, principal 직접 주입/핸들러 직접 호출은 회귀 증거로 쓰지 않는다. DSN 없는 PASS는 PG 테스트가 skip하므로 DB 검증을 대체하지 못한다. 감사 detail에는 게임 식별자만 쓰는 기존 코드 유지.
- 차선 후보: README RealmGuard 서버 재현 설명 정합성 개선 (가치 2 / 위험 1 / S) — 이미 이 정렬 변경이 별도 착지했거나 10분 내 실제 PG 회귀를 구성할 수 없으면 선택. README 44행의 공식 결과를 `server_received_telemetry_v1`만으로 설명하고 완전한 서버 시뮬레이션이 아니라고 하는 문장을 42행, `internal/api/realmguard.go`의 `replayRealmGuardBattle` 호출, `realmguard_replay.go:realmGuardReplayMethod`, `docs/security.md` 44–46행과 맞춘다. README 한 단락만, RealmGuard 서버 재현과 telemetry의 추가 검증 역할을 구분하고 Defense 설명은 유지. 이 차선의 코드 근거는 직접 확인했으며 문서 빌드/PDF 재생성은 필요하지 않다.

범위·근거와 실제 정찰 결과:
- 기준 main@a84073a, VERSION 0.7.31. 최근 관리자 목록 정렬 PR은 `admin.go`에만 적용됐고 공개 목록은 별개다. 설치키 계약 테스트는 이미 9003e19로 착지했으므로 반복하지 않는다.
- `migrations/001_initial.sql`의 games는 id PK, slug UNIQUE, name은 NOT NULL뿐이다. `gameInput.normalize`·`createGame`에서도 name 중복을 막지 않으며, 실제 관리자 POST로 동일 이름 6개를 생성했다. `/api/v1/games`는 `api.go:Router`의 requireAuth/enforceAPIKeyPermissions 그룹에 있다. 여기서 “공개”는 active 카탈로그라는 뜻이며 익명 접근 허용이라는 뜻이 아니다.
- 정찰은 제품 소스를 수정하지 않고 `go run ./cmd/igame` + 일회용 PG17로 서버를 띄워 실제 POST `/api/v1/auth/login`, 관리자 POST/PUT/DELETE와 GET `/api/v1/games`를 호출했다. 이름이 같은 6개를 limit=2로 순회하며 첫 페이지 첫 행의 description만 바꾼 5회 중 1회에서 동일 UUID 중복 1개·누락 1개를 관측했다. 나머지 4회는 중복이 없었다.
- 주 회귀 레시피는 별도 실측했다. 한 INSERT로 동일 이름의 6개를 고정 UUID 끝자리 6,5,4,3,2,1 순서로 넣고 실제 로그인 HTTP로 0/2/4 offset을 조회했다. 5회 모두 실제 끝자리 순서는 `[5,6,4,3,2,1]`, 새 계약의 예상은 `[1,2,3,4,5,6]`이었다. 이 예상 순서를 테스트에서 직접 계산/명시하고 비교한다. UUID 역순 삽입은 fixture일 뿐이며 production SQL의 문자열 모양을 단정하지 않는다.
- 수정 후 Green은 정찰에서 미확인(코드 변경 금지). 구현자는 새 테스트를 먼저 실행하여 Red를 남기고 변경 후 재실행한다. 새 이름 대조군 및 통합 fixture의 Red 재현도 구현 시 확인할 부분이다.
- 기존 검증 실측: `make test-db DSN=...` PASS(api 31.407s, database 1.828s), `go test ./cmd/... ./internal/... ./migrations/... -count=1` PASS, `go test ./internal/config -count=1` PASS. 새 테스트명은 제안이며 현재 파일/함수는 아직 없다. vet/race는 정찰 미실행이며 구현 후 검증이다.

실행 순서와 체크포인트(전부 구현자가 자동 판정, 사람 승인 불필요):
1. [미착수] 새 테스트 파일에서 `admin_total_pg_test.go:newAdminListFixture`를 재사용한다. 이 fixture는 `migratedPool`, `insertTestUser`, `insertTestSession`, 실제 `New(...).Router()`를 사용한다. 테스트마다 전용 스키마이므로 같은 고정 UUID를 다른 테스트에서 재사용해도 되며 q에는 UUID 기반 고유 이름 접두사를 사용한다. `admin_order_pg_test.go:pageIDs`는 호출 경로를 받으므로 재사용 가능하다. `insertGamesSharingOneTimestamp`는 이름이 서로 다르므로 이번 fixture에 그대로 쓰지 않는다. 신규 테스트에서 순서 단정이 실제 응답을 보고 실패하는지 DSN을 지정해 `go test ./internal/api -run '^TestCatalogList' -count=1 -v`로 확인한다. 실패가 연결/로그인/fixture 실패면 정렬 Red로 기록하지 않는다.
2. [미착수] catalog.go의 ORDER BY 한 곳만 변경하고 같은 명령을 실행해 Green을 확인한다. description 수정은 `admin.go:updateGame`의 PUT이 모든 입력 필드를 저장한다는 점에 주의한다. name/slug/game_url/game_type/status 및 사용하는 필터 필드를 유지한 본문을 보내고 description만 다르게 한다. 기존 `putGame`은 description을 받지 않아 이번 수정 검증에 그대로 쓰면 안 된다. PUT 응답의 변경값과 유지값도 단정한다.
3. [미착수] docs/api.md 한 문장을 변경한 뒤 위의 race 반복·전체 DB·전체 Go·vet·format/diff 검증을 끝낸다. 테스트 이름이 다른 경우 검증 명령도 함께 갱신하고 0개 테스트 실행을 통과로 처리하지 않는다. 해결하려고 다른 목록/fixture의 리팩터가 필요해지면 범위를 넓히지 말고 과제서에 불일치를 기록한 뒤 차선으로 전환한다.

일회용 DB 준비(정찰에서 모두 실행한 명령의 컨테이너 이름만 구현자용으로 분리):
```bash
docker run -d --rm --name igame-impl-order-db --tmpfs /var/lib/postgresql/data -e POSTGRES_PASSWORD=igame -e POSTGRES_USER=igame -e POSTGRES_DB=igame -p 127.0.0.1:15432:5432 postgres:17-alpine
docker exec igame-impl-order-db pg_isready -U igame -d igame
# 준비 완료 후 실행. 아직 준비되지 않았다면 readiness만 다시 확인한다.
docker exec igame-impl-order-db psql -U igame -d igame -v ON_ERROR_STOP=1 -c 'CREATE SCHEMA igame_test_extensions; CREATE EXTENSION pgcrypto WITH SCHEMA igame_test_extensions'
# 위 검증 명령을 실행한 다음 자신이 만든 컨테이너만 제거한다.
docker rm -f igame-impl-order-db
```
포트가 사용 중이면 다른 빈 포트를 고르고 DSN에도 반영한다. 기존 서비스/컨테이너를 재사용하거나 제거하지 않는다. 정찰의 컨테이너는 종료·제거한다. pgcrypto를 public에 설치하면 `migratedPool`이 의도적으로 실패한다.

대안 비교·추정 근거:
- 최소 수정(g.name,g.id + HTTP 회귀)을 권고한다. 정상 관리 API로 동명 게임을 만들 수 있고 출력까지 문제를 관측했으며 제품 1파일로 좁힐 수 있다.
- 다른 두 목록까지 확장하면 타임스탬프·필터 fixture가 더 필요하므로 이번에는 제외한다. keyset pagination은 클라이언트/응답 계약 변경이라 45분 범위를 넘는다. 무변경/문서만은 동명 게임의 실제 불안정성을 남긴다. 카탈로그 데이터 정렬을 클라이언트에서 다시 해도 서버 OFFSET에서 이미 누락된 행을 복구하지 못한다.
- bottom-up 추정: fixture/명시적 순서 Red 10–15분, 한 줄 수정·Green 3–5분, 설명 수정·전체 검증 8–10분 = 기본 21–30분. 알려진 위험(PG 준비·fixture 차이)용 contingency 5–10분을 별도로 두어 총 26–40분, 신뢰는 중간(통계로 보정한 확률 아님). 별도 management reserve는 0분이며 범위 확장을 흡수하지 않는다. 과거 관리자 정렬 작업과 같은 제품 1파일+회귀+문서 형태라 크기는 M으로 비교하되 그 작업의 실제 소요 시간은 미확인이다.
- 가장 큰 가정은 단순 HTTP probe와 새 테스트 fixture에서도 명시적 순서 Red가 재현된다는 것이다. 1단계에서 재평가하며, 무작위 데이터/실행 계획 설정 변경으로 억지 Red를 만들지 않는다.
- 적용 스킬 원문: [estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md), [implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md), [solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md). 전용 Skill 도구가 없어 로컬 원본과 PMO sources 목록을 읽었다. 외부 비용 추정 기준을 원용하지 않고 저장소 실측/명시적 가정으로 추정했다.

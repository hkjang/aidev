# 과제서 — 2026-10-07-175805-igame-improve (igame)

- 과제: 관리자 목록 페이지네이션에 유일한 tiebreak 을 넣어 OFFSET 경계에서 행이 겹치거나 사라지지 않게 하기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `listUsers`(internal/api/admin.go:571)·`listAuditLogs`(:709)·`adminListGames`(:746)은 전부 `ORDER BY created_at DESC LIMIT $2 OFFSET $3` 로만 정렬한다. `created_at` 이 같은 행이 둘 이상 있으면 두 페이지 요청 사이에서 순서가 보장되지 않아 같은 행이 두 페이지에 나오거나 어떤 행은 어느 페이지에도 나오지 않는다 — 감사 추적에서는 "페이지를 끝까지 넘겼는데 그 행이 없다" 가 된다. 같은 저장소의 랭킹 질의는 이미 `ORDER BY score DESC,created_at,id`(realmguard.go:1879) 로 유일 키까지 정렬해 계약이 한쪽만 갈라져 있다. 세 질의에 유일 컬럼 하나를 덧붙이면 페이지 분할이 결정론이 된다.

- 수용 기준:
  1) 동일 `created_at` 을 가진 행이 여러 개 있는 상태에서 `GET /api/v1/admin/users?limit=1&offset=0..N-1` 로 끝까지 넘기면 각 행의 `id` 가 **정확히 한 번씩** 나오고(중복 0, 누락 0), 중간에 그 행들 중 하나를 수정해 물리적 위치가 바뀌어도 그대로다.
  2) 같은 성질이 `GET /api/v1/admin/audit-logs?limit=1&offset=…` 에서도 성립한다(audit_logs 는 `id` bigint 가 있어 `a.created_at DESC,a.id DESC` 가 완전한 tiebreak 이다).
  3) 테스트는 **수정 전에 Red** 여야 한다. Red 를 못 만들면 아래 "Red 만드는 법" 의 두 번째 방법까지 시도하고, 그래도 안 되면 이 과제를 포기하지 말고 "견고성" 으로 범위를 좁혀라 — 그때는 "ORDER BY 에 유일 키가 포함된다" 를 동작으로 고정하는 테스트(동일 created_at 6행을 limit=2 로 3페이지 넘겨 합집합이 전체와 같고 중복이 없음 + 페이지 2 요청 전에 UPDATE 를 한 번 끼워 넣음)만 남기고, 과제서에 "Red 재현 실패" 를 정직하게 적어라. 소스 문자열 검사(`strings.Contains(sql, "id")`)는 금지다.
  4) 기존 `total` 계약(2026-10-04 의 `pagedTotal`)과 감사 CSV 내보내기 결과가 바뀌지 않는다 — `admin_total_pg_test.go`·`auditexport_pg_test.go` 가 그대로 통과한다.

- Red 만드는 법(둘 중 되는 것을 써라. 둘 다 실제 PG17 전용 스키마에서, 실제 `Router()` + 실제 관리자 세션 쿠키로):
  1) **동일 created_at + 중간 UPDATE** (권장, 물리 순서를 확실히 바꾼다): 한 `INSERT … SELECT` 문으로 users 6행을 넣어 `created_at` 을 전부 같게 만든다(`now()` 는 트랜잭션 타임스탬프라 한 문장 안에서 동일). `limit=2&offset=0` 으로 1페이지를 읽고, 그 다음 아직 보지 못한 행 하나에 `PATCH /api/v1/admin/users/{id}` (display_name 변경 — `created_at` 은 건드리지 않는다)를 보내 힙 튜플을 테이블 끝으로 옮긴다. 그 뒤 `offset=2`, `offset=4` 를 읽어 세 페이지의 `id` 합집합을 센다. tiebreak 가 없으면 중복/누락이 나온다.
  2) **top-N heapsort 경계**: UPDATE 없이 동일 `created_at` 8~12행에 대해 `limit=1` 로 offset 0..N-1 을 전부 읽는다. offset+limit 이 달라지면 PG 의 top-N heapsort 크기가 달라져 동점 행의 출력 순서가 달라질 수 있다 — 환경 의존이므로 1) 이 되면 1) 을 쓴다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/api/admin.go:571` `listUsers` — `ORDER BY created_at DESC` → `ORDER BY created_at DESC,id DESC`.
  - `internal/api/admin.go:709` `listAuditLogs` — `ORDER BY a.created_at DESC` → `ORDER BY a.created_at DESC,a.id DESC`.
  - `internal/api/admin.go:746` `adminListGames` — `ORDER BY g.created_at DESC` → `ORDER BY g.created_at DESC,g.id DESC`. (`gameSelect`(catalog.go:120)의 별칭이 `g` 이고 `g.id` 를 이미 선택하는 것을 이번에 확인했다. 단 `gameSelect` 는 `catalog.go:130` 의 공개 목록도 쓰는 상수이므로 **상수 본문은 건드리지 말고** `adminListGames` 의 `ORDER BY` 절만 바꿔라.)
  - `internal/api/<새 이름>_pg_test.go` — 위 Red. **이미 있는 것을 그대로 재사용하라**(이번에 직접 열어 확인했다): `admin_total_pg_test.go:38-53` 의 `adminListFixture` + `newAdminListFixture(t)` 가 `migratedPool(t)`(정의: `admin_pg_test.go:29`, 호출마다 `api_test_UUID` 스키마) → `insertTestUser` → `insertTestSession` → 실제 `New(pool,nil,log).Router()` 를 `httptest.NewServer` 로 띄우는 것까지 다 해 준다. `patchUser(t, id, displayName)`(:79)이 바로 Red 레시피 1) 의 `PATCH`다. `list`(:56)는 `(status, total, len(items))` 만 돌려주므로, **같은 패키지의 새 파일에 `adminListFixture` 의 메서드를 하나 더 붙여** 페이지의 `id` 목록을 돌려받아라(기존 파일은 수정하지 않는다). 동일 `created_at` 6행은 `insertTestUser` 를 6번 부르면 안 된다(호출마다 문장이 달라 `created_at` 이 달라진다) — `insertTestUser` 의 컬럼 목록을 베껴 `INSERT INTO users(...) SELECT … FROM generate_series(1,6)` **한 문장**으로 넣어라.
  - (선택, 한 줄) `docs/api.md` 공통 규칙에 "페이지된 목록은 유일 키까지 정렬되어 페이지 경계가 결정론이다" 한 줄.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l .` (무출력) / `go vet ./...` / `go build ./...`
  - `go test ./cmd/... ./internal/... ./migrations/...`
  - `make test-db DSN='postgres://igame:igame@127.0.0.1:15432/igame?sslmode=disable&search_path=public,igame_test_extensions'` — **DSN 없이 돌린 `go test ./...` 는 DB 검증이 아니다**(PG 회귀가 조용히 skip 된다). 실행 후 `api_test_`/`migrate_` 잔여 스키마 0개 확인.
  - 신규 테스트만: `go test ./internal/api/ -run <새TestName> -race -count=3` (IGAME_TEST_DSN 설정)
  - `bash scripts/check-release-contract.sh` (v0.7.29)
  - 인과 확인용 변이: 세 `ORDER BY` 의 tiebreak 을 하나씩 되돌려 각 테스트가 Red 가 되는지 보고 매번 원복 후 녹색 재확인.

- 위험과 피할 것:
  - **범위를 넓히지 말 것.** `extended.go:581`(점수 심사 목록)·`catalog.go:130`(게임 목록, `ORDER BY g.name`)·`catalog.go:990`(내 세션 기록)도 같은 문제이지만 **이번 회차에는 건드리지 않는다** — 프로덕션 파일을 1개로 유지한다. 남은 것은 ideas.json 의 후속 항목으로 남겨라.
  - `usersFilter`·`auditLogFilter` 상수와 `pagedTotal`, 감사 CSV 내보내기(`auditexport.go`)는 손대지 말 것 — 2026-10-04 에 "목록과 내보내기가 같은 술어를 읽는다" 를 계약으로 고정했다. 정렬만 바꾼다.
  - 오류 코드 문자열·응답 필드명은 프런트/테스트와의 계약이다. 응답 모양을 바꾸지 말 것.
  - 보호 경로(auth, migrations, .github/workflows)는 건드리지 않는다. 새 환경변수 금지. 새 인덱스나 마이그레이션 추가 금지 — `ORDER BY` 에 유일 키를 더하는 것만으로 충분하고, 마이그레이션 체크섬은 불변이다.
  - PG 서브테스트는 부모 pool 을 공유한다. 행 수를 절대 건수로 단정하지 말고 고유 username 접두사와 델타로 단정할 것(과거에 여기서 여러 번 깨졌다).
  - 도달성은 정직하게 적을 것: 동일 `created_at` 은 한 트랜잭션에서 여러 행을 쓰는 경로(일괄 등록, 한 요청이 감사 행을 여럿 남기는 경우)에서 생기고 정상 설치에서 흔하지는 않다 — 운영 장애가 아니라 페이지 경계의 결정론을 세우는 수정이다. 이것을 "데이터 유실 버그" 로 과장하지 말 것.

- 차선 후보: `ENCRYPTION_KEY` 파싱의 거부 경로를 회귀로 고정 (가치 2 / 위험 1 / 작업량 S, 테스트 전용 1파일). `internal/config/config_test.go` 의 `TestParseEncryptionKey` 는 happy 3개와 `"short"` 하나뿐이다. 이번에 읽어 확인한 계약: `validateKey`(config.go:85)는 **정확히 32바이트만** 허용하므로 `secretbox.New` 가 받아들이는 16·24바이트도 기동에서는 거부된다(`cmd/igame/main.go:59` 가 `config.Load` 뒤에 `secretbox.New` 를 호출하므로 config 가 더 엄격한 관문이다). 고정할 것: `base64:`/`hex:` 의 16·24·31·33바이트 거부, `"base64:"`·`"hex:"` 단독(0바이트) 거부, 홀수 길이 hex 거부, 패딩 없는 base64 거부(`StdEncoding`), `Load` 가 키의 앞뒤 공백을 `TrimSpace` 로 떼는 것, 그리고 **오류 메시지가 키 값을 노출하지 않는 것**(부트스트랩 암호 테스트가 이미 같은 단정을 한다). 에러 문구는 비교하지 말고 "에러가 났다" 만 단정. 검증: `go test ./internal/config/... -count=1 -v`, `go test ./internal/config/ -race -count=3`.

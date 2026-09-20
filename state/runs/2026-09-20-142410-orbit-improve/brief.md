# 과제서 — 2026-09-20 orbit

- 과제: `orbitRange` 의 사람×교류 데카르트 곱을 두 스칼라 서브쿼리 `least` 로 바꾸고, 실제 postgres 를 끼는 opt-in 테스트로 같은 값·같은 행 수를 증명한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/server/timetravel.go:140` 의 `SELECT least(min(p.created_at),min(i.occurred_at)) FROM people p LEFT JOIN interactions i ON i.user_id=p.user_id WHERE p.user_id=$1` 은 조인 조건이 `user_id` 뿐이라 사람 N × 교류 M 행을 만들어 min 하나를 구한다(300명 × 1만 교류 = 300만 행). 이 함수는 `data.go:740` 에서 **현재 화면 `/orbit` 을 열 때마다** 호출되므로 우주 화면 첫 로드가 데이터가 쌓일수록 느려진다. 두 테이블을 각각 한 번씩 훑는 스칼라 서브쿼리로 바꾸면 O(N+M) 이 되고 결과는 동일하다(`interactions.person_id REFERENCES people(id) ON DELETE CASCADE` — `001_init.sql:111` — 이라 사람 없는 교류가 없으므로 "사람이 0명이면 NULL" 이라는 옛 질의의 성질도 그대로다).

- 수용 기준:
  1) `orbitRange` 의 SQL 이 `SELECT least((SELECT min(created_at) FROM people WHERE user_id=$1),(SELECT min(occurred_at) FROM interactions WHERE user_id=$1))` 꼴로 바뀌고, 반환 타입·시그니처·호출부(`data.go:740`)·응답 필드 `earliest_at` 는 그대로다.
  2) `internal/server/timetravel_db_test.go`(신규) 에 `ORBIT_TEST_DATABASE_URL` 이 비어 있으면 `t.Skip` 하는 테스트를 두고, 진짜 `store.Open`(마이그레이션 001~005 적용) 위에 (a) 사람 0명 → `nil`, (b) 사람만 있고 교류 없음 → `min(created_at)`, (c) 사람의 `created_at` 보다 앞선 교류가 있음 → 그 `occurred_at`, (d) 다른 사용자의 더 오래된 사람·교류가 섞여 있어도 내 것만 본다 — 네 경우가 새 질의로 맞는지, 그리고 같은 데이터에 옛 질의 문자열을 직접 돌린 값과 새 질의 값이 같은지를 확인한다. 테스트는 `*Server{store: st}` 로 실제 `orbitRange` 메서드를 부른다(SQL 문자열 검사 금지).
  3) `ORBIT_TEST_DATABASE_URL` 없이 `go test -race ./...` 가 skip 으로 초록이고(CI 는 postgres 없음), 이 세션에서는 docker 로 postgres 를 띄워 환경변수를 주고 한 번 실제로 통과시킨 결과(행 수·EXPLAIN 한 줄)를 회차 노트에 적는다.

- 건드릴 파일:
  - `internal/server/timetravel.go:orbitRange` — SQL 한 줄 교체. 주석에 "두 테이블을 따로 훑는다 — 조인하면 사람×교류 곱이 된다" 한 줄.
  - `internal/server/timetravel_db_test.go`(신규) — `openTestStore(t)` 헬퍼(환경변수 없으면 skip, `store.Open(ctx, dsn, nil)`, 테스트 끝에 만든 행 삭제 또는 고유 user_id 로 격리) + `TestOrbitRangeMatchesLegacyJoin` 하나에 서브테스트 (a)~(d). 시드에 필요한 `users`/`people`/`relationships`/`interactions` 의 NOT NULL 열은 `internal/store/migrations/001_init.sql` 에서 직접 확인할 것(이 과제서는 열 목록을 확인하지 않았음 — 미확인). `relationships` 행은 `orbitRange` 가 읽지 않으므로 넣지 않아도 된다(FK 가 요구하면 넣는다).
  - `docs/ARCHITECTURE.md` — 손대지 않아도 됨. 굳이 적는다면 "테스트" 절에 `ORBIT_TEST_DATABASE_URL` 로 켜는 DB 테스트 한 줄만.

- 검증 명령:
  - `gofmt -l internal/ && go vet ./...`
  - `go test -race ./...` (DB 없이 — skip 확인, `-v -run TestOrbitRange` 로 SKIP 문구 확인)
  - DB 를 끼고: `docker run -d --name orbit-recon-pg -e POSTGRES_PASSWORD=orbit -e POSTGRES_DB=orbit -p 127.0.0.1:55470:5432 postgres:16-alpine` (이 환경에 docker 와 `postgres:16-alpine` 이미지가 있음을 확인함; 포트 55432/55434/55439/55444/55450/32774 등은 다른 컨테이너가 쓰는 중이라 피할 것) → `ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55470/orbit?sslmode=disable' go test -race -run TestOrbitRange -v ./internal/server/` → 끝나면 `docker rm -f orbit-recon-pg`.
  - `cd web && npm ci && npx vitest --run && npm run build` — 프런트는 안 건드리지만 릴리즈 게이트라 한 번.

- 위험과 피할 것:
  - `orbitAt`(같은 파일 56행) 의 포함 규칙(`first_met`)은 건드리지 말 것 — 구간 시작점에 `first_met` 을 더하는 것은 동작 변경이라 이번 범위 밖.
  - `auth.go`·`throttle.go`·마이그레이션은 손대지 않는다. 새 마이그레이션 없음(미머지 브랜치들이 006~009 를 선점).
  - 미머지 auto/2026-09-18-1033 브랜치에 `newDBServer` 라는 DB 테스트 헬퍼가 있다(파일명 미확인). 충돌을 줄이려 헬퍼 이름을 `openTestStore` 로 다르게 두고 새 파일에만 쓴다. 환경변수 이름 `ORBIT_TEST_DATABASE_URL` 은 그 브랜치와 **같게** 유지해 나중에 CI 한 줄로 둘 다 켜지게 한다.
  - 테스트는 `users` 표에 진짜 사용자 행을 넣어야 FK 가 통과한다 — `Bootstrap` 을 쓰지 말고(관리자 계정 생성·설정 행 부작용) 직접 INSERT 할 것. `people.id`/`users.id` 가 uuid 이면 `internal/id` 패키지 `id.New()` 를 쓴다(uuid 형식인지 미확인 — 001_init.sql 확인).
  - 옛 질의를 테스트 안에 문자열로 남겨 새 질의와 비교하는 것은 "회귀 기준" 용도이고, 이 문자열이 소스에 있는지 검사하는 식의 테스트는 쓰지 말 것(운영자 규칙).
  - 데이터를 실제로 늘려 시간을 재는 벤치마크는 하지 말 것 — 45분에 안 맞고, 정확성 동등 + 행 수 감소(EXPLAIN 의 조인 노드 사라짐) 로 충분하다.

- 차선 후보: Time Travel API 를 문서에 올리기 — `openapi.go` 의 `/orbit` 에 `at` 쿼리 파라미터(RFC3339 또는 `YYYY-MM-DD`)와 응답 `earliest_at`/`historical`/`at` 를 적고, `docs/guide.md` 2.2 절과 `docs/API.md` 에 한 문단씩; `openAPI` 핸들러를 httptest 로 불러 `/orbit` 의 `parameters` 에 `at` 가 있는지 확인하는 Go 테스트 (가치 2 / 위험 1 / S). 지금 `docs/`·`openapi.go` 어디에도 `?at=`·`earliest_at` 가 없음을 grep 으로 확인했다(v0.5.0 기능이 문서에 없다).

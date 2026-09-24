# 과제서 — 2026-09-24 orbit

- 과제: `orbitAt` 의 "그날 곁에 있던 사람" 포함 규칙과 시점별 기억 수·마지막 교류를 실제 postgres 로 고정 (가치 3 / 위험 1 / 작업량 M)

- 왜: Time Travel 의 심장인 `internal/server/timetravel.go:orbitAt` 은 세 갈래 포함 규칙(`p.created_at<=at` OR `first_met<=at::date` OR 그 이전 교류 존재), 시점 필터가 걸린 기억 수, 1년 창 밖까지 보는 `last_interaction_at` 을 한 줄짜리 SQL 과 주석으로만 지키고 있는데 테스트가 하나도 없다(`grep orbitAt internal/server/*_test.go` → 없음; 붙어 있는 `orbitRange` 만 `timetravel_db_test.go` 가 덮는다). 이 저장소는 최근 네 회차 중 세 번이 Time Travel 주변을 건드렸으므로, 회귀 기준을 만들어 두면 다음 회차가 이 SQL 을 손볼 때 "오늘 등록하며 과거 교류를 함께 적은 사람이 통째로 사라지는" 주석 속 실패를 사람이 눈으로 확인하지 않아도 잡힌다.

- 수용 기준:
  1) `ORBIT_TEST_DATABASE_URL` 이 없으면 기존처럼 SKIP 하고, 있으면 실제 `store.Open`(마이그레이션 001~005) 위에서 `(&Server{store: st}).orbitAt(ctx, userID, at)` 를 직접 불러 검사하는 테스트가 추가되어 초록이다. 프로덕션 코드(`timetravel.go`·`data.go`)는 **바꾸지 않는다** — 이번 과제는 테스트 공백 보강이다.
  2) 포함 규칙 세 갈래가 각각 따로 증명된다: (a) `created_at` 이 `at` 이후여도 `at` 이전 교류가 있으면 보인다 (b) `created_at`·교류 모두 `at` 이후여도 `first_met` 이 `at` 이전이면 보인다 (c) 셋 다 `at` 이후면 안 보인다 (d) `relationships` 행이 없는 사람은 `JOIN relationships` 때문에 아예 안 나온다 (e) 다른 사용자의 사람은 섞이지 않는다.
  3) 시점 필터가 증명된다: `memory_count` 는 `coalesce(occurred_at,created_at)<=at` 인 `status='approved'` 기억만 센다(`at` 이후 기억 제외, `status='pending'`/`'draft'` 제외, `occurred_at IS NULL` 이면 `created_at` 으로 판정); `last_interaction_at` 은 `at` 이전 교류 중 가장 늦은 것이고 `at` 이후 교류는 무시되며, 1년보다 오래된 교류 하나뿐인 사람도 `nil` 이 아니라 그 시각이 나온다(점수 창과 달리 창이 없다는 성질). 교류가 하나도 없으면 `nodes[i]["last_interaction_at"]` 는 `nil` 이다.
  - 되도록: `closeness`/`momentum` 이 `at` 기준으로 계산됨을 한 경우로 고정 — 같은 데이터에서 `at` 을 교류 직후와 2년 뒤로 두면 closeness 가 줄어든다(`relationshipMetrics` 의 1년 창은 `at.AddDate(-1,0,0)`). 시간이 모자라면 이것만 뺀다.

- 건드릴 파일:
  - `internal/server/timetravel_db_test.go` — 여기 한 파일에만 더한다. 새 `TestOrbitAtHistoricalSnapshot` 과 시드 헬퍼를 추가한다. 기존 `TestOrbitRangeMatchesLegacyJoin`·`openTestStore`·`seedUser`·`seedInteraction`·`sameInstant` 는 그대로 쓴다.
  - 새 헬퍼(확인한 스키마 그대로):
    - `seedRelationship(t, st, userID, personID)` → `INSERT INTO relationships(id,user_id,person_id) VALUES($1,$2,$3)` — NOT NULL 은 이 셋뿐이고 importance/closeness/stable_x/stable_y/categories/relationship_label/anchored 는 DEFAULT 가 있다. `orbitAt` 이 `JOIN relationships` 라 **모든 시험 인물에 이 행이 필요하다**.
    - `seedMemory(t, st, userID, personID, occurredAt *time.Time, status string)` → `INSERT INTO memories(id,user_id,person_id,title,content_cipher,key_version,occurred_at,status) VALUES($1,$2,$3,'시험 기억','',1,$4,$5)` — `content_cipher`·`title`·`key_version` 이 NOT NULL, `occurred_at` 은 nullable(그 경우 `created_at` 으로 판정되는 것을 시험한다), `status` CHECK 는 draft/pending/approved/rejected.
    - `first_met` 은 `people.first_met date` 컬럼이다(001_init.sql:84). 기존 `seedPerson` 시그니처를 바꾸면 `TestOrbitRangeMatchesLegacyJoin` 호출부까지 건드리게 되니, `seedPersonFirstMet(...)` 를 따로 두거나 `seedPerson` 뒤에 `UPDATE people SET first_met=$2 WHERE id=$1` 을 거는 쪽을 택한다.
  - 확인 방법은 `orbitAt` 이 돌려주는 `[]map[string]any` 에서 `node["id"]`·`node["memory_count"]`·`node["last_interaction_at"]`·`node["closeness"]` 를 꺼내 쓴다(키 이름은 timetravel.go:120~130 에 있는 그대로). 두 번째 반환값 `contexts` 는 `r.categories` 기본값이 `'[]'` 라 빈 맵이 정상이다.
  - `orbitAt` 이 읽는 `r.anchored` 는 `003_anchored_relationships.sql` 이 `boolean NOT NULL DEFAULT false` 로 더한 컬럼이라 `seedRelationship` 에서 넣지 않아도 된다(확인함).

- 검증 명령 (이 저장소에서 실제로 도는 것. docker 와 `postgres:16-alpine` 이미지는 이 기계에 이미 있다):
  ```
  docker run -d --rm --name orbit-pg-improve -e POSTGRES_PASSWORD=orbit -e POSTGRES_DB=orbit -p 127.0.0.1:55471:5432 postgres:16-alpine
  # 준비될 때까지 몇 초. 55433/55439/15434 는 다른 세션이 이미 쓰고 있으니 피한다.
  ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55471/orbit?sslmode=disable' \
    go test -race -count=1 -v ./internal/server -run 'TestOrbit'
  go test -race ./...        # DSN 없이도 초록이어야 한다(새 테스트는 SKIP)
  gofmt -l internal/server   # 출력 없음
  go vet ./...
  docker rm -f orbit-pg-improve
  ```
  웹은 이번에 건드리지 않으므로 `npm` 은 돌리지 않아도 된다.

- 위험과 피할 것:
  - **프로덕션 SQL 을 "고치지" 말 것.** 시험을 쓰다 보면 `first_met<=$2::date` 의 시간대 의존(세션 TimeZone 으로 캐스팅)이나 교류 조회에 LIMIT 이 없는 점이 눈에 띌 텐데, 이번 과제는 지금 동작을 있는 그대로 고정하는 것이다. 지금 동작을 기대값으로 적고, 고칠 거리는 ideas.json 으로만 남긴다. 규칙을 고치면서 동시에 시험을 쓰면 "시험이 무엇을 증명했는지" 가 사라진다.
  - `seedUser` 의 `t.Cleanup` 이 사용자 행을 지워 FK cascade 로 정리한다. 새 표(memories/relationships)도 `user_id` FK 가 `ON DELETE CASCADE` 라 따로 지울 필요 없다. 공유/운영 DB 를 DSN 으로 주지 말 것.
  - 시각은 `time.Date(..., time.UTC)` 로 고정해 쓰고, 돌려받은 `last_interaction_at` 은 `time.Time` 이라 `==` 가 아니라 `.Equal()` 로 비교한다(기존 `sameInstant` 와 같은 이유 — pgx 가 돌려주는 위치 정보가 다를 수 있다).
  - 보호 경로(`internal/server/auth.go`·`throttle.go`·`internal/secure`·`internal/store/migrations`·`.github/workflows`)는 건드리지 않는다. 마이그레이션을 새로 만들 일이 없는 과제다.
  - 교훈에서: 시험을 쓴 뒤 **한 번은 일부러 깨 보라** — 예컨대 포함 규칙에서 `EXISTS (... interactions ...)` 절을 뺀 질의로 갈아 보고 해당 하위 시험이 빨개지는지 확인한 뒤 되돌린다(2026-09-20 회차가 같은 방식으로 시험의 유효성을 증명했다). 통과만 확인한 시험은 아무것도 증명하지 못한다.

- 차선 후보: **Time Travel API(`GET /orbit?at=`, `earliest_at`, `historical`)를 `internal/server/openapi.go` 와 `docs/API.md` 에 적기** — `openapi.go:13` 의 `/orbit` 은 `operation("Orbit 그래프 조회","orbit:read")` 한 줄뿐이고 `at` 매개변수도, 응답의 `earliest_at`/`historical`/`at` 필드도 어느 문서에도 없다. API 키로 호출하는 쪽이 이 기능을 알 방법이 없다. 순수 문서·스키마 작업이라 위험 1/작업량 S. 다만 `docs/cru-manual.md` 의 API 설명은 지금 코드와 어긋난 곳이 있으니 **실제 핸들러(`data.go:getOrbit`, `timetravel.go:parseOrbitAt`·`writeOrbitAt`)를 정본으로 삼아** 쓸 것.

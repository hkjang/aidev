- 과제: 과거 Orbit(`GET /orbit?at=`) 응답의 `links` 가 그날 없던 사람을 가리키는 끊긴 참조를 담는 것을 고치기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/server/timetravel.go:writeOrbitAt`(147~184줄)은 `nodes` 를 `s.orbitAt` 의 포함 규칙(그 시점에 이미 곁에 있던 사람)으로 거르면서, `links` 는 `internal/server/data.go:orbitLinks`(697줄)가 준 **오늘의 전체 목록**을 그대로 담는다. 그래서 같은 응답 안에서 `links[*].a`/`links[*].b` 가 `nodes` 에 없는 id 를 가리킨다 — 2026-09-26 회차가 맞춘 "현재/과거 같은 자원, 같은 모양" 계약이 링크 쪽에는 아직 안 걸려 있다. 웹 캔버스는 우연히 견디지만(`web/src/components/OrbitCanvas.tsx:421~425` 가 `byId` 맵에 없는 끝점을 `continue` 로 건너뜀) API 키로 `?at=` 만 부르는 호출자는 끊긴 참조를 받아 그래프를 그릴 수 없다. 고치면 과거 응답 자체로 닫힌 그래프가 된다.
- 수용 기준:
  1) `?at=` 응답의 모든 `links[*].a`·`links[*].b` 가 같은 응답의 `nodes[*].id` 집합 안에 있다.
  2) 현재 시점(`?at=` 없음) 응답은 한 글자도 바뀌지 않는다 — `getOrbit` 의 `links` 는 지금처럼 `orbitLinks` 전부를 담는다. `links` 는 계속 `[]`(배열)로 나가고 `null` 이 되지 않으며, 순서(`LIMIT 5000` 질의 순서)와 각 항목의 키 `a`/`b`/`kind`/`strength` 도 그대로다.
  3) 실제 postgres 테스트가, **그 시점 뒤에 생긴 사람과 이어진 연결은 과거 응답에서 빠지고, 둘 다 그때 있던 사람끼리의 연결은 남는다**는 것을 실제 핸들러 `getOrbit` 을 `?at=` 로 불러 증명한다. 고치기 전에 먼저 돌려 빨개지는 것을 확인하고 나서 고칠 것(이 저장소의 관례다).
- 건드릴 파일 (프로덕션 1개):
  - `internal/server/timetravel.go:writeOrbitAt` — `s.orbitLinks` 결과를 받은 뒤, `nodes` 의 `"id"` 값으로 `map[string]bool` 집합을 만들어 두 끝점이 모두 그 안에 있는 링크만 남긴다. 결과는 반드시 `make([]map[string]any, 0)` 으로 시작해 빈 경우에도 JSON `[]` 로 나가게 할 것(지금 `orbitLinks` 도 그렇게 한다). 왜 그렇게 하는지 한국어 주석으로 두세 줄 — 이 파일은 "무엇을" 이 아니라 "왜" 를 적는 관례다(11~19, 57~62, 164~166줄 참고). `orbitLinks` 의 시그니처·질의는 건드리지 말 것(현재 경로와 공유한다).
  - `internal/server/timetravel_db_test.go` — 새 헬퍼 2개 + 새 테스트 1개.
- 테스트에 필요한 사실 (실제로 마이그레이션과 기존 테스트를 열어 확인함):
  - 기존 헬퍼가 그대로 쓸 수 있다: `openTestStore`(23줄, DSN 없으면 `t.Skip`), `seedUser`(41줄, `t.Cleanup` 이 users 행을 지워 FK cascade 로 정리), `seedPerson(t, st, userID, createdAt) string`(53줄), `seedInteraction`(62줄), `callOrbit(t, s, userID, at) map[string]json.RawMessage`(146줄 — `s.getOrbit` 을 httptest 로 직접 부르고 200 이 아니면 실패), `decodeTime`(170줄).
  - **`orbitAt` 은 `JOIN relationships` 라 relationships 행이 없는 사람은 `nodes` 에 아예 안 나온다.** main 에 `seedRelationship` 헬퍼는 **없다** — 새로 만들어야 한다. `relationships` 의 NOT NULL 중 기본값 없는 것은 `id,user_id,person_id` 뿐이다(`internal/store/migrations/001_init.sql:90~106`). 즉 `INSERT INTO relationships (id,user_id,person_id) VALUES ($1,$2,$3)` 로 충분하다.
  - `person_links` 헬퍼도 없다. 테이블에 `CHECK (person_a <> person_b)` 와 **`CHECK (person_a < person_b)`** 가 걸려 있으므로(`internal/store/migrations/004_person_links.sql:8~21`) 짝을 그냥 넣으면 실패한다 — 프로덕션의 `normalizeLink`(같은 패키지, `data_test.go`/`server_test.go` 의 `TestNormalizeLinkIsOrderIndependent` 가 덮는다)를 그대로 불러 정규화할 것. `kind` 기본값 `'knows'`, `strength` 기본값 `.5` 라 `INSERT INTO person_links (id,user_id,person_a,person_b) VALUES ($1,$2,$3,$4)` 로 충분하다. id 는 `id.New()`(uuid).
  - `people.created_at` 은 `timestamptz`, `first_met` 만 `date`. `orbitAt` 의 포함 규칙은 `p.created_at<=at OR (first_met IS NOT NULL AND first_met<=at::date) OR EXISTS(그 시점 이전 교류)` 세 갈래다(`timetravel.go:63`). "그 시점 뒤에 생긴 사람" 을 만들려면 `seedPerson(..., at.AddDate(0,1,0))` 처럼 created_at 을 뒤로 두고 교류도 넣지 말 것 — 셋 중 하나만 걸려도 포함된다.
  - 시각 비교는 `==` 가 아니라 `.Equal()`. `nodes`/`links` 는 `json.RawMessage` 로 오므로 `[]map[string]any` 나 전용 struct 로 한 번 더 언마샬해야 한다.
  - 제안하는 하위 시험 2개: (a) 과거 시점 — 사람 A·B 는 그 전부터 있고 C 는 그 뒤에 생김, 링크 A-B 와 B-C 를 넣고 `?at=` 응답에 A-B 만 남는지 + 남은 링크의 끝점이 모두 nodes 안에 있는지; (b) 같은 데이터로 `?at=` 없이 부르면 A-B·B-C 두 개가 다 있는지(현재 경로 무변경 회귀).
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l .` (출력 없어야 함), `go vet ./...`, `go test -race ./...` (DSN 없으면 새 시험은 SKIP — 초록이어야 함)
  - DB: 빈 포트로 `docker run -d --name orbit-pg-links -e POSTGRES_PASSWORD=orbit -e POSTGRES_DB=orbit -p 127.0.0.1:55491:5432 postgres:16-alpine` 뒤
    `ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55491/orbit?sslmode=disable' go test -race -count=1 -v ./internal/server -run TestOrbit`
    → 기존 `TestOrbitRangeMatchesLegacyJoin`·`TestOrbitAtResponseCarriesEarliestAt` 과 함께 전부 PASS 여야 한다. **포트 55433·55439·15434·55470·55471·55481 은 다른 세션이 쓸 수 있으니 피할 것.** 끝나면 컨테이너를 지울 것. 공유·운영 DB 금지.
  - 웹은 건드리지 않으므로 `npm` 은 돌릴 필요 없다(원하면 `cd web && npm run test -- --run` 은 그대로 통과해야 한다). CI 는 postgres 없이 돌아 새 DB 시험을 SKIP 한다 — 로컬에서 DSN 을 주고 직접 돌린 결과를 보고할 것.
- 위험과 피할 것:
  - `internal/server/data.go:getOrbit`(715줄)과 `orbitLinks`(697줄)는 **손대지 말 것**. 현재 경로가 같은 함수를 쓰므로 `orbitLinks` 안에서 거르면 현재 응답까지 조용히 바뀐다(수용 기준 2 위반). 거르는 자리는 `writeOrbitAt` 안이다.
  - `orbitAt` 의 SQL·포함 규칙·`contexts` 집계는 이번 과제가 아니다. 성능(LIMIT, GROUP BY) 개선도 섞지 말 것 — 별 항목으로 보류 중이다.
  - `person_links` 에 시간 필터(created_at<=at)를 거는 쪽으로 넓히지 말 것. 이 모듈의 선언(`timetravel.go:17~19`)은 "사용자가 직접 정하는 값은 오늘의 값을 쓴다, 되살아나는 것은 교류가 만든 거리와 흐름" 이다. 링크는 사용자가 정하는 값이므로 오늘의 값을 쓰되 **응답 안에서 닫혀 있게만** 만드는 것이 이번 범위다. 판단 근거를 주석에 남길 것.
  - 보호 경로 안 건드림: `auth.go`/`throttle.go`/`internal/secure`/`internal/store/migrations`/`.github/workflows` — 이번 과제는 전부 무관하다. 마이그레이션 새 파일도 필요 없다.
  - 과거 교훈: 2026-09-24 회차는 DB 테스트를 잘 짜고도 `npm ci` 네트워크로 verify 가 죽었다. 이번엔 웹을 안 건드리니 그 길은 피하지만, verify 가 `npm ci` 를 돌린다는 점은 기억할 것.
  - 커밋은 한국어 한 줄: `fix(orbit): …` 형식. 커밋 1개.
- 차선 후보: `internal/server/openapi.go:openAPI` 에 핸들러 렌더 테스트를 붙이기 — 이 핸들러를 부르는 테스트가 **0개**다(`grep -h '^func Test' internal/server/*_test.go` 로 확인). 지난 회차가 `/orbit` 항목을 리터럴로 풀어 쓰면서(13~27줄) 나머지 아홉 경로와 모양이 갈라졌는데 아무것도 고정하지 않는다. `httptest` 로 불러 200·유효 JSON·`paths` 키 10개 목록·각 operation 이 `summary`/`description`/`responses` 를 갖는지만 고정하면 다음에 항목을 리터럴로 풀 때 오타가 잡힌다. DB 불필요, 프로덕션 코드 무변경 (가치 2 / 위험 1 / 작업량 S).

---
### 추정 근거 (basis of estimate)
- 분해: (1) `writeOrbitAt` 링크 필터 ~10줄 (2) `seedRelationship`·`seedPersonLink` 헬퍼 ~16줄 (3) 하위 시험 2개 테스트 함수 ~60줄 (4) docker postgres 띄우고 DB 시험 실행·정리.
- 방법: bottom-up. 유사 비교(analogous)로도 확인 — 2026-09-26 회차의 `earliest_at` 과제가 거의 같은 모양(프로덕션 1파일 소폭 + 같은 테스트 파일에 핸들러 수준 DB 시험)이었고 한 세션에 끝났다.
- 범위: 30~60분(10회 중 8회). 포함: 위 네 조각과 gofmt/vet/go test. 제외: 웹 빌드·vitest(이번 변경과 무관), openapi/문서 갱신(응답 키가 늘지 않으므로 `docs/API.md` 수정 불필요 — 다만 구현자가 Time Travel 절에 한 문장 더하고 싶으면 그건 여유분 안이다).
- 가정(틀리면 계획이 깨지는 곳): docker 로 `postgres:16-alpine` 을 이 기계에서 새 포트에 띄울 수 있다. 못 띄우면 DB 시험은 SKIP 된 채로만 남고 수용 기준 3 을 증명할 수 없다 — 그때는 차선 후보(openapi 렌더 테스트, DB 불필요)로 갈아탈 것.
- 여유분(contingency): 포트 충돌·마이그레이션 첫 적용 대기로 10분. 관리 예비(management reserve)는 이 과제에 두지 않는다.

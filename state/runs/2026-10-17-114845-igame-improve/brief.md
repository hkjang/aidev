- 과제: 공개 카탈로그 `q` 검색의 리터럴 특수문자(`% _ \`)와 태그 정확일치 계약을 실제 PG/Router 회귀로 고정 (가치 3 / 위험 1 / 작업량 M)

- 왜: `searchPattern`(internal/api/api.go:853)이 `% _ \` 를 escape 하므로 `q=%` 는 "와일드카드로 전부 일치" 가 아니라 "이름/설명에 `%` 를 담은 행만" 을 뜻해야 하는데, 현재 이 계약을 보는 테스트는 `api_test.go:125-138` 의 **헬퍼 출력 비교 한 곳뿐**이다 — 엔드포인트가 그 헬퍼를 정말 거치는지, 그리고 같은 핸들러의 태그 arm(`$2=ANY(g.tags)`, 원문 그대로)이 escape 된 문자열을 받지 않는지는 증명되어 있지 않다. 운영자 지침("프로덕션 배선을 통과하는 실행 테스트로 증명할 것 — 소스 문자열 검사나 헬퍼 출력을 근거로 삼지 말 것")에 정확히 걸리는 공백이고, 실수로 `$3`→`$2` 로 바뀌거나 `searchPattern` 호출이 빠지면 `q=%` 가 전체 카탈로그를 돌려주는데(필터가 적용된 척하며 아무것도 안 하는 상태) 지금은 아무 테스트도 붉어지지 않는다.

- 수용 기준:
  1) `q=%` 가 이름/설명에 리터럴 `%` 를 가진 행만 돌려주고, 같은 픽스처의 다른 active 행은 하나도 섞이지 않는다(와일드카드 해석 아님을 `items` 의 id 집합으로 단정).
  2) `q=_`, `q=user_id` 형태의 `_` 1글자 와일드카드도 리터럴로 취급된다(`_` 를 담지 않은 행 제외).
  3) `q=C:\Users` 같은 역슬래시 포함 항은 escape 이중화(`\\`)를 거쳐 리터럴로 일치하고, PG 에러(invalid escape)나 빈 결과가 아니다.
  4) 태그 arm 회귀: `q=<태그 원문>` 은 그 태그를 가진 행을 돌려주고, `%` 를 담은 태그는 `q=%` 로 일치하지 **않는다**(태그는 정확일치이므로) — 즉 escape 된 `$3` 이 태그 비교에 흘러들지 않음을 단정.
  5) 대조군: 특수문자 없는 평범한 부분일치 `q=<prefix>` 가 픽스처 전체를 돌려준다(필터 자체가 죽지 않았음).
  6) 인과 증명: `catalog.go:130` 의 `searchPattern(q)` 를 `q` 로 바꾸는 변이 1개를 넣으면 1)~3) 중 최소 하나가 FAIL 하고, 복원하면 PASS 한다(변이는 반드시 되돌릴 것 — 커밋에 제품 변경 없음).

- 건드릴 파일 (프로덕션 0개 · 신규 테스트 1개):
  - `internal/api/catalog_search_pg_test.go` (신규) — `TestCatalogSearchLiteralSpecials` 와 `TestCatalogSearchTagExactMatch`.
    - 픽스처는 `newAdminListFixture`(admin_pg_test.go) 를 그대로 재사용한다 — `f.pool` / `f.server`(실제 `New(pool,nil,logger).Router()` 의 httptest) / `f.cookie`(실제 DB 세션 쿠키) / `f.admin` 를 준다. **`newRankingFixture`(rankings_pg_test.go)는 3000명+ANALYZE 로 무겁다 — 쓰지 말 것.**
    - 행 삽입은 `catalog_order_pg_test.go:36-57` 의 방식을 따라 `INSERT INTO games(id,slug,name,description,tags,game_url,status)` 로 **명시적 UUID 와 고유 prefix 슬러그**를 쓰고, `t.Cleanup` 에서 `DELETE FROM games WHERE slug LIKE $1||'-%'` 로 정리한다(`insertTestGame`(catalog_pg_test.go:27)은 이름이 고정 'Play limit test' 이고 description/tags 를 못 넣어 이 과제에는 맞지 않는다 — 수정하지 말고 쓰지 말 것).
    - 응답 읽기는 `catalog_order_pg_test.go:64-80` 의 `page` 헬퍼와 같은 모양으로 신규 파일 안에 자체 헬퍼를 둔다(기존 헬퍼는 `limit=2` 고정이므로 수정하지 말고 복사하되 `limit=200` 로). 응답은 `{"items":[],"limit":n,"offset":n}` 뿐이고 `total` 은 없다.
    - 쿼리 문자열은 반드시 `url.Values{}.Encode()` 로 만든다 — `%`/`\` 를 날것으로 붙이면 URL 파싱에서 깨져 테스트가 `q` 를 전달하지 못하고 거짓 PASS 한다.
    - 픽스처 행 설계 예: `<p>-plain` (name `<p> plain`), `<p>-pct` (name `<p> 50% off`), `<p>-und` (name `<p> user_id`), `<p>-bs` (description `C:\Users\<p>`), `<p>-tag` (tags `{<p>-tagonly}` 이고 이름/설명에는 그 문자열이 없음), `<p>-pcttag` (tags `{%}`). 모두 `status='active'`, `game_url='/games/search/'`.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - PG 준비: `docker run -d --name igame-pg -e POSTGRES_PASSWORD=igame -p 15432:5432 postgres:17-alpine` 후
    `CREATE SCHEMA igame_test_extensions; CREATE EXTENSION pgcrypto WITH SCHEMA igame_test_extensions;`
    (public 에 확장을 만들면 `migratedPool` 이 거부한다). DSN 예:
    `postgres://postgres:igame@localhost:15432/postgres?sslmode=disable&search_path=public,igame_test_extensions`
  - `IGAME_TEST_DSN=… go test ./internal/api -run '^TestCatalogSearch' -count=1 -v` → **skip 0개**를 눈으로 확인(DSN 없으면 전부 skip 되고 그 PASS 는 증거가 아니다).
  - 변이(`searchPattern(q)`→`q`) 적용 후 같은 명령으로 Red 확인 → 되돌림 → 다시 Green.
  - `IGAME_TEST_DSN=… go test ./internal/api -run '^TestCatalogSearch' -race -count=5`
  - `make test-db DSN=…` (api 약 31–37s / database 약 2s)
  - `go test ./cmd/... ./internal/... ./migrations/... -count=1` · `go vet ./cmd/... ./internal/... ./migrations/...` · `gofmt -l internal/api/catalog_search_pg_test.go` · `git diff --check`
  - `bash scripts/check-release-contract.sh` (v0.7.34 기준 통과해야 함)
  - 실행 후 잔여 `api_test_`/`migrate_` 스키마 0개와 `igame_test_extensions` 보존 확인.

- 작업분해·추정 (bottom-up, 범위는 80% 신뢰):
  1) 픽스처 6행 삽입 + cleanup — 15–25분. 2) page 헬퍼 + 특수문자 3케이스 — 15–25분. 3) 태그 정확일치 대조군 — 10–15분.
  4) 변이 Red/Green + race + make test-db — 15–25분. 합계 **55–90분**(기대 70분).
  - contingency(알려진 변동: PG 컨테이너 준비 재시도, URL 인코딩 실수 1회) +15분 — 이 과제 안에서 쓴다.
  - management reserve(미식별 범위) +15분 — 스키마는 확인했다: `migrations/001_initial.sql:80-105` 의 games 는 `description text NOT NULL DEFAULT ''`, `tags text[] NOT NULL DEFAULT ARRAY[]::text[]` 이므로 위 INSERT 는 그대로 통과한다 — 넘기면 범위를 줄이고(기준 1·2·5만) 남긴 것을 노트에 적을 것.
  - 기준 중 하나가 "지금 동작과 다르다"로 밝혀지면 **제품 코드를 고치지 말고** 그 사실을 회차 노트에 적고 테스트를 현재 동작에 맞춘 뒤 비평 단계로 올려라(이번 과제는 테스트 공백 보강이며 동작 변경 권한이 없다).

- 위험과 피할 것:
  - **제품 코드는 수정하지 않는다.** `searchPattern`/`likeEscaper`(api.go:839-857)·`catalog.go:130` 의 SQL·`gameSelect` 를 바꾸지 말 것. 운영자 지침: 같은 종류의 파서가 여러 경로(admin users/audit/auditexport/catalog)에 있으므로 **통합하거나 태그 arm 을 escape 쪽으로 넓히려 하지 말 것** — 현재 동작을 고정하는 데까지만.
  - 기존 테스트 파일(`api_test.go`, `catalog_order_pg_test.go`, `catalog_pg_test.go`, `admin_pg_test.go`)과 공용 헬퍼를 수정하지 말 것 — 다른 PG 테스트가 같은 픽스처를 쓴다.
  - 보호 경로 금지: `.github/workflows`, `migrations/`(체크섬), `auth.go`/세션 발급, `secretbox`, Go 핀 8곳. 새 인덱스·새 환경변수·docs 외 문서 변경 없음.
  - `docs/api.md` 는 이번에 바꾸지 않아도 된다(검색 escape 계약이 문서에 적혀 있지 않고, 적으면 "알 수 없는 필드" 절처럼 코드와 어긋날 위험이 생긴다). 꼭 적고 싶으면 한 문장만, `## 공통 규칙` 끝에.
  - 과거 교훈: 정렬/목록 PG 테스트는 `limit=1` 단독 순회에서 플래키했다 — 여기서는 `limit=200` 한 번 조회 + id 집합 비교로 끝낼 것. 같은 스키마에 다른 테스트의 games 행이 있으니 **단정은 반드시 `<prefix>` 로 좁힌 집합 비교**로 할 것(전체 개수 단정 금지).

- 차선 후보: 「키 외 필수 환경변수(DSN/BOOTSTRAP_ADMIN) 누락 집계·비밀 비노출을 `config.Load` 에서 회귀로 고정 (가치 2 / 위험 1 / 작업량 S)」 — PG 가 준비되지 않아 위 과제의 skip 0 개를 보장할 수 없을 때 선택한다. `internal/config/config_test.go` 한 파일만 건드리고 `t.Setenv` 로 네 필수 변수를 통제하며, 동시 누락 시 오류 메시지에 모든 누락 키가 모이는지와 오류에 비밀 값이 실리지 않는지를 단정한다. 검증: `go test ./internal/config -count=1 -v` · `-race -count=3`.

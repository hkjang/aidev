# 과제서 (2026-10-04, base main@2692aae)

- 과제: OpenAPI 가 공표하는 **enum 질의 파라미터**를 `migrations/` 의 `CHECK (col IN (...))` 와 대조하는 불변식 테스트 (가치 3 / 위험 1 / 작업량 S)

- 왜: 필터 값 목록이 두 곳에 손으로 적혀 있다 — 문서를 만드는 `internal/api/parameters.go` 의 `choice(...)` 호출과, 그 값을 실제로 받아 주는 `migrations/*.sql` 의 `CHECK (col IN (...))` 제약 — 그런데 둘을 묶는 것이 아무것도 없다. 기존 `TestDocumentedQueryParametersMatchTheHandler` 는 질의 키의 **이름**만 보고, `TestDocumentedSortValuesMatchTheQueryBuilder` 는 `sort` 의 enum 만 본다. 한쪽만 고치면 `go build`·`go vet`·기존 5개 계약 테스트가 전부 green 인 채로 문서가 거짓말을 한다: 공표한 값이 DB CHECK 에 없으면 그 필터는 0건을 돌려주거나 쓰기에서 23514 로 터지고, DB 가 받는 값이 문서에 빠지면 생성 클라이언트가 유효한 필터를 쓸 수 없다. **오늘 드리프트는 없다 — 이것은 그물이지 버그 수정이 아니다.**

- 수용 기준:
  1) 새 테스트가 `api.OpenAPI()` 가 **실제로 내보내는 문서**를 걸어 `schema["enum"]` 이 있는 질의 파라미터를 전부 모으고, 파라미터 이름을 camelCase→snake_case 로 바꾼 컬럼명으로 `migrations.Files`(embed FS) 에서 뽑은 **유효** CHECK 집합과 **집합으로** 대조한다(순서 무관). 어긋나면 `operation: 파라미터 — document {A,B}, migrations {A,C} (NNN_x.sql:line)` 처럼 어느 쪽이 무엇이고 제약이 어디 있는지 찍는다.
  2) 모호·비-DB 키는 **명시 제외 표**에 이유와 함께 적고, 제외 표에도 없고 CHECK 에도 없는 enum 파라미터가 새로 생기면 테스트가 **실패**한다(조용히 빠져나가지 못한다). 오늘의 제외 대상과 이유(실측):
     - `sort` — 정렬 키이지 열 값이 아니다. 기존 `TestDocumentedSortValuesMatchTheQueryBuilder` 가 본다.
     - `prompt` (`parameters.go:137`, 값 `none`) — OIDC 요청 파라미터, DB 열이 아니다.
     - `status` — `migrations/` 에 `status IN (...)` 가 **11개**, 서로 다른 집합(`ACTIVE/RESOLVED/IGNORED`, `OPEN/RESOLVED/ACCEPTED`, `ACTIVE/EXPIRED`, `OPEN/ACCEPTED/DISMISSED/COMPLETED`, `queued/sent/failed`, `PLANNED/RECOGNIZED/CANCELLED` …)이라 컬럼명만으로는 어느 표인지 정할 수 없다.
     - `severity` — 3개 중 **2개 집합**: VOC 는 `LOW/NORMAL/HIGH/CRITICAL`(`008_customer_voice.sql:34`), signals·risks 는 `LOW/MEDIUM/HIGH/CRITICAL`(`011_crm_intelligence.sql:20,51`). 둘 다 문서에 그대로 있어 **오늘은 맞지만**(`parameters.go:114` vs `:208`,`:214`) 컬럼명만으로는 구분 불가.
     - `forecastCategory` — `migrations/` 에 `forecast_category` CHECK 가 **없다**(실측). DB 가 제약하지 않는다.
  3) 테스트가 실제로 표에 닿고 있음을 **개수 가드 2개**로 증명한다(저장소 관용 — `internal/server/openapi_contract_test.go` 의 `knownIntQueries`, `internal/crm/search_test.go` 의 `knownComparisons`): 읽어 낸 enum 파라미터 수와 파싱한 CHECK 제약 수가 파일 수준 상수 미만이면 `t.Fatalf`. 상수를 1 올려 실제로 red 가 나는 것을 확인할 것.
  4) 섭동으로 red 를 눈으로 볼 것(그리고 전부 `git checkout --` 로 되돌릴 것):
     - `parameters.go:104` 의 `entityType` 에서 `"CONTRACT"` 하나 삭제 → 그 파라미터를 쓰는 operation 만 red.
     - `migrations/016_voice_workspaces.sql:47` 의 `knowledge_status` 집합에 값 하나 추가 → `knowledgeStatus` 를 쓰는 operation 만 red.
     - 아래 "최신 승자" 규칙 검증: `migrations/014_momento_provider.sql:9` 의 `provider` CHECK(=`010` 의 것을 ALTER 로 교체)이 `010_visitor_analytics.sql:11` 을 덮는지 — 테스트 안에서 유효 집합을 출력하거나 임시 assert 로 확인.
  5) **프로덕션 코드 0줄 변경.** 기존 테스트 이름·실패 문구는 한 글자도 바꾸지 않는다.

- 건드릴 파일 (테스트 1개, 신규):
  - `internal/api/enum_contract_test.go` (신규, `package api`) — 지금 `internal/api/` 에는 테스트 파일이 **하나도 없다**(`openapi.go`, `parameters.go` 뿐). 새 테스트 `TestDocumentedEnumValuesMatchTheMigrations` 를 여기 둔다. 라우터가 필요 없고 `api.OpenAPI()` 와 `migrations.Files` 만 쓰므로 `internal/server` 로 갈 이유가 없다.
    - 문서 걷기: `api.OpenAPI()` → `["paths"]` → path → method → `["parameters"]` → `["in"] == "query"` 인 것만 보고 `["schema"]["enum"]` 을 꺼낸다. **`$ref` 항목은 `map[string]any` 다**(`parameters.go:307` 이 `map[string]any{"$ref": "#/components/parameters/fields"}` 를 넣는다 — 이번 회차 실측, 저장소에 `$ref` 는 이 한 자리뿐). 따라서 "map 이 아니면 건너뛴다" 로는 안 걸러진다 — `"$ref"` 키가 있거나 `"schema"` 가 없는 항목을 건너뛸 것. **`enum` 은 `[]string` 이다** (`parameters.go:46` 이 `q.enum`(`[]string`)을 그대로 넣는다) — `[]any` 로 단정하면 패닉 대신 조용히 0건이 되니 `[]string` 으로 받고, 아니면 `t.Fatalf`.
    - 마이그레이션 걷기: `migrations.Files` 를 `fs.ReadDir` 로 읽고 **파일명 사전순**(= `001`…`017` 번호순, 실행 순서와 같다)으로 돌면서 정규식으로 `CHECK ( <col> IN ( '<v>', … ) )` 를 뽑는다. 같은 컬럼명이 다시 나오면 **뒤에 나온 것이 이긴다**(`ALTER TABLE … ADD CONSTRAINT`/`ADD COLUMN` 이 앞의 제약을 교체한다 — `014_momento_provider.sql:9` 이 `provider` 를, `016_voice_workspaces.sql:56` 이 `event_type` 을 실제로 그렇게 덮는다). 같은 컬럼명이 **서로 다른 집합**으로 여러 번 나오면 그 컬럼은 "모호" 로 표시해 제외 표와 맞춘다.
    - 대조 대상(실측으로 오늘 전부 일치, 컬럼명 하나당 집합이 하나뿐인 6개):
      `entityType`→`entity_type` (`011:21`,`011:47` 동일) · `voiceType`→`voice_type` (`008:11`,`008:30` 동일) · `sentiment`→`sentiment` (`011:19`) · `priority`→`priority` (`011:94`) · `causeEvidence`→`cause_evidence` (`016:43`) · `knowledgeStatus`→`knowledge_status` (`016:47`).
    - camel→snake 변환은 테스트 안의 작은 헬퍼로 충분하다(위 6개는 전부 기계적 변환으로 맞는다 — 실측).
  - 그 밖의 파일은 건드리지 않는다. 프로덕션 파일 **0개**.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/api/ -run TestDocumentedEnumValuesMatchTheMigrations -v`
  - `go test ./internal/server/ -run 'TestDocumented|TestEveryRouted|TestEveryDocumented|TestProjectionParameter|TestEveryPathTemplate' -v` (기존 7건이 그대로 PASS 해야 한다)
  - `go test ./...` (이전 회차 기준 22 패키지 ok / FAIL 0) · `go vet ./...` · `go build ./...`
  - `gofmt -l internal/api/enum_contract_test.go` 무출력 · `git diff --check` 무출력
  - `./scripts/check-env-contract.sh` · `./scripts/check-static-assets.sh`
  - `make test` 는 **돌리지 않아도 된다** — 이번 변경은 Go 테스트 1파일이고 `make test` 의 `npm ci` 가 네트워크를 탄다(ETIMEDOUT 이력).

- 위험과 피할 것:
  - **`migrations/` 의 SQL 을 수정하지 말 것.** 보호 경로다(새 스키마는 새 번호 파일이고, 이번 과제는 새 번호를 만들지 않는다). 섭동으로 넣은 변경은 반드시 `git checkout -- migrations/` 로 되돌리고 최종 `git status` 가 테스트 1파일뿐임을 확인할 것.
  - `internal/api/parameters.go`·`openapi.go` 를 "맞추려고" 고치지 말 것 — **오늘 6개 전부 일치한다**. 불일치를 찾았다고 생각되면 먼저 실제 CHECK 문장을 열어 확인하고, 그래도 불일치면 과제서가 틀린 것이니 노트에 적고 문서를 고치지 말고 그 자리를 제외 표에 넣지도 말 것(구현자 판단으로 보고).
  - `status`·`severity` 를 컬럼명만으로 대조하려 하지 말 것 — 집합이 여러 개다(기준 2 의 실측). operation→표 손 매핑을 추가하면 파일 수와 위험이 함께 커진다. 이번 회차 범위 밖이고 다음 회차 후보로 남긴다.
  - `admin_operations.go` 의 ILIKE·`AdminPages.tsx`·`mcp/server.go` 는 이번 과제와 무관하다. 재포매팅 금지(한 줄이 매우 길다).
  - 과거 교훈: **grep 결과를 증거로 내지 말 것** — 테스트가 실제로 `api.OpenAPI()` 출력과 `migrations.Files` 를 읽고, 섭동으로 red 가 나는 것을 보여야 한다. 개수 가드 없이 통과하는 스캐너는 영원히 green 인 빈 테스트다(2026-10-02 회차에서 기존 ESCAPE 검사가 정확히 그랬다).
  - `.github/workflows/` 는 **0줄** 건드릴 것(2026-09-06 릴리즈 롤백·자율화 강등 이력).

- 차선 후보: **`release.yml` 의 `Test source` 단계에 프런트 테스트 추가** — `ci.yml` 과 `Makefile:test` 는 프런트 27건을 돌리지만 `release.yml` 은 안 돌린다(2회차 전 실측, 이번 회차 **미재확인**). 보호 경로이고 2026-09-06 이 경로에서 롤백·강등이 났으므로, 고를 경우 변경 전에 `release.yml` 의 해당 단계를 실제로 열어 확인하고 로컬에서 같은 명령(`npm --prefix web test`)이 exit 0 인 것을 재현한 뒤에만 한 줄 추가할 것. 3순위는 `web/package.json` 에 `esbuild` 를 devDependency 로 선언(`login.test.ts` 가 vite 전이 의존성에 얹혀 있다 — `package-lock.json` 재생성이 네트워크를 탄다).

## 실측 (이번 회차에 확인한 수)
- `migrations/` 의 `CHECK ( <col> IN (` 제약은 **42개, 12파일**(멀티라인 정규식으로 센 수: `011` 12 · `008` 6 · `016` 4 · `001` 4 · `002`·`003`·`004` 각 3 · `006`·`009` 각 2 · `010`·`014`·`017` 각 1). 컬럼별 분포는 `status` 11 · `severity` 3 · `voice_type`·`resource`·`provider`·`event_type`·`entity_type` 각 2 · 나머지 18개 컬럼 각 1 — 합 42 로 일치한다. 개수 가드 상수는 **42** 로 시작하고, 43 으로 올려 `t.Fatalf` 가 나는 것을 확인할 것.
- 멀티라인 주의: 제약의 **시작**(`CHECK (event_type IN (`)은 전부 한 줄 안에 있지만 **값 목록**은 줄을 넘는 자리가 있다(`016_voice_workspaces.sql:56` 의 `event_type`). 값을 줄 단위로만 긁으면 그 제약의 값이 잘린다 — 파일 전체를 읽어 `IN (` 뒤 닫는 괄호까지를 한 덩어리로 받을 것.

## 미확인 (구현자가 착수 전에 확인할 것)
- `migrations.Files`(`github.com/hkjang/relio/migrations`) 를 `internal/api` 테스트에서 import 해도 순환이 없다는 것은 소스로 확인했지만(`migrations/embed.go` 는 `embed` 만 import), 실제로 컴파일되는지는 **이번 회차에 돌려 보지 않았다**.
- `001_initial.sql` 의 4개 제약이 어느 컬럼인지는 열어 보지 않았다. enum 질의 파라미터로 공표되는 6개 키와는 무관하지만, 개수 가드와 "모호" 판정에는 포함된다.
- `migrations.Files` 를 `internal/api` 테스트에서 import 할 때 순환은 없다(`migrations/embed.go` 는 `embed` 만 import). 다만 실제로 컴파일되는지는 미확인.

# 과제서 — 2026-10-05-202733-relio-improve

- 과제: `status`·`severity` enum 을 operation→(표, 컬럼) 손 매핑으로 대조해 `enumParametersNotBackedByACheck` 의 두 제외 항목을 닫기 (가치 3 / 위험 1 / 작업량 M)

- 왜: 2026-10-04 에 머지된 `TestDocumentedEnumValuesMatchTheMigrations` 는 enum 질의 파라미터 26개 중 `status`·`severity` 를 **컬럼명만으로는 어느 표인지 알 수 없다**는 이유로 명시 제외해 두었다(`internal/api/enum_contract_test.go:43-50`). 그런데 이 둘이 `migrations/` 의 CHECK 42개 중 **14개**(status 11표 + severity 3표)를 차지하고, 공표 쪽에서도 13개 (operation, 파라미터) 쌍을 차지하므로 가장 큰 구멍이다. operation 별로 어느 표를 필터하는지는 한 줄씩 손으로 적을 수 있고(아래 표), 그러면 `GET /opportunities?status=` 가 `opportunities.status` CHECK 와, `GET /voices?severity=` 가 `customer_voices.severity` CHECK 와 묶여서 한쪽만 고쳐도 `go build`·`go vet`·기존 계약 테스트가 전부 green 인 채로 공표 스키마가 거짓말을 하는 일이 막힌다.

- 수용 기준:
  1) `internal/api/enum_contract_test.go` 에 `(operation, 파라미터) → 표` 손 매핑이 생기고, 매핑이 있는 쌍은 `effectiveCheckValues` 의 컬럼명 기준 접기를 **건너뛰고** `migrationCheckConstraints` 가 돌려주는 `map[column]checkConstraint` 를 `column{table: <매핑한 표>, name: snakeCase(파라미터)}` 로 **직접** 찾아 집합 비교한다. `status`·`severity` 는 `enumParametersNotBackedByACheck` 에서 제거된다.
  2) 13개 쌍 전부 오늘 통과한다(아래 "오늘의 실측" 표가 정본). 실패 메시지는 어느 쪽이 무엇인지와 CHECK 자리를 찍는다 — 기존 형식 `%s: %s — document %s, migrations %s (%s:%d)` 를 그대로 재사용할 것.
  3) 손 매핑이 **세 번째 손 목록**이 되므로 양방향 완전성 가드를 둔다:
     - (a) 매핑에 있는 쌍이 문서에 더는 공표되지 않으면 실패(= 낡은 매핑 항목). 메시지에 "operation no longer publishes this enum parameter; drop the mapping entry" 류를 넣을 것.
     - (b) 매핑이 가리킨 `(표, 컬럼)` 에 CHECK 가 없으면 실패(= 표 이름 오타·표 이름 변경). "the mapping points at <표>.<컬럼> but no CHECK constrains it" 류.
     - (c) 매핑에도 없고 제외에도 없는 enum 파라미터가 모호하거나 미해결이면 기존 두 `t.Errorf` 로 계속 실패한다(기존 동작 유지 — 새 파라미터가 조용히 빠져나가지 못한다).
  4) 매핑이 실제로 돌고 있다는 개수 가드를 저장소 관용대로 파일 수준 상수로 둔다: `knownMappedEnums = 13`(매핑 경로로 비교된 쌍 수). 올려서 실패 문장을 눈으로 확인하고 되돌릴 것. 기존 `knownEnumQueries = 26`·`knownCheckConstraints = 42` 는 **바꾸지 말 것**(이번 변경으로 둘 다 안 바뀌어야 정상이다 — 바뀌면 걷기를 망가뜨렸다는 신호).
  5) 섭동으로 red 를 눈으로 확인하고 전부 `git checkout --` 로 되돌린다(최소 3종, 아래 "권장 섭동" 참고).
  6) 기존 테스트 이름·실패 문구를 바꾸지 않는다. 프로덕션 코드 0줄, `migrations/` 0줄, `.github/workflows/` 0줄.

- 건드릴 파일: **1파일뿐**
  - `internal/api/enum_contract_test.go`
    - `enumParametersNotBackedByACheck` (:43) — `"status"`, `"severity"` 두 항목 제거. `sort`·`prompt`·`forecastCategory` 세 항목은 **그대로 둘 것**(각각 sort 키, Keycloak 파라미터, CHECK 부재 — 오늘도 참이다).
    - 새 패키지 수준 변수 (예: `enumParameterTables map[string]map[string]string`, operation → 파라미터 → 표) 추가. 표 이름은 아래 실측표에서 그대로 옮길 것.
    - `TestDocumentedEnumValuesMatchTheMigrations` (:264) 안의 파라미터 루프 — 분기 순서를 **① 매핑 → ② 제외 → ③ 기존 모호/미해결** 로 둘 것. 지금은 ②가 맨 앞(:290)이다.
    - 새 상수 `knownMappedEnums` 를 `knownEnumQueries`·`knownCheckConstraints` 옆(:35-36)에 추가.
  - `migrationCheckConstraints`(:161)·`effectiveCheckValues`(:227)·`enumQueryParameters`(:89)·`snakeCase`(:75)·`sortedSet`(:251) 은 **시그니처 변경 없이 그대로 재사용**한다. `migrationCheckConstraints` 는 이미 `(표, 컬럼)` 키에 파일명 사전순 최신 승자를 담아 돌려주므로 매핑 조회에 바로 쓸 수 있다(:214-219 에서 덮어쓴다).

- 오늘의 실측 (정찰이 `internal/api/parameters.go` 와 `migrations/` 를 직접 열어 **눈으로 대조**한 표 — 비교를 코드로 돌려 본 것은 아니다. 착수 전에 임시 `t.Logf` 로 실제 (operation, 파라미터) 쌍 목록을 한 번 출력해 이 표와 맞는지 확인하고 임시 코드를 지울 것):

  | operation | 파라미터 | 표 | 공표 집합 (parameters.go) | CHECK (migrations) |
  |---|---|---|---|---|
  | `GET /voices` | status | `customer_voices` | RECEIVED, IN_REVIEW, IN_PROGRESS, PENDING_CUSTOMER, RESOLVED, CLOSED, REJECTED (`:112`) | `008_customer_voice.sql:35` 동일 |
  | `GET /voices/export` | status | `customer_voices` | 같음(`voiceFilters()` 공유) | 같음 |
  | `GET /opportunities` | status | `opportunities` | OPEN, WON, LOST (`:167`) | `001_initial.sql:255` 동일 |
  | `GET /signals` | status | `signals` | ACTIVE, RESOLVED, IGNORED (`:210`) | `011_crm_intelligence.sql:32` 동일 |
  | `GET /risks` | status | `risks` | OPEN, RESOLVED, ACCEPTED (`:215`) | `011_crm_intelligence.sql:62` 동일 |
  | `GET /insights` | status | `insights` | ACTIVE, EXPIRED (`:222`) | `011_crm_intelligence.sql:82` 동일 |
  | `GET /recommendations` | status | `recommendations` | OPEN, ACCEPTED, DISMISSED, COMPLETED (`:231`) | `011_crm_intelligence.sql:104` 동일 |
  | `GET /approvals` | status | `approval_requests` | PENDING, APPROVED, REJECTED, CANCELLED (`:253`) | `001_initial.sql:453` 동일 |
  | `GET /admin/mail/deliveries` | status | `mail_deliveries` | queued, sent, failed (`:263`) | `017_mail_notifications.sql:36` 동일 |
  | `GET /voices` | severity | `customer_voices` | LOW, NORMAL, HIGH, CRITICAL (`:114`) | `008_customer_voice.sql:34` 동일 |
  | `GET /voices/export` | severity | `customer_voices` | 같음 | 같음 |
  | `GET /signals` | severity | `signals` | LOW, MEDIUM, HIGH, CRITICAL (`:208`) | `011_crm_intelligence.sql:20` 동일 |
  | `GET /risks` | severity | `risks` | LOW, MEDIUM, HIGH, CRITICAL (`:214`) | `011_crm_intelligence.sql:51` 동일 |

  = 쌍 13개(status 9 + severity 4). **오늘 드리프트는 없다 — 이것은 그물이지 버그 수정이 아니다.** 공표되지 않는 `status` CHECK 3개(`revenue_schedules` `004:48`, `account_plans` `003:24`, `personal_keys` `001:491`)는 질의 파라미터가 없으므로 매핑에 넣지 말 것 — 이 그물은 문서→migrations 한 방향이다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/api/ -run TestDocumentedEnumValuesMatchTheMigrations -count=1 -v`  ← 정찰이 변경 전 돌려 `ok github.com/hkjang/relio/internal/api 0.005s` 확인함
  - `go test ./internal/server/ -run 'TestDocumented|TestEveryRouted|TestEveryDocumented|TestProjectionParameter|TestEveryPathTemplate' -count=1 -v`  (기존 7건 무변경 확인)
  - `go test ./...`  (기대: 23 ok / FAIL 0)
  - `go test -race ./internal/api/`
  - `go vet ./...` · `go build ./...` · `gofmt -l internal/api/enum_contract_test.go`(무출력) · `git diff --check`(무출력)
  - `./scripts/check-env-contract.sh` · `./scripts/check-static-assets.sh`
  - `make test` 는 **돌리지 말 것** — Go 테스트 1파일 변경이고 `npm ci` 가 네트워크를 탄다(지난 두 회차 선례).

- 권장 섭동 (전부 눈으로 보고 `git checkout --` 로 되돌릴 것):
  1. `parameters.go:167` 의 `choice("status", …, "OPEN", "WON", "LOST")` 에서 `"LOST"` 삭제 → **`GET /opportunities` 한 건만** red, `document {OPEN, WON}, migrations {LOST, OPEN, WON} (001_initial.sql:255)`.
  2. `parameters.go:114` 의 voices severity `"NORMAL"` → `"MEDIUM"` → **`GET /voices` 와 `GET /voices/export` 두 건만** red(표 공유가 실제로 작동한다는 증거). `GET /signals`·`GET /risks` 는 green 으로 남아야 한다 — 여기서 함께 red 가 나면 컬럼명 접기를 아직 쓰고 있는 것이다.
  3. `migrations/017_mail_notifications.sql:36` 의 CHECK 에 `'bounced'` 추가 → `GET /admin/mail/deliveries` 만 red. 반드시 `git checkout -- migrations/` 로 되돌리고 `git status` 로 확인.
  4. 기준 3(a) — 매핑에 가짜 항목(예: `"GET /nope": {"status": "opportunities"}`) 추가 → 낡은 매핑 실패.
  5. 기준 3(b) — `GET /approvals` 의 표를 `approval_requests` → `approvals` 로 오타 → "CHECK 없음" 실패.
  6. 기준 4 — `knownMappedEnums` 13→14 로 올려 실패 문장 확인.

- 위험과 피할 것:
  - **보호 경로 전부 회피**: `internal/auth`·`oidc`·`internal/server/public.go`·`.github/workflows/` 를 건드리지 않는다. `migrations/` 는 섭동으로만 만지고 반드시 되돌린다(최종 `git status --short` 가 테스트 1파일이어야 한다).
  - **두 파서를 합치지 말 것** — 기존 `effectiveCheckValues` 의 컬럼명 접기 경로는 `entityType`·`voiceType`·`sentiment`·`priority`·`causeEvidence`·`knowledgeStatus` 6개를 계속 지킨다. 매핑은 그 경로를 **대체하지 말고 우회**만 하게 둘 것. 접기 로직을 "표 매핑으로 통일" 하려 들면 6개를 위한 매핑 항목이 또 늘고 범위가 터진다.
  - 제외 staleness 가드(:317-326)는 `status`·`severity` 가 `ambiguous` 에 있다는 전제로 `continue` 한다. 두 항목을 제외 표에서 빼면 그 분기는 자연히 안 돈다 — 가드 코드를 지우지 말 것(`forecastCategory` 에 뒤늦게 CHECK 가 생기는 경우를 아직 잡는다).
  - `GET /voices` 와 `GET /voices/export` 는 `voiceFilters()` 를 **공유**하므로 매핑 항목이 두 개 필요하다(파라미터 선언은 한 자리, 공표는 두 operation). 한쪽만 적으면 기준 3(c) 로 red 가 난다.
  - `migrations/016_voice_workspaces.sql:52` 의 `WHERE status IN ('RESOLVED','CLOSED')` 는 부분 인덱스 술어이고 CHECK 가 아니다 — `checkInConstraint` 정규식(:68)이 `CHECK\s*\(` 를 요구하므로 걸리지 않는다. `knownCheckConstraints = 42` 가 그대로여야 한다.
  - 과거 교훈: 소스 문자열 grep 을 증거로 제출하지 말 것 — 섭동으로 red 가 나는 것을 실제 명령 출력으로 보일 것. 그리고 손 매핑으로 **결함을 증명하려 하지 말 것**(오늘 결함은 없다. 이 회차의 산출물은 그물이다).

- 차선 후보: **`severity` 만 먼저 닫기** (가치 3 / 위험 1 / S) — 위 설계를 그대로 쓰되 매핑에 severity 4쌍만 넣고 `enumParametersNotBackedByACheck` 에서 `"severity"` 만 제거하며 `"status"` 는 "operation→표 매핑이 아직 없다" 로 이유를 갱신해 남긴다. 1순위가 한 세션에 안 들어오면 이 조각만 담고 `status` 는 다음 회차로 넘길 것. 그 다음 차선은 `internal/mcp/server.go:683` 의 `record_voice_response` eventType enum `{CUSTOMER_CONTACT, COMMENT, ESCALATED}` 과 그 값을 **실제로 집행하는** `internal/voice/service.go:821` 의 하드코딩 삼중 비교를 묶는 계약 테스트 (가치 2 / 위험 1 / S) — 오늘 두 자리는 일치하고 `migrations/008:70`·`016:56` 의 `customer_voice_events.event_type` CHECK 는 의도된 상위 집합이라 집합 동일이 아니라 **부분집합** 으로 확인해야 한다(정찰이 세 자리 모두 열어 확인).

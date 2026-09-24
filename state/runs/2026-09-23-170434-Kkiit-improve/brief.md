# 과제서 (2026-09-23, base main@ee4c612)

- 과제: 승인 정책의 배열 조건(service_types·seller_levels)에 잘못된 타입이 들어오면 저장 단계에서 400 으로 거부하기 (가치 3 / 위험 2 / 작업량 M)

- 왜: `internal/httpapi/approvals.go:49 validateApprovalPolicy` 는 숫자 조건 세 개(min_amount·max_amount·quality_score_below)와 steps 만 검사하고 `service_types`·`seller_levels` 는 전혀 보지 않는데, 매칭 쪽 `internal/httpapi/talents.go:637 approvalConditionsMatch` 는 `conditions["service_types"].([]any)` 로 타입 단언에 실패하면 그 조건을 **조용히 건너뛴다**. 그래서 API 로 `"service_types": "design"` 이나 `["design", 3]` 같이 저장하면 200/201 이 돌아오지만 정책은 "디자인만" 이 아니라 **모든 상품**에 걸려, 즉시 공개되어야 할 상품까지 검토 대기열로 들어간다(README 637 줄이 조건 키 다섯 개를 공개 계약으로 적고 있으므로 API 직접 호출은 정상 경로다). 저장 때 거부하면 관리자가 잘못 적은 자리를 바로 알 수 있고, 화면(`web/src/pages/AdminPage.tsx:310 ApprovalConditions`)이 이미 "일부 조건 확인 필요" 로 표시하는 규칙과 서버 규칙이 같아진다.

- 수용 기준:
  1. `POST /api/v1/admin/approvals/policies` 와 `PUT /api/v1/admin/approvals/policies/{id}` 가 `conditions.service_types` 또는 `conditions.seller_levels` 에 배열이 아닌 값, 문자열이 아닌 원소, 공백뿐인 문자열이 들어오면 **둘 다** 400 `invalid_policy` 를 돌려주고 DB 행이 만들어지거나 바뀌지 않는다(GET 으로 확인).
  2. 400 본문 message 가 문제 키 이름(`service_types` / `seller_levels`)을 담아 관리자가 어디를 고칠지 알 수 있다. 빈 배열 `[]` 과 키 자체가 없는 경우는 지금처럼 그대로 허용한다(매칭은 `len(values) > 0` 에서만 거르므로 동작이 바뀌지 않아야 한다).
  3. 정상 배열 조건(`{"service_types":["design"]}`)으로 만든 정책은 종전과 똑같이 동작한다 — 실제 라우터·실제 PostgreSQL 로 상품을 공개해 `service_types` 가 맞는 상품은 `review_pending`, 맞지 않는 상품은 즉시 공개되는 것이 테스트로 증명되어야 한다(저장 검증만 고치고 매칭이 깨지지 않았음을 같은 값의 다른 경로에서 확인하는 것이 목적이다).

- 건드릴 파일:
  - `internal/httpapi/approvals.go:49 validateApprovalPolicy` — 숫자 키 검사 뒤에 `service_types`·`seller_levels` 검사를 추가한다. 규칙은 화면과 같게: 키가 있으면 `[]any` 여야 하고 모든 원소가 `string` 이며 `strings.TrimSpace(...) != ""`. 실패 이유를 호출자에게 전달해야 하므로 반환형을 `(string, bool)`(예: reason, ok) 로 바꾸는 것을 권한다. 새 헬퍼를 파일 안에 두고 `numericValue` 옆에 붙이면 된다.
  - `internal/httpapi/approvals.go:130 createApprovalPolicy`, `:150 updateApprovalPolicy` — 두 호출부 모두 새 반환값을 받아 `writeError(w, 400, "invalid_policy", ...)` 메시지에 이유를 넣는다. **한쪽만 고치지 말 것**(저장 경로가 둘이다).
  - `internal/httpapi/auth_test.go:90 TestValidateApprovalPolicyRejectsInvalidConditions` — 기존 단위 테스트가 `validateApprovalPolicy` 를 bool 로 쓰므로 반환형을 바꾸면 여기도 고쳐야 한다. 배열 조건 사례(문자열 그대로, 원소가 숫자, 공백 문자열, 정상 배열, 빈 배열)를 이 테스트에 덧붙인다.
  - `internal/httpapi/integration_test.go` — 기존 `TestIntegrationApprovalQueueReportsTheWait`(3124 줄) 또는 `TestIntegrationEditingAnApprovedTalentReturnsToReview`(1928 줄) 옆에 새 통합 테스트를 하나 추가한다. 헬퍼는 `integrationServer`(39 줄)와 `sellTalent`(369 줄)를 그대로 쓴다. 수용 기준 1·3 을 실제 HTTP → 실제 DB 로 통과시킨다.
  - 문서: 필요하면 `README.md:637` 의 조건 설명에 "서비스 유형·판매자 등급은 문자열 배열" 한 구절만 덧붙인다(선택). **프런트(`AdminPage.tsx`)는 건드리지 말 것** — 이미 같은 규칙으로 표시하고, `save`(350 줄)는 알려진 폼 필드에서만 조건을 재구성하므로 잘못된 배열을 보내지 않는다. 따라서 `internal/ui/dist` 재빌드도 불필요하다.

- 검증 명령:
  - `go test ./internal/httpapi/ -run 'ValidateApprovalPolicy' -count=1 -v` — 단위. **고치기 전에 먼저 새 테스트가 실패하는 것을 보고 나서 구현할 것**(TDD).
  - `go test ./cmd/... ./internal/...` — 전체(DSN 없으면 통합은 SKIP). 이번 정찰에서 `go test ./internal/httpapi/` 는 0.34초에 통과하는 것을 확인했다(Go 1.26).
  - `make test-integration KKIIT_TEST_DSN=postgres://...` — 버릴 PostgreSQL 16 인스턴스 필요(`-run Integration -count=1 -timeout 300s`). 과거 회차는 임시 `postgres:16-alpine` 컨테이너로 79~84초에 전체 통합을 돌렸다. **이번 회차에서 docker 가용성은 확인하지 못했다(미확인)** — 컨테이너를 못 띄우면 수용 기준 1·3 은 단위 테스트만으로는 증명되지 않으니, 그 경우 그 사실을 회차 노트에 적고 차선 후보로 바꿀 것.
  - `go vet ./cmd/... ./internal/...`, `gofmt -l internal` (make check 첫 줄이 gofmt 를 본다).

- 위험과 피할 것:
  - **알 수 없는 조건 키는 이번에 거부하지 말 것.** 지금 `validateApprovalPolicy` 는 모르는 키를 통과시키고 매칭은 무시한다. 함께 조이면 기존 행을 PUT 으로 다시 저장할 수 없게 되고, 보류 아이디어 "알 수 없는 승인 조건 편집 시 보존" 과 섞인다. 이번 범위는 **배열 두 키의 타입**뿐이다.
  - **매칭 쪽(`approvalConditionsMatch`)의 타입 단언을 넓히거나 문자열을 배열로 보정(coerce)하지 말 것.** 값을 추측해 고치면 관리자가 오타를 눈치채지 못한 채 다른 정책이 돌아간다. 운영자 지침("파서를 넓히지 말 것", "실제 동작이 바뀌지 않는 수정은 넣지 말 것")에 걸린다. 고치는 자리는 **쓰기 검증 한 곳**이다.
  - 빈 배열·키 없음의 현재 동작을 바꾸면 기존 정책의 적용 범위가 달라진다. 반드시 통과시켜라.
  - 보호 경로 `auth.go`·`middleware.go`·`orders.go`·`finance.go`·`internal/database/migrations`·`.github/workflows` 는 손대지 않는다. 마이그레이션은 필요 없다 — `001_initial.sql:388` 에 승인 정책 시드가 없어 이번 검증으로 깨질 기존 행이 저장소 안에 없다(운영 DB 의 기존 행은 미확인이지만, UI 편집은 조건을 폼 필드에서 다시 만들므로 자동으로 정상화된다).
  - 통합 테스트는 전역 `apiUnderTest` 때문에 병렬화 금지(`t.Parallel()` 쓰지 말 것). 테스트 계정·상품 이름은 기존처럼 `uniqueName` 으로.
  - 빌드 산출물(`internal/ui/dist`)을 커밋하지 말 것.

- 차선 후보: **README 환경변수 계약을 "필수 4개 + 선택 SHUTDOWN_DRAIN_SECONDS" 로 정리 (가치 2 / 위험 1 / S)** — `internal/config` 의 `Load` 가 `SHUTDOWN_DRAIN_SECONDS`(기본 5, 0~120, 파싱 실패 시 5)도 읽는데 README 와 `config.Config` 주석은 환경변수가 네 개뿐이라고 적는다. 코드를 바꾸지 않고 문서·주석만 실제 동작과 맞추면 되므로 DB·docker 가 없어도 끝낼 수 있다. 착수 전에 `internal/config/config.go` 의 Load 를 읽고 기본값·범위를 직접 확인할 것.

## 견적 근거 (basis of estimate, bottom-up)
- 검증 헬퍼 + 두 호출부 수정: 20분 / 단위 테스트(먼저 실패 확인 포함): 10분 / 통합 테스트 작성: 15분 / 전체 테스트·통합 컨테이너 실행: 15분. 합계 **40~70분, 신뢰도 8/10**.
- 포함: Go 코드·테스트·README 한 구절. 제외: 프런트 변경, dist 재빌드, 화면 캡처, 알 수 없는 키 처리, PDF 가이드 재생성.
- 가장 크게 기대는 가정: **버릴 PostgreSQL 16 을 띄울 수 있다**(docker 미확인). 이것이 틀리면 수용 기준 3 을 증명할 수 없고 과제는 차선 후보로 넘어간다 — 여기서 계획이 깨진다면 그 자리다.

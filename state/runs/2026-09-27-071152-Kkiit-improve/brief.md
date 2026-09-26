# 과제서 (2026-09-27, Kkiit)

- 과제: 신고 접수(`createReport`)가 모든 INSERT 실패를 409 "이미 접수되어 처리 중인 신고가 있습니다." 로 보고하는 것을 중복(23505)만 409 로 가르기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/httpapi/reports.go:74-79` 의 `createReport` 는 `INSERT INTO reports(...)` 의 **모든** 오류를 `writeError(w, 409, "report_already_open", ...)` 로 묶는다. `reports_open_unique_idx`(`012_reports.sql:6`, reporter_id·resource_type·resource_id, state IN ('open','reviewing'))에 걸린 진짜 중복만 그 안내가 맞고, 그 외의 저장 실패(DB 장애, 인코딩 거절 등)는 중복이 하나도 없는데도 "이미 접수되어 처리 중" 이라고 답해 신고자가 신고가 접수됐다고 오해하고 포기한다. 2026-09-26 회차에서 `createCoupon` 의 동일한 결함(모든 INSERT 오류 → 409)을 고칠 때 만든 `isUniqueViolation`(`internal/httpapi/coupons.go:19`, SQLSTATE 23505)이 같은 패키지에 이미 있으므로, 같은 기준을 신고 경로에도 적용하면 관리자·신고자가 보는 안내가 실제 원인과 일치한다.

- 수용 기준:
  1) 같은 신고자가 같은 대상에 열린 신고를 한 번 더 올리면 **종전대로** 409 `report_already_open` 이 나온다(회귀 없음).
  2) 중복이 아닌 저장 실패는 409 가 아니라 500 `report_failed`("신고를 접수하지 못했습니다." 같은 문구)로 나오고, 응답 본문의 `error.code` 가 `report_already_open` 이 아니다.
  3) 테스트는 `isUniqueViolation` 분류기를 두 방향으로 변이시키면(항상 참 / 23505 를 절대 안 맞춤) 각각 실패해야 한다 — 즉 분류가 실제로 동작하는 것을 증명한다. 실제 라우터 + 버릴 PostgreSQL 로 HTTP→DB 왕복으로 확인할 것(손으로 만든 대역 금지).

- 건드릴 파일 (프로덕션 2개 + 테스트/문서):
  - `internal/httpapi/reports.go:createReport` — INSERT 의 `err` 를 `isUniqueViolation(err)` → 409 `report_already_open`, 그 외 → 500 `report_failed` 로 가른다. 검증·권한·대상 존재 확인(40~72행)은 그대로 둔다.
  - (필요하면) `internal/httpapi/coupons.go:isUniqueViolation` — 위치만 그대로 쓰고 수정하지 말 것. 쿠폰 전용 주석이 걸린다면 주석 한 줄만 일반화하거나, 같은 파일에 두고 그대로 재사용한다(파일 이동·리네임 금지 — 쿠폰 회차 테스트가 이 함수를 본다).
  - `internal/httpapi/integration_test.go:TestIntegrationReportTakesDownATalentAndHidesTheSeller`(1016행~, 1023~1029행에 이미 201 + 409 중복 사례가 있다) — 여기에 붙이거나 별도 `TestIntegrationReport...` 를 새로 만든다. 헬퍼: `integrationServer(t)`(40), `newClient`(104), `client.do(method, path, body, wantStatus)`(124), `uniqueName`(168), `sellTalent`(377), `operatorClient`(544). 주의: `do` 는 기대 상태코드가 다르면 `t.Fatalf` 하므로 "409 가 아님" 을 확인할 때는 기대값을 500 으로 명시하거나 `raw`/직접 `http.Client` 로 상태코드를 읽는다.
  - `docs/openapi.yaml` — POST `/reports`(785행 `post:`, 801~804행 `responses:`)에 `'201'`·`'400'`·`'409'` 만 있고 `'500'` 이 없다(확인). `'500': { $ref: '#/components/responses/Error' }` 한 줄만 추가한다. 같은 자리의 `'404'`(대상 없음, `reports.go:69`) 누락은 이번 과제 범위 밖 — 손대지 말 것.
  - 프런트(`web/`)·`internal/ui/dist` 는 건드리지 않는다. 화면은 이미 `error.message` 를 띄운다(2026-09-26 회차에서 쿠폰으로 확인된 관례).

- 비중복 실패를 실제로 일으키는 방법 (**미확인 — 착수 시 먼저 확인할 것**):
  - 1순위: `details` 에 NUL 문자를 넣는다 — 예 `{"resource_type":"talent","resource_id":"<uuid>","reason":"spam","details":"a\u0000b","evidence":[]}`. PostgreSQL 의 text/jsonb 는 0x00 을 저장할 수 없어 INSERT 가 23505 가 아닌 오류(대략 SQLSTATE 22021 `invalid byte sequence for encoding "UTF8": 0x00`)로 실패할 것으로 본다. pgx 가 서버 이전에 클라이언트 쪽에서 거절하더라도 `err != nil` 이고 `isUniqueViolation(err)==false` 이므로 수용 기준 2 의 관찰(409 가 아님)은 그대로 성립한다. 핸들러 앞단 검증(40~72행)은 NUL 을 막지 않는다 — `decodeJSON`(`internal/httpapi/types.go:102`)은 `encoding/json` 의 `Decode` + `DisallowUnknownFields` 뿐이므로 JSON 으로 유효한 `\u0000` 을 그대로 Go 문자열로 통과시키고(확인), `details` 에 대한 유일한 검사는 룬 길이 4000 이하다(`reports.go:53`). 서버가 실제로 어떤 오류를 내는지는 **미확인** — 착수하면 이 요청 한 건으로 먼저 현재 동작(409 `report_already_open`)을 재현해 보고 시작할 것.
  - 2순위(1순위가 통하지 않으면): `evidence` 배열 원소 값에 NUL 을 넣어 jsonb 인코딩을 거절시킨다.
  - 3순위: 그래도 실제 HTTP 로 비중복 실패를 못 만들면, 수용 기준 2 는 `isUniqueViolation` 분기를 직접 부르는 단위 테스트(비-PgError·다른 SQLSTATE 를 넣어 409 가 아님을 확인)로 증명하고, 통합 테스트는 수용 기준 1(진짜 중복 → 409)만 담는다. 이 경우 과제서에 "비중복 실패는 실제 HTTP 로 재현하지 못했다" 를 PR 설명에 명시할 것. **소스 문자열 검사(grep)를 증거로 쓰지 말 것.**
  - 만약 NUL 입력을 400 으로 막는 것이 옳다고 판단되더라도 **이번 회차 범위에 넣지 말 것**(입력 검증 정책 변경은 별 과제). 이번 변경은 오류 분류 한 곳뿐이다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l cmd internal` (출력 없어야 함)
  - `go vet ./cmd/... ./internal/...`
  - `go test ./internal/httpapi/ -run TestIntegrationReport -count=1` (KKIIT_TEST_DSN 필요)
  - 전체: `KKIIT_TEST_DSN=postgres://... go test ./cmd/... ./internal/...` — `internal/httpapi` 가 80~95초 걸린다. DSN 은 버릴 PostgreSQL 16(docker)로; 계정·주문·신고 행이 남으므로 운영 DB 금지. `make test-integration KKIIT_TEST_DSN=...` 도 있다.
  - DSN 없이 통과한 `go test` 는 통합 검증이 아니다(통합 테스트가 SKIP 된다).
  - **미확인**: 이번 세션에서 docker 가용성을 확인하지 못했다(권한 프롬프트로 `docker version` 을 못 돌렸다). 2026-09-26 회차는 docker 29.7.2 로 통합을 실제 실행했다.

- 위험과 피할 것:
  - `internal/httpapi/integration_test.go` 는 전역 `apiUnderTest` 를 쓰므로 테스트 병렬화(`t.Parallel()`) 금지.
  - `reports.go` 의 다른 부분(`resolveReport`, `reports.go:284` 의 `UPDATE users SET status='suspended'`)은 계정 정지·원장에 닿는다. 이번 과제는 `createReport` 한 함수만 건드린다.
  - 보호 경로(`auth.go`, `middleware.go`, `identity.go`, `database/migrations`, `.github/workflows`)는 건드리지 않는다. 마이그레이션 추가 불필요(스키마 변경 없음).
  - `isUniqueViolation` 을 다른 파일로 옮기거나 시그니처를 바꾸면 쿠폰 쪽 테스트가 같이 흔들린다 — 재사용만 하라.
  - 같은 `err != nil || tag.RowsAffected() == 0` / "모든 오류를 한 코드로" 패턴이 `keys.go:174`, `webhooks.go:175`, `discovery.go:196`, `admin_users.go:72`, `approvals.go:229`, `coupons.go:349`(deleteCoupon) 등 10곳 이상에 있다. **이번에는 손대지 말 것** — 파일 수가 불어나면 사람 손을 다시 타고, 대부분은 고유 제약이 없어 증거 설계가 다르다.
  - 빌드 산출물(`internal/ui/dist`) 커밋 금지.

- 차선 후보: README 환경변수 계약을 필수 4개(`POSTGRES_DSN`/`BOOTSTRAP_ADMIN`/`BOOTSTRAP_ADMIN_PASSWORD`/`ENCRYPTION_KEY`) + 선택 `SHUTDOWN_DRAIN_SECONDS`(기본 5, 0~120) 로 정리 — `internal/config/config.go` 의 `Load`/`intFromEnv` 가 정본이고 README 48행 표에 선택 항목이 빠져 있다. DB·docker 없이 끝난다. 1순위가 성립하지 않는 조건: docker/PostgreSQL 을 띄울 수 없어 통합 왕복을 전혀 만들 수 없을 때.

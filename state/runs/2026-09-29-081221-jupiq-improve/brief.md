- 과제: 메일 발송 기록 API의 잘못된 status를 400 invalid_query로 거부 (가치 2 / 위험 1 / 작업량 S)
- 왜: `GET /api/v1/mail/deliveries`의 `mailDeliveries`는 status를 검증하지 않아 `status=faild` 같은 오타도 정상 조회로 처리하지만 OpenAPI는 queued·sent·failed만 선언한다. 오타를 발송 기록 없음으로 오인하지 않도록 명확한 입력 오류를 반환하고, 기존 정상 필터·공백 정규화 동작은 보존한다.
- 수용 기준:
  1) `strings.TrimSpace` 후 빈 값 또는 `mail.StatusQueued`·`mail.StatusSent`·`mail.StatusFailed`만 통과한다. `nonsense`, `SENT`, `sent,failed`, 내부 공백이 있는 `se nt`는 400이며 JSON `error.code=invalid_query`, `error.message`는 status와 허용값을 안내한다. 입력 원문을 메시지에 반사하지 않는다.
  2) 미지정·빈 문자열·공백뿐(탭·NBSP 포함)은 전체 조회, ` sent `·탭/NBSP로 감싼 sent는 sent 조회로 종전과 동일한 200이다. URL은 `url.Values.Encode`로 만들어 HTTP 파싱과 Store까지 같은 입력이 흐르도록 한다. 대소문자 자동 보정은 하지 않는다.
  3) queued·sent·failed 각 실제 기록을 만들어 요청 필터에 맞는 행만 오는 것을 ID·Status로 단언한다. 전체 조회와 각 필터에서 Total·Summary는 종전처럼 전체 집계이며 limit 기본값·상한 동작을 바꾸지 않는다.
  4) 실제 PostgreSQL·store.Store·auth.Service의 세션·`New(...).Handler()`를 거친 HTTP 통합 테스트로 1~3을 증명한다. 테스트 이름에 Integration을 넣는다. 수정 전 잘못된 status가 200인 실패, 수정 후 통과를 확인하고 검증 분기만 잠시 제거하면 잘못된 status 단언이 다시 실패해야 한다. Fake Store, 직접 만든 Principal만으로 결함 증명을 대체하지 않는다.
- 건드릴 파일:
  - `internal/api/mail_handlers.go:mailDeliveries` — 기존 queryIntOrReject 성공 뒤 status를 TrimSpace하고 위 상수로 switch 검사, 거부 시 apiError 후 return, 통과한 같은 값을 ListMailDeliveries에 전달. 이미 strings와 mail을 import하므로 새 추상화나 의존성 불필요. registerMail의 settings:read는 그대로 둔다.
  - `internal/api/mail_deliveries_integration_test.go` (신규) — 실제 HTTP 배선 회귀 테스트. 프로덕션 파일은 1개, 테스트 포함 2개를 목표로 한다.
- 검증 명령:
  - 정찰 실측: `go test -count=1 ./internal/api ./internal/store` 통과(api 0.045s, store 0.006s). DSN 없는 실행이라 DB 동작 증명은 아님.
  - 구현 후: `go test -count=1 -v -run TestMailDeliveriesStatusIntegration ./internal/api` (제안 테스트명, DSN 필수); `make test-integration`; `go test -count=1 ./...`; `go vet ./...`; `gofmt -l internal/api/mail_handlers.go internal/api/mail_deliveries_integration_test.go`.
  - DB 준비는 README의 실제 절차를 따른다. 예: `docker run -d --name jupiq-mail-status-it -e POSTGRES_DB=jupiq_test -e POSTGRES_USER=jupiq -e POSTGRES_PASSWORD=it -p 127.0.0.1:55439:5432 postgres:16-alpine`, 준비 완료 후 `export JUPIQ_INTEGRATION_TEST_DSN='postgres://jupiq:it@127.0.0.1:55439/jupiq_test?sslmode=disable'`. 포트·이름 충돌이면 별도 빈 값 사용. 종료 시 본인이 띄운 컨테이너만 `docker rm -f jupiq-mail-status-it`로 제거. 정찰은 컨테이너를 생성하지 않았으며 위 포트 가용성은 미확인.
  - 새 통합 테스트가 SKIP이면 미검증이다. `make test-integration`은 DSN 없으면 실패한다. SMTP·브라우저 실행은 필요 없다.
- 위험과 피할 것: 공백 정규화는 `internal/store/mail.go:ListMailDeliveries`가 이미 적용하므로 raw enum 검사로 바꾸면 회귀다(초안에서 이 부분을 정정했다). Store SQL·Total/Summary 의미·메일 발송·auth·migrations·workflows·VERSION·가이드 PDF·의존성은 범위 밖이다. 반복 쿼리 키 처리는 기존 URL.Query().Get 관례를 유지한다. 본 작업을 공용 enum 프레임워크나 다른 API 검증으로 넓히지 않는다.
- 차선 후보: OpenAPI `/users` page_size maximum 100을 실제 공용 pageBounds 상한 200과 일치시키는 문서 수정 — 1순위가 이미 구현됐거나 명시적으로 임의 status 허용 계약이 발견된 경우만 전환. 이번에 같이 구현하지 않는다.

근거와 테스트 배선:
- 기준 main@e2e11c1, VERSION 1.8.5. 실제 열람: mail_handlers.go:19(mailDeliveries), store/mail.go:119(ListMailDeliveries), openapi.yaml:631~639(enum), mail/service.go:Delivery·StatusQueued/Sent/Failed, helpers.go:queryIntOrReject·data·errorBody, server.go:New·Handler, middleware.go:require.
- `internal/api/profile_email_integration_test.go:TestProfileEmailValidationHTTPIntegration`의 store.Open → auth.NewService → CreateSession → SecureCookie → New(...).Handler() 배선을 따른다. 단 프로필 테스트 계정에는 settings:read 보장이 없으므로 `internal/store/mail_integration_test.go:TestMailStoreContractsIntegration`의 Seed 관리자 생성 패턴을 조합한다. Seed 후 실제 GetUser로 세션을 만들고 권한 주입 대역은 쓰지 않는다.
- 기록은 실제 `RecordMailDelivery`로 세 건 생성하고 둘을 `FinishMailDelivery`로 sent·failed로 만든다. marker와 생성 ID로 정리하며 실제 SMTP를 호출하지 않는다. data()가 `{"data":...}`로 감싸므로 내부를 `store.MailDeliveryPage`로 디코드한다. DB를 공유할 경우 다른 행이 있을 수 있으므로 marker ID 포함 여부와 필터된 모든 행의 Status를 검사하고 전역 Total을 무조건 3으로 단언하지 않는다.
- query_params_test.go의 기존 nil Store 테스트는 잘못된 정수에 대한 선례일 뿐 이번 실제 HTTP 통합 검증을 대신하지 않는다. SettingsPage.tsx:MailDeliveriesCard는 status를 보내지 않고 summary만 사용한다.

대안 비교와 선택 가정:
- API 입구의 작은 switch(선택): 1개 프로덕션 파일, 문서 enum·오류 관례와 일치하며 Store 직접 호출 계약은 그대로다. 핵심 가정은 문서 밖 상태를 성공으로 받아야 할 제품 요구가 없다는 것; 열람한 가이드·프런트에는 그런 요구가 없다.
- Store에 검증 오류형 추가: 모든 호출자를 통제할 때 타당하지만 오류 매핑·직접 Store 계약까지 넓어진다. 이번 HTTP 입력 문제에는 과하다.
- 무변경/문서 enum 제거: 기존 느슨한 동작을 공식 계약으로 삼을 때 가능하지만 오타를 기록 없음과 구별하지 못한다. 기존 문서가 세 상태를 명시하므로 선택하지 않는다.

실행 계획(모두 미착수, 사람 승인 체크포인트 없음):
1. 위 테스트 파일에 정상·공백 사례와 실제 배선을 먼저 추가하고 지정 `go test ... -run TestMailDeliveriesStatusIntegration`으로 정상 경로가 통과하는지 확인한다. 체크포인트: 실제 인증 200과 fixture 정리 확인; 현실이 다르면 과제서부터 갱신한다.
2. 같은 테스트에 잘못된 status 사례를 추가해 기존 200을 재현한 뒤, mailDeliveries에 검사만 넣고 동일 명령으로 통과시킨다. 수정 전 실패는 기록하되 이 단계는 실패 상태로 인계하지 않는다. 체크포인트: 수용 기준 1~3 및 검사 제거 시 재실패 확인.
3. make test-integration·전체 Go 테스트·vet·gofmt 검사를 실행한다. 체크포인트: 통합 SKIP 여부·프로덕션 파일 수·diff 범위를 확인하고 테스트 결과를 journal에 적는다.

작업량 근거(pmo): bottom-up 추정으로 실제 HTTP fixture·정상 사례 12~16분, 거부 사례·핸들러 수정 6~9분, 되돌림 확인·검증·정리 7~10분 = 기본 25~35분. 알려진 불확실성(DB 시작·Seed 권한·fixture 충돌)에 예비 5~10분을 별도로 두어 총 30~45분을 예상한다(정찰자의 중간 신뢰도 추정, 통계적 확률·완료 보장 아님). 관리 예비는 배정하지 않는다; 새 범위 발견 시 확장 대신 과제서를 갱신한다. 지난 세 limit 보정·usage 응답 보정은 더 많은 프로덕션 파일도 한 회차에 끝났으나 실제 소요 분이 없어 유사 사례는 범위의 정성적 교차 확인으로만 썼다. PostgreSQL 이미지·Go 캐시를 사용할 수 있다는 가정이며 환경 준비가 길어지면 남은 시간을 재산정한다.

스킬: 전용 Skill 도구는 세션에 없었고 로컬 headcount의 pmo/skills/estimating-and-contingency, technology/skills/implementation-planning, technology/skills/solution-exploration 각 SKILL.md를 읽어 위 범위·대안·분해 추정·검증 체크포인트에 적용했다. pmo references/sources.md도 읽었으나 외부 원전 기반의 정량 비용·확률 주장은 하지 않았다.

- 과제: chat 요청의 첫 JSON 뒤 추가 JSON·쓰레기 문자를 공급자 호출 전에 400으로 거부 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/server/providers.go:chatCompletions`는 `json.Decoder.Decode`를 한 번만 호출하여 정상 객체 뒤의 추가 입력을 무시하며, 실제 서버에서 후행 `{}`·`null`·`xyz` 모두 HTTP 200과 upstream 1회 호출을 재현했다. 본문 끝을 확인하면 잘못된 요청이 정상 AI 호출로 실행되는 것을 막고 호출자가 `invalid_json`으로 원인을 알 수 있다.
- 수용 기준:
  1) 인증된 `POST /v1/chat/completions`의 정상 chat 객체 뒤에 두 번째 객체(` {}`), JSON null(` null`), 비JSON 쓰레기(` xyz`)가 붙으면 모두 400 `invalid_json`; 첫 공급자 선택·upstream 호출 전에 반환한다. 두 번째 Decode가 성공(nil)한 경우도 반드시 거부한다.
  2) 정상 단일 객체 및 그 뒤 JSON 공백(space/tab/CR/LF)은 HTTP 200, upstream 호출 정확히 1회, upstream의 정상 JSON 답변이 최종 client까지 전달된다. 사용자 messages 및 기존 허용 확장 필드(예: 작은 정수 seed=42)를 보존하고 provider_id는 upstream에서 제거된다.
  3) 실제 전용 PostgreSQL의 Migrate/Seed → `New(db,cipher,logger).Handler()` → `signIn` → 공급자 생성 API → 실제 HTTP chat 경로를 통과하는 통합 테스트가 1)·2)를 증명한다. 거부 요청마다 upstream 호출 수 증가가 0인지 검사한다. 직접 주입한 principal·가짜 DB·핸들러 직접 호출·소스 문자열 검사는 증거로 쓰지 않는다.
  4) 기존 4 MiB MaxBytesReader를 유지하고, 정상 객체 뒤 공백을 붙여 총 본문이 4 MiB를 넘는 경우도 끝 검사 중 오류를 성공 EOF로 취급하지 않아 400 `invalid_json` 및 upstream 0회가 된다. 이 경계 사례의 수정 전 실제 HTTP 재현은 미확인; 위 세 후행 입력 사례는 재현 완료다. 정상 JSON 안 문자열의 `{}`·`null`·`xyz`는 손상시키지 않는다.
- 건드릴 파일:
  - `internal/server/providers.go:chatCompletions`(558행 시작, 569행 이후 파서) — 기존 Decoder를 변수로 보관하고 첫 Decode 성공 직후 같은 Decoder로 추가 값을 읽어 정확히 `io.EOF`인 경우만 계속한다. 두 번째 읽기는 `json.RawMessage` 등으로 받을 수 있다. `io`는 이미 import되어 있다. 기존 `invalid_json` 코드/한국어 메시지를 그대로 사용하고 내용을 응답·감사에 덧붙이지 않는다. 프로덕션 코드는 이 1파일만.
  - `internal/server/chat_request_json_integration_test.go`(신규) — `TestChatRequestJSONContract`. 실제로 읽은 `chat_response_content_type_integration_test.go:TestChatNonStreamingResponseContentTypeContract`의 25~125행 셋업을 참고하고 `db_error_integration_test.go:signIn`을 재사용한다. 그 파일과 기존 helper는 수정하지 않는다. `sessionCredentials.do`는 body를 json.Marshal하므로 raw 후행 입력을 보낼 때 사용하지 말고, 실제 `http.NewRequest` + `bytes.NewReader`에 session.cookies와 session.csrf를 넣는다. 명시적 `stream:false`, `Content-Type: application/json`, 유한 HTTP client timeout, 고루틴 안전한 upstream 카운터(atomic 또는 mutex)를 사용한다. 정상 fixture upstream은 실제 httptest HTTP 서버이며 운영의 client/Handler 배선은 그대로 둔다.
  - `docs/api.md:OpenAI 호환 chat`(252행 이후) — chat 본문은 단일 JSON 문서이며 후행 공백만 허용, 추가 JSON/쓰레기는 400 `invalid_json`이라는 계약 한 문장. 다른 절 수정은 제외한다.
- 검증 명령:
  - 전용 폐기 DB 준비(55561 사용 가능 여부 확인, 이미 쓰는 컨테이너/DB 재사용 금지): `docker run --rm -d --name ai-admin-json-contract-pg -p 127.0.0.1:55561:5432 -e POSTGRES_USER=ai_admin -e POSTGRES_PASSWORD=test-password -e POSTGRES_DB=ai_admin_test postgres:16-alpine`; `docker exec ai-admin-json-contract-pg pg_isready -U ai_admin -d ai_admin_test`가 성공한 후 진행. 정찰에서 같은 이미지·포트로 생성/실행/정리에 성공했으며 정찰 컨테이너는 이미 제거했다.
  - `export TEST_POSTGRES_DSN='postgres://ai_admin:test-password@127.0.0.1:55561/ai_admin_test?sslmode=disable'`
  - 신규 테스트 먼저: `go test -count=1 -run '^TestChatRequestJSONContract$' -v ./internal/server` — 수정 전 후행 객체/null/쓰레기의 200 및 upstream 호출을 red로 기록하고 수정 후 모든 사례 PASS를 확인한다. 파일이 없는 정찰 단계에서는 이 명령을 실행한 적 없으며, 신규 테스트 추가 후에만 유효하다.
  - 관련 회귀: `go test -race -count=1 -run 'Test(ChatRequestJSONContract|ChatNonStreamingResponseContentTypeContract|ChatAnswerCutShortIsNotDeliveredAsComplete|RelayChatBodyReportsWhyTheAnswerStopped)$' -v ./internal/server`
  - 전체: `go test -race -count=1 ./...`; `make lint`; `go build ./...`; `git diff --check`. 같은 DB의 테스트 프로세스를 병렬 실행하지 않는다. 종료 후 `docker stop ai-admin-json-contract-pg`.
  - 정찰 실제 실행: 같은 DSN으로 `go test -count=1 -run 'Test(ChatNonStreamingResponseContentTypeContract|ChatAnswerCutShortIsNotDeliveredAsComplete|RelayChatBodyReportsWhyTheAnswerStopped)$' -v ./internal/server` → 최상위 3개 PASS, SKIP 없음, 서버 1.639초. 전체/race/lint/build/웹 검증은 이번 정찰에서 미실행.
- 위험과 피할 것: 공용 `response.go:decodeJSON`(2 MiB·DisallowUnknownFields)은 변경하거나 chat에 재사용하지 않는다. chat의 기존 4 MiB 한도·자유 확장 필드·float64 숫자 파싱·모델/토큰 정책·인증/CSRF·stream/relay/audit 배선을 보존한다. 첫 값이 null인 별도 결함은 이번 범위 밖이다. `UseNumber` 도입은 큰 seed 보존뿐 아니라 토큰 지수 표기 계약도 바꿀 수 있으므로 이번에 섞지 않는다. auth, migrations, workflows, 웹/dist, VERSION/CHANGELOG 수정 금지. b5146f2 scope 검증(verify-failed), f66d25c preferences 검증(미머지), 8e65c33 비스트리밍 절단 테스트(다른 브랜치에 구현됨)는 재구현하지 않는다. 검출 문자열/요청 원문을 감사 details에 넣지 않는다. 테스트 DB는 schema를 DROP CASCADE하므로 운영/공유 DB 사용 금지.
- 차선 후보: 프로필 email 빈 문자열의 NULL 삭제·생략/null 유지 계약 테스트·문서화 (가치 1 / 위험 1 / S) — 1순위가 이미 다른 작업으로 해결된 경우에만 선택한다. `users.go:updateProfile`의 `Email *string` 및 `emailSupplied` SQL을 읽어 동작은 확인했지만 전용 HTTP 재현은 미확인이다. `displayName`은 필수로 포함하고 기존 동작을 400으로 바꾸지 않는다.

실행 순서와 체크포인트(전 단계 미착수, 사람 승인 단계 없음):
1. 신규 통합 테스트를 추가하여 위 단독 명령으로 red를 확인한다. 환경/배선 실패를 제품 결함 red로 기록하지 않는다. 기대와 다르면 과제서를 갱신하고 다음 단계로 넘어가지 않는다.
2. providers.go의 첫 Decode 바로 다음에 EOF 검사만 추가한다. 단독 명령의 전체 PASS와 upstream 호출 횟수를 확인한 뒤 다음 단계로 진행한다.
3. docs/api.md 계약을 적고 관련 race·전체 Go 테스트·lint·build·diff 검사를 수행한다. 실패 시 원인을 적고 범위 내에서 해결하며 외부 후보를 추가하지 않는다.

접근 비교와 선정 근거: chat만 같은 Decoder로 EOF 확인하는 방법을 권고한다(프로덕션 1파일, 기존 입력 표현 유지). 전체 bytes 읽기 후 json.Unmarshal은 가능한 대안이지만 본문 버퍼링과 4 MiB 경계 처리 변경이 더 크다. 공용 decodeJSON 강화는 여러 API 계약에 영향을 줘 별도 회차로 보류한다. 현상 문서화만 하는 방법은 구현 부담은 작으나 잘못된 요청이 upstream 호출되는 문제를 남긴다. 권고의 핵심 전제는 요청 본문이 여러 JSON을 묶어 보내는 스트림 API가 아니라 단일 chat 요청이라는 것이며, 현재 docs/api.md 예제·map payload 처리와 일치한다.

작업량 근거와 예비 시간: bottom-up으로 DB/테스트 red 10분, 최소 수정·green 6분, 성공/경계/호출 횟수 단정 보완 8분, 문서 3분, 회귀 검증·정리 8분 = 기본 35분. 알려진 불확실성(DB 준비·큰 본문 HTTP 동작)에 대응하는 contingency 5분을 별도로 두며, 총 35~45분에 끝날 가능성을 중간 신뢰도로 판단한다(통계적 신뢰구간 아님). 과거 프로덕션 1파일+통합 테스트+문서 회차와 산출물 규모를 비교해 S가 타당하다고 교차 확인했으며, 과거 소요 시간은 기록이 없어 시간 비율을 꾸미지 않았다. 새 범위용 management reserve는 배정하지 않으며 45분을 넘길 조짐이면 범위를 늘리지 않고 원인/검증 상태를 남긴다.

증거: 같은 main@71dad05의 `go run ./cmd/ai-admin`을 전용 DB로 실행하여 실제 로그인/공급자 생성 후 HTTP 호출했다. `http-reproduction.json`에 정상·후행 공백 200/1회, 후행 객체/null/쓰레기 200/1회가 기록되어 있다. 정찰은 저장소 코드를 수정하지 않았다.
스킬 적용: 전용 Skill 도구는 노출되지 않아 설치된 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`와 `references/sources.md`, `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`를 직접 읽었다(뒤 두 경로의 공통 prefix는 같은 headcount/plugins). 추정 근거·범위/예비 시간 분리, 순차 실행/검증 체크포인트, 대안 비교를 반영했다. 외부 비용·편익 수치나 기관의 추정 보증을 사용하지 않았다.

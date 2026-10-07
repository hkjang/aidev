- 과제: 비스트리밍 chat의 upstream 연결 절단이 실패 응답·감사 기록으로 남는 계약을 통합 테스트로 고정 (가치 2 / 위험 1 / 작업량 S)
- 왜: `TestChatAnswerCutShortIsNotDeliveredAsComplete`는 `stream:true`만 검증하고, `TestRelayChatBodyReportsWhyTheAnswerStopped`의 비스트리밍 사례도 정상 완료뿐이라 실패의 최종 HTTP·감사 결과를 보호하지 못한다. `stream:false`로 실제 공급자 연결을 끊는 통합 테스트를 추가하면 반쪽 JSON이 정상 완료되거나 감사 성공으로 기록되는 회귀를 잡을 수 있다.
- 수용 기준:
  1) 정상 공급자 JSON을 `stream:false`로 요청하면 HTTP 200과 원문 전체를 오류 없이 받으며, 해당 요청의 `ai.chat` 감사가 정확히 1건이고 `result=success`, `details.complete=true`, `details.stream=false`, 실패 reason 없음이다.
  2) 공급자가 HTTP 200 + `application/json` 헤더와 JSON 일부를 전송한 뒤 연결을 강제로 끊으면, 호출자는 응답 본문을 정상 완료로 읽지 못한다. 충분한 본문을 먼저 내보내 헤더가 도착하는 사례로 구성하고 HTTP 200 이후 `io.ReadAll` 오류와 전달 바이트가 전송한 prefix에 속함을 확인한다. 뒤에 500 JSON을 덧붙이는 것은 허용하지 않는다.
  3) 절단 요청의 `ai.chat` 감사가 정확히 1건이고 `result=failure`, `details.reason=upstream_read_failed`, `details.complete=false`, `details.stream=false`다. complete/stream은 누락된 값을 false로 간주하지 말고 JSON boolean 키의 존재와 값까지 확인한다. resource_id는 생성한 공급자 ID와 일치하고 같은 요청의 success 행은 없어야 한다.
  4) DB Migrate/Seed → `New(db,cipher,logger).Handler()` → 실제 로그인·공급자 생성 API → 실제 HTTP `/v1/chat/completions` 경로를 통과한다. 기존 스트리밍 절단 테스트와 Content-Type 테스트를 보존하고 신규 테스트는 전용 DB에서 SKIP 아닌 PASS, `-race -count=3` 반복에도 PASS한다.
- 건드릴 파일: `internal/server/chat_truncation_integration_test.go:TestChatAnswerCutShortIsNotDeliveredAsComplete`의 셋업을 참고하여 같은 파일에 `TestChatNonStreamingTruncationAuditContract` 추가(정상·절단 두 서브테스트). **프로덕션 파일 0개, 테스트 파일 1개**. 기존 테스트의 대규모 공통 헬퍼 추출은 필요 없다.
- 검증 명령: 아래 실행 절차 참조. 정찰에서는 기존 관련 테스트를 실제 DB로 실행해 PASS를 확인했고, 아직 없는 신규 테스트의 성공은 미확인이다.
- 위험과 피할 것: `providers.go`, `server.go`, auth/session, migrations, workflow, 웹·내장 dist, VERSION·CHANGELOG 수정은 범위 밖이다. 비스트리밍의 모든 실패 사유를 한꺼번에 재현하지 않는다. 원래 동작이 맞는 테스트 보강 과제이므로 수정 전 테스트가 PASS해도 정상이며, 실패를 만들기 위해 프로덕션 코드를 바꾸지 않는다. 감사 details에 프롬프트·응답 원문을 넣지 않는다. 미머지 preferences와 verify-failed key-scope 과제를 재시도하지 않는다.
- 차선 후보: 프로필 email 빈 문자열의 NULL 삭제와 생략/null의 유지 계약을 통합 테스트·문서로 고정 (가치 1 / 위험 1 / S). 1순위 공백이 구현 시작 시 이미 해소된 경우에만 선택하며, 동작을 400으로 바꾸지 않는다. 현재 `users.go:updateProfile`와 `docs/api.md:97`, `profile_unknown_field_integration_test.go`가 근거다.

범위·근거와 구현 순서 (아래 단계는 모두 미착수; 사람이 지켜보지 않는 회차이므로 사람 승인 체크포인트 없음)

1. 기존 테스트를 실행해 셋업을 확인한다. `chat_truncation_integration_test.go:22-125`는 실제 upstream의 Flush/Hijack/Close, 실제 service.Client, signIn/session.do를 이미 사용한다. 헬퍼 정의는 `db_error_integration_test.go:100`의 signIn과 sessionCredentials.do다. chat 요청 자체는 ResponseRecorder로 하지 말고 httptest.NewServer의 URL로 보낸다. 체크포인트: 아래 기존 테스트 3개가 SKIP 없이 통과하면 다음 단계.
2. 같은 파일에 정상/절단 통합 테스트를 추가한다. `providers.go:chatCompletions`는 결과를 audit한 뒤 `panic(http.ErrAbortHandler)`로 중단하고, `relayChatBody`는 정상 EOF와 read error를 구별한다. 원래 스트리밍 샘플의 작은 SSE 문자열을 그대로 사용하면 비스트리밍 버퍼에 갇혀 Client.Do부터 EOF가 날 수 있으므로, 유효 JSON 전체(예: 128 KiB 이상의 문자열을 포함하는 JSON)를 만들고 그 중 64 KiB 이상 prefix만 보내는 fixture를 사용한다. upstream Content-Length에는 전체 길이를 쓰고 prefix를 Write/Flush한 뒤 Hijack/Close하여 본문 절단을 만든다. 정상 case는 전체를 보낸다. 응답 헤더만 보내고 종료하거나 짧은 JSON을 정상 EOF로 끝내는 것은 이 과제가 아니다.
   모든 chat 요청에 `stream:false`, `Content-Type: application/json`, `Accept-Encoding: identity`를 명시한다. 서버 Handler에 JSON 압축 middleware가 있어 반복 문자의 압축이 버퍼링을 다시 유발할 수 있다. 클라이언트 timeout은 유한하게 두고, 시간 경과 sleep 대신 실제 I/O 완료를 기준으로 판단한다. 요청마다 유효한 고유 `X-Request-Id`(ASCII, 80자 이하)를 보내 `action='ai.chat' AND request_id=$1 AND resource_id=$2`로 감사 행을 조회한다. 전역 최신 감사 행을 잡거나 앞선 정상 case의 행을 재사용하지 않는다. 체크포인트: 신규 테스트 단독 PASS 및 필수 boolean 키의 존재 단정 확인.
3. 관련 테스트를 race로 반복하고 전체 Go 회귀·lint·build를 실행한다. 같은 DB를 쓰는 테스트 프로세스는 순차 실행한다. 체크포인트: 모든 명령 exit 0 및 새 통합 테스트 PASS 확인 후 구현 완료. 새 테스트가 실제 프로덕션 결함을 드러내면 증거를 기록하고 과제서를 수정하여 범위 재평가하며, 몰래 런타임 변경으로 확대하지 않는다.

실제 검증 명령 (저장소 루트, Docker 사용 가능할 때)

```bash
docker run --rm -d --name ai-admin-chat-audit-20261007 -e POSTGRES_USER=ai_admin -e POSTGRES_PASSWORD=test-password -e POSTGRES_DB=ai_admin_test -p 127.0.0.1:55551:5432 postgres:16-alpine
# pg_isready가 accepting connections인지 확인 후 진행. 포트 충돌 시 새 포트와 DSN을 함께 변경한다.
docker exec ai-admin-chat-audit-20261007 pg_isready -U ai_admin -d ai_admin_test
export TEST_POSTGRES_DSN='postgres://ai_admin:test-password@127.0.0.1:55551/ai_admin_test?sslmode=disable'
go test -count=1 -run 'Test(ChatAnswerCutShortIsNotDeliveredAsComplete|ChatNonStreamingResponseContentTypeContract|RelayChatBodyReportsWhyTheAnswerStopped)$' -v ./internal/server
go test -race -count=3 -run 'Test(ChatNonStreamingTruncationAuditContract|ChatAnswerCutShortIsNotDeliveredAsComplete|ChatNonStreamingResponseContentTypeContract|RelayChatBodyReportsWhyTheAnswerStopped)$' -v ./internal/server
go test -race -count=1 ./...
make lint
go build ./...
docker stop ai-admin-chat-audit-20261007
```

테스트는 두 schema를 DROP하므로 전용 폐기 DB만 사용한다. 실패해도 자신이 만든 컨테이너를 정리한다. `TEST_KEYCLOAK_ISSUER` 없는 로컬에서는 실제 OIDC E2E가 SKIP되며 CI는 Keycloak을 띄운다. 앱 코드 변경이 없어 npm/build 산출물 갱신은 필요 없다.

정찰 확인 결과: main@71dad05, VERSION 1.2.35, 작업 트리 깨끗함. postgres:16-alpine 전용 컨테이너(55551)에서 첫 Go 검증 명령 exit 0, 서버 패키지 1.275초, 대상 최상위 테스트 3개 모두 PASS(SKIP 없음). 컨테이너는 정리했다. 신규 비스트리밍 대형 prefix fixture·race 반복·전체 Go·lint/build는 아직 미실행이다. 파일명 검색상 저장소에 CLAUDE.md/AGENTS.md/독립 roadmap 없음, TODO/FIXME/roadmap 검색도 README·docs·internal·web/src에서 결과 없음.

대안 비교·견적 근거 (요청한 세 스킬 적용)
- 최소안인 relayChatBody 단위 사례 추가는 싸지만 실제 응답 중단과 감사 배선을 증명하지 못해 선택하지 않았다. 선택안은 기존 HTTP/DB fixture를 빌려 테스트 1파일로 최종 동작을 확인한다. 네 실패 사유 전체 재현은 TCP write/cancel/timeout의 경합을 다뤄야 하므로 후속으로 남긴다. 아무것도 하지 않으면 비스트리밍 통합 공백이 남는다.
- 가장 큰 가정: 현재 비스트리밍 런타임이 의도대로 작동하고, 대형 prefix+identity 설정으로 실제 본문 read error를 안정적으로 관찰할 수 있다는 것(신규 fixture 실행은 미확인).
- Bottom-up 예상: 기존 셋업 확인 4~5분, 정상/절단 fixture와 감사 단정 12~16분, 반복·전체 검증과 정리 7~9분 = 기본 23~30분. 알려진 불확실성인 HTTP 버퍼링·fixture 안정화에 별도 contingency 5~10분을 두어 총 28~40분, 45분 안 완료 확신은 중간 수준의 정성 판단이며 통계적 보장은 아니다. 미지의 추가 scope에 대한 management reserve는 배정하지 않는다(확대 대신 분리).
- 비교 근거: 9/27 Content-Type 통합 테스트가 이미 같은 실제 HTTP/DB 구조를 쓰며, 이번 기준 테스트는 1.275초였다. 과거 작업자의 실제 코딩 소요 시간은 제공되지 않아 analogous 방법의 독립 숫자 추정은 미확인이고, 근거 없는 속도 배율은 쓰지 않았다. 첫 신규 통합 실행 결과가 예상과 다르면 남은 작업량을 다시 적는다.
- 적용 스킬: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md`. 전용 Skill 도구가 없어 설치된 파일을 직접 읽었다. pmo의 references/sources.md도 읽었으며 외부 비용·편익 수치나 통계적 신뢰수준을 인용하지 않고 저장소 관찰에 기반해 추정했다.

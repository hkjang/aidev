## 2026-10-09
- 선택: AI 제공자 SSE의 프레임 경계·다중 data 줄·EOF 처리를 실제 HTTP 회귀 시험으로 고정 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: internal/server/ai_test.go에 TestProxyAIStreamFraming을 추가해 9개 fixture를 LF/CRLF로 각각 실행하며 다중 data 결합, 연속 delta의 순서/중복, 이벤트 이름 초기화, 마지막 개행 없는 EOF flush, 주석/빈 줄/잘못된 JSON/DONE 무시와 다음 정상 프레임, DONE 단독의 빈 출력을 고정했다. 실제 httptest 제공자 → proxyAIStream 내부 HTTP client/transport → scanner → extractDelta → sendSSE → ResponseRecorder 경로에서 모든 반환 오류가 nil이고 최종 SSE 전체가 고정 기대 문자열과 같은지 확인했으며, 기존 정상화 시험 유지·프로덕션 0파일·시험 1파일(92줄 추가)로 커밋 8ac45f8을 만들었다. 지정 특정 시험 명령에서 신규 18개 하위 시험과 기존 정상화가 PASS(1.049s), 기존 AI 가드/UTF-8 시험 PASS, go test -race -count=1 ./... PASS(config/secure/server/scripts), go vet ./...·go build ./...·gofmt -l internal/server/ai_test.go·git diff --check 종료 0이며 DB 시험은 ORBIT_TEST_DATABASE_URL 미설정으로 SKIP(실 DB·프런트·전체 인증/설정 라우터는 미검증).
- 실패 재현: 못 함 — 기존 동작을 고정하는 시험 공백 보강 과제이므로 새 시험은 프로덕션 수정 없이 첫 실행부터 PASS했다(1단계 4개 하위 시험 PASS, 확장 후 18개 PASS). 재현된 결함이나 수정한 원인은 없으며, 실패를 만들려고 프로덕션 코드나 기대 출력을 인위적으로 바꾸지 않았다.
- 보류 아이디어: AI 제공자 비-2xx 및 scanner 길이 초과 오류의 DB 없는 HTTP 회귀 시험 (가치 2 / 위험 1 / 작업량 S) — 차선 유지, 이번 프레임 과제를 그대로 수행했으므로 전환하지 않음.
- 보류 아이디어: AI 제공자 endpoint 정규화와 요청 헤더·본문 전달 계약 회귀 시험 (가치 2 / 위험 1 / 작업량 S) — 별도 HTTP 입력 계약으로 보류.
- 보류 아이디어: orbit_list_memories의 DB limit 적용 (가치 2 / 위험 2 / 작업량 S) — 성능 실측 후 호출자 범위에서만 검토.
- 보류 아이디어: decodeJSON의 뒤따르는 JSON·쓰레기 거부 (가치 2 / 위험 2 / 작업량 S) — 전체 입력 수용 계약 변경이라 별도 검토.
- 과제서: 채택 — 현재 parser와 기존 정상 단일 프레임 시험이 과제서의 근거와 일치했고 수용 기준 전부를 프로덕션 수정 없이 실제 HTTP 실행으로 확인했다.

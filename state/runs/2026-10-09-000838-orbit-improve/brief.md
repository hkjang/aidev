- 과제: AI 제공자 SSE의 프레임 경계·다중 data 줄·EOF 처리를 실제 HTTP 회귀 시험으로 고정 (가치 2 / 위험 1 / 작업량 S)
- 왜: `internal/server/server_test.go:TestResponsesStreamNormalization`은 빈 줄로 끝나는 정상 프레임 하나만 확인해, `proxyAIStream`의 여러 data 줄 결합·이벤트 초기화·EOF flush가 퇴행해도 잡지 못한다. 실제 HTTP 제공자에서 경계 입력을 보내고 최종 delta SSE 전체를 비교하면 사용자가 받는 AI 문장의 누락·중복·오염을 DB 없이 검출할 수 있다.
- 수용 기준:
  1) `httptest.NewServer`가 여러 data 줄로 나눈 유효 JSON을 보내면 delta 하나가 정확히 나온다. 예: `event: response.output_text.delta\ndata: {"delta":\ndata: "안녕"}\n\n` → `event: delta\ndata: {"text":"안녕"}\n\n`. JSON 문자열 내부를 실제 개행으로 쪼개는 잘못된 fixture는 쓰지 않는다.
  2) 서로 다른 두 delta 프레임을 이어 보내면 순서대로 정확히 두 번 출력된다. LF 및 CRLF 입력을 모두 검사한다. 빈 줄에서 이벤트 이름이 초기화되는지 확인하려면 이름 있는 delta 프레임 뒤에 event/type 없는 `data: {"delta":"나오면 안 됨"}` 프레임을 둔다. 뒤 프레임은 출력되지 않아야 한다.
  3) 마지막 data 줄 뒤에 줄바꿈도 빈 줄도 없는 유효 프레임이 EOF에서 정확히 한 번 출력된다. 이는 지금 Orbit의 EOF 동작을 고정하는 것이며 SSE 표준 준수 여부를 새로 판정하는 작업이 아니다.
  4) 주석(`: ping`)·추가 빈 줄·`data: [DONE]`·잘못된 JSON 프레임은 delta를 만들지 않고, 다음 유효 프레임은 정상 출력한다. 마지막 `[DONE]`만 보낸 경우에도 출력은 비어 있고 반환 오류가 없다. `[DONE]`을 받은 뒤 즉시 연결을 종료한다는 새로운 계약은 만들지 않는다.
  5) `(&Server{}).proxyAIStream`을 실제 호출하여 내부에서 생성하는 HTTP client/transport → 제공자 응답 → scanner → extractDelta → sendSSE → `httptest.ResponseRecorder.Body` 경로를 통과한다. 각 정상 사례의 반환 오류가 nil이고 최종 본문이 기대 바이트열과 정확히 같아야 한다. contains 검사만으로 중복·추가 이벤트를 놓치지 않는다. DB 없이 새 시험이 실제 실행되며 기존 정상화·AI 가드·UTF-8 시험도 통과한다.
- 건드릴 파일: `internal/server/ai_test.go:신규 TestProxyAIStreamFraming` — 테이블 기반 경계 사례와 필요 최소 시험 전용 헬퍼 추가. 프로덕션 변경 0파일, 테스트 변경 1파일. 읽기 참고: `internal/server/ai.go:proxyAIStream/extractDelta/sendSSE`, `internal/server/server_test.go:TestResponsesStreamNormalization`, `internal/server/ai_db_test.go:TestStreamAIPersonLookup/TestStreamAIProviderFailureIsLogged`. 기존 정상화 시험은 그대로 둔다.
- 검증 명령:
  - 저장소 루트: `go test -race -count=1 -v ./internal/server -run '^(TestResponsesStreamNormalization|TestProxyAIStreamFraming)$'` — 새 이름의 RUN/PASS가 나오는지 확인한다. 현재 기준에는 새 함수가 없으므로 기존 시험만 실행된다.
  - 저장소 루트: `go test -race -count=1 ./...`
  - 저장소 루트: `gofmt -l internal/server/ai_test.go` 및 `git diff --check`
  - 정찰 실측: `go test -race -count=1 ./...` 종료 0; config/secure/server/scripts PASS. ORBIT_TEST_DATABASE_URL 미설정이며 DB 시험은 openTestStore의 Skip 경로다. 새 경계 fixture 실행 및 위 신규 이름 대상 시험은 구현 후 검증 대상(정찰 미확인). 프런트 설치·빌드·실 DB 시험은 실행하지 않았다.
- 위험과 피할 것: parser 추출·공통화·의존성 추가·HTTP transport 주입·프로덕션 코드 수정은 범위 밖이다. auth/session·migrations·.github/workflows·승인 흐름·공유 SQL·safeAIError·slog/감사 기록을 건드리지 않는다. 직접 부르는 함수는 streamAI가 실제 사용하는 프록시이며, 전체 인증/설정 라우터까지 검증했다고 주장하지 않는다. nil store 패닉을 성공 근거로 삼지 않는다. 제공자 handler의 고루틴 안에서는 t.Fatal/Fatalf 대신 t.Error/Errorf와 return을 쓴다. sleep·실 외부 API·실 자격증명 없이 로컬 httptest 서버를 정리한다. 공급자 오류 이벤트의 의미·브라우저 parser·HTTP chunk 도착 시각은 별도 과제다. 범위 밖 결함을 발견하면 과제서에 기록하고 조용히 고치지 않는다.
- 차선 후보: `proxyAIStream`의 비-2xx 및 4MiB scanner 한도 초과 시 오류 반환 회귀 시험 — 현재 프레임 경계 시험이 이미 별도 변경으로 충분히 보강된 사실을 구현자가 확인할 때만 전환. 실제 httptest 서버의 500/429 응답 및 4MiB보다 큰 단일 data 줄로 nil 아닌 오류와 delta 없음 확인; 프로덕션 변경 없이 ai_test.go 한 파일에 한정한다. 로그 원문 정책 변경이나 provider 오류 이벤트 해석은 포함하지 않는다.

선택 근거와 대안 검토

- 최소 선택(채택): 기존 동작의 HTTP 경계 시험만 보강한다. 실제 parser 상태가 이어지는 입력과 최종 출력 비교에 가치가 있고, 공유 입력 계약·DB·권한을 바꾸지 않는다.
- 별도 SSE parser 모듈/라이브러리로 교체: 재사용성은 생기지만 현재 버그 실측 없이 의존성과 계약을 바꾸므로 제외한다.
- DB를 포함한 streamAI 통합 시험으로 모든 경계를 검증: 설정/인증 연계까지 다루지만 이미 ai_db_test.go가 정상·실패 SSE를 검사한다. 새 경계 시험까지 DB에 묶으면 CI에서 Skip되므로 이번에는 제외한다.
- 현상 유지: 현재 확인된 사용자 장애는 없으므로 가능한 선택이다. 다만 수동 parser의 경계 공백을 한 테스트 파일에서 닫을 수 있어 이번 작은 과제로 시험 보강을 선택한다.
- MCP limit 개선은 실측 없는 성능 변경, decodeJSON 강화는 모든 요청의 수용 계약 변경, approvalID 가드는 권한 경로라 우선순위가 낮다. 내보내기 문서 작업은 앞 회차 성공 기록이 있지만 pinned 5c97a1a에는 아직 없으므로 중복 추진하지 않는다.
- 핵심 가정: 현재 프레임 처리 동작을 보존하는 회귀 시험이 목적이다. 새 사례가 예상과 달리 실패하면 표준 해석을 임의로 바꿔 구현하지 말고 원인을 기록해 과제서를 갱신한다.

실행 순서·검토 지점 (구현 상태: 1~3 완료 — 아래 구현 검증 기록 참조)

1. `ai_test.go`에 TestProxyAIStreamFraming을 추가하고 다중 data 줄/연속 프레임/CRLF를 먼저 작성한다. 증명: 위 특정 시험 명령. 체크포인트: 구현자 자동 검토, 사람 승인 없음; 모든 기대 출력이 고정 fixture이고 프로덕션 parser를 복제하지 않았는지 본다.
2. 같은 시험에 이벤트 초기화/EOF/무시할 프레임 사례를 추가한다. 증명: 같은 특정 시험 명령에서 각 하위 시험 PASS와 최종 출력의 정확한 비교. 체크포인트: 자동 검토; 예상과 다르면 현재 사실과 계획을 먼저 맞춘다.
3. 전체 Go race 시험·gofmt 조회·diff 검사를 수행한다. 체크포인트: diff가 시험 파일 한 개뿐인지 확인하고 실제 실행 결과/Skip 한계를 인계한다. 각 단계 완료는 증명 명령을 실행한 뒤 이 문서에 표시한다.

작업량 근거·예비 시간

- 추정 방식: 현재 코드와 기존 HTTP 시험을 근거로 한 bottom-up 판단. fixture/서버 구성 5–7분, 경계 사례와 정확한 출력 단언 12–16분, 전체 검증·diff 검토 5–7분: 기본 22–30분.
- 알려진 불확실성 예비(contingency): JSON 개행 fixture 오류 또는 고루틴 시험 실패 전달 진단에 3–10분을 별도 확보한다. 합계 25–40분, 확신 중간(통계적 신뢰구간이 아닌 정찰 판단). 이번 전체 Go 시험은 수초에 완료되어 DB·npm 설치 시간을 포함하지 않는다.
- 관리 예비(management reserve): 이 과제에 할당하지 않음. 새 프로덕션 결함·외부 표준 변경은 45분에 억지로 넣지 않고 별도 후보로 남긴다. 실제 회차별 작업시간 데이터가 없어 유사 사례 기반 수치 교차 검증은 미확인이다.
- 범위·가정·분해·불확실성을 기록하고 실측으로 갱신하는 방식은 스킬이 지정한 [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)의 추정 절차를 참고했다. 위 분 단위 수치는 GAO 수치가 아니라 이 저장소에 대한 판단이다.
- 적용 스킬 원문: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md`. Skill 전용 도구는 미제공되어 로컬 원문을 읽었다.


## 구현 검증 기록 (2026-10-09)
- [x] 1단계: 다중 data·연속 프레임을 LF/CRLF로 작성 후 `go test -race -count=1 -v ./internal/server -run '^(TestResponsesStreamNormalization|TestProxyAIStreamFraming)$'` 실행. 신규 4개 하위 시험 및 기존 정상화 시험 PASS, `ok github.com/hkjang/orbit/internal/server 1.024s`, 종료 0.
- [x] 2단계: 이벤트 초기화·EOF·주석/빈 줄·DONE 후 정상·잘못된 JSON 후 정상·DONE 단독(빈 줄/EOF)을 추가하고 같은 명령 실행. `=== RUN TestProxyAIStreamFraming` 및 LF/CRLF 각 9개 하위 시험 모두 PASS, 기존 정상화 PASS, `ok github.com/hkjang/orbit/internal/server 1.049s`, 종료 0.
- [x] 3단계: `go test -race -count=1 ./...` 종료 0(config 1.010s, secure 1.010s, server 1.081s, scripts 1.039s). `go vet ./...`, `go build ./...`, `gofmt -l internal/server/ai_test.go`, `git diff --check` 모두 종료 0, 무출력. diff는 internal/server/ai_test.go 한 파일 92줄 추가뿐이다.
- 기존 AI 가드·UTF-8 및 DB Skip 확인: `go test -race -count=1 -v ./internal/server -run '^(TestStreamAIRejectsMalformedPersonIDBeforeDB|TestStreamAILetsWellFormedPersonIDReachDB|TestSafeAIError|TestStreamAIPersonLookup|TestStreamAIProviderFailureIsLogged)$'` 종료 0, 앞 세 시험 PASS. 뒤 두 DB 시험은 ORBIT_TEST_DATABASE_URL 미설정 메시지와 SKIP를 직접 확인했다.
- 자동 검토: 기대 본문은 고정 문자열이며 parser 복제나 transport 주입 없음. provider handler는 Errorf+return만 사용, 서버는 defer Close. 반환 오류와 전체 본문을 검사하고 기존 정상화 시험은 유지했다.
- 실패 재현: 못 함 — 원래 정상 동작의 시험 공백을 보강하는 과제이며 새 시험은 첫 실행부터 PASS. 프로덕션 코드를 수정하지 않았고 결함을 인위적으로 삽입하지 않았다. systematic-debugging의 결함 원인/수정/되돌림 검증은 해당 없음; 재현된 실패가 없다.
- 검증 한계: 실 DB·프런트 설치/시험/빌드·전체 인증/설정 라우터·브라우저 parser·HTTP chunk 시점은 미검증. EOF 및 DONE은 현재 Orbit 동작만 고정한다. 정찰 추정 25–40분은 참고값이며 단계별 작업시간은 별도로 계측하지 않았다.
- 구현 적용 스킬: technology:completion-verification, technology:systematic-debugging, technology:test-driven-development. Skill 전용 도구가 없어 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills 아래 각 SKILL.md 원문을 읽었다. 실패 선행을 재현하지 못한 사실은 사용자 절차의 예외 보고 지시에 따라 명시한다.

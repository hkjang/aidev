- 과제: notify의 Telegram·Slack 전송 실패 로그에서 자격증명 노출 차단 (가치 4 / 위험 1 / 작업량 S)
- 왜: internal/notify/notify.go의 (*Notifier).send는 HTTP 전송 오류를 slog에 원문 그대로 넘겨 Telegram 토큰이 들어간 /bot<token>/sendMessage URL과 SlackWebhook 전체 URL을 로그에 남긴다. 두 경로에서 로그 기록 전에 비밀을 제거하면 연결 장애 때 알림 자격증명이 운영 로그로 유출되는 것을 막으면서 실패 진단을 유지할 수 있다.
- 수용 기준:
  1) Telegram 연결 실패 로그에는 TelegramToken 원문이 없고, Slack 연결 실패 로그에는 SlackWebhook 원문 및 비밀 경로가 없다. 플랫폼과 전송 실패 사실은 식별 가능해야 하며 요청 본문·chat ID·원문 오류를 별도 속성으로 추가하지 않는다.
  2) 동일 Notifier에 두 플랫폼을 설정했을 때도 두 오류 로그 모두 안전하다. 로그용 문자열만 정리하고 실제 HTTP 요청 URL·폼/JSON·Slack 굵게 변환·Send의 비동기/best-effort 동작·client()의 sync.Once 초기화는 보존한다.
  3) 실제 httptest.Server와 http.Client를 사용한다. 긴 가짜 Telegram 토큰과 /services/TEAM/CHANNEL/UNIQUE_SECRET 형태의 로컬 Slack URL을 설정하고 서버를 닫아 실제 연결 실패를 유발한다. 공개 Send → 실제 고루틴 → send → HTTP 오류 → slog 기록을 지나 두 플랫폼의 실패 로그가 실제로 존재하며 토큰/Slack 비밀 경로가 없는지 증명한다(로그가 비어서 통과하면 안 됨). 고정 sleep 대신 완료 관측 가능한 thread-safe slog.Handler/채널과 제한시간으로 두 로그를 기다린다. 글로벌 slog 기본값은 cleanup에서 복원하고 이 테스트는 병렬화하지 않는다.
  4) 기존 TestSendTelegram/TestSendSlack/TestStructLiteralDoesNotPanic/TestSendFailuresAreSwallowed가 계속 통과한다. 새 로그 테스트가 수정 전 비밀 노출 때문에 실패하고 수정 후 통과함을 확인하고 notify -race 및 전체 검증을 통과한다.
- 건드릴 파일: internal/notify/notify.go:(*Notifier).send — Telegram/Slack의 두 전송 오류 분기에서 slog에 전달하기 전에 자격증명을 제거(필요하면 작은 비공개 로그 정리 함수 추가); internal/notify/notify_test.go — 실제 HTTP 연결 실패 및 공개 Send 경로의 로그 회귀 테스트 추가. internal/copilot/telegram.go:(*Bot).call의 strings.ReplaceAll 마스킹은 읽기 참고만 하며 수정하지 않는다.
- 검증 명령: 저장소 루트에서 `go test -count=1 -run TestSendFailuresAreSwallowed -v ./internal/notify`(현재 노출 재현); 구현 후 `go test -count=1 -race ./internal/notify`, `gofmt -l internal/notify/notify.go internal/notify/notify_test.go`, `go vet ./...`, `go build ./...`, `go test -count=1 ./...`.
- 위험과 피할 것: Telegram만 고치고 Slack의 같은 원문 오류 경로를 남기지 말 것. 가장 작은 해결은 로그 문자열에서 비어 있지 않은 설정 비밀을 치환하는 것이며, 빈 문자열 치환·정규식 과잉 마스킹·안전하지 않은 fallback으로 원문 오류를 다시 반환하는 구현을 피한다. Slack은 URL 자체가 자격증명이므로 전체 URL을 정리하고 테스트에서는 고유 비밀 경로도 검사한다. 공통 로깅 프레임워크 도입, 재시도/메시지 분할/Slack HTTP 상태 처리 확장, copilot·toss 인증/토큰 캐시·broker 주문·workflows 변경은 범위 밖이다. 실제 자격증명/.env/외부 API를 사용하지 말고 zzdbg 태그 테스트를 실행하지 않는다. 이전 notify nil 방어·첫 테스트 작업을 재수행하는 과제가 아니다.
- 차선 후보: scripts/check.sh 로컬 Go 검증 진입점 — 첫 과제가 이미 동등하게 해결되어 새 회귀 테스트가 수정 전에도 통과할 때만 선택. 새 scripts/check.sh에서 루트 기준 gofmt 검사(자동 수정 금지) → go vet ./... → go build ./... → go test -count=1 -race ./...를 실패 즉시 중단하고 README 빌드 절에 실행법을 적는다. 기존 scripts/check_dashboard.sh는 브라우저 의존 별도 검사로 유지하며 workflow·운영 스크립트는 수정하지 않는다.

근거 및 구현 순서:
- 정찰 기준 main@e09d8bdb, 작업 트리 깨끗함. notify.go의 두 slog.Warn(..., "err", err) 분기를 직접 읽었고 기존 TestSendFailuresAreSwallowed의 닫힌 서버 재현에서 `telegram 전송 실패 err="Post .../botT/sendMessage..."`, `slack 전송 실패 err="Post http://127.0.0.1:<port>..."`가 출력됨. T는 기존 테스트의 가짜 토큰이다. 실제 운영 로그 유출 이력은 미확인이고 Slack 비밀 경로를 붙인 회귀 재현은 구현자가 추가해야 한다.
- copilot/telegram.go:call은 이미 자기 토큰을 마스킹하지만 notify는 별도 전송 경로다. 이번에는 notify의 두 플랫폼 경로를 함께 다룬다.
- 대안 비교: scripts/check.sh(3/1/S)는 유용하지만 확인된 비밀 노출보다 가치가 낮다. toss 재시도 테스트(3/2/S)는 인증 보호 구역, swing 청산/보유 한도(3/3/M)는 매매 의미 변경이라 제외했다.
- 예상 30분 + 예비 15분: 실제 Send 로그 회귀 테스트/Red 12분 → 두 오류 분기 최소 수정 8분 → race·전체 검증 10분 → 비동기 로그 동기화/실패 분석 예비 15분. 시간이 밀려도 두 플랫폼 중 하나를 누락하거나 보안 단언을 완화하지 않는다.
- 정찰 검증: go1.26.7 linux/amd64; gofmt -l . 출력 없음; go vet ./..., go build ./..., go test -count=1 ./..., go test -count=1 -race ./internal/notify 통과. broker 7.713s, engine 2.007s; 전체 벽시계 시간은 미계측.
- 요청 스킬 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration는 사용 가능한 도구 목록에 Skill/skills.list/skills.read가 없고 /home/hkjang/.codex, /home/hkjang/.claude, /mnt/c/Users/USER/projects/aidev의 SKILL.md 검색에서도 찾지 못함. 고유 절차·반환 형식은 미확인/미적용이며 위 추정·대안 비교·구현 순서는 사용자 프롬프트에 따른 독립 작성이다.

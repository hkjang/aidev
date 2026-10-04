# 회차 노트 2026-10-04-190215-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:02] base pinned — main@3c54ccf
- [러너 19:02] autonomy release — 

## 정찰 노트
- 1순위로 고른 것: `streamAI`(ai.go:86~90)가 `proxyAIStream` 의 err 를 버려 AI 스트림 실패가 사용자에게도 운영자 로그에도 아무 원인을 남기지 않는 것. respond.go:47·export.go:85 의 slog.Error 관례가 이 경로만 비어 있어 결함 성격이 분명하고 프로덕션 1파일로 끝난다.
- 보류 1순위였던 safeAIError 룬 버그는 단독 과제로 기각 수준이다 — `grep -rn safeAIError --include=*.go .` 가 정의 한 줄만 돌려준다(호출자 0, 죽은 코드). 그래서 로그 배선 과제에 흡수했다: 배선되는 순간 실제 버그가 된다.
- SSE 파서 시험(12회 연속 차선)은 rejected 로 내렸다. ai.go:243~283 을 실제로 읽어 보니 네 동작이 다 있고 결함이 없어 순수 공백 보강이다.
- 과제서에서 추측으로 남긴 것: (1) `slog.SetDefault` + JSONHandler 로 레코드를 읽는 캡처 방식을 실행해 확인하지 못했다(대안도 과제서에 적었다). (2) `httptest.ResponseRecorder` 가 http.Flusher 를 만족한다는 것은 기존 200 SSE 시험이 통과하는 것으로 간접 확인했을 뿐이다.
- 구현자가 조심할 것: 수용 기준 2(클라이언트 취소는 ERROR 로 남기지 않기)를 빼먹으면 사용자가 탭을 닫을 때마다 오탐 ERROR 가 쌓인다. 그리고 `s.audit`(ai.go:94)에는 제공자 오류 원문을 절대 넣지 말 것 — slog 한 줄까지가 범위다.
- [러너 19:07] scout done — AI 스트림이 제공자 오류로 실패할 때 원인이 서버 로그에 한 줄도 남지 않는 것을 고치고, 그 용도로 쓰려

## 구현 노트
- 무엇/왜: `ai.go` 의 `streamAI` 오류 분기에 `slog.Error("AI 스트림이 중간에 끊겼습니다", "user", u.ID, "model", settings.Model, "error", safeAIError(err))` 한 줄을 더해 제공자 실패 원인이 운영자 로그에 남게 했고, 그 과정에서 호출자가 0이던 `safeAIError` 의 `message[:300]` 룬 쪼개짐을 `utf8.RuneStart` 되돌림으로 닫았다. 프로덕션 1파일.
- 사용자 응답 바이트열은 일부러 건드리지 않았다 — `sendSSE`+`Flush` 를 먼저 보내고 그 뒤에 로그를 남기므로, 취소든 제공자 오류든 200/`meta`→`error`/`done` 없음이 모든 경우에 똑같다. 새 시험이 두 경로의 기대 바이트열을 둘 다 고정한다.
- 확신 없는 곳: (1) 취소 하위 시험은 제공자 핸들러가 호출 중에 요청 컨텍스트를 끊고 응답을 쓰지 않은 채 붙잡는 방식이다. `client.Do` 가 취소를 먼저 보는 것에 의존하므로 이론상 경합 여지가 있다 — 그래서 `event: done` 이 나오지 않는 것을 먼저 확인해 오류 분기 도달을 못박았다(현재 -race 로 반복 통과). (2) 제공자 쪽 `r.Context().Done()` 은 클라이언트 취소 후에도 뜨지 않았다(처음 구현에서 5초 상한이 그대로 소진됐다) — 그래서 `release` 채널로 핸들러를 풀어 주고 상한은 보험으로만 남겼다. 이유는 끝까지 파지 않았다.
- 일부러 하지 않은 것: `s.audit`(ai.go:95)에 원인을 넣지 않았다(감사 출력이 제공자 응답 원문을 되내보낸다). `proxyAIStream`·`extractDelta`·`sendSSE`·SQL·사용자 문구·상태코드는 손대지 않았다. 새 오류 코드도 없다. `ai.go:243` 이 제공자 본문 8KiB 를 오류에 담는 것(이제 slog 로 나간다)의 스크럽은 실제 사례를 확보할 때까지 ideas.json 에 보류로 남겼다.
- 다음 역할이 조심할 것: `TestStreamAIProviderFailureIsLogged` 는 `ORBIT_TEST_DATABASE_URL` 이 있어야 돈다(없으면 SKIP). `TestSafeAIError` 는 DB 없이 CI 에서 실제로 돈다. 두 시험 모두 전역 상태(AI 설정 행, `slog` 기본 로거)를 건드려 `t.Parallel()` 금지 — 로거는 `captureLogs` 가 `t.Cleanup` 으로 원복한다. 검증은 격리 postgres 포트 55659 로 했고 컨테이너는 정리했다.
- 돌린 것: `gofmt -l .`(무출력) · `go vet ./...` · `go build ./...` · `go test -race -count=1 ./...` DSN 있음/없음 양쪽 초록. 인과는 `slog.Error` 한 줄 제거 / 룬 되돌림 제거를 각각 따로 해서 빨강→복원→초록으로 확인했다. 웹(`npm ci`/vitest/vite build)은 돌리지 않았다 — 이번 변경에 web 파일이 0개다.
- [러너 19:15] brief accepted — 채택 — 지정한 자리(`ai.go:86~90` 의 버려진 err, `ai.go:306~312` 의 `message[:300]`, `log/slog` import 없음, `safeAIError` 호출자 0, `preserve
- [러너 19:15] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 인과를 직접 재현했다: 복제본에서 프로덕션 `ai.go` 만 main 으로 되돌려 두 시험이 원장과 글자까지 같은 출력으로 실패하는 것을 봤고, 구현자가 확인하지 못한 고리(ai.go:99 취소 가드)도 그 줄만 지워 `클라이언트_취소는…` 가 ai_db_test.go:324 에서 실패하는 것을 확인했다 — 세 하위 시험에 항상 참인 것이 없다.
- 구현자가 '이론상 경합' 으로 남긴 취소 하위 시험은 격리 postgres(포트 55707)에서 `-race -count=5` 5/5 통과, 전체 `go test -race -count=1 ./...` DSN 주고 초록, gofmt/vet/build 통과. 컨테이너·복제본 정리했다. 취소 판정이 respond.go:44·auth.go:335 와 같다는 주석도 사실로 확인(`errors.Is(err,nil)` 은 Go nil 가드로 false).
- **릴리즈 노트가 알아야 할 것**: 이번에 고친 분기를 지키는 `TestStreamAIProviderFailureIsLogged` 는 CI 에 postgres 가 없어 머지 후 항상 SKIP 된다. CI 에서 실제로 도는 것은 `TestSafeAIError` 뿐이다.
- **다음 회차용 우려(차단 아님)**: 새 slog 줄이 ai.go:259 의 제공자 본문 8KiB 를 300바이트까지 운영자 로그로 보낸다 — 입력을 되읊는 제공자라면 사람 이름·기억 본문이 섞인다. 자체 호스팅 로컬 로그 + 상한 + internalError 관례와 같은 등급 + `s.audit` 는 깨끗해서 notes 로 내렸다. 본문 스크럽은 ideas.json 보류 항목이고, 다중 운영자/호스팅 배포가 생기면 보류가 아니라 과제다. `/ai/stream`(server.go:62)은 레이트 리밋이 없어 실패 요청마다 ERROR 한 줄이 쌓인다(기존 성질, 로그 분량만).
- 보지 않은 것: web/(변경 0개라 npm·vitest·vite 미실행), 제공자가 실제로 input 을 오류 본문에 되읊는지(실사례 미확보 — 위 우려의 전제).
- [러너 19:21] review approved — 리뷰 승인 (risk=low)
- [러너 19:21] pr created — https://github.com/hkjang/orbit/pull/19
- [러너 19:23] ci passed — 검사 1개 모두 success
- [러너 19:23] merge done — 5926ade
- [러너 19:29] release published — v0.7.9
- [러너 19:30] assets verified — v0.7.9 자산 1개 (이전 v0.7.8: 1)

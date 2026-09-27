# 회차 노트 2026-09-27-082200-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:22] base pinned — main@71adc90
- [러너 08:22] autonomy release — 

## 정찰 노트
- 네 회차 연속 "미확인"으로 이월되던 chat Content-Type 항목을 이번에 `providers.go:646-661`에서 직접 판정했고 실재하는 계약 불일치였다 — 이 저장소의 지배적 결함 유형(같은 핸들러 두 경로의 계약 불일치)과 정확히 같은 모양이라 1순위로 골랐다. providers.go의 `available_models` JSON 무시(M, 경로 여럿)와 workflow.go `errKeyStale`(직전 회차가 같은 함수를 건드림)은 제쳤고, 문서 한 문장짜리 감사 CSV 수정은 가치가 낮아 차선으로 남겼다.
- 추측으로 적은 것: 공급자 `streamDefault` 기본값(그래서 요청에 `"stream": false` 명시를 지시했다), `web/src`가 chat 응답 Content-Type을 읽는지 여부. 둘 다 과제서에 "미확인"으로 표시했다.
- 구현자가 조심할 것: 새 거부 분기는 반드시 `w.WriteHeader`(`:662`) 앞에 둘 것. 허용 집합을 `application/json`+파라미터+`+json`+빈 값보다 좁히면 멀쩡한 공급자가 502가 된다. 스트리밍 분기와 `relayChatBody`는 별개 계약이니 함께 손대지 말 것.
- 테스트·린트·빌드는 이번 정찰에서 실행하지 않았다(읽기만). 프로필은 VERSION 1.2.28/base 71adc90으로 갱신했다.
- [러너 08:26] scout done — 비스트리밍 chat 프록시가 upstream `Content-Type`을 검증 없이 그대로 재전송하는 문제를 스트리밍 경로와 같은

## 구현 노트
- 무엇/왜: `providers.go`의 비스트리밍 chat 분기를 스트리밍과 대칭으로 바꿨다 — upstream `Content-Type`을 `mime.ParseMediaType`으로 파싱해 `application/json`·`application/…+json`·header 없음만 통과시키고 응답 헤더는 `application/json; charset=utf-8`로 고정, 그 외(파싱 실패 포함)는 본문을 버리고 `w.WriteHeader` **앞에서** 502 `ai_response_invalid` + 감사 `reason=invalid_response_content_type`. 커밋 ac2ed50(프로덕션 1파일 + 신규 테스트 + `docs/api.md`).
- 확신 없는 곳·검증 못 한 것: ① 이것은 **의도된 계약 엄격화**다 — `text/plain`으로 JSON을 돌려주던 공급자는 이제 502가 된다(과제서가 명시한 수용 기준 2). 실제 그런 공급자가 운영에 있는지는 확인 못 했다. ② `+json` suffix 허용은 과제서의 "권장" 항목이라 내 판단으로 넣었고 테스트로 고정했다. ③ 테스트의 "Content-Type 없음" 케이스는 `w.Header()["Content-Type"] = nil`로 net/http 스니핑을 억제하는데, 이 동작은 Go 런타임 내부 규약에 기댄다(red 단계에서 실제로 header 없이 도착함을 관측해 확인은 했다).
- 일부러 하지 않은 것: `relayChatBody`·`chatRelay`·스트리밍 분기·`panic(http.ErrAbortHandler)` 절단 처리는 손대지 않았다(별개 계약, 기존 테스트가 걸려 있음). 새 사유를 `docs/api.md:275`의 `details.reason` 목록에 끼우지 않았다 — 그 목록은 "200이 나간 뒤 중단된 사유" 전용이라 헤더 전 거부 사유를 섞으면 계약이 흐려진다. 대신 새 비스트리밍 문단에 사유를 적었다. VERSION·CHANGELOG·`internal/ui/dist`는 건드리지 않았다(`web/src`의 `streamJson`은 응답 Content-Type을 읽지 않아 웹 변경 불필요).
- 다음 역할이 조심할 것: 새 테스트 `TestChatNonStreamingResponseContentTypeContract`는 **`TEST_POSTGRES_DSN`이 있어야 돌고 없으면 조용히 SKIP** 하며, 실행 시 `ai_admin`/`aiportal` 스키마를 DROP 한다 — 공유·운영 DB 금지. 이번 검증은 포트 55471의 전용 `postgres:16-alpine`로 했다(`go test -race -count=1 ./...` 전체 통과, internal/server 119.7s / `make lint` / `go build ./...`).
- [러너 08:33] brief accepted — 채택 — 지정된 결함·코드 위치(`providers.go:646-661`)·재사용 셋업(`chat_truncation_integration_test.go`)·허용 집합이 모두 현재 코�
- [러너 08:33] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 원장에 `- 실패 재현:` 줄이 없어 red 상태를 직접 재현했다: main의 `providers.go`만 되돌리면 새 테스트 7개 subtest 전부 FAIL하고 `html_is_rejected`/`plain_text_is_rejected`가 `status=200 want=502 body={"id":"chatcmpl-1",...}` 를 찍는다 — 수정 전 코드가 upstream media type과 본문을 그대로 재전송했다는 증거이고, HEAD에서는 7개 모두 PASS(SKIP 아님). 테스트는 진짜 바뀐 경로를 지난다.
- 전용 `postgres:16-alpine`(포트 55487, 실행 후 삭제)로 `go test -race -count=1 ./...` 전 패키지 통과(internal/server 117.7s), `make lint`·`go build ./...`·`gofmt -l` 모두 깨끗. 기존 chat 테스트는 전부 스트리밍 경로라 이번 비스트리밍 계약 변경에 걸리지 않음을 확인했다. 구현자 자기의심 ③(header 없음 케이스)은 red 실행이 증명했다 — 스니핑이 개입했다면 `text/plain`이 왔을 것이다.
- 못 본 것: 웹 UI 실제 동작(`npm test`/`npm run build` 미실행 — `web/` 변경이 없고 `AiPlaygroundPage.tsx:94`가 스트리밍 경로만 쓰는 것은 코드로 확인), 실운영 공급자가 어떤 Content-Type을 보내는지.
- 승인이어도 남는 우려 — **릴리즈 노트 필수 항목**: 의도된 호환성 축소다. JSON을 `text/plain`/`text/html`로 라벨하던 비스트리밍 공급자는 이제 502 `ai_response_invalid`를 받고 우회 플래그가 없다. 코드 1파일이라 revert로 완전히 되돌아온다(마이그레이션 없음). 부차: `charset=euc-kr` 도 `charset=utf-8`로 덮어쓴다(규격상 옳고 스트리밍과 대칭).
- 보안·법무 부서 차단 없음: 공격면을 좁히는 변경이고 새 인가 경로·식별자·비밀값·의존성·개인정보 수집이 없다. 다음 회차 후보는 `available_models` JSON 파싱 무시와 `workflow.go` `errKeyStale` — upstream 무검증 릴레이 지점은 `providers.go:655`가 유일했고 닫혔다.
- [러너 08:42] review approved — 리뷰 승인 (risk=low)
- [러너 08:42] pr created — https://github.com/hkjang/ai-admin/pull/33
- [러너 08:51] ci passed — 검사 2개 모두 success
- [러너 08:51] merge done — ac2ed50
- [러너 09:04] release published — v1.2.29
- [러너 09:06] assets verified — v1.2.29 자산 2개 (이전 v1.2.28: 2)

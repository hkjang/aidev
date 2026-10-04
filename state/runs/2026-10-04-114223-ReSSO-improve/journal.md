# 회차 노트 2026-10-04-114223-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:42] base pinned — main@d48962d
- [러너 11:42] autonomy release — 
- [러너 11:47] scout done — Token Endpoint 가 판정하지 못한 Client 인증에 401 대신 500 `server_error` 로 답하게 하기 (가치 4 / 위험 2 / 작업량 

## 구현 노트
- 무엇/왜: token 이 `errClientAuthUndecided`(새 센티널)를 보고 500 `server_error` 를 쓴다. `clientAuthUndecided` 는 `(undecided, ours)` 를 돌려주고 `ours` 만 감싼다 — 끊긴 호출자는 감싸지 않아 500 을 못 받는다. introspection·revocation 무변경(계약이 달라 별도 회차). 프로덕션 파일 1개(oidc.go).
- 확신 없는 곳: 없음 — 네 단언 모두 빨간 것을 먼저 봤고(401→500 둘, `ours` 게이트를 빼면 `status="500"` 이 20→21), 끊긴 호출자 경로는 `faulted()` 리더로 직접 단언한다. 다만 `make lint` 가 뱉는 npm audit 건수는 이번에 세지 않았다(ideas.json 에 미확인으로 적음).
- 일부러 안 한 것: `resso_token_errors_total` 증가 없음(이 시점 `grant_type` 은 `client.GrantTypes` 와 대조 전인 미검증 입력 — 475-481 Realm 분기와 같은 판단, 주석에 남김). `writeClientAuthError` 는 손대지 않음 — 세 Endpoint 의 계약이 셋이라 헬퍼에서 합치면 안 된다.
- 다음 역할이 조심할 것: 두 테스트는 실제 PostgreSQL 이 있어야 돈다(`eval "$(scripts/test-services.sh)"`). `…SeparatesABrokenDigest…` 는 `clients` 를 ACCESS EXCLUSIVE 로 잠그므로 락이 밖으로 새면 같은 컨테이너의 다른 테스트가 전멸한다(`release()` 가 `t.Cleanup`). `answered()` 는 status 를 **일부러** 안 보고(핸들러 종료 신호), `faulted()` 가 `status="500"` 만 본다 — 둘을 합치지 말 것.
- 문서: `docs/operations.md:90` 은 문장을 더한 게 아니라 v0.9.97 의 "응답은 둘 다 401 `invalid_client`" 를 **고친** 것이다(이 변경으로 틀린 말이 됨). 같은 이유로 oidc.go 875행 부근 주석의 "answer identically" 문단도 고쳤다. README 지표 표는 계열이 늘지 않아 무변경.
- [러너 12:04] brief accepted — 채택 — 근거(891행이 응답을 일부러 남겨 둔 것, 875행 부근 주석이 스스로 'that is a change per endpoint' 라고 지목한 것, 401 을 
- [러너 12:04] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: oidc.go:511 분기와 818-826 ours 게이트의 공격 경로(비-UTF8 client_id·Argon2 대기열 둘 다 호출자가 500 을 못 올림 — password.go:143-155 로 확인), introspection(1171)·revocation(1302)이 감싼 오류를 받아도 401 유지, docs/operations.md 외 invalid_client 문서 부패 없음. 실제 PostgreSQL 로 go test -race ./internal/httpserver 전체 통과(113.8s), gofmt·golangci-lint 0 issues.
- 못 본 것: make lint 전체(govulncheck·eslint)와 web vitest — 이번 diff 에 JS·의존성 변경이 없어 생략. 수정 전 코드로 되돌린 재실행은 하지 않았다(단언이 401→500 으로 뒤집혀 자명).
- 승인이어도 남는 우려(릴리즈 노트용): 401 은 RP 가 재시도하지 않지만 500 은 백오프 재시도를 부른다 — clients 테이블 장애 중 이미 죽어가는 DB 로 재시도 트래픽이 새로 들어온다. 본문은 "retry after a short delay" 라면서 Retry-After 는 없다(이 Endpoint 의 다른 500 넷과 동일). token 라우트 5xx 경보 임계 재검토 필요.
- 다음 회차: 센티널이 생겼으니 introspection(200 active=false + stage 라벨)·revocation(503)은 분기 추가만 남았고, operations.md:90 이 "아직 401" 이라고 이미 적어 두었으니 그 문장도 함께 고쳐야 한다.
- [러너 12:11] review approved — 리뷰 승인 (risk=low)
- [러너 12:11] pr created — https://github.com/hkjang/ReSSO/pull/36

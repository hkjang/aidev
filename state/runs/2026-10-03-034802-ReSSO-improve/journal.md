# 회차 노트 2026-10-03-034802-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:48] base pinned — main@bdfeeb2
- [러너 03:48] autonomy release — 

## 구현 노트
- 바꾼 것: `authenticateOIDCClient` 가 store 장애를 틀린 Secret 과 같이 다뤄 두 Rate limiter 예산을 깎고(장애가 끝난 뒤에도 429 로 잠긴 채 남는다) `client_auth_failures` 에 세던 것을, 새 `clientAuthUndecided` 로 갈랐다 — Limiter 무접촉 + `resso_client_auth_errors_total{stage}` + ERROR 한 줄. 응답은 무변경(954da4b, 프로덕션 파일 2개).
- **확신 없는 곳**: ① `password.VerifyContext` 가 context 취소로 오류를 내는 경우가 이제 Limiter 를 깎지 않는다 — 호출자가 요청을 중단해 만들 수 있는 유일한 경우다. 답을 받지 못하므로 oracle 이 되지 않는다고 판단했고 그 논거를 주석에 적었지만 **실험으로 증명하지는 않았다**. 비평가가 먼저 볼 곳. ② `stage="secret"` 분기(저장된 digest 가 깨진 경우 등)는 테스트가 지나지 않는다 — 테스트는 `stage="client"` 만 단언한다. ③ 주소 Limiter(200회)가 Client Limiter 보다 먼저 걸리는 경우는 단언하지 않았다.
- 일부러 하지 않은 것: 응답 코드. 이 헬퍼를 공유하는 세 Endpoint 의 이쪽 장애 계약이 서로 다르므로(token 500 / revoke 503 / introspection 200 active=false) 헬퍼에서 합치지 않고 ideas.json 의 후속 과제로 남겼다. 지금은 세 곳 모두 401 `invalid_client` 그대로다.
- **main 에 관한 사실**: 프로필이 1순위로 지목한 `RedeemAuthorizationCode` 500 변경은 `5bed9dc`(브랜치 `auto/2026-10-01-1412`)로 이미 있는데 **main@bdfeeb2 에 없다**. 같은 날 형제 PR #34 는 머지됐다 — 사람이 받지 않은 접근으로 보고 고르지 않았다. 정찰이 이 판정을 다시 읽을 것.
- 다음 역할이 조심할 것: 새 테스트는 PostgreSQL 이 있어야 돈다(`eval "$(scripts/test-services.sh)"` 를 **같은 셸에서**). `clients` 테이블을 RENAME 하므로 `t.Cleanup` 복구가 빠지면 뒤따르는 테스트가 전멸한다. Limiter 는 그 한 Server 인스턴스의 메모리라 테스트가 순서 의존적이다 — 사례를 재배열하지 말 것(특히 (a) 가 (b)(c) 보다 먼저여야 수정 전 실패가 핵심 단언에서 난다).
- [러너 04:06] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 테스트가 수정을 정말 고정하는지 직접 재현했다 — 프로덕션 두 파일만 reverse apply 하면 integration_test.go:1649 의 핵심 단언 (a) 가 `429` 로 실패한다(원장에 `실패 재현:` 줄이 없어 리뷰어가 돌렸다). 세 호출자의 계약 불변, 비밀값 비노출, 라벨 카디널리티, 개인정보·라이선스는 문제 없다.
- 거절 사유: `clientAuthUndecided`(oidc.go:855) 의 주석이 단언한 "호출자가 고를 수 없다" 가 거짓이다. `client_id=bad-%ff` → PostgreSQL 22021 → ErrNotFound 아님 → 미인증 호출자가 `resso_client_auth_errors_total{stage="client"}` 와 ERROR 로그를 한도 없이 올리고 두 Limiter 를 영구 우회한다(실험으로 확인). operations.md:93 이 과거 사고로 기록한 `auth_time` 선례와 같은 모양이며, operations.md:90·README.md:192 의 "오르면 이쪽 문제" 설명이 사실과 다르다.
- 수리가 먼저 볼 파일: `internal/httpserver/oidc.go`(verifyOIDCClient 의 `client` 단계 — 조회 전 `client_id` UTF-8 유효성 검증, 또는 호출자 유발 오류를 ErrNotFound 와 같은 "실제 답" 으로) → `docs/operations.md:90`·`README.md:192` 문구.
- 못 본 것: `stage="secret"` 경로(legacy argon2 digest 를 가진 Client 에만 닿는다)와 주소 Limiter 선행 사례는 테스트가 없어 읽기로만 판단했다. Basic 인증 헤더로 같은 바이트를 보내는 경로도 코드상 동일하지만 실측하지 않았다.
- 다음 회차에 남는 것: 세 Endpoint 의 이쪽 장애 응답(500/503/200) 통일은 여전히 미착수 — 이번 변경이 의도적으로 남긴 후속 과제다.
- [러너 04:11] review rejected — 리뷰 거절: internal/httpserver/oidc.go:855 clientAuthUndecided 의 핵심 전제가 거짓이다. 주석(oidc.go:836-841)은 "nothing excluded here can be reached on demand: the errors left are a 

## 수리 노트
- 맞았던 지적: 전부. `client_id=bad-%ff` / `bad-%00`이 SQLSTATE 22021로 `ErrNotFound`를 비껴간다는 것을 store 프로브로, 그리고 가드를 되돌린 실행에서 `resso_client_auth_errors_total{stage="client"}`가 20→22로 오르고 `client_auth_failures_total`은 21에 멈추는 것으로 확인했다. 틀린 지적은 없었다.
- 고친 방법: `oidc.go`에 가드를 넣는 대신 `store.ClientByIdentifier`가 `text`에 담길 수 없는 식별자를 조회 전에 `ErrNotFound`로 판정한다(`storableIdentifier` — UTF-8 유효성 **과** NUL 둘 다 봐야 한다. `utf8.ValidString("\x00")`은 true다). `auth_time` clamp 선례와 같은 모양이고, 같은 구멍을 가진 authorization(`authorization_errors{stage="client"}`)·logout(`client_unavailable`) 호출자까지 닫힌다. 응답·집계는 등록되지 않은 식별자와 동일.
- 여전히 확신 없는 곳: ① 구현 노트의 ①(`password.VerifyContext` ctx 취소가 Limiter를 깎지 않는 것)은 이번에도 실험하지 않았다 — 지적 범위 밖이라 손대지 않았다. ② `stage="secret"` 분기는 아직 테스트가 지나지 않는다(VerifyClientSecret은 uuid만 DB에 넘기므로 호출자가 고를 수 있는 입력은 없다고 읽기로 판단). ③ 세 Endpoint의 이쪽 장애 응답 통일(500/503/200)은 여전히 후속 과제.
- 다음 역할이 조심할 것: 새 단언 (g)는 (f) 뒤에 와야 한다(주소 Limiter 누적 카운트에 의존). `clients_test.go`의 단위 테스트는 DB 없이 돈다.
- [러너 04:19] repair done — # 수리 요약 (59b6c34)  비평가의 지적이 맞았다. 재현 확인: `ClientByIdentifier`에 `"bad-\xff\xfe-utf8"` 또는 `"bad-\x00-nul"`을 넘기면 PostgreSQL이 `invalid byte sequence for 
- [러너 04:34] review timeout — 단계 제한 시간 초과
- [러너 04:34] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 04:34] pr created — https://github.com/hkjang/ReSSO/pull/35

# 회차 노트 2026-09-27-071202-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:12] base pinned — main@955ef7a
- [러너 07:12] autonomy release — 

## 정찰 노트
- 골랐다: token Endpoint의 Realm 조회 실패가 400 `invalid_grant`로 나간다(oidc.go:428-432). `realmFromPath` 호출자 중 `realmLookupFailed`를 쓰지 않는 마지막 하나이고, 같은 함수 489-497행이 "invalid_grant는 RP에게 토큰을 버리라는 뜻"이라고 이미 적어 둔 그 논지가 Realm 조회에만 적용되지 않았다 — 최근 다섯 회차가 전부 채택된 계열(이쪽 장애 ≠ 진짜 없음)이면서, 이번 것은 기록만이 아니라 응답 자체가 틀린 건이라 가치가 한 단 높다.
- 제친 것: userinfo 카운터(로그에 이미 stage가 있어 더하는 값이 작다 — 차선으로 남겼다), 접근 로그 client_id(7회 연속 보류라 rejected로 내렸다), login 409(6회차 재현 실패로 rejected), CSP·id_token_hint(정책·규범 미확인).
- 추측으로 적은 것: 응답을 500 `server_error`로 지정했다 — RFC 6749 §5.2 원문을 확인하지 않았고, 근거는 이 Endpoint가 이미 그 쌍을 쓰고(oidc.go:494·504) 문서가 그렇게 적혀 있다는 것(ADMIN_GUIDE.md:570)뿐이다. 새 코드 `temporarily_unavailable`은 만들지 말라고 못 박았다.
- 구현자가 조심할 것: `ErrNotFound`(없는·꺼진 Realm)는 400 그대로 둘 것, `ALTER TABLE realms RENAME`은 그 서버의 거의 모든 경로를 죽이므로 로그인·토큰 발급을 RENAME 전에 끝내고 cleanup에서 되돌릴 것, `resso_token_errors_total`의 라벨은 `grant_type`이니 `realm` 값을 넣지 말 것.
- 프로필은 profile.md로 갱신했다(HEAD·버전과 oidcCORS 공백이 v0.9.92로 해소된 것만 반영, 나머지는 2026-09-26판 그대로).
- [러너 07:18] scout done — Token Endpoint가 Realm 조회 실패를 `invalid_grant`로 답하는 것을 고치기 — 이쪽 장애는 500 `server_error`로 (가치 4 

## 구현 노트
- 무엇/왜: `token`(oidc.go:428)의 `realmFromPath` 오류를 `s.realmLookupFailed(r, "token", err)`로 갈라 이쪽 장애만 500 `server_error`로 답하고 `endpoint=token` 로그를 남긴다. `invalid_grant`는 RP가 code/refresh token을 버리는 신호라, 읽지도 못한 grant 때문에 장애가 끝나면 살릴 수 있던 세션이 전부 버려졌다. 프로덕션 파일은 oidc.go 하나 + 테스트 + docs/operations.md 한 문장.
- 확신 없는 곳: 없는 Realm·꺼진 Realm이 400 그대로인 것은 테스트 (c)(d)가 직접 단언하지만, `realmFromPath`의 **다른** 호출자들의 응답은 이번 변경과 무관하므로 따로 확인하지 않았다(코드상 무변경). 에러 문구 `...retry after a short delay`는 과제서 예시를 그대로 썼고 외부 문서/번역 대조는 하지 않았다(다른 500 문구들도 영문 원문이다).
- 일부러 안 한 것: 지표·감사 항목 미추가(`resso_token_errors_total`의 유일 라벨 `grant_type`을 읽기 전에 실패 — 주석으로 근거 남김). `temporarily_unavailable` 미도입. `handleAuthorizationCodeGrant`/`handleRefreshGrant`/`handleClientCredentialsGrant` 안의 `invalid_grant`들과 `UserByID` 장애 합침은 손대지 않음(별 과제, ideas.json에 있음).
- 다음 역할이 조심할 것: 새 테스트는 실제 PostgreSQL이 필요하다 — `eval "$(scripts/test-services.sh)"` 를 같은 셸에서 먼저. 테스트가 `ALTER TABLE realms RENAME` 으로 장애를 만들고 `t.Cleanup` 으로 되돌린다; 중간에 죽이면 같은 컨테이너의 뒤 테스트가 줄줄이 깨지므로 그때는 `ALTER TABLE realms_hidden RENAME TO realms` 를 손으로 실행할 것. `make test`가 `webui/dist/index.html`을 바꾸므로 커밋 전 복원(이번에 복원함).
- [러너 07:30] brief accepted — 채택 — 근거(428-432행이 `ErrNotFound` 구분 없이 400 `invalid_grant`, 489-497행에 같은 논지가 이미 있음, `realmLookupFailed`를 쓰지 않
- [러너 07:30] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). main@955ef7a 워크트리에 새 테스트만 얹어 돌려 실패를 직접 재현했고(integration_test.go:1058, 400 invalid_grant — 원장 출력과 동일), 브랜치에서는 PASS, `go test -race ./internal/httpserver -count=1` 전체 115.6s exit 0, gofmt·vet 깨끗. 테스트는 (b)에서 같은 refresh token이 복구 후 200인 것까지 봐서 "버려지지 않았다"를 진짜로 증명한다. 없는·꺼진 Realm이 400 invalid_grant + 무음 로그로 남는 것도 (c)(d)가 단언 — Realm 존재 오라클은 생기지 않았다(보안·법무 소견 없음).
- 못 본 것: `make lint`·vitest·store 패키지는 재실행하지 않았고(원장 기록 신뢰), 다른 `realmFromPath` 호출자는 코드상 무변경만 확인했다.
- 남는 우려 1: oidc.go:440-442와 operations.md:86의 "grant_type 값을 읽기 전에 실패한다"는 근거가 부정확하다 — `r.ParseForm()`이 423행에서 이미 돈다. 결론은 옳고 진짜 근거는 "미검증 caller 입력을 라벨에 넣지 않는다"(카디널리티)다. 다음 회차가 주석만 그 근거로 바꾸면 깔끔하다.
- 남는 우려 2: ADMIN_GUIDE.md:570의 `resso_token_errors_total{stage=…}`는 라벨 이름이 틀렸고(실제 `grant_type`, metrics.go:56) 새 operations.md 문장과 정면으로 어긋난다 — 이번 변경 전부터의 오류지만 이제 눈에 띈다. 그리고 기존 표 테스트(integration_test.go:10376)에 token 행이 없어 "모든 OIDC Endpoint" 표가 계열을 다 담지 못한다. 릴리즈: 워크트리의 uncommitted `webui/dist/index.html`을 함께 커밋하지 말 것.
- [러너 07:36] review approved — 리뷰 승인 (risk=low)
- [러너 07:37] pr created — https://github.com/hkjang/ReSSO/pull/30

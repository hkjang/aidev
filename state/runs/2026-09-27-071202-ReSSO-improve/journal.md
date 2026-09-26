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

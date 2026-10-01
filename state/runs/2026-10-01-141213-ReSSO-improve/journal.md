# 회차 노트 2026-10-01-141213-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:12] base pinned — main@8604422
- [러너 14:12] autonomy release — 

## 정찰 노트
- 코드가 스스로 남긴 지목을 골랐다: oidc.go:578-585 주석이 "아직 안 한 변경" 둘을 명시하는데, RotateRefreshToken 쪽은 store 변경(306-312의 ErrNotFound 합침)이 필요하고 RENAME 으로는 v0.9.95 분기가 먼저 잡혀 새 분기에 도달조차 못 한다 — 그래서 장애 주입이 깨끗한 RedeemAuthorizationCode 쪽을 택했다. 차선(지표 배선 고정)은 프로덕션 가치가 낮아 밀었다.
- 확인한 것: 없는·만료된·불일치 코드는 모두 store.ErrNotFound 로 모인다(store/oidc.go:118-127·180-182, oidc.go:509-511). ValidatePKCE 는 센티널 없는 errors.New 넷(pkce.go:10-25) — 래핑 없이 가르면 PKCE 불일치가 500 이 된다.
- 추측으로 적은 것: 센티널 이름·본문 문안은 제안이다. store/oidc.go:169-175(재사용 처리 중 Exec/Commit 실패)가 새 500 통에 들어가는 것은 코드로 확인했으나 테스트 재현은 미확인 — 주석으로만 밝히고 범위를 넓히지 말 것.
- 구현자가 조심할 것: 코드는 한 번 쓰면 소비되므로 장애용·복구용·PKCE용 코드를 각각 따로 채굴할 것. authorization_codes RENAME 은 단언 전 즉시 복구(90초 수명 + 공유 컨테이너).
- 기각: 비화면 경로 CSP 좁히기 — 10회 연속 보류였고 직전 정찰의 권고대로 rejected 로 내렸다.
- [러너 14:17] scout done — 코드 교환이 `RedeemAuthorizationCode` 의 저장소 장애를 `invalid_grant` 로 답하는 것을 500 `server_error` 로 (가치 3 / �

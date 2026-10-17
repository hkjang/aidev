# 회차 노트 2026-10-17-121900-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:19] base pinned — main@69b62df
- [러너 12:19] autonomy release — 

## 정찰 노트
- 보류 1순위였던 `baoKVMutateVersions` 의 0·음수 버전은 결함이 아니라 compatibility.md:97 에 명시되고 secrets_openbao_test.go:67 이 단정하는 계약이어서 기각했다. LIKE 이스케이프·URL userinfo·OpenBao 본문 상한도 모두 이미 닫혀 있어 기각했고, 그 결과 "같은 서버 안에서 한 곳만 다른 게이트" 인 openbao.go:241 (`r.ContentLength > 0`)을 골랐다 — resource_handlers.go:521·ai.go:228·oidc.go:68 은 모두 `!= 0` 이다.
- 추측으로 적은 것: (a) 수정 전 chunked `{"ttl":"1h"}` 가 실제로 12시간 토큰이 되는 것은 **코드 독해**이고 실행 재현은 못 했다(store·DB 가 필요). (b) 실제 OpenBao/Vault 의 chunked 본문 처리는 오프라인에서 확인 불가 — 근거는 호환성이 아니라 "호출자가 보낸 값을 버리지 않는다" 와 저장소 내부 일관성이다.
- 구현자가 조심할 것: `decodeBaoJSON` 을 공유 수정하면 transit/kv 의 "빈 본문 → 400" 계약이 깨진다. 선택적 본문 전용 헬퍼로 분리하고, **빈** chunked 본문이 400 이 되지 않는지 반드시 기존 계약 테스트로 묶을 것.
- 기준 1)을 증명하려면 `sessionCreator` seam 추가가 필요하다(server.go:220-232 의 nil-이면-store 패턴 그대로). seam 주입 테스트는 배선 결함을 못 보므로 "불리지 않았다" 를 카운터로, 전달된 ttl 을 인자로 단정하게 했다.
- 프로필은 1일 전 것이 그대로 유효해 새로 쓰지 않았다. 어긋난 한 줄: 기준이 이제 main@69b62df / `scripts/version.sh` 정본 **v0.2.39** 다(v0.2.38 아님).
- [러너 12:25] scout done — OpenBao `auth/token/create` 가 길이를 알 수 없는(chunked) 요청 본문을 조용히 버려 `ttl`·`display_name` 을 무시하고 �

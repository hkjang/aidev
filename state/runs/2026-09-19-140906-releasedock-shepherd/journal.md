# PR 처리기 노트 2026-09-19-140906-releasedock-shepherd — releasedock PR #18
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 세 지적 모두 맞았다: 원본 코드에 새 테스트를 돌리면 X-Forwarded-Host 로 만든 aud 의 토큰이 200, 거절 로그에 request_id 없음, fetch 중 캐시된 kid 가 뮤텍스 대기, 실패 fetch 가 요청마다 반복됨을 그대로 재현했다. 틀린 지적은 없었다.
- 고친 방법: requestOrigin 폴백 삭제(+ 설정 해석 함수에서 *http.Request 인자 제거로 구조적으로 차단), withAuth Warn 에 request_id, 키 캐시는 잠금 밖 single-flight fetch + attempted/lastErr 로 실패도 10s 스로틀. 선택 항목(putGeneralSettings 에서 publicUrl 비우기 거절)도 넣었다.
- 검증: go test ./... 전부 통과(임시 Postgres), 새 동시성 테스트 -race -count=5 통과. 커밋 ff47e4e 하나, push 안 함.
- 확신 없는 곳: (1) 콜드 캐시에서 fetch 가 실패하면 10s 동안 모든 토큰이 캐시된 503 을 받는다 — 의도한 스로틀이지만 Keycloak 이 잠깐 튄 직후 복구가 최대 10s 늦다. (2) inflight 대기자가 ctx 취소로 빠질 때 503 을 주는데, 요청 ctx 가 짧은 클라이언트에는 401 보다 나은 선택인지 판단이 남아 있다.

## 심사 노트
- 확인: go test ./... 전부 통과(임시 Postgres 16, 통합 테스트 skip 아님), MCP OAuth 테스트 -race -count=3 통과. 새 테스트를 d0a9bdc 에 옮겨 돌리면 X-Forwarded-Host aud→200, request_id 없음, 캐시된 kid 가 fetch 에 블록, 실패 fetch 반복이 그대로 재현됨 — 세 지적의 수리는 진짜다. 마이그레이션은 ADD COLUMN IF NOT EXISTS 만, 기본값 꺼짐, ViaAPIKey 게이트 3곳 모두 SSO 주체에 적용, 빈 scope 는 거부.
- 결함 1(재현): mcpoauth.go needsFetch 가 TTL(5분) 지난 뒤엔 캐시에 있는 kid 도 fetch 대상으로 보고, mcpSigningKey 가 그 요청을 fetch 완료(최대 10s)까지 잡아 둔다. 느린 Keycloak 이면 TTL 후 10s 마다 10s 씩 모든 MCP SSO 인증이 멈춘다 — 캠페인 규칙 "보유 kid 는 TTL 지나도 즉시 반환" 위반, 이전 거절 사유 3의 "느린 Keycloak 이 전부 멈춤" 이 절반만 고쳐졌다. 임시 테스트로 재현했다(2건 모두 블록).
- 결함 2: docs/MCP.md:32 가 삭제된 "요청의 호스트에서 만듭니다(마지막 수단)" 폴백을 아직 문서화한다.
- 못 본 것: 실제 Keycloak 과의 E2E, 프런트(SettingsPage) 렌더링. 수리 노트의 "확신 없는 곳" (1)(2) 는 의도된 설계로 보고 통과시켰고, fetch 수행자의 ctx 취소가 실패로 기록되어 콜드 캐시가 10s 503 되는 점은 노트로만 남겼다.
- 권고 fix: 두 결함은 이 브랜치 위에서 작게 고칠 수 있다. 방향 자체는 맞다.

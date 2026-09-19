# PR 처리기 노트 2026-09-19-134956-Momento-shepherd — Momento PR #14
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 지적 3건 모두 맞았음. 1번은 임시 테스트로 Host: other-app + aud=http://other-app/mcp 가 200 을 받는 것을 재현한 뒤 고침. 2번은 코드 확인(락 안 NewProvider·실패 미캐시), 3번은 로거가 LevelError 라 Warn 이 버려지고 단언이 없던 것 확인.
- 고친 방법: r.Host 폴백 제거 + active() 에 Resource 필수 + 저장 시 400; discovery 는 락 밖 per-issuer 단일 비행 + 30초 실패 캐시; chi RequestID 추가 + 거절 warn 에 request_id, 테스트는 버퍼 로거 서버로 원문 오류·request_id 단언(커밋 23a78df).
- 지적과 다르게 한 곳: "another issuer" 표 항목은 다른 키로 서명되어 go-oidc 가 서명 단계에서 먼저 거절하므로 기대 문구는 "failed to verify signature". "issued by a different provider" 는 우리 키로 서명하고 iss 만 바꾼 "claiming another issuer" 항목을 추가해 단언. "unsigned"(`a.b.`)는 LooksLikeJWT 를 통과하지 못해 go-oidc 에 닿지 않으므로 세션 경로 401 + SSO 로그 없음 을 단언하고, alg=none 항목을 별도로 추가.
- 확신 없는 곳: discovery 를 요청 ctx 로 돌리므로 선두 요청이 취소되면 대기자들이 한 번 더 시도한다(abandoned 루프) — 폭주는 아니지만 취소가 잦은 환경에서 재시도가 한 번씩 늘 수 있음. request_id 는 거절 warn 에만 넣었고 requestLog 에는 넣지 않음(최소 훅).
- 로컬 검증에 CI 와 같은 `go test -race` 를 썼고, scale 테스트 1회 타이밍 flake 는 원본 HEAD 와 동일한 테스트라 무관하다고 판단.

## 심사 노트
- 확인한 것: 이전 거절 사유 3건을 각각 변이로 되돌려(Host 폴백 복원 / 로그에서 error·request_id 제거 / 락 안 discovery·실패 미캐시) 새 테스트가 실제로 실패하는지 확인했고, 임시 Postgres 로 `go vet` + `go test -race ./...` 와 web `mcpOauth.test.mjs` 전부 통과함.
- 보호 파일: 018 은 enabled:false 로 추가만(ON CONFLICT DO NOTHING), 공개 경로 추가는 RFC 9728 메타데이터 2개뿐(비활성 시 404), 세 관리자 게이트가 Programmatic() 으로 통일돼 oauth 주체 거부, go-oidc v3.16 이 JWKS 를 Background ctx 로 만드는 것을 소스에서 확인.
- 못 본 것: 실제 Keycloak 26 상대 E2E 는 없음(fake IdP 로 대신); chi RequestID 가 인바운드 X-Request-Id 를 그대로 쓰는 점은 결함으로 보지 않음.
- 권고: approve / merge, risk medium(인증 경로지만 기본 꺼짐·추가만·revert 가능).

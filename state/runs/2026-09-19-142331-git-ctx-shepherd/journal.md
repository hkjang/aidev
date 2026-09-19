# PR 처리기 노트 2026-09-19-142331-git-ctx-shepherd — git-ctx PR #30
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 네 지적 모두 맞았다: request_id 누락, Host 유래 aud 분기, 락 안 discovery·WithoutCancel·실패 미캐시, 그리고 테스트의 request_id 미단언. 추가로 발견: "버퍼 전체에 request_id= 포함" 식 단언은 미들웨어의 http_request 줄 때문에 필드가 없어도 통과하므로 거절 줄 단위로 단언했다.
- 고친 방법: `logOAuthRefusal(w,r,…)`, `mcpResource(ctx)` 는 ui.publicUrl/비기본 cfg.PublicURL 만 인정하고 `mcpOAuthActive` 가 비활성 사유 반환(새 상수 `config.DefaultPublicURL`), `resourceVerifier` 는 락 밖 discovery + 30초 실패 캐시(호출자 취소는 미캐시). 각 테스트를 원본 코드에 대해 돌려 실패함을 확인한 뒤 고정. 커밋 f4e558f.
- 확신 없는 곳: (1) `cfg.PublicURL` 이 상수 기본값과 같으면 미설정으로 보는 판정 — 운영 환경에선 env 로 바꿀 길이 없어 실질적으로 ui.publicUrl 만 의미 있음. (2) 검증기 자체(키셋 fetch)는 여전히 `WithoutCancel` ctx 로 만든다 — 키 로테이션이 첫 호출자 취소에 깨지지 않게 하려는 원 의도를 유지한 것이며, discovery 만 요청 ctx 다. (3) 웹 로그인 `get()` 의 같은 패턴은 지시대로 손대지 않았다.

## 심사 노트
- 확인: gofmt/vet/build 깨끗, `go test ./...` 32 패키지 전부 통과, 콘솔 `node --check`·test/web 통과. 새 테스트 4종을 변이로 검증(request_id 제거·DefaultPublicURL 유도 복원·락 안 discovery·Restricted() 축소 → 각각 실패)해 캠페인 규칙 1~4 수리는 실제로 고정됨. `p.KeyID != ""` 게이트는 남은 곳 없음(규칙 5).
- 결함(거절): `oauthPrincipal` 의 계정 조회(mcpoauth.go:303-306)가 `u.subject=? OR u.username=?` 로 username 폴백을 한다. 웹 로그인 `upsertIdentity` 는 subject 만 본다(auth.go:1012). 스크래치 복사본에서 재현: sub=kc-mallory, preferred_username=break-glass-admin 토큰이 break-glass-admin 행(복구 로그인 후 영구 active + user_roles platform-admin)에 매핑돼 tools/list 에 get-platform-status 가 나옴. PR 이 스스로 내건 "토큰이 플랫폼 롤을 만들지 않는다·같은 조회만 한다" 불변식 위반.
- 못 본 것: 실제 Keycloak 상대 E2E 는 없음(가짜 IdP 만). 웹 로그인 `get()` 이 여전히 락을 잡은 채 discovery 하므로 웹 discovery 가 멈추면 /mcp SSO 도 v.mu 에서 기다림 — 기존 동작이라 거절 사유엔 넣지 않음. 실패 캐시 시 `v.resource=nil` 로 동시 성공분을 지울 수 있는 작은 경합도 사유 외.
- 권고: fix — subject 단독 조회로 좁히고, 외부 subject + 기존 username(특히 break-glass-admin) 토큰이 거부되는 테스트 추가.

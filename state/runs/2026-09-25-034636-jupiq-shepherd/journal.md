# PR 처리기 노트 2026-09-25-034636-jupiq-shepherd — jupiq PR #17
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 지적 3건 다 맞았다. 재현: 옛 oidcLoginTokens 로도 TestOIDCLoginTokensAreDistinct 통과, `providers:` 제거·캐시 우회 두 돌연변이 모두 auth/api 테스트 녹색.
- 고침(6129751): `var randomToken = secure.RandomToken` seam, Service.oidcSettings seam(테스트가 DB 없이 로그인 구동), oidc_wiring_test.go 4개 — 난수 실패 거부(3 draw 각각), NewService().providers 고정, 손대지 않은 캐시로 실제 Discovery 히트 1, OIDCLogin→OIDCCallback E2E 히트 1.
- 세 돌연변이를 각각 되살려 해당 테스트가 실패함을 확인했다(원복 완료). `go test ./...` 전부 ok, auth 는 PASS 24·SKIP 0.
- 확신 없는 곳: httptest IdP 는 SafeHTTPClient 가 loopback 을 막아 비loopback IPv4 에 바인딩하며, 그런 주소가 없는 러너에서는 두 네트워크 테스트가 skip 된다(여기서는 skip 0). E2E 는 토큰 교환 실패에서 멈추므로 ID token 검증·UpsertOIDCUser 는 여전히 미검증(Store 가 구체 타입이라 DB 없이는 불가).
- Service.oidcSettings 는 테스트 전용 seam이라 프로덕션 경로는 nil로 두었다. 비평가가 이 seam 자체를 문제 삼는다면 대안은 store 인터페이스화뿐인데 범위를 크게 넘는다.

## 심사 노트
- 승인(merge). 이전 거절 3건을 돌연변이로 직접 재현해 각각 지정 테스트가 FAIL 함을 확인했다: `providers:` 제거→TestNewServiceInstallsTheProviderCache, OIDCLogin 캐시 우회→TestOIDCLoginAndCallbackShareOneDiscovery("after the callback = 2, want 1"), 엔트로피 err 무시→TestOIDCLoginRefusesWhenATokenDrawFails. 전부 원복해 워킹트리 clean.
- 실행 증거: `go test -count=1 -v ./internal/auth/` PASS 24·SKIP 0(네트워크 테스트 2개가 실제로 실행됨을 -v 로 확인), `-race` ok, `go vet ./internal/...` 무출력, `go test ./...` 전 패키지 ok. 배선 테스트는 손대지 않은 newOIDCProviderCache→실제 oidc.NewProvider→실제 SafeHTTPClient 를 탄다.
- 보호 파일 점검 통과: 새 공개 경로·미들웨어 우회·권한 확대·마이그레이션 없음, 캐시 키에 verify_tls 포함이라 TLS 재활성화 후 insecure 클라이언트 재사용 없음, 비밀값 비노출, 코드·문서뿐이라 revert 가능. 엔트로피 실패 fail-closed 는 빈 state/PKCE 로 로그인하던 기존 결함을 없앤 보안 개선.
- 못 본 것: 이 머신에 pdftotext 가 없어 ADMIN_GUIDE.pdf 본문 문자열은 대조 못 했다(/Image 72·/Font 51·23쪽이 origin/main 과 동일한 구조 지표로만 확인). E2E 는 토큰 교환에서 멈춰 ID token 검증·nonce 불일치·UpsertOIDCUser 는 여전히 미검증 — PR 이전에도 그랬으므로 회귀 아님.
- 비차단 참고: Service.oidcSettings 는 테스트 전용 seam(프로덕션 nil, 설정처는 테스트 1곳뿐)이라 수용하되 다음 auth 작업 때 정리 후보. forget() 반복 호출로 캐시를 비울 수 있으나 Discovery 는 분당 120회 제한된 로그인 시작에서만 나가 최악이 PR 이전 동작이다. randomToken 전역 var 는 t.Parallel 이 생기면 깨진다.

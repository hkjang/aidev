# 수리 요약 — jupiq PR #17 (커밋 6129751)

- 지적 3건 모두 재현됨: 옛 oidcLoginTokens 로 되돌려도 TestOIDCLoginTokensAreDistinct 통과, NewService 의 `providers:` 제거와 OIDCLogin 의 캐시 우회 둘 다 `go test ./internal/auth/ ./internal/api/` 녹색.
- 고침: `var randomToken = secure.RandomToken` seam + Service.oidcSettings seam(테스트가 DB 없이 OIDCLogin/OIDCCallback 을 구동). oidc_wiring_test.go 추가 — (a) 3번의 draw 를 차례로 실패시켜 OIDCLogin 이 에러를 돌려주고 URL·쿠키·만료를 내보내지 않음, (b) NewService(...).providers 와 그 필드들이 살아 있음, (c) 손대지 않은 newOIDCProviderCache() 로 실제 oidc.NewProvider+SafeHTTPClient 가 httptest IdP 를 두 번 조회해도 Discovery 히트 1, (d) OIDCLogin→OIDCCallback E2E 로 Discovery 히트 1·토큰 엔드포인트 도달.
- 돌연변이 검증: 세 돌연변이(난수 에러 무시 / NewService 캐시 제거 / OIDCLogin 직접 discovery)를 각각 되살리면 해당 테스트가 실패함을 확인 후 원복.
- 검증: `go test -count=1 ./...` 전부 ok, `go test -v ./internal/auth/` PASS 24·SKIP 0, `gofmt -l` 깨끗, `go vet ./...` 통과. httptest IdP 는 SafeHTTPClient 가 loopback 을 막으므로 비loopback IPv4 에 바인딩(없으면 skip).
- 미실행: `make lint` 의 프런트 단계(web/node_modules 없어 tsc not found) — 이 변경은 Go 전용.

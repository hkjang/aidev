# fix-summary — igame PR #40

- 문제: CI 의 `Scan reachable Go vulnerabilities` 가 exit 3. 원인은 이 PR 의 ORDER BY 변경과 무관하며, 저장소가 고정한 Go 1.26.6 의 stdlib 취약점 10건(GO-2026-6603/6605/6607/6608/6609/6610/6611/6612/6613/6617, net/http·net/textproto·crypto/tls, 모두 `Fixed in go1.26.9`)이 reachable 이기 때문이다. main@5969bec 도 지금 돌리면 같은 자리에서 붉다.
- 재현: `GOTOOLCHAIN=go1.26.6 govulncheck@v1.6.0 ./...` → EXIT=3, "affected by 10 vulnerabilities", 트레이스 #105–#130 이 CI 로그와 동일. `go1.26.7` 도 EXIT=3, `go1.26.9` 만 EXIT=0 — 버전이 원인임을 양방향으로 확인.
- 수정: Go 핀 1.26.6 → 1.26.9 를 일관되게 갱신(`go.mod`, `Dockerfile` builder·라벨, `scripts/check-release-contract.sh`, ci/release workflow 의 라벨·`go version -m` 단언, docs 2곳). 게이트나 테스트는 느슨하게 하지 않았고, PR 의 catalog.go/테스트/docs 변경은 그대로다.
- 검증(로컬, go1.26.9): `govulncheck ./...` = "No vulnerabilities found" EXIT 0 · `make check-contract` · `make lint` · `make test`(Go 8 pkg + SDK 9 + web 246) · `make test-race` · `make build` · `make docker-build` 후 라벨 `1.26.9`·바이너리 `go1.26.9` 확인 · PG(postgres:17-alpine) `make test-db` ok(api 37.3s/database 2.0s), PR 의 History 테스트 3개 PASS, 잔여 테스트 스키마 0개.
- 못 돌린 것: Anchore/Grype 최종 이미지 스캔(액션 전용, 베이스는 scratch 그대로라 취약점이 늘 수 없음)과 `npm audit --omit=dev`(`make deps` 가 "found 0 vulnerabilities" 보고).

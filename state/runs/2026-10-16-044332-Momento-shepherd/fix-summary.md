# fix-summary — Momento PR #32 (2865192)

- 지적은 맞았다. 실측: `net.ParseCIDR("::ffff:0:0/96")` 의 `(*net.IPNet).String()` 은 `0.0.0.0/0`, `::ffff:10.0.0.0/104` 는 `10.0.0.0/8`, `::ffff:192.168.1.1/128` 은 `192.168.1.1/32` — 주소 체계가 바뀌고 마스크가 96 비트 짧아진다. 띄운 postgres:17-alpine 에서 main 이 저장하던 값이 `::ffff:0.0.0.0/96` 임과 `'203.0.113.9'::inet <<= '::ffff:0.0.0.0/96'` = f / `<<= '0.0.0.0/0'` = t 도 확인했다(collector.go:431 판정 역전).
- 고친 방법: `normalizeCIDR` 이 넷마스크 폭 128 이고 파싱된 주소가 `To4() != nil` 인 경우(IPv4-mapped)만 분기해 `netip` 으로 `::ffff:a.b.c.d/N` 글자를 직접 만든다. 판정(accept/reject)은 여전히 `net.ParseCIDR` 하나가 정하고, 그 외 경로는 종전 `network.String()` 그대로다. 주석도 이 분기와 collector 영향을 적도록 고쳤다.
- 테스트: `TestNormalizeCIDRKeepsTheAddressFamilyOfIPv4MappedIPv6` 신규(`::ffff:0:0/96`·`::ffff:10.0.0.0/104`·`::ffff:192.168.1.1/128`·`::ffff:0.0.0.0/96`·호스트비트 `::ffff:10.1.2.3/120`, 기대값 손으로 고정), 핸들러 DB 표에도 `::ffff:` 두 줄 추가. 고치기 전 이 둘이 위 다섯 값 그대로 red → 고친 뒤 green.
- 게이트 실측(DSN 준 로컬): `go vet ./cmd/... ./internal/...` exit 0, `go test -race ./cmd/... ./internal/...` 전 패키지 ok, `govulncheck ./cmd/... ./internal/...` "No vulnerabilities found", `gofmt -l internal/httpapi/` 비어 있음. 02:49 의 CI test=failure 는 govulncheck 였고 이 브랜치의 6cef4b3 가 이미 해결한 상태다.
- 안 돌린 것: web·sdk 게이트와 Docker 빌드(이 PR 은 web/sdk 무변경). 넷마스크가 96 보다 짧은 mapped 표기(예: `::ffff:10.0.0.0/64`)는 `::/64` 로 나가는데 — 체계는 유지되고 실제 포함 대역이 맞지만 — main 에서는 22P02 로 500 이던 값이라 표에 넣지 않았다.

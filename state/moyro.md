# moyro 자율 개선 기록

## 2026-09-02
- 선택: 링크 프리뷰 SSRF 가드의 DNS 리바인딩 취약점 수정 및 links 패키지 첫 테스트 추가 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `server/internal/links`의 `safeDialContext`가 호스트를 조회해 모든 응답 주소를 검사한 뒤 원래 호스트명을 `net.Dialer`에 그대로 넘겨 DNS를 두 번 해석했다. 리바인딩 응답이 검사 시점에는 공인 IP, 연결 시점에는 127.0.0.1을 반환하면 인증 사용자용 `/api/v4/link_preview_image` 프록시가 루프백·사설망으로 유도될 수 있었으므로, 검증한 주소로 직접 다이얼하도록 `dialTargets`를 분리하고 CGNAT·192.0.0.0/24·멀티캐스트·예약 대역을 차단 목록에 추가했다. 테스트가 하나도 없던 패키지에 URL 추출, OpenGraph 파싱, 주소 정책, 주소 고정, 캐시 축출 테스트 12개를 추가했고 `go vet ./...`, `go test -race ./...`(44 패키지 통과), `scripts/check-source-sizes.sh`로 검증했다. 웹 변경이 없어 webapp 빌드는 손대지 않았다.
- 보류 아이디어: (1) 로드맵의 create-post 인가·멤버십 2회 쿼리를 단일 쿼리로 병합 — PostgreSQL 통합 테스트 환경이 필요해 보류. (2) `ratelimit.Limiter.Middleware`가 JSON 본문을 `http.Error`로 써서 Content-Type이 text/plain으로 나가는 문제. (3) 테스트가 전혀 없는 `invites`/`sidebar`/`userstatus`/`postacks` 패키지의 단위 테스트 보강. (4) 로드맵의 메시지 목록 가상화(로드된 모든 행이 DOM에 남음) — 작업량 L이라 단일 세션 범위 초과.
- 릴리즈: v0.2.10 (2026-09-02)

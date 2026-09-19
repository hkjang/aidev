# 회차 노트 2026-09-20-064416-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:44] base pinned — main@461dbda
- [러너 06:44] autonomy release — 

## 구현 노트
- 무엇·왜: `/momento/*` ReverseProxy 가 수집기 응답의 `Set-Cookie` 와 CSP·X-Frame-Options 등 보안 헤더를 Kkiit 오리진으로 그대로 흘리고(중복 헤더 값), 기본 Transport 라 침묵하는 수집기에 WriteTimeout 60초까지 잡혀 있었다 → `ModifyResponse` 로 8종 제거 + `netguard.Client(10s, allowPrivate=true).Transport` 로 dial 5초·응답 헤더 10초 상한(55b989d). 별도 커밋으로 gofmt 위반 파일 포맷 + `make check` 첫 줄 gofmt 게이트(2124f03).
- 확신 없는 곳: (1) `momentoProxyTimeout` 을 `var` 로 두고 테스트가 300ms 로 바꾼 뒤 `t.Cleanup` 으로 되돌린다 — 테스트가 `t.Parallel` 이 아니라 안전하지만, 누군가 병렬로 바꾸면 경합. (2) ADMIN_GUIDE.pdf 는 md2pdf 로 다시 구웠고 쪽수(28)는 같지만 pdftotext 가 없어 새 문장이 PDF 본문에 들어갔는지 텍스트로는 확인하지 못했다(원본 md 는 확인). (3) 실제 Momento 수집기가 스트리밍/긴 응답을 쓰는지 모른다 — `ResponseHeaderTimeout` 이라 헤더만 10초 안에 오면 본문은 제한하지 않으므로 문제는 없을 것으로 본다.
- 일부러 하지 않은 것: 보안 헤더 목록에 `Access-Control-*` 은 넣지 않았다(같은 오리진 프록시라 브라우저가 쓰지 않지만 수집기가 의도적으로 보낼 수 있고 무해). netguard 의 private 차단은 켜지 않았다 — Momento 는 보통 사내 호스트라 막으면 기본 구성이 깨진다. 프런트·openapi 변경 없음(`/momento/*` 는 openapi 밖).
- 다음 역할이 조심할 것: 통합 테스트는 `KKIIT_TEST_DSN` 의 버릴 수 있는 Postgres 가 필요(postgres:16-alpine 컨테이너로 79초 통과 후 삭제). `make check` 는 이제 gofmt 가 어긋나면 go test 이전에 멈춘다. 새 테스트 `TestMomentoProxyGivesUpOnASilentCollector` 는 실시간 0.3초를 기다린다.
- [러너 06:52] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인: 두 커밋 전체 diff, `netguard.Client` 본문(dial 5s·ResponseHeaderTimeout=timeout), `securityHeaders` 미들웨어와 8종 제거 목록의 대응, `momentoProxy()` 가 라우터에서 한 번만 호출돼 Transport 가 공유되는 것, httpapi 패키지에 `t.Parallel` 이 없어 `momentoProxyTimeout` 전역 교체가 안전한 것, `TestMomentoProxy*` 3개 통과 + go vet, gofmt -l 깨끗함. 구현자가 못 본 PDF 는 pypdf 로 열어 28쪽·`Set-Cookie`·`10초` 문장이 본문에 들어간 것을 확인했다.
- 테스트 유효성: 헤더 테스트는 수정 전 코드에선 ReverseProxy 가 Add 로 붙여 CSP 값이 2개가 되고 Set-Cookie 가 통과해 실패하며, 침묵 수집기 테스트는 수정 전엔 영원히 기다려 go test 타임아웃으로 실패한다 — 둘 다 변경을 실제로 고정한다.
- 승인이지만 남는 우려: netguard Transport 는 `Proxy: http.ProxyFromEnvironment` 와 ForceAttemptHTTP2 가 없어 기본 Transport 와 달리 HTTP(S)_PROXY 환경변수를 더 이상 타지 않고 HTTP/1.1 로만 간다. 수집기를 egress 프록시 뒤에 둔 설치가 있다면 이 릴리즈에서 조용히 502 가 난다 — 릴리즈 노트에 적어 둘 것.
- 범위: gofmt 커밋은 포맷 1곳 + Makefile 1줄 + README 한 줄로 작고 분리돼 있어 이탈로 보지 않았다. 통합 테스트(KKIIT_TEST_DSN)는 이 세션에서 돌리지 않았다(변경 파일은 포맷만).
- [러너 06:54] review approved — 리뷰 승인 (risk=low)
- [러너 06:54] pr created — https://github.com/hkjang/Kkiit/pull/8
- [러너 06:55] ci passed — 검사 없음 — 정책으로 허용
- [러너 06:55] merge done — 2124f03
- [러너 06:57] release published — v0.4.3
- [러너 06:58] assets verified — v0.4.3 자산 1개 (이전 v0.4.2: 1)

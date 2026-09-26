# 회차 노트 2026-09-26-203219-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:32] base pinned — main@3ba1767
- [러너 20:32] autonomy release — 

## 정찰 노트
- 404/500 분리 계열은 다섯 회차째라 이번엔 다른 결을 골랐다. `r.ContentLength > 0` 게이트는 세 handler(social.go:271·workflow.go:110·admin.go:367)가 공유하고 posts.go:1032만 올바른 `!= 0` 형태여서 저장소 안에서 계약이 갈려 있고, followTopic은 chunked로 weight 80을 보내면 50으로 저장하며 200을 돌려준다 — 조용히 틀린 데이터가 남는 유일한 후보였다. `resolveReport`(admin.go:382) 500 분리는 안전하지만 같은 계열 반복이라 차선으로 내렸다.
- 추측으로 남긴 것(구현자가 red 단계로 확인할 것): (1) `io.NopCloser(strings.NewReader(...))`를 준 Go client 요청이 실제로 chunked로 나가 서버에서 `r.ContentLength == -1`이 되는지 — 이게 거짓이면 과제 전체의 근거가 사라진다. (2) `httptest.NewServer` 경유에서 세션 쿠키·CSRF가 그대로 통하는지(기존 테스트는 전부 `handler.ServeHTTP` 직접 호출이라 미확인). 막히면 대조군 200을 먼저 세워 원인을 갈라라.
- 이번 회차에서 `go test`·`go vet`·docker는 하나도 실행하지 않았다(읽기만). 코드 근거는 실제로 연 파일에서 확인했으나 동작 재현은 미확인이다.
- 조심할 것: `decodeJSON`(server.go:706) 본체를 고치면 본문 필수 handler 수십 곳이 빈 본문을 통과시킨다. 새 함수 + 세 call site 치환으로만 끝낼 것. `social.go:282`(followTopic의 통합 404)는 지난 회차 verify-failed 자리라 이번 변경과 섞지 말 것.
- 부수 발견: 2026-09-25 회차의 `972113f`(followTopic 500 분리)와 `ab36254`(CI e2e 실패 요약)는 main@3ba1767에 없다. 두 건 다 재구현이 아니라 그 PR을 살리는 문제이므로 정찰 과제 후보에서 제외했다.
- [러너 20:38] scout done — `r.ContentLength > 0` 게이트가 chunked 요청의 JSON 본문을 통째로 버리는 것을 고친다 (가치 3 / 위험 2 / 작업량 M)

## 구현 노트
- 무엇·왜: `followTopic`·`reviewApproval`·`adminResolveReportAlias`의 `r.ContentLength > 0` 게이트가 chunked(ContentLength == -1) 요청의 본문을 버려 weight 80이 50으로 저장되고 반려 사유가 400으로 거절됐다. server.go에 `decodeOptionalJSON`을 신설(EOF 관용은 `ContentLength < 0`에만)하고 세 call site만 치환했다 — `decodeJSON` 본체는 무변경(1d8cf40, 프로덕션 4파일).
- 확신 없는 곳·검증 못 한 것: **HTTP/2 경로는 미검증** — chunked만 실제로 재현했고 "content-length 없는 HTTP/2도 -1"은 net/http 문서 근거이지 이 저장소에서 돌려본 것이 아니다. 프런트·e2e·vitest는 무변경이라 미실행이고 `make lint`의 frontend ESLint 구간은 node_modules 미설치로 실행 불가(Go 구간 = vet + staticcheck 2025.1.1은 통과). `make image`·시각 회귀·govulncheck 미실행.
- 일부러 하지 않은 것: `social.go:282`의 통합 404 분리(지난 회차 verify-failed 자리 — 실패 원인이 갈리지 않게 분리), `admin.go:382 resolveReport`의 500 분리(차선 후보로 남김), `posts.go:1032`(이미 올바른 `!= 0`), `admin.go:284`·`mcp.go:222`의 의도적 본문 폐기, OpenAPI(이미 문서화된 계약을 지키게 하는 수정이라 무변경 — route 120개 유지).
- 다음 역할이 조심할 것: 새 테스트는 **DB가 있어야 돈다**(`MOINA_TEST_POSTGRES_DSN` 없으면 조용히 skip). 검증은 throwaway `postgres:17-alpine`로 했고 `--- SKIP` 0줄·최상위 `TestPostgreSQL*` 36건 PASS를 확인했다. 테스트가 `httptest.NewServer`를 쓰는 것은 이 저장소에서 처음이다(기존은 전부 `handler.ServeHTTP` 직접 호출) — 실제 전선을 타야 전송 계층이 chunked를 만들기 때문이고, `req.ContentLength = -1` 직접 대입은 배선을 우회하므로 쓰지 않았다. handler 앞에 ContentLength를 기록하는 얇은 wrapper를 둔 것은 대역이 아니라 관측점이다(프로덕션 `server.Handler()`가 그대로 요청을 처리한다).
- red 증거: 수정 전 chunked 5케이스만 FAIL(`응답 following=true weight=50 기대 weight=80` 등), content-length 13케이스는 전부 PASS. 과제서 예시 경로가 `/api/...`였으나 실제는 `/api/v1/...`(server.go:152)이라 첫 red는 전부 404였고, 경로를 바로잡아 "틀린 이유의 실패"를 걷어낸 뒤 의도한 red를 얻었다.
- [러너 20:47] brief accepted — 채택 — 과제서의 근거(세 handler의 `ContentLength > 0` 게이트, posts.go:1032의 올바른 `!= 0`, decodeJSON 706줄, integration 테스트 관례)
- [러너 20:47] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 임시 postgres:17로 red/green 양쪽을 직접 돌렸다 — HEAD에서 20개 서브테스트 PASS·SKIP 0, 별도 worktree에서 세 call site만 되돌리니 정확히 5건 FAIL(weight=50 기대 80 등)로 구현 노트의 red 기록과 일치. 테스트는 진짜로 바뀐 경로를 지난다. backend `go test -race ./...`·vet·staticcheck·gofmt·`make check`(120 route) 모두 통과.
- 못 본 것: HTTP/2 경로(구현자가 스스로 표시한 자리) — 나도 재현 못 했다. 다만 분기 값이 chunked와 같은 `r.ContentLength` 하나뿐이라 잔여 위험은 낮다. 프런트·e2e·`make image`·시각 회귀는 무변경이라 미실행.
- 회귀 표면 한정 확인: `ContentLength >= 0` 경로는 수정 전과 동일 동작이고, 기존 호출자(frontend DiscoveryPages.tsx:16·69는 body 없는 POST, mcp.go:222는 GET+빈 body, admin.go:284는 NoBody+0)는 전부 영향 없음.
- 승인이어도 남는 우려 — 릴리즈 노트에 적을 것: chunked/HTTP-2로 깨진 JSON·미정의 필드를 보내던 외부 클라이언트는 전엔 조용한 200이었고 이제 400 invalid_json이다(의도한 계약이지만 눈에 보이는 변화).
- 다음 회차 후보(이번 변경이 만든 것 아님): openapi.yaml:896 `OptionalObject`의 `additionalProperties: true` vs handler의 `DisallowUnknownFields`, openapi.yaml:397 follow POST에 400 문구 부재, 한도 초과 본문의 400 vs 1bd4408의 413 선례.
- [러너 20:52] review approved — 리뷰 승인 (risk=low)
- [러너 20:52] pr created — https://github.com/hkjang/moina/pull/33
- [러너 21:00] ci passed — 검사 2개 모두 success
- [러너 21:00] merge done — 1d8cf40
- [러너 21:14] release published — v0.1.38
- [러너 21:20] assets verified — v0.1.38 자산 1개 (이전 v0.1.37: 1)

# 회차 노트 2026-09-29-035520-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:55] base pinned — main@7b2deec
- [러너 03:55] autonomy release — 

## 정찰 노트
- 포트폴리오(discovery.go:196/211)를 골랐다. admin_users.go:72 는 가치가 한 단계 높지만 계정 상태·세션 폐기에 붙어 있어 "보호 경로를 피하라" 는 지침과 부딪혀 차선으로 내렸다. 문서 대조·README 계열은 증거 설계가 M 이상이거나 값이 너무 작다.
- 코드로 확인한 것: portfolios.title 이 text NOT NULL(002:16), validate() 가 NUL 을 안 거른다(discovery.go:97-119), 라우트가 talents.write(router.go:56-57), openapi put 에 '400'·'500' / delete 에 '404'·'500' 누락(openapi.yaml:169-185).
- 추측으로 남긴 것: 일반 register() 클라이언트가 talents.write 를 갖는지 미확인 — registerSeller 헬퍼를 쓰라고 적어 우회시켰다. docker 가용성은 명령 승인 거부로 확인 실패(지난 4회차 29.7.2).
- 구현자가 조심할 것: deleteMyPortfolio 의 500 경로는 HTTP 로 만들 수 없다(uuid 파라미터뿐). 증명했다고 쓰지 말 것 — 2026-09-28 deleteMyWebhook 과 같은 처지다.
- 파일 예산: 프로덕션 2개(discovery.go, docs/openapi.yaml) + 테스트 1개. dist·web 은 건드리지 말 것.
- [러너 03:57] scout done — 포트폴리오 수정·삭제의 DB 저장 실패를 404 "포트폴리오를 찾을 수 없습니다." 가 아니라 500 으로 알리기 (

## 구현 노트
- 무엇을/왜: `updateMyPortfolio`·`deleteMyPortfolio`(discovery.go)의 `err != nil || tag.RowsAffected() == 0` 을 갈라, DB 저장 실패는 500(`portfolio_save_failed` / `portfolio_delete_failed`), 행 없음만 404 `portfolio_not_found` 로 보고한다. 바로 위 `createMyPortfolio` 가 이미 500 을 내던 비대칭을 없앤 것. `docs/openapi.yaml` 의 put 에 '400'·'500', delete 에 '404'·'500' 추가.
- 확신 없는 곳·검증 못 한 것: **`deleteMyPortfolio` 의 500 경로는 실행으로 증명하지 못했다** — 파라미터가 uuid 둘뿐이라 HTTP 입력으로 DB 오류를 만들 수 없다. 테스트가 확인한 것은 DELETE 의 404 회귀(랜덤 uuid → 404, 본인 것 → 204)까지다. 2026-09-28 `deleteMyWebhook` 과 같은 처지이고 같은 판단을 따랐다. 비평가는 여기부터 보면 된다.
- 일부러 하지 않은 것: `POST /me/portfolios` 의 openapi '500' 누락(핸들러 discovery.go:173 이 실제로 낸다)을 발견했지만 과제서 범위(`/me/portfolios/{id}`) 밖이라 빼고 ideas.json 에 pending 으로 적었다. `web/`·`internal/ui/dist` 는 프런트가 `error.message` 를 그대로 띄우므로 건드리지 않았다. 나머지 24곳의 같은 패턴도 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트 `TestIntegrationPortfolioUpdateSeparatesSaveFailureFromNotFound` 는 **DSN 이 있어야 돈다**(없으면 SKIP — DSN 없는 통과는 검증이 아니다). `integrationServer` 가 전역 `apiUnderTest` 를 쓰므로 병렬화 금지. 검증은 버릴 PostgreSQL 16(docker 29.7.2)에서 DROP/CREATE 한 깨끗한 DB 로 했고 httpapi 92.0초.
- [러너 04:03] brief accepted — 채택 — docker(29.7.2)가 가용해 수용 기준 1~3 을 모두 실제 HTTP→실제 DB 로 증명했고, 과제서가 미확인으로 남긴 것(`registerS
- [러너 04:03] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 판정 approve(low, 차단 없음). 원장을 믿지 않고 직접 돌렸다 — 버릴 PG16 컨테이너 + `go test -c` 바이너리로, main 코드 빌드는 `status=404 want=500`(portfolio_not_found) 로 FAIL, HEAD 는 PASS. 테스트가 수정 전 코드에서 통과하지 않음을 확인했다.
- 소스 마운트 후 `internal/httpapi` 전체를 깨끗한 DB 에서 통합 포함 실행해 PASS, vet·gofmt 무결, openapi.yaml YAML 파싱으로 응답 코드 확인. 커밋 범위는 3파일뿐이고 dist·web 오염 없음.
- security·legal 소견 없음: 404/500 갈림이 소유권과 무관해(비소유자는 여전히 RowsAffected()==0 → 404) 열거 오라클이 안 생기고, 오류 문구에 개인정보가 없으며 새 수집·의존성이 없다.
- 못 본 것 / 남는 우려: `deleteMyPortfolio` 의 500 경로는 나도 HTTP 로 만들 수 없었다 — 실측된 것은 PUT 경로뿐이니 릴리즈 노트를 그렇게 적을 것. 작업 트리에 커밋 안 된 `internal/ui/dist/*` 재빌드 산출물이 있으니 릴리즈가 담지 않도록 주의.
- 다음 회차: 같은 패턴 24곳 중 `POST /me/portfolios` 의 openapi '500' 누락과 admin_users.go:72 가 ideas.json 에 대기 중이다.
- [러너 04:09] review approved — 리뷰 승인 (risk=low)
- [러너 04:09] pr created — https://github.com/hkjang/Kkiit/pull/15

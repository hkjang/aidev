# 수리 요약 (시도 1)

- 지적은 맞았다: `WebOriginAllowed`가 읽는 것이 `clients`(internal/store/clients.go:355)임을 직접 확인했고, 그래서 5b260c7이 추가한 `a CORS origin could not be checked against the Realm's registered ones`(middleware.go:137)는 docs/operations.md:33 표가 다루는 바로 그 장애의 **세 번째** 줄이다 — "다음 두 줄" 은 이 커밋으로 거짓이 되었다.
- 고친 것: docs/operations.md 의 "두 줄 / 둘 다" → "세 줄 / 셋 다", 표에 한 행 추가(메시지 / 어디서=프로토콜 Endpoint 앞단 CORS 검사 / 답=상태·본문 불변이고 CORS Header만 빠짐, 증상은 미등록 Origin과 동일해 RP 설정보다 이 줄을 먼저 볼 것, Origin 원문 없이 `realm`만 남음). 코드·테스트는 한 줄도 건드리지 않았다.
- 재검증: `gofmt -l` 깨끗, `go build ./...`·`go vet ./...` OK, 실제 PostgreSQL로 `TestIntegrationCORSSaysWhenItCouldNotCheckAnOrigin` PASS(SKIP 아님을 -v 로 확인), `go test -race ./internal/httpserver` 전체 108.999s ok.
- 새 커밋 cea3c36 (rebase/amend 없음). `webui/dist/index.html`은 회차 시작부터 더러웠던 빌드 산출물이라 커밋하지 않았다.

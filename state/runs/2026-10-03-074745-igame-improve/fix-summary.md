비평이 맞았다. `docs/api.md:98` 은 "DB 일시 장애면 두 경로 모두 503 oidc_unavailable + 로그" 라고 단언했지만, callback 은 설정보다 `DELETE FROM oidc_flows … RETURNING` 을 먼저 실행하고 그 실패를 `pgx.ErrNoRows`(진짜 만료·무효 state)와 전송 계층 실패(DB 도달 불가)를 구분하지 않고 둘 다 `400 invalid_state` 로 접어 로그도 남기지 않았다. 재현: 새 테스트가 수정 전 `status 400 code "invalid_state"` 로 실패했다.

문서가 아니라 **코드**를 고쳤다(두 선택지 중 커밋의 목적을 실제로 달성하는 쪽). `internal/api/auth.go:321-335` 에서 `ErrNoRows` 만 `400 invalid_state` 로 남기고 나머지는 `serverError(503, oidc_unavailable, …, err)` 로 보내 원인을 로그에 남긴다. state 선소비 순서는 바꾸지 않았다. 아울러 `docs/api.md:98` 에 callback 의 `400 invalid_state` 가 DB 장애를 뜻하지 않음을 명시하고, 이제 사실과 달라진 `oidc_unavailable_pg_test.go` 머리 주석을 갱신했다.

검증(모두 이번 세션 실측): 신규 `TestOIDCCallbackReportsAnUnreachableDatabaseAsAnOutage` 가 **실DB 없이** 503 + 로그를 단언한다 — 비평가가 못 닫았던 바로 그 공백이다. `go test ./...` 전체 PASS, `make lint` PASS, `make test-race` PASS, `gofmt -l internal/api` 무출력. `IGAME_TEST_DSN` 없어 pg 회귀 4개는 여전히 skip(5432 에 정체 불명 PG 가 떠 있었으나 버려도 되는 PG17 이 아니어서 쓰지 않았다). 삭제·약화한 단언 없음.

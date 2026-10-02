# 수리 요약 (8f2c73c)

지적이 맞았다. 재현: `clients`를 `ACCESS EXCLUSIVE`로 잠근 트랜잭션을 열어 요청을 조회에 묶어 두고(`pg_locks`에 그 대기가 나타나는 것을 확인) 호출자가 연결을 끊으면, 취소가 `clientAuthUndecided`까지 올라와 `resso_client_auth_errors_total{stage="client"} 1` + `ERROR a client authentication could not be decided ... error="context canceled"`가 남고 두 Limiter 어느 쪽도 깎이지 않았다 — 정상인 데이터베이스에 대해 인증 없는 호출자가 올린 장애 경보다. 주석의 "no attempt is spent and no attempt is made"는 거짓이었다.

고친 방법: `clientAuthUndecided`(oidc.go:892)가 `r.Context().Err() != nil`인 시도를 **집계도 기록도 하지 않고** 미판정으로 돌린다(이 Route에는 요청 context에 기한을 거는 미들웨어가 없으므로 그 값은 호출자가 떠났다는 뜻뿐이다). 예산을 깎지 않는 쪽을 명시적으로 골라 주석에 근거를 적었다 — 답이 전달되지 않아 oracle이 없고, 반대로 깎으면 누구나 남의 `client_id`로 20번 보내 놓고 끊어 그 Client와 그 주소를 세 Endpoint에서 잠글 수 있다.

테스트: `TestIntegrationClientAuthSeparatesABrokenDigestFromACallerThatHungUp`(실제 PostgreSQL + httptest + 프로덕션 `New()`). (a) 깨진 저장 digest(`hmac-sha256$this is not base64`)로 **`stage="secret"` 분기를 실제로 지나며** 20회 집계·로그·`failures_total` 미집계·예산 무접촉을 단언한다(1회차에 비어 있던 분기). (b) 잠금으로 묶은 요청을 끊어 어느 계열에도 세지 않고 로그도 남기지 않으며 예산도 깎지 않는 것을 단언한다. 가드 없이는 (b)의 두 단언이 실패한다(확인함).

문서: `docs/operations.md:90`과 `README.md:192`에 이 두 번째 구멍과 그것이 닫힌 방식, 그리고 호출자가 떠난 순간에 겹친 진짜 장애는 함께 버려진다는 것을 적었다.

검증: `make lint` 0 issues + govulncheck 0 + ESLint 통과, `make test` 전부 ok(httpserver 114.2s), SKIP 0건(`--- SKIP` 집계 0).

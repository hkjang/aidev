# 수리 요약 (59b6c34)

비평가의 지적이 맞았다. 재현 확인: `ClientByIdentifier`에 `"bad-\xff\xfe-utf8"` 또는 `"bad-\x00-nul"`을 넘기면 PostgreSQL이 `invalid byte sequence for encoding "UTF8" (SQLSTATE 22021)`을 돌려주고 `ErrNotFound`가 아니므로, 인증 없는 호출자가 `client_id` 한 필드로 `resso_client_auth_errors_total{stage="client"}`와 ERROR 로그를 한도 없이 올리며 두 Limiter를 영구 우회했다(수정 전 실행에서 20→21→22로 오르고 `client_auth_failures_total`은 21에 멈춘 것을 확인).

고친 방법: 저장소 경계에서 막았다 — `store.ClientByIdentifier`가 `client_id` text 컬럼에 담길 수 없는 식별자(UTF-8 불량 또는 NUL)를 조회 전에 `ErrNotFound`로 판정한다(`storableIdentifier`). `auth_time` clamp 선례와 같은 모양이고, 같은 구멍을 가진 authorization 엔드포인트·logout 호출자까지 한 번에 닫힌다. 응답은 무변경(401 `invalid_client`), 이제 등록되지 않은 식별자와 똑같이 집계되고 예산을 깎는다. `oidc.go`의 거짓 주석과 `docs/operations.md:90`을 사실에 맞게 고쳤다(README.md:192는 그 자체로 참이 되어 그대로 뒀다).

검증: `make lint` 0 issues, `go test -race ./internal/store ./internal/httpserver` 통과(84s/127s — 연동 테스트 실제 실행), `go vet ./...` + 나머지 패키지 통과. 프로덕션 가드만 되돌리면 새 단언 (g)와 store 단위 테스트가 실패하는 것을 확인했다.

# 수리 요약 (시도 1)
- 문제: 9d8a6d5가 `api/openapi.yaml:189` PATCH /posts/{postID} responses에서 '403'을 지웠으나, 이 route는 `auth.With(s.requirePermission("posts:write"))`(server.go:171) 뒤라 권한 없음 → `403 forbidden`(server.go:575), 쿠키 세션 CSRF 불일치 → `403 invalid_csrf`(server.go:563)가 실제로 나옵니다. 비평 지적이 맞음.
- 고침: responses에 '403'을 400·409와 함께 되돌림(3b95053). 형제 route POST/DELETE /posts와 같은 관례. 코드·테스트는 손대지 않음.
- 검증: `node scripts/check-openapi-routes.mjs` → 120개 통과 · `bash scripts/check-runtime-contract.sh` → 4개 통과 · `go build ./... && go vet ./internal/httpapi/ && go test -race ./internal/httpapi/` → ok (integration은 DSN 없어 skip, 이 커밋은 yaml만 바꿔 영향 없음).

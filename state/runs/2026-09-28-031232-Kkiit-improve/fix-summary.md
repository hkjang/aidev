# 수리 요약 (시도 1 거절 대응)

- 문제: 26ed72f 가 `deleteMyWebhook`(webhooks.go:202)에 500 `webhook_delete_failed` 를 새로 만들었는데 docs/openapi.yaml 의 delete 블록은 '204' 하나뿐이었다(기존 404 도 누락). 지적은 맞았다.
- 고침: docs/openapi.yaml:277-278 의 `delete.responses` 에 '404'·'500' 두 줄만 추가. 코드는 건드리지 않았다.
- 확인: YAML 파싱 OK(delete.responses = 204/404/500, 편집 위치가 웹훅 delete 임을 diff 로 확인), `gofmt -l cmd internal` 무출력, `go vet ./cmd/... ./internal/...` 통과, `go test ./cmd/... ./internal/...` 전부 ok.
- 미검증: KKIIT_TEST_DSN 이 없어 통합 테스트는 SKIP 됐다(이번 변경은 문서 전용이라 영향 없음).
- 남은 판단: DELETE 도 invalid_id 로 400 을 낼 수 있으나 이는 모든 `{id}` 경로에 공통인 기존 문서 누락이라 이번 수리 범위 밖으로 뒀다.

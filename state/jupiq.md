# jupiq 자율 개선 기록

## 2026-09-03
- 선택: OpenAPI 문서와 등록 경로 계약 검증 테스트 추가 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `register*` 함수 시그니처를 `*http.ServeMux`에서 최소 `router` 인터페이스로 바꿔 테스트가 등록 경로를 수집할 수 있게 하고, `internal/api/openapi_contract_test.go`에서 openapi.yaml의 paths와 양방향(경로→문서, 문서→경로)으로 대조하도록 했다. probe·별칭·항상 405인 승인 쓰기 경로는 이유를 적은 예외 목록으로 관리하며, 이 검증으로 드러난 누락 `GET /auth/oidc/callback`을 문서에 추가했다. 임시로 가짜 경로를 등록해 테스트가 실제로 드리프트를 잡는지 확인했고, `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(16파일 49개) 모두 통과했다.
- 보류 아이디어: `internal/api/helpers.go`의 사용되지 않는 `parseTimeQuery` 제거(boundedTimeRange로 대체됨) / `internal/collector` 커버리지 8.5% 보강 / `serveSPA`의 해시 자산에 장기 Cache-Control 부여 / 로그인 리미터 `succeeded`가 ip 키를 정리하지 않는 동작에 대한 테스트·문서화
- 릴리즈: v1.3.0 (2026-09-03)

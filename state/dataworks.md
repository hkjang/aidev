
## 2026-09-02
- 선택: API surface audit 복구 + 미문서화 라우트 86건 카탈로그 등록 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `cmd/api-surface-audit` 가 리브랜딩으로 사라진 `cmd/vibe/main.go`·`sdk/typescript/vibe.ts` 를 읽으며 경고 후 빈 문자열을 반환해, CLI/SDK 계약 검사가 0건 기준으로 항상 통과하는 무력 상태였다. 소스 경로를 정정하고 읽기 실패를 즉시 실패로 바꿨으며, mux 에 등록됐지만 `apiEndpoints` 에 없어 `/openapi.json` 에서 빠져 있던 라우트 86건을 핸들러가 실제 허용하는 메서드대로 추가했다. 같은 불변식을 `go test ./...` 안의 회귀 테스트(라우트 커버리지·죽은 항목·항목 형식)로 고정했고, 가짜 라우트를 넣어 가드가 실제로 실패하는 것까지 확인했다. 검증: `go build ./...`, `go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0 종료.
- 보류 아이디어: GitHub Actions CI 부재(build/vet/test/api-surface-audit 게이트 추가) / `go vet ./...` 잔여 4건(테스트의 err 미검사 3건, json 태그 중복 1건) 정리 / `gofmt -l` 미정렬 64개 파일 일괄 포맷 / `parseWindow` 가 음수·과대 duration 을 그대로 받아 미래 시각 since 를 만드는 엣지케이스 / `internal/dataworks` 패키지 테스트 커버리지 보강
- 릴리즈: v0.9.36 (2026-09-02)

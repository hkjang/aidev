# 수리 요약 (수리 시도 1)

- 비평 두 건 모두 맞았다. 실측 재현: `/m%6Fmento/collect/%2E%2E%2F%2E%2E%2Fadmin` → 수집기 `r.RequestURI` = `/base/collect/../../admin` (접두사 이탈 그대로), `/%6Domento/collect/a%2Fb` → `/base/collect/a/b`. ServeMux 가 디코드된 경로로 매칭하므로 `TrimPrefix(EscapedPath(), …)` 가 no-op 이 되고 `EscapedPath()` 가 Path 를 재escape 해 수정 이전 동작으로 degrade 했다.
- 고친 방법: `momentoProxy` 가 `r.URL.EscapedPath()` 에 리터럴 `"/momento/"` 접두사가 없으면 404(gate 와 같은 메시지)로 거절하고, Rewrite 는 그 검증된 `escapedPath` 를 쓴다(tracking.go:223-233, 244-247). Path/RawPath 가 항상 유효한 짝으로 남는다.
- 테스트: 과장된 보증 대신 접두사 인코딩 변형 2건을 서브테스트로 추가 — 404 이고 **수집기가 요청을 전혀 보지 않음**까지 단언한다(tracking_test.go:478-501). 프로덕션 검사만 되돌리면 두 서브테스트가 202 로 실패하는 것을 확인했다(인과 확인).
- 검증: `bash scripts/verify.sh` exit 0(프런트 59/59, 500kB 청크 경고는 기존), `go test ./... -count=1`·`go test ./internal/httpapi -race -count=5`·`go vet ./...`·`gofmt -l internal/` 모두 통과. 테스트 삭제·단언 완화·검증 명령 수정 없음.
- 미검증: `/momento%2Fcollect/...` 는 핸들러에 도달하지 않음(SPA fallback 200)을 실측했으므로 세그먼트 경계 인코딩은 비해당. 실제 Momento 수집기의 `%2F` 해석, 브라우저 실환경·이미지 E2E·CI 는 여전히 미실행.

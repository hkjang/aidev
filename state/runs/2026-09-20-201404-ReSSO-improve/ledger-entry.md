## 2026-09-20
- 선택: UserInfo POST의 폼 본문을 기존 프로토콜 POST와 같은 1MiB로 제한 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: userInfo POST ParseForm 직전에 http.MaxBytesReader(w, r.Body, 1<<20) 한 줄을 추가해 인코딩된 폼이 1MiB를 초과하면 기존 400 JSON invalid_request로 거절하며 호환표에 상한을 명시했다(커밋 3060a9b). 실제 PostgreSQL·서명 키·세션·IssueUserTokens·New(...).Handler()·httptest HTTP 서버로 본문/헤더 토큰 × 고정 길이/chunked × 정확히 1MiB/1MiB+1 여덟 경우를 검사하여 수정 전과 제한 재제거 시 초과 네 경우가 모두 200으로 실패하고 수정 후 모두 통과함을 확인했으며, 기존 GET·POST 인증/중복/쿼리/불량 토큰 계약도 통과했다. make lint(golangci-lint 0 issues, govulncheck 호출 경로 취약점 0, ESLint)·make test(Go race 전 패키지, 연동 SKIP 0, go vet, 프런트 29파일/161테스트, 빌드)·git diff --check 통과; 빌드가 변경한 webui/dist/index.html은 복원했다.
- 보류 아이디어: README 통합 테스트 준비를 test-services.sh로 통일 (가치 2 / 위험 1 / 작업량 S)
- 보류 아이디어: 잘못된 logout POST 폼을 세션 종료 전에 거절 (가치 2 / 위험 2 / 작업량 M)
- 보류 아이디어: 인가 접근 로그에 client_id 남기기 — 인가 Endpoint일 때만, 길이 제한 (가치 2 / 위험 1 / 작업량 S)
- 보류 아이디어: login 409 request_already_used의 직전 로그인 쿠키 삭제를 실제 경쟁 테스트로 확인 (가치 2 / 위험 2 / 작업량 M)
- 과제서: 채택 — UserInfo만 Reader 제한이 누락되었고 실제 초과 요청의 200 응답을 재현해 과제서의 근거와 수용 기준을 확인했다.

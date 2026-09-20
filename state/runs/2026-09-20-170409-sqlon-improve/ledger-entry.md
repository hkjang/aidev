## 2026-09-20
- 선택: 프로파일 카탈로그 GET API 5개의 인증·프로파일 권한 배선 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 실제 로그인→Register mux→doReq 테스트로 수정 전 익명/무효 인증 GET 5개가 200, 타인 비공개 상세가 200, 인증 사용자의 목록이 비는 결함을 재현하고 requireActor 및 목록 withUser 배선을 수정했다(커밋 f213052). 소유·shared·use grant·admin, 실제 발급 MCP 키·폐기 키·메타 모드 마스터 토큰, 목록/count와 상세 권한 일치, 임시 JSON summary/content, 실제 PostgreSQL 드라이버의 제한된 로컬 Unix 소켓 연결 실패, 단독 모드 AdminToken 유무를 검증했으며 지정 회귀 테스트·go vet ./...·go test ./... -count=1·CLI 빌드·git diff --check가 모두 통과했다. 라이브 DB 성공/브라우저는 미검증이며 요청 technology 스킬 3개는 도구·로컬·리소스에서 찾지 못했고 숨김 경로에서 뒤늦게 발견한 superpowers TDD/systematic-debugging/verification-before-completion을 보조로 읽어 점검했다.
- 보류 아이디어:
  - requireAdmin 사용자 컨텍스트·admin 감사 actor 기록 (가치 3 / 위험 1 / 작업량 M) — 선행 actor 변경 통합 여부를 먼저 확인.
  - docs/auth.md·rest-api.md·security.md 정본 작성 (가치 3 / 위험 1 / 작업량 M) — 모두 0바이트, 실제 API 소스 확인 필요.
  - GitHub Actions go vet·go test CI (가치 3 / 위험 1 / 작업량 S) — workflows 보호 경로, 별도 회차.
  - 빈 docs/README.md 문서 색인 (가치 3 / 위험 1 / 작업량 S) — 실제 존재하는 비어 있지 않은 가이드만 연결.
- 과제서: 채택 — 현 HEAD의 인증 및 컨텍스트 누락을 실제 HTTP로 재현했고 지정된 5개 GET 경계만 수정하여 수용 기준을 충족했다.

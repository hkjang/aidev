## 2026-09-21
- 선택: 비동기 쿼리 잡의 10분 만료를 조회·취소 경로에서도 적용 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: jobView·cancelJob의 기존 잠금 안에서 prune을 호출해 추가 제출 없이도 완료 후 10분이 지난 잡을 삭제하고 404를 반환하도록 수정했다(c941cb9); FinishedAt 기준·엄격한 > 비교·TTL·기존 권한 계약은 유지했다. 실제 로그인→Register mux→POST 제출→백그라운드 실행 테스트에서 수정 전 첫 GET의 SQL 노출/200과 첫 취소의 200을 각각 재현했고, 수정 후 404·저장소 제거·미만료 소유자/admin·401/403/404·실제 TCP 연결 중 오래된 잡 보존을 확인했다. go test ./internal/mcp -run 'TestAsync' -count=1, 대상 -race(16.164초), go test ./... -count=1, go vet ./..., go build ./..., git diff --check 모두 통과했다.
- 보류 아이디어:
  - requireAdmin 사용자 컨텍스트·감사 actor (가치 3 / 위험 1 / 작업량 M) — 선행 변경 미통합, 재구현하지 않음.
  - docs/auth.md·rest-api.md·security.md 정본 작성 (가치 3 / 위험 1 / 작업량 M) — 실제 라우트 확인 후 별도 진행.
  - GitHub Actions go vet·go test CI (가치 3 / 위험 1 / 작업량 S) — workflows 보호 경로로 보류.
  - 빈 docs/README.md 문서 색인 (가치 3 / 위험 1 / 작업량 S) — 차선 후보 유지.
- 과제서: 채택 — 현 코드의 결함을 실제 HTTP 회귀로 재현했고 두 접근 경로에 최소 수정으로 만료 계약을 적용했다.

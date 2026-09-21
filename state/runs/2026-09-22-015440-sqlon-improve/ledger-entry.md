## 2026-09-22
- 선택: 프로파일 PUT에서 visibility 생략 시 기존 공개 범위를 보존 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 실제 로그인 쿠키와 Register mux를 통과하는 HTTP 회귀에서 shared 프로파일의 생략/빈 값 PUT 6건이 private 저장·무grant 사용자 목록 누락을 일으킴을 먼저 확인하고, 생성에만 private 기본값을 적용하며 수정에서는 기존 visibility를 권한 검사·저장·응답에 동일하게 사용하도록 고쳤다(ccbd0e7). 52개 테스트 경우로 소유자/admin/manage/use/익명, shared/private, 생략/빈 값/명시/잘못된 값, 실제 name 저장과 거절 시 전체 정의 불변을 검증했으며 지정 표적 테스트·go test ./... -count=1·go vet ./...·go build ./...·git diff --check가 통과했다. 요청 technology 스킬 3개는 제공 도구·로컬 검색에서 찾지 못해 고유 절차/반환 형식은 미확인이고 실제 PG 통합·브라우저는 검증하지 않았다.
- 보류 아이디어:
  - requireAdmin 사용자 컨텍스트·admin 감사 actor (가치 3 / 위험 1 / 작업량 M) — 선행 변경 미통합, 재구현하지 않음.
  - docs/auth.md·rest-api.md·security.md 정본 작성 (가치 3 / 위험 1 / 작업량 M) — 실제 라우트 검증 후 별도 진행.
  - GitHub Actions go vet·go test CI (가치 3 / 위험 1 / 작업량 S) — workflows 보호 경로로 보류.
  - 빈 docs/README.md 문서 색인 (가치 3 / 위험 1 / 작업량 S) — 주과제 결함 재현으로 차선 미선택.
- 과제서: 채택 — 현재 코드에서 결함을 실제 HTTP로 재현했고 지정된 3개 파일만 수정하여 생성 기본값과 공개 범위 변경 권한 계약을 유지했다.

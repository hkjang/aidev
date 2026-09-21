## 2026-09-21
- 선택: notify Telegram·Slack 전송 실패 로그의 자격증명 마스킹 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 두 전송 오류 로그에서 웹훅 전체 URL과 Telegram 토큰을 치환하고 요청 URL·본문·비동기 동작·sync.Once는 보존했다(커밋 1d5ecc68). 실제 공개 Send와 닫힌 httptest 서버를 거치는 회귀 테스트가 수정 전 두 플랫폼 비밀 노출로 실패하고 수정 후 통과했으며, notify -race·gofmt 무출력·전체 vet/build/test를 확인했다. 요청된 technology:completion-verification·systematic-debugging·test-driven-development는 사용 가능한 도구 및 로컬 SKILL.md 검색에서 찾지 못해 고유 절차·반환 형식은 미적용하고 과제서의 TDD·검증 절차를 수행했다.
- 보류 아이디어:
  - scripts/check.sh 로컬 Go 검증 진입점 (가치 3 / 위험 1 / 작업량 S)
  - GitHub Actions CI 추가 — workflow 권한 미확인 (가치 4 / 위험 3 / 작업량 S)
  - 스윙 청산 시 진입 다음 봉부터 순회 — 매매 계약 확인 필요 (가치 3 / 위험 3 / 작업량 M)
  - Slack HTTP 4xx/5xx 응답 경고 보강 — 이번 범위와 분리 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 실제 두 오류 분기의 원문 로깅과 신규 회귀의 비밀 노출 실패로 과제서 근거가 확인됐다.

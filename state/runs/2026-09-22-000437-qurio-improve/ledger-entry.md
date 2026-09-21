## 2026-09-22
- 선택: make lint에 읽기 전용 Go 포맷 검사 연결 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: check-go-format을 .PHONY 타깃 및 lint 선행 조건으로 추가해 네 소스 루트의 포맷 불일치와 gofmt 구문 오류를 실패로 전달하고, help/README에 검사·수정 방법을 설명했다. 실제 make/gofmt 임시 복사본 34건(정상 2건, 네 루트별 일반/테스트/integration 파일 및 구문 오류의 직접 검사·lint 실패)에서 모든 소스 파일 SHA-256 불변과 vet/npm 전 중단을 확인했으며 npm ci 후 make lint, make help, make -n lint, 부분 Go 테스트, go build ./..., git diff --check가 통과했다. 커밋 a432f3f; 요청한 technology 회사 스킬과 Skill 도구는 검색에서 찾지 못해 그 절차·반환 형식은 미확인이고 사용자 지정 절차로 완료했다.
- 보류 아이디어: TestUserProfilePreferencesPostgres cleanup 순서 수정 (가치 3 / 위험 2 / S) — 실제 DB 잔존 재현 필요.
- 보류 아이디어: store 통합 테스트의 명시적 DB 연결 오류 Skip 판정 개선 (가치 3 / 위험 2 / M).
- 보류 아이디어: mail 선행 머지 후 만료 승인 알림 훅 연결 (가치 3 / 위험 1 / S).
- 보류 아이디어: Keycloak RP-initiated logout 연동 (가치 3 / 위험 3 / M).
- 과제서: 채택 — 현재 lint의 포맷 검사 누락과 네 루트의 드리프트 0건을 확인했고 지정 범위 내에서 수용 기준을 충족했다.

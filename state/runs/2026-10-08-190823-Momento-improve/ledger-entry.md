## 2026-10-08
- 선택: Custom Dimension 등록·갱신 실패를 한국어로 안내한다 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: describeDimensionError를 추가해 서버 여섯 코드와 REQUEST_FAILED를 한국어로 안내하고 DimensionsAdmin 저장 Alert 한 곳에 연결했다(프로덕션 2파일+테스트 1파일); INVALID_DIMENSION은 두 라벨과 첫 글자·허용 문자·1~128자 규칙을 함께 안내하며 500의 원문은 중립 본문 아래 detail로만 보존한다. 순수 테스트 11개를 먼저 추가해 원문 반환 구현에서 9개 실패를 확인했고, 구현 후 270/270 통과 및 대조 구현 복원 시 같은 실패를 재확인했으며 실제 프로덕션 앱의 HTTP 오류 9사례와 편집·성공 초기화·목록 재조회·삭제·버튼 조건을 포함한 브라우저 10/10 시나리오가 통과했다. web·SDK 최종 게이트(audit 포함, SDK 29/29)와 Go race·vet·govulncheck가 exit 0이며 DB 통합은 DSN 미설정으로 skip, Docker와 실제 DB 장애 재현은 미실행이다; Vite 청크 경고 및 govulncheck 비호출 취약점 보고와 하네스의 두 실패·수정 이력은 journal에 남겼다.
- 실패 재현: `not ok 3 - Custom Dimension INVALID_DIMENSION 는 한국어로 고칠 곳이나 다음 행동을 안내한다` / `+ 'name and property_key must use letters, numbers, underscore, dot, or hyphen'` (web-red-identity.log, npm test exit 1; 새 11건 중 9건 실패·기존 259건 통과).
- 보류 아이디어: 설정 저장 실패 한국어 안내 (가치 2 / 위험 2 / 작업량 M) — 여러 설정 그룹 계약 확인 필요.
  Custom Dimension 입력 칸에 규칙 helperText 안내 (가치 2 / 위험 1 / 작업량 S) — 이번 오류 안내와 분리.
  보존 정책 빈 입력의 Number("")→0 변환 수정 (가치 2 / 위험 2 / 작업량 M) — 빈 문자열 상태·payload 분리 필요.
  README 개발 명령의 cd 누적 수정 (가치 3 / 위험 1 / 작업량 S) — 차선 유지, 이번 범위 제외.
- 과제서: 채택 — 여섯 서버 코드·PropertyKeyPattern·upsert·원문 저장 Alert가 현재 코드와 일치하여 지정한 3파일 범위로 수용 기준을 구현·검증했다.

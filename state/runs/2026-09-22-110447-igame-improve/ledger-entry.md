## 2026-09-22
- 선택: bootstrap 암호의 bcrypt 72바이트 상한을 DB 초기화 전에 검증 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: bc8791d에서 config.Load에 최소 12 rune 검사 이후 최대 72 UTF-8 바이트 검사를 추가하고 README·오프라인 설치 설명을 맞췄다. t.Setenv 기반 경계 테스트의 수정 전 실패와 수정 후 통과, 원문·공백·분해형 유니코드 보존 및 오류의 암호 미포함을 확인했고 실제 cmd/igame 기동 5종으로 DB 파싱 이전 거부와 정상 길이의 기존 경로를 검증했다. 지정 Go 테스트·vet·빌드·릴리즈 계약·diff 검사 통과; IGAME_TEST_DSN 미설정으로 PG 테스트는 skip이며 요청 technology 조직 스킬은 도구·로컬 검색에서 미발견으로 미적용이다.
- 보류 아이디어:
  - README RealmGuard 서버 재현 설명 정합성 개선 (가치 2 / 위험 1 / 작업량 S)
  - API PostgreSQL fixture의 테스트별 스키마 격리 (가치 3 / 위험 2 / 작업량 M)
  - Migrate context 취소의 롤백·재시도 계약 검증 (가치 3 / 위험 1 / 작업량 M)
  - architecture 비밀 저장·키 회전 설명 정합성 검토 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 현재 Load·main·EnsureBootstrapAdmin 배선이 정찰 근거와 일치하여 지정한 최소 범위로 구현했다.

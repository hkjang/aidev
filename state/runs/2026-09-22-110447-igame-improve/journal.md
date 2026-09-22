# 회차 노트 2026-09-22-110447-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:04] base pinned — main@abd8579
- [러너 11:04] autonomy release — 

## 정찰 노트
- bootstrap 72바이트 사전 검증 선택: 실제 기동에서 초과 암호가 DB 파싱까지 통과함을 확인했고, 정책 변경·실DB 동기화·PDF 작업 없이 작은 수정으로 끝난다.
- Go 전체 테스트·릴리즈 계약·bcrypt 초과 길이 테스트 통과; 실DB 전체 bootstrap, Web/SDK, 외부 감사·원격 CI는 미확인.
- 암호 원문·공백·다국어를 보존하고 12 rune 최소/72 byte 최대를 구분한다. auth/migrations/workflows 및 지난 audit-release 작업 재구현 금지.
- 조직 스킬 3개는 도구·로컬 경로에서 미발견으로 절차 미확인. 차선은 소스와 대조한 README 서버 재현 설명 정정이다.

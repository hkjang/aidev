## 2026-09-22
- 선택: 수정 과제 — 지정 릴리즈 실패의 엔진별 스킬 입력 복구 (가치 4 / 위험 2 / 작업량 M)
- 결과: 실패
- 요약: 과제서의 외부 러너 편집 금지와 최초 릴리즈 계약 미확보가 유지되어 구현하지 않았다. Skill 호출 도구가 없어 요청된 technology 세 원본을 직접 읽고 registry·run_agent/run_codex·release-prompt 및 Release/ReleaseSafety와 sim 실행 경로를 확인했으나 실제 자식 실패 인과는 미입증이다. 이번 테스트·빌드·실제 자식 재현·수정 전후 회귀는 미실행이며 코드·커밋·저장 failed JSON 변경 없이 필수 기록만 작성했다; 기존 기준선이나 이 기록은 개선 성과가 아니다.
- 보류 아이디어:
  - 외부 러너 소유 환경의 registry 기반 스킬 전달 및 실제 자식 회귀 검증 (4/2/M, pending)
  - 최초 릴리즈 대상 패키지·버전·태그·노트·자산 계약 확보 (4/3/M, pending)
  - AdvancedPolicyView 미저장 편집 보존 (3/2/M, pending)
  - Catalog·Directory·Content 기존 테스트 wrapper 정리 (2/1/S, pending)
- 과제서: 채택 — 지정 과제와 외부 편집 금지를 유지하며 pending/blocked로 기록한다; 현 앱 회차의 실행 가능한 수정으로 재승인하지 않고 세 수용 기준 미충족을 명시한다.

## 2026-09-21
- 선택: 수정 과제 — 스킬 전달 복구의 외부 이관 및 최초 릴리즈 관례 차단 기록 (가치 4 / 위험 2 / 작업량 M)
- 결과: 변경없음
- 요약: 수정 과제: 스킬 전달 복구, 릴리즈 관례 미확정으로 전체 릴리즈 미완료 — 스킬 전달 복구 자체도 미완료다. 실제 원본 SKILL.md는 존재하고 run_agent의 Claude용 Skill 안내가 원본 경로 없이 run_codex로 전달되는 코드를 재확인했지만, registry의 builder.surface와 COMPANY.md 15·32행이 쓰기를 해당 회차 worktree로 제한하며 현재 run.json의 프로젝트는 aiportal-front-admin이므로 외부 aidev 러너 소유 작업 환경으로 이관한다. bash -n(exit 0), test_gate.py Release 5개 및 전체 27개 통과(기존 ResourceWarning), 이전 release.json에 대한 gate.py release는 exit 1 / ok:false / state:failed로 재현했으며, 실제 Codex 폴백 통합 재현·신규 테스트·수정·커밋·앱 빌드·실제 서버/UAT는 수행하지 않았다.
- 보류 아이디어:
  - 외부 aidev 소유 작업 환경에서 registry 기반 원본 경로 전달 수정 및 실제 run_agent→Codex 자식 프로세스 회귀 검증 (4/2/M)
  - 최초 릴리즈 버전·태그·노트 정책의 운영 결정: 루트 0.0.0/V2 0.1.0과 태그 부재만으로 결정하지 않음 (4/3/M)
  - AdvancedPolicyView 미저장 편집 보존 (3/2/M)
  - Catalog·Directory·Content 기존 테스트 wrapper 정리 (2/1/S)
- 과제서: 차선 — 결함 근거는 현재 코드와 일치하나 회차의 쓰기 표면은 앱 worktree이므로 과제서가 명시한 외부 이관 경로를 선택했다; 수정 완료나 릴리즈 성공을 뜻하지 않는다.

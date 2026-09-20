## 2026-09-21
- 선택: [수정 과제] 릴리즈 입력 복구 — 회사 스킬 탐색 안내와 버전 근거 정본화 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 수정 과제 — 릴리즈 입력 복구, 전체 릴리즈 미검증. AGENTS.md에 활성 세션이 요구한 스킬만 Skill 도구 우선·부재 시 실제 원문 읽기·누락 시 이름/경로/미확인 기록을 안내하고, 두 문서의 초기 릴리스 표를 docs/RELEASE.md로 통일해 기존 0.0.1/2025-01-01 기재와 실제 릴리스 미확인 상태, 현재 세 값 0.0.0 및 최초 lock 버전 필드 부재를 보존했다. 지정 Git/JSON 재조회와 실제 스킬 5개 읽기, 정본 명령 실행, 링크 대상 읽기, bash -n 및 git diff --check가 통과했으며 다음 버전 정책은 미해결이다.
- 보류 아이디어:
  - 외부 Codex 폴백 원문 경로 전달: 5/2/M, 외부 aidev 소유이며 동적 검증 필요.
  - useAppList 비배열 캐시 방어: 3/2/S, 이번 문서 과제와 무관하여 보류.
  - globalLoading 병렬 요청 참조 카운트: 3/3/S, 호출 짝 감사 선행.
  - 테스트 가이드의 없는 문서 링크 정리: 2/1/S, 별도 문서 과제로 보류.
- 과제서: 채택 — 실제 원문 접근 가능 여부와 Git/JSON 결과가 과제서에 일치하며 AGENTS 추가를 금지하는 저장소 규칙은 발견되지 않았다.

검증 및 스킬 반환 기록:
- 재현/원인: 수정 전 두 표의 단정적 0.0.1 기재와 현재 0.0.0, 최초 lock 필드 부재를 직접 조회했다. 실패 release.json과 외부 run.sh/release-prompt.md/test_sim.py의 run·HappyPath·Agents 및 sim/run_sim.sh를 읽어 Skill 안내에 원문 경로가 없음을 확인했다. 앱 CI 실패를 입증하는 로그는 없어 앱 빌드 결함으로 분류하지 않았다.
- 수정: 원문 탐색 진입점과 버전 근거 정본화, 문서 4개만 변경. 과제서 체크포인트 1~4 완료. CI·러너·headcount·버전·태그·원격 서비스는 변경하지 않았다.
- 실행: git rev-parse --is-shallow-repository → false; git tag --sort=-creatordate → 빈 출력; git log --oneline -60 및 파일 이력 조회 성공; 지정 Python → CURRENT 0.0.0 0.0.0 0.0.0, 최초 lock ABSENT ABSENT, 기존 두 표 확인, READ OK 5개. 정본의 두 bash 블록도 실제 실행하여 종료 0; 링크 대상 7개 읽기 성공; bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh 및 git diff --check 종료 0.
- 스킬: Skill 도구 없음. technology:completion-verification/systematic-debugging/test-driven-development의 실제 SKILL.md를 읽고 적용했다. 파일 읽기를 Skill 호출로 기록하지 않는다.
- 테스트 추가 없음: 실행 코드가 없는 문서 변경이므로 문구 문자열 회귀 테스트는 만들지 않았다. 앱 npm ci/test/build, 전체 sim, 실제 모델/releaser, 동적 quota 폴백은 미실행. Red/Green 및 수정 되돌림에 의한 릴리즈 인과 검증도 미실행이며 전체 장애 해결 완료로 보고하지 않는다.
- 잔여: 다음 버전 증가·커밋·태그 관례는 승인된 정책 또는 실제 릴리즈/배포 증거가 필요하다. 버전 파일 존재로 이력 전무 skipped 조건 불충족; 독립된 동일 실패 2회는 미확인.

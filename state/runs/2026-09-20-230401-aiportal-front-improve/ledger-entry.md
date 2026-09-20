## 2026-09-20
- 선택: [수정 과제] Codex 폴백 스킬 경로·버전 증거 기록 및 aidev 담당 이관 (가치 5 / 위험 2 / 작업량 M)
- 결과: 변경없음
- 요약: 현재 작업 표면은 aiportal-front@dd65af7이며 외부 aidev 구현 회차로 이관되지 않아, 과제서의 명시적 외부 수정 금지에 따라 러너·앱 코드와 테스트를 수정하지 않았다. aidev/bin/run.sh:245~340, agents/registry.json, release-prompt.md, tests/test_sim.py 및 tests/sim/run_sim.sh를 읽어 Claude용 Skill 안내의 Codex 전달과 원문 경로 누락을 정적으로 재확인했고, release 2개·scout 3개 원문 파일의 읽기 가능 여부 및 package/lock 0.0.0 대 문서 0.0.1 모순도 확인했다. 이번 실행은 bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh 통과와 파일 확인뿐이며, gate 27건 통과는 앞선 정찰 증거이고 신규 폴백 재현·Agents·전체 sim·앱 test/build는 미실행이므로 수정 완료나 릴리즈 성공/skipped를 주장하지 않는다.
- 보류 아이디어:
  - Codex 엔진별 원문 경로 전달 및 quota 폴백 회귀: aidev 별도 구현 회차 필요; pending 유지.
  - 릴리즈 버전 근거 불일치: 실제 배포 기록과 증가 관례 확인 전 미해결.
  - useAppList 비배열 캐시 방어: 이번 우선 과제와 무관하여 보류.
  - globalLoading 병렬 요청 참조 카운트: 호출 짝 감사 전 보류.
- 과제서: 차선 — 지정 구현 파일은 외부 aidev 저장소에 있고 이번 회차에서 직접 수정하지 말라는 지시가 유지되어, 허용된 회차 기록으로 이관 근거를 남겼다.

### 이관 근거와 검증 한계
- Skill 호출 도구는 제공된 callable 도구 목록에서 찾지 못했으나 technology:completion-verification, technology:systematic-debugging, technology:test-driven-development의 실제 SKILL.md를 읽고 적용했다. 앞선 정찰 설명과 달리 systematic-debugging과 test-driven-development에는 Return contract 절이 존재한다.
- 재현: 실패 기록은 ../2026-09-20-211415-aiportal-front-improve/release.json의 failed. 이번 동적 quota 폴백 재현은 미실행이므로 정적 전달 결함과 실패 기록의 일치를 인과관계 회귀 검증 완료로 취급하지 않는다. 동일 원인 두 실패는 독립 확인하지 않았다.
- 원인 근거: run_agent가 dept_note를 prompt에 붙인 후 run_codex에 그대로 전달한다. run_codex는 registry/HEADCOUNT_DIR/SKILL.md 경로를 안내하지 않는다. tests/sim/bin에는 claude와 gh만 있고 codex 스텁이 없다.
- release 원문: /mnt/c/Users/USER/projects/headcount/plugins/marketing/skills/product-launch/SKILL.md 및 /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/release-and-deployment/SKILL.md — 모두 읽기 확인.
- scout 원문: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md — registry의 3개 모두 읽기 확인.
- 수정 및 추가 테스트: 없음. 외부 범위 차단 때문에 Red/Green 및 수정 되돌림 검증 미실행. 이관 후 가짜 claude/codex로 release/scout 및 직접 review 호출, HEADCOUNT_DIR 재정의·공백·NO-HEADCOUNT·agents.headcount=false·활성 상태 누락 이름/경로 보고를 검증해야 한다. 실제 모델 호출 금지.
- 이관 후 검증: aidev 루트에서 bash -n bin/run.sh, PYTHONDONTWRITEBYTECODE=1 python3 tests/test_gate.py, 신규 등록 폴백 테스트 단독 실행, PYTHONDONTWRITEBYTECODE=1 python3 tests/test_sim.py Agents, PYTHONDONTWRITEBYTECODE=1 python3 tests/test_sim.py. 기존 gate 27건을 포함해 확인하되 이번 결과로 대체하지 않는다.
- README.md:438 및 docs/01-시작하기.md:359는 0.0.1(2025-01-01), package.json 및 lock 두 버전 필드는 0.0.0. release-prompt.md 절차 5는 버전 파일도 없는 경우만 skipped를 허용하므로 스킬 경로 복구와 별개 장애다.
- 앱 CI 잡 실패를 증명하는 로그는 없고 외부 에이전트 단계의 실패 기록이 있다. .gitlab-ci.yml·gate·릴리즈 규칙·버전·태그·headcount 원문은 수정하지 않았고 커밋·배포·원격 전송도 하지 않았다.

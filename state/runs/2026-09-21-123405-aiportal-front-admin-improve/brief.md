- 과제: 수정 과제 — 릴리즈 실패의 스킬 전달·최초 릴리즈 정책 차단 해소 (가치 4 / 위험 2 / 작업량 M)
- 왜: 외부 aidev/bin/run.sh는 Claude 전용 Skill 안내를 Codex 폴백에 그대로 전달하면서 원본 경로를 제공하지 않아 실제 존재하는 스킬을 찾지 못하게 한다. 전달을 복구하고 별개인 최초 릴리즈 정책의 근거까지 확보해야 같은 실패의 반복을 끝낼 수 있다.
- 수용 기준: 1) 러너 소유 worktree에서 실제 run_agent → run_codex 자식 프로세스가 registry에 지정된 스킬 원본과 상대 참조를 읽고, Claude 경로도 유지된다. 2) 누락 스킬·미정 릴리즈 관례는 명시적으로 실패하며 기존 실패/CI/자산 게이트는 그대로 유지된다. 3) 같은 폴백 검증이 수정 전 실패·수정 후 통과하고 전달 수정만 되돌리면 다시 실패해야 한다. 최초 릴리즈 정책 근거가 없는 동안 전체 릴리즈 성공으로 기록하지 않는다.
- 건드릴 파일: 현재 앱 저장소는 없음. 착수 조건 충족 뒤 aidev/bin/run.sh:dept_note/run_agent/run_codex — 엔진별 스킬 안내와 registry 기반 원본 경로·참조 기준 전달; aidev/tests/test_sim.py 및 tests/sim/run_sim.sh — 실제 러너와 자식 프로세스를 통과하는 회귀 검증 보강. agents/registry.json은 읽기 정본으로 유지. release-prompt.md·bin/gate.py는 변경 대상 아님.
- 검증 명령: 아래 실행 결과와 단계별 계획 참조. 앱 npm build는 이 외부 러너 결함의 검증이 아니다.
- 위험과 피할 것: 앱 .gitlab-ci.yml·auth·router·package/lock 버전·전역 Codex/Claude 설정 변경 금지. 태그/CHANGELOG를 임의로 신설하거나 failed를 skipped/released로 바꿔 통과시키지 않는다. 실제 운영 fixer.sh/run.sh 실행 금지(원격 동기화·알림·큐 변경 부작용). 소스 문자열 검사나 성공 응답만 반환하는 가짜 에이전트를 폴백 증거로 삼지 않는다.
- 차선 후보: 없음 — 우선 과제가 고정되어 있으므로 앱 기능 개선으로 바꾸지 않는다. 현재 조건에서는 아래 차단 판정을 원장에 남기는 것으로 끝내며 이전 외부 이관 과제를 다시 수행했다고 기록하지 않는다.

착수 판정: **현재 앱 회차에서는 blocked, 수정 미완료**.
이전 2026-09-21-111404 회차의 handoff.md를 읽었으며 같은 외부 이관은 이미 남아 있다. 현재 run.json도 project=aiportal-front-admin이고 builder.surface는 그 회차 worktree다. 현재 앱에서 실행 가능한 S/M 코드 수정으로 이 실패를 해결할 근거는 없다. 외부 파일이 보인다는 이유로 수정 권한이나 회차 소유권을 추정하지 않는다. 구현자는 앱 변경/중복 이관 문서/형식적 커밋을 만들지 말고 '수정 과제 — 선행조건 미충족, 미완료'로 기록한다. 이 과제의 pending은 미해결 표시이며 착수 가능 표시가 아니다.

확인한 증거:
- aidev/bin/run.sh:245~259 agent_plugin_args/dept_note는 Claude plugin-dir 및 Skill 호출 안내를 만들고, :290/:308/:314~327에서 그 prompt를 원본 경로 없이 Codex로 전달한다.
- headcount/plugins/{pmo,technology,marketing}/skills/<이름>/SKILL.md 원본은 존재한다. 요청한 정찰 스킬 3개와 릴리즈 스킬 2개를 직접 읽었다. Skill 호출 도구는 현재 제공되지 않는다. 로컬 원본 읽기는 실제 폴백 자식 프로세스 검증과 다르다.
- root package.json=0.0.0, upgrade/admin-v2/package.json=0.1.0; git tag 출력 없음. git log --all -G '"version"' -- package.json upgrade/admin-v2/package.json에는 최초 추가 81b150f/e7bded2만 있다. 기존 release-context.md에 GitHub Release 목록이 비어 있고 workflow 없음. 원격 현황은 이번에 재조회하지 않았다.
- release-prompt.md 절차 2~3은 기존 증가/태그/노트 관례를 요구하고 절차 5는 버전 파일까지 없어야 skipped 허용한다. 스킬 전달만 고쳐 전체 릴리즈 성공을 약속할 수 없다.
- 이번 추가 발견: aidev/bin/fixer.sh:53~61은 status=failed인 저장된 release.json을 큐에 적재하며 실패 횟수·워크플로 존재를 확인하지 않는다. run.sh:1688~1690은 모든 FIX_PROJECT에 '같은 이유로 두 번 워크플로 실패' 문구를 붙인다. 따라서 프롬프트 문구 자체는 두 번의 실제 CI 실패 증거가 아니다. fixer.sh는 최근 no-change와 별개로 release.json을 읽는다. 실제 재적재 횟수는 미확인이며 반복 원인은 정적 배선 근거로만 판단했다.

외부 러너 소유 회차에서의 실행 순서(현재 회차에서는 모두 미착수):
1. 담당 worktree와 쓰기 표면을 확정한다. 증거: 해당 회차 run.json/project 및 git rev-parse --show-toplevel. 자동 체크포인트; 현재 조건이므로 여기서 중지한다.
2. tests/test_sim.py·tests/sim/run_sim.sh의 격리 방식을 활용해 실제 run_agent 폴백 자식 프로세스가 release/scout 원본·상대 참조를 읽는 테스트를 먼저 추가한다. 외부 CLI 경계만 통제하고 운영 함수는 대역으로 바꾸지 않는다. 증거: 새 테스트가 현재 전달 누락 때문에 실패(새 테스트 이름/명령은 구현 시 확정; 지금 존재한다고 주장하지 않는다). 실패 이유 확인 뒤 다음 단계.
3. run_codex까지 registry 기반 원본 경로와 상대 리소스 기준을 전달하고, 엔진에 맞는 안내를 생성한다. 최소 release/scout, 원본 누락, headcount 비활성, Claude 정상 경로를 검사한다. 단계 2와 같은 테스트를 실행해 통과하고 변경 복원 시 실패를 확인한다.
4. bash -n bin/run.sh; python3 -B tests/test_gate.py; python3 -B tests/test_sim.py. 전체 sim은 각 시나리오 10~40초로 문서화되어 있어 45분 초과 시 확장하지 말고 실제 소요를 기록한다. gates 통과를 릴리즈 정책 결정으로 해석하지 않는다.
5. 최초 버전/태그/노트 대상(root/V2)과 관례의 명시적 운영 결정이 존재할 때만 그 근거를 따라 릴리즈를 재검증한다. 결정이 없으면 failed 유지. 이 결정·실제 게시·서버 UAT는 45분 전달 수정 범위 밖이다.

이번 정찰에서 실제 실행한 명령/결과:
- bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh → exit 0.
- TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-123405-aiportal-front-admin-improve/assets/test-temp PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release → 5개 통과, 기존 ResourceWarning 있음.
- python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-102422-aiportal-front-admin-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-102422-aiportal-front-admin-improve → exit 1, ok:false/state:failed. 원래 실패 판정 재현이지 수정 후 통과가 아니다.
- 앱 의존성 node_modules 없음. 앱 설치/빌드/전체 sim/실제 Codex 폴백/서버 UAT 미실행. 정찰은 코드 수정·커밋하지 않았다.

대안 비교 및 추정 근거:
- 최소 해법: 소유 러너에서 스킬 전달 복구, 관례 결정은 별도 선행조건. 실제 결함을 고치며 Claude 동작을 유지하므로 선택.
- 확장 해법: 플러그인 설치·전역 설정까지 재구성. 범위/회귀 위험이 커 제외.
- 앱 문서에 외부 절대 스킬 경로를 심는 우회: 머신 종속이고 최초 정책 차단도 못 풀어 제외.
- 현재 상태 유지: 앱 회차에서 유일하게 가능한 처리지만 수리 완료가 아니므로 명시적으로 blocked 기록.
- 하향식 시간 상한은 45분, bottom-up은 재현 10~15분 + 전달 수정 10~15분 + 대상 검증/기록 8~10분 = 28~40분. 알려진 변동(격리 harness 조정)에 contingency 5분을 별도로 두어 33~45분, 낮은 신뢰도의 작업 추정이며 성공 확률로 보증하지 않는다. 비교 가능한 완료 이력이 없으므로 유사 사례 추정은 불가. 전체 sim 시간·관례 결정 대기는 제외하고 별도 재산정한다. management reserve는 이 회차에 배정된 근거가 없다.
- 적용 스킬: headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md(분해·범위·예비시간), technology/skills/implementation-planning/SKILL.md(변경/증거/체크포인트), technology/skills/solution-exploration/SKILL.md(대안·가정). estimating references/sources.md도 읽었으며 외부 비용/편익 수치는 사용하지 않았다.

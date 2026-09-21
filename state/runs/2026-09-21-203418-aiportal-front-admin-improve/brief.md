- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 실패는 앱 워크플로 실행 실패가 아니라 스킬 탐색 실패와 최초 릴리즈 관례 미확정이며, 외부 러너는 이를 앱 수정 큐에 다시 넣고 모든 과제에 “두 번 실패”라고 붙인다. 실제 원인과 수정 소유 범위를 일치시켜야 반복 no-change를 줄일 수 있지만 현재 앱 worktree에서 고칠 수 있는 원인 파일은 확인하지 못했다.
- 수용 기준: 1) 실제 run_agent→run_codex 자식 경로에서 registry의 릴리즈 스킬 두 원본 및 상대 참조를 읽을 수 있고, 없는 Skill 도구를 필수로 요구하지 않는다. 2) 근거 있는 최초 릴리즈 계약(대상 패키지·버전 증가·태그·노트·자산)이 확보된 뒤 같은 릴리즈 절차가 성공하고, 관례 미확정/스킬 부재는 계속 실패로 남는다. 3) 실제 자식 경로의 수정 전 실패/수정 후 성공 및 기존 gate의 실패 차단을 함께 증명한다; 가짜 에이전트·소스 문자열 검사·기존 gate 5개 통과만으로 복구 성공을 주장하지 않는다.
- 건드릴 파일: 현재 앱 안에는 확인된 원인 수정 파일 없음. 외부 소유 환경에서만 /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — registry 기반 원본 경로·엔진별 사용 안내 전달; 같은 저장소 tests/test_sim.py:run 및 tests/sim/run_sim.sh — 기존 가짜 엔진 검사의 한계를 보완할 실제 자식 회귀가 필요(새 테스트 파일·함수명은 미확인). release-prompt.md:절차 1~6은 계약 확인 대상이며 skipped 조건을 바꾸는 수정 대상이 아님.
- 검증 명령: 아래 실행 기록 참조. 실제 자식 회귀 명령은 현재 저장소에 존재하는 것을 확인하지 못했으므로 만들어낸 명령을 제시하지 않는다. 앱 V2의 npm run build는 이 실패의 재현 명령이 아니다.
- 위험과 피할 것: 이 회차의 project는 aiportal-front-admin이며 구현 표면은 그 worktree다. 앱 auth·CI·deploy·버전 파일을 대리 수정하거나 외부 aidev 코드를 이 회차에서 편집하지 않는다. 태그/릴리즈 관례를 임의 신설하거나 release.json을 성공으로 덮어쓰거나 gate·워크플로를 느슨하게 만들지 않는다. 스킬 전달만 고쳐도 최초 계약 문제는 남는다.
- 차선 후보: 동일 실패의 자동 적재 시 원인·소유 범위 보존 — bin/fixer.sh의 failed release 적재 및 run.sh의 fix_note 생성이 대상인 외부 aidev 과제다. 이번 앱 회차의 대체 구현으로 선택하지 않는다. 앱 UI 개선으로 바꿔치기할 수 없다.

착수 판정: blocked / 지정 과제 pending. 위 과제는 우선 배정 때문에 유지한 것이며 실행 가능한 45분 앱 과제로 재추천한 것이 아니다. 이전 여섯 회차와 동일한 소유 범위·관례 부재가 유지되어 이번에는 구현 가능한 척 하는 재이관이나 재시도 지시를 하지 않는다. 이 과제서 작성·차단 기록·기존 검사 통과는 수정 성과가 아니다. 현재 제약으로 “수정 후 동일 검증 통과”를 달성할 수 없음이 정찰 결론이다.

확인 근거 (2026-09-21, HEAD 01fedba)
- 요청된 세 스킬을 /mnt/c/Users/USER/projects/headcount/plugins/{pmo,technology}/skills/<이름>/SKILL.md에서 직접 읽었다. 사용 가능한 도구 목록에 Skill/skills.list/skills.read가 없다. 릴리즈 스킬 두 원본도 실제 존재하며 읽었으므로 “원본 자체가 없음”과 “자식에 전달되지 않음”을 구분한다.
- run.sh:245~331: Claude에는 --plugin-dir를 넘기고 dept_note는 Skill 호출만 안내한다. Codex는 같은 prompt를 받아 env -i로 실행되며 headcount 원본 경로가 전달되지 않는다. 실제 과거 실패 자식의 모든 파일 접근은 미확인; 정적 배선과 저장 실패 사유가 일치한다는 수준이다.
- bin/fixer.sh:52~61: *.release.json의 status=failed를 프로젝트 이름으로 적재한다. run.sh:1688~1690은 실패 횟수를 세지 않고 “두 번 실패” 문장을 붙인다. 실제 workflow run ID와 실패 step 로그는 미확인이다.
- .github/workflows, 추적 CLAUDE.md/AGENTS.md/CHANGELOG, 로컬 태그 없음. .gitlab-ci.yml은 기존 앱의 branch별 build/copy다. docs/OPERATIONS_RUNBOOK.md와 upgrade/admin-v2/deploy/README.md는 GitLab 정적 배포를 설명하며 버전·태그 최초 계약을 제공하지 않는다. 원격 Release 유무는 이번에 조회하지 않았다.
- package.json은 0.0.0, upgrade/admin-v2/package.json은 0.1.0. release-prompt.md 절차 5의 “태그도, 버전 파일도, 릴리즈 노트도 없음” 조건에 맞지 않는다. 버전 파일만 있는 상황의 운영 결정을 대신 만들어서는 안 된다.
- agents/registry.json builder.surface와 COMPANY.md 규칙 1의 “구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다”를 확인했다. 앱 과제서로 외부 소유를 바꾸는 것은 해결이 아니다.

실행한 검증과 한계
1. 저장소 cwd: `node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json` → exit 0, runtime-config OK.
2. `bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh /mnt/c/Users/USER/projects/aidev/bin/fixer.sh` → exit 0. 구문만 검사한다.
3. `TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-203418-aiportal-front-admin-improve python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v` → 5/5 OK, exit 0, 기존 ResourceWarning. 임시 파일도 허용된 회차 폴더 안에만 생성했다. evaluate_release의 자료형·자산·상태 검사이며 스킬 로딩 검사가 아니다.
4. `python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-203418-aiportal-front-admin-improve` → exit 1, ok:false/state:failed. 저장된 실패 재판정이며 릴리즈 전체 재실행은 아니다.
5. 실제 자식 회귀, npm ci/verify/build, 서버/UAT, 전체 sim은 미실행. V2 node_modules 없음. tests/test_sim.py는 가짜 GitHub/에이전트 사용; 그 통과도 실제 스킬 접근 증거가 아니다.

해결 경로 비교 및 실행 단계 (외부 소유 환경과 계약이 확보될 때에만 적용)
- 최소 변경: registry를 단일 원천으로 유지하고 엔진에 맞는 스킬 경로/안내를 전달한다. 가장 작은 배선 수정이나 최초 관례 결정을 대체하지 못한다.
- 확장안: 실패 원인·소유자·재시도 조건을 큐에서 구조화한다. 반복을 줄이지만 이번 45분 원인 수리보다 범위가 크므로 별도 pending.
- 새 구성요소 없는 안: 이미 정해진 릴리즈 계약을 찾아 기존 절차를 실행한다. 현재 계약 근거가 없으므로 성립하지 않는다.
- 현 상태 유지: 게이트 실패를 보존한다. 현재 허용 범위에서 가능한 판정이지만 복구나 개선 성공은 아니다.
- 단계 1 / 대기: 외부 소유 환경과 기존 계약의 근거 확인. 증명은 run.json·registry·결정 문서이며 근거 없으면 다음 단계에 진입하지 않는다. 현재 앱 구현자에게 승인 질문을 하라는 뜻이 아니다.
- 단계 2 / 미착수: 실제 생산 함수/자식 프로세스를 거치는 실패 재현 테스트를 마련하고 엔진별 안내를 수정한다. 기존 가짜 sim에 성공 문구만 넣지 않는다. 체크포인트는 실패/성공 로그를 검토하는 코드 리뷰다.
- 단계 3 / 미착수: 같은 자식 검증과 위 gate 검사를 실행하고, 계약에 맞는 새 릴리즈 결과를 gate로 검사한다. 스킬 누락·관례 미확정 음성 경로가 계속 실패해야 한다. 배포/푸시는 이 정찰 과제의 범위 밖이다.

견적 근거
- 방법: bottom-up, 배선 수정 10~15분 + 실제 자식 회귀 15~20분 + 검토·기록 5분 = 기본 30~40분. 알려진 변동(자식 실행 대기) contingency 5분 별도, 합계 35~45분은 낮은 확신의 조건부 범위이며 통계적 P80이 아니다.
- 비교: 앞선 동일 앱 회차 여섯 번이 no-change였다는 analogous 근거상 현재 전제를 유지한 45분 완료 전망은 성립하지 않는다. 두 방법의 차이는 코딩량이 아니라 소유 범위·최초 계약 미확보 때문이다.
- 운영 결정 대기·새 릴리즈 규약 설계·배포는 제외하며 소요 미확인. management reserve는 정찰이 배정하지 않는다. 실자식 재현을 마련한 시점에 재견적한다.
- 방법 지침: 읽은 pmo:estimating-and-contingency 및 references/sources.md. 외부 비용 통계·확률치는 사용하지 않았다.

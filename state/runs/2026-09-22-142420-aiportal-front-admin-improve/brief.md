- 과제: 수정 과제 — 지정 릴리즈 실패의 엔진별 스킬 입력 및 최초 릴리즈 계약 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장 실패에는 필수 스킬 접근 실패와 최초 릴리즈 관례 부재가 함께 기록돼 있고, 현재 run_agent/run_codex는 Claude용 Skill 안내만 같은 prompt로 전달한다. 두 입력을 모두 갖춘 실제 릴리즈 자식이 기존 검증을 통과해야 실패가 해결되며, 앱 변경이나 기준선 통과만으로는 해결되지 않는다.
- 수용 기준: 1) 실제 run_agent → run_codex 경로의 자식 실행 기록에서 registry의 릴리즈 두 스킬 원본 및 상대 참조를 읽고 적용했음이 확인된다(Claude 경로·headcount 비활성화 동작도 유지). 2) 근거가 있는 대상 패키지·다음 버전·태그·노트·자산·검증 계약을 입력받은 동일 실패 재현에서 결과가 기존 release/manifest/CI 검사에 통과하고, 입력 누락 시 실패 차단은 유지된다. 3) 수정 전 실패/수정 후 통과를 같은 실제 실행 경로로 보이고 Release 보호 테스트도 통과한다. fake 에이전트·prompt 문자열 검사·앱 runtime 검사만으로 완료하지 않는다.
- 건드릴 파일: 현재 앱 저장소에서 입증된 원인 수정 파일 없음. 소유 프로젝트 aidev에서만 bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — registry로부터 엔진별 스킬 원본 접근 입력 구성; bin/run.sh:release_context/release_project — 확인된 릴리즈 계약과 조회 실패를 구분하여 전달; tests/test_gate.py:Release 및 tests/test_sim.py:ReleaseSafety — 기존 보호 검사는 유지하며 실제 자식 배선 회귀를 별도로 추가할 위치 검토. agents/registry.json은 경로의 정본으로 읽고 하드코딩 명부 복제 금지. release-prompt.md 절차 2~5의 관례 준수와 skipped 조건을 완화하지 않는다.
- 검증 명령: 아래 실행 기록 및 조건부 구현 순서 참조.
- 위험과 피할 것: 정찰은 코드·커밋을 만들지 않는다. 현재 회차는 aiportal-front-admin worktree이므로 외부 aidev 원본을 수정하는 지시로 해석하지 않는다. 앱 .gitlab-ci.yml/auth/deploy 및 버전 파일에 무관한 변경을 넣거나 임의 최초 버전·태그를 정하지 않는다. 저장 failed JSON 덮어쓰기, gate 완화, fake 성공, 큐 삭제/자동 skipped 처리로 실패를 숨기지 않는다. 자격증명 원문 출력 금지.
- 차선 후보: 없음 — 자동 배정 실패를 앱 UI/DX 과제로 대체하지 않는다. 아래 신규 후보는 다음 배정 검토용이다.

현재 판정: pending/blocked, 미착수. 이전 no-change 과제를 실행 가능하다고 재승인하지 않는다. 이번에도 수정 및 동일 검증 통과 요구는 충족하지 못했다. 원인은 코드 수리 전에 필요한 소유 worktree와 최초 계약이 이 회차 입력에 없기 때문이다. 사용자 절대 규칙(코드 수정 금지, 지정 회차 폴더만 쓰기)과 COMPANY.md 규칙 1의 표면 제한을 모두 지킨다. 이것은 스킬이 요구한 승인 절차가 아니며 질문하거나 외부 이관을 실행하지 않았다.

직접 확인한 근거
- 앱 HEAD 01fedba, git status --short 비어 있음, git log -30 확인. 추적 AGENTS.md/CLAUDE.md/CHANGELOG/.github 파일과 로컬 태그 검색 결과 없음. 루트 0.0.0 / V2 0.1.0; 전체 버전 이력 불변은 과거 실패 보고이고 이번 재검증하지 않았다.
- README.md, docs/ROADMAP.md, docs/TESTING_GUIDE.md, docs/OPERATIONS_RUNBOOK.md, 두 package.json, V2 vite.config.ts, .gitlab-ci.yml(자격증명 마스킹)을 읽었다. GitLab은 branch build/copy 방식이며 이 실패의 Skill 입력 경로는 없다. 원격 최신 Release/실제 두 workflow 실패는 미확인.
- aidev/bin/run.sh 246~335: Claude에는 plugin-dir/Skill, Codex에는 같은 prompt만 전달. 스킬 원본 경로 전달이 없다는 정적 근거이며 실제 자식 실패 전체 인과를 입증한 것은 아니다.
- aidev/bin/run.sh:release_context는 gh 조회 오류도 '(없음)'으로 표시할 수 있다. release_project는 release-prompt의 관례 입력을 조합해 자식에 넘기며 gate 실패 시 중단한다.
- aidev/bin/fixer.sh 52~60은 status=failed 스냅샷을 적재한다. run.sh 1690의 '두 번 실패' 문구는 일반 실패에도 고정으로 붙는다. 이를 실제 두 워크플로 실패 증거로 사용하지 않는다.
- tests/test_gate.py:Release는 스키마·자산 경계 검사다. tests/test_sim.py:ReleaseSafety는 가짜 GitHub/에이전트 경로이므로 실제 스킬 접근 증거가 아니다.

구현 순서 — 소유 환경 및 계약이 갖춰진 후에만 적용하는 계획
1. [차단] aidev worktree에서 최초 실패 당시 엔진·phase·입력·출력 및 원격 조회 성공 여부를 고정하고, 대상 package/version/tag/notes/assets 계약의 근거를 확보한다. 증명: 실제 자식 실행 transcript와 계약 원문을 회차 증거로 보존. 현 앱 회차에 이를 만들어 넣지 않는다. 체크포인트: 기계적으로 입력 충족 여부 확인, 미충족이면 다음 단계 금지.
2. [대기] registry 기반으로 Claude와 Codex 각각 읽을 수 있는 스킬 입력을 구성한다. 상대 references 접근과 off-switch도 유지한다. 증명: 운영 run_agent/run_codex를 경유한 실제 자식의 원본 읽기 기록. 테스트가 없는 현재 저장소에 가상의 실행 명령을 지어내지 말고 이 단계에서 실제 회귀 진입점과 정확한 명령을 과제서에 추가한다. 체크포인트: 소스 문자열 검사를 통과 증거로 쓰지 않는다.
3. [대기] 근거가 확보된 최초 계약을 기존 release_project 입력 경로로 연결하고 실패 케이스를 재실행한다. 계약 없을 때 계속 차단, 있을 때 기존 gate/manifest 통과를 확인한다. 체크포인트: 계약을 정할 권한/입력이 없다면 45분 범위를 벗어난 것이므로 완료 처리하지 않는다.
4. [대기] 아래 Release 보호 검사와 실제 자식 회귀를 같은 입력으로 재실행하고 원장에 수정 전후 증거를 기록한다. 외부 게시·태그 push는 이 정찰/수리 검증의 범위 밖이다.

실제 실행한 검증(이번 회차 validation.json에 stdout/stderr/exit code 보존)
```bash
# cwd=/mnt/c/Users/USER/projects/aidev, 임시 생성물도 이번 회차 안으로 제한
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-142420-aiportal-front-admin-improve/tmp python3 -B tests/test_gate.py Release -v
# 앱 cwd
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-142420-aiportal-front-admin-improve
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
```
결과: Release 5개 exit 0(ResourceWarning), 저장 실패 gate exit 1/state=failed, runtime exit 0. 모두 변경 전 기준선이며 복구 성공이 아니다. 실제 자식/수정 후 회귀/ReleaseSafety/전체 npm 테스트·빌드/서버 UAT 미실행. ReleaseSafety는 고정 /tmp/aidev-sim 사용으로 현재 쓰기 경계를 벗어나므로 실행하지 않았다.

대안 비교 및 추정(요청 스킬 적용)
- 권고: 소유 러너의 엔진별 원본 입력을 복구하고 기존 계약을 연결한다. 기존 명부를 재사용하고 gate는 그대로 둔다. 가장 큰 가정은 최초 계약과 소유 worktree가 확보된다는 것이다.
- 앱에 새로운 릴리즈 정책을 정하는 대안: 권한 있는 최초 계약이 있을 때만 가능하며 지금은 근거가 없다.
- 공용 스킬 배포 시스템을 새로 만드는 대안: 여러 엔진이 늘어나면 재검토할 수 있지만 지금 45분 범위 초과.
- 현 상태 유지: 거짓 성공은 막지만 반복 실패를 해결하지 못하므로 완료로 인정하지 않는다.
- bottom-up 조건부 추정: 입력/재현 5~8분 + 배선 수정 10~15분 + 실제 회귀/보호 검사 10~15분 + 기록 3~5분 = 28~43분. 환경 재실행 변동 contingency 5~10분 별도, 합계 33~53분; 주관적 저신뢰 범위이며 통계적 신뢰수준을 주장하지 않는다. 계약 결정·실제 배포·새 시스템 구축은 제외, management reserve는 배정 없음. 비교 가능한 성공 이력이 없어 유사 추정 교차검증 불가. 따라서 전체 복구를 45분 완료로 약속하지 않는다.
- 적용 원본: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 technology/skills/{implementation-planning,solution-exploration}/SKILL.md. Skill 도구는 없어서 파일 직접 읽기로 대체; pmo references/sources.md도 확인했고 외부 비용 모델 수치/공식은 사용하지 않았다.

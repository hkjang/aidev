- 과제: 수정 과제 — 지정 릴리즈 실패의 엔진별 스킬 입력 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 릴리즈 단계가 요구하는 스킬의 원본은 존재하지만 run_agent가 Claude용 Skill 안내를 Codex에 그대로 넘기고 원본 위치를 전달하지 않아, 저장된 실패 사유와 부합하는 전달 결함이 남아 있다. 엔진별로 같은 원본에 접근하도록 하고 최초 릴리즈 계약까지 검증하면, 입력 누락과 앱 결함을 혼동하지 않고 같은 실패의 복구 여부를 판정할 수 있다.
- 수용 기준: 1) release 역할의 registry 두 스킬과 상대 참조를 Codex 자식이 실제 읽은 증거가 있으며 Claude plugin 전달 및 headcount 비활성화 동작도 보존된다. 2) 승인 출처가 있는 대상 패키지·다음 버전·태그 형식·노트·자산 계약을 같은 릴리즈 입력에 연결하고, 동일 실패 조건에서 로컬 릴리즈 검증을 끝낸다. 계약 누락·스킬 누락은 계속 차단한다. 3) 수정 전 실패/수정 후 성공을 같은 자식 회귀로 입증하고 Release/ReleaseSafety의 기존 거부 조건이 유지된다. 단순 JSON status 변경, 모의 성공 응답, 기준선 5개 통과만으로 완료하지 않는다.
- 건드릴 파일: 현 aiportal-front-admin 앱 worktree에는 지정 원인을 고칠 대상 파일이 없다. 실제 소유 저장소 /mnt/c/Users/USER/projects/aidev의 bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — registry로부터 엔진별 스킬 입력 구성; tests/test_sim.py:ReleaseSafety 및 tests/sim/run_sim.sh — 기존 모의 검증을 보존하고 자식의 실제 스킬 접근 검증 추가; release-prompt.md:절차 1~6 및 bin/run.sh:release_context/release_project — 승인된 최초 계약을 입력으로 연결하되 기존 검증 유지. agents/registry.json은 읽을 원천이며 스킬 이름 하드코딩 금지. 이 목록은 소유 환경에서의 변경 설계로, 현 앱 회차의 외부 파일 편집 지시가 아니다.
- 검증 명령: 아래 ‘실행 및 증거’ 참조. 현재 재현된 것은 수정 전 기준선이며 수정 후 통과는 미확인.
- 위험과 피할 것: 코드를 고치지 않는 정찰 역할과 지정 회차 폴더 밖 쓰기 금지를 지킨다. COMPANY.md 규칙 1의 구현 표면 제한도 유지한다. .gitlab-ci.yml/auth/router/deploy 및 앱 버전 파일을 우회 수정하지 않는다. release gate, CI, 비밀 검사, 자산 경계, skipped 조건을 느슨하게 만들지 않는다. 최초 버전/태그를 0.0.1 또는 0.1.1로 추정하지 않는다. 저장 failed JSON·큐·정책을 바꿔 성공처럼 만들지 않는다.
- 차선 후보: 같은 실패의 최초 릴리즈 계약 입력 연결 — 스킬 전달이 실제 자식에서 정상임이 입증된 경우에만 동일 과제 안에서 원인을 좁힌다. 다른 앱 개선으로 대체하지 않는다.

현재 판정과 반복 방지

상태: pending/blocked, 수용 기준 1~3 미충족. 이번 앱 회차에서 실행 가능한 수정으로 재승인하지 않는다. 앞선 여섯 회차와 같은 이관 문서를 ‘개선 완료’로 세지 않는다. 사용자 절대 규칙상 정찰이 직접 고쳐 검증할 수 없으며, 현 run.json project는 aiportal-front-admin이다. 외부 소유 코드를 수정할 수 있는 별도 실행 범위와 최초 릴리즈 계약이 확보되지 않은 상태에서 45분 완료를 약속할 수 없다. 이 문서는 원인 증거 및 조건부 구현 계획이지 이관 실행이나 권한 변경 기록이 아니다.

직접 확인한 근거

- git log -30: HEAD 01fedba; git status --short 빈 출력. 로컬 태그 없음. 현재 추적 파일 검색에 CLAUDE.md/AGENTS.md/CHANGELOG/.github workflows 없음. 상위 디렉터리의 AGENTS.md/CLAUDE.md도 없음.
- README.md, docs/ROADMAP.md, docs/OPERATIONS_RUNBOOK.md, tests/README.md, 두 package.json, V2 vite.config.ts 및 deploy/README.md 확인. 루트 0.0.0, V2 0.1.0. 전체 과거 버전 불변과 현재 원격 Release 부재는 이번 미확인.
- .gitlab-ci.yml은 브랜치 build/copy 배포다. 자격증명 URL은 마스킹하여 읽었다. dev_build_main core 모드와 ofc_deploy_dev 중복 rules는 지정 스킬 실패의 직접 원인이 아니다.
- 외부 run.sh:246~335의 agent_plugin_args/dept_note/run_agent/run_codex 확인. dept_note release를 원본에서 추출하여 bash로 실행한 출력은 skill-delivery-baseline.txt에 보존했다. Skill 도구 요구만 있고 SKILL.md 원본 경로는 없다. run_codex 명령은 Claude pargs를 받지 않는다. 실제 자식이 어디까지 탐색했는지와 전체 실패 인과는 미입증.
- headcount/plugins/technology/skills/release-and-deployment/SKILL.md 및 marketing/skills/product-launch/SKILL.md는 현재 존재하며 읽었다. 이 세션의 callable 도구 목록에는 Skill/skills.read 도구가 없다.
- release-prompt.md 절차 5는 태그·버전 파일·노트 모두 없는 경우에만 skipped. 현재 두 package.json 때문에 적용 불가. 본문에는 최초 버전 결정 계약이 없다.
- bin/fixer.sh:52 이후 failed snapshot 적재, run.sh:1690 고정 문구 확인. ‘두 번 실패’ 문구만으로 GitHub workflow 2회 실행을 입증할 수 없다. workflow ID/실패 step 로그는 미확인.
- tests/test_gate.py:Release는 타입·경로·파일 유효성 검사이고 tests/test_sim.py:ReleaseSafety는 모의 릴리즈/자산 검사다. 실제 Codex의 원본 스킬 읽기를 검증하지 않는다. run_sim.sh는 /tmp/aidev-sim.*을 강제로 만들므로 현재 쓰기 경계에서는 실행하지 않았다.

조건부 실행 순서 — 현재 모두 pending

0. 소유 환경과 승인된 최초 계약을 먼저 확인한다. 대상과 권한이 확보되지 않으면 구현 착수 불가를 유지한다. 앱 코드에 임시 스킬 사본이나 릴리즈 관례를 만들지 않는다. 사람에게 질문을 보내거나 별도 승인 플로우를 시작하지 않는다.
1. 소유 aidev worktree에서 registry의 release 스킬을 읽어 엔진별 입력을 구성한다. Claude는 기존 plugin 경로, Codex는 읽을 수 있는 정규 SKILL.md 경로와 상대 참조 기준 디렉터리를 받게 한다. 파일 누락 시 구체적 누락 입력으로 실패하며 headcount 비활성 설정을 존중한다. 검증: bash -n bin/run.sh 및 기존 Release 테스트. 체크포인트: 자동 검증 통과 후 다음 단계; 별도 사람 검토 없음.
2. 실제 호출 인자를 검사하는 모의 프로세스 회귀와 격리된 실제 자식 읽기 회귀를 추가한다. 두 엔진, 비활성 설정, 원본 누락, 상대 참조를 포함한다. 새 테스트 함수·CLI 이름은 아직 없으므로 있는 명령처럼 쓰지 않는다. 완료 전 구현자가 정확한 회귀 명령과 수정 전후 결과를 이 문서에 추가해야 한다. 체크포인트: 실제 자식 증거 없으면 완료 표시 금지.
3. 승인된 최초 계약을 release 입력으로 연결하고 기존 절차로 로컬 검증한다. 계약 부재 시 failed 유지; 계약을 임의 발명하지 않는다. tests/test_sim.py ReleaseSafety와 Release 테스트를 실행하고 저장된 원본 실패 JSON은 보존한다. 체크포인트: 같은 실패 검증의 통과 및 보호 거부 케이스 유지까지 확인해야 전체 복구 완료다. 원격 전송은 이 정찰/로컬 검증 계획 범위 밖이다.

실행 및 증거

회차 출력 디렉터리를 O=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-165433-aiportal-front-admin-improve 로 둔다. tmp는 이번 생성됨.

```bash
# cwd: /mnt/c/Users/USER/projects/aidev
TMPDIR="$O/tmp" python3 -B tests/test_gate.py Release -v
# 실제 결과: 5 tests OK, exit 0; 파일 핸들 ResourceWarning 있음
python3 -B bin/gate.py release state/aiportal-front-admin.release.json --out-dir "$O"
# 실제 결과: exit 1, state=failed (기존 실패 보존)
# cwd: /home/hkjang/.cache/auto-improve-wt/aiportal-front-admin
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
# 실제 결과: exit 0
```

validation.json에 명령·cwd·stdout/stderr·exit code 보존. 소유 환경에서 쓸 기존 명령은 `python3 -B tests/test_sim.py ReleaseSafety -v`이며 이번 미실행. 앱 전체 npm ci/verify/build·서버/UAT·실제 자식 재현·수정 후 회귀도 미실행이다. 보호 테스트 통과는 수정 증거가 아니다.

대안 비교와 추정 근거

- 선택: registry 기반 엔진별 원본 전달. 가장 작은 원인 수정이며 스킬 사본 유지 비용이 없다. 원본 경로 접근이 실제 자식에도 허용된다는 가정은 회귀에서 검증해야 한다.
- 확장안: 엔진 공통 스킬 패키징 계층. 다수 엔진에 유용하지만 현재 45분 범위를 넘으므로 보류.
- 새 구성요소 없는 안: Claude만 재실행. 사용량 제한이 해소됐다는 근거가 없으며 최초 계약 부재도 남아 복구안으로 채택하지 않는다.
- 현상 유지: 권한 경계상 지금의 정직한 상태지만 과제 완료나 복구로 세지 않는다.
- 상향식 추정(소유 환경·계약 확보 후): 입력 구성 10~15분, 호출/자식 회귀 10~15분, 보호 검사·기록 5~10분 = 25~40분; 알려진 경로/참조 변동 예비 5분 별도. 30~45분은 낮은 확신의 조건부 예상이며 통계적 확률 보장이 아니다. 외부 계약 결정 대기와 실제 전체 릴리즈 빌드는 제외; 관리 예비는 미배정.
- 유사 사례 교차 확인: 지난 여섯 회차 no-change는 성공 소요시간 자료가 아니므로 유사 추정값을 만들지 않는다. 따라서 전체 복구 45분 충족은 입증되지 않았다. 첫 자식 재현 후 재추정한다.
- 적용 스킬: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md. 첫 경로의 references/sources.md도 읽었으며 위 시간은 외부 권위 수치가 아닌 코드 범위 기반 잠정 추정이다.

후보 원장

ideas.json의 기존 62개를 보존하고 신규 2개를 추가했다. done인 Catalog·Content query 무효화는 관련 커밋/회귀 파일을 확인하여 유지했다. pending의 완료·기각을 증명할 새 변경은 없으며 런타임 미확인인 항목은 추측으로 전환하지 않았다. 새 후보는 중첩 HTML root 절대 경로 누락(2/1/S), HTML 주석 외부 URL 오탐(2/2/S)이며 회차 fixtures로 각각 exit 0/1을 재현했다. 지정 실패보다 우선하지 않는다.

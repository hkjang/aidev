- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 자동 적재된 실패는 앱 빌드 오류가 아니라 스킬 전달 누락과 최초 릴리즈 계약 부재를 보고하고 있다. 두 원인을 실제 소유 범위에서 해소해야 같은 실패를 앱 수정 회차에 반복 배정하는 낭비와 근거 없는 릴리즈를 막을 수 있다.
- 수용 기준: 1) 실제 run_agent→run_codex 자식이 registry의 release 스킬 두 원본과 상대 참조를 읽고 그 증거를 남긴다. 2) 기존 운영 결정에 근거한 대상 패키지·다음 버전·태그·노트·자산 계약이 확보되어 실제 릴리즈 경로가 기존 검사들을 통과한다. 3) 수정 전 동일 자식 경로 실패/수정 후 성공을 재현하고, 스킬 파일 부재·계약 미확정 입력은 계속 차단됨을 증명한다. 기존 gate 단위 테스트 통과, 이관, 재진단, 문서만 생성한 것은 수정 완료가 아니다.
- 건드릴 파일: 현재 aiportal-front-admin worktree에는 확인된 원인 수정 대상 없음. 원인 소유 파일은 /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args,dept_note,run_agent,run_codex — 엔진별 스킬 안내·원본 경로 전달; /mnt/c/Users/USER/projects/aidev/agents/registry.json — 역할·스킬 정본(읽기); /mnt/c/Users/USER/projects/aidev/tests/test_sim.py:run,Agents 및 tests/sim/run_sim.sh — 실제 배선 회귀의 기존 출발점(현재 fake claude이므로 성공 근거로 불충분); /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 1~6 — 운영 계약 대조(완화 금지). 이 목록은 외부 파일 편집 허가가 아니다.
- 검증 명령: 아래 실행 기록과 소유 환경에서의 추가 검증 조건 참조.
- 위험과 피할 것: 코드·커밋 금지인 정찰 역할을 지킨다. COMPANY.md 규칙 1 및 registry.builder.surface에 따라 현재 앱 구현자가 외부 aidev를 수정할 수 없다. 앱 .gitlab-ci.yml/auth/router/deploy/package 버전을 대신 고치거나 최초 태그·CHANGELOG를 발명하지 않는다. release.json을 released/skipped로 바꾸기, skipped 조건 확대, gate 완화, 실패 큐 삭제는 금지한다. 비밀값은 기록하지 않는다.
- 차선 후보: 없음(지정 우선 과제). 소유 범위/운영 계약이 준비되지 않은 상태에서 UI·테스트 청소로 대체하지 않는다.

현재 착수 판정: **pending / blocked — 이 앱 회차에서 실행 가능한 45분 수정 과제로 성립하지 않음**.
이전 여섯 회차의 같은 외부 수정 계획을 다시 실행하도록 지시하지 않는다. 구현자는 아래 증거로 기존 차단이 유지됨을 알 수 있으며, 소유 범위와 운영 계약이 실제로 달라지기 전에는 같은 탐색·테스트·이관을 반복할 필요가 없다. 이 문서가 회차 배정이나 운영 정책을 변경하지는 않는다. 원장에는 `수정 과제 — 미완료(blocked)`로 기록해야 한다.

확인한 사실:
- HEAD 01fedba, git status --short 빈 출력. 최근 git log -30 확인. README, ROADMAP, OPERATIONS_RUNBOOK, TESTING_GUIDE, tests/README, 양쪽 package.json, Vite 설정 및 화면·테스트를 읽음. 앱 CLAUDE.md/AGENTS.md와 .github는 발견하지 못함.
- .gitlab-ci.yml은 main/develop별 기존 앱 build/copy이며 release 스킬 호출이 없다. ofc_deploy_dev 중복 rules, dev_build_main의 core 모드는 별도 문제이고 지정 실패와 인과가 입증되지 않았다.
- aidev/bin/run.sh:1688~1690은 FIX_PROJECT에 무조건 “같은 이유로 두 번 실패”를 붙인다. 이 문구는 실제 워크플로 실행 2건의 증거가 아니다. fixer.sh:52~61은 failed release.json을 원래 앱 이름으로 적재한다.
- run_agent는 Claude용 dept_note/Skill 안내를 추가한 prompt를 run_codex에 그대로 전달한다. run_codex에는 headcount 경로 전달이 없다. 실제 폴백 재현은 미실행이므로 정적 전달 결함과 실제 모델 동작 증거를 구분한다.
- 원본은 /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/release-and-deployment/SKILL.md 및 marketing/skills/product-launch/SKILL.md에 존재한다(직접 읽음). 현재 가용 도구 이름을 검색했으나 Skill 호출 도구 없음. 원본 발견만으로 과거 릴리즈 복구를 주장할 수 없다.
- 로컬 tag 없음, root 0.0.0/V2 0.1.0. release-prompt 절차 5는 “태그도, 버전 파일도, 릴리즈 노트도 없음”일 때만 skipped. 원격 이력은 이번 미조회, 최초 릴리즈 승인 계약은 미확인.

소유 환경이 준비된 경우의 구현 순서(현재 전부 pending):
1. aidev 소유 worktree와 운영 계약의 실제 근거를 확보한다. 체크포인트: 자동으로 범위를 넓히지 않으며, 최초 관례는 release-prompt의 “관례를 새로 정하는 건 사람의 일”을 따른다. 이번 무인 정찰에서는 질문하거나 승인을 만들어내지 않는다.
2. run_agent/run_codex가 registry 기반 원본 경로를 실제 자식에게 전달하도록 최소 변경한다. Claude 경로, headcount off 스위치, 검토 부서 스킬 및 상대 참조를 보존한다. 체크포인트: 자식이 읽은 파일과 종료 상태를 검증한 뒤 다음 단계로 간다.
3. 실제 생산 함수와 실제 Codex 자식 경로로 수정 전후 회귀를 남긴다. 문자열 포함 검사나 fake claude 출력은 대체 증거가 아니다. 현재 실행 가능한 해당 회귀 테스트 명령은 없으며 새 테스트의 실제 실행 명령을 구현자가 기록해야 한다.
4. 스킬 접근 성공 뒤에도 최초 릴리즈 계약 부재는 독립 차단임을 확인한다. 계약 확보 후 같은 릴리즈 검사 및 gate를 통과해야 전체 완료다. 검증 실패 시 작업 상태를 계획에 갱신하며 범위를 넓히지 않는다.

이번 실행한 검증(앱 cwd, 모두 읽기/출력 폴더 내 임시 파일만):
```bash
PYTHONDONTWRITEBYTECODE=1 TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-192420-aiportal-front-admin-improve python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v
bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh /mnt/c/Users/USER/projects/aidev/bin/fixer.sh
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-192420-aiportal-front-admin-improve
```
결과: Release 5/5 OK(기존 ResourceWarning), bash 구문 진단 없음, runtime-config OK. 마지막 gate는 exit 1 / ok:false / state:failed. 이는 기존 실패 자료의 판정 재현이며 원본 릴리즈 작업 자체의 재실행이 아니다. 앱 npm ci/verify/build, 실제 서버/UAT, 전체 sim과 실제 모델 폴백은 미실행. sim/run_sim.sh는 /tmp 강제 쓰기이고 fake 도구를 쓰므로 이번 허용 출력 범위 및 증거 기준에 맞지 않는다.

대안 비교 및 추정:
- 최소 원인 수리(권고, 조건부): 소유 aidev에서 원본 전달 수리. 버전 관례 부재는 별도 운영 입력으로 처리. 앱 worktree에서 즉시 수행 불가.
- 장기 엔진 추상화: 범용 capability/스킬 resolver는 범위 L, 이번 45분 과제에 과함.
- 새 부품 없이 원본 수동 읽기: 이번 정찰에는 가능하지만 생산 전달 배선을 고치지 못하므로 해결안으로 기각.
- 아무것도 바꾸지 않고 실패 재실행/재이관: 이미 no-change 반복, 해결안으로 기각. 현재는 사실상 차단 상태만 정직하게 유지한다.
- Bottom-up 조건부 공수: 배선 10~15분 + 실제 자식 회귀 15~20분 + 기록 5분 = 30~40분, 알려진 경로 차이 contingency 5분 별도. 35~45분은 낮은 확신의 판단 범위이며 통계적 신뢰구간이나 완료 약속이 아니다. 운영 결정·권한 준비 대기와 전체 릴리즈는 제외하며 총공수는 미확정. Management reserve는 이 회차에서 배정하지 않는다. 유사사례 여섯 no-change는 선행조건 미충족을 보여 주므로 전체 과제를 M/45분으로 보증할 근거가 되지 않는다.

적용 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration의 로컬 SKILL.md 원본을 읽었다. Skill 도구 호출과 동등하다고 주장하지 않는다. 추정 스킬 references/sources.md도 확인했으며 외부 지침의 수치/규범은 인용하지 않고 위 공수는 이번 정적 조사에 근거한 판단으로 한정했다.

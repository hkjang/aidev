- 과제: 수정 과제 — 지정 릴리즈 실패의 스킬 입력 복구 및 최초 릴리즈 계약 검증 (가치 4 / 위험 2 / 작업량 M)
- 왜: 외부 러너는 Claude용 Skill 호출 안내를 Codex에도 전달하지만 원본 위치를 제공하지 않으며, 저장 실패에는 최초 릴리즈 관례 부재도 함께 기록돼 있다. 엔진별 스킬 입력을 검증하고 별도 릴리즈 계약을 확보해야 같은 실패를 앱 변경으로 오진하지 않고 실제 복구를 증명할 수 있다.
- 수용 기준: 1) 실제 release 단계의 Codex 자식이 registry에 지정된 두 원본과 필요한 상대 참조를 읽었다는 증거가 있고, 존재하지 않는 Skill 호출 요구가 없다. 2) 대상(root 또는 V2), 다음 버전, 태그 사용 여부·형식, 노트 위치·형식, 자산·검증 명령에 대한 권위 있는 최초 계약이 입력돼 임의 추정 없이 준비된다. 3) 동일 입력의 수정 전 실패/수정 후 성공을 로컬 격리 환경에서 재현하고 Release·ReleaseSafety 보호 검사를 유지한다. 기존 저장 failed JSON을 바꾸거나 gate를 느슨하게 만들어 통과시키면 불합격이다.
- 건드릴 파일: 현 앱 저장소에는 입증된 원인 수정 파일 없음. 원인 소유 저장소 /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — registry 기반 스킬 안내를 엔진별로 구성; agents/registry.json — 스킬 매핑의 입력으로 사용(불필요한 변경 금지); tests/test_sim.py:ReleaseSafety 및 tests/sim/run_sim.sh — 전달 계약 회귀와 격리 실행 보강. release-prompt.md:절차 2~5 — 현재 제약의 근거이며 완화 대상 아님. 최초 계약 저장 위치·생산자는 미확인으로 임의 파일을 만들지 않는다.
- 검증 명령: 아래 실행 결과와 명령 참조. 앱 unit/build는 본 실패의 재현 검사가 아니다.
- 위험과 피할 것: 정찰은 코드·커밋 금지. COMPANY.md 규칙 1의 “구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다”에 따라 이 앱 회차에서 외부 aidev 편집을 지시하지 않는다. auth, migrations, .gitlab-ci.yml, .github/workflows, package 버전, 원격 상태와 저장 release.json은 건드리지 않는다. 최초 버전·태그를 만들어 과거 관례처럼 보이게 하지 않는다. 보호 테스트 통과와 원장 기록을 복구 성과로 세지 않는다.
- 차선 후보: 없음 — 고정된 우선 과제이므로 다른 앱 개선으로 대체하지 않는다. 소유 환경과 최초 계약이 없는 상태에서 차선 작업을 수행해 릴리즈 복구로 기록하지 않는다.

## 착수 판정과 이전 실패의 반영
pending / blocked. 이 문서는 고정된 우선 과제의 인계 기록이며, 반복 기각된 외부 수정 계획을 현 앱에서 실행 가능하다고 재승인하지 않는다. 현재 run.json의 project는 aiportal-front-admin이다. 코드 금지·회차 폴더 밖 쓰기 금지와 지정 실패의 실제 수정 요구를 동시에 충족하는 구현은 확인되지 않았다. 따라서 “45분 내 전체 복구 가능”이라고 판정할 수 없다. 질문이나 외부 상태 변경 없이 필요한 선행조건을 남긴다.
이전 회차와 달리 이번에는 dept_note release를 함수 단독으로 실행해 출력이 Skill 호출만 요구하고 SKILL.md 경로를 포함하지 않는 것을 확인했다. 이것도 실제 Codex 자식의 접근 실패 전체를 재현한 것은 아니다.

## 확인한 근거
- HEAD 01fedba202323e348b66382e17098cb84400dd76, git log -30, 깨끗한 git status. 추적 AGENTS.md/CLAUDE.md/CHANGELOG/.github 파일과 로컬 태그 검색 결과 없음. README, package.json 두 개, docs/ROADMAP.md·운영/테스트 문서 관련 부분, V2 vite.config.ts와 deploy/README.md 확인. src와 V2 src의 TODO/FIXME 검색 결과 없음.
- .gitlab-ci.yml은 legacy branch build/copy이며 보고된 외부 release 단계가 아니다. dev_build_main core 모드와 ofc_deploy_dev 중복 rules는 별도 후보이고 본 실패 인과는 미확인. 자격증명은 출력 시 마스킹했다.
- run.sh 246~335: plugin-dir는 Claude 호출에만 사용; dept_note는 Skill 도구 안내만 생성; run_agent는 동일 prompt를 run_codex로 넘긴다. registry release의 marketing:product-launch 및 technology:release-and-deployment 원본은 현재 로컬 headcount/plugins 아래 존재한다.
- release_project는 release-prompt로 자식을 실행하고 gate.py:evaluate_release로 결과를 판정한다. release-prompt 절차 5의 skipped는 버전 파일도 없어야 한다. 현재 root 0.0.0/V2 0.1.0이므로 적용 불가; 전체 로컬 버전 이력 불변은 저장 실패 보고의 주장으로 이번 재검증하지 않았다.
- release_context는 gh 오류도 “(없음)”으로 출력한다. 실제 원격에 릴리즈가 없는지는 미확인이다. fixer.sh는 failed snapshot을 적재하고 run.sh 1690은 두 번 실패 문구를 붙인다. 두 개의 실제 workflow 실행 ID·로그는 확보되지 않았다.
- tests/test_gate.py:Release는 결과 스키마·자산 경계 테스트다. tests/test_sim.py:ReleaseSafety는 가짜 에이전트이며 tests/sim/run_sim.sh는 /tmp 경로를 고정 생성한다. 현재 쓰기 제한 때문에 sim은 실행하지 않았다.

## 소유 환경에서의 조건부 구현 순서 (현재 모두 미착수)
1. aidev를 쓰기 표면으로 하는 회차에서 기존 실패 입력을 fixture로 고정하고, Claude 제한→Codex 폴백 호출에 전달된 prompt/원본 경로를 포착하는 회귀 테스트를 추가한다. 기존 run_agent 전체를 무방비 source하지 말고 도구 stub·격리 상태를 사용한다. 검증: 신규 테스트가 현 배선에서 실패해야 한다. 체크포인트: 자동 검증 실패가 본 전달 누락 때문인지 확인, 다르면 계획 수정.
2. registry로부터 원본 절대경로와 상대 참조의 기준 디렉터리를 계산해 Codex가 읽도록 전달한다. Claude plugin-dir 동작, NO-HEADCOUNT 및 agents.headcount=false를 보존하고 누락 파일을 명시적 오류로 다룬다. 경로 공백·명부 외 경로·비활성화 케이스를 포함한다. 검증: 1의 회귀 테스트와 Release 보호 검사 통과. 체크포인트: 실제 자식 읽기 증거 없이는 완료 처리하지 않는다.
3. 별도 최초 릴리즈 계약이 확보되면 그 입력으로 로컬 준비를 재현한다. 미확보라면 스킬 전달 부분만 해결로 구분하고 전체 수정 과제는 pending 유지한다. 검증: 동일 release 절차의 수정 전후 결과 및 기존 ReleaseSafety. 체크포인트: 원격 게시 없이 준비 검증까지, 이 회차에서 게시·정책결정은 하지 않는다.
신규 회귀 테스트의 파일명·명령은 아직 존재하지 않아 실행 가능 명령으로 꾸며 쓰지 않았다. 구현자는 1단계에서 실제 명령을 이 계획에 추가해야 한다.

## 실행한 검증 (변경 전 기준선)
앱 루트에서:
```bash
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
```
exit 0.
aidev 루트에서(임시 파일도 회차 폴더에 제한):
```bash
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-132418-aiportal-front-admin-improve/tmp python3 -B tests/test_gate.py Release -v
```
5개 통과, exit 0, 파일 핸들 ResourceWarning 있음.
```bash
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-132418-aiportal-front-admin-improve
```
exit 1 / state=failed. 저장 결과 판정 재현이지 실제 릴리즈 자식 실행 재현이 아니다.
소유 환경에서 필요한 기존 명령: `python3 -B tests/test_sim.py ReleaseSafety -v` (현재 미실행, 테스트마다 수십 초, /tmp 쓰기 필요).
모든 실행 stdout/stderr와 dept_note 출력은 validation.json에 기록했다. npm 설치·unit/build·실제 자식·수정 후 회귀·서버/UAT는 미실행이다.

## 대안 비교와 추정 근거
- 최소 변경: aidev 소유 환경에서 엔진별 스킬 입력만 수정 — 권고. 릴리즈 최초 계약 차단은 별도로 남으며 전체 성공을 약속하지 않는다.
- 공용 엔진 어댑터·스킬 설치 시스템 신설 — 확장성은 있으나 이번 M 범위를 넘으므로 제외.
- 앱 AGENTS.md/버전/CHANGELOG 추가 — 러너 입력 결함을 우회하고 최초 관례를 발명하므로 제외.
- 무변경·차단 보존 — 현 권한에서의 실제 결과. 해결로 집계하지 않는다.
추정 방식은 bottom-up: 실패 fixture 8~12분, 전달 수정 10~15분, 회귀/기록 7~13분 = 25~40분. 알려진 변동(경로·권한 차이) contingency 5분으로 부분 수정은 30~45분, 낮은 확신의 판단 범위이며 통계적 신뢰수준은 산출 불가다. management reserve는 배정 없음. 최초 계약 확보·실제 LLM 자식·게시·환경 재배정 대기는 제외되며 전체 복구 시간은 미확인이다. 유사 회차는 모두 no-change여서 성공 소요시간 비교 근거가 없고, 이로써 45분 약속을 정당화하지 않는다. fixture와 실제 자식 증거 확보 뒤 재추정한다.
요청 스킬 원본: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md (references/sources.md도 확인), technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md. Skill 도구는 목록에 없어 직접 읽기로 대체했다. 별도의 외부 추정 표준·수치 인용은 사용하지 않았다.

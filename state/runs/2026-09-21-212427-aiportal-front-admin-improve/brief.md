- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 실패는 앱 빌드 오류가 아니라 스킬 탐색 실패와 최초 릴리즈 계약 부재로 중단한 결과이며 동일 원인이 앱 수정 회차에 계속 배정되고 있다. 두 장애를 실제 소유 범위에서 해결하고 같은 실행 경로로 검증해야 반복 중지 없이 릴리즈를 진행할 수 있다.
- 수용 기준: 1) 실제 릴리즈 자식이 registry에 지정된 원본 스킬과 상대 참조를 읽는 증거가 있으며 도구가 없는 엔진에서도 지시를 수행한다. 2) 대상 패키지·버전 증가·태그·노트·자산 계약의 승인된 근거가 존재하고 그에 따른 로컬 릴리즈 검증이 통과한다. 3) 같은 프로덕션 호출 경로의 수정 전 실패/수정 후 성공과 스킬 누락·계약 부재의 실패 유지가 검증되어야 하며, 기존 게이트 단위 테스트나 앱 config 검사만으로 완료 처리하지 않는다.
- 건드릴 파일: 현재 앱 worktree 안에서 확인된 원인 수정 대상 없음. 외부 /mnt/c/Users/USER/projects/aidev/bin/run.sh의 agent_plugin_args, dept_note, run_agent, run_codex가 조건부 수정 후보이며 agents/registry.json은 스킬 이름·부서의 정본이다. 외부 tests/test_gate.py의 Release, tests/test_sim.py의 ReleaseSafety 및 tests/sim/run_sim.sh는 검증 범위 참고 자료다. 아래 소유 범위 전제가 해결되기 전에는 이 파일들을 편집하지 않는다.
- 검증 명령: 아래 실행 기록과 조건부 구현 검증 참조.
- 위험과 피할 것: 앱 auth·deploy·.gitlab-ci.yml·의존성·버전 파일은 이번 실패 원인과 연결된 증거가 없다. 게이트 완화, failed→skipped/released 변조, 임의 첫 태그/버전 관례, 외부 러너 직접 수정, 원격 전송, 가짜 에이전트의 성공을 실제 스킬 접근 증명으로 쓰는 일을 금지한다.
- 차선 후보: 최초 릴리즈 계약의 기존 승인 근거 복구 — 이미 승인된 자료가 실제로 발견된 경우에만 정본 문서와 검증에 연결한다. 현재 근거는 미확인이고 새로운 관례를 만들어 대체하지 않는다. 앱 UI 개선으로 우선 과제를 바꾸지 않는다.

## 착수 판정 — 현재 blocked, 재시도 지시 아님

현재 run.json의 project는 aiportal-front-admin, HEAD는 01fedba다. 외부 agents/registry.json의 builder.surface는 회차 worktree이고 COMPANY.md 규칙 1은 자기 표면 밖 쓰기를 금지한다. 정찰 자체는 사용자 절대 규칙상 지정 회차 폴더 외 쓰기가 금지된다. 따라서 이 과제서는 외부 수정 권한을 부여하지 않는다.

동일한 스킬 전달 복구 계획을 다시 수행하라는 과제로 채택하지 말 것. 이전 여섯 회차와 달라진 착수 전제를 찾지 못했다. 구현자는 소유 worktree와 승인된 최초 릴리즈 계약 중 하나라도 여전히 없으면 원장에 **수정 과제 — 미완료(blocked), 과제서 기각: 현 회차에서 실행 가능한 수정 대상 없음**으로 기록한다. 이 기록·과제서 작성·중지 자체는 개선 성과가 아니다. 동일 테스트를 반복하고 이를 진척으로 집계하지 않는다.

## 직접 확인한 원인과 증거

- Skill/skills 호출 도구는 현재 도구 목록에 없다. /mnt/c/Users/USER/projects/headcount/plugins/{pmo,technology}/skills 아래 요청 세 SKILL.md를 직접 읽었다. 릴리즈 두 원본도 같은 headcount의 marketing/technology 아래에 존재하고 읽었다. '원본 파일 부재'와 '자식에게 위치가 전달되지 않음'은 다르다.
- run.sh:245~259의 agent_plugin_args/dept_note는 Claude plugin-dir 및 Skill 호출 안내를 만든다. run_agent:283~308은 그 prompt를 run_codex로 넘기며 run_codex:314~331은 headcount 원본 경로나 엔진별 대체 독법을 전달하지 않는다. 이는 정적 배선 근거이며 과거 자식 실패의 전 원인에 대한 런타임 증명은 아니다.
- release-prompt.md 절차 5는 태그·버전 파일·노트가 모두 없어야 skipped를 허용한다. package.json은 0.0.0, V2는 0.1.0이고 버전 파일이 존재한다. 로컬 태그와 추적 CHANGELOG/릴리즈 파일은 없으며 문서는 GitLab 브랜치 기반 정적 배포를 설명한다. 이번 원격 Release 조회 및 전체 버전 이력의 재검증은 미실행이다.
- .github/workflows는 없고 .gitlab-ci.yml은 기존 앱 build/copy다. 실패 기록에는 해당 job 이름·실행 ID·실패 스크립트 출력이 없다. fixer.sh:52~61은 failed release.json을 앱 이름으로 적재하며 run.sh:1690은 공통 '두 번 실패' 문구를 붙인다. 실제 workflow 두 번 실행 증거는 미확인이다.
- tests/test_gate.py Release는 자산·타입 등의 게이트 검사이며 스킬 로딩을 테스트하지 않는다. test_sim.py는 가짜 에이전트, sim/run_sim.sh는 /tmp 고정 출력과 로컬 모의 원격을 사용한다. 이번 출력 경계 및 실제 자식 회귀 증거 요구에 맞지 않아 전체 sim을 실행하지 않았다.

## 단계별 계획과 체크포인트

1. [확인 완료/착수 차단] 위 소유 범위와 두 독립 장애를 비교한다. 증명: run.json, registry builder.surface, release-prompt 절차 5, 저장된 failed 사유. 사람 질문 없이 현재 회차의 착수 불가를 판단했다.
2. [미착수/전제 충족 시에만] 별도로 정당하게 배정된 소유 worktree에서 registry 기반으로 각 엔진에 같은 스킬 원본과 참조 경로를 제공한다. Claude 기존 경로와 off-switch 계약은 유지한다. 검증은 실제 run_agent→run_codex 자식이 파일을 읽는 회귀여야 한다. 현재 이를 수행하는 기존 테스트 명령은 확인하지 못했으므로 존재하지 않는 테스트 이름을 적지 않는다. 새 테스트를 만들었다면 정확한 명령·출력을 원장에 먼저 기록해야 한다.
3. [미착수] 승인된 릴리즈 계약이 확보된 후 그 계약의 빌드·버전·노트·자산 검증을 실행한다. 계약 없이 다음 버전을 추정하지 않는다. 스킬 접근만 복구돼도 이 단계는 별도 미완료다.
4. [미착수] 같은 실패 경로의 수정 전후 결과를 비교하고 원장에 수정 과제로 기록한다. 누락/실패를 차단하는 기존 gate는 유지한다. 원격 게시·운영 UAT는 이 앱 정찰 과제 밖이다.

## 실제 실행한 검증 (이번 회차)

앱 루트에서 실행:
```sh
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh /mnt/c/Users/USER/projects/aidev/bin/fixer.sh
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-212427-aiportal-front-admin-improve
```
각각 runtime-config OK, 셸 문법 exit 0, 저장된 실패 게이트 exit 1/ok:false/state:failed. 마지막은 저장 결과 재판정이며 실패한 릴리즈 자식의 재실행이 아니다. 수정이 없으므로 수정 후 통과 증거도 없다. Release 단위 테스트 재실행, 실제 자식 회귀, 앱 install/verify/build, UAT는 미실행이다.

의존성이 갖춰진 V2의 실제 명령은 `cd upgrade/admin-v2 && npm ci && npm run build`이며 build가 typecheck, vitest, Vite, runtime/offline/integrity 검사를 포함한다. 현재 node_modules가 없고 이번 정찰의 파일 쓰기 제한이 있으므로 실행하지 않았다. 이것은 스킬 전달 회귀의 대체 검사가 아니다.

## 대안 비교와 추정 근거

- 최소안: 소유 환경에서 기존 registry를 이용한 엔진별 원본 전달. 범위가 가장 작지만 최초 릴리즈 계약은 해결하지 못한다.
- 확장안: 전 엔진용 새 스킬 설치 계층. 범위 L, 45분을 넘길 가능성이 커 제외한다.
- 새 구성 없는 안: 기존 승인 계약과 스킬 원본을 직접 읽는다. 원본 독해는 이번에 가능했으나 승인 계약은 찾지 못해 릴리즈 성공에 불충분하다.
- 현재 권고: 전제 부재를 판정하고 동일 앱 수정 시도를 재개하지 않는다. 고정 우선 과제는 pending으로 보존하며 다른 기능을 끼워 넣지 않는다.

bottom-up 조건부 추정: 배선 수정 10~15분 + 실제 자식 회귀 15~20분 + 기록 5분 = 30~40분, 알려진 경로/환경 차이 예비 5분 별도. 소유 환경 및 승인 계약이 이미 제공된 경우에만 M/35~45분의 낮은 확신 추정이며 완료 확률을 수치로 보장하지 않는다. 이전 반복 no-change의 유사 사례는 현재 조건에서 45분 완료를 지지하지 않으므로 전체 복구 소요는 추정 불가다. 미정 계약/권한 확보는 범위 밖이며 관리 예비는 배정하지 않았다. 실제 자식 첫 재현 후 재추정한다.

적용 스킬 원본: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md. PMO references/sources.md도 확인했으며 위 시간은 외부 벤치마크가 아니라 작업 분해에 따른 가정이다.

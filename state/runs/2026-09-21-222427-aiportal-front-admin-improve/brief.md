- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 릴리즈 실패는 스킬 탐색 실패와 최초 릴리즈 관례 부재를 보고하며, 앱 변경 없이 동일 사유가 반복 적재되고 있다. 실제 원인 소유 환경과 릴리즈 계약이 갖춰져야 동일 검증의 실패→통과를 입증할 수 있고, 앱 CI 수정이나 문서만 추가해서는 이 실패가 복구되지 않는다.
- 수용 기준: 1) 실제 release→run_agent→run_codex 경로에서 요청된 두 스킬과 상대 참조를 읽은 증거가 있어야 한다. 2) 근거가 있는 대상 패키지·다음 버전·태그·노트·자산 계약으로 같은 릴리즈 검증이 통과해야 하며, 기존 failed 결과를 직접 바꾸거나 skipped 조건을 넓히지 않는다. 3) 실제 자식 프로세스로 정상 경로와 스킬 부재·정책 미확정 실패 경로를 검증한다; 모의 에이전트, 문자열 포함 검사, 기존 gate 단위 테스트만으로 완료 처리하지 않는다.
- 건드릴 파일: **현재 앱 worktree에서 승인된 원인 수정 파일은 확인되지 않았다. 이번 착수 파일 없음.** 아래 외부 경로는 읽은 근거이며 편집 지시가 아니다: /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex/release_project — 엔진별 스킬 전달 및 릴리즈 호출; 같은 저장소 release-prompt.md:절차 2·3·5 — 기존 관례 요구; bin/fixer.sh:52~61 — 실패 스냅샷 재적재; bin/gate.py:evaluate_release/cmd_release 및 tests/test_gate.py:Release — 결과 검증. 앱 .gitlab-ci.yml은 별도 브랜치 빌드·복사 파이프라인이며 이 실패의 원인 수정 대상으로 삼지 않는다.
- 검증 명령: 저장소 루트에서 `node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json` (이번 exit 0, 설정 기준선만); `bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh /mnt/c/Users/USER/projects/aidev/bin/fixer.sh` (이번 exit 0, 문법만). 원인 복구 후 같은 결과 검사 명령은 `python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release <실제 재실행의 release.json 절대경로> --out-dir <그 재실행 출력 디렉터리>`이다. 꺾쇠 값은 이번에는 존재하지 않아 실행 불가이며 임의 파일로 대체하지 않는다. 기존 저장 실패 재판정·gate 단위 테스트·전체 sim은 반복 실행하지 않았다. 실제 자식 회귀를 실행하는 기존 안전한 단독 명령은 미확인이다.
- 위험과 피할 것: 코드·커밋·태그·배포·원격 변경 금지(정찰). 구현도 현 회차 worktree 밖 소유 코드를 수정하지 않는다. 최초 버전 관례 신설, release-prompt 절차 5 확대, gate 완화, 실패 상태 덮어쓰기, 앱 auth/CI 변경, 큐 삭제를 해결책으로 쓰지 않는다. 실제 릴리즈 실행은 push 가능한 러너와 연결되어 있으므로 여기서 run.sh를 source하거나 release-only로 실행하지 않는다.
- 차선 후보: 없음 — 우선 과제가 지정되었으므로 앱 개선 후보로 교체하지 않는다. 최초 릴리즈 계약 확보는 동일 과제의 선행조건이며 현 구현자에게 새 관례 작성 과제로 넘기지 않는다.

## 착수 판정과 이전 기각 반영
**pending / blocked / 실행 가능한 수정 과제서 아님.** 이번 run.json의 project는 aiportal-front-admin이고 HEAD는 01fedba다. agents/registry.json의 builder.surface는 “그 회차의 worktree”, COMPANY.md 규칙 1은 “구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다.”이다. 동일한 외부 러너 수정 지시를 다시 채택하라고 요구하지 않는다. 지정 과제는 보존하되, 선행조건 변화가 없는 현재 상태에서 구현자는 이를 실행 가능 과제로 채택해서는 안 된다. 이 문서는 과제 범위·증거·차단 상태를 전달하며 성공 결과를 만들지 않는다.

## 확인된 사실과 미확인
- Skill 도구는 노출된 전체 도구 이름에서 찾지 못했다. headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 technology/skills/implementation-planning/SKILL.md, solution-exploration/SKILL.md 원본을 직접 읽어 절차를 적용했다. estimating-and-contingency/references/sources.md도 읽었다. 별도의 고정 반환 스키마는 없어 사용자 형식을 따른다.
- release-and-deployment 및 product-launch 원본도 로컬에 존재하고 읽힌다. 따라서 현재 머신에 파일 자체가 없다는 주장은 성립하지 않는다. 과거 실패 자식이 무엇을 탐색했는지는 미확인이다.
- run_agent는 Claude plugin-dir와 Skill 문구를 붙이고 run_codex에는 같은 prompt를 넘긴다. run_codex는 headcount 원본 경로나 엔진별 읽기 안내를 추가하지 않는다. 이는 정적 전달 공백이며 수정 시 전체 릴리즈 성공을 보장하지 않는다.
- 로컬 태그 없음, 루트 package.json 0.0.0, V2 package.json 0.1.0. release-prompt는 버전 파일도 없어야 skipped를 허용한다. 저장된 state/aiportal-front-admin.release.json은 failed이며 최초 정책을 찾지 못했다는 사유가 남아 있다. 원격의 현재 릴리즈·추가 운영 계약은 이번 미조회/미확인이다.
- fixer.sh는 failed 스냅샷을 프로젝트명으로 재적재한다. run.sh:1690의 “두 번 실패”는 공통 지시문으로, 두 CI 실행의 로그 증거가 아니다.
- tests/test_gate.py:Release는 evaluate_release의 스키마·자산만 검사한다. tests/test_sim.py 및 tests/sim/run_sim.sh는 가짜 에이전트 및 /tmp 출력을 사용하므로 실제 스킬 접근을 증명하지 못한다.

## 대안 비교 및 실행 순서
A. 앱 워크플로 변경: 원인 연결 없음, 보호 경로 위험으로 기각. B. 외부 러너 스킬 전달 복구: 원인 일부와 연결되지만 소유 범위 밖이며 이전과 같은 재착수 지시는 기각. C. 버전·태그 관례 임의 생성/skip 확대: 절차 위반으로 기각. D. 지정 실패를 미완료로 보존하고 확인된 차단을 정확히 기록: 현 정찰에서 가능한 결론으로 선택하되 수정 성과로 세지 않는다.
1. [완료] 소유 경계와 실제 실패 자료 대조 — 증거는 위 파일·함수와 run.json. 사람 질문 없음.
2. [차단] 소유 환경 및 기존/명시된 릴리즈 계약이 실제로 제공된 뒤 새 실행 계획을 작성해야 한다. 지금은 외부 변경이나 이관을 실행하지 않는다.
3. [미실행] 그 계획에서 실제 자식의 실패 재현→수정→동일 경로 통과→음성 회귀를 순서대로 검증한다. 기존 gate 단위 검사를 최종 완료 증거로 대체하지 않는다.
4. [미실행] 구현 완료 시 원장에 “수정 과제” 및 수정 전후 실행 증거를 기록한다. 현재 기록할 판정은 “수정 과제 — 미완료(blocked)”다.

## 추정 근거
M은 원인 두 축이 준비됐을 때의 조건부 분류다. 분해 추정은 전달 수정 10~15분, 실제 자식 재현·회귀 15~20분, 기록 5분, 알려진 실행 변동 대비 5분으로 35~45분이며, 통계적 신뢰수준은 산정하지 않았다. 유사 회차는 모두 no-change여서 45분 내 실제 복구 가능성을 뒷받침하지 않는다. 소유 환경 확보와 릴리즈 정책 결정은 범위 밖·대기시간 미정이며 관리 예비는 배정되지 않았다; 현재 전체 해결을 45분에 약속할 수 없다.

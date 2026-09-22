- 과제: 수정 과제 — 지정 릴리즈 실패 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 release.json은 스킬 접근 실패와 최초 릴리즈 관례 부재를 보고하며 실제 gate.py에서도 failed로 거부된다. 원인 소유 환경에서 스킬 전달과 명시적 릴리즈 계약을 충족해야 같은 실패의 반복 적재를 끝낼 수 있다.
- 수용 기준: 1) 실제 release 자식이 registry의 marketing:product-launch와 technology:release-and-deployment 원본 및 상대 참조를 읽었다는 실행 증거를 남긴다(Claude 정상 경로와 실제 Codex 폴백 경로 모두). 2) 근거 있는 대상 패키지·버전·태그·노트·자산 계약을 전달하고 기존 절차·빌드 검사를 통과한 새 결과를 생성한다; 계약 부재는 계속 차단한다. 3) 동일 입력의 수정 전 실패/수정 후 성공을 실제 자식과 gate.py release로 증명하며, 기존 Release 테스트의 잘못된 태그·자산 경계 거부가 유지된다. 가짜 에이전트 시뮬레이션이나 소스 문자열 검사는 기준 1·2의 대체 증거가 아니다.
- 건드릴 파일: 현 앱 worktree에는 확인된 원인 수정 파일 없음. 아래는 aidev 소유 환경에서만 적용할 조건부 계획이다: /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — registry에서 원본 경로를 해석하여 엔진에 맞는 접근 방법을 전달; release_project — 승인된 최초 계약 입력을 보존하여 전달. /mnt/c/Users/USER/projects/aidev/tests/test_gate.py:Release 및 tests/test_sim.py:ReleaseSafety — 기존 보호 회귀 유지(수정 필요성은 미확인). /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 1~6 — 읽기 기준; skipped 확대·새 관례 자동 생성 금지. 실제 자식 회귀 테스트 파일은 현재 존재 확인 못 했으므로 기존 파일인 것처럼 명명하지 않는다.
- 검증 명령: 아래 실행 위치·결과 표 참조. 실제 자식 회귀 명령은 현재 없음/미확인; 구현 완료 조건으로 반드시 마련해야 한다.
- 위험과 피할 것: 코드 수정·커밋 금지인 이번 정찰은 지정 경로의 기록만 작성한다. 앱 회차를 외부 러너 수정 회차로 간주하지 않는다. auth, migrations, .gitlab-ci.yml, workflow, release gate, 저장 release.json, 정책·큐를 성공처럼 바꾸지 않는다. 버전 파일 삭제로 skipped 조건을 만들거나 npm patch/태그 관례를 발명하지 않는다. CI URL의 자격증명 값을 기록하지 않는다.
- 차선 후보: 없음 — 우선 과제 고정으로 앱 UI/검사 개선을 대신 수행할 수 없다. 조건이 충족되지 않으면 미완료로 기록하고 동일 착수 불가 계획을 실행 가능 과제로 재승인하지 않는다.

현재 판정: pending / blocked. 과거 기각·no-change의 조건이 바뀌지 않았다. 이번 과제서는 복구 완료나 앱 구현 착수 승인이 아니며, 요구된 “수정 후 동일 검증 통과”는 미충족이다. 현 회차에서 실행 가능한 S/M 코드 수정 과제라고 주장할 근거가 없다.

확인한 인과와 한계:
- HEAD 01fedba, git log -30, 깨끗한 status 확인. 추적 CLAUDE.md/AGENTS.md/.github/CHANGELOG 및 로컬 태그 없음. 네 package/lock 파일의 git log --all + git show 전 이력에서 root 0.0.0 / V2 0.1.0만 확인. 원격 최신 Release는 미확인.
- .gitlab-ci.yml은 branch 기반 legacy build/copy. README, docs/ROADMAP.md, docs/TESTING_GUIDE.md, upgrade/admin-v2/deploy/README.md 및 deploy-changed.sh에서 이 실패에 대응하는 버전·태그 릴리즈 job을 찾지 못했다.
- run.sh:run_agent는 Claude용 --plugin-dir를 설정한 뒤 Skill 안내가 붙은 같은 prompt를 run_codex에 전달한다. run_codex에는 원본 경로 전달이 없다. 스킬 원본은 headcount/plugins에 존재하므로 ‘스킬 자체가 없다’와 ‘자식이 못 찾았다’를 구분한다. 정적 배선은 확인했지만 실패했던 실제 자식의 전체 원인 입증은 미완료다.
- release-prompt 절차 5는 태그·버전 파일·노트가 모두 없는 경우만 skipped. 버전 파일이 있는 현재는 해당하지 않는다. 스킬 경로만 고쳐도 최초 릴리즈 계약 문제는 남는다.
- fixer.sh 52~61은 failed snapshot을 적재하고 run.sh 1690은 두 번 실패 문구를 붙인다. 서로 다른 두 workflow 실행 실패를 증명하는 run ID/로그는 이번에 확보하지 못했다.
- COMPANY.md 규칙 1의 “구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다”와 이번 run.json의 project=aiportal-front-admin을 확인. 사용자 절대 규칙도 정찰 쓰기를 회차 폴더로 한정한다. 이 제약을 문서 한 장으로 변경할 수 없다.

대안 비교와 권고:
1. 최소 수정: 소유 환경에서 registry 기반 엔진별 스킬 전달 + 기존/승인된 최초 계약 연결. 원인 두 갈래를 다룰 수 있어 유일한 복구 권고지만, 쓰기 표면과 계약 확보가 선행 조건이다.
2. 확장형: 공통 스킬 배포 계층과 release 정책 시스템 신설. 여러 프로젝트에 유용할 수 있으나 L 범위여서 이번 제외.
3. 새 장치 없이 앱 README에 스킬 절대 경로만 추가. 환경 의존 경로가 제품 문서에 남고 계약 장애는 해결하지 못하므로 제외.
4. 현 상태 보존: 지금의 권한과 근거에서는 가능한 판정. 복구 성공으로 세지 않으며 다른 앱 과제로 우회하지 않는다.

소유 환경용 구현 순서(이번 회차에서는 미착수):
1. [pending] 승인된 수정 표면 및 최초 계약의 근거를 확보한다. 증명은 run/worktree 소유 정보와 대상 패키지·현재/다음 버전·태그 방식·노트 위치·자산 방법이 채워진 기록이다. 이 전제 없이 다음 단계 금지; 이번 세션에서 사용자 질문은 하지 않는다.
2. [pending] run_agent→실제 run_codex 경로에서 두 스킬 접근 실패를 게시 없는 격리 환경으로 재현한다. 실제 자식 결과와 읽은 원본 경로를 기록한다. 기존 ReleaseSafety의 가짜 에이전트 성공으로 갈음하지 않는다. 자동 체크포인트: 재현 원인이 예상과 다르면 계획 수정.
3. [pending] registry를 정본으로 엔진별 전달을 최소 수정한다. Claude 정상·Codex 폴백 양쪽이 같은 스킬과 상대 참조를 읽고 전역 설정을 변경하지 않는지 실제 자식으로 확인한다. 게이트는 그대로 둔다.
4. [pending] 근거 있는 최초 계약으로 릴리즈 절차와 동일 검사 재실행. 기준 1~3 모두 충족 시에만 원장에 ‘수정 과제 완료’; 아니면 미완료 및 남은 원인을 적는다. 원격 게시·운영 UAT는 이 복구 테스트 범위 밖이다.

검증(2026-09-22, 원문 출력은 validation.json):
| 위치 | 명령 | 결과·증명 범위 |
|---|---|---|
| /mnt/c/Users/USER/projects/aidev | TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-091416-aiportal-front-admin-improve/tmp python3 -B tests/test_gate.py Release -v | 5개 통과, exit 0, ResourceWarning. 변경 전 보호 기준선뿐 |
| /mnt/c/Users/USER/projects/aidev | python3 -B bin/gate.py release state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-091416-aiportal-front-admin-improve | exit 1, state=failed. 저장 결과의 차단 재현이며 실제 릴리즈 자식 재현은 아님 |
| upgrade/admin-v2 | node scripts/validate-runtime-config.mjs public/config/runtime.json | exit 0. 기본 config만 검증 |
| upgrade/admin-v2 | npm ci 다음 npm run build | package.json에서 확인한 실제 명령. 이번 미실행(node_modules 없음), verify(typecheck+vitest)·Vite·runtime/offline/integrity 포함 |
ReleaseSafety는 코드만 읽음; 현재 쓰기 제한 밖 시뮬레이션 생성 가능성 때문에 실행하지 않음. 실제 자식·수정 후 회귀·서버/UAT는 미실행.

견적 근거(estimating-and-contingency 적용):
전제 확보 후 bottom-up: 실패 재현 8~12분, 전달 수정 8~12분, 실제 자식 및 보호 검사 10~15분, 기록 3~5분 = 29~44분. 알려진 자식 지연 contingency 5~10분 별도이므로 34~54분, 통계적 신뢰구간 아닌 낮은 확신의 작업 추정이다. 유사법은 기존 여러 회차가 모두 no-change여서 성공 소요시간 표본이 없어 산출 불가하다. 최초 계약 결정과 환경 이관 소요시간은 미확인·제외, management reserve는 운영자 소관으로 배정하지 않는다. 따라서 전체 복구 45분 보장은 불가하며 M은 전제 확보 후 전달 수정 부분에만 해당한다.

사용 스킬: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md 원본을 직접 읽고 적용. 호출 가능한 Skill 도구 없음. pmo references/sources.md도 확인했으며 외부 문헌의 확률·예비비 수치를 인용하지 않았다. 릴리즈 두 스킬은 실패 원인 조사 목적으로 읽었다.

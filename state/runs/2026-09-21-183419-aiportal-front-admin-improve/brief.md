- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 릴리즈 실패는 앱 워크플로 오류로 입증되지 않았으며, 저장된 실패 사유와 러너 코드에서 엔진별 스킬 전달 누락 및 최초 릴리즈 정책 부재라는 서로 독립적인 차단을 확인했다. 두 조건을 실제 실행으로 해소해야 같은 실패의 재적재가 끝나며, 기존 검사 통과나 중지 기록만으로는 복구되지 않는다.
- 수용 기준: 1) 실제 run_agent → run_codex 자식이 registry에 지정된 두 릴리즈 SKILL.md와 상대 참조를 읽고 절차를 적용하며, Claude 정상 경로도 유지된다. 2) 명시된 프로젝트 릴리즈 계약에 따라 대상 패키지·다음 버전·태그·노트·자산이 정해지고 기존 검증 및 gate.py release를 통과한다. 3) 실제 프로덕션 배선을 통과하는 회귀 검증이 스킬 누락·잘못된 참조·정책 부재를 계속 차단함을 증명한다. 저장된 failed를 수동 released/skipped로 바꾸거나 게이트를 완화한 결과는 불합격이다.
- 건드릴 파일: 현재 aiportal-front-admin worktree에는 확인된 원인 수정 대상이 없다. 아래 aidev 파일은 원인 소유 경로이며 현재 구현자의 편집 목록이 아니다: /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — 엔진에 맞는 스킬 안내와 registry 기반 원본 접근 전달; /mnt/c/Users/USER/projects/aidev/tests/test_sim.py:run 및 tests/sim/run_sim.sh — 실제 자식 검증의 연결 후보(새 회귀 파일·함수는 아직 없음). /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2~5 — 최초 릴리즈 계약 필요성을 확인한 읽기 근거이며 skip 조건 변경 대상이 아니다.
- 검증 명령: 아래의 이번 실행 명령과 향후 복구 검증을 구분한다. 현재 실행 가능한 기존 검사만으로 수용 기준을 달성할 수 없다.
- 위험과 피할 것: 앱 auth/router, .gitlab-ci.yml, 버전·lockfile, 전역 Codex 설정, release.json 및 fix-queue를 우회 수정하지 않는다. 스킬 복사본을 앱에 심거나 새 릴리즈 관례를 임의로 만들지 않는다. 원격 전송·태그·배포·큐 재등록은 이 과제서에서 요청하지 않는다. 실제 효과 없는 문서 수정이나 별도 UI 개선으로 지정 실패를 해결했다고 쓰지 않는다.
- 차선 후보: 없음 — 지정 실패를 다른 앱 개선으로 대체하지 않는다. 원인 소유 범위와 최초 릴리즈 계약이 확보되지 않으면 미완료이며, 동일한 외부 이관·중지를 새 수정 성과로 반복하지 않는다.

착수 판정: **blocked, 앱 구현 세션에서 실행 가능한 S/M 수정안 없음**. 과제 제목은 자동 배정을 유지한 것이며 반복된 복구안을 새로 채택하라는 뜻이 아니다. 이전 여섯 회차와 같은 조건에서 이번에도 코드 수정·성공까지 약속하는 계획은 근거가 없다. 현재 run.json은 project=aiportal-front-admin이고 registry.builder.surface는 그 회차 worktree다. COMPANY.md 규칙 1은 “구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다.”라고 명시한다. 현재 프롬프트는 정찰의 코드 변경도 금지한다. 이 조건 아래 구현자가 외부 러너를 바꾸도록 권한이 있다고 추정하지 않는다.

확인한 인과와 한계:
1. 앱 HEAD는 01fedba, git log -30 확인. .github 없음; .gitlab-ci.yml은 브랜치별 legacy build/copy이며 V2 verify나 태그 릴리즈 job이 아니다. CI 자격증명은 마스킹하여 읽었고 값은 기록하지 않았다. 이번 원격 workflow 실행·실패 로그는 미확인.
2. run.sh:246~259의 agent_plugin_args/dept_note는 Claude plugin/Skill 안내를 구성한다. run_agent:283 이후는 그 안내가 붙은 prompt를 run_codex:314에 전달한다. run_codex의 env -i/exec 인자에는 headcount 원본 경로 안내가 없다. 다만 실제 릴리즈 자식 폴백 실행은 이번 미실행이므로 정적 전달 결함과 과거 실패 사유의 일치를 확인한 수준이다.
3. headcount/plugins/{marketing,technology}/skills/{product-launch,release-and-deployment}/SKILL.md는 실제 존재한다. 따라서 스킬 파일 부재 자체로 단정하면 안 된다. 이번 요청 세 스킬도 원본을 직접 읽었으며 callable Skill 도구는 없다.
4. root package.json은 0.0.0, V2는 0.1.0, 로컬 태그 없음. release-prompt 절차 5는 버전 파일도 없어야 skipped이며 “관례를 새로 정하는 건 사람의 일”이다. OPERATIONS_RUNBOOK/deploy README는 정적 배포와 offline cache를 설명하며 최초 버전/태그 계약은 제공하지 않는다. 원격 릴리즈·운영 계약은 미확인이다.
5. fixer.sh:53~61은 status=failed를 적재한다. run.sh:1688~1690은 실패 종류와 관계없이 “두 번 실패” 문구를 붙인다. run.sh:1873 이후는 결과와 관계없이 fix 큐 항목을 제거한다. 이 때문에 동일 failed의 재적재가 가능한 구조지만 이번에 fixer를 실제 실행하지 않았으며, 문구 수정만으로 릴리즈가 성공하지 않는다.

구현 계획과 체크포인트(조건부, 이번에는 모두 미착수):
1. 원인 소유 aidev 작업 범위 및 운영자가 정한 최초 릴리즈 계약이 있어야 착수한다. 필요한 계약은 root/V2 대상, 버전 증가, 태그 유무/형식, 노트 위치/언어, 자산/빌드 대상이다. 현재 둘 다 확보되지 않았으므로 여기서 구현 미착수로 기록한다. 재이관 자동 호출이나 사용자에게 같은 질문 반복은 하지 않는다.
2. 그 조건이 충족된 별도 소유 작업에서는 registry를 단일 원천으로 Claude 안내와 Codex 원본 경로 안내를 생성한다. 모든 run_codex 호출 경로와 headcount 비활성화 경로도 확인한다. 증명: 실제 프로덕션 함수를 통해 생성된 자식이 실제 파일과 상대 참조를 읽은 실행 근거. 단순 prompt 문자열 검사·가짜 성공 응답·기존 sim의 fake agent만으로 완료하지 않는다. 체크포인트: 누락 경로 실패와 정상 경로 성공을 함께 확인 후 다음 단계.
3. 승인된 릴리즈 계약을 적용하고 해당 패키지 build/검증, 릴리즈 산출물 검사를 실행한다. 아래 gate 명령은 새 산출물에 대해서도 통과해야 한다. 체크포인트: 원격 게시 전까지 기존 품질 게이트 유지. 실패하면 범위 확대 대신 실패 근거를 기록한다.
4. 원장에는 '수정 과제'로 수정 파일·변경 전후 재현·성공 검증을 적는다. 현재는 '미완료(blocked), 코드 수정 없음, 기존 검사만 실행'으로 기록해야 하며 no-change를 복구 완료로 집계하지 않는다.

이번 실제 검증(앱 저장소 cwd에서 실행):
```bash
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-183419-aiportal-front-admin-improve/test-tmp PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v
bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh /mnt/c/Users/USER/projects/aidev/bin/fixer.sh
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-183419-aiportal-front-admin-improve
```
결과: Release 테스트 5개 OK(기존 ResourceWarning), bash -n 통과, runtime-config OK. 마지막 gate는 exit 1/ok:false/state:failed로 기존 차단 재현. 테스트 임시 파일은 이번 출력 폴더 안에만 생성했다. 앱 npm ci/verify/build/UAT 및 실제 모델 폴백·수정 전후 회귀는 미실행. test_sim.py는 fake agent를 쓰며 run_sim.sh가 /tmp를 강제하므로 이번 쓰기 제한과도 맞지 않아 실행하지 않았다.
향후 앱 검증 명령은 `cd upgrade/admin-v2 && npm ci && npm run build`(package.json에 존재, 수분 이상)이나 현재 node_modules가 없고 이를 릴리즈 복구 증거로 실행한 적 없다. 새 실제 자식 회귀 테스트는 아직 존재하지 않으므로 실행 가능한 기존 명령처럼 꾸며 쓰지 않는다.

대안 비교:
- registry 기반 엔진별 원본 전달: 스킬 탐색 실패의 최소 수정. 외부 소유 작업 필요, 최초 릴리즈 계약까지 대체하지 못한다.
- 전역 플러그인 설치·앱 내 스킬 복제: 관리 표면/정본이 늘며 최초 정책 차단도 남으므로 기각.
- 버전 임의 증가 또는 skipped 확대: 관례 신설/게이트 완화에 해당하므로 기각.
- 기존 중지·이관 반복: 코드와 실제 출력이 개선되지 않은 접근이므로 수정 성과로 기각. 현재 blocked 사실은 그대로 보고한다.

추정 근거: 스킬 전달 수정 단독은 원인 소유 작업에서 재현 8~10분 + 배선 수정 10~15분 + 실제 자식/음성 검증 12~15분, 합계 30~40분으로 예상한다. 경로/자식 실행 변동 대응 contingency는 별도 0~5분, management reserve는 배정하지 않았다. 이는 정적 코드 규모에 따른 bottom-up 판단으로 신뢰 낮음; 성공한 유사 회차가 없어 통계적 보장이나 두 번째 추정법으로 검증하지 못했다. 정책 결정 대기와 전체 릴리즈 성공은 이 M 추정에서 제외되며 45분 완료 가능하다고 주장하지 않는다. 가장 큰 가정은 원인 소유 범위와 최초 계약이 확보된다는 것이고 현재 성립하지 않는다.

적용 스킬: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md(작업 분해·범위 추정·예비 분리), technology/skills/implementation-planning/SKILL.md(변경/증명/체크포인트), technology/skills/solution-exploration/SKILL.md(대안 비교). PMO references/sources.md도 읽었으며 외부 비용 기준·통계 수치는 사용하지 않았다. 릴리즈 두 스킬은 실패 원인 조사 자료로 읽었고 릴리즈 실행은 하지 않았다.

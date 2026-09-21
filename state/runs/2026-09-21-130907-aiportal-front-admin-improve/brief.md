- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 수정 (가치 4 / 위험 2 / 작업량 M)
- 왜: 실제 실패는 headcount 스킬 전달 누락과 최초 릴리즈 관례 부재인데, 외부 러너가 이를 앱 워크플로 2회 실패로 안내해 같은 앱 회차에서 해결 불가능한 작업을 반복한다. 스킬 전달을 실제 자식 프로세스까지 복구하고 별도의 릴리즈 정책 근거를 확보해야 동일 실패를 끝낼 수 있다.
- 수용 기준: 1) 실제 run_agent → Claude 한도 실패 → run_codex → 자식 프로세스 경로에서 registry에 지정된 release 스킬 두 원본과 상대 참조를 읽을 수 있으며 없는 Skill 도구를 필수로 요구하지 않는다. 2) headcount 비활성·누락 파일·Claude 정상 경로를 구분하고 기존 권한·원격 전송 차단·release 게이트를 유지한다. 3) 회귀 검증은 수정 전 실패/수정 후 성공을 보이고, 실제 릴리즈 완료는 승인된 최초 릴리즈 정책 근거와 그 정책의 빌드·검증을 충족한 별도 release.json에 대해서만 기록한다.
- 건드릴 파일: 현재 앱 worktree에는 적합한 수정 파일 없음. 외부 aidev 소유 작업에서만 bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — 엔진별 스킬 전달; tests/sim/run_sim.sh 및 tests/sim/bin/claude — 실제 폴백 분기 입력; tests/test_sim.py — 프로덕션 호출 경로 검증; tests/sim/bin/codex(신규 후보) — 별도 자식 프로세스에서 전달된 원본을 실제로 읽는 경계 실행 파일. agents/registry.json은 읽기 기준이며 이번 수정으로 권한을 확대하지 않는다.
- 검증 명령: 아래 실행 결과와 구현 검증 순서 참조. 기존 test_gate.py Release는 전달 버그를 검증하지 않으므로 그것만 통과시켜 완료 처리하지 않는다.
- 위험과 피할 것: 앱 auth/·migrations·.gitlab-ci.yml·package/lock 버전·배포 스크립트는 이 실패의 수정 대상이 아니다. 스킬 이름만 프롬프트에 복사하거나 Skill 도구가 있다고 주장하기, 원본 상대 참조가 깨지는 본문 복사, 버전 관례 임의 신설, failed를 skipped/released로 바꾸기, 게이트 완화, 기존 no-change 이관 문서 재생성·형식적 앱 커밋을 금지한다.
- 차선 후보: 없음 — 우선 과제 고정 및 두 번의 외부 이관/no-change 이력 때문에 무관한 앱 개선이나 같은 중지 작업을 대체 성과로 선택하지 않는다. 실행 가능한 수정이 없다는 판정 자체를 성공으로 세지 않는다.

현재 판정과 권한

이번 정찰은 읽기 전용이다. 코드 수정·커밋·태그·릴리즈 성공 판정을 하지 않았다. run.json의 project는 aiportal-front-admin이고 HEAD는 01fedba, worktree는 깨끗하다. /mnt/c/Users/USER/projects/aidev/agents/registry.json의 builder.surface는 ‘그 회차의 worktree’이고 COMPANY.md의 규칙 1은 구현자의 표면 밖 쓰기를 금지한다. 따라서 이 과제는 pending/blocked이며 현재 앱 구현자에게 외부 러너 편집을 지시하는 권한 부여가 아니다. 이미 실패한 ‘외부 이관 후 종료’를 새 구현 과제로 다시 채택하지 않는다. 현 제약 아래 45분 안에 릴리즈 전체를 완료할 수 있는 과제는 확인되지 않았다.

확인한 근거

1. aidev/bin/run.sh:245–259의 agent_plugin_args/dept_note는 HEADCOUNT_DIR의 Claude 플러그인과 Skill 안내를 구성한다. :283–312의 run_agent는 그 안내를 붙인 prompt를 Codex 폴백에도 전달한다. :314–332의 run_codex는 env -i와 codex exec를 사용하지만 원본 스킬 경로·부서 플러그인 전달이 없다. 실제 Codex 자식 재현은 이번에 미실행이므로 정적 배선 결함과 런타임 증거를 구분한다.
2. 원본은 /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/release-and-deployment/SKILL.md와 /mnt/c/Users/USER/projects/headcount/plugins/marketing/skills/product-launch/SKILL.md에 존재하며 읽었다. 현재 도구 목록에는 Skill 호출 도구가 없었다. 정찰용 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration도 같은 headcount 경로의 원본을 직접 읽어 적용했다.
3. aidev/release-prompt.md 절차 5는 태그·버전 파일·노트가 모두 없어야 skipped를 허용한다. 앱에는 root 0.0.0/V2 0.1.0 버전 파일이 있고 태그 목록은 비었다. docs/OPERATIONS_RUNBOOK.md와 upgrade/admin-v2/deploy/README.md는 브랜치 기반 정적 배포·npm cache를 설명한다. 최초 버전·태그·노트 운영 결정은 미확보이며 스킬 전달 수정만으로 해소되지 않는다.
4. aidev/bin/fixer.sh:53–61은 저장된 status=failed를 큐에 넣는다. run.sh:1688–1690은 원인과 관계없이 ‘워크플로가 두 번 실패’ 문구를 붙인다. 실제 workflow 재시도 분기는 retry_release_workflow(:974 부근)에 별도로 있다. 이 회차 실패 파일에는 태그가 비었고 실패 job/run ID가 없다. 실제 GitHub 워크플로 2회 실패는 미확인이다.
5. .github/workflows는 없고 앱 .gitlab-ci.yml은 main/develop의 레거시 build/copy이다(민감 URL은 가려 읽음). tests/test_gate.py:Release는 상태·타입·자산 검증 5개뿐이다. tests/sim/run_sim.sh는 실제 run.sh를 실행하지만 현재 모의 claude는 성공을 반환하고 모의 codex 실행 파일도 없다. 기존 테스트 통과는 이 폴백 결함의 반증이 아니다.

소유 러너 작업 환경에서 수행할 구현 계획 (현재는 전 단계 미착수)

1. 변경: 기존 sim harness에 Claude 한도 실패와 별도 Codex 자식 실행 시나리오를 추가한다. 원본 스킬 내용·references 접근 성공을 자식이 실제 파일 읽기로 보고하도록 하며, run_agent/run_codex를 대역 함수나 추출한 코드 복제본으로 대체하지 않는다. 증명: python3 tests/test_sim.py의 새 대상 테스트가 현 코드에서 원본 위치 미전달 때문에 실패해야 한다. 기존 harness가 수정되어야 하므로 새 테스트의 클래스명/명령은 구현 시 확정한다. 체크포인트: 예상한 실패를 확인한 뒤 2로 진행; 다른 이유로 실패하면 계획을 고친다.
2. 변경: registry의 phase별 skills에서 원본 절대 경로와 그 디렉터리의 참조 읽기 안내를 생성해 Codex 호출에 전달한다. Claude용 안내가 중복·상충하지 않게 원래 작업 프롬프트와 엔진 안내를 분리한다. 미존재 파일은 명시적 진단으로 실패하고 headcount=false/NO-HEADCOUNT는 기존 off 계약을 지킨다. 증명: 같은 자식 프로세스 테스트 성공, Claude 정상 경로와 off/누락 경로 회귀 통과. 체크포인트: 자동 검증 통과; 사람이 검토할 별도 예외 권한은 가정하지 않는다.
3. 변경: 성공 여부를 원장에 분리 기록한다. 증명: bash -n bin/run.sh, python3 tests/test_gate.py, python3 tests/test_sim.py(전체는 여러 분 이상 가능). 전달 수정 부분을 되돌렸을 때 새 테스트만 다시 실패하는지 확인한다. 로컬 경계 실행 파일은 전달/읽기 계약만 증명하며 모델의 스킬 준수까지 증명하지 않는다. 실제 재릴리즈는 최초 정책 근거를 확보한 후에만 별도 수행한다. 그 근거가 없으면 전체 릴리즈는 미완료이다.

이번 정찰에서 실제 실행한 검증

- bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh → exit 0.
- bash -n /mnt/c/Users/USER/projects/aidev/bin/fixer.sh → exit 0.
- TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-130907-aiportal-front-admin-improve/validation-tmp PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v → 5개 통과, 기존 unclosed file ResourceWarning 있음.
- PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-130907-aiportal-front-admin-improve → exit 1, ok:false/state:failed. 기존 실패의 재현이며 수정 후 통과 아님.
- 앱 node_modules 없음. npm ci/build, 전체 sim, 실제 Codex 폴백, 서버/UAT 및 원격 workflow 조회는 미실행. 이전 193개 앱 테스트 통과는 이전 회차 기록이다.

대안 및 견적 근거

- 최소안: 원본 경로를 엔진별로 전달하고 실제 폴백 배선을 검증한다. 결함에 직접 대응하고 전역 설치가 없어 추천한다. 가장 큰 가정은 소유 러너 작업 환경이 별도로 제공된다는 것이다.
- 확대안: 모든 엔진용 스킬 패키징/전역 설치 계층 신설은 표면과 회귀 범위가 커 이번 M 범위에서 제외한다.
- 무변경안: 기존 failed를 유지하면 허위 릴리즈는 막지만 반복 실패는 해결되지 않는다. 현재 앱 회차의 상태가 여기에 해당하며 완료로 처리하지 않는다.
- 버전 신설/skip 확대/앱 문서 우회는 승인된 정책 근거가 없어 허용 가능한 해결안이 아니다.
- Bottom-up 예상: 회귀 배선 12–18분, 전달 수정 8–12분, 검증·기록 8–10분 = 28–40분. 알려진 harness 조정 여유 0–5분을 별도 두어 28–45분이며 통계적 신뢰수준은 미측정, 신뢰는 낮음이다. 이전 두 회차는 no-change여서 유효한 유사 구현 소요 자료가 없다. 최초 정책 결정·소유 환경 마련·실제 릴리즈는 견적 밖이고 소요 미확인; 이를 45분 완료 약속으로 해석하지 않는다. 관리 예비는 배정하지 않았다.

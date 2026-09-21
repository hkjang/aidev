- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 릴리즈 실패는 원본 스킬 탐색 실패와 최초 릴리즈 계약 부재이며, 앱 CI가 두 번 실패했다는 증거는 없다. 원인 소유 범위와 검증 대상을 맞춰야 같은 앱 회차의 no-change 반복을 끝내고 실제 릴리즈 복구를 검증할 수 있다.
- 수용 기준: 1) 실제 run_agent→Claude 한도→run_codex 경로에서 registry의 릴리즈 스킬 두 원본과 상대 참조를 자식이 읽고 절차를 적용한다. 2) 명시적으로 결정된 대상 패키지·다음 버전·태그·노트·자산 계약에 따라 릴리즈 검증이 성공한다(계약은 현재 미확보). 3) 동일 경로의 수정 전 실패/수정 후 성공 증거 및 스킬 부재·정책 미확정 시 실패 유지 증거가 있어야 하며, 기존 gate.py release의 차단을 유지한다.
- 건드릴 파일: 현재 aiportal-front-admin worktree에는 확인된 원인 수정 대상 없음. 외부 원인 위치는 /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — 엔진별 원본 스킬 전달; /mnt/c/Users/USER/projects/aidev/tests/test_sim.py 및 tests/sim/run_sim.sh — 실제 러너 경유 회귀 검증의 현행 진입점. 이 경로들은 현재 구현자의 쓰기 표면이 아니며 이 과제서로 권한을 확장하지 않는다.
- 검증 명령: 아래 실행 가능한 현행 명령과 검증 한계를 따른다.
- 위험과 피할 것: 앱 .gitlab-ci.yml/auth/router, 버전·lockfile, gate.py, release-prompt.md 절차 5를 우회 목적으로 바꾸지 않는다. 운영 release.json·fix-queue·registry를 직접 변경하지 않는다. 기존 스킬 전달 이관/착수 중지를 완료 과제로 재계상하지 않는다. Skill 도구가 없는 환경에서 도구를 호출했다고 기록하지 않는다.
- 차선 후보: 없음 — 지정 과제를 대체할 수 있는 앱 내부 원인 수정은 확인되지 않았다. 일반 UI/DX 후보로 전환하거나 중복 이관 문서를 구현 성과로 만드는 것은 허용하지 않는다.

현재 판정: **미완료(blocked), 구현 착수 가능한 앱 수정안 없음.** 이 파일은 반복된 외부 수정안을 이번 앱 worktree에서 실행하라는 지시가 아니다. 사용자가 지정한 수정 목표는 유지하되, 수용 기준을 만족시킬 수 없는 배정이라는 사실을 명시한다. 45분 안에 전체 복구가 가능하다고 보장할 근거는 없다.

확인한 사실과 근거

1. 앱 HEAD 01fedba, 최근 git log -30 확인. 로컬 태그 출력 없음, .github/·루트 CLAUDE.md·AGENTS.md 없음. root package.json 0.0.0, V2 package.json 0.1.0. README, docs/ROADMAP.md, TESTING_GUIDE.md, OPERATIONS_RUNBOOK.md, V2 deploy/README.md, QUICK_WINS.md와 CI를 읽었다. 최초 태그 계약은 확인하지 못했다. 원격 Release 현황은 이번에 조회하지 않았으므로 미확인이다.
2. state/aiportal-front-admin.release.json은 status=failed, version/tag가 비어 있고 스킬 탐색 실패·관례 부재를 이유로 기록한다. run.sh:dept_note는 Skill 안내를 만들고 run_agent는 그 안내를 붙인 prompt를 run_codex에 그대로 넘긴다. run_codex 호출에는 headcount 원본 경로 전달이 없다. 실제 모델 폴백 재현은 미실행이다.
3. 원본은 /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/release-and-deployment/SKILL.md와 marketing/skills/product-launch/SKILL.md에 실제 존재하며 이번에 읽었다. 스킬 본문만으로 이 앱의 다음 버전과 태그 관례가 정해지지는 않는다.
4. bin/fixer.sh:53–61은 모든 failed 릴리즈를 앱 이름으로 적재한다. run.sh:1688–1690은 큐의 발생 원인과 무관하게 두 번 실패 문구를 붙인다. run.sh:1873 이후는 수정 회차 결과와 무관하게 해당 큐 행을 지우고, failed release 상태는 남을 수 있다. 재적재 루프의 정적 근거이며 스케줄러를 실제 실행한 증거는 아니다.
5. 현재 run.json은 project=aiportal-front-admin. 외부 agents/registry.json의 builder.surface는 그 회차 worktree이고 COMPANY.md 규칙 1은 자기 표면 밖 쓰기를 금지한다. 이 사실이 이전 중지 판정을 뒤집을 새 권한이나 앱 내부 수정점을 제공하지 않는다.
6. .gitlab-ci.yml은 브랜치 기반 레거시 build/copy. dev_build_main의 core 모드, ofc_deploy_dev의 중복 rules는 별도 위험이며 이번 저장된 실패 원인으로 연결할 근거가 없다. 민감 URL은 기록하지 않았다.

구현 계획과 체크포인트

- 현재 단계 [완료]: 원인·소유 범위·저장 실패와 실제 검증 결과 대조. 증거는 verification.txt. 사람에게 질문하지 않았으며 코드 변경도 없다.
- 현재 단계 [blocked]: 현재 구현자는 앱 파일을 임의로 수정하지 않고 원장에 '수정 과제 — 미완료(blocked): 원인 수정 표면과 릴리즈 계약 미확보'로 기록한다. 이는 수용 기준 충족이나 새 개선 성과가 아니다. 같은 이관 문서를 다시 만들지 않는다.
- 후속 소유 작업의 설계 참고 [미실행]: 외부 러너 소유 회차가 실제로 배정되면 registry에서 엔진별 스킬 전달을 만들고 Claude의 플러그인 경로와 Codex의 원본 읽기 경로를 각각 검증한다. 비활성 정책과 파일 부재도 검사하며 전역 설정을 바꾸지 않는다. 실제 run.sh 경로를 통과시키고 원본 파일 읽기를 관찰해야 한다. 손으로 만든 성공 JSON, 추출한 함수만 실행, 소스 문자열 검사, 가짜 모델의 성공 응답만으로 완료 처리하지 않는다.
- 후속 릴리즈 [blocked]: 최초 릴리즈 계약이 실제로 확보된 뒤 그 계약의 로컬 명령으로 릴리즈를 재현한다. 현재는 정확한 전체 성공 명령을 정할 수 없다. 게이트 단위 테스트 성공을 전체 릴리즈 성공으로 대체하지 않는다.

이번에 실제 실행한 명령(저장소 루트 기준)

```bash
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-173416-aiportal-front-admin-improve PYTHONDONTWRITEBYTECODE=1 python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v
bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh
bash -n /mnt/c/Users/USER/projects/aidev/bin/fixer.sh
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-173416-aiportal-front-admin-improve
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
```

결과: Release 5개 통과(exit 0, 기존 ResourceWarning), bash 문법 두 개 exit 0, 저장 failed 결과 gate exit 1/ok:false/state:failed, runtime config exit 0. 원본 전달/최초 릴리즈 복구를 증명하지 않는다. node_modules가 없어 앱 verify/build는 미실행. tests/sim/run_sim.sh는 /tmp 출력 경로가 고정되어 이번 파일 쓰기 제한하에서는 실행하지 않았다. 실제 서버/UAT도 미실행이다.

대안 비교와 추정 근거

- 엔진별 원본 전달: 코드상 누락에 직접 대응하지만 외부 소유 환경이 필수다. 이전과 같은 앱 회차에 재배정하는 안은 실행안으로 다시 추천하지 않는다.
- 앱 문서에 임의의 최초 릴리즈 규칙 신설: 정책 결정을 가장하므로 기각한다.
- failed/skipped 게이트 완화: 실패 은폐이며 명시 금지이므로 기각한다.
- 현상 유지: 구현 성과 없음. 현재 허용 범위에서 정직한 판정이며 지정 과제는 pending으로 남긴다.

M은 권한·원본·테스트 환경이 갖춰진 후 스킬 전달 수정 부분만의 잠정 크기다. bottom-up 참고치: 경로 재현 8–12분, 전달 수정 10–15분, 실제 경로 검증 10–15분, 기록 2–3분 = 30–45분(낮은 확신, 통계적 신뢰구간 아님). 알려진 폴백 접근 변수의 예비 시간 5–10분은 별도이며 총 35–55분이다. 최초 릴리즈 정책 결정과 외부 소유 환경 확보는 추정 밖이고 management reserve는 책정하지 않았다. 과거 회차는 모두 no-change여서 성공 유사사례 기반 교차 추정은 불가하다. 따라서 이번 배정의 전체 복구를 45분짜리 실행 가능 과제로 제시하지 않는다.

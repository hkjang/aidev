- 과제: 수정 과제 — 지정 릴리즈 실패의 엔진별 스킬 입력 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 외부 aidev 러너의 run_agent는 Claude용 Skill 안내를 붙인 prompt를 Codex 폴백에도 그대로 넘기고, run_codex에는 headcount 원본 경로를 전달하는 배선이 없다. 엔진이 실제로 읽을 수 있는 스킬 입력을 제공하면 보고된 스킬 탐색 실패를 고칠 수 있지만, 최초 릴리즈 계약 부재는 별도 선행 입력이므로 이것만으로 전체 릴리즈 성공을 주장할 수 없다.
- 수용 기준: 1) 실제 run_agent → run_codex 경로의 자식이 registry에 지정된 릴리즈 두 스킬 원본 및 필요한 상대 참조를 읽은 실행 증거가 남는다. 2) 기존 관례 또는 소유자가 확정한 최초 계약(대상 root/V2, 동기화 버전 파일, 다음 버전·태그·노트·자산·검증)을 사용해 동일 릴리즈 절차를 로컬에서 완료한다; 계약 미확보는 failed를 유지하며 skipped/released로 바꾸지 않는다. 3) 수정 전 실패하던 실제 자식 재현이 수정 후 통과하고 Release 보호 검사 및 원래 앱 검증이 계속 통과한다; 대역 자식·소스 문자열 검사·기존 5개 테스트만으로 완료하지 않는다.
- 건드릴 파일: 현 aiportal-front-admin worktree 안에는 입증된 직접 수정 대상 없음. 원인 소유 저장소의 /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — registry 기반 엔진별 원본 경로 전달과 기존 Claude plugin 전달 보존; /mnt/c/Users/USER/projects/aidev/agents/registry.json — 기존 phase/skills 계약을 읽되 이름을 별도 하드코딩하지 않음; /mnt/c/Users/USER/projects/aidev/tests/test_sim.py:ReleaseSafety 및 tests/sim/run_sim.sh — 실제 자식 검증을 추가할 때 기존 대역 검사와 구분하고 임시 쓰기 경계를 지킴. 이는 소유 환경에서의 수정 명세이며 현 회차 외부 편집 지시가 아니다.
- 검증 명령: 아래 실행 기록과 조건부 검증 순서를 따른다.
- 위험과 피할 것: 앱 package.json 버전·CHANGELOG·태그 관례를 임의 생성하지 않는다. .gitlab-ci.yml, auth, deploy, gate.py, release-prompt 절차 5, 저장 failed JSON을 통과 목적으로 변경하지 않는다. 운영 run.sh 전체를 source/실행하면 네트워크·푸시·worktree 정리 부작용이 있으므로 진단용으로 실행하지 않는다. 외부 코드 수정은 COMPANY.md 규칙 1의 구현 표면 밖이다.
- 차선 후보: 없음(지정 실패 우선). 관련 선행 작업은 최초 릴리즈 계약 확보이며 앱 UI·오프라인 검사 후보로 대체하지 않는다.

현재 판정: pending/blocked. 이 문서는 실행 가능한 앱 수정 과제로 재승인하는 문서가 아니다. 이전과 같은 차단 기록이나 기준선 통과를 개선 성과로 다시 제출하지 않는다. 정찰 역할의 코드 변경 금지와 쓰기 경계 때문에 수정 및 수정 후 동일 검증 통과 요구는 이번에 충족하지 못했다.

관찰한 근거
- HEAD 01fedba, git log -30 및 깨끗한 git status 확인. 로컬 태그 없음. rg 파일 검색에서 AGENTS.md/CLAUDE.md/CHANGELOG/.github workflow 발견 못 함. 원격 최신 상태와 전체 버전 이력은 이번 미확인.
- root package.json은 0.0.0, upgrade/admin-v2/package.json은 0.1.0. release-prompt.md 절차 5의 태그·버전 파일·노트 모두 없는 skipped 조건과 다르다.
- 실제 앱 .gitlab-ci.yml과 docs/OPERATIONS_RUNBOOK.md, upgrade/admin-v2/deploy/README.md는 브랜치 build/copy 및 오프라인 cache 배포를 설명한다. 이 문서에서 다음 태그·버전 관례는 확보하지 못했다. 자격증명은 마스킹 후 읽었고 값은 기록하지 않았다.
- run.sh 246~335: agent_plugin_args는 Claude plugin-dir, dept_note는 Skill 호출 안내, run_agent는 같은 prompt를 run_codex로 전달한다. 실제 자식 실패의 전체 인과는 미입증이다.
- run.sh:release_context는 gh 조회 오류를 '(없음)'으로 표시할 수 있다. release_project는 gate 실패 시 게시 전에 돌아간다. fixer.sh 52~61은 failed snapshot을 다시 적재하고 run.sh 1690은 두 번 workflow 실패 문구를 붙인다. 실제 workflow 두 실행의 ID/로그는 확보하지 못했다.
- tests/test_gate.py:Release의 5개는 상태·태그·타입·자산 경계를 검사한다. tests/test_sim.py:ReleaseSafety는 모의 도구 기반으로 실제 Codex 스킬 접근 증거가 아니다. tests/sim/run_sim.sh의 /tmp/aidev-sim 고정 쓰기는 현 회차 제한 밖이다.

실행 기록(수정 전 기준선, validation.json 원문)
```bash
# cwd: /mnt/c/Users/USER/projects/aidev
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-152428-aiportal-front-admin-improve/tmp PYTHONDONTWRITEBYTECODE=1 python3 -B tests/test_gate.py Release -v
# exit 0, 5 tests OK, ResourceWarning 있음

# cwd: /home/hkjang/.cache/auto-improve-wt/aiportal-front-admin
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
# exit 0
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-152428-aiportal-front-admin-improve
# exit 1, state=failed — 고쳐진 릴리즈가 아님
```

소유 환경에서의 조건부 실행 계획(모든 단계 미착수)
1. 실제 자식 재현을 먼저 만든다. run_agent/run_codex production 배선, 실제 Codex 프로세스, registry의 두 원본과 참조 읽기를 증명하되 게시 기능은 실행하지 않는 격리 경계를 준비한다. 현재 제공된 테스트에는 이 검증 명령이 없다; 있는 척 명령을 만들지 않는다. 체크포인트: 동일 스킬 접근 실패를 재현 못 하면 원인 가설을 고치고 다음 단계에 가지 않는다. 사람 질문은 이번 회차에 하지 않음.
2. 원인 소유 worktree에서 엔진별 입력 전달만 고친다. Claude의 plugin-dir 유지, Codex에는 검증된 절대 원본 경로/참조 기준을 전달하며 전역 설정은 변경하지 않는다. 증명: 1의 같은 실제 자식 재현 + 위 Release 명령. 체크포인트: 실제 읽기 증거와 보호 검사 모두 통과해야 다음 단계.
3. 최초 계약을 확보한 경우에만 release_context/release_project의 동일 로컬 준비 절차를 검증한다. V2 대상이면 package scripts에 있는 `npm ci`, `npm run verify`, `npm run build`를 V2 cwd에서 실행한다(이번 미실행). root 대상이면 crypto-js tarball/환경 입력 선행 필요. 체크포인트: 계약 및 대상 검증이 없으면 전체 복구 완료 금지. 서버/UAT·원격 게시는 별도이며 이번 수행하지 않는다.

대안과 선택
- 최소 수정: 엔진별 스킬 입력 전달. 확인된 구조적 차이를 좁게 고치므로 선택했지만 전체 릴리즈 복구와 구분한다.
- 확장안: 공통 스킬 로더/컨텍스트 스키마를 새로 구축. 적용 범위가 커 45분 과제로 부적합.
- 새 구성 없는 안: 사람이 매번 원본 위치를 알려 주거나 Claude만 재시도. 현재 무인 운영 및 폴백 요구를 충족하지 못한다.
- 앱에 릴리즈 관례를 새로 만들거나 skipped 범위를 넓힘: 근거 없는 계약 또는 게이트 완화이므로 기각.
가장 큰 가정은 소유 러너 worktree와 실제 자식 검증 환경을 확보할 수 있다는 점이다. 현재 run.json의 project는 aiportal-front-admin이며 이 가정이 충족되지 않았다.

추정 근거
M은 소유 환경/재현/계약이 이미 확보된 경우의 입력 전달 수정 범위만 뜻한다. 바텀업 작업 가정: 전달 수정 10~15분 + 실제 자식 검증 10~15분 + 보호 검사/기록 5분 = 25~35분; 알려진 환경 변동 예비 5~10분을 별도로 둬 30~45분이며 경험적 낮은 확신 범위다(통계적 신뢰구간 아님). 관리 예비·정책 결정·새 harness 구축·전체 최초 릴리즈 시간은 포함하지 않았고 상한 미확인이다. 유사 회차는 반복 no-change여서 완료 소요 비교 자료가 없으므로 45분 전체 복구 약속을 하지 않는다.

스킬 적용
Skill 호출 도구는 사용 가능한 도구 목록에서 찾지 못했다. 다음 원본을 직접 읽어 대체 적용했다:
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md (references/sources.md도 확인; 본 추정은 로컬 작업 가정이며 외부 표준 수치 인용 없음)
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
릴리즈 두 스킬 원본도 존재함을 확인하고 읽었다. 이는 실제 릴리즈 자식에게 스킬이 전달됐다는 증거가 아니다.

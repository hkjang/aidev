- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 외부 aidev의 run_agent가 Claude 전용 Skill 안내를 run_codex로 넘기면서 원본 경로를 전달하지 않고, 이 앱에는 최초 릴리즈 버전·태그·노트 관례의 근거도 확보되지 않았다. 실제 전달 복구와 릴리즈 계약 확보가 모두 증명돼야 같은 실패가 해소되며 앱 CI 완화나 버전 추측은 해결이 아니다.
- 수용 기준: 1) 실제 run_agent→run_codex 자식 실행에서 registry의 release 스킬 두 원본과 필요한 상대 참조를 읽고 절차에 사용함을 확인한다. 2) 근거가 명시된 앱 최초 릴리즈 계약(대상 패키지, 버전, 태그, 노트, 자산)을 사용해 동일 릴리즈 단계의 로컬 검증이 통과한다. 3) 전달 결함 회귀가 수정 전 실패·수정 후 통과하고, 원본 누락·잘못된 자산·정책 미확정은 계속 차단됨을 증명한다. 게이트 단위 테스트 통과, 중지 기록, 이관 문서만으로 완료하지 않는다.
- 건드릴 파일: 현재 앱 worktree 안에는 확인된 수정 대상이 없다. 원인 위치는 외부 `/mnt/c/Users/USER/projects/aidev/bin/run.sh`:agent_plugin_args/dept_note/run_agent/run_codex(241–345행)의 엔진별 스킬 전달; 외부 `tests/test_sim.py`:run/ReleaseSafety 및 `tests/sim/run_sim.sh`의 실제 러너 배선 회귀다. `release-prompt.md`:절차 2–5는 관례 제약을 확인한 참조 파일이며 skip 조건을 바꿀 대상이 아니다. `agents/registry.json`의 스킬 목록을 정본으로 사용하고 앱에 스킬 사본을 추가하지 않는다.
- 검증 명령: 아래 실행 결과 및 후속 계획 참조. 이 앱의 npm build는 외부 전달 결함 검증을 대체하지 않는다.
- 위험과 피할 것: 앱 `.gitlab-ci.yml`, auth/, 버전 파일을 원인 없이 수정하지 않는다. gate.py 완화, failed→skipped 조작, 임의 최초 태그, 전역 Codex 설정 덮어쓰기, 원격 전송 금지. 외부 러너 변경은 현재 builder.surface(그 회차 worktree) 밖이다. 지정 과제를 일반 UI 개선으로 대체하지 않는다.
- 차선 후보: 없음 — 지정 실패의 수용 기준을 충족하는 앱 내부 대안이 확인되지 않았다. 동일한 중지/이관 접근은 앞선 회차에서 반복 실패했으므로 새 수정 성과나 차선으로 제시하지 않는다.

현재 판정: **blocked, 구현 가능한 45분 앱 과제로 성립하지 않음**. 이 문서는 재이관 실행을 지시하지 않는다. 구현자는 현재 범위에서 외부 코드를 편집하거나 운영 결정을 만들어 내지 말고, 실제 수정 증거가 없는 한 원장에 `수정 과제 — 미완료(blocked)`로 기록해야 한다. 정찰 산출물 작성 완료와 릴리즈 복구 완료는 별개다.

확인한 근거
- 앱 HEAD=01fedba, 최근 git log -30 확인. CLAUDE.md/AGENTS.md는 앱 파일 검색에서 발견되지 않음. README, docs/ROADMAP.md, OPERATIONS_RUNBOOK.md, TESTING_GUIDE.md, tests/README.md, V2 package.json/vite.config.ts 및 목록·정책 화면을 읽었다.
- .gitlab-ci.yml은 GitLab 브랜치 기반 루트 build/copy이며 민감 remote URL은 가려 읽었다. GitHub workflow 파일은 없고 로컬 태그도 없다. root=0.0.0/V2=0.1.0. 전체 버전 이력 재조사는 이번에 하지 않았으며 최초부터 같은 값이라는 내용은 이전 기록 근거다.
- 외부 fixer.sh:53–61은 status=failed를 모두 큐에 넣고 run.sh:1688–1690은 원인 구분 없이 '두 번 실패'를 붙인다. 실제 GitHub workflow 2회 실패 로그는 미확인이다. 문구 정정만으로 이번 릴리즈 실패를 고쳤다고 할 수 없다.
- 현재 run.json project=aiportal-front-admin. 외부 COMPANY.md:15,32 및 registry builder.surface는 작업 표면을 해당 worktree로 제한한다.
- 요청된 정찰 스킬 3개는 로컬 headcount/plugins/{부서}/skills/{이름}/SKILL.md를 직접 읽어 적용했다. Skill 호출 도구는 사용 가능한 도구 목록에 없다. 릴리즈의 marketing:product-launch, technology:release-and-deployment 원본도 존재함을 직접 읽었지만 그 사실만으로 자식 전달 성공을 주장하지 않는다.

실행 결과(이번 정찰)
1. `TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-151413-aiportal-front-admin-improve/assets PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v` → 5개 통과. Release.setUp의 파일 close 누락 ResourceWarning 발생. 테스트 임시 파일은 이번 산출물 디렉터리 안에만 생성.
2. `bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh /mnt/c/Users/USER/projects/aidev/bin/fixer.sh` → exit 0.
3. `PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state` → exit 1, ok:false/state:failed. 저장된 기존 실패의 차단 재현이며 복구 성공이 아니다.
4. V2 node_modules 없음. 앱 설치·build·UAT, 실제 모델 폴백, 전체 sim 미실행. sim/run_sim.sh는 /tmp에 쓰므로 이번 정찰의 산출물 경로 제한하에서 실행하지 않았다.

실행 계획(원인 소유 worktree와 명시적 최초 릴리즈 계약이 확보될 경우에만 유효; 현재 모두 pending)
1. registry 기반 스킬 이름→실제 경로 해석을 엔진 호출 경계에 전달한다. Claude 경로를 유지하면서 Codex에는 읽을 원본 절대경로·상대 참조 기준을 제공한다. 증명: 실제 run_agent 폴백 자식이 원본 파일에 접근; 소스 문자열 검사나 성공을 하드코딩한 가짜 에이전트는 증거 불가. 체크포인트: 자식 접근 확인 전 다음 단계 착수 금지, 별도 사람 확인 요청 없음.
2. tests/test_sim.py와 sim 실행 경계를 통해 수정 전/후 동일 시나리오를 재현한다. 기존 `python3 tests/test_gate.py Release -v`, `python3 tests/test_sim.py ReleaseSafety -v`를 소유 worktree에서 실행한다. 기존 sim은 gh/claude 대역을 쓰므로 실제 모델의 원본 읽기 증거와 구분한다. 새 회귀 명령은 테스트를 만든 뒤 과제서/원장에 정확히 추가해야 하며 현재 존재한다고 쓰지 않는다. 체크포인트: 기존 안전 차단과 새 회귀가 모두 통과해야 한다.
3. 확보된 최초 릴리즈 계약과 같은 입력으로 로컬 릴리즈 검증을 재실행하고 변경 SHA·명령·exit code·결과를 원장에 남긴다. 정책이 여전히 없으면 전체 복구 미완료로 남긴다. 현재 계약별 빌드 명령은 미확인; V2가 대상일 때만 `cd upgrade/admin-v2 && npm ci && npm run build`를 적용한다. 체크포인트: 자산/버전 게이트 성공 확인; 게시·태그 푸시는 이 구현 과제 범위 밖.

대안 비교 및 추정 근거
- 최소 수정: 외부 전달 경계 복구가 실제 스킬 누락 원인을 겨냥한다. 단, 현재 쓰기 표면 밖이며 정책 차단은 별도다.
- 확대안: 엔진 공통 스킬 로더/등록 체계 재설계는 작업량 L이며 45분 범위에서 제외한다.
- 새 장치 없는 안: 앱에 스킬 사본·릴리즈 관례를 작성하는 접근은 정본 중복·근거 없는 정책 신설이므로 제외한다.
- 무변경: 현재 권한 경계를 지키는 사실상의 상태이나 이미 반복된 실패이며 개선 성과로 인정하지 않는다.
- 바텀업 조건부 추정: 재현 8–12분 + 전달 수정 8–12분 + 회귀 9–11분 + 기록 3–5분 = 28–40분, 알려진 변동 예비 5분으로 33–45분. 신뢰도 낮음(실측 기반 확률 아님). 정책 결정·소유 범위 확보·실제 모델 대기 시간은 제외하며 이들이 미확보인 현재 전체 완료시간은 산정 불가다. 관리 예비는 별도 운영 소유이며 임의 배정하지 않는다. 유사 사례 5회가 no-change/실패여서 유사 추정은 현재 앱 회차 완료 가능성을 지지하지 않는다. 첫 실제 재현 후 다시 산정한다.

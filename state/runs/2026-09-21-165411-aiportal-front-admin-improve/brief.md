- 과제: 수정 과제 — 릴리즈 러너의 엔진별 headcount 스킬 전달 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 외부 aidev/bin/run.sh의 run_agent는 Claude용 Skill 안내를 run_codex에도 넘기지만 Codex가 읽을 원본 SKILL.md 경로를 전달하지 않는다. 엔진별 전달을 복구하면 스킬 탐색 실패를 제거할 수 있으나, 별도 차단인 최초 릴리즈 관례 부재까지 해결됐다고 볼 수는 없다.
- 수용 기준: 1) 실제 run_agent → 사용량 한도 폴백 → run_codex 자식에서 registry의 release 스킬 두 원본과 상대 참조를 읽을 수 있고, 직접 run_codex 경로도 같은 계약을 지킨다. 2) Claude 플러그인 전달과 headcount 비활성화 계약은 유지되며 원본 누락은 명시적 실패로 남고 임의 skip/성공 처리가 없다. 3) 동일한 프로덕션 배선 회귀가 수정 전 실패·수정 후 통과하고 기존 Release 게이트가 유지된다. 최초 버전·태그·노트·자산 계약이 없는 이 앱의 전체 릴리즈는 여전히 미완료로 판정한다.
- 건드릴 파일: /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — registry와 HEADCOUNT_DIR를 공통 근거로 엔진별 안내를 만들고 폴백·직접 호출 모두 원본 접근 보장; /mnt/c/Users/USER/projects/aidev/tests/test_sim.py:run 및 ReleaseSafety — 실제 러너 경로의 전달 회귀 추가; /mnt/c/Users/USER/projects/aidev/tests/sim/run_sim.sh — 격리 실행의 headcount 입력과 자식 증거 연결; /mnt/c/Users/USER/projects/aidev/tests/test_gate.py:Release — 기존 차단 계약 검증용이며 완화하지 않음. 모두 실제 읽은 파일이다. 신규 테스트 이름·자식 검증 파일은 구현자가 정하며 아직 존재하지 않는다.
- 검증 명령: 아래 실행 기록과 단계별 명령 참조. 앱 테스트로 외부 전달 복구를 입증하지 않는다.
- 위험과 피할 것: 앱 .gitlab-ci.yml, auth/, router, package 버전, gate.py, release-prompt.md의 skipped 조건, 운영 registry·전역 설정·자격증명은 변경하지 않는다. 스킬 복사본을 앱에 넣거나 CLAUDE.md/AGENTS.md로 외부 환경을 우회하지 않는다. 재이관 문서·실패 삭제·정책 끄기·다른 앱 버그 수정은 이번 과제의 대체 성과가 아니다.
- 차선 후보: 없음 — 자동 배정 우선 과제이므로 다른 앱 개선으로 바꾸지 않는다. 외부 소유 회차가 준비되지 않았으면 미완료(blocked)를 정확히 기록한다. 이 기록 자체는 수정 성과가 아니다.

## 착수 판정과 범위
현재 run.json의 project는 aiportal-front-admin이고 작업 트리는 /home/hkjang/.cache/auto-improve-wt/aiportal-front-admin이다. aidev/agents/registry.json의 builder.surface는 “그 회차의 worktree”, COMPANY.md 규칙 1은 “구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다.”이다. 따라서 이번 앱 구현 회차에 허용된 원인 수정 파일은 없다. 이 과제서는 외부 파일 편집 권한을 부여하지 않는다. 반복된 중지·이관 접근을 다시 수행하지 말고 기존 pending 과제를 유지한다. 실제 복구 계획의 실행 전제는 aidev 소유 작업 환경이며 현재 충족되지 않았다.
사용자의 정찰 절대 규칙에 따라 이번 세션은 코드·커밋을 만들지 않았고, 회차 디렉터리 안에만 기록했다. 전체 실패 복구와 수정 후 통과를 달성했다고 쓰면 안 된다.

## 근거와 실패 구분
- run.sh:241–335: agent_plugin_args는 Claude의 --plugin-dir를 구성하고 dept_note는 Skill 도구만 지시한다. run_agent의 폴백은 이미 붙인 안내를 그대로 사용하며 run_codex는 HEADCOUNT_DIR/원본 목록을 자식에 제공하지 않는다. 이는 정적 확인이고 실제 모델 자식 회귀는 미실행이다.
- 요청된 정찰 스킬 세 개와 릴리즈 스킬 두 개는 /mnt/c/Users/USER/projects/headcount/plugins/<department>/skills/<name>/SKILL.md에 존재하며 직접 읽었다. 현재 callable 도구 목록에 Skill/skills.read는 없다.
- fixer.sh:53–61은 status=failed 파일을 적재한다. run.sh:1688–1690은 일반 실패에도 “워크플로가 두 번 실패”를 붙인다. 이번 실패가 실제 GitHub CI 두 번 실패라는 근거는 미확인이다.
- 앱 .github/workflows는 없고 .gitlab-ci.yml은 브랜치 기반 레거시 build/copy다. 저장된 aiportal-front-admin.release.json은 에이전트의 스킬 탐색·관례 부재 실패이며 앱 빌드 실패 로그가 아니다.
- 로컬 태그 목록은 비었고 root package/lock은 0.0.0, V2 package/lock은 0.1.0이다. release-prompt.md 절차 5는 버전 파일도 없어야 skipped를 허용한다. 임의 버전 증가나 skipped 확대는 금지한다. 원격 릴리즈의 실제 존재 여부·과거 조회 실패 여부는 이번 세션에서 미확인이다.

## 대안 검토
1. 최소 수정: 기존 registry/HEADCOUNT_DIR를 이용한 엔진별 원본 경로 안내. 별도 설치·스킬 복제 없이 양쪽 호출 경로를 검증할 수 있어 추천한다. 경로를 주면 실제 자식이 읽을 수 있다는 가정은 런타임 회귀로 확인해야 한다.
2. 플러그인 설치·전체 스킬 자동 배포 체계: 범위가 커지고 전역 상태 관리가 생겨 45분 과제로 부적합하다.
3. 앱 문서에 스킬 복제: 소유권 경계를 우회하고 정본이 둘이 되므로 기각한다.
4. 변경 없이 중지/이관 반복: 안전한 미완료 판정은 가능하지만 이미 반복 실패했으므로 구현 대안·성과로 선정하지 않는다.

## 실행 계획 — 외부 소유 환경에서만
각 단계는 pending이며 사람 확인을 새로 요구하지 않는다. 경계 밖 실행은 이 계획으로 승인되지 않는다.
1. 재현: tests/test_sim.py의 run을 이용해 실제 bin/run.sh를 구동하고 Claude 사용량 제한 시 Codex 자식까지 도달하는 시나리오를 만든다. 경로에 공백이 있는 HEADCOUNT_DIR, release 단계의 두 원본, 상대 참조를 포함한다. 소스 문자열 검사·run_agent 재구현을 증거로 사용하지 않는다. 검사점: 새 테스트가 수정 전 전달 누락 때문에 실패하는지 확인한다.
2. 수정: 엔진 선택 전에 공유 입력을 확정하고 Claude에는 기존 플러그인, Codex에는 읽을 수 있는 절대 원본 경로와 상대 참조 기준을 제공한다. 폴백 경로와 직접 호출(run.sh의 review 호출 포함)이 동일하게 해석해야 한다. 검사점: bash -n bin/run.sh 및 추가한 실제 배선 테스트 통과.
3. 회귀: headcount 켜짐/꺼짐, 원본 누락, 두 스킬과 참조 읽기, Claude 정상 경로를 검증한다. 경계용 CLI 대역으로 인수/프로세스 전달을 검증했다면 실제 Codex 실행과 구분해 보고하며 그것만으로 모델 원본 읽기 성공을 주장하지 않는다. 검사점: python3 -B tests/test_gate.py Release -v 및 python3 -B tests/test_sim.py ReleaseSafety -v. 기존 sim은 /tmp 고정 생성이므로 이번 정찰 쓰기 제한에서는 실행하지 않았다.
4. 완료 판단: 원장에 '수정 과제'로 수정 전후 명령·exit·자식 원본 읽기 증거를 적는다. 버전 계약 미확보와 전체 릴리즈 미완료는 별도 유지한다. 이번 앱의 전체 릴리즈 재검증은 대상 패키지·버전·태그·노트·자산의 명시적 운영 근거가 확보된 뒤에만 가능하며, 45분 전달 수정 범위에 포함하지 않는다.

## 이번 정찰에서 실제 실행한 검증
작업 디렉터리는 앱 루트이며 로그는 같은 회차 verification.txt에 있다.
- bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh /mnt/c/Users/USER/projects/aidev/bin/fixer.sh → exit 0.
- TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-165411-aiportal-front-admin-improve/assets PYTHONDONTWRITEBYTECODE=1 python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v → 5개 통과, exit 0. 기존 unclosed file ResourceWarning 발생. 임시 파일은 지정 경로에만 생성 후 정리했다.
- python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json → exit 1, ok:false/state:failed. 저장된 실패에 대한 차단 재현이며 원인 프로세스 재현이나 복구 성공이 아니다.
- node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json → exit 0. 앱 runtime 설정의 독립 검사일 뿐 릴리즈 복구 증거가 아니다.
- 실제 폴백/모델 실행·수정 전후 회귀·전체 sim·npm 설치·앱 build/test·서버/UAT는 미실행. node_modules 없음.

## 작업량 근거
Bottom-up 추정: 재현 10–15분 + 전달 수정 10–15분 + 회귀/원장 10분 = 기본 30–40분, 알려진 경로/환경 차이 예비 5분 별도 = 35–45분. 외부 소유 환경과 기존 sim 확장이 가능한 경우에만 M이다. 실측 유사 작업 완료 데이터가 없어 확률 신뢰도는 산정하지 않았고 추정 확신은 낮음이다. 모델 실실행 지연·소유 환경 준비·최초 릴리즈 운영 결정 대기는 제외하며 관리 예비는 배정하지 않았다. 재현 단계에서 범위가 늘면 수치를 갱신하고 전체 실패 복구를 45분으로 약속하지 않는다.

## 적용한 스킬
Skill 호출 도구 부재로 로컬 원본을 직접 읽어 적용했다: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md. PMO references/sources.md도 읽었으며 위 숫자는 외부 통계가 아닌 이번 파일 범위를 기반으로 한 추정이다.

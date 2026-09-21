- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건의 재개 조건 확정 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 releaser가 다음 버전·커밋·태그 관례를 결정할 근거를 받지 못해 실패했으며 현재 Git/JSON 재조회에도 새 근거가 없다. 적용 가능한 출처가 확보될 때만 입력을 보완해야 같은 무변경 구현을 반복하거나 검증 조건을 낮추는 일을 피할 수 있다.
- 수용 기준: 1) aiportal-front와 기준 SHA에 적용되는 승인된 증가 정책 또는 검증 가능한 실제 릴리즈 출처에 다음 증가 단위, 커밋·태그 형식, 노트·자산·검증 절차가 연결된다. 2) 해당 근거가 확인된 뒤에만 docs/RELEASE.md에 출처와 적용 범위를 보완하고 구현 대상 및 동일 실패 단계의 재현 절차를 확정한다. 3) 기존 실패 JSON은 계속 거부되어야 하며 실제 releaser의 새 결과와 동일 게이트 검증이 모두 통과해야 수정 완료다; 회귀 5개 통과·문서 검사·합성 released JSON·앱 빌드만으로 대체하지 않는다.
- 건드릴 파일: 현재 배정 가능한 저장소 수정 파일 없음. 조건부 docs/RELEASE.md: 다음 버전 판단의 근거와 미확인 항목 — 실제 출처 확보 후 필요한 사실만 반영. package.json:version 및 package-lock.json:version/packages[""].version은 읽기 전용 근거이며 임의 증가 금지. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 읽은 진단 경로이며 이번 저장소 수정 대상으로 배정하지 않는다.
- 검증 명령: 아래 재현 명령 참조. 실제 releaser는 run.sh의 release_project가 실행하는 모델 단계이며 독립 로컬 전용 명령은 미확인이다. 전체 run.sh를 source/실행하면 배포까지 갈 수 있으므로 재현용으로 실행하지 않는다.
- 위험과 피할 것: 새 버전·태그 관례 작성, 0.0.0으로 되돌리기, 실패 JSON 수정, gate/CI 완화, version 파일 삭제로 skipped 유도, 외부 러너 수정, auth/session/router 변경, 원격 전송을 금지한다. 원격 인증 실패나 빈 context는 원격 이력 부재 증거가 아니다. Skill 접근은 이미 최신 실패에서 성공했으므로 접근 안내를 또 수정하지 않는다.
- 차선 후보: 없음 — 우선 과제가 고정되었으므로 일반 개선으로 대체하지 않는다. 신규 출처가 없으면 기존 pending을 인계하고 동일 BLOCKED 구현을 신규 배정하지 않는다.

진입 판정: BLOCKED / 수정 미완료. 이번 회차에 새 승인 정책·실제 릴리즈 출처가 전달되거나 발견되지 않았다. 구현자는 이 문서만 받은 상태라면 반복 조사·gate·npm 검사를 실행하거나 무변경을 구현 성과로 보고하지 말고, 진입 조건 미충족으로 기존 pending을 유지한다. 이는 과거 no-change 접근을 다시 과제로 실행하라는 지시가 아니다.

확인한 근거:
- HEAD e938e8e, non-shallow 37커밋, git log -30 및 package 이력 확인, 로컬 태그 0개. package/lock 3필드 0.0.0, 최초 package 0.0.0·최초 lock 두 필드 부재. CLAUDE.md와 .github/workflows는 없음.
- .gitlab-ci.yml은 main/develop 전용 Runner의 npm 빌드·복사 배포다. 버전 결정 실패의 실행 스크립트로 볼 근거가 없다.
- 실패 원본: /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 2026-09-21-001359-aiportal-front-improve/release.json. 앞 실패는 스킬 접근과 버전 근거 문제, 뒤 실패는 스킬 원문을 읽은 뒤 버전 근거 문제다. 앞 기록의 최초 lock 설명은 실제 Git과 다르다.
- release_project는 run_agent 결과를 cmd_release/evaluate_release에 넘겨 ok=false에서 push 전에 종료한다. retry_release_workflow의 GitHub run 재시도는 별도 경로다. 실패 run ID·step 로그 미확인으로 같은 GitHub step 2회 실패라고 확정하지 않는다.
- release_context는 gh 오류를 '(없음)'으로 출력하고 release_project는 fetch 오류 뒤 계속한다. 코드상 진단 문제지만 이번 실패와 인과 미확인, 외부 소유이므로 이번 수정으로 배정하지 않는다.

계획과 확인점(각 단계 순차; 별도 사람 승인 요청 없음):
1. [미충족] 신규 출처의 프로젝트·SHA·접근 성공 여부와 정책 내용을 대조한다. 증명: 원문/실제 태그 커밋·주석과 JSON 이력. 출처 없으면 이후 단계에 진입하지 않는다.
2. [미착수] 출처가 지시하는 최소 입력 보완을 정하고 파일·검증 계획을 갱신한다. 증명: 출처 대조와 git diff --check. 범위가 바뀌면 이 계획부터 수정한다.
3. [미착수] 같은 모델 판단과 동일 게이트를 재검증한다. 증명: 새 실행 로그·결과 JSON·종료 코드. 실제 실행 경로가 확정되지 않은 현재 성공을 약속할 수 없다.

검증 명령(저장소 루트에서 실행 가능; 외부 Python 검사는 -B와 회차 내부 TMPDIR 사용):
```bash
git rev-parse --is-shallow-repository
git tag --sort=-creatordate
git log --oneline -60
git log --oneline -- package.json package-lock.json
git diff --check
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-012421-aiportal-front-improve PYTHONDONTWRITEBYTECODE=1 python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
```
이번 실행: 임시 디렉터리를 회차 폴더 아래 생성하고 종료 후 정리하는 Python 래퍼로 Release 5개 통과(exit 0, unclosed file ResourceWarning). 두 실패 원본 각각 gate exit 1/state failed/ok false. 이는 수정 전 실패 거부 재현이지 수정 후 통과가 아니다. 앱 npm ci/test/build, 실제 releaser, 전용 Runner, 원격 조회는 미실행. package에 npm test 및 npm run build:dev는 존재하지만 현재 node_modules 없음; 이 명령은 버전 관례를 증명하지 않는다.

대안 비교와 추정:
- 최소 입력 복구: 실제 출처가 이미 제공될 때만 적합. 재사용할 기존 Git/JSON이 있으면 새 실행 구성은 불필요하므로 이 경로를 권고한다.
- 정식 릴리즈 자동화/정책 신설: 장기 확장은 가능하지만 현재 권한·근거·45분 범위를 벗어나므로 제외.
- 외부 context 오류 구분: 진단 품질 개선 가치가 있으나 소유 범위 밖이고 버전 관례 자체를 제공하지 못한다.
- 무변경 유지: 현재 적용하는 상태 처리; 해결책/완료 실적으로 계산하지 않는다.
- Bottom-up 조건부 작업: 출처 대조 5–10분, 문서 입력 보완 5–10분, 검증·기록 10–15분 = 20–35분. 알려진 경로 차이 contingency 0–10분 별도, 합계 20–45분. 관리 예비는 배정하지 않는다. 출처 획득 대기·외부 배포·자동화 신설은 제외. 신뢰도 낮은 계획 범위이며 확률 추정이 아니다. 과거 회차는 모두 no-change여서 성공 작업의 유사 추정으로 교차 검증할 수 없고 입력 확보 시 재추정한다.

스킬 적용: 호출 가능한 Skill 도구가 없어 실제 파일 존재 확인 후 아래 원문을 직접 읽었다(성공한 Skill 호출 아님). 별도 고정 반환 스키마 없음. 범위·대안·단계별 증명·확인점·분해 추정·예비를 이 문서에 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
- estimating-and-contingency/references/sources.md도 읽었다. 외부 GAO/HM Treasury 원문은 미열람; 본 시간 범위는 외부 권위의 수치·확률을 인용한 추정이 아니다.

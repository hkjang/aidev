- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 필수 근거 결손으로 실행 배정 불가 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 releaser는 다음 버전과 릴리즈 커밋·태그 관례를 결정할 출처가 없어 실패했고 이번 재조회에도 새 출처가 없다. 적용 가능한 실제 근거를 입력으로 연결해야 실패를 해결할 수 있으며, 같은 무변경 구현을 반복하거나 게이트를 완화하는 것은 해결이 아니다.
- 수용 기준: 1) aiportal-front와 기준 SHA에 적용되는 승인된 정책 또는 실제 릴리즈 이력에서 증가 단위·커밋/태그 형식·노트·자산·검증 절차를 확인한다. 2) 확인된 출처에 한해 docs/RELEASE.md의 판단 입력을 보완하고 실제 실패 단계에 전달되는 것을 입증한다. 3) 기존 failed JSON은 계속 거부되고, 보완된 입력을 사용한 실제 releaser의 새 결과와 동일 gate 검증이 통과해야 완료다; 합성 released JSON이나 앱 빌드·회귀 테스트 통과로 대신하지 않는다.
- 건드릴 파일: 현재 실행 가능한 수정 파일 없음. 조건부 docs/RELEASE.md:「미확인 항목과 한계」및 판단 근거 — 새 출처를 확보한 경우에만 사실과 적용 범위 반영. package.json:version, package-lock.json:version/packages[""].version은 읽기 전용 근거. 직접 읽은 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 진단 경로이며 이번 저장소의 수정 대상이 아니다.
- 검증 명령: 아래 실제 실행 명령 및 결과 참조. 실제 모델 판단의 독립 로컬 전용 실행 명령은 미확인이다. run.sh 전체 실행/source는 원격 전송 경로가 있어 재현 명령으로 사용하지 않는다.
- 위험과 피할 것: 관례 임의 신설, 버전 증가, 버전 파일 삭제로 skipped 유도, 게이트·CI 완화, 실패 JSON 조작, 외부 러너 수정, 인증·세션·라우터 변경을 금지한다. 과거 스킬 안내 수정은 최신 실패에서 이미 원문 접근에 성공했으므로 재선정하지 않는다. 빈 GitHub context나 인증 실패를 원격 이력 부재로 단정하지 않는다.
- 차선 후보: 없음 — 우선 과제가 고정되어 일반 개선으로 교체할 수 없다. 신규 출처가 없으면 기존 pending만 유지하며 동일 BLOCKED 구현을 신규 실행 과제로 배정하지 않는다.

진입 판정: **BLOCKED / 수정 미완료**. 이 brief는 기존 미해결 건의 입력 계약이며 반복 조사·무변경 구현을 다시 수행하라는 과제가 아니다. 구현자는 새 출처가 인계되지 않으면 수용 기준 1 미충족으로 기록하고 2·3에 착수하지 않는다. 이번 회차에는 코드를 수정하지 않았으므로 “고친 뒤 동일 검증 통과”는 달성하지 못했다.

확인한 실행 경로와 근거:
- HEAD e938e8e, non-shallow, 총 37커밋, 로컬 태그 0개. git log -30 및 package/lock 이력 직접 조회. 현재 버전 세 필드 0.0.0, 최초 package 0.0.0·최초 lock 두 필드 부재. CLAUDE.md와 .github/workflows 없음.
- 실패 원본은 /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 2026-09-21-001359-aiportal-front-improve/release.json. 앞 실패의 최초 lock 설명은 Git과 불일치한다. 뒤 실패는 스킬 원문 접근 뒤 버전 판단에서 중단됐다.
- release_project → run_agent → release.json → cmd_release/evaluate_release. status != released면 ok=false로 push 전에 반환한다. 기존 실패를 거부하는 동작은 정상이며 여기서 통과시키도록 수정할 근거가 없다.
- retry_release_workflow의 GitHub run 재시도는 별도 경로다. run ID·실패 step 로그 미확인으로 이 두 JSON을 같은 GitHub Actions 단계의 2회 실패라고 확정하지 않는다.
- .gitlab-ci.yml은 main/develop의 전용 Runner에서 npm 환경별 빌드 후 서버 복사. 현재 실패의 버전 결정 단계와 별개이며 실제 배포 성공은 미확인이다.
- release_context의 gh 오류→'(없음)' 처리와 release_project의 fetch 오류 후 계속 진행은 직접 읽었다. 진단 결함 후보지만 이번 실패와 인과는 미확인이고 외부 저장소 소유이므로 이번 해법으로 배정하지 않는다.

조건부 순차 계획(사람에게 질문하지 않음):
1. [미충족] 신규 출처의 적용 프로젝트·SHA·접근 성공과 정책 내용을 확인한다. 증명은 실제 원문 또는 태그/커밋/JSON 이력이며 없으면 중단한다.
2. [미착수] 실제 출처가 있을 때만 docs/RELEASE.md의 입력 근거를 최소 보완하고 실행 경로 및 파일별 계획을 갱신한다. 증명: 출처 대조, git diff --check. 범위가 달라지면 먼저 계획을 수정한다.
3. [미착수] 실제 releaser에 입력 전달 후 새 로그·결과 JSON과 동일 gate의 종료 코드로 검증한다. 기존 실패 JSON 거부도 유지한다. 단독 게이트는 버전 관례의 진실성까지 검증하지 않으므로 모델 판단 증거가 필수다.

이번 실행과 재현 명령(저장소 루트):
```bash
git rev-parse --is-shallow-repository
git tag --sort=-creatordate
git log -30 --oneline
git log --oneline -- package.json package-lock.json
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-023420-aiportal-front-improve PYTHONDONTWRITEBYTECODE=1 python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
git diff --check
```
- Release 5개 테스트 exit 0. 테스트의 unclosed file ResourceWarning 존재. 이번에는 회차 내부 TemporaryDirectory 래퍼에서 실행하고 임시 파일을 정리했다.
- 위 gate 명령과 09-20-211415 원본에 대한 같은 명령은 각각 exit 1 / state=failed / ok=false. 수정 전 실패 거부를 확인한 것이며 수정 후 통과가 아니다.
- npm test, npm run build:dev는 package.json에 존재하지만 node_modules 없음. 앱 설치/테스트/빌드, 실제 releaser, 원격 조회, 전용 Runner 검사는 미실행이다. 실패 원인과 무관한 설치/빌드로 성공을 주장하지 않는다.

대안 비교:
- 기존 출처를 사용한 최소 입력 복구: 출처 확보 시 가장 작은 변경이며 유일한 조건부 권고. 핵심 가정은 적용 가능한 실제 출처를 확보할 수 있다는 것.
- 새 릴리즈 정책/자동화 도입: 향후 확장성은 있지만 관례 신설 금지와 45분 범위를 위반하여 제외.
- 외부 context 오류 구분: 별도 진단 개선으로 유효할 수 있으나 정책 내용을 복구하지 못하고 소유 범위 밖이다.
- 무변경 유지: 현재 상태 처리이며 해결책이나 구현 성과로 계산하지 않는다.

추정 근거:
- Bottom-up: 출처 대조 5–10분 + 입력 문서 보완 5–10분 + 검증/기록 10–15분 = 20–35분. 알려진 실행 경로 차이 contingency 0–10분을 별도로 두어 조건부 합계 20–45분. 관리 예비는 미배정.
- 출처 획득 대기, 원격 배포, 새 자동화 개발은 제외. 신뢰도 낮은 계획 범위이며 통계적 성공 확률이 아니다. 과거 회차는 no-change뿐이라 유사 성공 사례로 두 번째 추정을 검증할 수 없고, 출처 확보 및 실제 재현 경로 확정 시 재추정해야 한다. S는 입력 보완 범위이며 전체 릴리즈 완료 시간 보장이 아니다.

스킬 적용 기록:
호출 가능한 Skill 도구가 없어 후보 경로의 존재·읽기 가능 여부를 확인한 뒤 아래 원문을 직접 읽었다. 성공한 Skill 호출이 아니며 별도 고정 반환 스키마는 없었다. 대안·범위·순차 계획·증명·확인점·분해 추정과 예비 구분을 이 과제서에 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
- estimating-and-contingency/references/sources.md도 읽었다. GAO/HM Treasury 외부 원문은 미열람이며 외부 권위의 수치나 확률을 인용하지 않았다.

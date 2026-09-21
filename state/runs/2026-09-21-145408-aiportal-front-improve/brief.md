- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건, 실행 가능한 구현 배정 없음 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 release-prompt.md의 다음 버전·커밋·태그 결정을 뒷받침할 정책 또는 실제 관례가 없어 릴리즈 에이전트가 failed를 반환한다. 근거 있는 입력을 복구해야 같은 실패를 해결할 수 있으며, 현재 입력으로 관례를 창작하거나 게이트를 완화하면 요구를 위반한다.
- 수용 기준: 1) 다음 버전·커밋·태그·노트·자산 방식과 검증 절차를 결정할 승인 정책 또는 실제 릴리즈 출처가 확보되고 현재 Git/JSON과 대조된다. 2) 그 출처가 제공된 경우에만 docs/RELEASE.md에 출처와 확인 범위를 기록하며 기존 명령으로 앱 검사를 수행한다. 3) 실제 releaser가 해당 입력으로 판단·수행한 결과를 원래 gate.py release로 검사하여 exit 0/ok=true를 얻고 커밋·태그·노트·자산이 출처와 일치함을 확인한다; fixture의 status 변경이나 gate 단위 테스트 성공은 이 기준을 충족하지 않는다.
- 건드릴 파일: 현재 즉시 수정할 저장소 파일 없음. 신규 출처가 생겼을 때만 docs/RELEASE.md: 확인한 근거/미확인 항목과 한계 — 실제 출처를 정본에 반영. package.json 및 package-lock.json의 루트 version/packages[""].version은 조회 대상이며 이번 구현에서 임의 증가 금지. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md: 절차 2·3·5, bin/run.sh: release_context/release_project/retry_release_workflow, bin/gate.py: evaluate_release/cmd_release, tests/test_gate.py: Release는 읽기 전용 원인 추적 대상이다.
- 검증 명령: 아래 명령과 검증 범위를 따른다. 현재 새 입력이 없어 수정 후 동일 검증 통과 명령을 제시할 근거는 미확인이다. 원격 전송을 포함하는 run.sh 전체 실행은 금지한다.
- 위험과 피할 것: 보호 경로 src/api 인증·storage·router·.gitlab-ci.yml 및 외부 실행기/게이트를 바꾸지 않는다. 0.0.0→0.0.1, v 접두사, release 커밋 양식을 새로 정하거나 버전 파일을 지워 skipped 조건을 만들지 않는다. 실패 JSON을 released로 조작하거나 문서 보강·앱 테스트·스텁 sim을 릴리즈 성공으로 세지 않는다. 이전 6회처럼 같은 조사만 반복하는 BLOCKED 구현을 다시 수행하지 않는다.
- 차선 후보: 없음 — 고정 수정 과제이므로 다른 앱 개선으로 대체하지 않는다. 신규 출처가 없으면 기존 pending을 유지하고 구현·릴리즈 미완료를 정확히 기록한다.

현재 판정 및 원인 근거

- 정찰 전용 역할: 코드 수정·커밋을 하지 않는다. 수용 기준 1 미충족, 2·3 미착수. 이번에 해결됐다는 주장은 불가하다.
- HEAD e938e8e, non-shallow 37커밋, 로컬 태그 0개. git log -30 및 패키지 이력 조회. 현재 세 버전 필드는 0.0.0, 최초 package는 0.0.0/최초 lock 두 필드 부재. 저장소 .github/workflows 및 전용 릴리즈 스크립트 발견 못 함. GitLab CI는 브랜치별 Runner 빌드·복사 배포이다.
- 직접 읽은 실패: state/runs/2026-09-20-211415-aiportal-front-improve/release.json과 2026-09-21-001359-aiportal-front-improve/release.json. 전자는 스킬 접근 문제를 함께 기록하고, 후자는 원문 접근 후에도 정책 근거 부족으로 실패한다. 두 실패 JSON은 존재하지만 GitHub Actions workflow의 독립 실패 2회/run ID/실패 step은 미확인이다.
- 실제 경로: release_project → release_context → run_agent → release.json → gate.py cmd_release/evaluate_release. failed는 게이트에서 차단되고 push/CI/tag 단계 전에 반환한다. 따라서 현재 자료로 앱 빌드 실패 원인을 지정할 수 없다.
- release_context는 gh 조회 실패를 '(없음)'으로 출력하고 release_project는 tag fetch 실패 뒤에도 계속한다. 이미 보류된 외부 진단 후보이며 이번 실패의 직접 원인 입증과 외부 수정 권한은 없다. 빈 문맥만으로 원격 이력이 없다고 단정하지 않는다. 이번 원격 조회는 하지 않았다.

구현 인계 순서와 체크포인트

1. [미충족] 인계에 새 정책/실제 릴리즈 출처가 포함됐는지 확인한다. 변경 표면은 아직 없고 증거는 그 출처 원문 및 아래 Git/JSON 조회다. 없다면 반복 조사·gate·앱 검사로 시간을 쓰지 말고 기존 미해결 상태를 기록한다. 사람에게 질문하는 체크포인트는 두지 않는다; 입력 충족 여부가 재개 조건이다.
2. [미착수] 새 출처가 있을 때만 정본의 근거를 갱신한다. 정본을 정책 창작 문서로 바꾸지 말고 충돌 시 계획부터 고친다. 증거는 git diff --check 및 실제 출처와 diff 대조. 버전·태그 생성은 구현 단계가 아닌 기존 릴리즈 단계에 맡긴다.
3. [미착수] 재개 후 npm ci, npm test, npm run build:dev를 실행한다. 앱 검증 뒤 별도 releaser의 실제 결과를 원래 gate로 검사해야 한다. 현재 읽은 코드에 모델 버전 판단만 안전하게 재실행하는 전용 로컬 명령은 발견하지 못했다. 이 연결을 확보하기 전 45분 내 전체 수정 완료를 약속하지 않는다.

실제 실행한 검증(저장소 루트 기준)

```bash
git rev-parse --is-shallow-repository
git tag -l
git log -30 --oneline
git log --format='%h %s' -- package.json package-lock.json
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-145408-aiportal-front-improve python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
git diff --check
git status --short
```

Release 단위 테스트 5개 OK(자산 파일 미닫힘 ResourceWarning 존재). 실제 기존 실패 JSON 게이트 재검사 exit 1/failed/ok=false. 이 검사는 기존 실패 결과의 거부를 재현했을 뿐 모델 판단 재실행·수정 전후 인과 검증이 아니다. node_modules 없어 npm ci/test/build는 실행하지 않았다. 게이트 입력 원본·실행기 무변경.

해법 비교 및 추정 근거

- 기존 실제 관례/승인 정책 입력 복구: 선택한 유일한 재개 경로. 기존 규칙을 유지하지만 입력 확보 시간은 미확인이다.
- 외부 조회 오류 구분 개선: 진단 품질에는 도움이 될 수 있으나 외부 소유이며 이 저장소의 버전 정책 결손을 해결한다고 입증되지 않아 미선정.
- 새 정책·워크플로 신설: 현재 사용자의 관례 창작 금지와 충돌해 제외. 현상 유지: 입력 없는 현재에는 정직한 상태 처리지만 수정 완료로 세지 않는다.
- 핵심 가정은 권위 있는 신규 출처가 확보될 수 있다는 점이다. 이 가정이 아직 성립하지 않는다.
- Bottom-up 조건부 작업 추정: 출처 대조 5–10분, 정본 반영 5–10분, 검사·기록 10–15분 = 20–35분. 알려진 검사 변동 예비 5–10분을 별도 반영하면 25–45분. 낮은 확신의 계획 범위이며 통계적 신뢰구간이나 완료 약속이 아니다. 외부 출처 확보·배포 대기·정책 결정·모델 재실행 연결 확보는 제외되어 전체 완료시간은 미확인. 관리 예비는 배정하지 않는다. 유사법은 기존 반복 no-change로 성공 비교 자료가 없어 수치 산정 불가; 출처 확보 시 재추정한다.

스킬 적용 기록

호출 가능한 Skill 도구 없음. 아래 실제 원문을 읽었으며 성공한 Skill 호출로 가장하지 않는다. 세 스킬에 별도 고정 반환 스키마는 없으며 범위·증거·체크포인트, 대안 비교·가정, 분해 추정·예비 분리를 이 과제서에 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
- estimating-and-contingency/references/sources.md도 읽음. 외부 GAO/Green Book 본문은 미조회이며 이를 근거로 확률·비용·편익을 주장하지 않는다.

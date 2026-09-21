- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 왜: 실제 실패는 외부 releaser가 다음 버전·커밋·태그 관례를 확정하지 못해 release.json에 failed를 남긴 것이며, 현재 앱에서 수정할 실패 스크립트는 확인되지 않았다. 기존 관례를 뒷받침할 출처를 복구해야 같은 릴리즈 절차를 진행할 수 있으나 이번에도 신규 출처가 없어 반복 무변경 조사를 구현 성과로 배정하지 않는다.
- 수용 기준: 1) 승인된 정책 또는 검증 가능한 실제 릴리즈 출처가 다음 버전·커밋 메시지·태그 형식/주석 여부·노트·자산·검증 명령을 결정할 수 있고 출처 위치가 기록된다. 2) 그 근거에 따른 최소 변경만 구현되며 package.json 및 package-lock.json의 두 버전 필드가 일치하고 기존 gate/워크플로 조건이 보존된다. 3) 실제 releaser의 버전 결정 단계가 근거를 사용하여 진행되고 동일 검증이 통과해야 한다; 실패 JSON의 status 치환, 합성 released JSON, gate 단위 테스트 통과만으로 완료 판정하지 않는다.
- 건드릴 파일: 현재 수정 승인 대상 없음. 재개 근거가 들어온 경우에만 docs/RELEASE.md: 릴리즈 판단 근거에 출처 추가 여부 판단, package.json:version 및 package-lock.json:version/packages[""].version — 근거가 정한 값 동시 반영. 외부 /mnt/c/Users/USER/projects/aidev/bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, release-prompt.md:절차 2·3·5, tests/test_gate.py:Release는 직접 읽은 원인 분석 자료이며 이 앱 회차 수정 대상이 아니다.
- 검증 명령: 저장소 루트에서 `git diff --check`, `git status --short`, `git rev-parse --is-shallow-repository`, `git tag --list`, `git log -30 --oneline`. 아래 실행 결과와 재개 조건을 구분한다.
- 위험과 피할 것: 버전 0.0.1을 추정하거나 태그·릴리즈 정책을 신설하지 않는다. .gitlab-ci.yml/auth/router/storage를 우회 수정하지 않는다. gate/원본 실패 JSON/외부 runner 수정, skipped 전환, 원격 전송 금지. 이전 스킬 경로 안내는 이미 적용됐고 최신 실패는 스킬 접근 뒤 발생하므로 같은 문서 PR을 반복하지 않는다.
- 차선 후보: 없음 — 우선 과제가 고정되어 무관 앱 개선으로 대체할 수 없다. useAppList 비배열 캐시 방어(가치 3/위험 2/S)는 향후 별도 회차 후보일 뿐 이번 구현자가 선택할 대안이 아니다.

현재 판정: 수용 기준 1 미충족, 2·3 미착수. 수정 및 동일 검증 통과 미완료. 코드 수정 가능한 배정이 성립하지 않으므로 구현자는 새 근거가 인계되지 않으면 추가 반복 조사·gate·npm 검증을 돌리지 말고 기존 pending을 유지한다. 이 문서는 새 BLOCKED 구현 과제가 아닌 고정 미해결 건의 인계다.

직접 확인한 근거:
- HEAD e938e8e, 전체 37커밋, non-shallow, 로컬 태그 없음. package 0.0.0, lock 루트/루트 패키지 0.0.0; 최초 ab700ba에서 lock 두 필드 부재. package/lock 이력에 증가 없음.
- docs/RELEASE.md는 증가 정책이 아니다. .github/workflows와 CLAUDE.md 없음. .gitlab-ci.yml은 전용 Runner의 브랜치 빌드·복사 배포이며 해당 실패 로그가 아니다.
- 2026-09-20-211415 및 2026-09-21-001359 회차 release.json 직접 읽음. 전자는 스킬 결손도 포함, 후자는 스킬 접근 성공 뒤 정책 결손. GitHub Actions 동일 step 실패 2회/run ID는 미확인이다.
- run.sh release_project는 run_agent 결과를 gate에 전달하고 ok=false이면 원격 push 이전 반환한다. release_context는 gh 오류를 빈 이력과 혼동할 수 있지만 이번 실패의 직접 원인이라는 증거가 없으며 이미 보류된 외부 후보이다. 비어 있는 release-context.md로 원격 이력 부재를 확정하지 않는다.

실행한 검증(저장소 루트에서 실행 가능):
```bash
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-171424-aiportal-front-improve/home PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release
PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
```
첫 명령: 5 tests OK, exit 0, ResourceWarning 있음. 두 번째: exit 1, state=failed, ok=false. 실제 CLI의 기존 실패 거부를 재현했으며 모델 버전 판단 재실행이나 수정 후 Green이 아니다. 테스트 임시 파일은 이번 회차 home 아래에만 생성했다. 앱 npm ci/test/build, 실제 releaser, sim, 원격 조회·배포는 미실행이다.

재개 계획과 점검 지점:
1. [대기] 새로운 출처가 인계되면 출처가 실제 대상 저장소의 정책/이력인지 확인하고 결정 표를 먼저 작성한다. 증명: 출처 원문 및 해당 Git/JSON 값 재조회. 사람 질문은 하지 않으며 출처가 없으면 여기서 종료한다.
2. [미착수] 출처가 요구하는 변경 파일과 정확한 명령으로 이 과제서를 갱신한 뒤 최소 변경한다. 증명: 버전 세 필드 일치 및 diff 확인. 실제 검사 명령이 미확인이므로 임의 release-check 스크립트를 만들지 않는다.
3. [미착수] 실제 릴리즈 경로를 로컬에서 재현하고 기존 검증을 실행한다. 앱 변경 시 설치 후 `npm test`, `npm run build:dev`를 별도 검사하며 이 두 명령이 릴리즈 관례 검증을 대신하지 않는다. 결과·미실행·한계를 원장에 남긴다. 원격 실행은 이 정찰 범위 밖이다.

대안 비교와 추정 근거:
- 기존 정책/실제 이력 복구: 유일하게 현재 계약을 만족시키는 최소 접근; 출처가 존재하고 제공되어야 한다.
- 외부 입력 수집 경로 보강: 여러 프로젝트에 도움이 될 수 있으나 외부 소유이고 이번 실패와 인과 미확인; 재선정하지 않음.
- 새 관례 도입/게이트 완화: 사용자 규칙에 어긋나므로 기각.
- 현재 무변경 유지: 해결은 아니지만 근거 없는 릴리즈를 만들지 않는 현재 처리. 반복 조사 자체의 가치 증가를 주장하지 않는다.
- S는 출처 확인·과제 갱신에 한정한 bottom-up 추정 10~20분(출처 읽기 3~5, 결정 표/파일 범위 3~7, 검증 명령/기록 4~8). 알려진 불확실성 예비 0~5분을 별도 두며 경영 예비는 배정하지 않는다. 통계적 신뢰수준은 미산정, 판단 신뢰도 낮음. 실제 수정·릴리즈 시간은 정책 입력 전 미확인이고 45분 완료를 약속하지 않는다. 기존 no-change 회차는 성공 구현 유사 추정의 근거가 될 수 없다.

스킬 적용 기록:
호출 가능한 Skill 도구 없음. 아래 원문을 파일로 읽었으며 성공한 Skill 호출로 간주하지 않는다. 세 원문에 별도 고정 반환 스키마는 없다. solution-exploration의 대안·가정, implementation-planning의 단계·증명·점검 지점, estimating-and-contingency의 범위·가정·분해 추정·예비 분리를 위에 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
estimating의 references/sources.md도 읽음. 외부 권위 문서 본문은 미조회이며 비용·확률 신뢰도에 관한 외부 권위 주장은 사용하지 않는다.

- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 현재 배정 불가/BLOCKED (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 releaser는 기존 버전 증가·커밋·태그 관례를 요구하지만 이 저장소에서 그 입력을 확보하지 못해 버전 결정 단계에서 실패한다. 실제 출처로 입력을 채우면 동일 실패 반복을 해소할 수 있지만 현재는 앱 코드 결함이나 45분 안에 실행 가능한 수정안이 입증되지 않았다.
- 수용 기준: 1) 다음 버전·증가 단위·커밋 양식·태그 사용/형식/종류·노트·자산 방식 각각에 승인 원문 또는 실제 과거 릴리즈 출처가 연결된다. 2) 그 입력에 근거한 실제 releaser 결과와 필요한 빌드/검사 출력이 남고, 버전 파일 세 필드와 실제 커밋·태그가 정책에 일치한다. 3) 새 실제 결과를 변경하지 않은 gate.py release에 넣어 exit 0/ok=true를 확인하고, 기존 failed 기록은 계속 exit 1로 차단된다; 가짜 released JSON·sim 통과는 완료 증거가 아니다.
- 건드릴 파일: 현재 저장소 수정 파일 없음. 회차 brief.md/ideas.json/profile.md/journal.md/ledger-entry.md/scout-checks.json만 정찰 기록. docs/RELEASE.md는 읽기 전용 근거 정본; package.json:version 및 package-lock.json:version/packages[""].version 갱신은 실제 정책 확보 후 후속 릴리즈 역할의 범위. 외부 /mnt/c/Users/USER/projects/aidev/bin/run.sh:release_context/release_project, release-prompt.md:절차 2·3·5, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release, tests/test_sim.py:HappyPath/Agents는 읽기만 했다. 이번 앱 구현자에게 외부 수정을 배정하지 않는다.
- 검증 명령: 아래 실제 실행 명령과 결과 참고. 현재 통과 명령으로 장애 해결을 입증할 수 없음.
- 위험과 피할 것: 새 관례/임의 0.0.1 증가/태그 생성/failed→skipped 전환/게이트 완화/워크플로 추가 금지. auth·storage·router·.gitlab-ci.yml·외부 러너·headcount 원문 변경 금지. 기존 문서 정본화와 승인 대기 과제는 이미 merged/no-change로 반복됐으므로 다시 구현하지 않는다. 원격 조회 오류나 공란을 릴리즈 없음으로 단정하지 않는다. 원격 전송·실서버 배포 금지.
- 차선 후보: 없음 — 고정 우선 과제를 앱의 무관한 개선으로 바꾸지 않는다. useAppList 캐시 방어는 별도 회차 후보일 뿐이다.

범위와 인계 판정

현재 정찰은 읽기와 회차 기록만 허용한다. 우선 문장의 “고친 뒤 통과”를 이번 정찰에서 달성한 것으로 표시하지 않는다. 이전과 같은 BLOCKED 과제를 새 해결책으로 포장하지 않고, 자동 배정은 유지하되 실행 가능한 수정안 미확보로 명시한다. 새 근거 없는 다음 구현자는 파일 변경·반복 조사·같은 gate 재실행을 하지 않고 이 차단 상태만 인계한다. 질문이나 추가 승인 요청은 하지 않는다.

확인 근거 (2026-09-21, HEAD e938e8e)

- AGENTS.md와 docs/RELEASE.md를 먼저 읽음. CLAUDE.md, .github/workflows, node_modules 없음.
- non-shallow 37커밋, 로컬 태그 0개. git log -30 및 패키지 파일 이력 조회. 현재 버전 세 필드 0.0.0, 최초 lock 두 필드 ABSENT. 로컬 결과는 원격 태그 부재 증거가 아니다.
- 실패 원문 두 건: state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 2026-09-21-001359-aiportal-front-improve/release.json. 전자는 스킬 탐색도 실패, 후자는 원문 접근 후 정책 입력 결손. 전자의 최초 lock 설명은 현재 조회와 달라 사실로 재사용하지 않는다.
- .gitlab-ci.yml은 브랜치별 전용 Runner build/cp 배포다. 실패한 앱 CI job/step 로그는 미확인. 자동 문구의 “워크플로 2회 실패”를 실제 GitHub job 실패 두 건으로 취급하지 않는다.
- run.sh release_project → run_agent → release.json → gate.py cmd_release/evaluate_release. gate는 failed를 정상 차단한다. 버전 판단은 에이전트 단계이며 gate 통과가 판단의 정당성을 증명하지 않는다.
- release_context는 gh 조회 오류를 (없음)으로 표시한다. 이 정적 코드 사실과 이번 과거 조회가 실패했다는 주장은 별개다. 이번 원격 조회는 미실행; 외부 소유 수정은 이번 범위 밖이다.
- test_gate.py Release는 상태/자산/타입 차단 테스트, test_sim.py는 사전 릴리즈 값과 가짜 에이전트에 의존한다. 버전 정책이 없는 실제 모델 판단을 증명하지 않는다.

후속 실행 계획과 체크포인트

1. [BLOCKED] 새 정책/실제 이력 입력이 회차에 추가된 경우에만 각 필드 출처를 기록한다. 증거: 원문 위치·확인 값·정본 대조. 사람 승인 요청을 새로 만들지 않는다; 근거가 없으면 여기서 종료. 같은 조사를 반복하지 않는다.
2. [미착수] 출처가 모두 확보되면 후속 릴리즈 역할이 정확한 수정 파일·실제 버전·커밋/태그 방식·검증 명령을 포함하도록 계획을 먼저 갱신한다. 중간 검토: 계획과 입력의 일치 확인. 현 과제서는 미정 값을 선택할 권한을 부여하지 않는다.
3. [미착수] 실제 릴리즈 단계 수행 후 원래 gate와 출처별 검증을 실행한다. 성공 JSON 직접 제조 금지. 체크포인트: 실제 결과/exit code/버전 세 필드/커밋·태그 증거를 함께 보고 판정한다. gate만 통과하거나 문서만 바뀌면 완료 아님.

이번 실행 명령 (저장소 루트; 전체 stdout/stderr는 scout-checks.json)

```bash
PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
# exit 1, ok=false, state=failed — 수정 전 실제 소비 단계 재현
bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh
# exit 0 — 문법만 확인
git diff --check
git status --short
# 각각 exit 0, 빈 출력
```

이번 gate 단위 테스트·전체 sim·앱 npm ci/test/build·실제 releaser·원격 조회·배포는 미실행이다. 이전 회차 gate 27건 통과를 이번 결과로 복사하지 않는다. 앱의 실제 스크립트는 npm test 및 npm run build:dev이며 일반 npm run build는 없다. 앱 테스트는 이번 버전 관례 결손을 고칠 수 없다.

스킬 적용 및 견적

호출 가능한 Skill 도구가 없어 아래 실제 파일을 읽었다(성공한 Skill 호출 아님). 별도 고정 반환 스키마는 없으며 절차를 이 과제서에 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md

대안 비교: (A) 기존 정본을 다시 정리하는 최소 문서 수정은 이미 수행됐고 장애가 남아 탈락. (B) 버전 정책을 자동 생성하는 확장안은 새 관례 금지와 충돌하여 탈락. (C) 실제 입력을 확보한 뒤 기존 경로로 재개하는 방식을 권고하되 지금은 입력이 없어 미착수. (D) 외부 조회 오류 구분은 유효한 별도 조사 후보이나 범위 밖이며 원인 해결 보장이 없다. 핵심 가정은 사용 가능한 승인 원문/실제 이력이 추가된다는 것이고 현재 충족되지 않았다.

작업량 S는 입력 확보 이후 계획 재작성·인계에만 해당한다. bottom-up으로 출처 대조 5–10분, 계획 갱신 5–10분, 검증 기준 정리 5–10분 = 15–30분의 낮은 신뢰도 실무 추정이며 통계적 신뢰구간이 아니다. 알려진 변동인 자료 해석 보완은 별도 contingency 0–10분, management reserve는 미배정. 정책 결정 대기·실제 빌드/릴리즈 시간은 제외되어 전체 해결 45분을 약속할 수 없다. 유사 과거 회차는 모두 입력 결손으로 끝나 완료시간 비교 근거가 되지 못한다.

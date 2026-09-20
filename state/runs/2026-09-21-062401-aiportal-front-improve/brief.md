- 과제: [수정 과제] 릴리즈 버전 결정 근거 복구 — 원격 조회도 인증 차단, 현재 구현 배정 불가/BLOCKED (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 release-prompt.md 2·3단계는 실제 기존 버전 증가·커밋·태그 관례를 요구하지만 현재 조회 가능한 Git/JSON에는 그 근거가 없어 두 release.json이 failed다. 유효한 정책 또는 실제 릴리즈 출처를 확보해야 기존 검증을 유지하면서 다음 버전을 결정할 수 있으며 앱 코드나 게이트를 바꾸는 것으로 이 결손을 해결할 수 없다.
- 수용 기준: 1) 다음 버전·증가 단위·릴리즈 커밋·태그 사용 여부/형식/종류·노트·자산·검증 방식을 결정할 실제 출처(승인 원문 또는 검증 가능한 과거 이력)가 확보되어 각 결정과 연결된다. 2) 출처가 확보된 경우에만 허용된 앱 파일의 최소 변경을 계획하고 해당 출처가 요구한 동일 검증을 로컬에서 통과시킨다; 새로운 관례 창작, failed→skipped/released 변경, 워크플로 조건 완화는 금지한다. 3) 테스트는 버전 파일 간 일치와 실제 정책 준수, 실패 시 게시 차단을 증명해야 하며 기록 검사나 게이트 단위 테스트 성공을 릴리즈 성공으로 사용하지 않는다. 현재 1 미충족, 2·3 미착수다.
- 건드릴 파일: 지금 수정 가능한 앱 파일 없음. 읽은 docs/RELEASE.md: 릴리즈 판단 근거/미확인 항목 — 승인 출처가 실제 생긴 뒤에만 연결 검토; package.json:version 및 package-lock.json:version/packages[""].version — 현 값 세 곳 0.0.0, 지금 변경 금지. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 원인 확인용 읽기 대상이며 이 앱 회차 수정 대상이 아니다.
- 검증 명령: 아래 읽기 전용 진단 및 조건부 검증 참조. 실제 동일 릴리즈의 통과 명령은 필수 정책·검증 관례가 없어 확정 불가(미확인); npm test/build로 대체하지 않는다.
- 위험과 피할 것: auth·storage·router·.gitlab-ci.yml·외부 러너/게이트 수정 금지. 과거 문서 정본화 재작업, 스킬 경로 수정 재선정, 근거 없는 0.0.1 증가, 릴리즈 이력 없음 단정, 큐 삭제/자동 릴리즈 비활성화로 성공 처리 금지. 코드 수정·커밋은 정찰 역할에서 금지. 새 정책 입력 없이 동일 조사·게이트 재실행·앱 구현을 반복하지 않는다.
- 차선 후보: 없음 — 고정 릴리즈 과제를 useAppList/문서 개선 또는 외부 러너 수정으로 대체할 권한과 인과 근거가 없다. 신규 출처가 없으면 구현자는 BLOCKED/미완료로 인계한다.

실행 순서와 체크포인트
1. [완료: 정찰] 현재 소스와 실패 경로 확인. HEAD e938e8e, non-shallow 37커밋, 로컬 태그 0개, package 이력도 모두 0.0.0. 최초 lock 두 버전 필드 부재를 재확인했다. .github/workflows와 CLAUDE.md 없음. 체크포인트: 인간 확인 요청 없이 아래 근거 충족 여부만 판단한다.
2. [BLOCKED] 이번 회차 입력에는 신규 승인 정책이 없다. 원격 읽기 명령도 인증 실패했다. 구현자는 새로운 출처가 전달된 경우만 1회 대조하고 계획을 수정한다. 출처가 없으면 저장소 무변경으로 종료하고 수정 미완료를 원장에 기록한다. 스킬이 별도 승인을 요구해서 멈추는 것이 아니라 사용자 AGENTS.md의 “확인되지 않은 관례를 새로 만들거나 릴리즈 판정 조건을 완화하지 않는다”와 release-prompt.md의 기존 관례 요구를 동시에 충족하지 못한 상태다.
3. [미착수] 근거가 생겨 범위가 확정된 경우만 구현·검증한다. 검증 실패 시 다음 단계로 진행하지 않으며 예상과 다르면 계획부터 수정한다. 정찰 과제는 릴리즈 커밋·태그·게시를 직접 수행하도록 권한을 추가하지 않는다.

이번 확인 및 검증 명령
- 저장소 루트: `git log -30 --oneline`, `git rev-parse --is-shallow-repository`, `git rev-list --count HEAD`, `git tag --list`, `git status --short`, docs/RELEASE.md의 첫 Git/JSON 블록(회사 스킬 5개를 추가 로드하는 별도 블록 제외). 실제 실행하여 현재/최초 JSON 및 package 버전 이력을 확인했다.
- 원격 읽기: `timeout 25 git ls-remote --tags origin` → exit 128, 인증 사용자 입력 불가. `timeout 25 gh release list --limit 10 --json tagName,name,publishedAt` → exit 4, 인증 없음. 태그/릴리즈가 없다는 결과가 아니다. 인증 설정 변경·토큰 생성/출력·원격 쓰기는 하지 않았다.
- 외부 게이트의 정확한 명령은 `python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve`이다. 이번 실행하지 않았다. 코드상 failed 입력을 차단하는 정상 동작이며 이를 재실행해도 모델의 버전 결정은 재현되지 않는다.
- 앱에 실제 있는 명령: `npm ci`, `npm test`, `npm run build:dev`. node v22.23.1/npm 10.9.8 확인, node_modules 없음. 이번 미실행이고 일반 `npm run build`는 없다. 해당 검증은 정책 입력 결손을 해소하지 못한다.
- 기록 검증: 생성한 ideas.json 필수 필드/범위/상태/기존 21개 보존과 profile 60줄 이내, journal 단일 정찰 노트, `git diff --check`/`git status --short`를 확인한다. 릴리즈 통과 검증과 구분한다.

원인과 대안 비교 (solution-exploration)
- 기존 출처 복구: 최소 범위이며 요구를 그대로 만족한다. 추천하지만 출처 접근/제공이라는 선행조건이 현재 충족되지 않는다.
- 정본/스킬 경로 재정리: 이전 회차 이미 병합됐고 최신 실패는 스킬 접근 이후 발생하여 재선정하지 않는다.
- 외부 release_context 오류 구분: gh 실패를 “없음”으로 표시하는 코드가 있으나 과거 실패 당시 인증 상태와 인과는 미확인이다. 별도 소유 프로젝트 후보로만 유지하며 앱의 차선 과제로 배정하지 않는다.
- 새 릴리즈 정책·게이트 완화: 사용자가 금지한 방법이다. 수행하지 않는다.
- 현 상태 유지: 해결은 아니지만 근거를 조작하지 않는 현재 실행 가능 결과다. 새 출처 없이 반복 BLOCKED 회차를 해결로 기록하지 않는다.

추정 근거 (estimating-and-contingency)
- 출처가 제공된 뒤의 한정 작업을 bottom-up으로 출처 대조 5–10분, 최소 파일 반영 5–10분, 지정 검증/기록 10–15분으로 분해한다(20–35분, 경험적 계획 범위, 통계적 신뢰수준 미측정). 유사 기록은 문서 정본화 1회 merged와 이후 반복 no-change이나 실제 소요시간이 없어 독립 유사 추정 수치를 만들지 않는다.
- contingency: 알려진 문구/검증 명령 정합성 확인 재작업 최대 5분을 별도 둔다. 관리 예비는 이 회차에 배정하지 않는다. 원격 인증 복구·새 정책 승인·실제 배포 시간은 범위 밖이며 45분 완료 약속에 포함할 수 없다. 현재는 구현시간 추정 자체가 조건부다.

스킬 사용 기록
호출 가능한 Skill 도구 없음. 아래 실제 원문을 읽고 절차를 적용했다(성공한 Skill 호출 아님, 고정 반환 스키마 없음).
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
GAO/Green Book 원문은 이번 열지 않았으며 권위 기반 비용·편익 분석이나 통계적 확률을 주장하지 않는다.

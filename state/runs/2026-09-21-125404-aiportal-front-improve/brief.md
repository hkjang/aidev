- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 왜: 실제 실패는 다음 버전·릴리즈 커밋·태그 관례를 결정할 근거가 없는 상태에서 발생했고, 현재 저장소에서도 그 근거를 찾지 못했다. 출처 있는 정책 또는 실제 릴리즈 이력을 확보해야 임의 관례나 판정 완화 없이 동일 릴리즈 절차를 실행할 수 있다.
- 수용 기준: 1) 증가 규칙·커밋 메시지·태그 형식·노트·자산 유무·검증 명령을 결정할 실제 출처와 적용 범위가 확보된다. 2) 그 출처에 따라 package.json과 package-lock.json의 두 버전 필드 및 필요한 노트가 일치하고, 출처가 요구하는 실제 검증이 통과한다. 3) 실제 releaser의 버전 결정 단계와 기존 gate를 통과한 실행 증거가 있으며 실패 JSON을 수동으로 성공으로 고치거나 sim 결과로 대체하지 않는다. 현재 1 미충족, 2·3 미착수다.
- 건드릴 파일: 현재 승인된 코드 수정 대상 없음. 입력이 확보된 뒤에만 docs/RELEASE.md:「미확인 항목과 한계」에 출처와 정책을 구분해 연결; package.json:version 및 package-lock.json:version/packages[""].version은 그 출처가 요구할 때 함께 갱신한다. 릴리즈 노트 경로·태그·커밋 형식은 미확인이라 지금 지정하지 않는다.
- 검증 명령: 저장소 루트의 `git diff --check`, `git status --short`, `git rev-parse --is-shallow-repository`, `git tag --sort=-creatordate`, `git log -30 --oneline` 및 아래 기록 검증 명령은 이번 실행 확인. 앱에는 `npm test`, `npm run build:dev`가 실제 scripts로 존재하지만 이번 미실행이며 node_modules 없음. 설치와 실행은 구현 범위·정책 입력이 확보된 뒤에만 하며 릴리즈 검증 대체로 삼지 않는다.
- 위험과 피할 것: 0.0.1 또는 v0.0.1을 임의 선택하지 말 것. 과거 README 0.0.1은 실제 릴리즈 미확인이다. docs/RELEASE.md 재정리·AGENTS 추가·스킬 폴백 복구를 반복 해법으로 배정하지 말 것. .gitlab-ci.yml, auth/storage/router, 외부 aidev/run.sh·gate·큐·정책은 수정 범위 밖이다. skipped/released로 바꾸기, 검사 삭제, 무관 앱 개선, 커밋·태그·원격 전송은 이번 정찰 범위 밖이다.
- 차선 후보: 없음 — 우선 과제 고정이므로 다른 개선으로 바꾸지 않는다. 외부 러너의 조회 실패/빈 목록 구분은 별도 소유자의 후보이며 이번 앱 실패를 고친 것으로 셀 수 없다.

판정과 구현자 인계

이번에 새로운 BLOCKED 구현 작업을 배정하지 않는다. 기존 미해결 수정 과제의 상태를 보존하며, 신규 입력 없이 같은 조사·gate·sim·앱 테스트를 반복하는 것은 완료 경로가 아니다. 이 파일은 실행 가능한 코드 수정안이 없다는 정찰 결과이며, 수정 또는 동일 검증 통과를 주장하지 않는다. 고정 과제 해결 요청과 관례 신설 금지·앱 저장소 범위 사이의 제약을 숨기지 않는다.

실제로 읽은 근거

- AGENTS.md, README.md, docs/RELEASE.md, docs/06·07·08·09의 관련 설정·로드맵·테스트 안내, package.json, vite.config.js, vitest.config.js, .gitlab-ci.yml. CLAUDE.md와 .github/workflows는 없다. src/docs TODO/FIXME 검색은 일치 없음.
- HEAD e938e8e, non-shallow 37커밋, 최근 git log -30, 로컬 태그 0개. 현재 package 및 lock 두 필드 0.0.0; 최초 package 0.0.0, 최초 lock 두 필드는 부재. package/lock 이력은 세 커밋이다.
- /mnt/c/Users/USER/projects/aidev/release-prompt.md: 절차 2·3은 실제 증가·커밋·태그 관례, 5는 버전 파일도 없을 때만 skipped를 허용한다.
- /mnt/c/Users/USER/projects/aidev/bin/run.sh: release_context, retry_release_workflow, release_project를 읽었다. release_project는 agent 결과를 gate로 평가한다. retry_release_workflow는 GitHub run ID와 태그가 있는 별도 경로다. release_context는 gh 오류도 '(없음)'으로 기록하므로 출력만으로 원격 이력 부재를 확정할 수 없다.
- /mnt/c/Users/USER/projects/aidev/bin/gate.py: evaluate_release, cmd_release는 failed를 거부하는 정상 경로다. /mnt/c/Users/USER/projects/aidev/tests/test_gate.py:Release는 결과 형식·자산·태그를 검사하며 모델의 버전 결정 정책을 증명하지 않는다.
- /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 2026-09-21-001359-aiportal-front-improve/release.json을 읽었다. 앞 기록은 스킬 탐색+정책 결손, 뒤 기록은 스킬 접근 후 정책 결손이다. 앞 기록의 최초 lock 0.0.0 설명은 실제 blob과 불일치한다. GitHub 워크플로 두 번 실패의 run ID·실패 단계 로그는 미확인이다.
- 이번 원격 조회·npm 설치·앱 test/build·gate·sim·실제 releaser·전용 Runner 배포는 미실행. 기존 실패 JSON의 재평가 통과를 요구하지 않는다. 실패 원인이 수정되지 않아 동일 릴리즈 검증 Green 증거는 없다.

입력 확보 이후의 조건부 순서 (현재 전부 미착수)

1. 정책 입력 출처를 확인하고 위 7개 결정 항목을 채운다. 체크포인트: 자료가 실제 권한 있는 정책 또는 실제 관례인지 확인; 새 관례를 에이전트가 만들어 채우지 않는다. 외부 입력 대기 시간은 45분 범위로 약속할 수 없다.
2. 입력에 따라 수정 파일·정확한 검증 명령을 다시 확정한다. 체크포인트: 버전 세 경로의 일치와 기존 실패 원인의 해소가 명확해야 진행한다. 현재는 미확인 명령을 만들어 두지 않는다.
3. 구현 세션에서 출처가 요구하는 검증 및 실제 releaser를 실행하고 기존 gate 결과를 확인한다. 체크포인트: production 경로 증거가 없으면 완료 표시하지 않는다; 원격 게시는 정찰 범위에 포함하지 않는다.

선택지 비교 및 추정 근거

- 실제 관례/승인 정책 복구: 기존 절차를 보존하는 유일한 적합 경로이나 입력 의존. 이를 기존 수정 과제로 유지한다.
- 임의 초기 버전 정책 도입: 작아 보여도 명시적 금지와 충돌하므로 기각.
- 외부 러너 진단 수정: 오류 관찰을 개선하지만 앱 범위 밖이고 정책 입력을 만들지 못하므로 별도 보류.
- 앱 DX 개선: 낮은 위험 후보지만 고정 실패를 해결하지 않으므로 이번 선택에서 제외.
- 입력 확보 후 작업 분해 추정: 출처 대조 5~10분, 버전/정본 반영 5~10분, 기존 검증·인계 10~15분, 알려진 입력 불일치 대응 contingency 0~5분 = 20~40분(S). 통계적 신뢰수준은 미측정이며 신뢰 낮음; 입력 확보와 장시간 빌드·외부 배포는 제외한다. management reserve는 배정하지 않았고 범위가 늘면 재추정한다. 무변경 회차는 성공 구현 시간의 유사 사례로 쓰지 않는다.

스킬 적용

호출 가능한 Skill 도구가 없어 다음 원문을 파일로 읽었다(성공한 Skill 호출 아님). 모두 고정 반환 JSON 스키마는 없으며 이 과제서의 선택지·가정·추정·단계별 증거/체크포인트로 절차를 적용했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
- estimating-and-contingency/references/sources.md도 읽었다. 외부 GAO/Green Book 원문은 조회하지 않았고 통계적 예비비·편익 수치의 근거로 주장하지 않는다. 위 시간은 정찰자의 작업 분해에 따른 조건부 추정이다.

기록 검증(릴리즈 성공 검증 아님)

`python3 /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-125404-aiportal-front-improve/verify-records.py`

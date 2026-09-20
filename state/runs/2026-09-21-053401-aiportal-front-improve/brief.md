- 과제: [수정 과제] 릴리즈 필수 정책 입력 결손 해소 — 구현 배정 불가/BLOCKED (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 release-prompt.md 절차 2·3은 기존 증가·커밋·태그 관례를 요구하지만 현재 저장소와 제공된 실패 기록에는 이를 결정할 출처가 없다. 출처가 확보돼야 실제 릴리즈를 재현할 수 있으며, 문서 정리나 게이트 완화로 이 결손을 해결했다고 처리할 수 없다.
- 수용 기준: 1) 다음 버전/증가 단위, 릴리즈 커밋 형식, 태그 사용 여부·형식·종류, 노트 위치·형식, 자산 및 검증 방식에 대해 승인된 원문 또는 실제 릴리즈 출처가 확보되고 서로 모순되지 않는다. 2) 그 근거에 따른 변경만 적용하며 package.json 및 package-lock.json의 두 루트 버전 필드가 일치한다; 게이트·워크플로·skipped 기준은 그대로 유지한다. 3) 근거에 명시된 실제 사전 검증과 실제 releaser 결과가 통과하며, 기존 failed JSON을 조작하거나 테스트용 released 객체를 주입한 결과를 통과 증거로 사용하지 않는다.
- 건드릴 파일: 현재 허용된 앱 수정 대상 없음. 근거 확보 후에만 docs/RELEASE.md: 확인된 출처·정책 연결, package.json:version 및 package-lock.json:version/packages[""].version 동시 반영 가능 여부를 재계획한다. 노트 파일명·태그·버전 숫자는 현재 미확인이라 지정하지 않는다. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 읽기 전용 진단 대상이며 이번 구현 파일이 아니다.
- 검증 명령: 저장소 루트의 `git diff --check` 및 `git status --short`는 이번 실행에서 통과/빈 출력이다. 앱 명령은 `npm ci`, `npm test`, `npm run build:dev`이며 node_modules가 없어 이번에는 실행하지 않았다. 아래 조건부 명령은 실제 게이트 CLI 배선에 대응하지만 이번에는 재실행하지 않았고 릴리즈 성공을 증명하지 않는다.
- 위험과 피할 것: auth/storage/router, .gitlab-ci.yml, 외부 aidev 전체, 실패 JSON 원본, 원격을 수정하지 않는다. 임의 0.0.1 증가·새 태그 관례·버전 파일 삭제·failed→skipped/released 변경 금지. 이미 병합된 스킬 탐색 안내/정본화 반복, 외부 폴백 경로 수정 재배정, 무관한 기능을 차선으로 구현하는 접근을 반복하지 않는다.
- 차선 후보: 없음 — 자동 배정의 원인 해소 범위에서 실행 가능한 대체안이 확인되지 않았다. useAppList 방어 등은 ideas.json에 보존하지만 이번 회차 대체 구현 대상이 아니다.

상태와 범위
- 수용 기준 1 미충족, 2·3 미착수. 이번 산출물은 정찰 인계이며 수정 완료나 릴리즈 통과가 아니다. 질문 없이 BLOCKED로 기록한다. 반복된 BLOCKED 인계를 새 해법으로 주장하지 않는다.
- HEAD e938e8e, non-shallow, 총 37커밋, 최근 git log -30 재조회, 로컬 태그 없음, 현재 package 및 lock 두 값 모두 0.0.0. CLAUDE.md와 .github/workflows 없음. 최초 lock 필드 부재는 정본/과거 기록 근거이며 이번 최초 커밋 재조회는 미실행. 원격 실제 이력은 미확인.
- .gitlab-ci.yml은 main/develop 전용 Runner의 build/cp 배포다. 태그 기반 릴리즈 워크플로가 아니며 이번 장애의 CI 실패 단계로 지정할 근거가 없다.
- 실제 읽은 실패 원본: /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 2026-09-21-001359-aiportal-front-improve/release.json. 전자는 스킬 탐색과 관례 부족, 후자는 스킬 원문 접근 후 관례 부족으로 failed다. 후자의 원인이 최신 고정 배정과 일치한다.
- run.sh release_project는 run_agent 결과를 gate.py release로 넘긴다. evaluate_release는 status != released일 때 실패로 반환한다. 이는 정상 차단이며 이 함수는 버전 정책을 결정하지 않는다. Release.test_valid 등은 입력 JSON/자산 검사이며 모델의 버전 판단을 검증하지 않는다.
- retry_release_workflow는 실제 gh run 실패/재실행을 조회하는 별도 경로다. 두 release.json의 존재는 GitHub workflow 재실행 실패 두 건의 증거가 아니다. 해당 run ID·실패 job 로그는 미확인이다.

실행 순서와 체크포인트
1. [BLOCKED] 이번 과제서와 회차 입력에 신규 정책 원문/실제 릴리즈 출처가 있는지 확인한다. 현재는 없다. 증명: 인용할 출처·정확한 버전/태그 계약을 제시할 수 있어야 한다. 이 체크포인트는 사용자·정본의 새 관례 금지에서 나온 필수 입력 조건이며 추가 승인 질문을 요구하는 스킬 규칙이 아니다. 입력이 없으면 구현·동일 재조사·gate 반복 없이 미해결로 종료한다.
2. [미착수] 입력이 새로 생기면 그 출처로 과제서의 실제 변경 파일·값·검증 명령을 먼저 확정한다. 불명확한 값은 추정하지 않는다. 사람 검토를 새로 요구하지 않되 승인된 정책에 별도 조건이 있으면 따른다. 여러 버전 소비 경로를 함께 검증한다.
3. [미착수] 확정된 계획을 한 단계씩 수행하고 실제 실패 단계와 같은 런타임 경로로 검증한다. 기존 실패가 차단되는 것은 Red 기준일 뿐 Green은 실제 정책에 기반한 새 결과여야 한다. 전체 releaser는 원격 전송을 포함하므로 이 정찰 명령으로 실행하지 않는다. 로컬 전용 검증 경로가 확인되지 않으면 통과 주장 없이 남은 제약을 기록한다.

조건부 게이트 확인 명령(저장소 루트에서 실행 가능; 지금 재실행할 이유 없음)
```bash
PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
```
기대는 exit 1/state failed다. 이것을 exit 0으로 바꾸는 것이 과제가 아니다. 실제 releaser 재현/수정 후 통과 확인은 필수 정책 결손 및 정찰 코드 변경 금지로 미수행이다. 과거 gate 27건 통과를 이번 검증으로 재사용하지 않는다.

대안 비교와 추정 근거
- 최소 변경: 검증 가능한 기존 출처를 정책 입력으로 연결 — 올바른 방향이나 현재 출처 없음.
- 확장안: 외부 러너의 정책 입력/조회 실패 분류 개선 — 외부 소유, 이번 원인과 인과 미확인으로 배정 불가.
- 새 구성 없는 안: 저장소를 유지하고 부족한 입력을 명시 — 현재 허용되는 처리로 채택하되 수정 성공은 아님.
- 문서 재정비·임의 버전 부트스트랩은 이미 실패한 접근 또는 명시적 금지라 후보로 채택하지 않는다.
- Bottom-up 작업 추정(입력 확보 후에만): 출처 대조 5–10분, 정책에 따른 파일 반영 5–10분, 로컬 검증·기록 10–15분 = 20–35분. 알려진 환경 차이 확인 여유 0–10분을 별도로 두어 20–45분, 신뢰 낮음(통계적 신뢰수준 아님). 정책 확보 대기·실제 배포·외부 러너 수정은 제외, management reserve는 책정하지 않는다. 과거 유사 회차는 no-change라 수정 완료시간의 비교 추정으로 사용할 수 없고 전체 완료시간은 산정 불가다.

스킬 적용
호출 가능한 Skill 도구 없음. 아래 실제 원문을 읽었으며 성공한 Skill 호출이 아니다. 별도 고정 반환 스키마는 없고, 대안 비교·조건부 단계별 증명/체크포인트·범위/가정/불확실성을 위에 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
추정은 이 코드/범위에 대한 정찰 판단이며 외부 권위 문서의 비용·편익·통계 수치를 사용하지 않았다.

- 과제: [수정 과제] 릴리즈 버전 결정 입력 결손 — 기존 미해결 건 유지, 이번 구현 배정 없음 (가치 5 / 위험 2 / 작업량 S)
- 왜: 실제 실패는 외부 releaser가 다음 버전·커밋·태그 관례를 결정하지 못한 단계이며, 앱 코드나 CI 실패로 입증되지 않았다. 승인된 정책이나 실제 릴리즈 근거가 있어야 판정 조건을 유지한 채 진행할 수 있으므로, 반복 BLOCKED 조사를 새로운 수정 성과로 배정하지 않는다.
- 수용 기준: 1) 다음 버전과 증가 단위, 커밋·태그 형식, 노트·자산·검증 절차를 뒷받침하는 승인 정책 또는 실제 이력의 출처가 제공되고 현재 저장소와 연결됨. 2) 그 근거를 바탕으로 파일별 변경과 실제 로컬 검증 절차를 재계획하며, 버전 소비 세 경로가 일치하고 기존 릴리즈 판정을 유지함. 3) 실제 releaser의 버전 결정부터 기존 검증까지 동일 조건으로 통과함을 출력과 함께 기록함; gate 스키마 통과·sim·문서 문자열 검사만으로 대체하지 않음. 현재 1 미충족, 2·3 미착수.
- 건드릴 파일: 이번 회차 수정 대상 없음. 읽은 대상은 docs/RELEASE.md(근거 정본), package.json:version 및 package-lock.json:version/packages[""].version, .gitlab-ci.yml(브랜치 빌드·복사 배포). 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_context/retry_release_workflow/release_project, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release, tests/sim/bin/claude:release 분기를 읽었으며 외부 파일 수정 지시가 아니다.
- 검증 명령: 이번 실행한 읽기 전용 검사는 `git rev-parse --is-shallow-repository`, `git rev-list --count HEAD`, `git tag --list`, `git log -30 --oneline`, `git log --oneline -- package.json package-lock.json`, `git diff --check`, `git status --short` 및 Python JSON/최초 blob 파싱이다. 앱 명령은 package.json에서 `npm test`, `npm run build:dev` 확인; node_modules가 없어 설치·실행하지 않았다. 실제 릴리즈 전용 로컬 명령은 저장소에 확인되지 않았다. 외부 gate CLI는 `python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release <실제 release.json 절대경로> --out-dir <해당 회차 폴더>`이나 결과 스키마/자산 경로 검사일 뿐 버전 결정을 수행하지 않는다. 새 근거 없이 gate/앱 테스트/실제 releaser를 반복 실행하지 말 것.
- 위험과 피할 것: 임의 0.0.1 증가, 가짜 태그/정책, failed를 released/skipped로 바꾸기, 버전 파일 제거, 워크플로 완화, 정본 재정리의 반복, 외부 러너 수정, 무관 기능 대체 금지. auth/storage/router 및 .gitlab-ci.yml 배포 명령을 건드리지 않는다. 현재 원격 이력은 미확인이고 빈 release-context는 원격 이력 부재의 증거가 아니다.
- 차선 후보: 없음 — 우선 과제 고정이므로 다른 앱 개선으로 대체하지 않는다. 별도 회차 후보 useAppList 비배열 캐시 방어(가치 3/위험 2/S)는 이번 구현 권한에 포함하지 않는다.

구현 인계 상태 및 순서:
1. 현재 상태는 배정 불가이며 이미 수행한 조사를 반복할 구현 작업이 없다. 새로운 정책/실제 릴리즈 출처가 들어온 경우에만 그 입력을 확인하고 계획을 수정한다. 현재 사용자에게 질문하거나 승인을 요청하지 않는다.
2. 재개 입력에는 출처, 적용 저장소·기준 커밋, 증가 규칙, 커밋/태그 형식, 노트 위치, 자산 방식, 실제 검증 명령이 필요하다. 일부만 있으면 임의 보충하지 않는다. 이 입력 확인은 기술적 진입 조건이며 이번 스킬이 추가한 승인 절차가 아니다.
3. 근거가 생기면 구현 담당자가 변경·증명·체크포인트를 구체화한다. 근거 없는 성공 판정 없이 종료한다. 현 시점 수정 후 동일 검증 통과는 미확인이고 수정 미완료다.

조사 근거:
- HEAD e938e8e, non-shallow, 37커밋, 로컬 태그 0개; 현재 버전 세 곳 0.0.0. 최초 package 0.0.0, 최초 lock 두 필드 부재를 직접 파싱했다.
- 2026-09-20-211415 및 2026-09-21-001359 회차의 release.json을 읽었다. 앞선 스킬 접근 문제는 후자에서 해소됐으므로 폴백 경로 안내를 이번 해결책으로 반복하지 않는다.
- .github/workflows 및 CLAUDE.md 없음. GitLab CI는 브랜치 빌드/복사이며 실제 CI 실패 로그 없음. GitHub 워크플로 2회 실패의 run ID/실패 step 로그는 미확인.
- 외부 release_project는 run_agent 결과에 gate를 적용하고, evaluate_release는 failed를 그대로 차단한다. tests/test_gate.py의 Release 테스트는 상태·자산·형식 검사이며 모델의 버전 결정을 검증하지 않는다. sim은 release 결과를 만들어 넣으므로 이번 문제의 해결 증거가 아니다.
- fix-queue.tsv의 해당 행은 태그가 빈 일반 릴리즈 실패 문구다. retry_release_workflow의 태그·실패 단계가 있는 2회 실패 경로와 동일하다고 단정할 수 없다. 큐/프롬프트의 표현 불일치 자체를 이 저장소의 수정으로 배정하지 않는다.

대안 비교와 추정:
- 승인 정책/실제 이력 입력 복구: 근본 원인에 대응할 수 있으나 현재 입력 없음. 최소 변화이며 기존 미해결 건으로 유지한다.
- 원격 이력 재탐색: 실제 이력이 있고 접근 가능할 때만 유효. 직전 인증 실패 후 새로운 접근 근거가 없어 반복하지 않는다.
- 외부 runner의 조회 실패 구분/배정 분류 개선: 별도 소유 저장소의 DX 개선이며 정책을 만들어 주지 않으므로 이번 해법 아님.
- 무변경 종료: 이번에 선택한 처리. 수정 성공은 아니지만 관례 발명과 동일 실패의 반복 구현을 피한다.
- Bottom-up 추정(새 입력이 완비된 뒤만): 입력 대조 5–10분, 변경 범위 계획 5–10분, 로컬 검증 10–15분 = 기본 20–35분; 알려진 환경 편차 contingency 0–10분을 별도 둔다. 45분 내는 입력 완비·로컬 실행 가능 조건이며 신뢰도 낮은 계획 범위로 통계적 신뢰구간이 아니다. 정책 확보 대기는 상한 미확인, 범위 외 변경의 management reserve는 운영자 소관/미산정. 유사 성공 회차가 없어 analogous 교차 추정은 불가하다.

스킬 적용 기록:
호출 가능한 Skill 도구 없음. 아래 SKILL.md 원문을 읽었으며 성공한 Skill 호출로 표시하지 않는다. 세 원문에는 별도 고정 반환 스키마가 없어 과제서에 대안·근거·단계·검증·추정 범위를 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
외부 비용 산정 권위 자료의 내용을 근거로 수치나 신뢰도를 주장하지 않는다. 이번 범위는 실제 파일 조사에 따른 조건부 작업 분해다.

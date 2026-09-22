- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 신규 실행 배정 불가, 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 releaser는 다음 버전·커밋·태그 관례를 확정하지 못해 failed를 반환하며, 현재 Git/JSON에서도 이를 복구할 신규 출처가 확인되지 않았다. 유효한 출처를 입력한 뒤 동일 릴리즈 절차를 통과해야 해결되며, 동일 BLOCKED 조사나 문서 추가만으로는 반복 실패가 고쳐지지 않는다.
- 수용 기준: 1) aiportal-front 및 적용 기준 SHA에 대한 실제 릴리즈 기록 또는 승인된 정책 원문을 확보하고 증가 단위·릴리즈 커밋/태그 양식·노트 위치/형식·자산 방식·검증 명령을 출처와 연결한다. 2) 그 출처에 따라 package.json 및 package-lock.json 두 필드 등 실제로 필요한 모든 버전 소비 경로를 일치시키고, 기존 관례에 맞는 로컬 릴리즈 결과를 만든다. 3) 실제 releaser가 만든 결과를 현재 cmd_release/evaluate_release로 검사하여 exit 0/ok=true를 확인하고 앱 검사 및 출처가 요구하는 검사 결과도 함께 남긴다. 임의 released JSON, 문자열 검사, sim/대역 통과는 증거가 아니다.
- 건드릴 파일: 지금은 없음. docs/RELEASE.md: 읽기 전용 근거 정본, 정책 출처가 실제로 확보된 경우에만 새 근거를 정확하게 연결. package.json: version, package-lock.json: version 및 packages[""].version — 근거 확보 뒤에만 동시 갱신. 릴리즈 노트 경로는 출처가 없으므로 미확인, 임의 파일 생성 금지. /mnt/c/Users/USER/projects/aidev/bin/run.sh: release_context/release_project/retry_release_workflow, bin/gate.py: cmd_release/evaluate_release, tests/test_gate.py: Release — 모두 읽은 외부 실행 경로이며 이번 저장소의 수정 대상이 아니다.
- 검증 명령: 아래 명령과 한계를 따른다. 현재 수용 기준 1 미충족, 2·3 미착수. 정찰은 코드 수정 금지 역할이므로 수정 후 통과를 주장하지 않는다.
- 위험과 피할 것: 관례 발명, 0.0.1 임의 증가, failed→skipped/released 재분류, gate/워크플로 완화, 외부 aidev 수정, auth/router/storage 기능 변경 및 커밋·태그·원격 전송 금지. GitLab 전용 Runner의 reset/cp 스크립트는 로컬 실행하지 않는다. 스킬 경로 안내만 추가하는 기존 접근은 최신 실패를 해결하지 못한다.
- 차선 후보: 없음 — 고정 수정 과제이므로 useAppList 등 무관한 개선으로 바꾸지 않는다. 신규 출처가 없으면 동일 BLOCKED 구현을 다시 수행하지 않고 기존 pending을 유지한다.

판정과 근거

- 기준 HEAD e938e8e1c10974e6afd49e49fe5667d7988961ad. non-shallow, 전체 37커밋, git log -30 직접 확인, 로컬 태그 0개, 작업 트리 변경 없음.
- docs/RELEASE.md를 먼저 읽고 package/lock의 변경 커밋별 JSON 재조회: 현재 세 필드는 0.0.0, 증가 이력 없음, 최초 lock 두 필드 부재. 0.0.1/2025-01-01은 정본상 실제 릴리즈 미확인이다.
- CLAUDE.md와 .github/workflows 없음. .gitlab-ci.yml은 main/develop의 브랜치 빌드 및 복사 배포. npm ci/test 단계 없음. 실패한 GitHub run ID/step 로그와 원격 이력은 이번 미확인.
- 외부 release-prompt.md 2·3은 실제 관례를 요구하고 5는 버전 파일도 없어야 skipped를 허용한다. 이 저장소는 그 조건에 맞지 않는다.
- 두 과거 release.json 원문을 읽었다. 09-20-211415는 스킬 접근 문제와 버전 문제를 함께 기록했고, 09-21-001359는 스킬 접근 후에도 버전 근거 부족으로 실패했다. 최초 lock 값에 대한 오래된 실패 사유의 부정확함은 현 Git/JSON 결과를 우선한다.
- 실행 경로는 release_project → run_agent → release.json → cmd_release/evaluate_release이며 failed는 원격 게시 전에 차단된다. retry_release_workflow는 별개의 GitHub 재실행 경로다. 두 JSON을 동일 Actions step 2회 실패 증거로 바꾸지 않는다.

구현 재개 계획과 체크포인트

1. 현재 상태: BLOCKED. 새 출처가 인계되면 원문과 적용 프로젝트/SHA를 먼저 대조하고 위 수용 기준 1의 필드를 채운다. 추가 질문/승인 요청을 만드는 단계가 아니라 실제 입력 유무를 확인하는 단계다. 입력 없으면 이후 실행 금지.
2. 입력 충족 뒤: 파일/버전/노트/자산/검증 명령이 확정된 계획으로 이 과제서를 갱신하고 순서대로 수행한다. 아직 정해지지 않은 버전이나 명령을 이 문서에서 추측하지 않는다. 실제 변경 범위가 45분을 넘으면 별도 분할한다.
3. 증명: 실제 릴리즈 산출물과 동일 게이트 결과, 앱 검사 결과를 원장에 기록한다. 실패가 유지되면 pending이며 완료 표시 금지. 현재 정찰에서는 코드/커밋을 만들지 않는다.

실행 가능한 검증 명령

저장소 루트의 읽기 전용 확인:
```bash
git rev-parse HEAD
git rev-parse --is-shallow-repository
git tag --list
git log -30 --oneline
python3 -B -c 'import json; p=json.load(open("package.json")); l=json.load(open("package-lock.json")); print(p["version"], l["version"], l["packages"][""]["version"])'
git diff --check
```
실제 실패 결과 재현(이번 두 건 각각 exit 1/state failed/ok=false 확인):
```bash
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
```
이 명령의 실패는 기준선이며 수정 후 통과가 아니다. 새 출처 없이 구현자가 다시 실행할 필요 없다. 재개 후에는 같은 gate.py release에 실제 새 release.json과 그 OUT_DIR를 전달한다(아직 파일이 없어 경로 미확정).
앱 검사는 package.json의 실제 명령 `npm ci`, `npm test`, `npm run build:dev`이다. node_modules가 없어 이번 미실행이며 설치/빌드는 정찰 쓰기 범위 밖이다. 릴리즈 검사의 대체가 아니다.

해결안 비교 및 추정 근거

- 선택: 실제 기존 릴리즈 증거를 복구하거나 이미 승인된 정책을 인계받아 입력 결손을 해결한다. 새 정책을 만들지 않으며 지금은 실행 가능한 출처가 없다.
- 외부 release_context의 조회 오류/빈 목록 구분 개선은 진단 품질 후보이나 이번 실패 해결과 인과가 미확인이고 외부 소유라 선택하지 않는다.
- 아무 변경 없이 기존 pending을 유지하는 것은 현재 제약에서 가능한 상태 처리다. 해결 성과로 세지 않는다. 범용 정책 프레임워크 도입은 범위가 크고 정책을 대신할 수 없다.
- 가정: 적용 가능한 신규 출처가 외부에 존재하고 별도로 제공될 수 있음(미확인). 이 가정이 없으면 구현 일정을 추정할 수 없다.
- S는 출처 확보 후 입력 점검/반영의 조건부 크기다. bottom-up 잠정치: 출처 점검 5–10분, 최소 반영 10–15분, 로컬 확인·기록 5–10분 = 20–35분, 알려진 입력 불일치 여유 0–10분, 총 20–45분. 통계적 신뢰 수준은 산정할 표본이 없으며 낮은 신뢰다. 유사 추정 비교: 직전 여섯 회차가 no-change여서 성공 소요시간 비교 불가; 전체 릴리즈의 45분 완료 약속으로 사용 금지.
- 출처 획득 대기·의존성 설치·전체 빌드·외부 Runner/배포는 위 추정에서 제외하며 소요 미확인. 미지 범위의 management reserve는 별도 미배정. 입력 확보 시 재추정한다.

스킬 적용 기록

호출 가능한 Skill 도구 없음. 아래 원문을 직접 읽었으며 성공한 Skill 호출로 간주하지 않는다. 별도 고정 반환 스키마 없음. 분해·추정 근거/예비 시간 구분, 변경·증명·체크포인트, 해결안 비교/선택/가정을 위에 기록했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/references/sources.md (목록 열람; 외부 권위 문서의 비용/확률 주장은 사용하지 않음)
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md

아이디어 재평가

직전 ideas.json 59개 제목/상태를 보존하고 새 후보 2개를 추가했다. 직접 읽은 범위에서 done/rejected로 바꿀 새 근거는 없으며 나머지는 동일 HEAD와 이전 기록 기준 보존이다. 모든 후보에 가치/위험/작업량이 있고 이번 고정 과제도 pending이다. 총 61개이며 신규 후보는 우선 과제의 대체가 아니다.

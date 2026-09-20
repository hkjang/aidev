- 과제: [수정 과제] 릴리즈 버전 결정 입력 결손 해소 — 실행 가능한 구현 배정 없음/BLOCKED (가치 5 / 위험 2 / 작업량 S)
- 왜: 실제 릴리즈 결과는 버전·커밋·태그 관례 근거 부족으로 failed이며, 앱 빌드나 GitHub 워크플로 단계의 실패 증거는 없다. 승인된 정책 또는 실제 릴리즈 이력으로 필수 입력이 채워져야 기존 검증을 유지한 채 릴리즈를 재개할 수 있다.
- 수용 기준: 1) 다음 버전·증가 단위·릴리즈 커밋 양식·태그 사용 여부/형식/종류·노트/자산 생성 방식 각각에 승인 원문 또는 실제 릴리즈 기록 출처가 연결된다(현재 미충족). 2) 출처가 정해 준 변경만 반영하고 package.json 및 package-lock.json의 두 버전 필드가 일치하며, 필요한 원래 검증과 자산 생성이 통과한다(미착수). 3) 실제 releaser가 생성한 새 결과를 수정하지 않은 gate.py release에 전달하여 exit 0, ok=true, state=released를 확인한다. 예전 failed JSON 편집, 대역 결과, gate 단위 테스트 통과만으로 3)을 충족하지 않는다(미착수).
- 건드릴 파일: 현재 승인된 수정 대상 없음. docs/RELEASE.md — 새 근거가 실제로 제공되었을 때만 출처·정책 연결을 검토; package.json:version 및 package-lock.json:version/packages[""].version — 정책 확인 후에만 동시 갱신. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 실제 읽은 진단 대상이며 수정 대상이 아니다. .gitlab-ci.yml은 브랜치 빌드·복사 배포이며 수정 금지.
- 검증 명령: 아래 명령/검증 상태 참고. 이번 정찰의 기록 검사와 릴리즈 해결 검증을 구분한다.
- 위험과 피할 것: 스킬 안내·정본 문서 재정비는 이미 머지됨. 같은 과제의 새 제목·반복 조사로 해결을 주장하지 않는다. 임의 0.0.1/1.0.0 증가, 태그 관례 발명, 버전 파일 삭제로 skipped 만들기, gate/workflow 완화, 외부 aidev 수정, auth/router/storage/배포 변경 금지. 보호된 실제 배포 스크립트 실행·커밋·태그·원격 쓰기도 이 정찰 범위 밖이다.
- 차선 후보: 없음 — 고정 배정이므로 useAppList·문서 교정으로 대체하지 않는다. 필수 입력이 없으면 수정 미완료/BLOCKED를 기록하고 종료한다.

현재 판정과 진입 조건

이 문서는 이전 BLOCKED 접근을 새 수정안으로 다시 채택하라는 지시가 아니다. 자동 배정은 유지하지만, 이번 저장소에서 45분 안에 완료 가능한 수정안을 확보하지 못했다. 구현자는 새 입력이 없다면 동일 Git 조사·gate 재현·문서 수정을 반복하지 않는다. 질문하지 말라는 지시를 준수하며 승인 요청도 만들지 않는다.

확인한 근거: HEAD e938e8e, non-shallow 37커밋, 로컬 태그 없음, 현재 세 버전 0.0.0, 패키지 변경 이력 3건에서 릴리즈 양식 미확인. CLAUDE.md 없음. GitHub workflows 없음. npm scripts에 일반 build 없음. .gitlab-ci.yml은 main/develop Runner의 환경별 빌드와 복사 배포이다.
실패 원본: /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json. 전자는 스킬 탐색과 버전 근거 결손, 후자는 원문 접근 후 버전 근거 결손이다. 독립 실패 두 건과 ‘동일 GitHub 워크플로 두 번 실패’는 구분한다. 후자의 최초 lock 필드 설명을 채택하며 전자의 최초부터 lock 0.0.0 설명은 사용하지 않는다.
실제 소비 경로: run.sh release_project → run_agent → release.json → gate.py cmd_release/evaluate_release. failed는 태그·자산 판정 전에 차단된다. 조회 오류를 (없음)으로 표시하는 release_context 문제는 존재하지만 이번 원인이라는 인과는 미확인이고 외부 저장소 수정은 허용 범위 밖이다. 원격 현재 상태는 이번 미조회.

조건부 실행 계획 (새 근거 도착 전 모두 미착수)

1. 제공된 새 출처를 위 수용 기준 1의 항목과 매핑한다. 변경 면은 docs/RELEASE.md의 사실 근거만; 사람이 이 세션을 지켜보는 승인 단계는 추가하지 않는다. 체크포인트: 필수 항목 공란이면 종료, 출처가 충돌하면 계획을 갱신하고 임의 선택하지 않는다.
2. 출처가 지정한 최소 버전·노트 파일만 수정한다. 체크포인트: JSON의 세 버전 일치 및 출처와의 일치 확인, 소스 기능 변경 없음. 구체 버전·태그·명령은 아직 미확정이므로 추측해서 실행하지 않는다.
3. 실제 releaser와 원래 검증 경로를 실행해 새 결과와 로그를 보존한다. 체크포인트: 수용 기준 3 충족 전 완료·released로 기록 금지. 기존 failed 입력은 음성 회귀 자료로 보존한다.

검증 명령과 한계

저장소 루트에서 이번 실행 성공: `git diff --check`, `git status --short`(빈 출력), `git rev-parse --is-shallow-repository`, `git rev-list --count HEAD`, `git tag`, `git log -30 --oneline`, `git log --oneline -- package.json package-lock.json` 및 Python JSON 읽기(0.0.0 세 값). 이는 해결 검증이 아니다.
기존 동일 차단 경로의 정확한 명령(이번에는 반복 실행하지 않음):
```bash
PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
```
위 명령의 기존 관찰은 exit 1/ok=false/state=failed이며 성공 목표가 아니다. 새 결과의 동일 명령은 경로만 실제 새 release.json과 그 회차 디렉터리로 바꾼다. 새 결과가 없어 현재 통과 명령을 제시할 수 없다.
앱 명령은 `npm ci`, `npm test`, `npm run build:dev`; node_modules가 없고 이번 원인과 무관하여 미실행. gate 단위 테스트·sim·실제 releaser·배포도 이번 미실행. 과거 27건 통과는 이번 결과가 아니다.

대안 비교와 추정 근거

권고는 새 정책 입력 전 무변경 유지이다(비용 최소, 해결은 미완료). 실제 기록을 복구하는 방안은 기록의 존재·접근성이 전제이고 현재 미확인이다. 외부 입력 수집 오류 교정은 별도 aidev 범위에서만 검토 가능하며 현재 릴리즈 해결을 보장하지 않는다. 임의 정책 발명과 gate 완화는 금지되어 후보가 아니다.
S는 입력이 완비된 뒤 기록/세 버전 동기화 10–15분, 검증·인계 10–15분의 bottom-up 작업 추정(20–30분)에만 해당한다. 알려진 입력 형식 차이 대응 예비 5–10분을 별도로 두면 25–40분이며 낮은 확신의 계획 범위이지 확률 보장이 아니다. 정책 확보 대기와 배포/새 자산 도구 개발은 제외되어 전체 해결 시간을 45분으로 보장할 수 없다. 관리 예비는 배정하지 않는다. 유사 회차는 모두 no-change여서 유추 방식으로 구현 소요를 보정할 성공 표본이 없다.

스킬 적용

callable Skill 도구 없음. 다음 원문을 실제 읽고 절차를 적용했으며 성공한 Skill 호출로 기록하지 않는다. 별도 반환 JSON 스키마는 원문에 없으므로 사용자 과제서 형식을 따른다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md: 작업 분해·가정·범위 추정·예비 분리. 외부 기관의 확률/비용 수치 사용 없음.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md: 변경/증명/체크포인트·범위 밖 항목 명시.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md: 서로 다른 대안과 입력 전제 비교.
차단 근거는 스킬의 추가 승인 요건이 아니라 사용자 절대 규칙과 AGENTS.md/docs/RELEASE.md의 새 관례 금지이다.

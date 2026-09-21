- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 releaser가 다음 버전·릴리즈 커밋·태그 관례를 확정할 출처 없이 중단했으며 현재 저장소에도 이를 확정할 새 근거가 없다. 적용 가능한 정책 또는 실제 릴리즈 출처를 확보하면 관례를 발명하거나 게이트를 완화하지 않고 릴리즈 준비를 재개할 수 있다.
- 수용 기준: 1) aiportal-front와 기준 SHA에 적용되는 출처에서 증가 단위, 커밋/태그 형식, 노트·자산·검증 절차를 확인하고 출처 위치와 적용 근거를 기록한다. 2) 출처 확인 뒤에만 정본 및 필요한 버전 필드를 일관되게 갱신하며 임의의 0.0.1 증가·과거 릴리즈 성공 주장을 하지 않는다. 3) 실제 releaser가 생성한 새 결과와 실제 산출물을 기존 cmd_release/evaluate_release에 넣어 통과하고, 출처가 요구하는 앱 검사도 통과해야 완료다. 과거 failed JSON 거부·5개 회귀 통과·문서 검사만으로 완료 처리하지 않는다.
- 건드릴 파일: 현재 실행 가능한 코드 수정은 미확정이다. docs/RELEASE.md:「확인한 근거」「미확인 항목과 한계」— 신규 적용 출처가 있을 때만 정본 보완; package.json:version 및 package-lock.json:version/packages[""].version — 근거 있는 릴리즈 단계에서만 같은 값으로 변경. .gitlab-ci.yml, 외부 /mnt/c/Users/USER/projects/aidev/bin/run.sh:release_project/release_context/retry_release_workflow, bin/gate.py:cmd_release/evaluate_release, release-prompt.md:2·3·5, tests/test_gate.py:Release는 직접 읽은 진단 근거이며 이번 수정 대상으로 배정하지 않는다.
- 검증 명령: 저장소 루트에서 `git log -30 --oneline`, `git rev-parse --is-shallow-repository`, `git tag --sort=-creatordate`, `git diff --check`. 아래 재현 명령과 결과를 참고한다. 앱 검사는 의존성 준비 후 `npm test`, `npm run build:dev`이며 이번에는 실행하지 않았다. `npm run build`, `build:prod`, `lint` 스크립트는 없다.
- 위험과 피할 것: 신규 근거 없는 반복 BLOCKED 구현을 다시 수행하지 않는다. 외부 실행기·게이트·실패 JSON·보호 경로(auth/router/storage/CI)를 고치거나 skipped/released를 위조해 통과시키지 않는다. 정찰은 코드·버전·커밋·태그·원격 전송을 하지 않는다. 원격 조회 실패/빈 context는 원격 릴리즈 부재의 증거가 아니다. 앱 검사 및 전용 Runner 배포와 버전 정책 판단은 별개다.
- 차선 후보: 없음 — 고정 우선 과제가 성립하지 않는다고 무관한 앱/DX 개선으로 대체하지 않는다. 기존 수정 과제는 pending 유지한다.

현재 판정: 정찰 완료 / 수정 미완료 / 실행 배정 보류. 이전에 반복한 「자료 없음 → 같은 BLOCKED 구현 실행」을 새 과제로 재배정하지 않는다. 구현자는 새 출처가 인계되지 않았다면 이미 확인한 gate·앱 검사·원격 조회를 반복하지 말고 미해결 상태를 보존한다. 이는 사용자 규칙의 근거 없는 새 관례 금지와 반복 실패 접근 금지에 따른 판단이며, 스킬의 추가 승인 요구가 아니다.

직접 확인한 근거:
- HEAD e938e8e1c10974e6afd49e49fe5667d7988961ad, non-shallow, 전체 37커밋, 로컬 태그 0개. 최근 git log -30과 패키지 파일별 변경 커밋의 JSON을 재조회했다. 현재 package와 lock 두 필드 0.0.0, package 이력 증가 없음, 최초 lock 두 필드 부재.
- CLAUDE.md와 .github/workflows 없음. .gitlab-ci.yml은 main/develop 전용 Runner의 fetch/reset → npm 환경별 빌드 → cp 배포다. CI 파일은 읽기만 했으며 서버 스크립트를 로컬에서 실행하지 않았다.
- 외부 release_project → run_agent → release.json → cmd_release/evaluate_release 경로는 ok=false에서 push 이전에 종료한다. retry_release_workflow는 별도의 GitHub run 재실행 경로다. GitHub run ID와 실패 step 로그는 미확인으로, 이번 두 JSON을 동일 GitHub step 두 번 실패라고 단정할 수 없다.
- 2026-09-20-211415와 2026-09-21-001359의 release.json을 읽었다. 후자는 스킬 원문 접근 후에도 버전 관례 부족으로 중단했다. 따라서 스킬 경로 안내를 다시 고치는 접근은 해결책이 아니다.
- release_context의 gh 오류/빈 결과 혼동과 tag fetch 오류 후 진행은 외부 소유의 별도 후보이며 이번 실패와의 인과가 미확인이다. 범위를 확장하지 않는다.

실행 계획과 체크포인트:
1. [미충족] 적용 출처 인계 확인. 출처는 프로젝트/기준 SHA, 원문 경로 또는 실제 태그·커밋, 증가 단위, 메시지·태그 형식, 노트 위치/언어, 자산 유무·생성법, 정확한 검사 명령을 포함해야 한다. 승인 정책은 실제 승인 기록이어야 하고 과거 관례는 조회 가능한 실물이 있어야 한다. 체크포인트: 근거 없으면 2로 가지 않는다; 사람에게 질문하는 단계는 이번 정찰에 없다.
2. [미착수] 출처와 충돌 없는 최소 변경만 산정한다. docs/RELEASE.md의 과거 미확인 표기를 근거 없이 지우지 않는다. 버전 변경이 정당화되면 세 JSON 필드 일치와 diff를 확인한다. 체크포인트: 새 사실이 계획과 다르면 계획을 수정하고 후속 실행을 중단한다.
3. [미착수] 릴리즈 담당 단계가 기존 절차로 실제 결과를 생성하고 동일 게이트 및 해당 출처의 검사를 통과한다. 정찰/일반 구현 단계에서 외부 run.sh 전체를 실행하지 않는다(게시·배포 부작용 포함). 통과 로그·입력 SHA·산출물 경로를 원장에 붙인 뒤에만 done으로 바꾼다.

재현 기록(이번 정찰 실행):
- `python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release` → 5 tests OK, exit 0. tempfile 쓰기는 TMPDIR을 이번 회차 내부 임시 디렉터리로 지정해 제한했고 종료 후 정리했다. 기존 테스트의 ResourceWarning(unclosed file)이 출력됨.
- `python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve` → exit 1 / failed / ok=false.
- 위 명령의 회차 경로를 `2026-09-21-001359-aiportal-front-improve`로 바꿔 실행 → exit 1 / failed / ok=false.
- 이 결과는 기존 실패의 거부와 게이트 회귀 확인이다. 수정·Red/Green·실제 releaser 재실행·수정 후 동일 검증 통과는 미완료다. 실패 원본 수정 없음. 원격 조회와 앱 설치/test/build 미실행, node_modules 없음.

대안 비교와 추정:
- 추천: 새로운 적용 출처가 실제 제공될 때 기존 수정 건을 재개. 변경면이 가장 작으며 정책 결손을 직접 해결하나 현재 필수 입력이 없다.
- 외부 수집기에서 조회 오류를 구분: 관측 정확성은 개선되지만 버전 관례를 만들지 못하며 별도 소유 저장소/인과 검증이 필요하여 이번 해법으로 제외.
- 릴리즈 정책을 신규 제정: 사람의 정책 결정 없이는 허용되지 않으므로 제외. 무변경 상태 보존은 해결 성과가 아니다.
- S는 출처가 완비된 뒤 정본 대조·최소 입력 반영 범위에만 적용한다. 작업 분해 추정은 대조 5~10분, 반영 5~10분, 제한 검증 5~10분으로 15~30분, 알려진 문서/필드 불일치 대응 여유 5~10분을 별도 둔다. 통계적 신뢰수준은 산정할 이력이 없어 미확인, 자신감 낮음. 전체 릴리즈 시간과 출처 확보 대기 시간은 추정 불가이며 관리 예비는 배정하지 않았다. 과거 회차는 no-change여서 성공 유사 사례 기반 추정에 사용할 수 없다. 새로운 출처·자산 요구가 나오면 재산정한다.

스킬 적용 기록:
호출 가능한 Skill 도구 없음. 아래 원문을 직접 읽어 적용했으며 성공한 Skill 호출로 간주하지 않는다. 세 원문에 별도 고정 반환 스키마는 없으며, 위 대안 비교·근거/가정/범위 추정·변경/증명/체크포인트로 절차를 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
- estimating-and-contingency/references/sources.md도 읽었다. 외부 비용/정책 이론이나 정량 신뢰수준을 주장하지 않았으며 외부 권위 자료 원문은 미조회다. 위 시간 범위는 이번 한정 작업의 비통계적 작업 분해 추정이다.

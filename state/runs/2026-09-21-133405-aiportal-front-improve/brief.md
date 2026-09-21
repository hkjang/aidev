- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건의 재개 계약 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 릴리즈 에이전트는 기존 증가·커밋·태그 관례를 요구하지만 현재 Git/JSON과 정본에는 이를 결정할 근거가 없다. 출처 있는 입력이 확보되어야 임의 정책이나 gate 완화 없이 실제 릴리즈 검증으로 진행할 수 있다.
- 수용 기준: 1) 승인된 정책 또는 실제 과거 릴리즈 출처가 다음 버전·커밋 메시지·태그 형식과 종류·노트·자산 생성 여부 및 검증 명령을 결정할 만큼 제공되고 정본과 모순되지 않는다. 2) 구현자는 그 근거로 최소 수정 범위를 확정하고 package.json, package-lock.json 루트 및 packages[""].version 세 값과 실제 릴리즈 결과를 일치시킨다. 3) 출처가 요구하는 실제 빌드/검사와 릴리즈 에이전트 실행이 성공하고, 새 결과를 동일 gate CLI로 검증한다. 기존 실패 JSON을 released로 바꾸거나 스텁/문자열 검사를 성공 근거로 삼지 않는다.
- 건드릴 파일: 지금 수정 허용 대상으로 확정된 앱 파일 없음. 신규 출처가 생길 때 docs/RELEASE.md: 확인한 근거/미확인 항목에 출처와 적용 범위 기록; package.json:version 및 package-lock.json:version/packages[""].version은 그 출처가 실제 증가를 요구할 때만 수정. 노트·자산 경로와 생성 함수는 현재 미확인으로 임의 신설 금지. 외부 aidev/release-prompt.md 절차 2·3·5, bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 직접 읽은 진단 자료이며 이 저장소 수정 대상으로 배정하지 않는다.
- 검증 명령: 아래 실제 실행 명령과 조건부 후속 검증 참조.
- 위험과 피할 것: 임의 0.0.1 증가·새 태그 관례, skipped 조건 확대, 실패 JSON 조작, .gitlab-ci.yml/.github/workflows 및 외부 러너/gate 변경, auth/storage/router 변경 금지. 승인 근거 없는 재조사·gate 반복만으로 구현 완료/수정 성공을 기록하지 않는다. 2026-09-08 롤백·강등 교훈과 앞선 여섯 무변경 배정을 반복하지 않는다.
- 차선 후보: 없음 — 우선 과제가 고정되어 무관한 앱 개선으로 교체하지 않는다. 신규 입력 없는 한 현재 미해결 건을 유지하고 구현을 새로 배정하지 않는다.

현재 판정: 수용 기준 1 미충족, 2·3 미착수. **수정 및 동일 검증 통과 미완료.** 이 문서는 새 BLOCKED 구현 작업이 아니라 기존 pending 수정 과제의 재개 계약이다. 구현자는 신규 정책/출처가 실제 추가되지 않았다면 반복 조사나 테스트를 실행하지 않고 미완료 상태만 인계한다. 질문·승인 요청은 하지 않는다. 정찰은 사용자 절대 규칙에 따라 코드를 수정하지 않았다.

근거와 실패 위치
- HEAD e938e8e, non-shallow, 전체 37커밋, git log -30 직접 조회, 로컬 태그 0개. 현재 세 버전 값은 0.0.0, 최초 package는 0.0.0이며 최초 lock의 두 필드는 부재였다. 원격 이력은 이번 미조회이므로 없다고 확정하지 않는다.
- CLAUDE.md와 .github/workflows는 없다. .gitlab-ci.yml은 main/develop 전용 Runner의 fetch/reset/build/cp 배포이며, 버전 릴리즈 검증 스크립트가 아니다. docs/RELEASE.md는 증가 정책이 아님을 명시한다.
- 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md의 2·3은 과거 관례를 요구하고 5는 버전 파일이 없는 경우까지 요구한다. 파일이 있으므로 skipped에 해당하지 않는다.
- run.sh:release_project는 run_agent 결과를 gate release에 전달하고 ok=false면 push 이전에 종료한다. gate.py:evaluate_release는 status != released를 거부한다. tests/test_gate.py:Release는 상태/태그/자산 검사이며 버전 선택을 수행하지 않는다.
- 2026-09-20-211415 및 2026-09-21-001359 회차 release.json 두 개를 직접 읽었다. 첫 기록에는 스킬 접근 실패와 최초 lock 값에 대한 부정확한 설명이 있고, 후자는 원문 접근 성공 뒤 정책 결손으로 실패했다. 두 실패 기록은 확인했지만 독립 GitHub Actions run ID/실패 step 로그는 미확인이다.
- release_context는 gh 오류를 '(없음)'으로 출력할 수 있고 release_project는 tag fetch 오류 후 계속한다. 이는 외부 진단 후보이나 이번 버전 정책 결손과 인과는 미확인이다. 이미 보류된 외부 수정안을 이번 앱 해법으로 재선정하지 않는다.

실행·검증 기록
저장소 루트에서 아래 CLI를 각 1회 실행했다. 둘 다 exit 1, ok=false, state=failed다. 실행 결과는 같은 폴더 release-recheck.json에 보존했다. 실패 결과를 소비하는 운영 CLI 경로 재현이며 에이전트의 판단을 새로 재현한 것은 아니다.
```bash
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
git diff --check
git status --short
```
앱 검증 명령은 package.json에서 확인한 `npm ci`, `npm test`, `npm run build:dev`이다. 현재 node_modules가 없고 이번에는 설치/test/build 미실행이다. 일반 `npm run build`, `build:prod`, `lint`는 없다. 앱 검사 통과도 정책 복구나 전용 Runner 배포 성공을 증명하지 않는다.

조건부 구현 순서와 체크포인트
1. 신규 근거를 전달받았는지만 확인한다. 없으면 구현 배정 없음으로 종료한다. 있으면 정확한 출처·허용 범위·검증 명령을 이 계획에 보충한다. 체크포인트: 수용 기준 1을 문서와 실제 자료로 대조; 추가 인간 승인을 임의 요구하지 않는다.
2. 근거가 지정한 최소 버전/노트/자산 변경을 하고 해당 검증 명령을 실행한다. 세 JSON 값 및 실제 태그 대상/메시지까지 일치시킨다. 체크포인트: 근거와 불일치하면 계획부터 수정; 실패를 성공으로 바꾸지 않는다.
3. 실제 릴리즈 실행 결과에 위와 같은 gate release CLI를 적용한다(입력/출력 경로만 새 회차로 변경). 실제 releaser 재실행과 게시 권한은 해당 역할 범위에서 처리하며 이 정찰은 수행하지 않는다. 정확한 앱 전용 릴리즈 검사 명령은 현재 미확인이다.

대안 비교와 추정 근거
- 채택: 기존 근거 복구. 새 실행 구성 없이 원래 계약을 만족할 수 있지만 출처가 필요하다.
- 확대안: 외부 러너가 정책 입력을 명시적으로 전달하도록 개선. 다른 저장소 소유이고 정책 내용을 만들어 주지 못하므로 배정하지 않는다.
- 자동 패치/새 정책: 사용자가 금지한 접근이라 제외. 유지: 입력이 없을 때 실패 상태를 보존하는 현재 선택이며 릴리즈 해결은 아니다.
- 추정은 bottom-up: 근거 대조 5–10분, 최소 반영 5–10분, 지정 검사 10–15분, 알려진 환경 차이 대응 contingency 0–10분 → 입력이 완전하고 검사 시간이 짧다는 전제하에 20–45분. 신뢰 낮음, 통계적 확률 아님. 과거 여섯 no-change 사례는 정책 획득 시간 추정에 쓸 성공 유사 사례가 아니다. 정책 획득·원격 배포 소요는 미확인으로 범위 밖; management reserve는 별도 운영 판단이며 추정에 넣지 않음. 45분 이상 검사 요구가 드러나면 재산정한다.

스킬 적용 기록
callable Skill 도구 없음. 아래 원문을 실제 파일로 읽어 절차를 적용했으며 성공한 Skill 호출로 간주하지 않는다. 별도 고정 반환 스키마는 없었다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md — 분해·가정·범위·contingency와 management reserve 구분 적용; references/sources.md도 읽음. 외부 권위 문서 본문은 미조회이며 정량적 비용/편익·확률을 주장하지 않음.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md — 변경·증명·체크포인트 및 진입 조건을 명시.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md — 실제 요구·대안·비선정 사유와 핵심 가정(정책 출처 확보)을 명시.

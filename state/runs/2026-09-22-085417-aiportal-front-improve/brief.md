- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 건 pending, 신규 구현 배정 불가 (가치 5 / 위험 2 / 작업량 S)
- 왜: 실제 releaser는 다음 버전·커밋·태그 관례의 근거 부족으로 중단했고 이번 Git/JSON 재조회에도 새로운 근거가 없다. 적용 가능한 출처가 확보되어야 기존 검사를 유지하면서 릴리즈 준비를 재개할 수 있다.
- 수용 기준: 1) 프로젝트 aiportal-front와 기준 SHA에 적용되는 실제 정책 또는 과거 릴리즈 출처에서 증가 단위·커밋/태그·노트·자산·검사 명령을 확인한다. 2) 그 출처에 따라 필요한 최소 변경을 수행하며 package.json 및 lock 두 버전 필드를 일치시키고 미확인 과거 기록을 성공으로 바꾸지 않는다. 3) 실제 releaser가 만든 새 결과·실제 산출물로 기존 cmd_release/evaluate_release가 통과하고 출처가 요구하는 검사도 통과해야 완료다. 실패 JSON 거부나 회귀 테스트 통과만으로 완료 처리하지 않는다.
- 건드릴 파일: 현재 실행 가능한 코드 수정은 미확정. docs/RELEASE.md:확인한 근거/미확인 항목과 한계 — 새 출처 확보 시에만 보완; package.json:version, package-lock.json:version/packages[""].version — 근거가 확정된 릴리즈 단계에서만 변경. 외부 실행기 및 CI는 아래 진단용으로 읽었고 수정 배정하지 않는다.
- 검증 명령: 저장소 루트에서 `git log -30 --oneline`, `git rev-parse --is-shallow-repository`, `git tag --list`, `git diff --check`. 아래에 외부 게이트의 정확한 실행 명령과 결과를 기록했다. 앱 명령은 `npm ci`, `npm test`, `npm run build:dev`이며 이번 앱 실행은 미수행이고 node_modules가 없다. build/build:prod/lint 스크립트는 없다.
- 위험과 피할 것: 새 출처 없는 같은 BLOCKED 구현을 재배정하지 않는다. 임의 0.0.1 증가, skipped/released 위조, 버전 파일 삭제, 실패 원본·게이트·외부 러너·.gitlab-ci.yml 변경을 금지한다. auth/router/storage와 앱 기능은 범위 밖이다. 조회 실패나 빈 context를 원격 이력 부재로 해석하지 않는다.
- 차선 후보: 없음 — 고정 실패 과제를 무관한 앱/DX 개선으로 대체하지 않는다. 근거가 없으면 기존 수정 건 pending을 유지하며 해결 성과로 세지 않는다.

판정: 정찰 완료, 수정 및 수정 후 동일 검증 통과 미완료. 이번 인계는 반복 무변경 구현을 새로 실행하라는 과제서가 아니다. 수용 기준 1이 충족되기 전에는 2·3을 시작하지 않는다. 질문이나 승인 요청 없이 현재 증거로 내린 판정이며 스킬이 별도 승인을 요구한 결과가 아니다.

직접 읽은 근거와 실패 경로:
- HEAD e938e8e1c10974e6afd49e49fe5667d7988961ad; non-shallow, 전체 37커밋, 로컬 태그 0개. git log -30 및 package/lock의 모든 파일 변경 커밋 JSON 재조회에서 버전 증가 없음. 현재 세 필드 0.0.0, 최초 lock 두 필드 부재.
- AGENTS.md, README.md, docs/RELEASE.md, docs/07-개선사항-권장사항.md, docs/09-테스트가이드-총론.md, docs/08-GitLab-CICD-가이드.md의 관련 부분, package.json, vite.config.js, vitest.config.js, .gitlab-ci.yml 확인. CLAUDE.md와 .github/workflows 없음. TODO/FIXME 검색 결과 없음. 19개 unit spec이 있으며 품질 전체 통과 여부는 미확인.
- 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:2·3은 기존 관례를 요구하고 5의 skipped에는 버전 파일도 없어야 한다.
- 같은 루트 bin/run.sh:release_project → run_agent → release.json → bin/gate.py:cmd_release/evaluate_release. failed 결과의 ok=false에서 push 이전 종료한다. .gitlab-ci.yml의 main/develop 전용 Runner 빌드·복사 배포와 다른 경로다.
- run.sh:retry_release_workflow는 별도 GitHub run 재실행 함수다. GitHub run ID/실패 step 로그는 미확인으로 두 failed JSON만으로 동일 Actions step 2회 실패를 확정하지 않는다.
- 2026-09-20-211415 및 2026-09-21-001359 회차의 release.json을 직접 읽음. 후자는 스킬 원문 접근 후에도 버전 근거 부족으로 실패했다. 스킬 경로 재수정은 이번 원인 해결책이 아니다.
- release_context의 gh 오류/빈 목록 혼동과 tag fetch 실패 후 진행은 보이지만 이번 결손의 직접 원인이라는 증거는 미확인. 외부 소유 개선을 이번 저장소 수정으로 전환하지 않는다.

실행 계획 — 변경 / 증명 / 체크포인트:
1. [미충족] 신규 출처를 확인한다. 프로젝트·적용 SHA·원문 경로/실제 태그·커밋·증가 단위·커밋/태그 형식·노트 언어/위치·자산 생성법/유무·검증 명령이 필요하다. 증명은 원문과 실제 Git/JSON 대조다. 체크포인트: 자료 없으면 재조회 루프·gate·앱 검사 반복 없이 기존 pending 유지.
2. [미착수] 확인된 출처에 맞춰 docs/RELEASE.md 및 필요한 입력만 최소 반영한다. 증명은 출처 대조, 세 JSON 필드 일치, git diff --check. 체크포인트: 정책 충돌 또는 자산 요구 등 범위 변화가 나오면 계획과 추정을 먼저 고친다.
3. [미착수] 릴리즈 담당 단계에서 기존 절차로 실제 결과를 만들고 동일 게이트 및 출처의 검사를 수행한다. 증명은 아래 명령의 실제 새 결과와 산출물, SHA 및 로그다. 체크포인트: 모두 통과한 뒤에만 원장 완료/done. 정찰/일반 구현이 외부 run.sh를 통째로 실행하면 게시 부작용이 있으므로 실행하지 않는다.

이번 로컬 재현:
- `python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release`: 5 tests OK, exit 0; 기존 unclosed file ResourceWarning 있음. TMPDIR을 이번 회차 내부 임시 디렉터리로 지정하고 종료 후 제거했다.
- `python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve`: exit 1, failed, ok=false.
- 위 두 회차 경로를 2026-09-21-001359-aiportal-front-improve로 바꾼 동일 명령: exit 1, failed, ok=false.
- 미래 완료 검사는 `python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-085417-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-085417-aiportal-front-improve` 이다. 이 경로에 실제 releaser가 새 결과를 만든 뒤에만 실행하며, 이번에는 그 결과가 없고 실행하지 않았다.
- 이 재현은 기존 거부 동작 확인이다. 원인 수정, Red/Green, 실제 releaser 재실행, 수정 후 통과, 앱 test/build, 원격 조회·배포는 미실행이다.

대안과 추정:
- 추천: 적용 출처를 확보한 뒤 기존 수정 건 재개. 가장 작은 변경으로 필수 입력 결손을 해소하지만 현재 출처 없음.
- 외부 수집 오류 구분: 관측 개선에는 유효할 수 있으나 정책을 제공하지 못하고 이번 원인과 인과 미확인. 별도 소유 범위라 제외.
- 새 정책 제정/검사 완화: 명시적 금지여서 제외. 무변경 유지도 해결 성과는 아니다.
- S는 출처가 완비된 뒤 대조 5~10분 + 최소 반영 5~10분 + 제한 검증 5~10분의 15~30분에만 적용한다. 알려진 자료 불일치 대응 여유는 별도 5~10분, 관리 예비는 미배정. 통계적 신뢰수준과 출처 확보·전체 릴리즈 소요는 미확인; 추정 자신감 낮음. 유사 성공 사례가 없어 두 번째 방법 비교는 수행 불가하며 새 입력 시 재산정한다.

스킬 기록:
호출 가능한 Skill 도구 없음. 아래 원문을 직접 읽어 적용했으며 성공한 Skill 호출로 간주하지 않는다. 고정 반환 스키마는 없고 대안 비교·작업 분해/가정/범위·순차 계획/증명/체크포인트로 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
- estimating-and-contingency/references/sources.md도 직접 읽음. 외부 권위 자료 원문은 미조회; 비용 이론/정량 신뢰수준의 권위를 주장하지 않으며 시간 범위는 이번 한정 작업의 비통계적 분해 추정이다.

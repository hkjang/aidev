- 과제: [수정 과제] 릴리즈 실패의 필수 정책 입력 결손 해소 (가치 5 / 위험 2 / 작업량 S)
- 왜: 실제 독립 회차 두 건의 release.json이 버전·커밋·태그 관례 부족으로 failed이고, 최신 회차는 스킬 원문 접근 이후에도 중단됐다. 유효한 릴리즈 근거가 있어야 실제 버전 결정을 재개할 수 있으며, 현재 상태에서 문서나 게이트만 바꾸면 실패 원인을 해결하지 못한다.
- 수용 기준: 1) 다음 버전·증가 단위·릴리즈 커밋 양식·태그 사용/형식/종류·노트/자산 방식에 대해 실제 이력 또는 승인 원문의 출처가 확보된다. 2) 그 출처에 따라 package.json 및 package-lock.json의 두 버전 필드를 함께 처리하고 실제 릴리즈 절차의 검증을 수행한다(태그/자산 미사용도 임의 추론하지 않는다). 3) 실제 releaser가 생성한 결과를 수정하지 않은 gate.py release가 통과하며 기존 실패 두 건은 계속 차단된다. 수동 released JSON, 소스 문자열 검사, sim 대역 통과는 성공 근거가 아니다.
- 건드릴 파일: 현재 구현 진입 불가이므로 저장소 변경 대상 없음. 근거 확보 뒤 docs/RELEASE.md: 릴리즈 판단 근거에 출처를 기록하고 package.json:version / package-lock.json:version 및 packages[""].version을 근거대로 함께 처리할 수 있다. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_project, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release, tests/test_sim.py:HappyPath/Agents, tests/sim/run_sim.sh는 읽기 전용 조사 대상이며 이번 앱 회차의 수정 대상으로 지정하지 않는다.
- 검증 명령: 아래 실제 실행 명령과 기대 결과 참조. 앱 npm ci/test/build는 이번 정찰에서 미실행이며 현재 node_modules가 없다.
- 위험과 피할 것: .gitlab-ci.yml·외부 러너/게이트·auth·storage 인증·태그·원격을 수정하지 않는다. 새 릴리즈 정책, 임의 0.0.1 증가, failed→skipped/released 치환, 실패 기록 수정, 스킬 안내/정본 문서 재수정, 외부 aidev 구현 재배정을 금지한다. 다른 앱 버그를 이 우선 과제의 해결로 대체하지 않는다.
- 차선 후보: 없음 — 우선 과제 고정이므로 별도 기능으로 전환하지 않는다. useAppList 비배열 캐시 방어는 미래 일반 개선 회차 후보일 뿐 이번 차선이 아니다.

## 구현 가능성 판정: BLOCKED / 수정 미완료
이번 과제는 자동 배정된 장애를 유지한 기록이지, 앞선 no-change 접근을 다시 구현하라는 지시가 아니다. 현 범위에서 45분 내 해결 가능한 코드 결함은 확정하지 못했다. **새 근거가 없다면 구현자는 문서·코드 수정이나 동일 재조사를 반복하지 말고 이 과제서를 차단 상태로 인계한다.** 수용 기준 1이 없으므로 2·3에 진입할 수 없다. 질문 금지에 따라 확인 질문은 하지 않았다. 이 차단은 스킬이 새로 요구한 승인 단계가 아니라 사용자 지시와 저장소 AGENTS.md의 새 관례 금지에서 발생한다.

## 실제 확인 근거
- HEAD e938e8e, non-shallow, 전체 37커밋, 로컬 태그 0개. git log -30 및 package 파일 이력 확인. package.json/lock root/lock packages[""].version 모두 0.0.0. 최초 ab700ba의 lock 두 필드는 ABSENT.
- CLAUDE.md 없음, AGENTS.md와 docs/RELEASE.md 읽음. README·docs/02·07 로드맵·09 테스트 안내·Vite/Vitest 설정 확인. TODO/FIXME 검색에서 별도 작업 목록 발견 못함.
- .github/workflows 없음. .gitlab-ci.yml은 main/develop 전용 Runner의 환경별 빌드·복사 배포이다. 이번 실패를 앱 CI 장애로 재분류할 근거 없음.
- 외부 release-prompt.md 절차 2·3은 실제 이전 증가/커밋/태그 방식 요구, 절차 5는 버전 파일도 없을 때만 skipped 허용. state/aiportal-front.policy.json의 autonomy=release와 allow_merge_without_ci=true는 버전 정책이 아니다.
- 실제 소비 경로: run.sh:release_project → run_agent → release.json → gate.py:cmd_release/evaluate_release. gate는 failed 입력을 정확히 거절한다.
- 독립 기록: ../2026-09-20-211415-aiportal-front-improve/release.json, ../2026-09-21-001359-aiportal-front-improve/release.json. 앞선 기록의 최초 lock 0.0.0 주장은 현재 직접 조회와 모순되므로 재사용 금지. 최신 기록은 스킬 읽기에 성공했다.
- 이번 신규 조사: gh release list --repo hkjang/aiportal-front --limit 100 --json tagName,name,publishedAt → 인증 필요로 실패. git ls-remote --tags origin → 인증 불가(exit 128). 따라서 원격이 비었다고 새로 확정하지 않는다. 이전 제공 목록의 공란과 원격 현재 상태 미확인을 구별한다. 토큰 생성/설정 변경 없음.

## 실행 순서와 체크포인트
1. [완료] 기존 실패의 입력 및 실제 gate 재현. 아래 두 기록 모두 exit 1, ok=false, state=failed. 사람 검토 없이 읽기/재현 완료.
2. [차단] 새로 제공되는 실제 이력/승인 원문이 있는 경우에만 다음 버전, 커밋, 태그, 노트, 자산 정책을 대조한다. 현재 출처 없음. 임의 기본값이나 승인 요청 템플릿 문서를 앱에 추가하지 않는다. 새 정책 결정은 정찰/구현자의 재량 범위가 아니다.
3. [미착수] 2가 충족되면 과제서를 출처와 구체 값/검증 방법으로 갱신한 뒤 구현한다. 기존 이력이 요구하는 test/build와 버전 세 필드 일치를 검증한다. 현재 그 이력 자체가 없으므로 릴리즈 전용 검증 명령은 미확인이다.
4. [미착수] 동일 실제 releaser의 결과를 원래 gate로 검증하고 성공일 때만 원장에 수정 완료를 기록한다. 현재는 수정 과제/차단으로만 기록. 외부 게시·배포는 정찰의 범위 밖.

## 실제 실행 명령과 결과
저장소 루트에서:
```bash
git rev-parse --is-shallow-repository
git rev-list --count HEAD
git tag --sort=-creatordate
git log -30 --oneline
git log --oneline -- package.json package-lock.json
git diff --check
bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh
PYTHONDONTWRITEBYTECODE=1 TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-020859-aiportal-front-improve/assets python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py
PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve
PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
```
27 gate 테스트 통과(ResourceWarning 있음), bash -n/diff --check 통과. 실제 테스트 TMPDIR은 위 assets 아래 scout-checks 하위 디렉터리를 사용했고 생성된 임시 테스트 자산은 제거했다. 명령별 원 출력/종료값은 scout-checks.json. 두 release 명령의 exit 1은 정상적인 실패 차단 재현이며 장애 해결 성공이 아니다. 전체 releaser·앱 test/build·배포·외부 사용자 검증 미실행. sim은 v0.0.1을 미리 만들고 모델 출력을 대역으로 공급하므로 실행하지 않았다.

## 대안 비교 및 추정 근거 (회사 스킬 적용)
- 최소 해법: 이미 존재하는 원격 이력/승인 원문 확보 후 기존 방식 적용. 새 관례가 필요 없지만 현재 인증·근거가 없어 차단됨. 이것만 수용 가능하다.
- 확장 해법: 외부 러너에 최초 릴리즈 정책 입력 기능 추가. 별도 소유 범위이며 실제 정책 결정은 여전히 필요해서 이번 선택에서 제외.
- 새 구성 없는 해법: 현 failed와 증거를 보존하고 구현 진입을 중단. 현재 정찰의 실행 결과이며 릴리즈 해법/완료로 세지 않는다.
- 문서 안내 반복·스킬 경로 복구 반복은 최신 원인과 무관하여 제외. 게이트 완화·임의 버전은 금지라 대안으로 취급하지 않는다.
- S는 근거가 이미 준비된 경우만 해당: 출처 대조 5~10분 + 세 버전/정본 반영 5~10분 + 기존 검증 10~15분 = 기본 20~35분, 알려진 검증 편차 여유 5~10분을 따로 두어 총 25~45분. 실측 표본 없는 낮은 신뢰의 bottom-up 추정이다. 이전 두 no-change 사례의 유사 추정은 입력이 없으면 완료 시간 상한을 정할 수 없음을 보여준다. 현재 45분 완료를 약속하지 않으며 정책 결정 대기/외부 배포는 제외, management reserve는 배정하지 않는다.
- 가장 큰 가정: 실제 승인 원문 또는 릴리즈 이력이 외부에 존재한다는 것. 존재 여부 미확인; 확인 전 재시도 비용을 반복 지출하지 않는다.
- Skill 호출 도구 없음. /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md, plugins/technology/skills/implementation-planning/SKILL.md, plugins/technology/skills/solution-exploration/SKILL.md 실제 원문을 읽었다. 별도 고정 반환 스키마 없음. 작업 분해·범위/여유 분리·검증/체크포인트·대안 비교를 이 과제서에 반영했다. sources.md의 외부 권위 자료 본문은 조회하지 않았으며 그 자료의 통계적 신뢰도/산식을 주장하지 않는다.

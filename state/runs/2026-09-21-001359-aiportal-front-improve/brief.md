- 과제: [수정 과제] 릴리즈 입력 복구 — 회사 스킬 탐색 안내와 버전 근거 정본화 (가치 5 / 위험 2 / 작업량 M)
- 왜: 외부 releaser는 Skill 도구만 찾다가 실제로 존재하는 원문을 읽지 못했고, 두 문서의 0.0.1 초기 릴리스 주장과 현재 패키지 0.0.0을 동시에 받아 판정하지 못했다. 앱 저장소에 원문 접근 진입점과 검증된 버전 증거를 남기면 같은 탐색 실패와 근거 오독을 줄일 수 있다.
- 수용 기준: 1) 신규 AGENTS.md가 이번 세션에서 요구된 스킬에 한해 Skill 도구 우선, 도구 부재 시 실제 SKILL.md 읽기, 원문 누락 시 이름·경로와 미확인 상태 기록을 안내한다(성공한 Skill 호출로 가장하지 않는다). 2) README.md와 docs/01-시작하기.md의 단정적인 초기 릴리스 표를 docs/RELEASE.md 단일 근거 문서 링크로 대체하고, 정본에는 기존 주장 0.0.1/2025-01-01을 삭제하지 않고 '기존 문서 기재, 실제 릴리스 미확인'으로 보존한다; 현재 package/lock 세 값 0.0.0, 최초 lock의 버전 필드 부재, GitLab 브랜치 배포와 버전 릴리즈의 차이를 정확히 기록한다. 3) 아래 명령으로 실제 스킬 5개 파일 읽기와 Git/JSON 근거 재조회가 성공하며, 결과를 원장에 '수정 과제 — 릴리즈 입력 복구, 전체 릴리즈 미검증'으로 기록한다. 문구 문자열 검사를 전체 릴리즈 통과 증거로 삼지 않는다.
- 건드릴 파일: AGENTS.md(신규, 현재 없음): 필수 스킬의 원문 탐색 및 docs/RELEASE.md 안내; docs/RELEASE.md(신규): 근거·미확인 항목·읽기 전용 재현 명령의 정본; README.md:434 버전 기록과 docs/01-시작하기.md:355 버전 기록 — 정본 링크로 통일. 기존 함수 수정 없음.
- 검증 명령: 아래 실행 블록. 문서 변경만이면 npm 설치/빌드는 필수 아님; 앱 동작까지 건드리게 되면 과제를 넓히지 말고 계획을 수정한다. 기존 앱 명령은 npm ci, npm test, npm run build:dev이나 이번 정찰에서는 실행하지 않았다.
- 위험과 피할 것: .gitlab-ci.yml, 외부 aidev/bin/run.sh·release-prompt.md·gate.py·registry, headcount 원문, package.json/lock, 태그, auth 및 원격 서비스는 수정하지 않는다. 0.0.1을 가짜 기록이라고 단정하거나 0.0.0으로 과거 기록을 덮어쓰지 않는다. 버전 파일이 있으므로 이력 전무 skipped 조건을 만족한다고 선언하지 않는다. 임의 패치 증가·새 릴리즈 관례·검증 완화·원격 호출 금지. 회사 스킬 비활성 세션에서 강제 로드하지 않는다.
- 차선 후보: [수정 과제] 버전 근거 정본화만 수행 — 저장소 AGENTS.md에 외부 도구 탐색 안내를 두는 것이 부적절하다는 실제 저장소 규칙이 발견되면 두 문서와 정본만 수정하고 스킬 문제는 미해결로 남긴다. 외부 러너 수정 이관 기록만 쓰고 끝내는 이전 차선은 반복하지 않는다.

범위와 한계
이 과제는 자동 배정된 릴리즈 장애의 입력 오류를 고치는 저장소 내부 작업이다. 전체 릴리즈 성공을 45분 안에 보장할 근거는 없다. release-prompt.md 절차 2/5가 요구하는 과거 증가·커밋·태그 관례는 발견되지 않았고 새 관례 신설은 금지되어 있다. 구현자는 입력 복구 성공과 releaser 전체 성공을 분리해야 한다. 다음 버전 결정에 필요한 승인된 정책 또는 실제 배포 증거가 없으면 그 잔여 장애를 명시한다. 실패를 skipped/released로 바꾸는 과제는 아니다.

확인한 원인과 실패 단계
- HEAD dd65af7, git status 깨끗함. CLAUDE.md/AGENTS.md/.github/workflows/node_modules 없음. 실제 CI는 .gitlab-ci.yml의 main/develop 빌드 및 서버 디렉터리 복사이다; CI 실패 로그는 미확인.
- /mnt/c/Users/USER/projects/aidev/bin/run.sh의 agent_plugin_args/dept_note/run_agent/run_codex를 읽었다. dept_note는 Skill 호출만 안내하고 run_agent의 quota 폴백은 같은 프롬프트를 run_codex로 전달한다. run_codex는 원문 경로를 추가하지 않는다. 앱 코드의 빌드 실패로 오인하지 말 것.
- /mnt/c/Users/USER/projects/aidev/release-prompt.md 절차 1~6과 tests/test_sim.py의 run/HappyPath/Agents 및 tests/sim/run_sim.sh를 읽었다. 기존 sim은 에이전트 결과를 스텁으로 생성하므로 실제 모델의 원문 발견이나 버전 판단이 고쳐졌음을 증명하지 못한다. 동적 Codex quota 폴백은 이번 미실행.
- 회차 2026-09-20-211415-aiportal-front-improve/release.json과 state/aiportal-front.release.json은 같은 실패 내용이다. 확인한 최근 5개 회차별 결과는 skipped 4건/failed 1건이며, 독립된 동일 원인 실패 2회는 미확인이다.
- git 최초 커밋 ab700ba의 package.json은 0.0.0이나 package-lock.json의 root version 및 packages[""].version은 모두 없다. 현재는 세 곳 모두 0.0.0이다. 과거 프로필과 실패 사유의 'lock 최초부터 0.0.0'은 부정확하므로 복사하지 말 것.

구현 순서와 체크포인트
1. pending — 아래 명령을 그대로 실행해 현재 기준을 다시 확인한다. 예상과 다르면 brief에 발견을 기록하고 증거 문구를 수정한다. 사람 승인 체크포인트 없음; 자동 증거 확인 후 다음 단계.
2. pending — AGENTS.md에 짧은 탐색 안내를 추가한다. HEADCOUNT_DIR가 세션에 명시되면 그것을 사용하고, 아니면 이번 기계에서 확인한 /mnt/c/Users/USER/projects/headcount를 '이 환경의 후보 경로'로만 명시한다. <root>/plugins/<department>/skills/<skill>/SKILL.md 규칙으로 현재 프롬프트가 요구한 스킬만 읽게 한다. 원문을 복사하거나 누락 절차를 면제하지 않는다. 아래 실제 파일 열기로 경로를 검증한 뒤 진행.
3. pending — docs/RELEASE.md에 근거를 모으고 두 기존 표를 상대 링크로 바꾼다. 아래 Git/JSON 결과와 정본을 대조하고 링크 대상 파일을 실제 열어 읽는다. 미래 버전/과거 배포를 추측하지 않는다.
4. pending — git diff --check와 범위 검토 후 회차 원장(ledger-entry.md)에 수정 과제, 실행 결과, 전체 릴리즈 미검증 및 잔여 관례 결정을 구분해 남긴다. 실제 releaser 재실행 없이 '장애 해결 완료'로 기록하지 않는다.

검증 명령 (앱 저장소 루트, 원격/작업 트리 쓰기 없음)
```bash
git rev-parse --is-shallow-repository
git tag --sort=-creatordate
git log --oneline -60
git log --oneline -- package.json package-lock.json README.md docs/01-시작하기.md
python3 - <<'CHECK'
import json, subprocess
from pathlib import Path
p=json.loads(Path('package.json').read_text())
l=json.loads(Path('package-lock.json').read_text())
assert p['version']==l['version']==l['packages']['']['version']=='0.0.0'
first=subprocess.check_output(['git','rev-list','--max-parents=0','HEAD'],text=True).strip()
for name in ['package.json','package-lock.json']:
    old=json.loads(subprocess.check_output(['git','show',f'{first}:{name}'],text=True))
    print(first,name,old.get('version','ABSENT'),old.get('packages',{}).get('',{}).get('version','ABSENT'))
for name in ['README.md','docs/01-시작하기.md']:
    old=subprocess.check_output(['git','show',f'{first}:{name}'],text=True)
    print(name,[line for line in old.splitlines() if '| 0.0.1 |' in line])
root=Path('/mnt/c/Users/USER/projects/headcount/plugins')
for name in ['pmo:estimating-and-contingency','technology:implementation-planning','technology:solution-exploration','marketing:product-launch','technology:release-and-deployment']:
    dept,skill=name.split(':')
    path=root/dept/'skills'/skill/'SKILL.md'
    content=path.read_text()
    assert content.strip(),path
    print('READ OK',name,path)
CHECK
git diff --check
git diff --stat
```
정찰 검증: 위와 같은 실제 Git/JSON·스킬 파일 읽기를 수행했다. 최초 조사 코드가 과거 lock에 version이 있다고 가정하여 KeyError를 냈고, 필드 부재를 표시하는 코드로 재검증했다. bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh도 통과했다. 앱 test/build 및 전체 sim, 모델 재실행, 배포는 미실행.

대안 비교와 추정 근거 (회사 스킬 적용)
- 선택: 저장소 진입점+근거 정본. 외부 소유권을 넘지 않고 실패 입력을 수정하나 자동 릴리즈 관례 부재는 남는다.
- 외부 러너 경로 전달 구현: 근본적인 엔진 간 전달 해결이지만 이전 과제에서 외부 표면 때문에 차선 처리되어 이번 선택에서 제외.
- 버전 임의 증가/관례 신설: 근거 부재 및 release-prompt 금지와 충돌하여 제외.
- 아무 변경 없이 재이관: 이전 no-change를 반복하므로 제외.
- bottom-up 추정: 근거 재확인 5~8분 + 안내/정본/링크 수정 10~15분 + 읽기 검증/원장 5~7분 = 기본 20~30분. 알려진 경로·문서 차이 여유 5~10분 별도, 총 25~40분(M), 확신 중간(통계적 신뢰구간 아님). 관리 예비 0분; 다음 버전 정책 수립·실제 릴리즈·배포는 범위 밖. 유사 성공 작업의 시간 측정값이 없어 유사 추정 교차검증은 불가.
- 읽은 요청 스킬: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 technology/skills/{implementation-planning,solution-exploration}/SKILL.md. Skill 도구가 없어 파일 원문을 읽었으며 세 원문에 별도 Return contract 절은 없다. PMO references/sources.md도 확인했지만 외부 비용 기준 수치는 사용하지 않았다.

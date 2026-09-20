- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 승인 근거 확보 후에만 구현 (가치 5 / 위험 2 / 작업량 S)
- 왜: 서로 다른 두 회차에서 버전 증가·릴리즈 커밋·태그 관례를 확정하지 못해 릴리즈가 실패했고, 최신 회차에서는 회사 스킬 원문 접근 문제는 해소되어 있다. 실제 승인 정책 또는 과거 릴리즈 근거를 입력으로 확보해야 같은 실패를 끝낼 수 있으며, 문서 재정리나 임의 0.0.1 증가는 해결이 아니다.
- 수용 기준: 1) 다음 버전, 증가 단위, 릴리즈 커밋 양식, 태그 사용 여부·형식·종류, 노트·자산 방식 각각을 승인 원문 또는 실제 릴리즈 기록에 연결한다. 현재는 이 기준 미충족으로 구현 진입 차단이다. 2) 근거가 확보된 경우에만 그 근거대로 필요한 버전 필드를 일치시키고 기존 릴리즈 절차를 수행한다. 근거가 없으면 저장소 무변경과 차단 사유를 원장에 기록하며 완료·released·skipped로 바꾸지 않는다. 3) 실제 releaser가 만든 결과를 수정하지 않은 gate.py에 전달해 통과해야 전체 해결이다. 현 실패 JSON의 gate 재실행은 exit 1/ok=false/state=failed이고, gate 단위 테스트 통과는 전체 해결을 증명하지 않는다.
- 건드릴 파일: 현재 확정된 앱 수정 파일 없음. docs/RELEASE.md: '미확인 항목과 한계' — 승인 근거 확보 시에만 출처·결정 사항 반영(현 문서 재작성 반복 금지). package.json:version / package-lock.json:version 및 packages[""].version — 승인된 증가 값이 있을 때만 동시 갱신. 이번 회차 journal.md — 수정 과제, 차단 입력, 실제 검증 결과를 기록. 외부 aidev/bin/run.sh:release_project/run_agent/run_codex, release-prompt.md:절차 2·5, bin/gate.py:evaluate_release/cmd_release는 읽기 전용 조사 대상이며 수정 대상으로 지정하지 않는다.
- 검증 명령: 아래 재현 명령 및 조건부 완료 검증 참조.
- 위험과 피할 것: 새 관례 신설, 0.0.0을 버전 부재로 취급, private:true를 skipped 근거로 사용, failed JSON을 released로 수동 변경, 게이트·프롬프트·CI·자율화 정책 완화 금지. AGENTS.md/docs/RELEASE.md 입력 안내는 이미 e938e8e에 병합되어 반복 과제로 삼지 않는다. 외부 aidev 변경 과제를 앱 구현자에게 다시 배정하지 않는다. auth/session/migrations 및 .gitlab-ci.yml, 원격 게시·배포는 건드리지 않는다. 이전 변경의 롤백이 버전 정책 부재를 해결한다는 근거도 없다.
- 차선 후보: 없음 — 우선 배정은 릴리즈 실패 수정이다. 진입 조건이 성립하지 않으면 다른 앱 기능이나 문서 작업으로 바꾸지 말고 '수정 과제: 필수 정책 입력 부재로 차단, 코드 변경 없음'을 기록한다.

실행 가능성 판정: BLOCKED (정찰 완료, 수정·릴리즈 성공 아님)

지시의 '45분 안에 수정 후 동일 검증 통과'와 '새 관례 신설/워크플로 완화 금지'를 동시에 만족할 근거가 없다. 질문하지 말라는 지시에 따라 질문 없이 차단 상태를 남긴다. 구현자가 추측으로 채워야 하는 과제서가 되지 않도록 조건부 변경과 현재 실행 가능한 조사를 분리했다. S는 근거 점검·차단 기록의 작업량이며 정책 결정이나 전체 릴리즈 완료의 추정치가 아니다.

확인한 근거

- HEAD e938e8e, non-shallow=false가 아니라 `git rev-parse --is-shallow-repository`의 출력이 false이다. 전체 37커밋, 로컬 태그 0개, 현재 package/lock 세 필드 0.0.0. 패키지 이력은 ab700ba/a7fcddd/27e86f9이며 버전 증가 근거 없음.
- /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json: failed, 스킬 탐색과 버전 관례 장애. 당시 최초 lock 이력 설명은 잘못되었으므로 복제하지 않는다.
- /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json: failed, 실제 스킬 읽기 완료 후에도 버전 결정 불가. 최신 release-context.md는 GitHub Release 목록 공란과 워크플로 '(없음)'이다. 원격 현재 상태를 새 조회한 것은 아니다.
- 독립된 실패 2회가 이제 확인되었다. state/aiportal-front.release.json을 별도 실패로 세지 않는다.
- state/aiportal-front.policy.json은 autonomy=release/allow_merge_without_ci=true만 지정하며 버전·태그 관례를 제공하지 않는다. 릴리즈 자동화 권한이 다음 버전 정책은 아니다.
- .gitlab-ci.yml은 main/develop의 전용 Runner에서 빌드·복사 배포한다. 앱 빌드 실패 로그나 태그 기반 릴리즈 작업은 발견하지 못했다.
- release_project는 release-prompt.md를 run_agent에 전달하고 결과에 실제 gate.py release를 호출한다. evaluate_release는 status=failed를 그대로 차단한다. 실패를 일으키는 버전 결정은 모델 프롬프트 절차 2이고, 수정할 앱 버전 계산 함수·전용 테스트는 발견되지 않았다.
- tests/test_sim.py의 run/HappyPath/Agents와 tests/sim/run_sim.sh를 읽었다. sim은 v0.0.1 태그를 미리 만들고 에이전트 결과를 대역으로 생성하므로 이 저장소의 무관례 상태나 실제 모델 판단을 재현하지 않는다. sim 통과로 해결을 주장하지 말 것.

순서·증명·체크포인트

1. [정찰 완료] docs/RELEASE.md → 실제 Git/JSON → 서로 다른 실패 두 건 → 외부 프롬프트/러너/게이트를 대조한다. 아래 명령으로 증명. 사람 확인 없이 진행.
2. [차단] 기존에 승인된 원문 또는 실제 과거 릴리즈 기록에서 수용 기준 1의 결정을 회수한다. 이번 확인 경로에는 없다. 새로운 승인 자료가 없는 현 상태에서는 다음 단계로 진행하지 않는다. 재량으로 정책을 만들지 않는다. 이는 별도 승인 요청이 아니라 필수 입력 부재 판정이다.
3. [조건부, 미착수] 근거가 추가된 경우 계획을 그 출처에 맞춰 먼저 갱신한 뒤 docs/RELEASE.md 및 필요한 세 버전 필드만 반영한다. 실행 코드 변경이 없다면 문구 검사 테스트를 추가하지 않는다. 기존 버전 검사·빌드 요구를 근거로 명령을 확정하고 실행한다. 근거 없는 커밋/태그 형식은 여전히 금지.
4. [조건부, 미착수] 실제 릴리즈 실행 결과와 원래 gate를 통해 동일 경로 통과를 확인한다. run.sh --release-only는 원격 push까지 하는 명령이므로 이 정찰/앱 수정 검증용으로 실행하지 않는다. 전용 릴리즈 역할의 결과 없이는 '전체 릴리즈 미검증'을 유지한다.

현재 실행한 재현 명령 (앱 루트 기준)

```bash
git rev-parse --is-shallow-repository
git tag --sort=-creatordate
git log -30 --oneline
git log --oneline -- package.json package-lock.json
python3 - <<'CHECK'
import json, subprocess
from pathlib import Path
p=json.loads(Path('package.json').read_text())
l=json.loads(Path('package-lock.json').read_text())
print('CURRENT',p['version'],l['version'],l['packages']['']['version'])
assert p['version']==l['version']==l['packages']['']['version']=='0.0.0'
first=subprocess.check_output(['git','rev-list','--max-parents=0','HEAD'],text=True).strip()
old=json.loads(subprocess.check_output(['git','show',first+':package-lock.json'],text=True))
print('INITIAL_LOCK',old.get('version','ABSENT'),old.get('packages',{}).get('',{}).get('version','ABSENT'))
CHECK
bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh
git diff --check
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-015400-aiportal-front-improve/assets PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py
PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
```

결과: Git/JSON·bash -n·diff 검사 정상, gate 테스트 27건 통과(테스트 파일의 미닫힌 파일 ResourceWarning 있음), 마지막 실제 gate 명령은 예상대로 exit 1. 과거 실패 JSON을 고치지 않는다. 이는 소비 단계 재현이며 모델 버전 결정 재실행은 아니다. 앱 node_modules 없음; npm ci/test/build와 외부 모델·전체 sim·배포 미실행. 앱 검증이 필요해지는 구현에서는 npm ci → npm test → npm run build:dev가 등록된 명령이며 npm run build는 없다.

대안 비교와 추정 (회사 스킬 적용 기록)

- 추천: 승인된 입력을 회수한 뒤에만 진행. 현재 차단 원인을 직접 다루고 새 정책을 만들지 않는다. 핵심 가정은 그런 입력이 따로 존재한다는 것이며 현재 미확인이다.
- 문서 안내 반복: 이미 완료되었고 직후 같은 버전 실패가 확인되어 기각.
- 임의 초기 버전/태그 또는 skipped 조건 확장: 명시적 금지와 충돌하여 기각.
- 외부 러너 스킬 폴백 수정: 이전 no-change 반복이고 현재 실패에서는 원문 접근이 성공했으므로 이번 해법으로 기각.
- Bottom-up: 입력 대조 5~10분 + 결정 출처 정리 5~10분 + 재현/원장 5분 = 15~25분, 알려진 조사 변동 여유 0~5분을 별도로 두어 15~30분. 정찰자의 낮은 확신 범위이며 통계적 신뢰구간·릴리즈 완료 보장이 아니다. 승인 입력 확보 대기와 외부 릴리즈는 제외, management reserve는 배정하지 않았다. 이전 문서 수정이 병합 후 재실패한 사례로 교차 점검했으며 소요시간 데이터가 없어 analogous 수치 추정은 하지 않는다. 45분 내 전체 해결 가능하다는 추정은 거부한다.
- Skill callable 도구 없음. /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md 실제 원문을 읽고 적용했다. pmo의 references/sources.md도 읽었다. 별도 Return contract/JSON 스키마는 세 원문에 없으며 사용자 과제서 형식을 따른다. 파일 읽기를 Skill 호출 성공으로 기록하지 않는다. 외부 비용·편익/정량 contingency 표준을 인용한 추정은 아니다.

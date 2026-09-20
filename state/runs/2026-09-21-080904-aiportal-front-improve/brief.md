- 과제: [수정 과제] 릴리즈 버전 결정 실패 — 정책 근거 결손으로 실행 가능한 수정 배정 없음 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 릴리즈 절차는 기존 증가·커밋·태그 관례를 요구하지만 현재 로컬 Git/JSON과 실패 기록에는 다음 버전을 확정할 출처가 없다. 실제 승인 정책 또는 과거 릴리즈 근거가 확보되어야 기존 검증을 유지하면서 이 실패를 해결할 수 있다.
- 수용 기준: 1) 다음 버전·증가 단위·릴리즈 커밋·태그 사용 여부/형식/종류·노트·자산·검증 방식에 대한 실제 출처를 확보하고 미확인과 구분한다. 2) 그 출처에 맞는 구체적인 변경 파일과 실패 검증을 재계획한 뒤에만 구현하며, package/lock의 세 버전 소비 경로가 일치한다. 3) 원래 릴리즈 에이전트의 버전 결정부터 실제 산출물 검증까지 로컬에서 통과하고 증거를 남긴다. 과거 failed JSON 재평가나 게이트 단위 테스트 통과만으로 3을 충족하지 않는다. 현재 1 미충족, 2·3 미착수.
- 건드릴 파일: 지금은 없음. 읽은 docs/RELEASE.md는 근거 정본이며 정책 생성 대상이 아니다. package.json:version 및 package-lock.json:version/packages[""].version은 향후 출처 확보 시에만 변경 후보. .gitlab-ci.yml의 ofc/core/int/dev 빌드·복사 job은 이번 버전 판단 실패의 수정 대상이 아니다. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_context/release_project, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release를 읽었으며 앱 구현 범위 밖이다.
- 검증 명령: 현 상태 기록 확인은 저장소 루트에서 `git diff --check`, `git status --short`, `git rev-parse --is-shallow-repository`, `git tag --sort=-creatordate`, `git log -30 --oneline`. JSON 확인 명령은 아래에 있다. 앱 표준 명령은 `npm ci`, `npm test`, `npm run build:dev`이나 이번 정책 결손을 검증하지 않으며 현재 node_modules가 없어 실행하지 않았다. 출처 없는 현재 상태에서는 동일 릴리즈를 통과시키는 유효한 로컬 명령을 제시할 수 없다.
- 위험과 피할 것: 새 증가·태그 관례/임의 0.0.1·skipped·released 만들기, gate/워크플로 완화, docs/RELEASE.md 재정리 PR, 스킬 경로 안내 반복, auth/storage/배포 수정 금지. 외부 러너의 GitHub 조회 오류 은폐를 확인했지만 두 실패와 인과는 미확인이므로 이 앱 과제로 재배정하지 않는다. 실패 JSON과 외부 러너도 수정하지 않는다.
- 차선 후보: 없음 — 우선 과제가 고정되어 useAppList 캐시 방어나 새 테스트·문서 후보로 바꿀 수 없다. 새 출처 없이 같은 BLOCKED 과제를 구현 가능한 수정인 것처럼 재실행하지 않는다.

현재 판정과 인계

이 문서는 완료한 수정이나 새 해결책이 아니라 고정 과제의 실행 불가 기록이다. 사용자 절대 규칙에 따라 정찰은 코드를 변경하지 않았다. 요청된 수정 후 동일 검증 통과는 미완료다. 구현자는 새 근거가 전달되지 않았다면 코드·문서·테스트를 임의로 만들거나 동일 조사를 반복하지 말고 이 상태를 인계한다. 승인 질문을 만들지 않는다.

확인 근거

- HEAD e938e8e, non-shallow, 총 37커밋, 로컬 태그 0개, 현재 JSON 세 값 0.0.0. 최근 git log -30과 패키지 변경 커밋 목록을 확인했다. 원격 조회는 이번에 반복하지 않았으며 원격 태그·Release 유무는 미확인이다.
- 2026-09-20-211415 및 2026-09-21-001359 회차 release.json을 읽었다. 후자는 스킬 원문을 찾은 뒤에도 버전 근거 부족으로 실패했다. 전자의 최초 lock 설명은 정본과 후자에 의해 정정되어 있다.
- .github/workflows, CLAUDE.md는 없고 .gitlab-ci.yml은 전용 Runner 배포다. GitHub workflow run ID와 두 번 재실행 로그는 미확인이다. 두 failed JSON을 GitHub Actions 두 실패로 동일시하지 않는다.
- release_project는 모델 실행 후 JSON을 gate release에 전달한다. evaluate_release는 released가 아니면 차단한다. tests/test_gate.py의 Release는 상태·태그·자산 경계를 검증하며 버전 선택 모델을 실행하지 않는다. run.sh 전체 실행은 원격 쓰기/배포를 포함하므로 로컬 재현 명령으로 쓰지 않는다.

단계와 체크포인트

1. [BLOCKED] 새 출처 입력 확인. 산출물: 출처와 다음 버전·커밋·태그·검증의 대응표. 증명: 실제 원문/태그/커밋과 비교. 새 입력이 없으므로 현재 실행할 작업 없음; 시간 경과나 반복 조사를 입력으로 간주하지 않는다.
2. [미착수] 출처가 생긴 경우 별도 계획 갱신. 파일·변경·정확한 로컬 재현 명령을 먼저 확정한다. 체크포인트: 원래 수용 기준과 범위가 유지되는지 확인; 정책을 새로 정해야 하는 경우 자동 구현 불가.
3. [미착수] 확정된 최소 변경과 원래 검증 실행. 실제 모델/산출물 배선을 통과한 결과만 통과 근거로 사용. 공개·배포 성공은 별도 증거 없이 주장하지 않는다.

선택지 비교 및 산정

- 실제 과거 릴리즈/승인 출처 복구: 요구를 만족할 수 있는 최소 선택지이나 현재 입력 없음. 출처가 생기면 재산정한다.
- 초기 릴리즈 정책을 새로 만드는 방안: 현재 금지된 범위이며 선택 불가.
- 외부 조회 오류 구분 개선: 별도 소유 프로젝트에서 고려할 수 있으나 현재 실패 인과 미확인, 앱 수정 대체 불가.
- 무변경 인계: 현재 유일하게 범위를 지키는 행동이며 해결·통과가 아니다. 핵심 가정은 아직 읽지 못한 승인된 관례가 존재할 수 있다는 것뿐이다.
- S는 기록 인계에 한정한다. bottom-up 작업 예상은 근거 정리 5~10분 + 인계·기록 검증 5~10분, 합계 10~20분(경험적 예상, 통계적 신뢰수준 미산정). 알려진 파일 접근/형식 문제 여유 0~5분은 별도이며 관리 예비는 배정하지 않았다. 정책 확보 대기와 실제 릴리즈 구현은 범위·시간 미확정이므로 45분 완료를 약속하지 않는다. 과거 회차는 no-change여서 성공 구현과의 유사 추정 자료로 사용할 수 없다.

스킬 적용 기록

호출 가능한 Skill 도구 없음. 다음 원문을 실제 읽고 적용했다(성공한 Skill 호출 아님, 세 원문에 고정 반환 스키마 없음).
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md: 분해·범위·가정·예비를 분리했다. 외부 비용/편익 수치나 정량 신뢰수준은 사용하지 않았다.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md: 변경·증명·체크포인트·미착수 상태 명시.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md: 목적·범위·대안·핵심 가정 비교.

기록용 JSON 재조회 명령(릴리즈 통과 검증 아님):
```bash
python3 - <<'CHECK'
import json
from pathlib import Path
p=json.loads(Path('package.json').read_text())
l=json.loads(Path('package-lock.json').read_text())
print(p['version'], l['version'], l['packages']['']['version'])
CHECK
```

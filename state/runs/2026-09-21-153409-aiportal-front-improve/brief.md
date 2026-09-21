- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 수정 가능한 근거 미확보 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 release-prompt.md 2·3단계는 기존 증가·커밋·태그 관례를 요구하지만, 현재 Git/JSON과 docs/RELEASE.md에는 다음 값을 결정할 출처가 없다. 해당 입력이 복구되어야 실제 릴리즈 검증으로 진행할 수 있으며 게이트 완화나 반복 무변경 조사를 수정 성과로 세면 안 된다.
- 수용 기준: 1) 다음 버전·증가 단위·커밋 메시지·태그 형식/주석 여부·노트·자산·검증 절차를 결정할 실제 과거 근거 또는 승인된 정책의 출처와 내용을 확보한다. 현재 미충족. 2) 그 출처로 정당화되는 최소 변경만 적용하며 package.json 및 lock의 루트/루트 패키지 버전이 일치한다. 현재 미착수. 3) 기존 실패 입력은 계속 거부하고, 근거에 따라 실제 생성한 새 결과를 동일 release 검증으로 통과시킨다. 게이트 단위 테스트나 실패 JSON 재검사만으로는 수정 후 통과를 증명하지 않는다. 현재 미완료.
- 건드릴 파일: 현재 확정 가능한 앱 수정 파일 없음. docs/RELEASE.md는 읽기 기준이며 정책으로 바꾸지 않는다. package.json:version, package-lock.json:version 및 packages[""].version은 수용 기준 1 충족 후에만 검토한다. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:2·3·5단계, bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 실제 읽은 진단 경로이며 이번 저장소의 수정 대상이 아니다.
- 검증 명령: 아래 실제 실행 명령 및 결과 참조. npm test와 npm run build:dev는 package.json에 있지만 node_modules가 없어 이번에 실행하지 않았다. 원래 버전 결정 검사의 대체로 사용하지 않는다.
- 위험과 피할 것: 0.0.1 임의 증가, 새 태그/릴리즈 커밋 양식, private 또는 0.0.0을 근거로 버전 파일을 없다고 취급, failed→skipped/released 변경, 실패 JSON 수정, gate/워크플로 완화 금지. auth/storage/router 및 .gitlab-ci.yml 전용 Runner reset/cp 배포는 건드리지 않는다. 외부 러너 수정도 이번 범위에 없다.
- 차선 후보: 없음 — 자동 배정된 릴리즈 실패를 무관한 앱·문서 개선으로 대체할 수 없다. 신규 출처가 없으면 기존 미해결 건을 유지하며 같은 BLOCKED 조사나 문서 재작성 과제를 구현 완료로 재배정하지 않는다.

현재 판정과 근거

이 과제서는 실행 가능한 수정 배정이 성립하지 않았다는 결과를 숨기지 않는다. 사용자 요구의 ‘관례 신설 금지·워크플로 완화 금지’와 현재 입력을 함께 만족하면서 수정 후 통과하는 방법은 확인하지 못했다. 반복된 no-change 접근이 성공하지 않았으므로 구현자는 이 조사를 다시 수행하거나 진단 문서를 다시 커밋하지 않는다. 신규 입력이 실제 들어온 경우에만 아래 재개 단계로 진행한다.

- 기준 HEAD e938e8e, non-shallow, 전체 37커밋, 로컬 태그 0개. git log -30과 패키지 변경 이력을 읽었다. 현재 세 버전 필드 모두 0.0.0; 최초 package는 0.0.0, 최초 lock 두 필드는 부재다.
- CLAUDE.md와 .github/workflows 없음. .gitlab-ci.yml은 main/develop 브랜치 빌드 및 서버 복사이며 이번 버전 결정 실패 단계가 아니다.
- 실제 실패 자료: /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json. 전자는 스킬 접근 실패도 포함하고, 후자는 스킬 원문 접근 뒤 정책 근거 부족으로 중단했다. 두 자료를 독립된 동일 GitHub workflow 실패 2회로 단정하지 않는다. GitHub run ID/실패 step은 미확인.
- release_project는 run_agent 결과를 gate로 읽고 ok=false이면 push 전에 종료한다. evaluate_release는 status가 released가 아니면 거부한다. 이는 입력된 실패를 제대로 거부하는 동작이다.
- release_context의 gh 오류→‘없음’ 처리 및 tag fetch 오류 후 계속은 확인했다. 이번 원인이라는 증거는 없고, 외부 소유 후보를 앱 수정으로 넘기지 않는다. 빈 release-context.md로 원격 이력 부재를 단정하지 않는다. 원격 재조회는 미실행.

실행 검증 (저장소 루트에서 실행, 로그: verification.json)

```bash
PYTHONDONTWRITEBYTECODE=1 TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-153409-aiportal-front-improve/assets python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
git diff --check
git status --short
```

첫 명령: 5 tests OK, exit 0; 기존 테스트의 파일 미종료 ResourceWarning 있음. 둘째: exit 1, ok=false/state=failed. diff 검사 exit 0, status 출력 없음. 테스트 임시 자산은 이번 run 안에만 생성하고 삭제했으며 실제 릴리즈 자산이 아니다. 이는 거부 동작 재현이지 실제 모델의 버전 판단 재실행 또는 수정 후 Green이 아니다.

신규 입력이 도착한 경우의 재개 순서

1. 출처 대조: 전달된 정책/과거 릴리즈를 정본 및 실제 Git/JSON과 맞춰 결정표를 작성한다. 근거가 충돌하면 계획을 수정하고 중단한다. 검증: 위 Git/JSON 값 재조회와 출처 원문 비교. 체크포인트: 필수 입력 충족 여부; 사람이 없는 이번 세션에서 승인 정책을 스스로 발명하지 않는다.
2. 최소 수정: 출처가 지정한 버전 필드·노트·태그/커밋 준비만 수행한다. 변경 파일과 명령은 출처가 생긴 뒤 계획에 확정한다. 검증: 세 버전 필드 일치 및 출처가 요구한 실제 빌드/검사. 체크포인트: 모든 검사 통과 후 다음 단계. 이 단계는 현재 배정되지 않았다.
3. 동일 검증: 실제 releaser의 새 release.json을 동일 gate.py release <새 결과 절대경로> --out-dir <그 회차 경로>로 검사한다. 실패 원본은 보존하고 결과 조작 없이 exit 0/ok=true와 실제 검증 로그를 함께 남긴다. 외부 releaser 실행·게시 권한은 이 정찰 과제서가 부여하지 않는다. 원장에는 ‘수정 과제’로 기록하고 기준 1~3 충족 전 done으로 바꾸지 않는다.

대안 비교 및 추정 근거

- 기존 관례/승인 정책 입력 복구: 가장 작은 해법. 근거가 실제 존재하고 제공될 수 있어야 한다. 이번 선택이 의존하는 핵심 가정이며 현재 미확인이다.
- 외부 실행기에 정책 입력/진단 체계 추가: 다중 저장소에 쓸 수 있지만 외부 소유·범위 확장이고 이번 실패를 해결한다는 인과가 없다. 이번 선택에서 제외.
- 새 구성 없이 현상 유지: 거짓 릴리즈를 막지만 실패는 해결하지 못한다. 현재의 불가피한 결과이며 새로운 개선 성과가 아니다.
- 새 관례 발명·게이트 완화는 허용 가능한 대안이 아니므로 후보로 채점하지 않았다.
- Bottom-up 조건부 작업량: 출처 확인 5~10분 + 최소 반영 10~15분 + 동일 검증/기록 10~15분 = 25~40분, 알려진 변동(출처 정합성 확인) contingency 최대 5분 별도. 관리 예비시간은 배정하지 않았으며 범위 확대 시 재추정한다. 신뢰도 낮음, 실측 통계나 45분 완료 보장이 아니다. 정책 확보 대기·외부 배포·대규모 도구 변경은 제외하므로 현재 전체 해결 시간은 추정 불가다. 유사 회차는 모두 no-change여서 성공 구현의 유추 추정 근거로 사용할 수 없다.

스킬 적용 기록

호출 가능한 Skill 도구가 없어 아래 원문을 파일로 읽었다. 성공한 Skill 호출이 아니며 세 원문에 별도 고정 반환 스키마는 없다. estimating의 분해/가정/범위/예비시간, planning의 변경/증명/체크포인트, exploration의 대안/핵심 가정을 위에 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
외부 비용·효익 통계나 표준 준수 주장은 하지 않는다. 추정은 이번 파일/절차 분해에 대한 조건부 판단이다.

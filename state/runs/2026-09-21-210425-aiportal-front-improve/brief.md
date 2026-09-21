- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 필수 근거 결손으로 실행 배정 불가 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 releaser는 다음 버전·커밋·태그의 기존 관례를 요구하지만 현재 확인 가능한 Git/JSON과 정본에 그 근거가 없다. 유효한 근거를 입력해야 동일 릴리즈 절차를 유지하면서 재개할 수 있으며, 반복 무변경 조사나 gate 통과 조작은 해결이 아니다.
- 수용 기준: 1) 승인된 정책 또는 실제 과거 릴리즈의 출처·적용 대상·증가 단위·커밋/태그 형식·노트 위치·자산 및 검증 절차가 확인된다. 2) 그 출처에 따라 필요한 파일만 변경하고 package.json 및 package-lock.json의 두 버전 필드가 일치하며 임의 관례나 판정 완화가 없다. 3) 원래 실패한 버전 결정부터 로컬 릴리즈 검증까지 실제로 통과한 로그와 산출물이 존재한다. gate 단위 테스트, 스텁 released JSON, 문구 검사는 이 기준을 대신하지 않는다.
- 건드릴 파일: 현재 확정된 저장소 수정 파일 없음. docs/RELEASE.md: 릴리즈 판단 근거 — 신규 출처가 실제 확보된 경우에만 출처 및 적용 내용을 기록; package.json:version 및 package-lock.json:version/packages[""].version — 증가 정책 확인 이후에만 수정 가능. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 읽은 진단 대상이며 이번 저장소 수정 대상이 아니다.
- 검증 명령: 아래 실행 기록과 재개 절차 참조. 이 저장소에 전용 릴리즈 검사 명령은 발견되지 않았다. 앱 명령은 npm ci, npm test, npm run build:dev가 존재하나 버전 결정 실패의 재현 명령은 아니다.
- 위험과 피할 것: 새 버전·태그 관례 발명, 0.0.0을 0.0.1로 임의 증가, private=true를 skipped 근거로 삼기, 실패 JSON 덮어쓰기, gate/워크플로 완화, 외부 실행기 수정, auth/storage/router 변경, 원격 push/배포를 금지한다. 최초 lock 버전은 부재였으며 최초부터 0.0.0이라는 옛 실패 사유를 재사용하지 않는다.
- 차선 후보: 없음 — 우선 과제가 고정되어 무관 개선으로 교체할 수 없다. 외부 release_context의 오류/빈 목록 구분은 별도 소유 과제이고 현재 실패를 해결한다는 인과 증거가 없다.

## 실행 배정 판정과 재개 계획
현재 수용 기준 1 미충족, 2·3 미착수. 이번 정찰은 코드 수정 금지 역할이므로 수정/Green을 주장하지 않는다. 최근 동일 BLOCKED 과제의 반복 no-change를 확인했으며 이를 새 구현 과제로 재배정하지 않는다. 새 출처 없이 구현자가 같은 조사·gate·앱 검사를 반복하는 것도 요구하지 않는다. 원장의 고정 수정 과제는 pending으로 남긴다.

1. [미충족] 새 출처가 인계됐을 때만 docs/RELEASE.md와 해당 출처를 대조한다. 증명: 출처의 파일/커밋/태그/승인 기록과 각 정책 필드를 과제서에 대응시킨다. 이번 인계에는 새 출처가 없다. 자동 체크포인트: 필드가 미확인인 채 다음 단계로 진행하지 않는다; 사람이 없는 이번 회차에 질문하지 않는다.
2. [미착수] 근거가 확보되고 기존 범위에 맞을 때만 버전/정본/출처가 지정한 노트를 최소 수정한다. 증명: Git diff와 JSON 세 필드 일치 검사, 출처가 요구하는 실제 로컬 검사. 노트 경로·태그 이름·증가값은 현재 미확인이므로 미리 만들지 않는다. 새 범위가 필요하면 계획을 먼저 갱신한다.
3. [미착수] 원래 releaser의 버전 결정 및 검증을 실제 입력으로 수행한 뒤 결과를 gate로 검사한다. 전체 bin/run.sh를 source/실행하면 원격 전송·worktree reset이 가능하므로 로컬 테스트 명령으로 사용하지 않는다. 현재는 안전한 독립 releaser 재현 명령도 확정할 근거가 없다. 출처가 지정한 검증과 실행 진입점을 확정하기 전 성공 판정을 하지 않는다.

## 직접 확인한 근거
- 기준 HEAD e938e8e, non-shallow, 전체 37커밋, 로컬 태그 0개. git log -30 및 package/lock 이력을 읽었다. 현재 세 버전 필드 0.0.0; 최초 package는 0.0.0, 최초 lock 두 필드는 없음.
- AGENTS.md, docs/RELEASE.md, README.md, docs/07 로드맵, docs/09 테스트 가이드, docs/06·08 관련 설정/빌드 절, package.json, vite.config.js, vitest.config.js, .gitlab-ci.yml을 확인했다. CLAUDE.md, .github/workflows, 전용 릴리즈 스크립트는 발견되지 않았다. src/docs TODO/FIXME 검색 일치 없음.
- .gitlab-ci.yml은 main/develop 전용 Runner에서 fetch/reset·환경별 build·cp를 수행한다. 외부 release_project는 run_agent 결과를 gate에 넣고 ok=false이면 push 전에 종료한다.
- 2026-09-20-211415와 2026-09-21-001359의 release.json은 서로 다른 파일이며 둘 다 failed다. 전자는 스킬 접근 실패도 포함하고 후자는 원문 접근 이후 버전 근거 부족이다. GitHub Actions의 동일 실패 단계 재실행 2회 및 run ID는 미확인이다.
- release_context는 gh 실패를 '(없음)'으로 출력하며 tag fetch도 실패 후 계속한다. 따라서 제공된 빈 목록만으로 원격 이력 부재를 확정하지 않는다. 원격 인증 실패는 이전 정찰 관측이고 이번에는 원격 조회를 반복하지 않았다.

## 이번 로컬 실행 기록
저장소 루트에서 실행했다. 테스트 임시 파일은 지정 회차 폴더 안에만 생성했다.
```bash
PYTHONDONTWRITEBYTECODE=1 TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-210425-aiportal-front-improve python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
```
결과: Release 5개 OK, exit 0 (기존 테스트의 unclosed file ResourceWarning). 실패 JSON 두 개는 각각 exit 1, state=failed, ok=false. 이는 실패 차단 재현이며 수정 후 통과가 아니다. npm 설치/test/build, 실제 releaser, sim, 외부 사용자 검증, 전용 Runner 배포는 실행하지 않았다.

## 대안 비교·견적 근거
최소 방안인 유효한 기존 정책/실제 이력 입력 복구를 고정 과제로 유지한다. 새 정책 도입은 지시 위반으로 제외한다. 외부 실행기 오류 구분은 향후 별도 과제로 가능하지만 소유 범위와 인과가 달라 제외한다. 현 상태 보존은 성공이 아니라 미완료 관리다. 가장 중요한 가정은 유효한 출처가 제공될 수 있다는 점이며 현재 미충족이다.
S는 입력이 제공된 이후의 소규모 정합성 작업만 뜻한다. 상향식 잠정 추정: 출처 대조 5–10분, 최소 수정 5–10분, 로컬 검사/기록 10–15분=20–35분, 알려진 출력 정합성 재검사 여유 5–10분을 별도 추가하여 25–45분. 실측 이력이 없어 신뢰 낮은 계획 범위이며 확률/납기 보장이 아니다. 출처 확보 대기·앱 설치·대형 빌드·외부 배포는 제외하고 필요해지면 재추정한다. 관리 예비비는 배정하지 않았다. 과거 회차는 구현 없이 끝나 유추 견적의 비교 표본으로 사용할 수 없다.

## 스킬 적용 기록
호출 가능한 Skill 도구 없음. 아래 원문을 파일로 읽었으며 성공한 Skill 호출로 간주하지 않는다. 별도 고정 반환 스키마는 없고, 위 대안 비교·근거/범위 견적·증명/체크포인트 계획에 절차를 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
- estimating-and-contingency/references/sources.md도 읽었다. 외부 비용모형/통계적 예비율을 적용하지 않았으며 위 시간은 이번 작업 분해에 따른 잠정 추정이다.

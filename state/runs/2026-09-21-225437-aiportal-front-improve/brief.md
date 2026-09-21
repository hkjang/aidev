- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 근거 결손으로 실행 배정 불가 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 release-prompt.md 2·3단계가 요구하는 기존 버전 증가·커밋·태그 관례를 현재 저장소 근거로 결정할 수 없어 릴리즈가 중단된다. 유효한 기존 관례 또는 승인된 정책 출처가 확보되면 임의 관례나 게이트 완화 없이 같은 릴리즈 검증을 진행할 수 있다.
- 수용 기준: 1) 적용 프로젝트·기준 SHA·출처와 증가 단위, 커밋/태그 형식, 노트/자산 및 검증 절차를 명시한 실제 이력 또는 승인 정책을 확보한다(현재 미충족). 2) 그 근거에 맞는 최소 수정과 검증 절차를 확정하고 버전 파일 세 값의 일치, 관례 준수, 필요한 산출물을 확인한다(현재 미착수). 3) 실제 releaser의 동일 판단 경로 및 기존 gate가 수정 후 통과하고 원장에 수정 전 실패/변경/수정 후 결과를 남긴다; failed JSON 변경·스텁 released 결과·회귀 테스트 통과만으로 성공 판정하지 않는다(현재 미완료).
- 건드릴 파일: 현재 변경 승인 가능한 구현 파일 없음. docs/RELEASE.md는 근거 확보 후에만 출처 추가 후보; package.json:version 및 package-lock.json:version/packages[""].version은 그 출처가 요구할 때만 대상 확정. 읽은 외부 /mnt/c/Users/USER/projects/aidev/bin/run.sh:release_context/release_project, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 진단 근거이며 이 저장소 수정 범위 밖이다. .gitlab-ci.yml은 브랜치 배포이며 이번 실패 수정 대상으로 지정하지 않는다.
- 검증 명령: 아래 실행 기록과 재개 절차 참조. 실제 릴리즈 판단을 단독으로 재실행하는 로컬 안전 명령은 미확인이다; run.sh 전체 실행을 검증 명령으로 제시하지 않는다(원격 쓰기 포함).
- 위험과 피할 것: 같은 BLOCKED 조사를 구현 과제로 다시 실행하지 않는다. 현재 인계에는 신규 정책 출처가 없으므로 구현자는 조사·앱 검사·게이트를 반복하지 말고 pending/미완료로 반환한다. 임의 0.0.1 증가, 최초 릴리즈 규칙 창작, failed→skipped/released 위장, 실패 JSON 수정, gate/워크플로 완화, auth/session/router 변경 및 외부 러너 수정 금지. 원격 인증 실패·빈 context를 이력 부재로 단정하지 않는다.
- 차선 후보: 없음 — 고정 릴리즈 수정 과제의 근거 확보가 불가능하면 무관한 앱/DX 개선으로 대체하지 않는다.

이번 판정과 근거

기존 미해결 건을 유지하며 새로운 구현 배정은 하지 않는다. 사용자가 금지한 반복 no-change 접근을 해결책으로 재선정하지 않았다. 정찰은 완료했으나 요청된 수정 및 동일 검증 통과는 미완료다.

HEAD e938e8e, non-shallow 37커밋, 로컬 태그 0개. docs/RELEASE.md를 먼저 읽고 git log -30 및 package/lock 이력을 재조회했다. 현재 세 버전 필드는 모두 0.0.0, 최초 package는 0.0.0이며 최초 lock 두 필드는 부재다. CLAUDE.md와 .github/workflows는 없고 전용 릴리즈 스크립트는 확인하지 못했다. README, docs/07 로드맵, docs/06·08·09 관련 절, package.json, vite/vitest 설정, .gitlab-ci.yml, src/docs TODO/FIXME(일치 없음)를 확인했다.

외부 release_project는 run_agent → release.json → cmd_release/evaluate_release 순서이며 ok=false에서 push 전에 종료한다. 09-20-211415 실패는 스킬 탐색 문제와 관례 결손, 09-21-001359 실패는 스킬 파일 접근 후 관례 결손이다. 독립 GitHub Actions 실행 ID/실패 단계는 미확인이다. 따라서 '같은 워크플로 단계 두 번 실패'나 앱 CI 결함으로 확정하지 않는다. release_context의 gh 조회 오류를 '(없음)'으로 표시하는 코드와 태그 fetch 오류 후 계속하는 코드는 존재하지만 이번 실패와 인과는 미확인이고 외부 소유다.

실행한 검증(저장소 루트에서 실행 가능)

```bash
python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
git diff --check
git status --short
```

첫 명령은 TMPDIR를 이번 회차 폴더 아래 verification-tmp로 지정하여 실행하고 임시 파일을 제거했다. 5 tests OK / exit 0이며 미닫힌 파일 ResourceWarning이 있다. 두 gate 명령은 각각 exit 1, state=failed, ok=false다. 실패 차단 재현이며 수정 후 통과 증거가 아니다. npm ci/test/build, 실제 releaser, 원격 조회·배포는 미실행. node_modules가 없다. 향후 앱 검사가 필요한 변경일 때만 npm ci → npm test → npm run build:dev를 사용하며 전용 Runner 검증과 구별한다.

재개 계획과 체크포인트

1. 신규 근거가 제공된 경우에만 적용 대상과 출처의 유효성을 확인하고 본 과제서의 파일·명령을 갱신한다. 증명: 출처와 실제 Git/JSON을 대조한 근거표. 체크포인트: 출처가 없으면 중단, 추가 사용자 질문은 하지 않는다. 현재 상태: 미충족.
2. 근거가 허용하는 최소 변경만 구현한다. 증명: 세 버전 필드 및 실제 커밋/태그 형식 대조와 그 근거에서 요구한 검사. 아직 형식과 전용 명령이 없으므로 추측하지 않는다. 체크포인트: 근거와 충돌하면 과제서를 수정하기 전 다음 단계 금지. 현재 상태: 미착수.
3. 동일 releaser 판단과 gate 결과를 확보한 후 기존 실패를 보존하고 원장에 전후 결과를 기록한다. 체크포인트: 실제 판단 재실행과 필요한 앱 검증 없이는 완료 불가; 게시·태그 생성은 정찰 범위 밖. 현재 상태: 미착수.

대안 비교와 추정

- 기존 관례/승인 정책 복구: 가장 작은 정당한 해법이나 필요한 입력이 현재 없다. 고정 과제로 유지한다.
- 외부 컨텍스트 수집기 정교화: 향후 진단에는 유용하지만 외부 소유이며 정책 부재를 해결한다는 인과가 없다. 이번 배정 제외.
- 현 상태 유지: 규칙 위반을 막지만 릴리즈를 완료하지 못한다. 현재 실행 판정이며 개선 성과로 세지 않는다.
- 버전 임의 증가/게이트 완화는 사용자 금지 조건 때문에 유효 대안이 아니다.

추정은 입력 확보 이후만 해당: 출처 대조 5–10분, 최소 반영 5–10분, 검증·기록 10–15분의 bottom-up 20–35분 + 알려진 변동 예비 0–10분 = 20–45분(S). 낮은 확신의 계획 범위이며 통계적 신뢰수준은 미산정. 유사 회차는 모두 no-change라 성공 구현의 유사 추정값을 만들 수 없다. 정책 확보 시간·외부 배포·대규모 빌드는 제외하므로 총 해결시간은 산정 불가; 관리 예비는 별도 미배정. 입력이 달라지면 재추정한다.

스킬 적용 기록

호출 가능한 Skill 도구 없음. 아래 원문을 실제 읽어 적용했으며 성공한 Skill 호출로 기록하지 않는다. 세 원문에 별도 고정 반환 스키마는 없다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md (references/sources.md도 읽음)
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
추정 방법·가정·예비 분리, 단계별 증명/체크포인트, 대안/핵심 가정(유효한 정책 입력 확보)을 위에 반영했다. 외부 비용편익 이론이나 정량 신뢰수준을 주장하지 않았다.

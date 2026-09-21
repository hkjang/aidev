- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 releaser는 현재 버전 0.0.0을 확인했지만 다음 버전·커밋·태그 관례의 출처가 없어 버전 결정 단계에서 실패한다. 적용 가능한 실제 근거가 입력되어야 기존 릴리즈 판정을 유지하면서 릴리즈를 재개할 수 있다.
- 수용 기준: 1) aiportal-front와 기준 SHA에 적용되는 승인 정책 또는 실제 과거 릴리즈 증거로 증가 단위·커밋/태그 형식·노트 위치·자산 방식·검증 명령을 확인한다. 2) 그 출처에 따라 package.json version과 package-lock.json의 version 및 packages[""].version을 일관되게 처리하고 정본에 출처를 연결하며, 근거 없는 관례와 skipped/released 우회가 없어야 한다. 3) 실제 releaser가 새 근거로 생성한 결과 및 산출물을 동일 cmd_release/evaluate_release로 검사해 통과하고, 기존 failed JSON 두 건은 계속 거부됨을 보여야 한다. 회귀 테스트나 수동 작성 released JSON만 통과한 것은 해결이 아니다.
- 건드릴 파일: 현재 승인된 저장소 변경 없음. 재개 입력 확보 후에만 docs/RELEASE.md: 확인한 근거/미확인 항목에 검증 가능한 출처 연결; package.json:version 및 package-lock.json:version,packages[""].version은 해당 정책이 실제 변경을 요구할 때 함께 처리. 외부 /mnt/c/Users/USER/projects/aidev/release-prompt.md: 절차 2·3·5, bin/run.sh:release_project/release_context/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 직접 읽은 진단 대상이며 수정 범위가 아니다.
- 검증 명령: 아래 실제 실행한 읽기 전용 명령과 결과를 참조한다. npm test / npm run build:dev는 실제 scripts지만 node_modules가 없어 이번 실행하지 않았으며 버전 판단을 대체하지 않는다.
- 위험과 피할 것: 신규 출처 없이 같은 BLOCKED 조사/구현을 다시 시작하지 말 것. 임의 0.0.1 증가, 문서의 1.0을 앱 버전으로 채택, 과거 실패 JSON 수정, 판정 조건 완화, .gitlab-ci.yml/외부 run.sh·gate.py 변경, auth/storage 관련 기능 혼입 금지. 원격 조회 실패/빈 context는 원격 이력 부재의 증거가 아니다. 정찰은 코드·커밋·태그·배포를 수행하지 않는다.
- 차선 후보: 없음 — 고정 릴리즈 수정 과제이므로 UI·문서 개선으로 바꾸지 않는다. 신규 입력이 없으면 기존 pending과 미완료를 인계하며 별도 무변경 구현을 새 성과로 만들지 않는다.

## 이번 판정과 원인
실행 가능한 코드 수정은 미확정이다. 사용자 절대 규칙의 읽기 전용 범위를 지켰으며, 수정 및 수정 후 동일 검증 통과는 미완료다. 과거 회차와 같은 입력 결손을 새 구현 과제로 재배정하지 않는다. 이 문서는 고정 수정 과제의 상태·재개 계약이며 해결 성과가 아니다.

- 기준: e938e8e1c10974e6afd49e49fe5667d7988961ad, non-shallow, 37커밋, 로컬 태그 0개. 최근 git log -30과 패키지 파일별 모든 변경 커밋의 JSON을 확인했다.
- 현재 세 버전 필드 0.0.0. 최초 package 0.0.0, 최초 lock 두 필드 ABSENT. 버전 파일이 있으므로 release-prompt 절차 5의 skipped 조건 미충족.
- .github/workflows 및 CLAUDE.md 없음. .gitlab-ci.yml은 main/develop 전용 Runner의 reset/build/cp이며 버전 릴리즈 증거가 아니다.
- 실패 원본: /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 2026-09-21-001359-aiportal-front-improve/release.json. 첫 기록에는 스킬 접근 실패도 있지만 둘째는 스킬 원문 접근 후 정책 결손으로 실패한다. 스킬 경로 안내를 다시 고치는 접근은 이번 해결책이 아니다.
- 실제 경로: release_project → run_agent → release.json → gate.py release/cmd_release/evaluate_release → ok=false이면 push 전 종료. retry_release_workflow는 별도의 GitHub run 재시도 경로다. GitHub run ID/실패 step 로그는 미확인으로, 두 JSON을 동일 GitHub 워크플로의 독립 실패 2회로 확정하지 않는다.
- release_context의 gh 오류/빈 목록 혼동과 tag fetch 오류 후 진행은 관측한 외부 진단 사항이다. 이번 정책 결손과 인과 미확인이고 외부 소유 변경으로 우회하지 않는다. 이번 원격 조회는 미실행.

## 재개 계획과 검토 지점
1. 대기: 새 근거가 인계될 때만 출처 원문과 대상 프로젝트/SHA를 검토한다. 입력에는 증가 단위, 커밋/태그 양식(주석 여부 포함), 노트 위치/양식, 자산 생성 여부/명령, 검증 명령이 필요하다. 없는 항목은 미확인으로 둔다. 검토 지점: 새 관례를 창작해야 하면 중단; 이미 승인된 근거가 충분하면 추가 확인 질문 없이 다음 단계로 간다.
2. 미착수: 확인된 범위만 정본 및 버전 필드에 적용한다. 세 필드 비교, git diff --check, 해당 정책의 실제 검사로 증명한다. 검토 지점: 정책과 코드가 어긋나면 계획을 수정하며 워크플로 우회 금지. 출처가 없으면 이 단계 진입 금지.
3. 미착수: 실제 releaser를 게시 없이 실행 가능한 기존 경로로 재현하고 새 release.json을 원본 gate에 넣는다. 외부 run.sh 전체 실행은 push·원격 부작용을 포함하므로 로컬 검증 명령으로 사용하지 않는다. 게시 없는 실제 모델 실행의 정확한 호출은 신규 입력 이후 실행기 담당 범위에서 확정해야 하며 현재 미확인이다. 이 조건이 미확정이면 전체 통과를 주장하지 않는다.

## 실행한 검증과 한계
저장소 루트에서 실행 가능한 명령:
```bash
git rev-parse HEAD --is-shallow-repository
git rev-list --count HEAD
git tag --list
git log -30 --oneline
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
git diff --check
```
두 gate 명령은 각각 exit 1 / state failed / ok false가 정상 재현 결과다. 외부 tests/test_gate.py의 실제 Release 클래스 5개도 importlib+unittest로 한 번 실행하여 OK. tempfile.tempdir만 회차 내부 TemporaryDirectory로 지정하고 임시 파일은 정리했다. 프로덕션 gate 함수와 CLI를 실행했으나 회귀 클래스의 성공 JSON은 수동 fixture이므로 실제 releaser 판단이나 수정 후 릴리즈 성공을 증명하지 않는다. 원본 JSON·gate 무변경. 앱 설치/test/build·외부 사용자 검증·Runner 배포 미실행.

## 대안 비교와 추정 근거
- 추천: 기존 승인 정책/실제 관례 복구. 변경 범위가 작고 기존 판정을 보존하지만 해당 출처가 필수다.
- 새 릴리즈 자동화 도입: 정책 결손을 해소하지 못하고 범위가 45분을 넘길 수 있어 제외.
- 새 변경 없이 문서/스킬 탐색 반복: 이미 no-change가 반복되고 최신 실패는 스킬 접근 후 발생하므로 재배정 제외.
- 현 상태 보존: 새 출처 없을 때의 정직한 결과이나 문제 해결로 계산하지 않는다. 판정 완화는 허용 가능한 대안이 아니다.
추정은 bottom-up 가정치다. 근거가 완비된 이후 출처 매핑 5–10분, 최소 반영 5–10분, 해당 로컬 검증 10–15분으로 20–35분, 알려진 명령/파일 조정 여유 0–10분을 별도 계상해 20–45분(S)이다. 완료 확률을 산정할 실측 표본은 없어 신뢰도 낮음. 출처 확보 대기·새 정책 수립·실제 배포·환경 설치는 포함하지 않으며 시간 미확정; 관리 예비비는 배정하지 않았다. 과거 유사 회차는 반복 no-change라 성공 소요 시간으로 비교할 수 없고, 출처 확보 후 반드시 재추정한다.

## 스킬 적용
호출 가능한 Skill 도구 없음. 다음 실제 원문을 직접 읽고 적용했으며 성공한 Skill 호출로 간주하지 않는다. 세 원문에 별도 고정 반환 스키마는 없었다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md: 작업 분해, 범위·가정·불확실성·예비 시간 분리. 외부 비용 지침의 수치/완료확률은 인용하지 않았다.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md: 파일·증명·검토 지점·단계 상태 인계.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md: 결과와 비범위 명시, 대안 비교와 핵심 가정(적용 가능한 신규 출처) 명시.

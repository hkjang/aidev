- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 pending의 재개 조건 명시 (가치 5 / 위험 2 / 작업량 S)
- 왜: 실제 release.json은 다음 버전·릴리즈 커밋·태그 관례의 근거가 없어 버전 결정 단계에서 중단했음을 기록한다. 적용 가능한 출처를 실제 releaser 입력에 연결해야 기존 검증을 유지하며 재개할 수 있다.
- 수용 기준: 1) aiportal-front 및 기준 SHA에 적용되는 정책 원문 또는 실제 릴리즈 근거로 증가 단위·커밋 메시지·태그 형식/주석 여부·노트 위치/언어·자산 규칙·검증 명령을 확정한다. 2) 출처가 실제 releaser 입력에 전달되고 package.json 및 package-lock.json 두 버전 필드가 일치하며 기존 실패 차단 조건이 유지된다. 3) 실제 releaser가 버전 결정을 완료해 만든 release.json이 기존 gate CLI에서 exit 0/ok=true/released로 판정되고 기존 failed 입력은 여전히 거부된다. 합성 JSON·sim·문서 검사 통과는 해결 증거가 아니다.
- 건드릴 파일: 현재 즉시 수정할 파일 없음. 재개 입력 완비 후 docs/RELEASE.md:미확인 항목과 한계 — 확인된 출처 연결; package.json:version 및 package-lock.json:version/packages[""].version — 릴리즈 역할이 결정한 값만 동시 반영. 직접 열람한 외부 aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_project/release_context/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 진단 근거이며 이 저장소 수정 대상이 아니다.
- 검증 명령: 아래 읽기 전용 Git/JSON 명령과 동일 gate CLI. 앱 검증 명령은 npm ci → npm test → npm run build:dev이며 이번 실행하지 않았다.
- 위험과 피할 것: 임의 0.0.1 증가·과거 문서 기재의 릴리즈 증거 승격·failed를 skipped/released로 변경·워크플로/게이트 완화 금지. auth/storage/router/.gitlab-ci.yml 변경 금지. 외부 run.sh 전체 실행은 원격 게시를 포함하므로 실행하지 않는다. 동일 BLOCKED 조사·무변경 구현을 새 성과로 세지 않는다.
- 차선 후보: 없음 — 고정 실패를 무관한 앱 개선으로 대체하지 않는다. 새 적용 출처가 없으면 기존 pending을 유지하고 같은 구현을 신규 배정하지 않는다.

## 판정과 범위
정찰 완료, 수정 미완료. 신규 적용 정책·실제 릴리즈 출처가 없어 수용 기준 1 미충족, 2·3 미착수다. 45분 안에 실행 가능한 수정안을 현재 제약 아래 확정하지 못했다. 이는 사용자 지시와 AGENTS.md/docs/RELEASE.md의 관례 신설 금지에 따른 결론이며 스킬에서 별도 승인을 요구한 결과가 아니다. 질문이나 승인 요청 없이 기록을 끝낸다. 코드 변경 금지에 따라 이번 회차는 과제서·기록만 작성했다.

## 직접 확인한 근거
- HEAD e938e8e1c10974e6afd49e49fe5667d7988961ad, non-shallow, 전체 37커밋, git log -30, 로컬 태그 0개. package/lock 파일별 변경 커밋 JSON에 증가 없음. 최초 package는 0.0.0, 최초 lock 두 필드는 ABSENT; 현재 세 필드는 0.0.0.
- AGENTS.md와 docs/RELEASE.md, README, docs/07 로드맵·docs/09 테스트·docs/08 CI 일부, TODO/FIXME 검색, package/vite/vitest/CI, 19개 unit spec 목록과 common.spec.js 일부를 읽었다. CLAUDE.md·.github/workflows·node_modules 없음. docs 전체와 전체 테스트 본문을 읽은 것은 아니다.
- .gitlab-ci.yml은 main/develop 전용 Runner의 fetch/reset/build/cp 배포다. 외부 release_project → run_agent → release.json → cmd_release/evaluate_release는 failed를 게시 전에 거부한다. 앱 CI 실패나 같은 GitHub Actions step의 독립 실패 2회라는 근거는 미확인이다.
- 직접 읽은 실패 파일: /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json. 후자는 스킬 원문 접근 후 버전 근거 부족을 기록한다. 첫 파일의 최초 lock부터 0.0.0이라는 설명은 실제 이력과 달라 채택하지 않는다.
- retry_release_workflow는 별도 Actions 재실행 경로이며 run ID/실패 step 로그는 미확인. release_context의 gh 오류→없음 처리와 fetch 오류 후 계속은 확인했지만 이번 실패와 인과 미확인이다. 이번 원격 조회는 미실행이며 빈 context로 원격 이력 부재를 단정하지 않는다.

## 재개 계획·증거·체크포인트
1. [미충족] 수용 기준 1의 신규 출처와 적용 프로젝트/SHA 및 결정값을 대조한다. 원문 출처와 Git/JSON이 증거다. 신규 입력 없는 반복 조사는 이 단계를 통과시키지 않는다. 기존 no-change 구현을 재배정하지 않는다.
2. [미착수] 출처가 완비되면 docs/RELEASE.md에 적용 범위를 연결하고 실제 입력 전달과 세 버전 필드 일치를 검증한다. 값이나 적용 범위가 불일치하면 계획을 수정한다. 구현자가 릴리즈 역할의 커밋/태그 생성을 대신하지 않는다.
3. [미착수] 게시와 분리된 실제 releaser 실행 결과를 동일 gate CLI로 검증한다. 안전하게 분리한 releaser 단독 실행 명령은 미확인으로, 재개 인계 시 확정해야 한다. 각 단계 증거가 확보된 뒤 다음 단계로 진행한다. 새 사람 승인 체크포인트는 추가하지 않는다.

## 검증 명령과 이번 실행 구분
저장소 루트에서 실제 실행 가능한 읽기 전용 근거 검사:
```bash
git rev-parse HEAD --is-shallow-repository
git tag --sort=-creatordate
git log -30 --oneline
git log --oneline -- package.json package-lock.json
python3 -c 'import json; from pathlib import Path; p=json.loads(Path("package.json").read_text()); l=json.loads(Path("package-lock.json").read_text()); print(p["version"], l["version"], l["packages"][""]["version"]); assert p["version"] == l["version"] == l["packages"][""]["version"]'
git diff --check
```
이번 Git/JSON 이력 조회와 git diff --check는 실행했다. 아래 CLI/테스트는 원문으로 인자를 확인했지만 신규 입력 없는 반복 실행을 피하려고 실행하지 않았다:
```bash
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-172424-aiportal-front-improve python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release
```
복구 후 동일 gate.py release에 실제 새 release.json과 OUT 경로를 넣는다. 기존 실패 파일은 보존한다. Release 테스트는 형식·경로 검사로 실제 모델의 버전 판단·태그 존재를 증명하지 않는다. 인계의 failed 두 건 exit 1/ok=false 및 Release 5개 OK는 이번 결과나 수정 후 성공이 아니다. 앱 설치/test/build, gate, sim, releaser, 원격 배포는 이번 미실행이다.

## 접근 비교와 추정 근거
권고는 적용 출처가 생긴 뒤 최소 입력 연결을 수행하는 경로다. 현재는 기존 pending 보존만 가능하다. 공통 release_context 진단 개선은 외부 소유이며 이번 인과 미확인으로 제외한다. 스킬 폴백은 최신 실패가 원문 접근 이후 발생해 재선정하지 않는다. 검증 완화와 임의 새 정책은 명시적 금지라 대안이 아니다. 가장 큰 가정은 적용 가능한 출처가 새로 들어온다는 것이다.
S는 입력 완비 후 출처 대조 5–10분, 반영 5–10분, 로컬 검증/기록 10–15분의 bottom-up 20–35분이다. 알려진 경로·명령 불일치의 contingency 5–10분은 별도이며 총 25–45분은 낮은 확신의 주관적 범위다. 출처 확보 대기·앱 빌드·전체 릴리즈·배포는 제외하므로 전체 해결 시간을 보장하지 않는다. 비교 가능한 성공 사례가 없어 analogous 교차검증은 불가하다. 관리 예비는 범위 변경 결정에 따로 남기며 합계에 숨기지 않는다. 외부 표준으로 수치나 확률을 인증하지 않는다.

## 후보 재평가
| 후보 | 가치/위험/작업량 | 판정 |
|---|---|---|
| 고정 릴리즈 입력 복구 | 5/2/S | pending, 신규 입력 없이 재배정 불가 |
| useAppList 비배열 캐시 | 3/2/S | 두 소비 경로에서 truthy 캐시→map 유지 |
| globalLoading 병렬 요청 | 3/3/S | boolean start/stop 유지, 런타임 미확인 |
| README 빌드 명령 정합성 | 2/1/S | 문제 해결의 build 스크립트 없음 |
| 테스트 가이드 링크 | 2/1/S | docs/09 자체 이름 불일치·12~14 부재 |
| 신규: Alert HTML 본문 입력 정책 | 4/3/M | v-html 확인, 외부 입력 도달과 허용 계약 미확인 |
| 신규: 두 Alert 경로 동시 노출 | 2/3/M | App.vue와 Alert.js 별도 인스턴스 확인, 실제 충돌 미확인 |
기존 69개 제목/상태를 보존하고 신규 2개를 추가했다. 직접 열람 범위에서 done/rejected로 바꿀 새 근거 없음. 나머지는 동일 HEAD와 이전 기록 기준 보존하며 개별 해결 여부·런타임 효과는 미확인이다. 모든 앱 후보는 고정 실패의 대체 과제가 아니다.

## 스킬 적용 기록
호출 가능한 Skill 도구가 없어 읽기 가능 여부 확인 후 아래 원문을 직접 읽었다. 성공한 Skill 호출로 간주하지 않으며 별도 고정 반환 스키마는 없다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md — 범위 분해, 가정, 추정 범위, contingency/관리 예비 분리.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md — 파일·증거·체크포인트와 미착수 상태 명시.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md — 결과·비범위·접근 비교·권고와 핵심 가정 명시.

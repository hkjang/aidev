- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 기존 미해결 건 유지 (가치 5 / 위험 2 / 작업량 S)
- 왜: 실제 실패 기록은 다음 버전·릴리즈 커밋·태그 관례를 확정할 근거가 없어 외부 releaser가 중단했음을 보여 준다. 적용 근거가 입력에 연결되어야 검사 완화 없이 재개할 수 있으나 이번에도 신규 출처가 없어 같은 무변경 구현을 신규 배정할 수 없다.
- 수용 기준: 1) aiportal-front와 기준 SHA에 적용되는 정책 원문 또는 실제 릴리즈 출처로 증가 단위·커밋 메시지·태그 형식과 주석 여부·노트 위치와 언어·자산 규칙·정확한 검증 명령을 확정한다. 2) 그 출처에 근거한 입력 복구가 실제 releaser에 전달되고 package.json 및 lock 두 필드가 일치하며 기존 실패 차단 조건이 유지된다. 3) 실제 releaser가 버전 결정을 완료하고 실제 생성한 release.json이 동일 gate CLI에서 exit 0/ok=true/released가 되며 기존 failed 입력은 계속 거부된다. 합성 JSON·sim·문서 검사 통과는 해결 증거가 아니다.
- 건드릴 파일: 현재 즉시 수정 가능한 파일 없음. 재개 조건 충족 후 docs/RELEASE.md:미확인 항목과 한계 — 확인된 적용 출처 연결; package.json:version 및 package-lock.json:version/packages[""].version — 릴리즈 역할이 확정한 값만 함께 반영. 직접 읽은 외부 aidev/release-prompt.md:절차 2·3·5, bin/run.sh:release_project/release_context/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 진단 근거이며 이 저장소 수정 범위 밖이다.
- 검증 명령: 아래 Git/JSON 및 gate CLI. 앱 변경 검증은 npm ci → npm test → npm run build:dev이며 이번 설치·test/build는 미실행이다. 앱 검사만으로 버전 근거 결손이 해결되지는 않는다.
- 위험과 피할 것: 임의 0.0.1 증가, 문서의 초기 릴리스 기재를 실제 릴리즈로 승격, failed를 skipped/released로 바꾸기, 게이트/워크플로 완화 금지. .gitlab-ci.yml·auth·storage·router를 변경하지 않는다. 외부 run.sh 전체 실행은 원격 게시를 유발할 수 있어 실행하지 않는다. 빈 context를 원격 이력 부재로 단정하지 않는다. 동일 BLOCKED 조사나 무변경 구현을 새로운 해결 성과로 세지 않는다.
- 차선 후보: 없음 — 고정 수정 과제를 무관한 개선으로 대체하지 않는다. 재개 입력이 없으면 기존 pending 유지로 끝내며 동일 구현 과제를 새로 채택하지 않는다.

## 판정
정찰 완료, 수정 미완료. 코드 변경 금지 및 관례 신설 금지 안에서 즉시 실행할 수 있는 45분 수정안을 확인하지 못했다. 이는 스킬이 추가 승인을 요구해서가 아니라 사용자 지시와 docs/RELEASE.md가 요구하는 적용 근거가 없기 때문이다. 질문하지 않는다. 수정 후 동일 검증 통과는 미확인이고 성공으로 보고하지 않는다.

## 직접 확인한 근거
- HEAD e938e8e1c10974e6afd49e49fe5667d7988961ad, non-shallow, 전체 37커밋, git log -30, 로컬 태그 0개. package/lock 파일별 변경 커밋 JSON에 증가 없음. 최초 package 0.0.0, 최초 lock 두 필드는 ABSENT. 현재 세 필드는 0.0.0.
- AGENTS.md·docs/RELEASE.md를 먼저 읽고 README·docs/07 권장 로드맵·docs/09 테스트 가이드 일부, TODO/FIXME 검색, package/vite/vitest/CI와 19개 unit spec 목록 및 common.spec.js 일부를 확인했다. CLAUDE.md·.github/workflows·node_modules 없음. 모든 docs 본문과 모든 테스트를 읽은 것은 아니다.
- 실제 .gitlab-ci.yml은 main/develop 전용 Runner의 reset/build/cp 브랜치 배포다. 버전 릴리즈 워크플로가 아니다. 외부 release_project → run_agent → release.json → cmd_release/evaluate_release는 failed를 게시 전에 거부한다.
- 직접 열람한 실패 JSON: /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json. 후자는 스킬 접근 성공 뒤 버전 근거 결손을 기록한다. 첫 JSON의 최초 lock부터 0.0.0이라는 서술은 실제 이력과 달라 채택하지 않는다.
- retry_release_workflow는 별도 GitHub 재실행 경로. Actions run ID/실패 step 로그 미확인으로 두 JSON을 동일 Actions 단계의 독립 실패 2회로 확정하지 않는다. 원격 조회는 이번 미실행.

## 재개 계획과 체크포인트
1. [미충족] 위 수용 기준 1의 신규 원문·적용 프로젝트/SHA·결정값을 대조한다. 원문과 Git/JSON이 증거다. 신규 근거 없는 반복 재조회는 이 체크포인트 통과가 아니다. 새로운 사람 승인 요청을 생성하지 않는다.
2. [미착수] 근거가 완비되면 docs/RELEASE.md에 출처와 적용 범위를 연결하고 실제 입력 전달을 확인한다. 값 불일치 시 계획을 수정하고 중단한다. 세 버전 일치 검사를 한다. 구현 역할은 릴리즈 커밋·태그를 대신 만들지 않는다.
3. [미착수] 게시와 분리된 실제 releaser가 만든 결과를 동일 gate CLI로 검증한다. 안전하게 분리된 releaser 단독 호출 명령은 미확인이므로 재개 인계 때 확정해야 한다. 외부 실행기 전체 실행이나 수작업 released JSON으로 대체하지 않는다. 단계별 증거 확인 후 다음 단계로 진행하며 미실행을 완료로 표시하지 않는다.

## 검증 명령과 이번 실행 범위
저장소 루트의 읽기 전용 근거 확인 명령:
```bash
git rev-parse HEAD --is-shallow-repository
git tag --sort=-creatordate
git log -30 --oneline
git log --oneline -- package.json package-lock.json
python3 -c 'import json; from pathlib import Path; p=json.loads(Path("package.json").read_text()); l=json.loads(Path("package-lock.json").read_text()); print(p["version"],l["version"],l["packages"][""]["version"]); assert p["version"]==l["version"]==l["packages"][""]["version"]'
git diff --check
```
동일 실패 판정 CLI와 게이트 회귀 테스트의 실제 경로/인자는 원문으로 확인했다. 신규 입력 없는 반복 실행을 피하기 위해 이번에는 실행하지 않았다:
```bash
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-155428-aiportal-front-improve python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release
```
복구 후에는 실제 새 release.json 경로와 OUT 디렉터리를 같은 gate.py release 명령에 넣어야 한다. 기존 실패 파일은 보존한다. 이전 인계의 failed 두 건 exit 1/ok=false 및 Release 5개 OK(ResourceWarning)는 이번 실행이나 수정 후 통과 증거가 아니다. Release 테스트는 형식·경로 판정 검사이며 실제 모델의 버전 판단·태그 실재를 입증하지 않는다.

## 접근 비교와 추정
권고는 적용 출처를 확보한 뒤 입력을 연결하는 경로다. 현재는 출처가 없어 기존 pending 유지가 유일하게 실행 가능한 처리다. release_context의 조회 오류→없음 처리와 tag fetch 오류 후 계속은 확인했지만 이번 실패와 인과가 미확인이고 외부 소유이므로 이번 수정안으로 택하지 않는다. 스킬 폴백 개선도 최신 실패가 원문 접근 뒤 발생하여 재선정하지 않는다. 가장 큰 가정은 적용 가능한 출처가 새로 제공된다는 것이다.
S 추정은 입력 완비 후 제한된 문서·입력 연결만 포함한다. Bottom-up으로 출처 대조 5–10분, 반영 5–10분, 로컬 검증/기록 10–15분으로 20–35분; 알려진 경로/명령 불일치 contingency 5–10분을 별도로 두어 25–45분이다. 주관적 낮은 확신의 범위이고 통계적 확률을 주장하지 않는다. 출처 확보 대기·앱 빌드·전체 릴리즈/배포는 제외하며 전체 해결 시간은 추정 불가. 비교 가능한 성공 사례가 없어 analogous 교차검증 불가. 관리 예비는 범위 확대 결정에 별도로 남기며 합계에 숨기지 않는다.

## 후보 재평가
| 후보 | 가치/위험/작업량 | 판정 |
|---|---|---|
| 고정 릴리즈 입력 복구 | 5/2/S | pending, 신규 근거 없어 재배정 불가 |
| useAppList 비배열 캐시 | 3/2/S | fetchTopApps/updateList 둘 다 truthy 캐시를 map으로 전달, pending |
| globalLoading 병렬 요청 | 3/3/S | boolean start/stop 유지, 런타임 재현 미확인 |
| README 빌드 명령 | 2/1/S | 문제 해결의 npm run build는 scripts에 없음, pending |
| 테스트 가이드 링크 | 2/1/S | docs/09 자체 파일명 불일치·12~14 부재, pending |
| 신규: Alert 콜백이 여는 다음 알림 | 3/3/M | onPrimary는 emit 뒤 close; 실제 전역 상태 연결 재현 필요 |
| 신규: Alert Enter 확인 접근 | 2/2/S | onKeydown preventDefault 및 onPrimary 주석; 실제 버튼 포커스 검증 필요 |
기존 67개 제목/상태를 보존하고 신규 2개를 더해 69개로 작성했다. 직접 열람 범위에서 done/rejected로 바꿀 새 근거 없음. 미열람 세부 항목은 이전 기록과 동일 HEAD 기준 보존하며 해결 여부·런타임 효과 미확인이다.

## 스킬 적용 기록
호출 가능한 Skill 도구 없음. HEADCOUNT_DIR 명시 없음. 후보 경로 존재와 읽기 가능 여부 확인 후 아래 원문을 직접 읽었고 성공한 Skill 호출로 간주하지 않는다. 별도 고정 반환 스키마는 없다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md: 범위 분해·추정 근거·확신·contingency와 관리 예비 분리. 외부 표준을 근거로 수치/확률을 인증하지 않는다.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md: 파일·변경·증거·체크포인트·미착수 상태 명시.
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md: 목표·비범위·다른 접근과 탈락 근거·핵심 가정 명시.

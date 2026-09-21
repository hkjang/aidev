- 과제: 수정 과제 — 릴리즈 러너 Codex 폴백의 headcount 원본 스킬 전달 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: aidev의 run_agent는 Claude용 Skill 안내를 Codex 폴백에도 그대로 넘기지만 registry에 적힌 스킬의 원본 경로·직접 읽기 안내는 전달하지 않는다. 원본을 실제로 읽을 수 있도록 전달하고 같은 자식 호출 경로를 검증하면 스킬 탐색 실패를 제거할 수 있으나, 최초 릴리즈 관례 부재는 별도의 미해결 조건이다.
- 수용 기준: 1) registry의 release 두 스킬과 상대 참조가 Codex 자식에 전달되어 실제 읽힌 증거가 있다. 2) 스킬 파일 누락·headcount 비활성화·Claude 정상 실행·Codex 폴백을 회귀 검증하며 기존 게이트와 원격 전송 제한이 유지된다. 3) 동일 입력의 수정 전 실패/수정 후 통과를 기록하고 최초 릴리즈 계약이 없는 fixture는 계속 차단된다; 프로젝트 릴리즈 복구 완료는 승인된 계약을 사용한 실제 release 결과와 게이트 통과까지 별도 증명한다.
- 건드릴 파일: 현재 앱 worktree에는 지정 실패의 원인 수정 대상 없음. 원인 소유 저장소 aidev의 bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — registry 기반 원본 경로 및 읽기 방법을 엔진별 전달; aidev/tests/test_sim.py:ReleaseSafety 및 tests/sim/run_sim.sh — 실제 전달 경로를 검사할 회귀 fixture 확장(기존 가짜 릴리즈 결과만으로 읽기 성공을 주장하지 않음). aidev/agents/registry.json은 읽기 입력이며 phase별 이름을 하드코딩하지 않는다. 실제 파일 열람 범위와 추가 설계는 아래 참조.
- 검증 명령: 아래 실행 결과와 구현 후 검증 순서를 따른다. 현재 앱에서는 `node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json` 실행 가능; 릴리즈 스킬 전달 복구를 증명하는 명령은 아니다.
- 위험과 피할 것: 코드·커밋 금지인 이번 정찰은 회차 산출물만 작성한다. 앱 auth·session·배포·.gitlab-ci.yml 변경으로 외부 러너 문제를 우회하지 않는다. gate.py, release-prompt 절차 5, 저장 release.json을 느슨하게 바꾸거나 failed를 skipped/released로 바꾸지 않는다. 최초 버전·태그·노트·자산 정책을 추측하지 않는다. 자격증명 값을 기록하지 않는다.
- 차선 후보: 최초 릴리즈 대상·버전·태그·노트·자산 계약의 근거 확보 — 같은 실패의 두 번째 장애물이며 현재 근거가 없으므로 자동 대체 구현 불가. 관련 없는 UI/DX 후보로 바꾸지 않는다.

착수 판정: pending / blocked — 현재 회차에서 실행 가능한 구현 과제로 승인하지 않는다.
이번 run.json은 project=aiportal-front-admin이고 registry의 builder.surface는 “그 회차의 worktree”다. COMPANY.md 규칙 1은 “구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다.”고 정한다. 실제 원인 경로는 /mnt/c/Users/USER/projects/aidev/bin/run.sh다. 따라서 위 파일 목록은 aidev 소유 회차에 필요한 수정 명세이며, 현재 앱 구현자에게 외부 편집 권한을 부여하지 않는다. 앞선 여섯 번의 no-change와 같은 외부 패치를 현재 실행 가능하다고 재발행하지 않는다. 앱 안에 스킬 사본·가짜 태그·관례 문서를 만들어 해결했다고 주장하는 것도 금지한다.
현재 지시의 “원인을 고친 뒤 통과”와 정찰의 “코드를 바꾸지 말라”는 후자를 지키고 구현 명세로 인계한다. 현재 정보로 45분 안에 프로젝트 릴리즈까지 완료한다는 약속은 성립하지 않는다. 상태 기록은 복구 성과가 아니다.

확인한 근거:
- run.sh:246 agent_plugin_args는 Claude --plugin-dir를 만든다. :254 dept_note는 스킬 이름만 포함한다. :290 run_agent에서 그 안내를 붙이고 :308에서 같은 prompt를 run_codex로 넘긴다. :314 run_codex에는 원본 경로 전달 구성이 없다. 이 정적 결함이 과거 자식의 전체 실패 원인임을 실제 재실행으로 입증하지는 않았다.
- 두 원본은 /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/release-and-deployment/SKILL.md 및 marketing/skills/product-launch/SKILL.md에 존재하며 직접 읽었다. 현재 사용 가능 도구 이름 검색에 Skill/skills.list/skills.read가 없다. 스킬 자체가 없다는 과거 문구는 현재 환경 전체의 사실이 아니다.
- release-prompt.md 절차 2/3은 기존 증가·태그 관례를 요구하고 절차 5는 태그·버전 파일·노트가 모두 없을 때만 skipped다. 네 package/lock 파일의 --all 로컬 이력을 이번에 직접 조회했고 root 0.0.0, V2 0.1.0만 존재한다. 로컬 태그, CHANGELOG/RELEASE 파일, .github/workflows 검색 결과 없음. 원격 Release 최신 조회는 미확인.
- .gitlab-ci.yml은 legacy 브랜치 기반 build/copy다. docs/OPERATIONS_RUNBOOK.md 및 V2 deploy/README.md는 정적 배포와 cache 반입을 설명한다. 이번 실패에 해당하는 GitHub workflow 파일·잡 로그는 제공되지 않았다.
- fixer.sh:52 이후는 failed snapshot을 다시 적재한다. run.sh:1690의 “같은 이유로 두 번”은 고정 문구다. 실제 두 workflow 실행의 증거로 세지 않는다.
- tests/test_gate.py:Release 5개는 형식·자산·태그 검사다. tests/test_sim.py:ReleaseSafety는 가짜 에이전트 결과에 대한 게시/차단 검사로, 스킬 읽기 검증이 없다.

구현 순서와 검증 체크포인트(소유 표면을 갖춘 aidev 회차용; 전부 미착수):
1. 동일 release 프롬프트/registry로 Codex 폴백에서 원본 위치를 전달하지 않는 실패 회귀를 추가한다. 구현자는 tests/test_sim.py의 기존 run/scn 및 tests/sim/run_sim.sh를 상세 재검토하여 실제 자식 argv와 읽기 결과를 수집해야 한다. 체크포인트: 수정 전 회귀 실패 확인, 사람 확인 추가 없음.
2. run_agent/run_codex 경계에서 registry로 원본 절대 경로를 해석하고 Skill 도구가 없는 경우 직접 읽기 안내를 준다. 상대 references를 따라갈 기준 경로를 유지하고 누락 스킬은 구체적인 오류로 남긴다. 공백 경로·비활성화·Claude 경로 유지도 검증한다. 체크포인트: 새 회귀 및 기존 Release/ReleaseSafety 통과. 아직 없는 테스트 명령을 이미 존재하는 명령처럼 쓰지 말고 구현 시 확정해 과제서에 기록한다.
3. 원격 전송을 차단한 실제 자식으로 두 원본 읽기 증거를 확인한다. 기존 실패 fixture는 최초 릴리즈 관례가 없으므로 여전히 차단되어야 한다. 소유자가 정한 최초 계약(대상 root/V2, 다음 버전, tag 또는 tag 없음, 노트 위치/양식, 자산 방식)이 확보되어야 프로젝트 실제 release 재검증에 진입한다. 체크포인트: 계약 미확보 시 프로젝트 릴리즈 복구 완료로 표시하지 않는다.

이번 실행한 명령과 결과:
```sh
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-043423-aiportal-front-admin-improve PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release
# exit 0, 5 tests OK; unclosed file ResourceWarning 있음
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-043423-aiportal-front-admin-improve
# exit 1, ok:false, state:failed — 저장 실패 결과 검사이며 새 자식 재현이 아님
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
# exit 0
```
구현 후 소유 저장소에서 추가 실행할 기존 명령: `python3 -B tests/test_sim.py ReleaseSafety` (이번 미실행, 각 시나리오 10~40초라는 파일 설명; 가짜 서비스 기반). V2 `npm ci`, `npm run verify`, `npm run build`는 package script 확인만 했고 node_modules가 없어 이번에는 실행하지 않았다. 실제 자식/수정 후 회귀/UAT 미실행.

대안 평가(technology:solution-exploration):
- 최소 수정인 registry 기반 경로 전달을 권고한다. 이미 존재하는 원본을 재사용하지만 aidev 소유 표면이 필요하다.
- 범용 엔진별 스킬 설치 계층은 여러 역할 확장에 유리하나 이번 45분 범위를 넘으므로 선택하지 않는다.
- 앱 저장소에 스킬을 복제하는 방식은 원본과 표류하고 최초 계약도 해결하지 못해 기각한다.
- 현 상태 유지·원장 반복은 완료가 아니다. 소유 회차 배정 없이 이를 구현 채택으로 세지 않는다.
가장 큰 가정은 aidev 소유 worktree와 실제 자식 실행 환경을 다음 구현에 제공할 수 있다는 점이며 현재 미충족이다.

견적(pmo:estimating-and-contingency):
bottom-up: 실패 회귀 8~12분 + 전달 수정 8~12분 + 기존/자식 검증 9~13분 = 기본 25~37분. 알려진 경로·fixture 차이 대응 contingency 5~8분을 별도로 두어 30~45분을 잠정 범위로 본다(통계적 80% 신뢰 구간이 아닌 낮은 확신의 작업자 추정). 경영 예비는 배정하지 않았다. 최초 정책 결정 대기·실제 배포·새 엔진 추상화는 제외한다. 유사 회차는 모두 no-change여서 속도 비교 근거가 되지 않으며, 소유 환경 확인과 첫 실패 회귀 후 재산정한다. 전체 릴리즈 복구는 현재 이 범위로 추정할 수 없다.

적용 스킬(원본 직접 읽기 대체):
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md (references/sources.md도 확인; 외부 통계·비용 수치는 사용 안 함)
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md

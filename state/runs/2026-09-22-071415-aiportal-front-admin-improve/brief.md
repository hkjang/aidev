- 과제: 수정 과제 — 지정 릴리즈 실패 복구의 착수 가능성 판정 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 릴리즈 실패는 앱 빌드 실패가 아니라 스킬 접근 실패와 최초 릴리즈 관례 부재를 보고하며, 같은 실패 스냅샷이 앱 개선 회차에 다시 적재되고 있다. 실제 원인 소유 범위와 릴리즈 계약을 확보해야 동일 검증으로 복구를 입증할 수 있고, 앱의 무관한 변경이나 기록만으로 성공 처리하는 일을 막을 수 있다.
- 수용 기준: 1) 실제 릴리즈 자식 실행에서 명부의 두 스킬 원본 및 상대 참조를 읽는 증거가 남는다. 2) 대상 패키지·다음 버전·태그·노트·자산에 대해 승인된 계약 또는 기존 관례를 근거로 로컬 릴리즈 결과를 만든다. 3) 수정 전 실패한 실제 경로를 수정 후 다시 실행하여 통과하고, gate.py release와 기존 보호 검사를 그대로 통과한다. 현재 세 조건 모두 미충족이며, 정찰 검사 통과는 대신할 수 없다.
- 건드릴 파일: 현 앱 worktree에는 지정 실패를 해결할 것으로 확인된 수정 대상 없음. 읽은 원인 후보는 외부 /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex/release_project, agents/registry.json, release-prompt.md:절차 2~5이다. 이들은 이번 앱 회차의 편집 지시가 아니다. tests/test_gate.py:Release와 tests/test_sim.py:ReleaseSafety는 현재 실패 원인의 실제 자식 회귀를 제공하지 않는다.
- 검증 명령: 아래 실제 실행 명령과 한계를 참조한다.
- 위험과 피할 것: 코드 변경·커밋을 하지 않는 정찰 범위를 준수했다. 구현자가 외부 러너를 앱 PR에서 고치거나, Skill 지시를 삭제하거나, skipped 조건/게이트를 완화하거나, 최초 버전·태그를 임의 생성하거나, 저장된 release.json을 성공으로 덮어쓰면 안 된다. auth·session·.gitlab-ci.yml·deploy 변경은 지정 실패의 근거가 없으므로 끼워 넣지 않는다. 비밀값은 출력하지 않는다.
- 차선 후보: 없음 — 우선 과제를 다른 앱 개선으로 대체할 수 없다. 외부 소유 범위와 최초 릴리즈 계약이 갖춰지기 전에는 지정 과제를 pending으로 유지한다.

## 착수 판정과 이전 실패에서 달라진 점
현재 판정은 blocked다. 45분 안에 실행 가능한 앱 수정 과제로 승인하지 않는다. 이전에 기각되거나 no-change였던 ‘외부 스킬 전달 패치를 앱 구현자에게 지시’하는 접근을 재발행하지 않는다. 이 문서는 복구 성과가 아니라 수정 과제의 미충족 조건과 진단 증거를 남긴 것이다. 사용자의 ‘코드 변경 금지’와 기록 경로 제한 때문에 정찰이 직접 수정 후 통과까지 수행할 수 없다.

직접 확인한 /mnt/c/Users/USER/projects/aidev/COMPANY.md 규칙 1은 ‘구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다.’이다. agents/registry.json의 builder.surface는 ‘그 회차의 worktree’이고 run.json의 project는 aiportal-front-admin이다. 따라서 외부 수정은 현재 앱 회차의 실행 계획으로 제시하지 않는다. 이는 스킬이 요구하는 추가 승인 절차가 아니다.

## 확인한 근거
- HEAD 01fedba; git log -30을 읽었고 작업 트리 변경 없음. 추적 AGENTS.md/CLAUDE.md/CHANGELOG/.github 파일과 로컬 태그 없음. 원격 Release 최신 조회는 미확인.
- 네 package/lock 파일을 git log --all 및 git show로 조회: root는 0.0.0, V2는 0.1.0만 확인. 버전 파일이 있으므로 release-prompt 절차 5의 ‘태그도, 버전 파일도, 릴리즈 노트도 없음’과 다르다.
- README, docs/ROADMAP.md, docs/OPERATIONS_RUNBOOK.md, docs/TESTING_GUIDE.md, V2 deploy/README.md와 .gitlab-ci.yml을 읽었다. 기존 GitLab 브랜치 빌드/정적 파일 복사는 최초 태그·버전 증가 계약을 정하지 않는다.
- run_agent는 Claude에 plugin-dir/Skill 안내를 구성한 prompt를 run_codex로 전달한다. run_codex 호출에는 스킬 원본 경로나 대체 읽기 안내가 추가되지 않는다. 이는 정적 원인 후보이며 실제 실패 자식의 전체 원인을 입증한 것은 아니다.
- headcount/plugins/{technology,marketing}/skills 아래 릴리즈 두 원본은 지금 실제 존재한다. 실패 당시 실행 환경에서 읽을 수 있었는지는 미확인. 현재 callable tool 목록에는 Skill/skills.list/skills.read가 없다.
- fixer.sh의 status=failed 순회는 저장된 상태를 다시 적재하고, run.sh:round_body의 ‘두 번 실패’ 문구는 고정 문자열이다. 문구 자체는 실제 두 실행 증거가 아니다.

## 적용 스킬과 대안 판단
Skill 도구가 없어 다음 원본을 직접 읽었다(도구 호출 성공으로 기록하지 않음).
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
견적 스킬의 references/sources.md도 확인했다. 아래 시간은 외부 통계가 아닌 관찰한 코드 범위에 대한 낮은 확신의 작업 추정이다.

대안 A: 앱의 CI/버전을 바꾼다 — 이번 실패 원인 증거와 최초 계약이 없어 배제.
대안 B: 외부 소유 회차에서 명부 기반 원본 접근을 복구하고 실제 자식으로 검증한다 — 원인 후보와 맞지만 현재 회차 범위 밖이며 이전 차단을 재승인할 수 없음.
대안 C: 수작업으로 스킬을 읽고 릴리즈를 재시도한다 — 이번 정찰에서 원본 접근은 확인했지만 프로덕션 배선 수정 증거가 아니고 계약도 없어 배제.
결론: 지정 실패는 유지하되 앱 구현 착수는 하지 않는다. 재개 판단의 핵심 가정은 외부 러너 소유 작업과 최초 계약이 별도로 제공될 수 있다는 것이며 현재는 미확인이다.

## 재개 조건부 계획 — 이번 회차 실행 지시가 아님
1. 원인 소유 worktree와 승인된 릴리즈 계약의 출처를 확보한다. 증거: 소유 surface와 계약의 실제 파일/내용. 체크포인트: 둘 중 하나라도 없으면 blocked 유지, 무관한 앱 후보로 전환 금지.
2. 그 소유 회차에서 실제 run_agent → run_codex 자식의 스킬 읽기 실패를 먼저 재현하고, 명부와 HEADCOUNT_DIR에 근거해 원본/상대 참조 접근을 복구한다. Claude 경로, 명시적 headcount 비활성화, 경로 공백도 함께 보존한다. 증거: 실제 자식 실행의 전후 로그. 소스 문자열 검사나 가짜 CLI 출력은 수용 증거가 아니다. 실행 가능한 격리 하네스는 현재 미확인이라 명령을 지어내지 않는다.
3. 승인된 동일 계약으로 로컬 릴리즈를 다시 실행하고 아래 gate를 실제 새 결과에 적용한다. 체크포인트: 자식 검증과 계약 모두 충족한 경우에만 진행. 릴리즈 원격 게시·태그 푸시는 이 계획에 포함하지 않는다.

견적 근거: 전제가 충족되고 실제 자식 하네스가 이미 있을 때 전달 수정 10~15분 + 자식 회귀 10~15분 + 게이트/기록 5~10분 = 25~40분, 알려진 경로 차이 대응 예비 5분으로 30~45분(M). 통계적 신뢰구간이 아니며 확신 낮음. 소유 범위 변경, 계약 결정, 하네스 신규 개발, 최초 빌드 및 외부 UAT는 포함하지 않았고 소요 미확인이다. 관리 예비는 별도 미배정이다. 이 전제들이 현재 충족되지 않아 전체 복구를 45분으로 약속하지 않는다.

## 실제 검증과 재현 명령
앱 cwd: /home/hkjang/.cache/auto-improve-wt/aiportal-front-admin
아래 OUT은 이 회차 경로이며 validation.json에 명령·exit·stdout·stderr를 보존했다.

```bash
OUT=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-071415-aiportal-front-admin-improve
TMPDIR="$OUT/validation-tmp" python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir "$OUT"
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
```
결과: Release 5개 exit 0(ResourceWarning 있음); 저장 실패 결과 gate exit 1/state=failed; runtime config exit 0. gate 실패 결과는 기존 상태 판정 재현이며 실제 릴리즈 자식 재현이나 수정 후 검증이 아니다. ReleaseSafety는 가짜 에이전트를 쓰므로 실제 스킬 접근 증거로 쓰지 않았다. npm 설치/unit/build, 실제 자식 재실행, 수정 후 회귀, 서버/UAT는 미실행.

## 보류 후보 재평가
ideas.json은 기존 48개를 보존하고 두 신규 후보를 더했다. 기존 query 무효화 done은 현재 두 applyRouteQuery의 invalidate 호출을 확인해 유지했다. 나머지 pending의 해결 완료 증거는 없으며 개별 런타임 미확인 항목을 done으로 바꾸지 않았다. 신규 두 항목은 실제 verify-offline.mjs CLI로 외부 object[data]와 CSS image-set 문자열 URL을 각각 넣어도 exit 0을 확인했다. 이 검증 공백은 지정 릴리즈 실패와 별개이며 차선 착수 지시가 아니다.

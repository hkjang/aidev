- 과제: 수정 과제 — 릴리즈 실패의 실행 환경·최초 릴리즈 계약 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 릴리즈 실패는 앱 빌드 결함이 아니라 스킬 접근 실패와 최초 릴리즈 관례 부재를 함께 보고하며, 현재 러너는 Claude 전용 Skill 안내를 Codex에도 그대로 넘긴다. 엔진별 스킬 전달과 근거 있는 릴리즈 계약을 각각 해결해야 같은 실패를 반복하지 않으며, 앱 변경이나 성공 상태 조작은 해결이 아니다.
- 수용 기준: 1) 소유 범위가 확보된 러너에서 실제 Codex 자식이 registry가 지정한 두 릴리즈 스킬 원본과 상대 참조를 읽었음을 증명한다. 2) 대상 패키지·버전 증가·태그·노트·자산 관례에 대해 기존 이력 또는 운영 결정 출처를 확보하고, 같은 릴리즈 경로에서 그 계약으로 생성한 결과가 기존 gate를 통과한다. 3) 원본 누락·계약 부재는 계속 차단되고 정상 입력은 실제 프로덕션 호출 경로를 통과하는 수정 전 실패/수정 후 성공 검증이 있어야 한다. 기존 Release 테스트 5개 통과, 소스 문자열 검사, Fake CLI만으로 완료 처리하지 않는다.
- 건드릴 파일: 현재 aiportal-front-admin worktree에는 확인된 원인 수정 파일 없음. 아래는 외부 소유 작업의 인계 대상이며 이번 구현자의 편집 허가가 아니다: `/mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex` — registry와 HEADCOUNT_DIR에서 엔진에 맞는 스킬 접근 정보를 전달; 같은 파일 `release_project` — 계약과 실제 릴리즈 결과의 연결을 검증; `/mnt/c/Users/USER/projects/aidev/agents/registry.json` — 역할·스킬의 정본으로 사용(하드코딩 목록 추가 금지). `release-prompt.md:절차 2~5`는 읽기 기준이며 실패를 통과시키기 위한 완화 금지. `tests/test_gate.py:Release`, `tests/test_sim.py:ReleaseSafety`는 기존 검증 범위 확인용이다.
- 검증 명령: 아래 '실행한 검증'의 명령은 현 환경에서 실행 확인했다. 수정 후 실제 자식 회귀 명령은 현재 존재하는 것으로 확인되지 않았으므로 미확인이다. 소유 환경에서 실행 가능한 회귀 진입점을 먼저 만들고 그 정확한 명령을 과제서에 기록해야 하며 임의 명령을 가정하지 않는다.
- 위험과 피할 것: 코드 편집·커밋 금지인 정찰이다. 구현자의 표면도 현재 앱 worktree이며 외부 러너 편집으로 확대하지 않는다. .gitlab-ci.yml/auth/deploy/버전 파일은 이번 실패의 확인된 원인 수정 대상이 아니다. 임의 v0.1.1, CHANGELOG, 태그 생성, failed→skipped/released 변경, gate 완화, 원격 전송, 같은 차단의 재탐색을 개선 성과로 세는 일을 금한다.
- 차선 후보: 없음 — 우선 과제가 고정되어 앱 UX·테스트·무결성 후보로 바꾸지 않는다. 실행 가능한 대상이 없으면 아래 판정으로 종료하고 성공을 주장하지 않는다.

## 착수 판정 — 이번 회차 pending / 실행 불가
이 문서는 종전의 기각된 외부 수정 접근을 재승인하는 실행 지시가 아니다. 현재 run.json은 project=aiportal-front-admin이고 registry.builder.surface는 '그 회차의 worktree'다. 직접 읽은 COMPANY.md 규칙 1은 '구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다'고 명시한다. 승인된 최초 릴리즈 계약도 확인하지 못했다. 따라서 이 회차에서 45분 내 끝낼 수 있는 허용된 원인 수정 과제는 성립하지 않는다. 구현자는 동일 탐색·차단 기록을 다시 채택/완료로 보고하지 말고 수용 기준 미달을 기록한다. 요청된 수정 완료·수정 후 검증 성공은 달성하지 못했다.

## 확인한 근거
- HEAD 01fedba, git log -30 읽음, 시작 git status --short 빈 출력, 로컬 태그 없음. CLAUDE.md/AGENTS.md/.github/workflows는 저장소 검색에서 없음.
- README.md, docs/ROADMAP.md, docs/OPERATIONS_RUNBOOK.md, docs/TESTING_GUIDE.md, tests/README.md, 두 package.json, V2 vite.config.ts, deploy/README.md와 .gitlab-ci.yml을 읽음. CI 자격증명 값은 마스킹했다. CI는 legacy branch build/copy이며 이 실패의 GitHub workflow 실행 ID는 제공되지 않았다.
- 네 package/lock 파일의 전체 로컬 --all 이력 조회 결과 root 0.0.0, V2 0.1.0만 존재. 원격 최신 Release 재조회는 미실행. 최초 값 존재는 릴리즈 증가 관례를 증명하지 않는다.
- run.sh:run_agent는 Claude plugin-dir와 Skill 안내를 만들고 run_codex에는 같은 prompt를 넘긴다. run_codex는 해당 원본 경로/대체 읽기 안내를 구성하지 않는다. 이는 정적 배선 결함 근거이며 과거 자식 실패 전체의 런타임 입증은 아니다.
- headcount/plugins/technology/skills/release-and-deployment/SKILL.md 및 marketing/skills/product-launch/SKILL.md는 실제 로컬에 존재한다. 현재 도구 목록에 Skill 호출 도구는 없다.
- release-prompt.md 절차 5는 태그·버전 파일·노트가 모두 없을 때만 skipped다. 버전 파일이 있는 현재 앱에 적용해 우회할 수 없다.
- bin/fixer.sh:52~61은 failed snapshot을 반복 적재하며 run.sh:1690은 고정된 '두 번 실패' 문구를 붙인다. 그 문구만으로 서로 다른 워크플로 실행 두 건을 입증할 수 없다.

## 실행한 검증 (이번 회차)
1. 저장 실패 재판정: `python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-035417-aiportal-front-admin-improve` → exit 1, ok:false/state:failed. 실제 릴리즈 재실행이나 수정 전후 회귀가 아니다.
2. 기존 단위 기준선: `TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-035417-aiportal-front-admin-improve python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release` → 5개 통과/exit 0, 미닫힘 파일 ResourceWarning 발생. 테스트 임시 파일은 허용된 회차 경로에만 생성했다. 이 테스트는 스키마·자산 판정만 검증한다.
3. 앱 독립 검사: `cd upgrade/admin-v2 && node scripts/validate-runtime-config.mjs public/config/runtime.json` → exit 0. 릴리즈 복구 증거가 아니다.
4. 미실행: 실제 Codex 자식 회귀, ReleaseSafety 시뮬레이션, npm ci/verify/build, 서버/UAT. V2 node_modules 없음. 설치 후 앱 명령은 `cd upgrade/admin-v2 && npm ci && npm run build`이며 릴리즈 원인 검증을 대신하지 않는다.

## 해결안 비교와 인계 순서
- 최소 해결안(권고): 러너 소유 작업에서 registry 기반 Codex 스킬 원본 전달을 복구하고, 별도 운영 계약 근거를 입력받아 기존 릴리즈 절차를 수행한다. 두 문제가 독립이므로 하나만 고치고 전체 완료로 세지 않는다.
- 확장안: 모든 엔진의 스킬 전달 계층 통합. 범위가 커 M/45분 범위를 벗어나므로 이번에는 선택하지 않는다.
- 새 구성 없는 안: 스킬 지원 Claude 경로를 사용. 실제 접근 성공 증거가 필요하고 최초 계약 부재는 해결하지 못한다.
- 현 상태 유지: 현재 소유 범위에서 가능한 유일한 결론이다. 해결안으로 점수화하거나 복구 성과로 계산하지 않는다.
- 인계 1(미착수): 러너 소유 worktree와 최초 계약 출처가 있는지 확인. 없으면 여기서 끝낸다. 새로운 사람 승인 요청을 이 정찰에서 만들지 않는다.
- 인계 2(미착수): registry에서 지정 스킬 경로·상대 참조를 실 엔진에 전달; 실제 자식에서 원본 읽기 실패/성공을 기록. 원본 누락도 차단되는지 확인한 다음 단계로 진행한다.
- 인계 3(미착수): 기존 계약으로 릴리즈 결과 생성 후 위 gate와 Release 테스트 및 실제 회귀를 통과시킨다. 원장에 '수정 과제'로 기록하되 위 3개 수용 기준 전부 통과한 경우에만 완료한다.

## 추정 근거
원본으로 읽은 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration 절차를 적용했다(Skill 도구 호출 성공으로 기록하지 않는다). 소유 범위·계약이 이미 확보됐다는 조건에서 분해 추정은 전달 수정 10~15분, 실제 자식 회귀 15~20분, 기록 5분으로 30~40분이며 알려진 실행 변동 여유는 최대 5분을 별도 둔다. 신뢰도는 낮고 통계적 80% 보장은 아니다. 과거 유사 회차는 모두 no-change라 완료 시간 비교 자료로 부적합하며, 이력 기반 점검 결과 선행 조건 없이 45분 완료를 약속할 근거가 없다. 관리 예비는 미배정이며 소유권·운영 계약 대기 시간은 추정에 숨겨 넣지 않는다.

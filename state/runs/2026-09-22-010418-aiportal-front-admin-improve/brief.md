- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 릴리즈 실패는 앱 빌드 실패가 아니라 스킬 접근 실패와 최초 릴리즈 관례 부재이며, 외부 러너가 이를 앱 수정 과제로 반복 적재하고 있다. 실제 릴리즈 자식의 스킬 접근과 근거 있는 릴리즈 계약을 함께 복구해야 같은 실패를 끝낼 수 있다.
- 수용 기준: 1) 실제 run_agent→run_codex 자식이 registry에서 지정한 릴리즈 스킬 두 원본 및 상대 참조를 읽고 적용한 실행 증거가 있다. 2) 대상 패키지·버전 증가·태그·노트·자산 규칙의 기존 근거 또는 운영 결정이 확인되며 임의 관례 생성 없이 같은 릴리즈 절차가 완료된다. 3) 수정 전 실제 자식 실패와 수정 후 같은 경로 성공을 입증하고, 실제 산출물에 기존 gate.py release와 기존 검증을 실행해 통과한다. 원장의 '수정 과제' 완료 표기는 세 조건을 모두 충족한 뒤에만 한다.
- 건드릴 파일: 현 회차 aiportal-front-admin worktree에서 확인된 원인 수정 대상 없음. 아래 외부 파일은 읽기 근거이며 이 회차의 편집 지시가 아니다: /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex/release_project — 엔진별 스킬 전달 및 실제 릴리즈 경로; aidev/agents/registry.json — 역할·스킬·소유 표면; aidev/release-prompt.md:절차 1~5 — 기존 관례 적용 계약; aidev/tests/test_gate.py:Release 및 tests/test_sim.py:ReleaseSafety — 기존 보조 검증의 한계.
- 검증 명령: 아래 '실행 결과와 인계 검증' 참조. 실패를 성공으로 바꾸는 기존 앱 내부 검증 명령은 확인되지 않았다.
- 위험과 피할 것: 코드·커밋 금지인 정찰 범위를 지킨다. 구현자도 registry builder.surface인 회차 worktree 밖을 임의 편집하지 않는다(COMPANY.md 규칙 1). gate·workflow·skipped 조건 완화, release.json의 수동 성공 처리, 임의 버전·태그·CHANGELOG 생성, 실패 큐 삭제, auth/CI/배포 설정 변경 및 무관한 앱 수정으로 대체하지 않는다. 외부 러너를 앱에 복사하거나 새 우회 스크립트를 앱에 넣지 않는다.
- 차선 후보: 없음 — 지정 실패를 앱 개선 후보로 바꾸지 않는다. 현 소유 범위에서 실행할 수 있는 수정 대상이 없다는 판정을 유지하며, 같은 차단 과제를 완료 가능한 작업으로 재발행하지 않는다.

## 현재 판정과 근거
수정 과제 pending/blocked. 이번 문서는 수정 성공 보고가 아니며, 구현 가능한 앱 패치를 발견했다는 뜻도 아니다. 이전 6회가 막힌 착수 전제는 이번에도 달라지지 않았다. 질문·외부 전달·상태 변경 없이 근거만 이번 회차에 보존한다.

- run.json: project=aiportal-front-admin, base HEAD=01fedba; git status --short 빈 출력, 최근 git log -30 확인.
- Skill 도구는 현재 callable 도구 목록에서 발견하지 못했다. 요청된 정찰 스킬은 /mnt/c/Users/USER/projects/headcount/plugins/{pmo,technology}/skills에서 SKILL.md 원본을 직접 읽었다. Skill 호출 성공으로 기록하지 않는다.
- 릴리즈에 필요했던 marketing/skills/product-launch/SKILL.md와 technology/skills/release-and-deployment/SKILL.md도 현재 로컬에 존재하고 읽을 수 있다. 과거 실패 당시 자식의 실제 파일 접근 가능성은 미확인이다.
- run.sh:run_agent는 Claude용 --plugin-dir와 Skill 안내를 구성하나 run_codex에는 같은 prompt만 넘긴다. run_codex 명령에는 스킬 원본 경로나 대체 읽기 안내를 추가하는 코드가 없다. 이는 전달 누락의 정적 증거이며 과거 실패 전체를 재현한 증거는 아니다.
- .github/workflows 없음. .gitlab-ci.yml은 branch별 legacy build/copy이며 저장된 실패에 job URL/실패 단계 로그가 없다. fixer.sh:52~61은 failed 스냅샷을 앱 이름으로 적재하고 run.sh:1690은 '두 번 실패' 문구를 덧붙인다. 문구 자체가 두 workflow 실행의 증거는 아니다.
- 로컬 태그 없음; root package/package-lock=0.0.0, V2 두 파일=0.1.0. 이번 전체 과거 버전 이력·원격 Release 목록 재조회는 미실행. README/docs/ROADMAP/TESTING_GUIDE 및 V2 deploy README는 빌드·브랜치 정적 배포를 설명하며 최초 릴리즈 관례를 제공하지 않는다.
- release-prompt 절차 5는 버전 파일까지 없는 경우만 skipped다. 현재 패키지 버전을 '최초 릴리즈 결정'으로 해석하지 않는다.

## 접근 비교와 추천
1. 앱 package 버전·release 문서 신설: 근거 없는 최초 관례를 만든다. 제외.
2. gate 또는 workflow를 완화: 지시 위반이며 실제 자식 문제를 고치지 않는다. 제외.
3. 외부 소유 환경에서 실제 엔진에 맞는 스킬 전달을 수정하고 릴리즈 계약에 따라 재검증: 인과적으로 맞는 추천 경로지만 현재 앱 표면에서 실행 불가.
4. 변경 없이 차단 사실 보존: 이번 정찰에서 실행 가능한 선택. 복구 성과로 세지 않는다.
가장 큰 가정: 외부 러너 소유 환경과 승인된 최초 계약이 별도로 확보되어야 한다. 현재 확보 사실은 미확인이다.

## 소유 환경이 확보된 뒤의 순서(현재 착수 지시 아님)
1. 외부 run.sh/registry의 실제 자식 실행을 격리된 출력 경로에서 재현한다. 증거: 실제 자식이 필요한 SKILL.md·상대 참조를 읽었는지와 실패 로그. fake Claude/gh 또는 소스 문자열 검사로 대체하지 않는다. 체크포인트: 원인 재현 실패 시 계획 수정, 다음 단계 진행 금지.
2. registry 기반으로 선택된 엔진이 원본을 발견할 수 있게 전달 경로만 수정한다. 증거: 동일 자식 호출의 수정 전/후 동작 비교. 체크포인트: 다른 엔진·역할 회귀 및 표면 소유 확인.
3. 기존 관례 또는 운영 결정으로 대상 패키지·버전·태그·노트·자산 계약을 확정해 기존 릴리즈 절차를 실행한다. 증거: 실제 결과 파일과 산출물에 아래 동일 gate 적용. 체크포인트: 계약 없이 진행하지 않는다. 본 정찰은 운영 결정을 요청하거나 대신 내리지 않는다.

## 실행 결과와 인계 검증
이번 실행(앱 루트 cwd, 파일 생성 없는 읽기 검증):
- `bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh` → exit 0.
- `bash -n /mnt/c/Users/USER/projects/aidev/bin/fixer.sh` → exit 0.
- `node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json` → exit 0, runtime-config OK.
- `python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-010418-aiportal-front-admin-improve` → exit 1, ok:false/state:failed。저장 결과 거부 재현이며 실제 릴리즈 재실행이 아니다.

소유 환경 확보 후 실제 새 결과에 대한 명령: `python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release "$RELEASE_FILE" --out-dir "$OUT_DIR"` (두 변수는 실제 자식 결과·허용 산출물 디렉터리로 설정).
앱 자체 검증은 V2 cwd `npm ci`, `npm run verify`, `npm run build`지만 현재 node_modules 없고 이번에는 실행하지 않았다. 이 명령들이 스킬 전달·최초 계약 실패를 검증하지는 않는다.
외부 보조 테스트 `python3 -B tests/test_gate.py Release`는 aidev cwd에서 실행하는 기존 테스트지만 /tmp 파일 생성과 파일 핸들 정리 문제가 있고 실제 자식 접근을 검증하지 않는다. tests/test_sim.py와 tests/sim/run_sim.sh는 fake gh/claude 및 /tmp 출력을 사용하므로 이번 미실행이며 수용 기준 1의 대체가 아니다.

## 견적 근거
방법: 읽은 함수 경계에 대한 bottom-up 조건부 견적. 원인 재현 8~10분, 전달 수정 10~15분, 실제 자식 회귀 10~15분: 기본 28~40분; 알려진 실행시간 변동 contingency 5분을 별도로 둬 33~45분. 이는 소유 환경·계약 확보 후 스킬 전달 부분만의 낮은 확신 견적이며 45분 완료 약속이 아니다. 운영 결정 대기·전체 빌드·서버/UAT·배포는 제외하고 management reserve는 배정하지 않는다. 유사 성공 사례가 없고 과거 회차는 모두 blocked라 비교 방식으로 일정 확신을 높일 근거가 없다.
적용 스킬: headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md와 references/sources.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md. 외부 원가 수치나 통계적 신뢰수준을 인용하지 않았다.

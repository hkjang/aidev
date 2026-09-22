- 과제: 수정 과제 — 지정 릴리즈 실패의 스킬 전달 및 최초 릴리즈 입력 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 실패는 앱 빌드 실패가 아니라 릴리즈 에이전트의 스킬 탐색 실패와 최초 릴리즈 관례 부재이며, 같은 실패 스냅샷이 앱 수정 회차에 반복 적재되고 있다. 원본 스킬 전달과 근거 있는 릴리즈 입력을 모두 확보해야 게이트를 유지하면서 실제 릴리즈 검증까지 진행할 수 있다.
- 수용 기준: 1) 실제 run_agent → run_codex 자식이 registry의 release 스킬 두 원본과 상대 참조를 읽었다는 실행 증거가 있고, Claude 경로와 headcount 비활성 동작도 유지된다. 2) root/V2 중 대상, 다음 버전, 태그 형식·유무, 노트 위치·양식, 자산·GitHub Release 사용 여부가 기존 기록 또는 운영 결정에 연결되어 추측 없이 결정된다. 3) 동일 실제 자식 재현을 변경 전/후 실행하여 스킬 접근 실패의 해소를 증명하고, 새 릴리즈 결과와 기존 Release/ReleaseSafety 보호 검증을 통과한다. 기존 failed JSON을 수정하거나 skipped로 바꾸어 통과시키면 불합격이다.
- 건드릴 파일: 현 앱 worktree에는 입증된 수정 대상 없음. 원인 소유 표면은 /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — registry 기반으로 실행 엔진이 읽을 수 있는 원본 경로·참조 기준 경로 전달; 같은 파일 release_project — 근거 있는 릴리즈 입력 연결 확인. /mnt/c/Users/USER/projects/aidev/agents/registry.json — 단계별 스킬의 정본으로 사용(임의 목록 중복 금지). /mnt/c/Users/USER/projects/aidev/tests/test_gate.py:Release 및 tests/test_sim.py:ReleaseSafety — 기존 보호 회귀 유지. 실제 자식 전달 회귀는 아직 없으므로 소유 저장소에서 추가해야 하며 파일명·명령은 현재 미확인. release-prompt.md 절차 2~5를 느슨하게 바꾸지 않는다.
- 검증 명령: 아래 실제 실행 명령과 실행 범위를 참조한다.
- 위험과 피할 것: 정찰은 코드·커밋 변경 금지, 쓰기는 이번 회차 폴더만 허용. 현 앱 구현자가 외부 러너를 편집하는 지시가 아니다. COMPANY.md 규칙 1의 “구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다”와 현재 run.json project=aiportal-front-admin을 확인했다. .gitlab-ci.yml/auth/session/deploy는 이번 실패의 입증된 원인이 아니므로 변경하지 않는다. 최초 버전·태그 관례 창작, 게이트 완화, 릴리즈 비활성화, 실패 큐 삭제, 가짜 에이전트 통과를 실제 복구로 보고하는 접근을 금지한다.
- 차선 후보: 없음 — 지정 실패를 무관한 앱 개선으로 대체하지 않는다. 아래 후보는 향후 회차용이며 이번 구현 착수 지시가 아니다.

착수 판정: pending/blocked. 이전에 반복 실패한 “앱 회차에서 외부 러너를 고치라”는 과제서를 실행 가능 과제로 재발행하지 않는다. 현재 허용 표면과 확보된 입력으로는 3개 수용 기준을 충족할 수 없다. 수정 후 동일 검증 통과 요구는 미충족이다. 아래는 원인 소유 저장소의 회차 및 최초 계약이 확보됐을 때만 실행할 인계 계획이다. 사람이 없는 이번 정찰에서 승인 요청이나 운영 상태 변경은 하지 않는다.

확인한 근거:
- git log -30 및 status: HEAD 01fedba, 깨끗한 작업 트리. 추적 CLAUDE.md/AGENTS.md/CHANGELOG/.github 파일과 로컬 태그 없음. root package.json 0.0.0, V2 0.1.0. 전체 과거 버전 이력은 이전 프로필의 근거이며 이번에는 재실행하지 않음; 원격 최신 Release 미확인.
- README.md, docs/ROADMAP.md, docs/TESTING_GUIDE.md, upgrade/admin-v2/deploy/README.md, 두 package.json, vite.config.ts와 .gitlab-ci.yml을 읽음. CI는 legacy 브랜치 build/copy, V2 문서는 정적 배포/cache 반입이다. 두 workflow 실패를 보여 주는 실행 ID·실패 단계 로그는 확보하지 못했다.
- fixer.sh의 failed 릴리즈 스냅샷 적재와 run.sh:1690의 고정 문구가 연결된다. 이 문구만으로 workflow가 두 번 실행됐다고 주장하지 않는다.
- run.sh:246~335는 Claude의 plugin-dir와 Skill 안내가 붙은 같은 prompt를 Codex에 넘긴다. 이 세션에는 Skill 도구가 없으나 /mnt/c/Users/USER/projects/headcount/plugins 아래 원본은 존재하고 읽을 수 있다. 정적 전달 결함 후보와 실제 이전 자식의 실패 전체 인과는 구분한다.
- release-prompt.md 절차 5는 태그·버전 파일·노트가 모두 없는 경우만 skipped이다. 버전 파일이 있는 현재 저장소에는 적용 불가. 스킬 전달만 고친 뒤 완료로 보고할 수 없다.

조건부 구현 순서(각 단계 증거 없으면 계획 수정, 다음 단계로 넘어가지 않음):
1. 소유 표면 및 최초 계약 확인: aidev 회차인지 확인하고 최초 계약의 근거 파일·결정 출처를 적는다. 체크포인트: 현재는 미충족, 외부 파일 수정 없이 종료. 사람의 새 승인이 필요하다는 임의 규칙을 추가하지 않고 이미 존재하는 결정부터 찾는다.
2. 실제 실패 재현: 실제 격리 환경·registry·run_agent 폴백을 통과하는 자식에서 원본과 references 접근 여부를 관찰한다. 가짜 codex/소스 문자열 검사는 인과 증거로 인정하지 않는다. 현재 이 재현용 공식 명령은 없으며 구현 시 먼저 작성·실행해 명령과 실패 결과를 계획에 고정한다.
3. 최소 전달 수정: Claude plugin-dir 동작을 보존하고 Codex가 읽을 원본 절대 경로와 상대 참조 기준을 전달한다. 활성/비활성 및 직접 Codex 진입 경로도 같은 계약으로 검증한다. 원본 내용을 임의 요약해 대체하거나 스킬 전역 설치·설정 덮어쓰기를 하지 않는다.
4. 새 결과 검증: 2의 동일 명령을 재실행하고 보호 검증 및 근거 있는 새 릴리즈 결과의 gate를 통과시킨다. 태그·원격 게시·서버 UAT는 이 정찰 범위 밖이다. 원장에는 수정 과제와 실제 변경·전후 결과를 적고 세 기준 충족 때만 done 처리한다.

실행한 명령(기준선, 수정 후 회귀 아님):
```bash
cd /mnt/c/Users/USER/projects/aidev
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-102411-aiportal-front-admin-improve python3 -B tests/test_gate.py Release -v
python3 -B bin/gate.py release state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-102411-aiportal-front-admin-improve
cd /home/hkjang/.cache/auto-improve-wt/aiportal-front-admin/upgrade/admin-v2
node scripts/validate-runtime-config.mjs public/config/runtime.json
```
결과: Release 5개 통과(exit 0, ResourceWarning), 저장 실패 gate exit 1/state=failed, runtime config exit 0. validation.json에 명령·cwd·stdout·stderr·exit 보존. 저장 gate 실행은 실패 상태 판정 재현이지 이전 에이전트 실행 재현이 아니다.
소유 환경의 추가 보호 명령: aidev cwd `python3 -B tests/test_sim.py ReleaseSafety -v` (이번 미실행; 가짜 GitHub/에이전트 테스트라 실제 스킬 접근 증명 불가). V2 앱 변경 시 `npm ci` 후 `npm run build`가 typecheck/unit/runtime/offline/integrity를 실행한다(이번 설치·빌드 미실행).

해결안 비교:
- 최소 엔진별 원본 전달: registry를 정본으로 유지하고 결함 표면이 좁아 선택. 전제는 소유 환경에서 실제 자식 접근이 재현되는 것.
- 공통 플러그인 설치/새 배포 파이프라인: 모든 엔진을 포괄하지만 전역 상태·스코프가 커 45분 과제에 부적합.
- 경로를 사람이 찾아 읽기: 이번 정찰은 가능했으나 자동 릴리즈 재현성의 해결은 아님.
- 차단 기록만 반복: 코드 효과가 없어 완료로 인정하지 않음. 현 표면의 차단 판정은 사실 기록일 뿐 대안 구현이 아님.

추정 근거(pmo:estimating-and-contingency): bottom-up으로 실제 자식 재현 10~15분, 전달 수정 10~15분, 회귀·인계 10분 = 30~40분; 알려진 환경 변동의 contingency 5분을 별도로 두어 35~45분. 최초 계약·소유 표면이 이미 확보된 좁은 수정만의 낮은 신뢰 추정이며 통계적 확률은 미산정. 유사 회차는 모두 no-change라 성공 소요시간을 제공하지 못해 analogous 교차 검증 불가. 계약 확보·원격 게시·UAT는 제외하고 별도 재산정, management reserve는 운영자 소유로 임의 배정하지 않는다. 전체 실패 복구가 45분 안에 가능하다는 약속은 하지 않는다.

적용 스킬: Skill 도구 부재를 도구 목록에서 확인하고 다음 원본을 직접 읽었다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
릴리즈 두 스킬도 존재·내용을 확인했다. 스킬이 이번 차단이나 승인 요청을 요구한 것이 아니라 사용자 절대 규칙·소유 표면·릴리즈 계약 부재에 따른 판정이다.

- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장된 실패는 스킬 접근 및 최초 릴리즈 계약 부재로 중단한 에이전트 결과이며, 이번에 확인한 앱 CI 실패 로그가 아니다. 실제 실패 경로를 복구해야 하며 앱 CI나 릴리즈 게이트를 느슨하게 바꿔서는 해결되지 않는다.
- 수용 기준: 1) 실제 release_project → run_agent → run_codex 경로의 자식이 명부의 두 릴리즈 스킬 원본과 필요한 상대 참조를 읽었다는 실행 증거가 있다. 2) 대상 패키지·다음 버전·태그·노트·자산의 근거 있는 최초 릴리즈 계약으로 동일 실패를 재현하고 수정 후 통과한다. 3) 원인 회귀 검증과 기존 Release/ReleaseSafety 검사를 모두 만족하고, 기존 실패·누락 자산·범위 밖 자산 차단은 유지한다. Fake CLI나 소스 문자열 검사만으로 1)을 충족하지 않는다.
- 건드릴 파일: 현재 앱 worktree 안에서 확인된 원인 수정 파일은 없음. 아래 외부 경로는 진단 근거이며 이번 구현자에게 편집을 지시하는 목록이 아니다: /mnt/c/Users/USER/projects/aidev/bin/run.sh: agent_plugin_args/dept_note/run_agent/run_codex/release_project — Claude 전용 스킬 안내를 Codex에 그대로 넘김; aidev/release-prompt.md: 절차 2·3·5 — 관례 추종 및 신규 관례 금지; aidev/tests/test_gate.py: Release — 결과/자산 검사; aidev/tests/test_sim.py: ReleaseSafety — 모의 게시 안전성 검사. .gitlab-ci.yml은 읽기만 했으며 본 실패의 수정 대상으로 선정하지 않았다.
- 검증 명령: 아래 실제 실행 결과와 명령 참조. 현재 저장 결과의 gate 실패가 기준선이며, 이를 억지로 통과시키거나 이전 테스트 통과를 복구 증거로 쓰지 않는다.
- 위험과 피할 것: 코드 변경·커밋 금지인 정찰 범위를 준수한다. 구현자의 표면은 현 앱 worktree다(registry.json builder.surface, COMPANY.md 규칙 1). 외부 러너 패치, 앱 안에 러너 복제, 가짜 스킬 shim, 무근거 버전/태그/CHANGELOG 신설, failed→skipped/released 변경, 보호 경로(auth/workflows) 변경을 하지 않는다. 동일 차단 과제의 재발행이나 문서 작성만을 수정 성과로 세지 않는다.
- 차선 후보: 없음 — 지정 실패와 무관한 보류 앱 개선으로 대체하지 않는다. 1순위의 전제가 성립하지 않는 현 상태는 구현 가능한 과제가 없다는 판정이다.

판정: pending / blocked. 이번 회차에서 착수 가능한 수정 과제는 발견하지 못했다. 위 제목은 자동 지정 과제의 추적 식별자이며 과거에 기각된 외부 러너 수정안을 실행 가능한 앱 작업으로 다시 발행한 것이 아니다. 수용 기준은 현재 모두 미충족이다. 구현자는 같은 파일 탐색을 반복하거나 무변경 기록을 성공으로 처리하지 말고 이 판정을 원장에 '수정 과제 — 미완료: 실행 가능한 원인 수정 대상 없음'으로 남긴다.

확인한 증거:
- 앱 HEAD 01fedba, git log -30 확인, git status --short 빈 출력. 루트 CLAUDE.md/AGENTS.md 및 .github/workflows 없음. 로컬 태그 목록 비어 있음. 전체 태그 원격 조회·전체 버전 이력 재검증은 이번에 미실행.
- README.md, docs/ROADMAP.md, docs/OPERATIONS_RUNBOOK.md, docs/TESTING_GUIDE.md, tests/README.md, package.json 두 개와 lockfile 버전, V2 vite.config.ts, playwright.config.ts 확인. TODO/FIXME 검색(src, V2 src, docs)은 출력 없음.
- .gitlab-ci.yml은 브랜치별 legacy 빌드/정적 파일 복사다. dev_build_main의 core 모드와 중복 rules는 알려진 별도 후보이며 본 실패 로그와 연결된 증거 없음. 비밀값은 복사하지 않았다.
- /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json은 status=failed, tag/version 빈 문자열. fixer.sh:52~61은 이 snapshot을 큐에 적재한다. run.sh:1688~1690은 큐 항목에 두 번 실패 문구를 무조건 붙인다. 실제 두 번 실행한 workflow ID/실패 단계는 미확인이다.
- run.sh:246~259, 283~337: Claude는 plugin-dir/Skill을 받지만 Codex에는 같은 prompt를 넘기고 원본 경로 안내를 추가하지 않는다. 이것은 정적 배선 확인이지 과거 자식의 전체 접근 실패 원인 입증이 아니다.
- root package/lock 0.0.0, V2 package/lock 0.1.0. release-prompt.md 절차 5는 '태그도, 버전 파일도, 릴리즈 노트도 없음'일 때만 skipped이며 '관례를 새로 정하는 건 사람의 일입니다'라고 명시한다. 스킬을 찾았다는 사실만으로 이 별도 계약 공백이 해소되지는 않는다.
- 현재 사용 가능한 도구 이름/설명에서 Skill 호출 도구를 찾지 못했다. 정찰 3개와 릴리즈 2개 SKILL.md의 실제 로컬 원본은 읽었다. 사용자에게 질문하지 않았으며 Skill 호출 성공이라고 보고하지 않는다.

실행한 검증(앱 루트 cwd):
```bash
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-021419-aiportal-front-admin-improve
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-021419-aiportal-front-admin-improve python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
```
- 각각 exit 1(ok:false/state:failed), exit 0(5 tests OK; 기존 unclosed file ResourceWarning 있음), exit 0. 테스트 임시 파일은 지정 회차 폴더에만 생성했다.
- Release 테스트는 기존 gate 동작의 기준선이다. 실제 릴리즈 자식 재현·수정 후 테스트가 아니다. sim/run_sim.sh는 /tmp 임시 폴더와 fake gh/claude를 사용하므로 이번에는 실행하지 않았다.
- V2 node_modules 없음. 설치/build/Vitest/UAT는 미실행. 앱 표준 명령은 V2 cwd에서 npm ci 후 npm run build(verify 포함)이지만, 현 정찰의 쓰기 제한 밖에 파일을 만들며 지정 실패를 검증하지 않으므로 실행하지 않았다.

실행 계획 및 체크포인트:
1. [완료] 실제 실패 snapshot, 소유 범위, CI/러너/스킬 경로 대조. 증명: 위 gate 명령은 저장된 failed를 계속 차단하며 run.json project는 aiportal-front-admin이다. 사람 확인을 요청하지 않았다.
2. [차단] 변경 전제: 외부 러너를 다룰 수 있는 소유 회차와 기존 운영 결정에 근거한 최초 릴리즈 계약. 현재 둘 다 확보되지 않았다. 본 정찰은 외부 회차 생성·이관·정책 변경을 하지 않는다.
3. [미착수] 전제가 다른 소유 회차에서 충족될 때 그 환경에서 실제 자식 실패를 재현하고 배선만 최소 수정한 후 동일 실행을 통과시킨다. 아직 존재하지 않는 회귀 명령을 실행 가능한 명령으로 꾸며 적지 않는다. 증명 명령과 기대 결과를 그 회차 과제서에 먼저 확정해야 한다.
4. [미착수] 기존 ReleaseSafety와 수정 전후 실제 자식 증거를 검토하고 원장을 완료로 바꾼다. 앱 자체 검사만으로 수용 기준을 대체할 수 없다.

대안 비교(technology:solution-exploration):
- 최소 변경: 소유 러너 회차에서 명부 기반 원본 안내를 전달. 배선 문제에 맞지만 이번 표면 밖이고 최초 계약은 별도다.
- 확장 설계: 엔진별 스킬/권한 어댑터 구축. 검증 범위가 커 45분 과제에 맞지 않는다.
- 새 구성 없이 직접 원본 읽기: 정찰에서는 성공했지만 다음 자식에 지속되는 수정이 아니고 최초 계약도 해결하지 않는다.
- 현 판정 유지: 지정 실패를 pending으로 보존하고 실행 불가를 명시한다. 현재 추천은 이것이며 수정 완료를 의미하지 않는다. 무근거 관례 생성이나 게이트 완화는 후보에서 제외한다.

추정(pmo:estimating-and-contingency):
- M은 소유 환경과 계약이 이미 준비된 최소 배선 수정만의 조건부 규모다. 전체 복구를 45분에 완료할 수 있다는 주장은 하지 않는다.
- bottom-up: 실제 자식 재현 10~15분, 최소 배선 변경 10~15분, 실제 자식/기존 게이트 회귀 10~15분, 원장 5분 = 기본 35~50분. 알려진 환경 변동 contingency 5~10분 별도, 합계 40~60분; 통계적 신뢰도 미산정, 판단 확신 낮음. 전제 대기와 원격 게시/UAT는 제외, management reserve는 배정하지 않음.
- 유사 사례 교차 확인: 과거 여섯 회차가 no-change이므로 전제 없는 45분 추정은 지지되지 않는다. 새 관례 결정을 이 범위에 숨겨 포함하지 않는다.
- 적용 원본: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md. 외부 문헌의 수치나 신뢰수준은 차용하지 않았다.

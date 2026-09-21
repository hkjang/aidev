# 수정 과제 이관 — 미완료

대상은 /mnt/c/Users/USER/projects/aidev이며 앱 변경으로 해결할 수 없다. 현재 회차 run.json은 aiportal-front-admin이고 COMPANY.md:15 및 :32, agents/registry.json의 builder.surface는 해당 회차 worktree를 쓰기 표면으로 정한다. 파일시스템 접근 가능 여부와 역할의 쓰기 표면은 별개로 판단했다. 사용자 과제서의 권한 부족 시 차선 지시에 따라 이 기록만 남기며 외부 메시지 전송은 하지 않았다.

## 확인한 근거
- bin/run.sh:245~259: agent_plugin_args는 Claude plugin-dir을 만들고 dept_note는 Skill 도구 사용을 지시한다.
- :290, :308, :314~327: Claude용 안내가 합쳐진 prompt를 run_codex로 넘기며 HEADCOUNT_DIR/원본 경로/상대 참조 기준을 전달하지 않는다.
- release 역할의 marketing:product-launch 및 technology:release-and-deployment 원본은 headcount/plugins/<department>/skills/<skill>/SKILL.md에 실제 존재하며 읽었다. 각 references/sources.md도 파일 목록에 존재한다. 이는 이번 세션의 접근 확인이며 실제 폴백 자식 프로세스의 접근을 검증한 것은 아니다.
- tests/test_sim.py와 tests/sim/run_sim.sh를 읽었다. 기존 harness는 가짜 gh/claude와 /tmp 임시 경로를 사용하고 Codex 전달 검증은 없다. 이번에는 실행하거나 수정하지 않았다.
- 앱 HEAD 01fedba, git status --porcelain 출력 없음, git tag 출력 없음. git log --all -G '"version"' -- package.json upgrade/admin-v2/package.json 결과는 81b150f, e7bded2의 최초 추가뿐이다. 루트 version 0.0.0/V2 0.1.0 확인.
- release-prompt.md 절차 2~3은 기존 증가·태그·노트 관례를 요구하며 절차 5는 버전 파일까지 없어야 skipped를 허용한다. 최초 릴리즈 정책은 운영 결정이 필요하며 이번에 만들지 않았다.

## 러너 소유 작업 환경의 다음 작업
1. 실제 run_agent → Codex 폴백 전달과 자식 프로세스의 원본 읽기를 통과하는 실패 테스트를 먼저 작성·실행한다. 성공 JSON만 반환하는 가짜 에이전트나 소스 문자열 검사로 대체하지 않는다.
2. registry를 정본으로 release/scout 포함 모든 역할에 절대 원본 경로와 상대 참조 기준을 전달한다. Codex용 안내에서 존재하지 않는 Skill 호출 요구를 없애고 Claude plugin-dir 동작은 유지한다.
3. 원본 누락 시 명시적 실패, 상대 리소스 읽기, 전역 설정 무변경을 검증한다. 결함을 복원하면 같은 테스트가 다시 실패하는지 확인한다.
4. Release·CI·비밀검사·자산 경로 조건은 수정하지 않는다. tests/test_sim.py 전체와 추가 통합 검증을 실행한다. 실제 통과 전에는 복구 완료라고 기록하지 않는다.

## 이번 실행 결과
- bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh: exit 0.
- PYTHONDONTWRITEBYTECODE=1 TMPDIR=<이 회차 디렉터리> python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release: 5 tests / OK / exit 0.
- 같은 환경에서 test_gate.py 전체: 27 tests / OK / exit 0. 기존 test_gate.py:104의 ResourceWarning 있음.
- python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-102422-aiportal-front-admin-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-102422-aiportal-front-admin-improve: exit 1, JSON ok:false/state:failed. 이것은 차단 재현이며 릴리즈 성공이 아니다.
- 신규 테스트·프로덕션 수정·수정 후 통합 검증·앱 빌드·실제 서버/UAT·커밋 없음. 전역 설정 변경 없음. 스킬 전달 복구 및 전체 릴리즈 모두 미완료.

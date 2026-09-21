- 과제: 수정 과제 — 자동 배정된 릴리즈 실패의 실제 원인 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 외부 aidev의 run_agent가 Claude 전용 Skill 안내를 Codex에 전달하지만 원본 스킬 경로는 전달하지 않으며, 이 앱에는 최초 릴리즈 버전·태그 정책 근거도 확인되지 않는다. 두 원인을 해결해야 같은 실패를 반복하지 않고 실제 릴리즈 검증을 통과할 수 있다.
- 수용 기준: 1) 실제 run_agent → 사용량 제한 → run_codex 경로에서 registry에 지정된 모든 스킬 원본과 상대 참조를 자식 프로세스가 읽는다. 2) 출처가 있는 최초 릴리즈 운영 결정으로 버전 대상·태그·노트·자산 계약이 확정되고 그에 따른 실제 릴리즈 결과가 기존 검증을 통과한다. 3) 수정 전 실패/수정 후 성공 회귀 검증과 기존 Release 안전성 테스트가 함께 통과하며 원장에 '수정 과제'로 기록한다. 기존 failed 결과는 계속 차단되어야 한다.
- 건드릴 파일: 현재 앱 worktree 안에는 확인된 수정 대상이 없다. 원인 파일은 외부 /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args,dept_note,run_agent,run_codex(241–340행) — 엔진별 스킬 원본 접근 전달; 외부 tests/test_sim.py:run,Agents,ReleaseSafety 및 tests/sim/run_sim.sh,tests/sim/bin/claude — 실제 러너 경유 폴백 회귀 검증. 새 테스트 경로와 함수명은 미정이며 현재 존재한다고 주장하지 않는다. release-prompt.md 절차 5는 읽기 근거이며 완화 대상이 아니다.
- 검증 명령: 아래 '확인된 명령과 결과' 참조. 앱의 npm run build는 이 외부 전달 결함이나 최초 릴리즈 정책을 검증하지 못한다.
- 위험과 피할 것: 앱 .gitlab-ci.yml, auth/, router.ts, 버전 파일을 우회 수정하지 않는다. 외부 원본 스킬 복제·임의 버전 증가·태그 신설·skipped 조건 확대·게이트 완화·failed 결과 덮어쓰기 금지. 과거 이관 문서를 새 성과로 다시 만들거나 '중지했다'를 구현 성공으로 세지 않는다.
- 차선 후보: 없음 — 우선 과제 강제 지정으로 앱의 다른 개선으로 대체하지 않는다. 외부 소유 작업 착수 권한과 최초 릴리즈 운영 결정이 없으면 수용 기준을 달성할 수 없다.

실행 가능성 판정: blocked. 이번 run.json의 project는 aiportal-front-admin이고 agents/registry.json의 builder.surface는 '그 회차의 worktree'다. COMPANY.md 15행과 32행도 같은 경계를 명시한다. 현재 정찰은 사용자 절대 규칙에 따라 run 디렉터리 산출물만 쓴다. 외부 러너 편집 권한이나 최초 릴리즈 결정을 새로 얻었다는 증거가 없다. 따라서 현재 제약 아래에서 45분 내 완료 가능한 수정 과제를 꾸며낼 수 없다. 이 문서는 기존 '이관/중지' 접근을 다시 실행하라는 과제가 아니라 지정 작업의 미충족 조건과 실제 구현 계약이다. 이번 정찰에서 코드 수정·복구 통과는 이루어지지 않았다.

확인된 원인과 범위
- run.sh:290은 Claude plugin이 있으면 dept_note를 프롬프트에 붙인다. :308은 같은 프롬프트로 run_codex를 호출하며, run_codex의 env -i / codex exec에는 HEADCOUNT_DIR나 원본 스킬 경로 전달이 없다. 실제 모델 자식이 실패하는 과정은 이번에 실행하지 않았으므로 정적 배선 근거와 런타임 입증을 구분한다.
- 요청된 정찰 스킬 세 개와 릴리즈 스킬 두 개의 원본은 /mnt/c/Users/USER/projects/headcount/plugins/<department>/skills/<name>/SKILL.md에 있다. Skill 호출 도구는 사용 가능한 도구 이름 검색에서 발견되지 않았다. 원본을 직접 읽는 방식으로 정찰 스킬을 적용했다.
- fixer.sh:53–61은 status=failed인 release.json을 큐에 넣고 run.sh:1688–1690은 원인과 무관하게 '워크플로 두 번 실패' 문구를 붙인다. 실제 CI 실패 횟수·실패 step 로그는 미확인이다. 이 문구를 근거로 .gitlab-ci.yml을 고치지 않는다.
- .github 없음, 로컬 태그 없음, 현재 root 0.0.0/V2 0.1.0, 릴리즈 노트 파일 검색 결과 없음. docs/OPERATIONS_RUNBOOK.md와 upgrade/admin-v2/deploy/README.md는 브랜치 정적 배포를 설명한다. 전체 과거 버전 불변 이력은 이번에 재검증하지 않았으며 이전 기록의 판단과 구분한다.

소유 범위와 정책이 확보된 이후의 구현 순서(현재 모두 미착수)
1. 격리된 실제 러너 실행으로 폴백 실패를 재현한다. tests/sim/run_sim.sh는 현재 /tmp에 쓰므로 이번 정찰의 쓰기 범위에서는 실행하지 않았다. 프로덕션 run_agent/run_codex를 실행하며 Claude·Codex 실행 파일 경계의 테스트 자식이 실제 파일을 읽도록 한다. 소스 문자열 검사만으로 성공을 판정하지 않는다. 체크포인트: 재현 실패 확인 후에만 구현; 별도 사람 검토는 추가하지 않는다.
2. registry의 phase별 skills를 원천으로 Codex용 경로 안내를 전달하고 상대 참조 접근도 검증한다. Claude 플러그인 경로는 유지한다. 직접 Codex 리뷰 진입점(run.sh:773)도 같은 스킬 해석 계약인지 확인한다. 체크포인트: 폴백·직접 진입·원본 누락의 결과 확인, 수정 되돌림 시 신규 회귀가 실패함을 확인한다.
3. 최초 릴리즈 정책은 운영자의 명시적 결정이 선행되어야 한다(release-prompt.md 절차 5: '관례를 새로 정하는 건 사람의 일입니다'). 지금은 그 결정이 미확보다. 결정 내용으로 실제 릴리즈 검증을 수행하고 폴백 복구만 끝났다면 전체 릴리즈 성공으로 기록하지 않는다. 이 단계는 임의로 통과시키지 않는다.

확인된 명령과 결과(2026-09-21 이번 정찰 실행)
```bash
TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-142408-aiportal-front-admin-improve PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release
bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh
bash -n /mnt/c/Users/USER/projects/aidev/bin/fixer.sh
PYTHONDONTWRITEBYTECODE=1 python3 /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-142408-aiportal-front-admin-improve
```
Release 5개 통과(ResourceWarning 있음), bash 구문 검사 통과, 마지막 명령 exit 1 / ok:false / state:failed. 마지막 실패는 보존해야 하는 정상 차단이며 수정 후 성공 증거가 아니다. tests/test_sim.py의 ReleaseSafety는 읽었으나 실행하지 않았다. 앱 node_modules 없음; 설치·빌드·서버/UAT·실제 모델 폴백 미실행.

대안 비교와 추정 근거
- 최소 수정: 외부 러너의 엔진별 스킬 전달 복구. 앱 범위 밖이고 정책 차단이 별도로 남는다.
- 확장 수정: 실패 소유자/실패 종류를 구조화해 큐 적재와 프롬프트 생성이 함께 읽게 한다. 반복 배정 문제에는 효과가 있지만 릴리즈 성공을 대신하지 못하며 이번 범위로 늘리지 않는다.
- 새 구성 없이 앱 문서에 스킬 절대 경로를 적는 방안: 로컬 경로 의존과 릴리즈 정책 부재를 남겨 채택하지 않는다.
- 기존 상태 유지: 복구는 아니지만 현 권한에서 허용되는 판정이다. 이관/중지만 반복한 세 회차를 성공 사례로 사용하지 않는다.
- M은 소유 범위가 확보된 뒤 스킬 전달 수정 부분만의 조건부 추정이다. bottom-up: 재현 10–15분 + 구현 10–15분 + 회귀 검증 10분 = 30–40분, 알려진 격리 환경 편차 예비 0–5분. 신뢰도 낮음(실제 폴백 테스트 없음), 운영 결정 대기·전체 릴리즈·관리 예비는 미포함/산정 불가. 전체 과제가 45분 안에 끝난다는 추정은 하지 않는다. 유사 과거 회차는 모두 미완료여서 기간 추정 근거로 쓸 수 없다.

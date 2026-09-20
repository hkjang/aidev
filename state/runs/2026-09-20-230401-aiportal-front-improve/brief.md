- 과제: [수정 과제] Codex 폴백에 headcount 스킬의 실제 파일 경로 전달 복구 (가치 5 / 위험 2 / 작업량 M)
- 왜: 외부 aidev 러너의 run_agent는 Claude용 Skill 호출 안내를 붙인 프롬프트를 run_codex에 그대로 전달하지만, Codex 호출에는 플러그인과 SKILL.md 경로가 전달되지 않아 실제 존재하는 필수 스킬을 찾지 못한다. 엔진에 맞게 동일 원문을 읽을 수 있도록 전달하고 폴백 회귀 검증을 추가하면 스킬 부재 오판을 제거하면서 기존 릴리즈 게이트를 유지할 수 있다.
- 수용 기준: 1) Claude 사용량 제한으로 Codex 폴백할 때 release의 2개, scout의 3개 스킬이 registry 기준으로 실제 읽을 수 있는 절대경로와 함께 전달되고 Codex에 없는 Skill 도구 호출을 강요하지 않는다. 2) HEADCOUNT_DIR 재정의·공백 경로·NO-HEADCOUNT·agents.headcount=false가 기존 의도대로 동작하며, 활성 상태에서 파일이 빠졌으면 누락 이름/경로를 명시하고 성공으로 위장하지 않는다. 3) 가짜 claude/codex를 사용하는 회귀 테스트가 수정 전 누락으로 실패하고 수정 후 통과하며, 기존 27개 gate 테스트와 전체 흐름 테스트가 유지된다. 4) aiportal-front의 버전 모순은 별도 미해결 장애로 남기고 실제 릴리즈 성공·skipped를 조작하지 않는다.
- 건드릴 파일: **현재 aiportal-front에는 구현 대상 파일이 없다. 아래는 다른 저장소 aidev의 이관 대상이며 이 회차 worktree에서 직접 수정하지 않는다.** /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex — 공통 registry에서 스킬 목록을 구하되 엔진별 안내와 원문 경로를 전달; /mnt/c/Users/USER/projects/aidev/tests/test_sim.py:run 및 Agents — 폴백 시나리오/검증 추가; /mnt/c/Users/USER/projects/aidev/tests/sim/run_sim.sh — 격리된 headcount fixture와 도구 스텁 경로; /mnt/c/Users/USER/projects/aidev/tests/sim/bin/claude — 사용량 제한 응답 시나리오; tests/sim/bin/codex(신규) — 전달 프롬프트와 원문 접근 확인, 실 모델 호출 금지. tests/test_gate.py와 bin/gate.py는 검증 대상으로만 유지.
- 검증 명령: aidev 루트에서 `bash -n bin/run.sh`, `PYTHONDONTWRITEBYTECODE=1 python3 tests/test_gate.py` (이번 27건 통과), `PYTHONDONTWRITEBYTECODE=1 python3 tests/test_sim.py Agents` 및 `PYTHONDONTWRITEBYTECODE=1 python3 tests/test_sim.py` (기존 명령, 이번 미실행; 전체는 시나리오당 10~40초로 오래 걸릴 수 있음). 신규 Codex 회귀는 실제 등록한 테스트 이름으로 단독 실행 후 전체 실행한다. aiportal-front의 `npm test`/`npm run build:dev`는 앱 검증일 뿐 이 스킬 전달 결함의 검증이 아니다.
- 위험과 피할 것: aiportal-front 범위의 구현자에게 외부 저장소 수정 권한이 있다고 가정하지 말 것. 이관 전이면 원장에 외부 원인/구현 차단으로 남기고 앱 변경·빈 커밋을 만들지 않는다. release-prompt.md 절차 5를 느슨하게 하거나 0.0.0을 무조건 skipped로 취급하지 않는다. 버전·태그·초기 릴리스 날짜를 임의로 변경하지 않는다. .gitlab-ci.yml, auth, migrations, 원격 전송·실배포, headcount 원문, gate 판정은 수정 대상 아님. 개인 CODEX_HOME에 스킬을 설치하는 방식은 다른 회차 오염을 낳으므로 피한다.
- 차선 후보: 같은 실패의 스킬 경로·버전 증거를 회차 원장에 남겨 aidev 담당으로 이관 — 외부 수정 범위가 확보되지 않을 때의 차선이며, useAppList 등 무관한 앱 수정으로 바꾸지 않는다.

확인한 근거와 한계
- 기준 HEAD dd65af7, 작업 트리 깨끗함. CLAUDE.md/AGENTS.md는 앱 저장소에 없음. 최근 git log -30, README, docs의 아키텍처/빌드/CI/테스트/개선 로드맵, TODO/FIXME 검색, package.json/vitest.config.js/vite.config.js 확인.
- 실제 실패 기록: ../aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json 및 agent-release.txt. release-context.md의 GitHub Release/워크플로 목록은 비어 있음. 이는 제공된 스냅샷이며 실시간 원격 조회는 하지 않았다.
- ../aidev/bin/run.sh:245의 HEADCOUNT_DIR, 246 agent_plugin_args, 254 dept_note, 286 run_agent, 314 run_codex를 읽었다. run_codex에는 HEADCOUNT_DIR/SKILL.md/agent_plugin_args 참조가 없고 $prompt를 그대로 전달한다. tests/sim/bin에는 gh/claude만 있으며 Codex 스텁이 없다.
- 실제 원문은 /mnt/c/Users/USER/projects/headcount/plugins/<department>/skills/<skill>/SKILL.md에 있다. Skill 도구는 제공 목록에 없어 호출하지 못했지만, 파일 읽기로 요청한 3개 정찰 스킬과 2개 릴리즈 스킬을 모두 확인했다. 기존 실패의 “로컬에 없음” 결론은 검색 범위 누락이다. 원문은 별도 JSON 반환 형식을 요구하지 않는다.
- 버전은 package.json, package-lock.json 최상위 및 packages[""] 모두 0.0.0, README.md:438과 docs/01-시작하기.md:359는 0.0.1(2025-01-01). 전체 이력 shallow=false, 태그 없음. release-prompt.md는 새 관례를 임의로 정하지 말라고 명시한다. 원문 스킬은 다음 버전을 결정해 주지 않는다. 따라서 경로 전달 수정만으로 릴리즈 전체가 성공한다는 보장은 없다.
- “같은 원인으로 두 번 실패”는 자동 배정 설명이다. 직접 읽은 09-19 결과는 skipped, 09-20 결과는 failed이므로 동일한 두 실패를 독립 확인하지 못했다. 09-19의 skipped를 정답으로 복제하지 않는다.
- .gitlab-ci.yml은 main/develop의 망별 빌드 및 PVC 복사이다. 이 파일의 실제 잡 실패 로그나 실패한 앱 테스트는 제시되지 않았다. 배포 스크립트는 실행하지 않았다.

실행 계획 (다음 구현 세션, 모든 단계 pending)
1. 작업 표면 확인: aidev 별도 작업 회차로 이관된 경우에만 그 저장소 지침과 현재 HEAD를 재확인한다. 이관되지 않았다면 ledger-entry.md에 수정 과제/외부 원인/미수정으로 기록하고 종료한다. 사람 질문 대신 권한 경계를 결과에 남기는 체크포인트다.
2. 재현: 테스트용 headcount 원문 fixture와 quota를 반환하는 claude, argv를 기록하는 codex 스텁으로 run_agent → run_codex를 실행한다. Codex 프롬프트에 원문 경로가 없다는 실패를 먼저 확인한다. 스텁 자체가 원문을 자동 주입하면 안 된다. 검증 후 다음 단계 진행; 사람 승인 체크포인트 없음.
3. 최소 수정: registry의 동일 스킬 집합에서 엔진별 안내를 생성한다. Claude에는 기존 플러그인 로드를 유지하고 Codex에는 로컬 원문 경로와 읽기 지침을 전달한다. 직접 run_codex를 호출하는 review 경로도 빠뜨리지 않는다. 오프 스위치와 파일 누락, 공백 경로를 테스트한다. 검증 후 다음 단계 진행.
4. 기존 gate 및 sim 검증을 수행하고 원장에 '수정 과제'와 수정 전/후 증거를 적는다. 스킬 전달 성공과 버전 판정 미해결을 분리 기록한다. 실 모델/실 릴리즈 재시도는 이 회귀 테스트 범위에 포함하지 않는다.

대안 비교 및 추정
- 선택: 엔진별 원문 경로 전달. 실제 존재하는 원문을 재사용하고 지속적인 검색 실패를 해결한다.
- 원문 전체를 프롬프트에 인라인: 경로 권한 문제는 줄지만 참조 파일/업데이트/프롬프트 크기 부담이 있어 후순위.
- 개인 환경에 플러그인 설치: 회차별 격리와 배포 재현성이 약해 제외.
- 매번 정찰이 절대경로를 찾아 노트로 전달: 이번 즉시 진단에는 유효하지만 러너 결함이 남는다.
- 추정 근거: bottom-up으로 fixture/실패 재현 8~10분, 전달 수정 8~12분, 표적·기존 검증 8~12분, 원장 3분 = 27~37분; 알려진 변동(공백 경로/검증 지연) contingency 5~8분을 별도 추가해 32~45분. 주관적 중간 신뢰, 통계적 확률 추정 아님. 별도 관리 예비는 배정하지 않는다. 전체 sim이 이 범위를 넘으면 검증을 줄이지 말고 미완료로 기록한다. 비교 가능한 엔진 전달 변경 이력이 없어 유추 견적은 미확인이다.
- 가장 큰 전제: 이관된 aidev 구현 세션이 외부 러너를 수정할 수 있다는 것. 이 전제가 없으면 이번 앱 회차 내 완전 복구는 불가능하다.
- 적용 스킬: headcount의 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration. sources.md도 읽었으며 외부 비용·법률·스토어 정책 주장은 하지 않았다. release-and-deployment/product-launch는 장애 진단 목적으로 읽었고 실제 출시 절차를 수행한 것은 아니다.

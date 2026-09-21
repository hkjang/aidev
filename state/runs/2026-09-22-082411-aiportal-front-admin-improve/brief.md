- 과제: 수정 과제 — 지정 릴리즈 실패 복구: 엔진별 스킬 전달과 최초 릴리즈 계약 연결 (가치 4 / 위험 2 / 작업량 M)
- 왜: 실제 실패 원장은 스킬 접근 불가와 최초 릴리즈 관례 부재를 함께 기록하며, 현재 앱에는 그 원인 후보인 릴리즈 러너 코드가 없다. 두 조건을 각각 증명하고 소유 환경에서 복구해야 같은 앱 회차의 no-change 반복과 허위 성공 판정을 막을 수 있다.
- 수용 기준: 1) Claude Skill 경로와 Codex 직접 파일 읽기 경로가 명부에 등록된 동일 원본 및 상대 참조를 실제 자식 환경에서 읽고, 원본이 없으면 구체적인 실패를 반환한다. 2) 출처가 있는 최초 대상 패키지·버전 증가·태그·노트·자산 계약으로 릴리즈 계획을 만들며, 계약이 없으면 기존 차단을 유지한다. 3) 동일 자식 재현의 수정 전 실패/수정 후 성공과 기존 Release·ReleaseSafety 보호 검증을 남기고, 앱의 실제 검증 및 기존 release gate까지 통과해야 복구 완료로 인정한다.
- 건드릴 파일: 현 aiportal-front-admin worktree에는 확인된 원인 수정 파일 없음. 원인 후보 소유 경로는 /mnt/c/Users/USER/projects/aidev/bin/run.sh: agent_plugin_args/dept_note/run_agent/run_codex — registry 기반 원본 경로·엔진별 읽기 안내 전달; 같은 파일 release_project — 확정된 계약 입력 연결; /mnt/c/Users/USER/projects/aidev/tests/test_sim.py: run/ReleaseSafety — 실자식 접근 회귀와 보호 조건 보존. /mnt/c/Users/USER/projects/aidev/release-prompt.md 절차 2~5는 계약 근거 확인 대상이며 skipped 조건 완화 대상이 아니다. 최초 계약의 실제 저장 위치·내용은 미확인.
- 검증 명령: 아래 실행 기록과 단계별 증명을 따른다. 앱 검증은 node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json. 소유 저장소의 보호 검증은 python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v. 전체 복구 검증에는 python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_sim.py ReleaseSafety -v 및 확정 대상 패키지의 빌드가 추가로 필요하다(이번 미실행).
- 위험과 피할 것: 코드를 바꾸지 않는 정찰 역할을 유지한다. COMPANY.md 규칙 1의 구현 표면은 그 회차 worktree이므로 이 과제서는 외부 aidev 편집 허가가 아니다. auth/migrations/.gitlab-ci.yml/.github/workflows, gate.py 성공 조건, release.json 저장 상태를 바꿔 우회하지 않는다. 임의 버전·태그·CHANGELOG 생성, 버전 파일 삭제로 skipped 만들기, unrelated UI 작업으로 대체하기 금지.
- 차선 후보: 없음 — 자동 배정된 실패이므로 별도 기능을 선택하지 않는다. 이 문서의 소유 환경 수정안은 이관 참고이며 현 앱 구현자가 착수할 수 있는 차선으로 재승인하지 않는다.

착수 판정 및 이전 회차와의 관계

상태는 pending/blocked이다. 현 앱 회차의 45분 구현 과제로 성립한다는 근거가 없으므로 반복된 ‘착수 가능성 판정’을 새 개선 성과로 제출하지 않는다. 사용자의 코드 변경 금지와 출력 경로 제한 때문에 정찰이 직접 수정·수정 후 성공을 수행할 수 없다. 해당 요구는 미충족으로 명시하고 소유 저장소에서 사용할 구체적인 구현 조건만 남긴다. 질문·정책 변경·원격 전송은 하지 않았다.

확인한 근거

- HEAD 01fedba, git log -30 확인, git status --short 빈 출력. 추적 CLAUDE.md/AGENTS.md/.github/CHANGELOG 없음. README, docs/ROADMAP.md, OPERATIONS_RUNBOOK.md, TESTING_GUIDE.md, tests/README.md, 두 package.json과 검사 스크립트 확인.
- .gitlab-ci.yml은 브랜치 build/copy이며 dev_build_main의 build:core 및 중복 rules 유지. 이번 실패를 해당 CI 잡 실패로 입증한 로그는 없다. 자격증명 값은 기록하지 않는다.
- run.sh:246~335에서 Claude에는 --plugin-dir와 Skill 도구를 전달하지만 run_codex에는 원본 경로 없이 같은 prompt를 전달한다. dept_note 함수만 Bash로 추출 실행한 결과 exit 0, 두 릴리즈 스킬 이름 존재, /skills/ 경로 없음. 이는 안내 생성의 실행 증거이며 실제 Codex 자식 실패의 전체 원인 입증은 아니다.
- 원본 /mnt/c/Users/USER/projects/headcount/plugins/marketing/skills/product-launch/SKILL.md와 technology/skills/release-and-deployment/SKILL.md가 존재하고 읽힌다. 현재 도구 목록에는 Skill 호출 도구가 없다.
- 네 package/lock의 git log --all 이력과 각 git show JSON을 확인: root는 0.0.0, V2는 0.1.0만 존재. 태그 없음. 절차 5의 ‘버전 파일도 없음’ 조건 불충족. 원격 최신 Release 및 승인된 최초 계약은 미확인.
- tests/test_gate.py:Release는 스키마·자산 경계만 검사한다. tests/test_sim.py:run/ReleaseSafety는 가짜 에이전트 시나리오여서 실제 원본 접근 증명이 아니다. fixer.sh:52~61은 failed snapshot을 적재하고 run.sh:1690의 ‘두 번 실패’는 고정 문구다.

소유 환경에서 실행할 순서 (모두 미착수)

1. 동일 registry의 release 스킬 경로를 해석하고 Codex prompt에 절대 SKILL.md 및 상대 참조 기준을 제공한다. 하드코딩 사본 대신 HEADCOUNT_DIR와 registry를 사용하고 원본 누락은 실패 처리한다. 증명: 실제 자식에서 두 원본의 이름과 상대 참조 하나를 읽은 결과를 확보한다. 체크포인트: 접근 성공 이전에 다음 단계로 가지 않는다; 별도 사람 승인 요청은 이 정찰에서 하지 않는다.
2. 확정된 최초 계약이 이미 존재하는지 소유 환경에서 확인한다. 없으면 임의로 선택하지 않고 차단 상태를 유지한다. 존재하면 release_project가 그 출처를 전달하도록 한다. 증명: 서로 다른 root/V2 버전을 혼동하지 않고 계약의 대상·증가·태그·자산을 재현한 계획. 체크포인트: 계약 미확보 시 이후 릴리즈 실행 금지.
3. 수정 전후 동일 실자식 회귀, Release/ReleaseSafety, 대상 앱 빌드, release gate를 순서대로 실행한다. 성공 JSON을 손으로 꾸미지 않는다. 체크포인트: 모든 증거가 있을 때만 원장 ‘수정 과제 완료’ 및 ideas done 전환.

이번 실제 검증 (validation.json)

- TMPDIR를 본 회차 tmp/로 지정하고 PYTHONDONTWRITEBYTECODE=1로 Release 5개 테스트 실행: exit 0. ResourceWarning(닫히지 않은 파일 핸들) 발생. 변경 전 보호 기준선일 뿐 복구 통과 아님.
- python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-082411-aiportal-front-admin-improve : exit 1, state=failed. 저장 실패 원장을 재판정했으며 실패 단계 자체를 재실행한 것이 아니다.
- 기본 runtime config 검사: exit 0. V2 node_modules 없음; npm ci/verify/build, 실제 Codex 자식, ReleaseSafety sim, 서버/UAT 및 수정 후 회귀 미실행.
- 새 후보용 HTML unquoted src 및 video poster의 외부 HTTPS fixture를 실제 verify-offline.mjs로 각각 검사: 둘 다 exit 0. 검사 공백만 입증했고 production bundle 영향은 미확인. 지정 실패와 무관하여 이번 구현 대상으로 택하지 않는다.

접근안 비교 및 추정 근거

- 권고: 원본 전달 최소 수정 + 별도 최초 계약 확인. 기존 명부/원본을 재사용하고 검증을 보존한다. 가장 큰 가정은 aidev 소유 표면과 계약 근거가 확보된다는 것인데 현 회차에서는 충족되지 않는다.
- 대안: 엔진 중립 스킬 패키징 전체 도입은 범위 L로 제외. 앱에 스킬 사본/임의 릴리즈 문서를 넣는 방식은 원인 소유 불일치와 관례 발명 때문에 제외. 현상 유지·차단 기록만 반복하는 방식은 복구 효과가 없어 완료 과제로 인정하지 않는다.
- Bottom-up 잠정 추정: 소유 환경 확보 후 전달 수정 10~15분, 회귀 10~15분, 기록·검토 5~10분 = 25~40분. 알려진 경로/참조 변동 contingency 5~10분을 별도로 더하면 30~50분, 신뢰 낮음(실측 분포 없음). 관리 예비는 배정하지 않았다. 계약 결정 대기·앱 설치/빌드·실자식 실행 지연은 이 범위 밖이며 전체 복구 45분 보장은 불가.
- 유사 추정 대조: 제공된 최근 여섯 회차가 no-change여서 성공 작업량의 비교 표본이 없다. 따라서 M은 전달 수정 부분에만 해당하며 전체 복구 약속으로 쓰지 않는다.
- 적용 스킬: headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md 직접 읽음. 외부 추정 수치나 신뢰 확률은 인용하지 않았다.

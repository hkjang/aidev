- 과제: 수정 과제 — 릴리즈 러너의 Codex 폴백에 실제 headcount 스킬 원본을 전달한다 (가치 4 / 위험 2 / 작업량 M)
- 왜: aidev/bin/run.sh의 run_agent는 Claude용 dept_note와 Skill 호출 지시를 붙인 프롬프트를 run_codex에 그대로 넘기지만, run_codex는 pargs/HEADCOUNT_DIR/스킬 경로를 전달하지 않아 존재하는 스킬을 에이전트가 찾지 못한다. 실제 원본 경로와 읽는 방법을 엔진에 맞게 전달하면 스킬 미발견이라는 반복 실패 원인을 제거하고, 별도로 남는 릴리즈 관례 부재를 정확히 판정할 수 있다.
- 수용 기준: 1) 실제 run_agent → Codex 폴백 실행 경로에서 release 역할의 marketing:product-launch 및 technology:release-and-deployment 원본 SKILL.md와 상대 참조 리소스를 읽을 수 있고, 존재하지 않는 Skill 도구를 호출하라는 안내가 없다. 2) scout에도 동일한 전달 경로를 적용해 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration이 누락되지 않으며 Claude의 기존 plugin-dir 전달은 유지한다. 3) 회귀 검증은 실제 프로덕션 전달 함수와 자식 프로세스를 통과하고 원본 파일 읽기 성공·원본 누락 시 명시적 실패·전역 설정 무변경을 증명한다; 소스 문자열 검사나 성공 JSON을 쓰는 가짜 에이전트만으로 통과를 주장하지 않는다. 4) 기존 release gate의 failed/skipped 차단·자산 경로 검사·CI·비밀검사 조건은 그대로 통과하고, 이 저장소의 릴리즈 관례 부재는 해결되지 않은 별도 차단으로 남긴다. 원장에 '수정 과제: 스킬 전달 복구, 릴리즈 관례 미확정으로 전체 릴리즈 미완료'라고 실제 결과대로 기록한다.
- 건드릴 파일: 아래는 모두 aiportal-front-admin 밖의 **aidev 러너 저장소**다. /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args, dept_note, run_agent, run_codex — registry의 역할별 스킬을 HEADCOUNT_DIR에 결합해 읽을 수 있는 절대 경로/상대 참조 기준을 엔진별로 전달한다; /mnt/c/Users/USER/projects/aidev/tests/test_sim.py 및 tests/sim/run_sim.sh — 실제 폴백 배선과 파일 접근 검증을 보강한다(기존 harness는 가짜 claude/gh이며 Codex 전달 검증이 없다). /mnt/c/Users/USER/projects/aidev/bin/gate.py:evaluate_release와 tests/test_gate.py:Release — 읽고 회귀 검증만, 판정 조건 수정 금지. aiportal-front-admin 소스·package·CI는 수정 대상이 아니다.
- 검증 명령: `bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh`; `PYTHONDONTWRITEBYTECODE=1 TMPDIR=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-111404-aiportal-front-admin-improve python3 /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release` (정찰에서 5개 통과, 기존 ResourceWarning 있음); `python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-102422-aiportal-front-admin-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-102422-aiportal-front-admin-improve` (정찰에서 ok:false/state:failed 재현, 출력 JSON도 확인할 것). 러너 수정 환경에서 `python3 tests/test_sim.py` 실행 가능하나 기존 harness는 /tmp에 쓰므로 이번 정찰에서는 실행하지 않았다. 새 전달 회귀 테스트의 실행 명령은 구현자가 실제 파일을 만든 뒤 원장에 추가해야 하며 현재 존재한다고 주장하지 않는다. 앱의 `npm run verify --prefix upgrade/admin-v2`는 vue-tsc 미설치로 exit 127; 릴리즈 실패와 별개다.
- 위험과 피할 것: 현 회차 구현 권한이 aiportal-front-admin에 한정되면 외부 러너를 몰래 편집하지 말고 이 과제의 외부 이관 필요를 ledger-entry.md에 기록한다. .gitlab-ci.yml, auth/, 정책 flag, package 버전·lockfile, 전역 CODEX_HOME 설정 변경 금지. 태그·CHANGELOG·최초 릴리즈 관례를 임의 생성하거나 버전 파일을 지워 skipped 조건을 맞추지 않는다. release-prompt.md 절차 5 확대, 실패를 성공/skipped로 치환, autonomy 강등/릴리즈 끄기로 통과시키기 금지. 요청된 '고친 뒤 같은 검증 통과'는 정찰의 코드 수정 금지와 충돌하므로 구현자 단계에서 수행해야 한다; 전체 릴리즈 성공은 현재 근거로 약속할 수 없다.
- 차선 후보: 동일 실패의 릴리즈 관례 결정 근거를 운영 기록으로 이관 — 러너 수정 권한이 없거나 스킬 전달이 이미 복구됐다면 앱의 무관한 UX 과제로 바꾸지 말고, 기존 버전 값·태그 없음·release-prompt.md 제약을 원장에 남겨 최초 릴리즈 정책을 정할 주체에게 넘긴다. 이는 수정 완료나 릴리즈 성공이 아니다.

## 확인한 근거와 남은 차단
- 앱 HEAD: 01fedba, 직전 query 경합 수정 15e032f 병합 확인. git status --porcelain 비어 있음.
- 실제 스킬은 /mnt/c/Users/USER/projects/headcount/plugins/{부서}/skills/{스킬}/SKILL.md에 있다. 요청된 세 스킬과 release의 두 스킬 원본 모두 읽었다. Skill 호출 도구는 현재 도구 목록에 없다. 기존 프로필의 '스킬 파일 없음'은 검색 범위가 좁았던 결론으로 정정한다.
- run.sh:245-259 agent_plugin_args/dept_note, :290 안내 생성, :308 폴백, :314-327 run_codex에서 경로/플러그인 누락을 확인. 경로를 전달하면 에이전트의 원본 접근 가능성이 해결된다는 판단이며 실제 Codex 재호출은 미실행이다.
- release-prompt.md 절차 2~3은 기존 증가·태그·노트 관례를 요구하고 절차 5는 '태그도, 버전 파일도, 릴리즈 노트도 없음'일 때만 skipped를 허용한다. 루트 0.0.0과 V2 0.1.0은 실제 버전 파일에 존재한다. git log --all -G '"version"' -- package.json upgrade/admin-v2/package.json에서 최초 추가 커밋만 확인했고 태그는 없다. docs/INDEX.md는 실제 release 근거가 있을 때만 version history를 기록하도록 한다.
- 2026-09-21-102422의 release.json은 failed, release-context.md는 Release 목록 공란/워크플로 없음. 앞선 09-20-085400 기록은 skipped이며, 입력의 '같은 실패 두 번'에 정확히 대응하는 두 번째 failed 원본은 미확인이다.
- .gitlab-ci.yml은 main/develop 브랜치 build/copy이고 실패 보고의 release 단계가 아니다. V2 deploy/create-offline-cache.sh는 lock hash 기반 npm cache, deploy/deploy-changed.sh는 문서상 정적 파일 배포이며 태그 관례의 근거가 아니다. build package script와 runtime/offline/integrity 스크립트도 읽었으나 보고된 실패 지점이 아니다.

## 실행 순서·검증 지점
1. [미착수] 러너 소유 작업 환경에서 위 원본과 실패 경로를 재확인하고 스킬 전달의 통합 재현을 먼저 만든다. 판정: 실제 자식 프로세스가 파일을 읽지 못하는지 확인. 앱 저장소에만 쓰기 가능한 경우 이 단계에서 외부 이관 기록으로 끝낸다; 사람에게 질문하지 않는다.
2. [미착수] 엔진별 안내를 생성하되 역할 registry와 원본 SKILL.md를 정본으로 유지한다. Codex에는 읽을 파일의 절대 경로·상대 참조 해석 방법을 주고 Claude에는 현재 plugin-dir를 유지한다. 파일 누락을 숨기지 않는다. 검증: 1의 동일 재현을 다시 실행해 원본 읽기 성공 및 누락 실패를 확인한다. 별도 사람 승인 checkpoint는 추가하지 않는다.
3. [미착수] 기존 Release 5개와 shell 구문 검증, 추가 통합 테스트를 실행하고 결과를 원장에 적는다. 실패하면 범위를 늘리지 말고 전달 수정만 되돌린다. 전체 릴리즈 관례 차단은 별도로 유지하고 '릴리즈 통과'로 기록하지 않는다.

## 대안 비교·산정 근거
- 선택: 러너가 원본 경로를 전달. 한 전달 경로에서 모든 역할을 복구하고 원본 중복을 피한다. 실제 수정 파일이 외부 러너라는 점이 가장 큰 전제다.
- 앱 README에 경로만 추가: 현 저장소만 완화하며 다른 역할/프로젝트의 누락은 반복되므로 차선으로도 선정하지 않음.
- 스킬을 전역 설치/복제: 전역 상태 변경과 정본 중복을 만들어 제외.
- 최초 릴리즈 관례 신설: 현재 사용자 제약 및 release-prompt.md와 맞지 않아 제외. 현 상태를 유지하고 근거를 이관하는 것은 권한 부족 시 유효한 차선이다.
- bottom-up 추정: 재현 8~12분, 전달 수정 10~15분, 회귀·기록 7~10분 = 25~37분. 알려진 불확실성(엔진별 파일 접근/인자) 예비 5~8분을 따로 두어 30~45분, 실측 이력이 없어 신뢰도 낮은 작업 추정이며 납기 보장이 아니다. 관리 예비는 배정하지 않음; 전역 설치·릴리즈 정책 결정·실제 배포는 범위 밖. 첫 재현에서 외부 환경 변경이 필요하면 재산정한다.
- 적용 스킬: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md의 절차를 사용. estimating references/sources.md를 읽었으며 외부 추정 표준을 적용·인용한 것이 아니라 이 조사에서 분해한 잠정 시간이다.

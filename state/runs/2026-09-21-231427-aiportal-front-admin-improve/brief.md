- 과제: 수정 과제 — 릴리즈 실패의 스킬 전달·최초 릴리즈 계약 복구 (가치 4 / 위험 2 / 작업량 M)
- 왜: 저장 실패는 앱 빌드 결함이 아니라 외부 러너의 스킬 전달 문제와 최초 릴리즈 관례 부재를 보고하며, 현재도 같은 실패 스냅샷이 앱 수정 큐에 들어온다. 두 원인을 소유 범위에서 해결하고 실제 실행으로 검증해야 반복 중지 대신 릴리즈를 완료할 수 있다.
- 수용 기준: 1) 실제 release→run_agent→run_codex 자식이 명부의 marketing:product-launch와 technology:release-and-deployment 원본 및 상대 참조를 읽는 실행 증거가 있다. 2) 대상 패키지·다음 버전·태그 유무/형식·노트 위치/양식·자산 범위를 정할 기존 근거 또는 운영 결정이 확보되어 원래 릴리즈 절차를 만족한다. 3) 같은 실제 경로의 수정 전 실패/수정 후 성공을 재현하고 생성 결과를 변경 없는 gate.py release로 검사한다; 실패·누락 스킬·경로 밖 자산은 계속 거부해야 한다.
- 건드릴 파일: 현재 aiportal-front-admin worktree에서 지정 실패를 고칠 파일 없음. 원인 소유 파일(열람만 함)은 /mnt/c/Users/USER/projects/aidev/bin/run.sh:agent_plugin_args/dept_note/run_agent/run_codex/release_project — 엔진별 원본 전달과 자식 실행 경계; /mnt/c/Users/USER/projects/aidev/agents/registry.json — release 역할의 부서·스킬 정본; /mnt/c/Users/USER/projects/aidev/release-prompt.md:절차 1~5 — 최초 릴리즈 계약 판단. 이 목록은 외부 파일 편집 허가가 아니다.
- 검증 명령: 아래 실제 명령 및 한계를 참조한다.
- 위험과 피할 것: .gitlab-ci.yml/auth/router/deploy와 package 버전을 이번 실패를 우회하려고 바꾸지 않는다. skipped 조건 확대, failed를 released로 덮어쓰기, 임의 태그·CHANGELOG 신설, 스킬을 앱에 복제하기, 외부 러너 직접 수정, 대역 sim 통과를 실제 자식 검증으로 세는 것을 금지한다. 코드·커밋·원격·저장 release.json 변경 없음.
- 차선 후보: 없음 — 지정 실패와 무관한 UI/테스트/문서 과제로 교체하지 않는다. 아래 보류 후보는 다음 별도 회차용이다.

현재 판정: pending / blocked, 현 회차 구현 착수 대상 없음. 이전에 기각된 외부 수정 지시를 실행 가능한 과제처럼 재발행하지 않는다. 이 문서는 복구 완료 또는 실행 가능한 45분 수정 약속이 아니라 지정 과제의 근거와 현재 불성립 판정이다. 구현자는 외부 수정에 착수하거나 같은 전체 탐색을 반복하지 말고, 전제가 그대로면 원장에 '수정 과제 — 미완료(blocked), 실행 가능한 수정 대상 없음'으로 기록한다. 이 기록은 개선 성과가 아니다.

직접 확인한 근거

- HEAD 01fedba, git status --short 출력 없음, git log -30 확인. 로컬 태그 없음. root package.json/package-lock.json 버전은 기존 프로필상 0.0.0이며 이번 package.json 직접 확인; V2 package.json은 0.1.0. 이번 전체 버전 변경 이력 재조사 및 원격 최신 릴리즈 조회는 미실행이다.
- CLAUDE.md/AGENTS.md, CHANGELOG/RELEASE 문서는 저장소 파일 검색에서 발견되지 않았다. .github/workflows 없음. .gitlab-ci.yml은 기존 GitLab branch build/copy이며 이번 저장 실패가 이 job에서 발생했다는 증거 없음. 자격증명 값은 출력 전에 마스킹했다.
- README, docs/ROADMAP.md, docs/TESTING_GUIDE.md, V2 deploy/README.md, V2 package.json/vite.config.ts 확인. 문서는 정적 파일 배포·npm cache 반입을 설명한다. 새 버전 증가/태그 관례 승인 근거는 찾지 못했다.
- run.json project=aiportal-front-admin; registry의 builder.surface='그 회차의 worktree'; COMPANY.md 규칙 1은 구현자가 자기 표면 밖을 쓰지 않도록 한다. 현재 요청도 정찰에게 코드 변경과 회차 밖 쓰기를 금지한다.
- run_agent는 Claude에만 plugin-dir를 주고 Skill 안내가 붙은 prompt를 run_codex에 재사용한다. run_codex는 원본 경로/직접 읽기 안내를 추가하지 않는다. 원본 파일은 /mnt/c/Users/USER/projects/headcount/plugins/{marketing,technology}/skills/{product-launch,release-and-deployment}/SKILL.md에 실제 존재한다. 과거 자식이 왜 탐색에 실패했는지 전체 런타임 원인은 미확인이다.
- release-prompt 절차 5는 태그·버전 파일·노트가 모두 없어야 skipped다. 버전 파일이 존재하므로 스킬 전달만 고쳐도 최초 릴리즈 계약 공백은 남는다.
- fixer.sh의 failed 스냅샷 루프는 앱 이름으로 적재한다. run.sh:1690의 '두 번 실패'는 주입 문구이며 서로 다른 두 워크플로 실행 증거가 아니다.
- gate.py:evaluate_release와 tests/test_gate.py:Release의 5개 테스트를 읽었다. 스키마·자산 경계 테스트이며 스킬 가용성·버전 선택을 증명하지 않는다. tests/test_sim.py:Agents 및 tests/sim/run_sim.sh는 가짜 gh/claude를 사용한다.

검증 기록과 명령 (이번 실행)

```sh
# 저장소 cwd: 성공 exit 0
node upgrade/admin-v2/scripts/validate-runtime-config.mjs upgrade/admin-v2/public/config/runtime.json
# 각각 실행: 성공 exit 0, 문법 검사일 뿐 복구 검증 아님
bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh
bash -n /mnt/c/Users/USER/projects/aidev/bin/fixer.sh
# 저장된 실패 판정 재확인: exit 1, ok:false, state:failed
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/aiportal-front-admin.release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-231427-aiportal-front-admin-improve
```

npm ci/build/verify, Release unit suite, sim, 실제 Codex 자식 회귀, Java/IAM/UAT는 이번 미실행. V2 node_modules가 없다. 앱 표준 검증은 V2 cwd의 npm ci 후 npm run verify / npm run build이나 지정 실패의 복구 증거로 대신하지 않는다. 실제 자식 회귀 전용 테스트 진입점은 확인하지 못했으므로 존재하지 않는 명령을 제시하지 않는다.

재개 계획과 점검점 (현 회차 실행 지시 아님)

1. 소유 aidev worktree와 최초 릴리즈 계약이 제공됐는지 확인한다. 증거: 소유 범위 설정 및 계약 문서. 현재 둘 다 미충족; 이 점검점에서 멈춘다. 사용자에게 질문하거나 운영 결정을 대신 만들지 않는다.
2. 소유 회차에서 registry 기반 경로와 직접 읽기 대안을 실제 Codex 자식에 전달한다. 증거: 실제 run_agent→run_codex 자식의 원본·상대 참조 읽기 로그 및 누락 시 실패. 스킬을 통째로 중복 관리하는 설계는 제외한다.
3. 확보한 계약으로 원래 릴리즈 절차와 동일 검증을 수행한다. 증거: 실제 산출물 및 변경 없는 gate.py release 결과. 외부 전송은 본 정찰 범위 밖이다. 단계별 증거 없이는 다음 단계를 완료 처리하지 않는다.

대안 비교 / 추정 근거

- 최소 원인 수정: 소유 러너에서 registry 기반 원본 전달 + 확보된 최초 릴리즈 계약 적용. 중복 정본이 없어 우선이나 현재 소유 범위/계약 전제가 없다.
- 엔진 공통 스킬 로더 전면 개편: 범위가 크고 45분 제한에 맞지 않아 제외.
- 앱에 스킬 복제 또는 새 릴리즈 관례 추가: 실패의 소유 경계를 숨기고 미승인 관례를 만들어 제외.
- 현재 유지: 이번 선택. 실패를 보존하고 구체적 불성립 근거를 남기되 이를 수정 성과로 세지 않는다.
- M은 소유권과 계약이 이미 해결된 뒤의 조건부 bottom-up 추정이다: 배선 10~15분, 실제 자식 회귀 10~15분, 게이트/기록 5분 = 25~35분; 알려진 자식 시작 지연 예비 5~10분을 별도 두어 30~45분. 신뢰도 낮음(통계적 신뢰구간 아님), 계약 대기/환경 확보/실제 빌드·배포 시간 제외, 관리 예비 미배정. 유사 성공 회차가 없어 유추법 교차검증 불가; 현재 전체 복구 소요시간은 산정 불가다.
- 적용 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration의 로컬 SKILL.md 직접 열람. Skill 도구 없음. 외부 전문 표준의 수치·통계적 보장은 사용하지 않았다.

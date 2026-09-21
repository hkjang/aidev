- 과제: [수정 과제] 릴리즈 버전 결정 입력 복구 — 필수 출처 결손, 실행 가능한 수정 미확정 (가치 5 / 위험 2 / 작업량 S)
- 왜: 외부 release-prompt.md 2·3단계는 실제 기존 증가·커밋·태그 관례를 요구하지만 현재 확인한 Git/JSON에는 그 근거가 없어 releaser가 중단된다. 유효한 근거를 확보하면 관례 창작이나 게이트 완화 없이 최소 수정과 동일 검증을 확정할 수 있다.
- 수용 기준: 1) aiportal-front와 기준 SHA에 적용되는 실제 릴리즈 이력 또는 승인 정책의 출처, 증가 단위, 커밋·태그 형식, 노트·자산·검증 방법을 확보한다(현재 미충족). 2) 그 출처를 따라 최소 변경하고 package.json:version 및 package-lock.json:version/packages[""].version의 정합성과 필요한 산출물을 검증한다(미착수). 3) 실제 releaser의 같은 버전 판단 경로와 기존 gate가 통과한 전후 증거를 원장에 남긴다(미완료); 합성 released JSON, 문자열 검사, 회귀 테스트 통과만으로 완료하지 않는다.
- 건드릴 파일: 현재 확정 가능한 구현 파일 없음. 근거 확보 시 docs/RELEASE.md에 출처를 연결하고, 근거가 요구할 때만 package.json:version 및 package-lock.json 두 필드를 수정 대상으로 확정한다. 직접 읽은 외부 /mnt/c/Users/USER/projects/aidev/bin/run.sh:release_context/release_project/retry_release_workflow, bin/gate.py:evaluate_release/cmd_release, tests/test_gate.py:Release는 진단 근거이며 이번 저장소 구현 범위 밖이다. .gitlab-ci.yml은 브랜치 배포 설정이므로 이번 버전 판단 실패 수정 대상으로 지정하지 않는다.
- 검증 명령: 아래 읽기 전용 재현 명령 참조. 실제 releaser 판단만 단독 실행하는 안전한 로컬 명령은 미확인이다. 원격 전송을 포함하는 run.sh 전체 실행을 검사 명령으로 사용하지 않는다.
- 위험과 피할 것: 신규 출처가 없는 동일 BLOCKED 조사·게이트 반복을 구현 성과로 재배정하지 않는다. 현재 인계로는 수정 시작 조건이 충족되지 않는다. 임의 패치 증가·최초 관례 창작·failed→skipped/released 변경·기존 실패 JSON 편집·게이트/워크플로 완화·외부 러너 수정·auth/session 변경 금지. 빈 context나 과거 원격 인증 실패를 원격 이력 부재로 단정하지 않는다.
- 차선 후보: 없음 — 고정 수정 과제이므로 무관한 개선으로 대체하지 않는다. 아래 후보들은 향후 회차용이다.

판정

정찰 산출물은 작성했으나 요청된 원인 수정 및 수정 후 동일 검증 통과는 미완료다. 이전 여섯 회차와 같은 no-change 작업을 새로운 실행 과제로 배정하지 않는다. 이번 문서는 기존 수정 과제의 미해결 상태와 재개 조건을 전달한다. 새 승인 정책·실제 관례 없이 45분 내 완료 가능한 해법은 확인하지 못했다.

직접 확인한 근거

- docs/RELEASE.md를 먼저 읽고 실제 Git/JSON 재조회: HEAD e938e8e, non-shallow 37커밋, 로컬 태그 0개. 현재 버전 세 값 0.0.0. 최초 package 0.0.0, 최초 lock 두 필드 없음. package/lock 이력에서 버전 증가 근거 없음.
- AGENTS.md, README, docs/07 로드맵 앞부분 및 docs/06·08·09 관련 절, 최근 git log -30, package.json, vite/vitest 설정, .gitlab-ci.yml, tests/unit 구성을 확인했다. CLAUDE.md와 .github/workflows 없음. src/docs TODO/FIXME 검색 일치 없음. node_modules 없음.
- 09-20-211415 release.json은 스킬 탐색 실패와 관례 결손, 09-21-001359는 실제 스킬 원문 접근 후 관례 결손이다. 최초 lock 값 관련 옛 실패 사유는 현재 조회로 정정했다. 앱 CI 실패 로그나 독립 GitHub workflow run ID/실패 step은 미확인이다.
- release_project는 run_agent → release.json → gate 경로이며 ok=false에서 push 전에 종료한다. retry_release_workflow는 별도 GitHub run 재시도 함수다. 이번 두 JSON을 그 함수의 같은 실패 step 2회 증거로 취급하지 않는다.
- release_context의 gh 실패를 '(없음)'으로 표시하는 처리 및 tag fetch 실패 후 계속하는 처리는 확인했다. 이번 실패와 인과가 입증되지 않았고 외부 소유이므로 해법으로 재선정하지 않는다.

실행한 검증과 한계

저장소 루트에서 실행 가능하며 아래 두 gate 명령은 각각 exit 1 / state=failed / ok=false였다.

```bash
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-20-211415-aiportal-front-improve
python3 -B /mnt/c/Users/USER/projects/aidev/bin/gate.py release /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve/release.json --out-dir /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-001359-aiportal-front-improve
python3 -B /mnt/c/Users/USER/projects/aidev/tests/test_gate.py Release -v
git diff --check
git status --short
```

Release 테스트는 TMPDIR를 이번 회차 폴더의 verification-tmp로 지정해 실행하고 생성된 임시 파일을 제거했다. 5 tests OK/exit 0, 미닫힌 파일 ResourceWarning 발생. 이는 기존 게이트의 회귀 검사 및 실패 거부 재현이며 모델의 판단 재실행·수정 후 통과가 아니다. 앱 npm ci/test/build, 실제 releaser, 원격 조회·배포 미실행. 앱 변경이 필요한 근거가 생기면 npm ci → npm test → npm run build:dev를 수행하되 전용 Runner 배포와 구분한다.

재개 시 단계·증명·체크포인트

1. 새 출처가 들어온 경우에만 적용 프로젝트·기준 SHA·정책 권한과 이력 신뢰성을 대조하고 이 과제서를 갱신한다. 증명은 출처/Git/JSON 근거표. 출처가 없으면 후속 단계 미착수로 유지한다. 사용자 질문이나 반복 조사를 추가하지 않는다.
2. 출처가 요구하는 파일과 최소 변경을 확정한다. 증명은 버전 세 값과 실제 커밋/태그/노트/자산 관례 대조 및 출처가 정한 검증 명령. 근거와 충돌하면 계획을 고치기 전 다음 단계 금지. 아직 명령과 다음 버전을 추측하지 않는다.
3. 실제 판단 경로와 동일 gate의 전후 결과를 확보해 원장에 수정 과제로 기록한다. 증명은 실제 releaser 결과와 기존 gate exit 0 및 필요한 앱 검사. 합성 성공이나 문서 작성만으로 완료 금지. 게시·커밋·태그는 정찰 범위 밖이다.

대안 비교

- 실제 이력/승인 정책 복구: 가장 작은 정당한 해법이며 고정 과제로 유지. 핵심 가정은 해당 출처를 확보할 수 있다는 것인데 현재 충족되지 않는다.
- 외부 컨텍스트 수집 개선: 진단 가치를 가지지만 소유 범위와 인과 확인이 선행되어야 한다. 이번 구현 배정 제외.
- 새 움직임 없이 미완료 유지: 현재 가능한 판정이지만 릴리즈 해결이나 개선 성과로 세지 않는다.
- 임의 버전 증가 및 게이트 완화는 사용자 금지 사항이므로 유효 대안이 아니다.

후보 재평가

ideas.json의 기존 45개 제목·상태를 보존하고 신규 2개를 추가했다. 직접 대조 범위에서 해결됐거나 무효화됐다는 새 근거는 없다. 일부 항목은 동일 HEAD와 이전 기록을 근거로 보존했고 개별 런타임 동작은 미확인이다.
- 고정 릴리즈 입력 복구: 가치 5 / 위험 2 / S, pending, 출처 결손.
- useAppList 비배열 캐시 방어: 3 / 2 / S, pending, fetchTopApps/updateList가 같은 map 경로로 진입.
- globalLoading 병렬 요청: 3 / 3 / S, pending, boolean 유지; 호출 짝 감사 필요.
- README 빌드 명령: 2 / 1 / S, pending, 문제 해결 절의 build 스크립트 부재.
- 신규 bootstrap 오류 안내: 2 / 2 / S, pending, main.js:bootstrap의 전체 catch가 config 실패 안내만 사용; 실제 렌더링 검증 필요.
- 신규 useAppList 구독 수명 검사: 3 / 2 / S, pending, 실제 Vue 마운트/해제 및 mitt 배선 회귀 검사 공백. 현재 누수 발생으로 단정하지 않는다.

추정 근거

입력 확보 이후의 조건부 bottom-up: 출처 대조 5–10분, 최소 반영 5–10분, 검증·기록 10–15분 = 20–35분. 알려진 검증 변동 예비 0–10분을 별도로 두어 20–45분(S), 낮은 확신의 계획 범위이며 통계적 신뢰수준은 산정하지 않았다. 유사 회차가 모두 no-change이므로 성공 구현의 유사 추정값과 비교할 근거가 없다. 정책 확보 시간·전용 Runner·외부 배포·대규모 빌드는 제외해 전체 해결시간은 미산정, 관리 예비는 미배정이다. 출처가 도착하면 범위와 시간을 재추정한다.

스킬 적용

호출 가능한 Skill 도구 없음. 아래 실제 원문을 읽었고 성공한 Skill 호출로 간주하지 않는다. 별도 고정 반환 스키마는 없으며 대안 비교·단계별 증명/체크포인트·추정 가정/예비 분리를 위에 반영했다.
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md 및 references/sources.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
외부 비용편익 이론이나 정량적 신뢰수준을 주장하지 않았다.

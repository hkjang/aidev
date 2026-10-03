# 회차 노트 2026-10-03-220731-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:07] base pinned — main@814482a
- [러너 22:07] autonomy release — 
- [러너 22:13] scout done — `cd web && npm test --silent` 가 깨끗한 작업 트리에서 exit 1 로 끝나는 원인 제거 — web 테스트 게이트를 자기충�

## 구현 노트
- 무엇을/왜: 깨끗한 트리에서 `cd web && npm test` 가 `sh: 1: vitest: not found`(EXIT=127)로 **0건 실행 후 빨개지던** 것을 `web/scripts/ensure-deps.mjs` + `package.json` 의 `pretest` 한 줄로 자기충족화. 그러자 그 아래에서 실제 레이스가 드러나 `AdminOperationsPages.test.tsx:92` 의 동기 `getByRole` → `findByRole` 로 고쳤다(단정 내용 동일). 커밋 9df8049, 파일 3개, 프로덕션 코드 0줄.
- **확신 없는 곳**: `AdminAskOriginalLauncher.test.tsx:110` 이 **전체 스위트에서만** ~1/3 확률로 깨진다. 원인 증명 실패해 **손대지 않았다** — 단독 3/3 통과, CPU 부하 무관, 응답 50ms 지연에도 통과(= '데이터 도착 전 드롭다운 오픈' 가설 반증). 단서: 같은 파일 테스트 1 은 68행 `waitFor` 후 같은 조작을 하는데 테스트 2 는 그 `waitFor` 가 없다. **비평가는 여기를 먼저 보라** — 내 변경과 무관한 선재 flake이고 그 파일 `git diff` 는 무출력이다.
- 일부러 안 한 것: 위 flake 의 시간초과 확대(금지된 땜질, 증명도 없음); `App.test.tsx` 버전 리터럴 정리(과제서가 섞지 말라고 명시); 차선 후보 `cteColumnList` 메모이제이션(우선 과제가 배정됨).
- **과제서 전제 불일치**: 수용 기준 4 의 `scripts/ci_workflow_test.go` 가 **이 base(814482a)에 없다**. `ci.yml:63`·`release.yml:99` 도 `-count=1` 이 없다 — 2026-10-01 의 커밋 12223b0 이 미머지로 보인다. 기준 4 의 실질(워크플로 불변)은 충족(`git diff .github/` 무출력).
- 다음 역할 주의: web 스위트는 이 기계에서 1회 ~160초(serial, maxWorkers:1)고 간헐 flake 가 있으니 **1회 red 로 단정하지 말고 재실행**할 것. `pretest` 는 `node_modules/.bin/vitest` 가 있으면 아무것도 하지 않는다(재설치 없음을 mtime 불변으로 확인). Go/통합 테스트는 이번 변경과 무관하다.
- [러너 22:50] brief accepted — 채택 — 분기 A 예측(`vitest: not found`)이 실행으로 정확히 맞았고 권고한 구현 형태(`web/scripts/ensure-deps.mjs` + `pretest`, `existsSy
- [러너 22:53] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 확인한 것: 레이스 근본원인을 코드로 독립 재구성(`AdminFeedbackPage.tsx:168` `setLoading(true)` + `:312` `loading ? Skeleton : table` → 선행 waitFor 가 풀리는 순간 대상 버튼은 언마운트. 구 `getByRole` 은 진짜 레이스였고 `findByRole` 은 role+이름 단정이 글자 그대로 동일 = 완화 아님); ledger 의 `실패 재현:` 줄이 그 증상과 일치하고 되돌려 red 까지 확인했다. pretest 의 비회귀 성질(green→red 불가)과 ci.yml:46→49·release.yml:72→75 의 `npm ci` 선행을 파일로 확인, /tmp 에서 스크립트를 호출해 cwd 오염 0·vitest mtime 불변(재설치 0)을 실측. 내가 돌린 게이트: `npm test` 3회 전부 EXIT=0 / 27파일·134테스트, `npm run typecheck` 0, `validate-release-contract.sh` 0, 트리 clean.
- 못 본 것: node_modules 를 실제로 지운 상태의 `npm ci` 설치 경로(네트워크 설치)와 Windows 분기는 실행하지 않았다 — 읽기로만 판단했다. e2e·Go 통합 테스트는 이번 변경과 무관해 돌리지 않았다.
- 승인이어도 남는 우려 ①: 구현자가 신고한 `src/components/AdminAskOriginalLauncher.test.tsx:110` flake 가 내 3/3 전체 스위트 실행에서 재현되지 않았다 — 반증은 아니다. 원인 후보는 두 번째 테스트에 첫 테스트(68행)가 가진 `/admin/users?search=` waitFor 가 없다는 것. 이번 PR 과 무관한 선재 결함(그 파일 diff 무출력)이고 CI 는 main 에서도 이미 전체 스위트를 돌렸으므로 위험 증가는 없다. **다음 회차 최우선 후보로 남긴다.**
- 우려 ②(릴리즈 노트 소재): `pretest` 는 `test` 만 보호한다 — 깨끗한 트리에서 `npm run lint`/`typecheck`/`build`/`test:e2e` 는 여전히 설치 상태로 빨개진다. 러너 검증 명령이 그중 하나로 바뀌면 같은 증상이 재발한다. 또 `npm ci` 는 web/node_modules 를 통째로 지우므로 로컬 패치는 소실된다(gitignore 된 산출물이라 수용).
- 검토 부서: security·legal 모두 차단 소견 없음. 프로덕션 코드 0줄·입력면 0·주입면 0, 개인정보/의존성/라이선스 변동 없음. 공급망 소견(`npm test` 가 레지스트리 설치를 유발 가능)은 락 고정 `npm ci` 로 완화되고 공격 경로가 없어 notes 로만 남겼다. 부수 확인: 과제서 수용 기준 4 의 `scripts/ci_workflow_test.go` 는 이 base 에 실제로 없다(구현자 보고가 맞다).
- [러너 23:04] review approved — 리뷰 승인 (risk=low)
- [러너 23:04] pr created — https://github.com/hkjang/qurio/pull/32
- [러너 23:19] ci passed — 검사 1개 모두 success
- [러너 23:19] merge done — 9df8049
- [러너 23:45] release ci-blocked — 릴리즈 커밋 CI: timeout — 제한 시간 안에 CI 완료를 확인하지 못함 (태그 보류)

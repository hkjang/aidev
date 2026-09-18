# 과제서 (2026-09-19, cutover, 오류 대응 회차)

## 진단 결과 — 먼저 읽을 것

지난 회차의 'error' 는 **저장소 결함이 아니라 러너의 예산 보류**다. 정찰이 실제로 확인한 근거:

- `state/runs/2026-09-18-163356-cutover-improve/run.json`: `started 16:33:56 → finished 16:33:57`, `outcome: "error"` — 1초 만에 끝났고 에이전트가 실행되지 않았다(agent-*.txt, verify.*, ideas.json 없음).
- 같은 디렉터리 `stages.json`: `{"improve": {"state": "hold", "reason": "회차 예산($22)이 오늘 남은 상한을 넘음"}}` — 배정 메시지의 `hold: budget` 과 일치.
- 이 저장소에는 **릴리즈 워크플로 파일이 없다**. `.github/`, `.gitlab-ci.yml`, `Jenkinsfile` 모두 부재(루트 목록으로 확인). package.json 의 스크립트는 `lint / test:unit / test:e2e / build / screenshots:guide` 뿐이며 릴리즈는 러너(aidev)가 밖에서 수행한다.
- 마지막으로 실제 검증이 돈 회차 `2026-09-17-035302-cutover-improve` 의 `verify.gate.json` 은 `{"ok": true, "state": "verified"}` 이고, `2026-09-19-035746-cutover-shepherd` 는 `outcome: "approve"`. 코드 쪽 실패 흔적이 없다.
- 이 브랜치(`auto/2026-09-19-0808`)는 `main`(fb951ea, PR #3 머지) 기준이며 작업 트리는 깨끗하다. 09-17 회차의 쿠키 Secure 변경(`auto/2026-09-17-0353`)은 아직 main 에 머지되지 않아 이 브랜치에는 `lib/adminSession.ts` 에 `resolveAdminCookieSecure` 가 없다(grep 0건).

따라서 "워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 원인을 고치라" 는 지시에서 **고칠 코드가 없다**. 워크플로를 느슨하게 만드는 것도 당연히 하지 않는다. 정찰은 `npm run test:unit` 을 직접 실행하지 못했다(도구 권한 거부) — 로컬 통과 여부는 **미확인**이며 구현자가 아래 검증 명령으로 재현해야 한다.

## 과제

- 과제: 수정 과제 — 예산 보류 오류 진단 확정, 로컬 검증 재현, 원장 기록 (가치 4 / 위험 1 / 작업량 S)
- 왜: 러너가 같은 이유(hold: budget)로 두 번 멈춰 'error' 로 적재됐지만 저장소에는 워크플로도 실패한 단계도 없다. 이번 회차에 "코드 결함 없음" 을 검증 결과와 함께 원장에 남겨야 다음 회차가 같은 오류 대응을 반복 배정받지 않는다.
- 수용 기준:
  1) `npm run lint`, `npx tsc --noEmit`, `npm run test:unit` 이 이 브랜치에서 통과한다(2026-09-14 회차 기준 node --test 65건; 숫자는 실행 결과로 적을 것).
  2) `npm run test:e2e` 가 통과한다. 이 환경에는 ms-playwright 브라우저가 없으므로 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome` 을 지정해 실행한다(09-17 회차 실측). 브라우저를 못 찾으면 그 사실을 "미확인" 으로 적고 1) 만으로 마감한다.
  3) 원장(ledger-entry)에 '수정 과제' 로 기록: 오류 원인 = 러너 예산 보류(stages.json 인용), 저장소에 릴리즈 워크플로 없음, 검증 재현 결과(통과 건수), **코드 변경 없음**. 이 회차의 올바른 결과는 "변경없음" 이다 — 억지로 코드를 바꾸지 말 것.
- 건드릴 파일: 없음(읽기만). 읽어야 할 것 — `package.json` 스크립트, `playwright.config.ts`(브라우저 경로 env), `lib/**/*.test.ts`(6개: mail/config·events·service·smtp, tracking/core·violations), `e2e/*.spec.ts`(4개).
- 검증 명령:
  ```
  npm run lint
  npx tsc --noEmit
  npm run test:unit
  PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e
  ```
- 위험과 피할 것: 워크플로·검증 스크립트를 느슨하게 만들지 말 것(배정 지시의 금지 사항). `lib/adminSession.ts`·`app/api/auth/**` 는 미머지 브랜치 `auto/2026-09-17-0353` 과 충돌하므로 건드리지 말 것. `data/activity.json` 은 e2e 가 격리 파일을 쓰는지 `e2e/global-setup.ts` 로 먼저 확인하고, 실행 뒤 `git status` 가 깨끗한지 확인할 것.
- 차선 후보: **러너가 이 회차에 반드시 코드 변경을 요구할 때만** 아래를 수행한다 — `validateActivityImport` 단위 테스트 추가 (가치 3 / 위험 1 / S).
  - 새 파일 `lib/activityData.test.ts` (node --test, 기존 테스트처럼 `'./activityData.ts'` 확장자 포함 import, Node 22 strip-types). `lib/activityData.ts:120 validateActivityImport` 의 실제 함수를 호출해 증명할 것(대역 금지):
    - 최상위가 객체가 아니면/`activities` 가 배열이 아니면 `ActivityImportValidationError` throw (133~143행)
    - 필드 검증: id 빈 문자열·120자 초과, parentId 빈 문자열, level 0/51, time·title 길이 초과, status 잘못됨 → issues 경로 `activities[i].<field>` (169~216행)
    - 레거시 `'진행중'` → `'진행'` 정규화 카운트 (209~212행)
    - id 중복(236~246), 최상위 level≠1(250~253), 자기 부모(257~259), 부모 부재(262~264), level 이 부모+1 아님(265~270), 순환(273~288: A→B→A 한 번만 보고)
    - issues 50건 초과 시 omittedIssueCount 증가 (19행 MAX_VALIDATION_ISSUES, 125~131행)
  - 주의: `activityData.ts` 는 모듈 로드 시 `process.cwd()/data/activity.json` 경로만 계산하고 파일은 읽지 않으므로(10~17행) 단위 테스트에서 import 해도 부작용이 없다 — 그래도 `readActivityData/writeActivityData` 는 호출하지 말 것. 반환 타입 `ValidatedActivityImport` 의 필드명(issues/omittedIssueCount/normalizedLegacyStatuses 등)은 파일 상단 타입 선언에서 확인하고 적을 것(정찰은 120행 이후만 읽었다).
  - 검증: `npm run test:unit` 신규 건 포함 통과, `npm run lint`, `npx tsc --noEmit`.

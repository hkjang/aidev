- 과제: 브라우저 없는 체크아웃에서 스모크 2건이 거짓 FAIL 로 뜨는 것 수정 (가치 5 / 위험 1 / 작업량 S)
- 왜: 이 저장소를 깨끗이 받아 `npm test` 를 돌리면 **항상** `84 passed, 2 failed, 6 skipped` 이 나온다(이번 정찰에서 실제로 확인). 두 FAIL 은 Playwright 브라우저가 없어서 run 이 `Launch browser` 에서 죽은 것인데, 그 사실을 감지하지 않고 엉뚱한 메시지로 실패를 보고한다 — 특히 하나는 "the run reached outside the policy"(네트워크 허용목록이 뚫렸다)라고 말한다. 네트워크 정책이 깨진 적이 없는데 깨졌다고 말하는 빨간 suite 는, 이 저장소에서 구현·비평·수리 역할이 진짜 회귀와 환경 소음을 구분할 수 없게 만든다.
- 수용 기준:
  1) 브라우저가 없는 환경에서 `npm test` 가 `0 failed` 로 끝나고, 두 검사는 다른 브라우저 의존 검사와 같은 `SKIP … no Playwright browser installed` 로 보고된다(= `84 passed, 0 failed, 8 skipped`).
  2) `SMOKE_REQUIRE_BROWSER=1` 일 때는 두 검사가 여전히 **fail** 한다(릴리즈 검증이 브라우저 없음을 통과시키면 안 된다). `requireBrowser` 가 참이면 skip 분기로 가지 않는 코드여야 한다.
  3) 브라우저가 있을 때의 실제 단정(step 타임라인, PROBE_OUTSIDE=403, blockedRequests)은 하나도 느슨해지지 않는다 — skip 분기는 "실행 로그에 Playwright 실행파일 없음" 이라는 신호가 있을 때만 타고, 그 밖의 실패는 그대로 FAIL 이어야 한다.
- 건드릴 파일 (테스트 하네스 1개, 프로덕션 코드 0개):
  - `tools/smoke-test.mjs:2785` `check("a run records per-step durations and the step that failed")` — `GET /api/runs/{runId}` 응답(그리고 필요하면 `/logs`)에 브라우저 없음 신호가 있으면 `return "skip"`. `check()` 는 콜백이 `"skip"` 을 돌려주면 SKIP 으로 기록한다(`tools/smoke-test.mjs:31` 부근 `check`).
  - `tools/smoke-test.mjs:2244` `check("a script run cannot reach a host outside the allowlist")` — 이미 `text` 에 run 로그 전체를 모으고 있다. `PROBE_OUTSIDE` 단정 **앞에서** `text` 를 검사해 브라우저 없음이면 `return "skip"`.
  - 같은 파일에 `/Executable doesn't exist|playwright install/i` 가 10곳(312, 984, 1168, 1312, 1545, 1652, 1833, 2325, 3059줄)에 복사돼 있다. `function browserMissing(text) { return !requireBrowser && /Executable doesn't exist|playwright install|Failed to launch.*ENOENT/i.test(String(text || "")); }` 한 개로 묶고 호출부를 바꾸는 것까지가 이 과제의 적정 범위다. **3059줄의 `Failed to launch.*ENOENT` 가 가장 넓은 패턴이니 그것을 공통 패턴으로 쓸 것.** 묶는 것이 번거로우면 2건 수정만 하고 묶는 것은 생략해도 수용 기준은 만족한다.
- 검증 명령 (이 저장소에서 실제로 돈 것):
  - 먼저 `npm install --no-audit --no-fund` — 체크아웃에 `node_modules` 가 없다(이번 정찰에서 설치해 확인, 2초).
  - `npm run check` (`node --check server.js`).
  - `npm test` — 수정 전 `84 passed, 2 failed, 6 skipped`, 수정 후 `84 passed, 0 failed, 8 skipped` 여야 한다. 약 3~5분.
  - `SMOKE_REQUIRE_BROWSER=1 npm test` 로 두 검사가 fail 로 돌아오는지 확인(브라우저가 없으므로 fail 이 정상이다). 브라우저가 있을 때 PASS 하는지는 이 환경에서 **미확인** — 브라우저를 설치할 수 있으면 `npx playwright install chromium` 후 한 번 더 돌릴 것.
- 위험과 피할 것:
  - `server.js` 를 건드리지 말 것. 이것은 하네스 결함이고 프로덕션 동작은 정상이다.
  - skip 조건을 넓게 잡아 "run 이 실패하면 skip" 으로 만들면 네트워크 허용목록 회귀가 영구히 숨는다. 반드시 로그에 Playwright 실행파일 부재 신호가 있을 때만 skip 할 것.
  - `requireBrowser` 분기를 빼먹으면 릴리즈 검증(`docs/RELEASE_CHECKLIST.md` 가 요구)이 무력화된다 — 수용 기준 2가 이것을 본다.
  - 로컬 `npm` 은 Node v20.19.2 로 도는데 `package.json` 의 `engines` 는 `>=22` 다(`node -v` 는 v22.23.1). 그래도 suite 는 돈다. Node 관련 실패가 나오면 이 불일치를 먼저 의심하고, `engines` 나 Node 버전을 바꾸려 들지 말 것(범위 밖).
  - 인증/principals/스케줄러/릴리즈 스크립트는 건드리지 말 것.
- 차선 후보: 녹화 중복 제거가 서로 다른 `selectOption`·`press` 단계를 삼키는 버그 — `server.js:7355` 부근 `startRecording` 의 onEvent 는 `previous.value === step.value` 만 비교하는데 `selectOption` 은 값을 `step.values` 에, `press` 는 `step.key` 에 넣는다. 같은 `<select>` 에서 600ms 안에 두 번 고르면 두 번째가 조용히 버려져 녹화본이 틀린 옵션을 선택한다. 비교 서명을 `[action, locator, value, values, key]` 로 넓히면 되지만, **녹화 검사는 브라우저가 없으면 전부 SKIP 이라 이 환경에서는 결함을 end-to-end 로 증명할 수 없다** — 1순위를 먼저 끝내 suite 를 초록으로 만든 뒤에 할 일이다.

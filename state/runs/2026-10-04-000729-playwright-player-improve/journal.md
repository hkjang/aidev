# 회차 노트 2026-10-04-000729-playwright-player-improve — playwright-player
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:07] base pinned — main@a2a44d4
- [러너 00:07] autonomy release — 

## 구현 노트
- 무엇을 왜: 릴리즈 게이트를 떨어뜨린 `npm test` 의 유일한 FAIL 은 `a person can record and edit a scenario from the playground page` 였고, 원인은 제품이 아니라 검사의 동기화였다. `#startRecordingBtn` 이 이미 `goto` 행 하나를 렌더해 두므로 `#refreshRecordingBtn` 뒤의 `wait-for [data-testid='recording-step'] visible` 은 이미 만족된 조건이라 즉시 돌아오고, 뒤이은 `allTextContents` 가 갱신 전 DOM(1행)을 읽는다. 프로덕션 배선 프로브로 증명: 같은 순간 서버는 stepCount 4 / [goto,click,fill,click] / warnings 0 인데 패널은 1행. 새로고침을 관측 가능한 사후조건까지 최대 10회 다시 누르도록, 삭제 뒤 1200ms 고정 대기는 행 수가 줄 때까지 폴링하도록 바꾸고 `before > 1` 단정을 추가했다. 프로덕션 코드 변경 0개(`tools/smoke-test.mjs` 1개 파일).
- **확신 없는 곳·검증 못 한 것**: ① 재현은 **확률적**이다. 전체 스위트를 동시에 돌려 부하를 준 1차 프로브에서 3/3 재현했고(실패 문자열이 verify 로그와 동일), 유휴에서는 3/3 통과, 다시 부하를 걸은 2차 A/B 에서는 current 모드가 2/2 통과했다. 즉 "부하를 걸면 항상 실패" 가 아니라 "부하에 따라 실패" 다 — 수정 후 통과가 수정 때문인지 운인지를 단일 스위트 실행만으로는 구분할 수 없다. 내 근거는 스위트 통과가 아니라 1차 프로브의 측정이다. ② (해소됨) 새 `assert/text` 가 쓰는 `expected` 키와 playground.js 가 보내는 `value` 키는 `server.js:6659` 의 `request.expected ?? request.value` 한 곳에서 같이 읽힌다 — 파서가 둘이 아니다. ③ 도커 빌드·체크리스트 3절은 돌리지 않았다(이 세션은 릴리즈가 아니다).
- 일부러 안 한 것: (a) 패널에 렌더된 녹화 리비전을 표시하는 **제품 쪽** 변경 — 증명된 결함은 검사에 있었고 프로덕션에 테스트 훅을 넣는 것은 별도 정당화가 필요하다(ideas.json 가치 2). (b) 스모크의 나머지 고정 `sleepMs` 대기들 — 같은 결함 부류지만 각각 부하 하에서 실패를 먼저 봐야 한다(ideas.json 가치 3).
- 다음 역할이 조심할 것: **직전 회차(2026-10-03)의 하네스 수정은 머지되지 않았고 main 에 전혀 없다.** base 는 pristine `main@a2a44d4` 이고 `tools/smoke-test.mjs:89` 는 여전히 `waitForHealth(url = baseUrl)` 다 — `superviseServer`, `tools/harness-selftest.mjs`, `npm run test:harness` 모두 없다. 그 PR 을 떨어뜨린 플레이크가 이번에 사라졌으니 다음 1순위는 그 알려진 좋은 diff 의 재상정이다(ideas.json 가치 5). 또 `npm install` 이 `package-lock.json` 의 version 을 0.1.4→0.16.0 으로 고치므로(이 세션 금지 범위) 커밋 전에 되돌렸다 — 그 드리프트는 main 에 그대로 남아 있다. 이 워크트리의 chromium 은 이 회차 HOME 아래에 깔려 다음 회차에는 남지 않는다(설치 수 분, 예산에서 먼저 떼어 둘 것).
- [러너 00:49] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인한 것: 새 `assert/text`(smoke-test.mjs:4100)가 수정 전 상태에서 구조적으로 통과 불가능함을 제품 코드로 역추적했다 — `startRecording` 이 시드하는 `goto` 는 locator 가 없어(server.js:7389) `describeLocator` 가 ''를 돌려주고 1행 패널에는 `testId=` 가 존재할 수 없다(원장 프로브의 `immediateHasTestId:false` 와 일치). 루프 전제도 확인: `onClick` 이 핸들러 동안 버튼을 disabled 로 둔다(playground.js:118-127), `assert/text` 타임아웃은 408 로 돌아오므로 `call` 이 던지지 않는다(server.js:3724), `renderRecording` 은 동기라 재렌더 중간 0행을 못 읽는다(playground.js:414).
- 못 본 것: 이 세션에는 chromium 이 없어 독립 재실행을 하지 않았다. verify.txt:147/152 의 `PASS` + `150/0/0` 기록만 읽었다. 도커 빌드·체크리스트 3절도 미확인.
- 승인이어도 남는 우려 ①: 새로고침 루프는 `assert/text` 가 앞 반복의 렌더로 만족될 수 있어 마지막 GET 이 비행 중인 상태로 탈출 가능하다. 그 응답이 삭제 재렌더 뒤에 도착하면 삭제 폴링이 10초를 태우고 `deleting from the panel left N of N` 으로 실패한다 — 구 코드가 더 취약했으므로 회귀는 아니지만, **플레이크 재발 시 실패 문자열이 달라져 있을 것**이다.
- 우려 ②: 실패 메시지가 구 `JSON.stringify(rows)` 에서 일반 문장(4106)으로 바뀌어, 녹화기가 `testId` locator 를 못 만드는 **제품** 회귀의 진단이 어려워졌다. `assert(before > 1)` 은 주석의 4행이 아니라 2행만 요구하므로 부분 렌더 회귀는 여전히 통과한다(구 코드보다 엄격해졌으므로 결함 아님).
- 보안·법무 차단 없음(테스트 1파일, 엔드포인트·인가·비밀값·의존성·락파일 무변경, 개인정보 무관, revert 로 완전 복구). risk low. 참고로 이 트리의 node 는 v22.23.1 로 프로필의 v20 경고는 해당하지 않았다.
- [러너 00:54] review approved — 리뷰 승인 (risk=low)
- [러너 00:54] pr created — https://github.com/hkjang/playwright-player/pull/1
- [러너 00:58] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)

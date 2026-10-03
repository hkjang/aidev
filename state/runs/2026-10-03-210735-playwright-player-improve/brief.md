# 과제서 — 2026-10-03-210735-playwright-player-improve

- **과제: 서버가 부팅 중 죽었으면 `waitForHealth` 가 56초를 헛돌지 않고 즉시 실제 원인을 보고하게 하기 (가치 5 / 위험 1 / 작업량 S)**

## 왜
이 저장소의 유일한 자동 검증은 `npm test`(= `node tools/smoke-test.mjs`) 인데, 서버 프로세스가 부팅 중 즉시 죽어도 harness 는 그 사실을 알아보지 못하고 `waitForHealth()` 의 120회×500ms 루프를 끝까지 돌린다. 이 워크트리에서 실제로 재현했고 **56.1초**를 소모한 뒤 남는 것은 `FAIL  harness — server did not become healthy on http://127.0.0.1:3911` 한 줄뿐이다(아래 "확인한 재현" 참고). 직전 회차가 `error: agent produced no result (TIMEOUT)` 로 결과 없이 끝났고, 검증 명령이 원인을 말하지 않은 채 분 단위로 시간을 먹는 이 경로가 1순위 용의자다. 고치면 모든 후속 회차의 첫 검증이 56초 → 1초 미만으로 줄고, 실패 메시지가 바로 조치 가능한 내용(`Cannot find module 'swagger-ui-dist/package.json'`)이 된다.

## 확인한 재현 (이 워크트리에서 실제로 실행)
```
$ node --version
v20.19.2
$ test -d node_modules → ABSENT
$ time npm test 2>&1 | tail -3
Node.js v20.19.2

FAIL  harness — server did not become healthy on http://127.0.0.1:3911

real	0m56.137s
user	0m0.399s     ← 서버는 0.3초 안에 죽었다. 56초는 전부 죽은 포트를 폴링한 시간이다.
```
서버 프로세스의 `MODULE_NOT_FOUND` 스택은 `child.on("exit")` 핸들러(`tools/smoke-test.mjs:159-163`)가 `console.error` 로 따로 토해내지만, `record()` 가 남기는 FAIL 줄에는 들어가지 않는다. 즉 요약만 읽는 사람·에이전트에게는 원인이 보이지 않는다.

## 수용 기준
1. **node_modules 가 없는 체크아웃에서 `npm test` 가 10초 안에** 0 아닌 코드로 끝난다 (현재 56초). 이 워크트리 그대로 측정 가능하다 — `npm install` 전에 먼저 확인할 것.
2. 그 때 출력되는 `FAIL  harness — …` 줄의 detail 에 **서버의 종료 코드와 stderr 첫 줄이 포함**된다. 즉 `Cannot find module 'swagger-ui-dist/package.json'` 가 요약 줄에서 읽힌다. `did not become healthy` 만 남으면 미달이다.
3. `waitForHealth` 를 쓰는 **세 호출 지점 모두** 같은 성질을 갖는다 — 프로세스가 죽으면 즉시 reject, 살아 있으면 기존 60초 예산 유지:
   - `start()` — `tools/smoke-test.mjs:141-166` (`child`, `serverLog`)
   - `spawnServer()` — `tools/smoke-test.mjs:256-267` (`proc`, `log`), restart recovery 검사 내부
   - `startExtraServer()` — `tools/smoke-test.mjs:105` 부근 (`process2`), URL_ALLOWLIST 검사용
4. **테스트가 증명해야 하는 것**: 부팅 중 죽는 서버를 harness 가 60초 기다리지 않는다는 것. 손으로 만든 대역을 쓰지 말고 **이미 저장소에 있는 실제 실패 배선**을 쓸 것 — `tools/smoke-test.mjs:1975-2007` 의 검사는 토큰이 중복된 `principals.json` 을 깔면 서버가 부팅에 실패하며 stderr 에 `shares a token` 을 남긴다는 것을 이미 실증한다. 같은 fixture 로 서버를 띄우고 새 startup 헬퍼가 **5초 안에** 종료 코드와 stderr 를 담아 reject 하는지 assert 하는 검사를 하나 추가한다(브라우저 불필요 → 오프라인에서도 돈다).
5. 정상 환경(`npm install` 완료)에서 `npm test` 의 pass/fail/skip 집계가 **새 검사 1건 추가분을 빼면 그대로**다. SKIP 개수는 건드리지 말 것.

## 건드릴 파일
- `tools/smoke-test.mjs` — **이 파일 한 개만.**
  - `waitForHealth(url = baseUrl)` (85-96행) — 감시할 자식 프로세스를 받게 한다(예: `waitForHealth(url, proc)`). 루프 각 회차에서 프로세스가 이미 종료했는지 확인하고(`proc.exitCode !== null` 또는 `exit` 이벤트를 미리 걸어 둔 플래그), 종료했으면 남은 대기를 버리고 종료 코드 + 수집된 로그의 첫 의미 있는 줄을 담아 throw 한다. 인자를 받지 않으면 지금과 똑같이 동작해야 한다(기존 호출 호환).
  - `start()` (141-166행) — `serverLog` 를 이미 모으고 있다. 이것을 새 헬퍼에 넘겨 실패 메시지에 넣는다. `child.on("exit")` 의 `console.error(serverLog.join(""))` 는 지우지 말 것(전체 스택은 여전히 유용하다).
  - `spawnServer()` (256-267행) — `log` 를 같은 방식으로 넘긴다.
  - `startExtraServer()` (105행 부근) — `process2` 와 그 stderr 버퍼를 같은 방식으로 넘긴다. stderr 를 아직 버퍼에 모으지 않으면 모으게 한다.
  - 수용 기준 4의 새 검사 추가.

## 검증 명령
```bash
# 1) 고친 직후, npm install 하기 전에 — 56초가 10초 안으로 줄고 원인이 요약에 나오는지
cd /home/hkjang/.cache/auto-improve-wt/playwright-player
time npm test 2>&1 | tail -5          # 기대: 10초 미만, FAIL 줄에 "Cannot find module 'swagger-ui-dist/package.json'"

# 2) 문법
npm run check                          # node --check server.js — 의존성 없이 돈다

# 3) 회귀 — 의존성을 깔고 전체 suite
npm install                            # 수 분 걸린다. 시간 예산을 여기에 먼저 떼어 둘 것
npm test 2>&1 | tail -30               # 기대: 마지막 "N passed, 0 failed, M skipped"
```

## 위험과 피할 것
- **`server.js` 를 건드리지 말 것.** 11581행이고 이 과제와 무관하다. 수정 범위는 `tools/smoke-test.mjs` 하나다.
- **검사를 느슨하게 만들어 통과시키는 것은 금지.** 이 과제는 실패를 *빨리, 정확히* 내는 것이다. 60초 예산 자체를 줄이거나 검사를 skip 으로 바꾸는 방향이 아니다. 서버가 **살아 있는데** 아직 안 떴을 때는 기존 60초를 그대로 줘야 한다 — 느린 머신에서 거짓 실패를 만들면 안 된다.
- **SKIP 집계를 건드리지 말 것.** 보류 아이디어에 "그룹 SKIP 과 검사 SKIP 이 두 줄로 찍힌다"(smoke-test.mjs:1313/1546/1834)가 있지만 **이번 회차 범위가 아니다.** 손대면 skipped 기대값이 바뀌어 다른 기록과 어긋난다. 같은 파일이니 유혹이 있을 것 — 참을 것.
- **node 버전 함정(미확인).** 이 환경의 node 는 **v20.19.2** 인데 `package.json` 의 `engines` 는 `>=22` 다. nvm 디렉터리는 이 세션의 권한으로 확인하지 못했다. `npm install` 후 전체 suite 가 node 20 에서 초록이 되는지는 **확인하지 못했다** — 만약 node 20 때문에 무관한 검사가 깨지면 그것은 이 과제의 회귀가 아니다. 그 경우 수용 기준 1·2(의존성 없이 측정 가능)로 판정하고 노트에 적을 것. `engines` 를 고치는 것도 이번 범위가 아니다.
- **보호 경로**: `docs/RELEASE_CHECKLIST.md` 가 릴리즈 검증을 `SMOKE_REQUIRE_BROWSER=1 npm test` 로 규정한다. 이 과제는 `requireBrowser` 분기를 전혀 건드리지 않으므로 릴리즈 계약은 그대로다 — 건드리지 말 것.
- **이미 비용을 아는 자리**: 직전 회차가 TIMEOUT 으로 아무 결과도 남기지 못했다. `npm install` 이 길다. **수용 기준 1·2 를 먼저 끝내고 커밋 가능한 상태를 만든 뒤** `npm install` 로 넘어갈 것. 순서를 뒤집으면 같은 TIMEOUT 을 되풀이한다.

## 차선 후보
**node_modules 부재를 `npm test` 시작 시점에 알아보고 한 줄로 안내하기** — `tools/smoke-test.mjs` 맨 앞에서 `node_modules/swagger-ui-dist` 존재를 확인하고 없으면 `npm install 을 먼저 실행하세요` 로 즉시 종료. 1순위가 성립하지 않을 때(예: `waitForHealth` 구조가 생각보다 얽혀 있을 때) 이쪽이 같은 증상의 더 얕은 처방이다. 1순위와 **같은 파일·같은 증상이므로 둘을 동시에 하지 말 것** — 1순위가 더 일반적인 수정(어떤 부팅 실패든 잡는다)이므로 그쪽을 우선한다.

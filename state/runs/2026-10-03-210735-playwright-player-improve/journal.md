# 회차 노트 2026-10-03-210735-playwright-player-improve — playwright-player
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:07] base pinned — main@a2a44d4
- [러너 21:07] autonomy release — 

## 정찰 노트
- 왜 이것인가: `.github/workflows` 가 없으므로 "릴리즈 워크플로" 는 `docs/RELEASE_CHECKLIST.md` 1절(`SMOKE_REQUIRE_BROWSER=1 npm test`) 이고, 실패 단계의 스크립트는 `tools/smoke-test.mjs` 하나다. 실측으로 원인을 잡았다 — 워크트리에 node_modules 가 없어 서버가 0.3초에 죽는데 harness 는 모르고 **56.137초**를 폴링한 뒤 `server did not become healthy` 만 남긴다. 직전 회차가 TIMEOUT 으로 결과 없이 끝난 것과 정확히 같은 성질(원인을 말하지 않고 벽시계를 먹는다)이라 이것을 1순위로 골랐다.
- 제친 후보: 녹화 중복 제거(가치 4)는 `npx playwright install chromium` 대용량 다운로드가 선행돼야 해서 TIMEOUT 직후 회차에 벽시계 위험이 가장 컸다. SKIP 두 줄 정리는 skipped 집계를 바꿔 다른 기록과 어긋나고 1순위와 같은 파일이라 제외. 직전 회차의 "스모크 2건 거짓 FAIL" 은 재선택하지 않았다 — 같은 브라우저 설치 선행 비용에 걸릴 자리다.
- 추측으로 적은 것(미확인): `startExtraServer()` 가 `waitForHealth` 를 호출한다고 적었으나 해당 줄을 직접 열어 확인하지 못했다(105행 부근). node 20.19.2 에서 전체 suite 가 초록인지도 미확인 — `engines` 는 `>=22` 다. 직전 회차가 추가했다는 `{ skip: reason }` 반환 경로는 **머지되지 않았다**(HEAD 는 a2a44d4) — 보류 아이디어의 그 전제는 틀렸고 ideas.json 에서 고쳐 적었다.
- 구현자가 조심할 것: **수용 기준 1·2 는 `npm install` 없이 이 워크트리에서 그대로 측정된다. 그것을 먼저 끝내고 커밋 가능한 상태를 만든 뒤 `npm install`(수 분) 로 넘어갈 것** — 순서를 뒤집으면 같은 TIMEOUT 을 되풀이한다. 그리고 검사를 느슨하게 하는 방향(60초 예산 축소·skip 승격)은 금지다. 서버가 살아 있는데 느린 경우에는 기존 60초를 그대로 줘야 한다.
- [러너 21:15] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇을 왜: 릴리즈 게이트는 `npm test` 의 **요약 줄과 종료 코드**를 읽는데, 서버 부팅이 실패하면 `stop()` 이 **이미 끝난** 자식의 `exit` 를 기다려 영원히 settle 되지 않았다(escalation 타이머가 `.unref()` 라 루프가 비면 못 뛴다). 결과: 종료 코드 **13**, 요약 줄 없음, 56.2초 소모 → 게이트에는 "결과 없음(TIMEOUT)" 으로 보인다. `stop()` 이 `exitCode/signalCode` 를 먼저 보게 하고, `waitForHealth(url, supervised)` 가 자식의 죽음을 즉시 서버 stderr 와 함께 보고하게 했다. `startExtraServer` 의 stderr 는 pipe 만 하고 아무도 읽지 않아(진단 유실 + 파이프 교착) superviseServer 로 배수한다. 측정: 56222ms/exit 13/4 FAIL → 558ms/exit 1/4 PASS, 버그를 되돌려 재확인.
- 검증한 것: 릴리즈 게이트 `SMOKE_REQUIRE_BROWSER=1 npm test` = **150 passed, 0 failed, 0 skipped, exit 0**. pristine HEAD 베이스라인도 150/0/0 — 체크 총수가 같아 내 변경이 검사를 늘리거나 줄이지 않았음을 확인했다.
- **확신 없는 곳·검증 못 한 것**: ① 게이트 **첫 실행**은 `a run records per-step durations` 가 FAIL 했다(`Launch browser` 의 `durationMs: -885`). 원인을 추적했다 — 그 실행만 chromium 설치 직후의 **최초 콜드 런치**였다. 같은 트리를 워밍 후 재실행하면 150/0/0 이고 pristine HEAD(워밍)도 150/0/0 이라, 실패는 내 diff 가 아니라 콜드 런치에 붙어 있다. 다만 **근본 원인(WSL2 클럭 점프 의심)은 증명하지 못했고, 재현도 1회뿐**이다. server.js:1196/1252 가 Playwright 의 `step.duration` 을 그대로 통과시키므로 음수가 API 로 그대로 나간다 — 방어적 정규화가 타당해 보이지만 결정적 재현이 없어 손대지 않았다(ideas.json 에 증거와 함께 남겼다). **비평가는 이 자리를 먼저 볼 것.** ② 도커 빌드·체크리스트 3절 이미지 검증은 돌리지 않았다(이 세션은 릴리즈가 아니다). ③ node 20 미확인 — 이 워크트리 node 는 v22.23.1 이고 정찰 노트의 v20.19.2 는 **틀렸다**.
- 일부러 안 한 것: `waitForHealth` 호출 11곳(그룹별 spawn)은 여전히 핸들 없이 호출돼 죽은 서버에 60초를 쓴다. 한 토큰씩이면 되지만 측정된 재현이 있는 경로만 고쳐 프로덕션 파일을 **1개**로 묶었다 — ideas.json 가치 4.
- 다음 역할이 조심할 것: `npm run test:harness` 는 포트(기본 **3951**, 스모크 범위 3911~3923 밖)를 직접 점유해 부팅을 의도적으로 깨뜨린다. node_modules 유무와 무관하게 돈다(각각 EADDRINUSE / MODULE_NOT_FOUND). 약 0.6초. 이 워크트리에는 `npm install` 과 `npx playwright install chromium` 을 모두 끝내 뒀다 — 다음 회차는 브라우저 선행 비용이 없고, **첫 콜드 런치는 이미 소비됐다**. `package-lock.json` 은 npm install 이 version 0.1.4→0.16.0 으로 고친 것을 **되돌렸다**(버전 필드는 이 세션 금지 범위) — 그 드리프트는 여전히 남아 있다.
- [러너 21:54] verify failed — 실패한 검증: npm test --silent (exit 1)

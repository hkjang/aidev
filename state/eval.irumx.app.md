## 2026-10-07
- 선택: 예산·시험 수 한도 동시 예약 시험 (설계 수용 시험 12번) (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `usage.ts:reserveSql` 은 한도 검사와 INSERT 를 조건부 INSERT 한 문장에 넣어 동시 예약을 막게 설계됐는데 그 동시성을 밟는 시험이 없었다. `tests/api-targets.spec.ts` 에 동시 예약 시험 1개를 더했다 — `estimate` 로 예약 없이 한 건의 시험 수를 재고, `GET /api/w/:ws/usage` 로 기준선을 읽어 딱 2건만 들어가는 한도를 맞춘 뒤 같은 업체 공간에 `POST .../experiments` 5건을 한꺼번에 보내 201 이 정확히 2개·나머지는 409 over_budget 인지, 끝으로 `trials` 합계가 기준선+성공분과 같은지 본다. 프로덕션 로직 변경 0개(util.ts 주석 1줄에서 없는 파일 `tests/guard.spec.ts` 참조만 실제 시험 경로로 고쳤다). 검증: `npm run build && npm test`.
- 실패 재현: `Error: 201,201,201,201,201` / `expect(received).toBe(expected) Expected: 2 Received: 5` — `reserveSql` 의 시험 수 조건을 빼고 같은 검사를 JS 로 먼저 읽게(SELECT 후 INSERT) 바꾼 상태에서 새 시험만 실패했고 **기존 순차 예산 시험은 그대로 통과**했다 — 새 시험이 기존 시험이 못 보는 바로 그 구멍을 막는다는 증거.
- 보류 아이디어: 큐 중복 전달 시험(수용 시험 10, 4/3/M — 큐에 두 번 넣는 시험 경로가 있는지 미확인) · 비밀 모양 문자열 가림을 원문·보고서 출력 경로로만 좁히기(수용 시험 23, 3/4/M) · build+unit 만 도는 최소 CI 워크플로(3/2/M, 보호 경로라 운영자 판단 필요) · D1 바인딩 100개 한도 배치 쪼개기 경계 단위 시험(3/1/S) · 시험 보조 수단 기록: `wrangler dev` 가 `dist/` 의 빌드 결과를 띄운다는 점을 docs 에 적기(2/1/S)
- 과제서: 채택 — 과제서의 근거(reserveSql 조건부 INSERT · 시험 공백 · estimate/usage 경로)는 코드와 그대로 맞았다. 다만 과제서가 적은 `api.raw` + `Promise.all` 로는 Playwright 요청 컨텍스트가 요청을 하나씩 보내 동시성을 전혀 밟지 못했고(회귀를 심어도 통과), 쿠키를 꺼내 Node 전역 `fetch` 로 보내야 비로소 재현됐다.

## 2026-10-07
- 선택: 시험 세션 캐시와 로컬 D1 비우기가 어긋나 두 번째 실행이 깨지는 것 고치기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `scripts/db-local.mjs` 가 `--persist-to` 디렉터리만 지우고 `tests/helpers.ts` 가 쓰는 `.wrangler/test-sessions`(로그인 쿠키 보관)는 남겨, `auth.ts` 의 `cookieCache`(5분) 때문에 DB 를 비운 직후에도 남은 쿠키로 `/api/me` 가 사람을 돌려주고 `workspaces` 만 비어 `ownerApi` 가 TypeError 로 터졌다. db-local 이 세션 보관도 함께 지우게 하고(한 줄), `login` 의 캐시 재사용 조건에 "세션이 DB 에 실제로 있다"는 신호(`me.workspaces?.length || me.operator`)를 더하고, `ownerApi` 는 업체 공간이 없으면 한국어 안내로 throw 하게 했다. 프로덕션 코드 0개 — 시험 보조 2개. 검증: 재현(끄고 5분 안에 재실행 → TypeError) → 고친 뒤 같은 절차로 두 번째 실행 통과 · 세션 캐시만 남고 DB 가 바뀐 상태를 억지로 만든 실행에서도 통과(캐시를 버리고 재로그인) · 업체 공간 없는 계정(`ops@eval.test`)에서 한국어 안내가 나오는지 임시 시험으로 확인(확인 뒤 삭제) · `npm run build && npm test` 전체.
- 실패 재현: `TypeError: Cannot read properties of undefined (reading 'id')` / `at ownerApi (tests/helpers.ts:94:38)` — run1 16:25:26 통과 후 `wrangler dev`(:8860)를 끄고 16:25:29 에 같은 명령(`npx playwright test --project=api --grep "주소 검사"`)을 다시 돌려 재현.
- 보류 아이디어: 예산·시험 수 한도 동시 예약 시험(수용 시험 12, 4/2/S — main 머지 상태 먼저 확인) · 예산 시험이 바꾼 업체 공간 한도를 teardown 으로 보장해 되돌리기(3/2/M) · 큐 중복 전달 시험(수용 시험 10, 4/3/M — 프로덕션 훅 필요) · 비밀 모양 문자열 가림을 원문·보고서 출력 경로로만 좁히기(수용 시험 23, 3/4/M) · 워크트리에 node_modules 가 없을 때 build 가 subset-font 스택만 남기고 죽는 것을 안내로 바꾸기(2/1/S)
- 과제서: 채택 — 과제서가 지목한 다섯 줄(db-local 의 rmSync · helpers 의 storageState 경로 · cookieCache 300초 · me.ts 의 workspaces 별도 조회 · ownerApi 의 `workspaces[0].id`)이 코드와 그대로 맞았고, 미확인으로 표시한 "5분 창 안에 두 번 돌려야 재현된다" 는 타이밍도 그대로 재현됐다.

## 2026-10-07
- 선택: GitHub Actions 검증 워크플로 추가 — 러너가 돌리는 검증 명령을 그대로 CI 로 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `.github/` 가 없어 러너 PR 이 전부 "CI 검사 없음" 으로 막혀 있었다. 러너가 이미 로컬에서 돌리는 두 명령(`npm run build` → `npm test`)을 그대로 `.github/workflows/ci.yml` 한 파일로 옮겼다(push·pull_request, ubuntu-latest, node 22, `npm ci`, `npx playwright install --with-deps chromium`, 실패 시 `test-results/` 올리기). 새 시험·고친 시험·프로덕션 변경 0개 — 바꾼 파일은 이 한 개뿐이다. 검증: CI 와 같은 조건(Cloudflare 자격증명 없음·빈 임시 홈·`npm ci` 로 새로 받은 의존성)의 워크트리에서 `npm run build`(종료 0, "빌드 점검 통과") 와 `npm test`("50 passed (1.5m)" — unit·api·desktop·mobile 전부) 를 실제로 돌려 통과를 확인했고, `python3 -c "yaml.safe_load(...)"` 로 워크플로가 파싱되고 `on`·steps 가 의도대로 읽히는지 확인했다.
- 실패 재현: 못 함 — 이 회차는 결함 수정이 아니라 이미 통과하는 검증 명령을 CI 로 옮기는 작업이라 "고치기 전에 실패하는 테스트" 가 성립하지 않는다. 대신 CI 에서 실제로 돌 수 있는지를 두 단계로 밟았다: ① 브라우저가 없는 상태에서 `npm test` → `browserType.launch: Executable doesn't exist at .../chromium_headless_shell-1243/...` 로 desktop·mobile 4개 실패(46 passed) — 워크플로에 `npx playwright install` 단계가 필요하다는 증거 ② chromium 을 받은 뒤 깨끗한 상태에서 다시 → `50 passed (1.5m)`.
- 보류 아이디어: 예산·시험 수 한도 동시 예약 시험이 main@202c416 에 아직 없다 — 머지 상태 확인 후 재적용(4/2/S) · 예산·취소 시험이 바꾼 업체 공간 한도를 `test.afterEach` 로 되돌리기 보장(3/2/M) · `tests/helpers.ts:94` 의 `me.workspaces[0].id` TypeError — 서버를 다시 띄워 로컬 D1 이 비워졌는데 세션 캐시가 남으면 이 줄에서 원인 모를 TypeError 가 난다(이번 회차에서 다시 맞았다, 3/1/S) · 큐 중복 전달 시험(수용 시험 10, 4/3/M — 프로덕션 훅 필요) · 비밀 모양 문자열 가림을 원문·보고서 출력 경로로만 좁히기(수용 시험 23, 3/4/M)
- 과제서: 채택 — 과제서가 지목한 그대로였다(`.github` 없음, 러너의 검증 명령은 `npm run build && npm test`). 과제서가 적지 않은 사실 하나: 시험이 브라우저 프로젝트(desktop·mobile)를 포함하므로 CI 에 `npx playwright install --with-deps chromium` 단계가 반드시 필요하다.


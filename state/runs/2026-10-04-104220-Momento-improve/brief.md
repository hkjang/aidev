# 과제서 — 2026-10-04-104220-Momento-improve

## 먼저: 배정된 「우선 과제」는 이미 main 에서 해결돼 있다 (증거 포함)

배정 문구는 "마지막 회차가 verify-failed, CI failed, PR open #23 / 릴리즈 워크플로가 같은 이유로 두 번 실패" 였다. 확인한 결과 **그 실패는 이미 고쳐져 릴리즈까지 났다**:

- 실패 기록 원문(`2026-10-04-090203-Momento-improve/ci-failure-60c4733bc0d2.txt`): 잡 `test`, 실패 단계 `Run npm ci && npm audit && npm run lint && npm test && npm run build` → `.github/workflows/ci.yml:55` (web). exit 1.
- 셰퍼드 회차(`2026-10-04-094340-Momento-shepherd`)가 `f539c2f fix(web): npm audit 를 막던 braces 취약 경로를 끊도록 typescript-eslint 를 8.48.0 으로 올린다` 로 고쳤고 PR #23 은 `3d18b32` 로 **머지됐다**.
- 그 뒤 main 끝 `ec7cb66 release: v0.34.55` 의 체크 4개 전부 `"conclusion": "success"` — `test`(01:00:31→01:05:55), `build`, `report-build-status`, `deploy` (`2026-10-04-095112-Momento-approve/ci-ec7cb668ffd9.json`).
- 수정이 실제로 뿌리를 끊었는지 직접 확인: `grep -n '"node_modules/braces"|"node_modules/micromatch"' sdk/package-lock.json web/package-lock.json` → **둘 다 0건**. 두 락파일에서 취약 경로가 사라졌다.
- `release.json` 의 `"github_release": false, "assets": []` 는 **실패 신호가 아니다** — v0.34.40·41·42·45·46·47·48·53·54·55 전부 같은 값이다(러너는 태그만 밀고 `release.yml` 이 퍼블리시한다). 즉 「릴리즈 워크플로가 두 번 실패」를 뒷받침하는 증거는 로컬에 없다.
- **미확인**: 이 샌드박스에서 `gh run list` 와 `npm ci` 가 모두 권한/네트워크로 막혀 `release.yml`("Offline image release") 의 런 이력은 직접 못 봤다. 구현자가 `gh run list --workflow=release.yml --limit 8` 을 돌릴 수 있다면 **먼저 그것을 보라**. 거기서 v0.34.54·55 가 실제로 붉다면 그 실패가 이번 과제이고 아래 과제는 차선으로 내린다.
- 같은 게이트가 `sdk` 에도 있어(`ci.yml:53`) 전염 여부를 봤는데 `sdk/package.json` 의 devDependencies 는 `esbuild 0.25.9` + `typescript 5.9.2` 뿐이고 braces 경로가 없다 → **전염 없음**.
- v0.34.53 이 web 에서 고친 `node` PATH 가림이 `sdk/package.json:9` 의 맨 `node --test` 에도 남아 있지만, `sdk/test/*.test.mjs` 두 개는 `node:assert`·`node:test`·`node:fs` 만 import 하고 `.ts` 를 전혀 import 하지 않으므로 **타입 스트리핑이 필요 없다 → 노출되지 않는다**(확인함). 이 아이디어는 rejected 로 내렸다.

따라서 "워크플로를 느슨하게 하지 말고 원인을 고치라" 는 요구에 대해 **고칠 원인이 남아 있지 않다**. `npm audit` 게이트에 `--audit-level` 을 붙이는 것은 바로 그 금지된 완화이므로 하지 않는다. 이번 회차는 아래 과제를 수행한다.

---

- 과제: 망 구분(네트워크) 추가 실패를 서버의 영문 문장·pgx 원문 대신 한국어로 안내한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `NetworksAdmin` 의 추가 Alert 이 `web/src/pages/AdminPage.tsx:3244-3246` 에서 `<Alert severity="error">{create.error.message}</Alert>` 로 서버 문장을 그대로 띄운다. CIDR 을 손으로 적는 칸이 있어 가장 흔한 실패가 `INVALID_CIDR`("CIDR is invalid") 이고, 500 일 때는 `admin.go:823` 의 `err.Error()` 가 pgx 원문째로 브라우저까지 흐른다. v0.34.54(사용자)·v0.34.55(사이트)가 같은 틀로 두 회차 연속 통과했으니 세 번째 조각을 같은 자리에 넣으면 끝난다.

- 수용 기준:
  1. 추가 다이얼로그가 `INVALID_CIDR`·`INVALID_PAYLOAD`·`NETWORK_CREATE_FAILED` 를 각각 **한국어**로 안내하고, `INVALID_CIDR` 안내는 사용자가 다음에 할 행동(예: `10.0.0.0/24` 처럼 넷마스크 경계에 맞춘 표기)을 말해 준다.
  2. 500(`NETWORK_CREATE_FAILED`) 의 **Alert 본문**에 Postgres/pgx 표지(`SQLSTATE`, `violates`, `relation`, `invalid cidr value`, `no rows in result set`)가 하나도 없다. 서버 원문은 지우지 않고 `detail` caption 으로만 남긴다(v0.34.55 가 정한 관례).
  3. 모르는 코드는 서버 메시지로 되돌린다(`default` 분기). 기존 `describeUserError`·`describeSiteError` 테스트 13건이 **글자 하나도 바뀌지 않고** 그대로 통과한다.
  4. 테스트가 증명할 것: 지금 Alert 과 똑같이 동작하는 **항등 스텁**(`error.message` 를 그대로 돌려주는 구현)으로 새 케이스가 실패해야 한다 — 즉 테스트가 결함을 짚는지 먼저 확인할 수 있어야 한다.

- 건드릴 파일 (프로덕션 2 + 테스트 1):
  - `web/src/pages/adminErrors.ts` — `describeNetworkError(error): UserErrorNotice` 를 **새로** 추가. `describeSiteError`(197행)를 그대로 본떠 `refusal(error)`(65행)로 code·message 를 읽고 `switch(code)`. 기존 공용 상수 `PASS_TO_ADMIN`(78) · `UNREADABLE_PAYLOAD`(80) · `LOST_REQUEST`(82) 를 **재사용**하고 새 문장을 만들지 말 것. `describeSiteError`·`describeUserError` 와 **합치지 말 것**(핸들러가 다르고 고칠 것이 다르다 — v0.34.55 가 같은 판단을 했다).
  - `web/src/pages/AdminPage.tsx` — (a) 78행 import 에 `describeNetworkError` 추가, (b) `SiteErrorAlert`(3343-3345) 바로 옆에 같은 모양의 `NetworkErrorAlert({error})` = `<AdminNoticeAlert notice={describeNetworkError(error)} />` 추가, (c) 3244-3246 의 Alert 을 `{create.error && <NetworkErrorAlert error={create.error} />}` 로 교체. `AdminNoticeAlert`(3350)·`UserErrorNotice`(adminErrors.ts:28)·`describeUserError`·`describeSiteError` 는 손대지 말 것.
  - `web/test/adminErrors.test.mjs` — 새 케이스 추가. 기존 `apiError` 헬퍼를 그대로 쓸 것. **import 는 `"../src/pages/adminErrors.ts"` 처럼 `.ts` 확장자까지 적어야 한다**(빼면 `ERR_MODULE_NOT_FOUND`).

- 서버 쪽 사실 (읽고 확인한 것 — 문구를 거짓으로 쓰지 않도록)
  - `createNetwork` 는 `internal/httpapi/admin.go:800`. 오류는 셋뿐이다: `INVALID_PAYLOAD`(400, `decodeJSON` 의 `err.Error()`), `INVALID_CIDR`(400, `"CIDR is invalid"`, 813행 — `net.ParseCIDR` 실패), `NETWORK_CREATE_FAILED`(500, `err.Error()`, 823행 — INSERT 실패).
  - **`network_ranges` 에 UNIQUE 제약이 없다** — `internal/database/migrations/001_initial.sql:64-70` 은 `id PRIMARY KEY`, `name text NOT NULL`, `cidr cidr NOT NULL`, `description`, `internal`, `created_at` 뿐이다. 그러니 **500 을 "이미 등록된 망/CIDR" 이라고 설명하면 거짓이다.** v0.34.55 가 `sites.name` 에서 똑같은 함정을 피했다 — 500 문구는 중립으로 두라.
  - **추측(미확인, 기대지 말 것)**: 컬럼 타입이 Postgres `cidr` 이고 Go 는 `net.ParseCIDR` 로만 검사하므로 `10.0.0.5/24` 처럼 넷마스크 오른쪽에 비트가 선 값은 Go 를 통과하고 DB 가 거절할 가능성이 있다(그렇다면 `INVALID_CIDR` 400 이 아니라 `NETWORK_CREATE_FAILED` 500 으로 온다). **Postgres 로 재현하지 않았다.** 재현 없이 500 문구를 "넷마스크 경계" 로 특정하지 말고 중립으로 두되, 기준 1 의 표기 안내는 재현이 필요 없는 `INVALID_CIDR`(400) 쪽에 달아라.
  - 화면에서 `disabled={!form.name || !form.cidr}`(3250행 근처)가 빈 이름·빈 CIDR 을 이미 막으므로 `INVALID_PAYLOAD` 는 드물다 — 문구에 공들이지 말고 `UNREADABLE_PAYLOAD` 재사용으로 끝낼 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run lint && npm test && npm run build` — test 는 `"${npm_node_execpath:-node}" --test test/*.test.mjs`. 현재 191건이고 **늘어난 수만큼만** 늘어야 한다.
  - 특정 파일만: `cd web && node --test test/adminErrors.test.mjs`
  - Go 는 손대지 않지만 혹시 건드렸으면 `go vet ./internal/httpapi/`
  - **주의**: 이 정찰 세션에서는 `npm ci` 와 `gh` 가 둘 다 막혀 **아무 명령도 실행하지 못했다**. 위 명령은 지난 다섯 회차가 실제로 돌려 통과시킨 것들이고, worktree 에 `web/node_modules` 가 없어 `npm ci` 가 수 분 선행한다.

- 위험과 피할 것:
  - `.github/workflows/*` 를 건드리지 말 것 — 이번 회차의 배정 실패는 이미 머지돼 해결됐고, `npm audit` 게이트 완화는 명시적으로 금지다.
  - `internal/httpapi/admin.go` 를 건드리지 말 것. 서버 검사가 권한·검증의 정본이고 화면은 거울일 뿐이다. 이번 과제는 **화면 문구만**이다.
  - `UNIQUE` 가 없는 테이블에 "중복" 문구를 붙이는 것 — 위에 적은 대로 거짓이 된다.
  - `.ts` 확장자 누락(값 import 에도 필요하다 — type-only 만 예외).
  - 기존 테스트는 출력 문자열을 **글자 그대로** 단언한다. 공용 상수 세 개의 문장을 고치면 사용자·사이트 테스트가 함께 깨진다 — 재사용만 하고 수정하지 말 것.
  - `npx prettier --check` 를 게이트로 쓰지 말 것(저장소에 prettier 의존성·CI 단계가 없고 main 의 여러 파일이 이미 실패한다).
  - 임시 검증 하네스와 `web/dist` 는 커밋 전에 지우고 `git status` 로 3파일만 담긴 것을 확인할 것.

- 차선 후보: **`gh run list --workflow=release.yml --limit 8` 이 실제로 붉은 런을 보여 주면 그것이 1순위다**(배정이 가리킨 「두 번 실패」). 그게 아니라면 2순위는 남은 저장·재발급 계열 Alert 중 **한 핸들러만** — `AdminPage.tsx:2410` 한 곳과 그 서버 핸들러. 핸들러 단위로 쪼갠 두 회차가 모두 통과했으니 묶지 말 것.

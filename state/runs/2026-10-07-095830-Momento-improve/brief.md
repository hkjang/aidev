# 과제서 — 2026-10-07-095830-Momento-improve (수정 과제)

> 먼저 읽을 것: 배정된 실패(「릴리즈 워크플로가 같은 이유로 두 번 실패」, PR #27)의 **직접 원인은 이미 main@9d82c28 에 고쳐져 있다**.
> 이 세션에서 확인한 것: `97e3063` 이 `web/package-lock.json` 의 `node_modules/source-map-js` 를 1.2.2 로 올렸고(3723-3732행: version·resolved·integrity 모두 1.2.2 로 정합),
> `sdk/package-lock.json` 에는 `source-map-js` 항목이 아예 없다(`grep -n source-map-js web/package-lock.json sdk/package-lock.json` 로 확인).
> 이전 회차 shepherd 의 기록(`state/runs/2026-10-07-090652-Momento-shepherd/fix-summary.md`)도 같은 말을 한다:
> 실패한 단계는 ci.yml:55 의 `npm ci && npm audit && …`(working-directory: web)이고 원인은 `source-map-js@1.2.1` 의 GHSA-68fv-2mgg-jv7q(high)였다.
> **그 3줄을 다시 고치는 것은 no-op 이다.** 아래 과제는 "같은 이유로 두 번" 이 세 번째가 되지 않게 하는, 같은 실패의 남은 절반이다.
>
> **미확인(이 세션의 한계)**: 이 환경의 샌드박스가 네트워크를 막아 `gh run list`/`gh run view` 와 `npm audit`·`npm ci` 를 **한 번도 실행하지 못했다**.
> 따라서 "실패한 단계가 ci.yml:55 였다" 는 shepherd 기록과 커밋 메시지(97e3063)에 근거한 것이고 CI 로그로 직접 보지 않았다. `web/node_modules`·`sdk/node_modules` 는 둘 다 없다.
> 구현자는 네트워크가 있으면 **먼저 `cd web && npm ci && npm audit` 를 돌려 지금 main 이 녹색인지 확인**하고 시작하라. 만약 거기서 **다른** 권고(advisory)로 빨간불이면 그것이 이번 과제다 — 그때는 아래 1순위를 버리고 그 권고를 이 저장소의 관례(`overrides` + 잠금, 아래와 같은 방식)로 막아라.

- 과제: `web` 의 보안 하한을 매니페스트와 잠금 파일이 **함께** 담게 맞춘다 — `overrides` 에 `source-map-js` 를 더하고, 잠금 파일 루트의 `overrides` 누락을 없앤다 (가치 3 / 위험 2 / 작업량 S)

- 왜: 최근 두 회차의 CI 적신호가 모두 `npm audit` 이 **dev 전용 전이 의존성**에서 터진 것이었고, 두 번 다 수리가 `package-lock.json` 을 손으로 몇 줄 고치는 것으로 끝났다. 그런데 이 저장소는 이미 그 상황을 위한 자리를 갖고 있다 — `web/package.json:41-43` 의 `"overrides": { "js-yaml": "^4.3.2" }`. `source-map-js` 는 그 자리에 들어가지 않았고(매니페스트에는 흔적이 없다), 더해 **잠금 파일 루트 `packages[""]`(web/package-lock.json:7-39)에는 `overrides` 키가 아예 없다** — `grep -n '"overrides"' web/package-lock.json` 이 0건이다. 즉 지금 하한을 실제로 붙잡고 있는 것은 잠금 파일의 한 항목뿐이고, 누가 `npm install` 로 잠금을 다시 만들면 매니페스트에 적히지 않은 하한은 사라진다(`web/package-lock.json:3459` 는 postcss 가 요구하는 범위를 여전히 `"source-map-js": "^1.2.1"` 로 적고 있다). 고치면 하한이 잠금 재생성을 견디고, 같은 권고가 세 번째로 CI 를 세우지 못한다.

- 수용 기준:
  1) `web/package.json` 의 `overrides` 가 `js-yaml` 과 `source-map-js` 둘을 담고, 각 하한이 잠금이 실제로 받는 버전과 같다(`"js-yaml": "^4.3.2"`, `"source-map-js": "^1.2.2"`).
  2) `web/package-lock.json` 의 루트 `packages[""]` 가 같은 `overrides` 객체를 담는다(**npm 이 쓴 것** — `npm install --package-lock-only` 의 결과를 쓰고 손으로 짜 넣지 말 것). 그리고 `npm ci` 뒤 `npm ls source-map-js` 가 `1.2.2`, `npm ls js-yaml` 이 `4.3.2` 를 말하며, **다른 패키지의 resolved 버전은 하나도 움직이지 않았다**(`git diff --stat web/package-lock.json` 와 `git diff web/package-lock.json | grep '"version"'` 로 확인해 보고서에 적을 것).
  3) 실패했던 CI 단계 그대로가 통과한다: `cd web && npm ci && npm audit && npm run lint && npm test && npm run build` 가 종료 0 이고 테스트 234 개 전부 통과. 더해 **하한이 매니페스트로 옮겨졌음을 실행으로 증명**한다 — 잠금 파일을 치우고(`cp package-lock.json /tmp/lock.json && rm package-lock.json`) `npm install --package-lock-only` 로 매니페스트만으로 다시 만든 잠금이 `source-map-js` ≥ 1.2.2, `js-yaml` ≥ 4.3.2 를 담는지 보고, 그 뒤 **최소 diff 쪽 잠금으로 되돌린다**. (되돌리기 전후를 `git diff --stat` 로 비교해 커밋에 담기는 것이 최소 쪽임을 확인할 것.)

- 건드릴 파일 (프로덕션 2개 — 더 늘리지 말 것):
  - `web/package.json:41-43` `overrides` — `"source-map-js": "^1.2.2"` 한 줄을 더한다(키 순서는 알파벳순으로 `js-yaml` 다음). `dependencies`·`devDependencies`·`engines`·`scripts` 는 **한 글자도 손대지 말 것**(특히 `scripts.test` 의 `"${npm_node_execpath:-node}"` — v0.34.53 이 CI 가림을 고친 자리다).
  - `web/package-lock.json` — `npm install --package-lock-only` 로 재생성. 기대하는 diff 는 루트 `packages[""]` 에 `overrides` 블록이 생기는 것 + `source-map-js`/`js-yaml` 항목 변화 없음. **그 이상이면 멈추고 아래 '위험' 의 되돌림 경로를 쓸 것.**

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 실패했던 단계 그대로: `cd web && npm ci && npm audit && npm run lint && npm test && npm run build`
  - 해상도 확인: `cd web && npm ls source-map-js` / `cd web && npm ls js-yaml`
  - sdk 쪽 게이트(ci.yml:53, web 보다 **먼저** 돈다 — 여기서 빨간불이면 web 단계는 아예 실행되지 않는다): `cd sdk && npm ci && npm audit && npm run typecheck && npm test && npm run build`
  - Go 쪽은 이번 변경과 무관하나 회귀 확인용으로: `go vet ./cmd/... ./internal/...` (DB 가 없으면 통합 테스트는 조용히 skip 된다)
  - `git status` 로 **2파일만** 담겼는지 확인(`web/node_modules`·`web/dist` 는 커밋 전에 지울 것)

- 위험과 피할 것:
  - **`.github/workflows/*` 를 건드리지 말 것.** `npm audit` 에 `--audit-level`·`--omit=dev` 를 붙이는 것은 게이트를 느슨하게 하는 것이고 이번 회차의 금지 사항이다. `ci.yml:53·55` 의 명령 문자열은 그대로 둔다.
  - **`vite`(7.3.6)·`postcss`·`@vitejs/plugin-react` 를 올리지 말 것** — 의존성 메이저/부 업그레이드는 이 회차 범위 밖이고, `source-map-js` 는 `postcss` 의 전이 의존성이라 유혹이 있다.
  - **잠금 재생성이 번지는 것이 이 과제의 유일한 실질 위험이다.** `npm install --package-lock-only` 는 범위 안의 최신으로 전이 의존성을 끌어올릴 수 있다. `git diff web/package-lock.json` 에서 `source-map-js`·`js-yaml`·루트 `overrides` 외의 `"version"` 줄이 바뀌었으면 그건 사실상 의존성 업그레이드다 → **그 잠금을 버리고**, `package.json` 의 `overrides` 만 더한 뒤 잠금 루트의 `overrides` 객체는 npm 이 쓴 모양 그대로 최소로 반영하는 쪽을 택하라(그 뒤 `npm ci` 가 깨끗히 끝나는 것을 반드시 보여줄 것 — 매니페스트와 잠금이 어긋나면 `npm ci` 가 EUSAGE 로 죽는다). 둘 다 번지면 **`package.json` 변경만 담고 잠금은 손대지 않는 것도 받아들일 수 있다** — 그때는 수용 기준 2)를 못 채운 이유를 보고서에 명시할 것.
  - `npm ci` 는 잠금만 보고 설치하므로 **`overrides` 자체는 CI 에서 해상도를 바꾸지 않는다**. 이 과제의 값은 '잠금을 다시 만들 때 하한이 남는다' 는 것뿐이다 — 그 이상을 주장하지 말 것(기존 `js-yaml` 항목도 지금은 같은 의미에서 장식이다).
  - `sdk` 는 손대지 말 것 — `source-map-js` 가 없음을 확인했다(위 grep).
  - `internal/httpapi/admin.go`·`internal/auth`·`internal/database/migrations`·`docs/openapi.yaml` 은 이번 과제와 무관하다.
  - 과거 교훈: 순수 `.ts` 모듈 import 는 **`.ts` 확장자까지** 적어야 한다(이번 과제에 새 모듈은 없지만 테스트를 추가하려 든다면 해당). 기존 테스트가 출력 문자열을 글자 그대로 단언하는 곳이 많다.
  - 과거 교훈: `npx prettier --check` 를 게이트로 쓰지 말 것(저장소에 prettier 가 없고 main 의 여러 파일이 이미 실패한다).
  - 과거 교훈(2026-10-07 회차가 남김): `/admin` 화면에 브라우저 하네스를 쓰면 `useUnsavedWarning`(web/src/components/useUnsavedWarning.ts:7)의 `beforeunload` 가 같은 탭 재이동을 막는다 — 사례마다 새 탭을 쓸 것. 이번 과제는 브라우저가 필요 없다.

- 차선 후보: **보존 정책 다섯 칸에 입력 중 범위 helperText 검증을 붙인다** (가치 2 / 위험 1 / S) — 2026-10-06 CIDR 회차가 `cidrRule.ts` 로 한 것과 같은 구조로 순수 모듈 `web/src/pages/retentionRule.ts` 를 두고, `AdminPage.tsx` 의 `RetentionAdmin` 다섯 칸(라벨 「Raw Event (개월)」·「Aggregation (개월)」 등)의 `helperText` 와 `adminErrors.ts` 의 `describeRetentionError` 안내가 **같은 범위 상수**를 쓰게 한다. 주의: 「Realtime (시간)」 의 기존 helperText 는 '현재 적용되지 않습니다' 를 함께 말하므로 문장을 통째로 바꾸면 의도를 지운다 — 그 칸은 범위 문구를 **덧붙이기**만 할 것. 1순위가 성립하지 않는 경우(= 지금 main 의 `npm audit` 이 이미 녹색이고 잠금 재생성이 번져 2파일로 담을 수 없는 경우)에 이것을 고르라.

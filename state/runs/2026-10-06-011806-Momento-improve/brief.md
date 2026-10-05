- 과제: 「망 대역 추가」 폼이 CIDR 표기를 보내기 전에 화면에서 거른다 — 순수 모듈 `cidrRule.ts` + helperText, 그리고 Alert 과 폼이 같은 문장을 쓰게 한다 (가치 2 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/AdminPage.tsx:3235-3239` 의 「CIDR」 칸은 손으로 적는 자유 입력인데 「추가」 버튼은 `disabled={!form.name || !form.cidr}`(3250행)로 **빈 칸만** 막는다. 그래서 `10.20.30` 처럼 비트 수가 없거나 `10.20.30.0/33` 처럼 범위를 넘은 값은 왕복을 한 번 돌고 서버의 `INVALID_CIDR`(internal/httpapi/admin.go:812-814, `net.ParseCIDR`)을 받아야 알 수 있다. v0.34.56 이 그 답을 한국어로 만들어 두었으니(`adminErrors.ts:272-305` `describeNetworkError`) 다음 단계는 왕복 자체를 없애는 쪽이다. 같은 작업으로 안내 문장의 정본을 하나로 모은다 — 지금 그 문장은 `adminErrors.ts:286` 안에만 있고 폼에는 helperText 가 아예 없어서, 사용자는 **틀린 뒤에만** 올바른 표기법을 읽는다.

- 수용 기준:
  1) 실제 브라우저(빌드한 dist)에서 「CIDR」 칸에 `10.20.30`(비트 수 없음), `10.20.30.0/33`(IPv4 범위 밖), `10.20.30.0/abc`(숫자 아님), `1.2.3.4.5/24`(옥텟 5개)를 넣으면 「추가」 가 `disabled` 이고 칸 아래에 한국어 안내가 보이며, **서버에 도달한 POST /api/v1/networks 가 0건**이다(왕복이 사라졌다는 관찰 가능한 증거).
  2) `10.20.30.1/24`(호스트 비트가 켜진 값)는 **막지 않는다** — 「추가」 가 눌리고 POST 가 실제로 나간다. `net.ParseCIDR` 은 이것을 통과시키므로(이미 `adminErrors.ts:279-283` 이 그 사실을 적어 두었다) 화면이 막으면 서버보다 좁아진다. 대신 오류가 아닌 안내(색을 오류로 쓰지 않는 helperText)로 "대역의 시작 주소는 10.20.30.0/24 입니다" 처럼 계산한 네트워크 주소를 보여 준다. **Postgres `cidr` 컬럼이 이 값을 거절하는지는 이 환경에서 DB 로 재현하지 못했다(미확인) — 그러니 거절한다고 단정하는 문구를 쓰지 말 것.** `adminErrors.ts:288-299` 의 `NETWORK_CREATE_FAILED` 중립 문구도 그대로 둔다.
  3) 올바른 값 `10.20.30.0/24`, `0.0.0.0/0`, `2001:db8::/32` 에서는 오류 표시가 없고 버튼이 열린다. IPv6(콜론이 있는 값)은 비트 수(0~128)만 보고 주소 본문은 판정하지 않는다 — 판정 못 하는 것은 통과시킨다.
  4) 테스트가 증명할 것: (a) 차단 집합이 `net.ParseCIDR` 이 거절하는 집합의 **부분집합**이라는 것 — 위 2)·3) 의 값들이 전부 "차단 아님" 으로 단언된다, (b) 지금 코드와 똑같이 동작하는 **항등/상수 스텁**(무엇이든 통과시키는 판정)으로 새 단언이 실제로 실패한다, (c) `web/test/adminErrors.test.mjs:619` 의 `INVALID_CIDR` 문장 단언이 **글자 하나 고치지 않고** 통과한다(문장을 상수로 빼내도 문구가 바뀌지 않았다는 증거).

- 건드릴 파일 (프로덕션 3 + 테스트 1):
  - `web/src/pages/cidrRule.ts` (신규) — 순수 모듈. `export const CIDR_RULE`(현재 `adminErrors.ts:286` 의 문장에서 뒷부분을 **글자 그대로** 옮긴 것)과 판정 함수 하나, 예: `judgeCIDR(text: string): { blocking?: string; hint?: string }`. 이 저장소의 테스트 가능성 패턴(`passwordRule.ts`/`roleScope.ts`/`adminErrors.ts`)을 그대로 따른다.
  - `web/src/pages/adminErrors.ts:278-287` — `INVALID_CIDR` 분기가 `CIDR_RULE` 을 import 해서 쓰게 한다(`import { CIDR_RULE } from "./cidrRule.ts";` — 이 파일은 node:test 가 직접 읽으므로 **`.ts` 확장자 필수**, `passwordRule.ts` import 가 그 선례다). 출력 문장은 바이트 단위로 동일해야 한다.
  - `web/src/pages/AdminPage.tsx` — `NetworksAdmin`(3192행~)의 CIDR TextField(3235-3239)에 `error`/`helperText` 를, 「추가」 버튼(3246-3251)의 `disabled` 에 차단 판정을 더한다. import 는 이웃과 맞춰 **확장자 없이**(`from "./cidrRule"`, 76행 `./passwordRule` 선례). `helperText={PASSWORD_RULE}`(3563행)이 폼이 규칙을 상시 안내하는 선례다.
  - `web/test/cidrRule.test.mjs` (신규) — `import { ... } from "../src/pages/cidrRule.ts";`(확장자 필수).
  - 타이핑 중 소음은 구현자 판단: `/` 가 아직 없는 "미완성" 상태는 빨간 오류가 아닌 중립 안내로 두되 버튼은 닫아 둔다(버튼이 왜 닫혔는지 읽을 거리가 칸 아래에 항상 있어야 한다). 2026-10-05 회차가 JSON 칸에서 매 글자 파싱을 피한 것과 같은 이유다.

- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` — 이 worktree 에 `web/node_modules` 가 **없으므로 `npm ci` 가 선행**(수 분). `npm test` 는 기준선(직전 회차 기록으로 211건 — 이 환경에서 재실행하지 않았다, 미확인)에서 늘어나야 한다.
  - 프로덕션 배선 확인(이 저장소에서 다섯 회차가 쓴 방법): `npm run build` 의 `web/dist` 를 `/api/v1/me`·`/api/v1/networks` 를 흉내 낸 임시 서버에 올리고 headless Chrome(`google-chrome` 설치돼 있음, `puppeteer-core` 는 `/tmp` 에 따로 설치)으로 `/admin?section=networks` 를 열어 칸을 **실제로 채우고** 버튼 `disabled`·helperText DOM·서버에 도달한 POST 건수를 읽는다. 임시 하네스는 `/tmp` 에만 두고 `web/dist` 는 커밋 전에 지운 뒤 `git status` 로 4파일만 담긴 것을 확인한다.
  - Go 는 손대지 않으므로 `go test` 는 불필요하다(돌려도 `MOMENTO_TEST_POSTGRES_DSN` 이 없어 통합 테스트는 조용히 skip 된다).

- 위험과 피할 것:
  - **서버가 정본, 화면은 거울.** `internal/httpapi/admin.go`(`createNetwork` 포함)·`internal/auth`·마이그레이션·`.github/workflows` 는 열지 말 것. 화면 검증을 이유로 서버 검사를 건드리지 않는다.
  - **화면이 서버보다 좁아지면 안 된다.** 판정이 애매한 값(IPv6 본문, 호스트 비트, 선행 0 같은 표기)은 통과시킨다. 차단은 `net.ParseCIDR` 이 확실히 거절하는 것만.
  - 기존 테스트가 **출력 문자열을 글자 그대로 단언**한다(`adminErrors.test.mjs:619`). 문장을 상수로 옮길 때 문구를 "개선" 하지 말 것 — 바꾸면 그 테스트도 고쳐야 하고, 그러면 기준 4(c) 의 증거가 사라진다.
  - `2376-2383` 행의 설정 화면 「신뢰할 Reverse Proxy CIDR」 은 **다른 엔드포인트(settings PUT)·다른 서버 검사(admin.go:720-731)** 다. 같은 문자열 모양이라고 묶어서 고치지 말 것 — 이번 범위는 `NetworksAdmin` 폼 하나다(쪼갠 다섯 회차가 모두 통과했다).
  - `npx prettier --check` 를 게이트로 쓰지 말 것(의존성·CI 단계가 없고 main 의 20여 파일이 이미 실패한다). `gofmt -l` 의 `internal/segment/segment_test.go` 는 선재 잡음이다.
  - `npm test` 가 깨지면 Node 가림을 먼저 보라 — `web/package.json` 의 test 는 `npm_node_execpath` 를 쓴다(v0.34.53). 그 줄은 건드리지 말 것.

- 차선 후보: 보존 정책 저장 실패(`AdminPage.tsx:2956` 의 `{save.error.message}` Alert — 이번 정찰에서 행 번호 확인)를 서버 영문·pgx 원문 대신 한국어로 안내한다 — v0.34.54~57 이 네 차례 통과시킨 패턴 그대로(`adminErrors.ts` 에 `describe*` 를 **새로** 두고 `refusal`·문장 상수만 재사용, 핸들러 하나만). 해당 서버 핸들러의 코드 집합을 먼저 열어 확인할 것(이번 정찰은 열지 않았다 — 미확인).

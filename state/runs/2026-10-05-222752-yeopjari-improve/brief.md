- 과제: IndexNow 알림이 publicUrl 설정 오류에 트랜잭션 뒤에서 터지지 않게 하고, 키·요청 본문 계약을 테스트로 묶는다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `pingIndexNow` 는 주석에 "never throws" 라고 적혀 있고 `jobs/disclosures.ts:296` 에서 **트랜잭션 커밋 뒤에** 호출되는데, 첫 줄 `new URL(env.publicUrl)` 이 try 블록 밖에 있어 `YEOPJARI_PUBLIC_URL` 이 스킴 없는 값(예: `yeopjari.bid`)이면 TypeError 가 호출자까지 올라간다(= 아침 공시 작업이 글을 다 올린 뒤 실패로 기록되고 `/api/v1/internal/indexnow` 는 500). 또 이 파일은 2회차 연속으로 손이 갔는데도 단위 테스트가 하나도 없어서, 키 형식(`/^[0-9a-f]{32}\.txt$/`, `public-pages.ts:775` 의 키 파일 게이트)이나 요청 본문 모양이 바뀌어도 아무것도 깨지지 않는다.
- 수용 기준:
  1) `pingIndexNowStatus` 가 어떤 `env.publicUrl` 값(빈 문자열, `yeopjari.bid`, `::`)에도 예외를 던지지 않고 문자열을 돌려준다(`skipped` 또는 `error …`). `pingIndexNow` 도 같은 입력에서 `false` 를 돌려준다.
  2) 정상 설정(`https://…`)에서는 지금 동작이 그대로다: dev/offline/`http:`/빈 경로 목록은 `skipped`, 그 외에는 `https://api.indexnow.org/indexnow` 로 POST 1회, `response.ok` 면 `'sent'`, 아니면 `'http <status>'`, fetch 가 throw 하면 `'error …'`(80자 절단).
  3) 테스트가 증명할 것: (a) 잘못된 publicUrl 에서 throw 하지 않음 — **수정 전 red** 가 되어야 한다(지금은 TypeError). (b) `indexNowKey` 가 32자 소문자 16진수라서 `public-pages.ts` 의 `/^[0-9a-f]{32}\.txt$/` 게이트를 통과하고, 같은 env 에서 두 번 부르면 같은 값이다(파생 키이므로). (c) 보낸 본문의 `host`/`key`/`keyLocation` 이 `env.publicUrl` 의 origin 과 `indexNowKey` 와 일치하고, `urlList` 가 `origin + path` 로 만들어지며 중복 경로가 한 번만, 1000개까지만 들어간다. (d) 요청에 `user-agent` 헤더가 있다(c730b67 이 고친 회귀 — 워커 fetch 가 UA 없이 거부당했다).
- 건드릴 파일:
  - `packages/core/src/seo/indexnow.ts:30 pingIndexNowStatus` — `new URL(env.publicUrl)` 을 try 안으로 옮기거나 그 전에 파싱을 감싸서 실패 시 `skipped`(설정 오류는 알림이 아니라 운영 설정 문제이므로 `error …` 도 허용 — 어느 쪽이든 수용 기준 1을 만족하면 된다). dev/offline/`https:` 판정 순서와 반환 문자열 값은 바꾸지 말 것(`ops.ts:477` 이 그 문자열을 운영자에게 그대로 돌려준다).
  - `packages/core/src/__tests__/indexnow.test.ts` (신규) — 위 (a)~(d). 가짜 env 는 기존 테스트에 선례가 없으니 최소 객체를 캐스팅해 만들면 된다: `{ dev: false, offline: false, publicUrl: 'https://yeopjari.bid', keys: { tokenSign: new Uint8Array(32) } } as unknown as Env` (`env.ts:14~45` 의 `Env`·`keys.tokenSign: Uint8Array`). `hmacHex` 는 `crypto.subtle` 을 쓰므로 Node 22 에서 그대로 돈다(`util/crypto.ts:91`). fetch 는 `globalThis.fetch` 를 테스트 안에서 바꿔 끼우고 `afterEach` 로 되돌릴 것(`vi.spyOn(globalThis, 'fetch')` 가능).
  - 프로덕션 파일은 1개(`indexnow.ts`)만. `ops.ts`·`jobs/disclosures.ts`·`public-pages.ts` 는 읽기만 하고 수정하지 말 것.
- 검증 명령:
  - 이 워크트리에 `node_modules` 가 없다(`node_modules/.bin/vitest` 없음). 먼저 `npm ci` (지난 회차에 Node v22.23.1에서 성공).
  - `npx vitest run packages/core/src/__tests__/indexnow.test.ts` — 먼저 (a) 가 실패(red)하는 것을 확인하고 고친 뒤 green.
  - `npm test` (vitest run 전체, 현재 24파일) · `npm run lint` · `npm run typecheck`. 여유가 있으면 `npm run check`.
- 위험과 피할 것:
  - 테스트에서 실제 네트워크를 때리지 말 것. `https://api.indexnow.org` 로 진짜 POST 가 나가면 CI 없는 이 저장소에서 아무도 못 잡는다 — fetch 스텁 없이 '정상 설정' 경로를 실행하는 테스트를 쓰지 말 것.
  - `indexNowKey` 의 파생 방식(`hmacHex(tokenSign, 'indexnow-key').slice(0,32)`)을 바꾸지 말 것. 바꾸면 이미 검색엔진에 알린 키 파일 경로가 죽는다. 테스트는 **형식과 결정성**만 고정하고 특정 해시값을 하드코딩하지 말 것(키가 env 에서 오므로 값 고정은 의미 없음).
  - `env.ts` 의 `publicUrl` 검증을 추가하는 쪽으로 번지지 말 것(`public-pages.ts`·`rankings.ts`·`tools.ts` 가 모두 `new URL(ctx.env.publicUrl)` 를 쓰므로 범위가 커진다). 이번에는 커밋 뒤 호출되는 indexnow 만 계약을 지키게 한다.
  - 보호 경로(auth/session, db/migrations, .github/workflows, deploy)는 건드리지 않는다. `routes/admin/ops.ts` 는 `auth: 'internal'` 경로라 수정 금지.
  - 과거 교훈: 소스 grep 이나 타입만으로 "증명했다" 고 쓰지 말 것. 실제로 스텁한 fetch 가 받은 요청 본문·헤더를 읽어서 assert 할 것.
- 차선 후보: Node StaticFiles 의 실제 파일 기반 HEAD·SPA·경로 격리 테스트 (`apps/server/src/static.ts` 의 `StaticFiles.serve/shell/resolve`, 임시 디렉터리와 실제 `Request` 로 1파일) — 지난 회차의 차선이 그대로 남아 있고, server 워크스페이스에 추적된 테스트가 없다는 공백을 메운다. shell fallback 계약과 symlink 정책은 바꾸지 말 것.

# 과제서 — 2026-10-07-175813-irumx-www-improve

- 과제: 빌드 스크립트가 쓰는 미선언 전이 의존성(harfbuzzjs·fontverter) 명시 + 미선언 import 검사 추가 (가치 4 / 위험 1 / 작업량 S)

- 왜: `scripts/verify-build.mjs:18-19` 가 `harfbuzzjs`, `fontverter` 를 직접 import 하는데 `package.json` 의 `devDependencies` 에는 둘 다 없다(선언: @astrojs/check, @astrojs/sitemap, @playwright/test, astro, playwright, subset-font, typescript, wrangler). 지금 동작하는 것은 `subset-font@2.9.0` 의 전이 의존성(`fontverter@^2.0.0`, `harfbuzzjs@^1.6.1` — package-lock.json:3484, 3574, 5515 에서 확인)이 npm 호이스팅으로 `node_modules` 최상단에 올라오기 때문일 뿐이다. subset-font 가 의존성을 바꾸거나 호이스팅이 달라지면 `npm ci && npm run build` 가 `ERR_MODULE_NOT_FOUND` 로 바로 깨지고 배포가 막힌다. 선언을 바로잡고, 같은 실수가 다시 들어오지 못하게 미선언 bare import 를 빌드에서 걸러낸다.

- 수용 기준:
  1) `package.json` 의 `devDependencies` 에 `harfbuzzjs`(설치된 1.6.2 기준 `^1.6.2`), `fontverter`(설치된 2.0.0 기준 `^2.0.0`)가 들어가고 `package-lock.json` 이 같이 갱신된다. 버전은 지금 lock 에 있는 것과 같은 것을 고른다(새 버전으로 올리지 말 것).
  2) 새 스크립트 `scripts/verify-deps.mjs` 가 `scripts/*.mjs`, `worker/*.ts`, `src/**/*.ts` 의 bare import(상대경로·`node:`·`astro:` 접두사 제외)를 모아, `package.json` 의 `dependencies`/`devDependencies` 에 없는 것이 하나라도 있으면 그 이름을 출력하고 종료 코드 1 로 끝낸다. `npm run build` 의 첫 단계로 끼운다.
  3) 증명: 1) 적용 후 `node scripts/verify-deps.mjs` 가 통과하고, `package.json` 에서 `harfbuzzjs` 줄을 임시로 지우면 같은 명령이 `harfbuzzjs` 를 지목하며 종료 코드 1 로 실패한다(확인 후 원복). 그리고 `npm run build` 가 끝까지 통과한다.

- 건드릴 파일 (프로덕션 파일 2개 + 신규 1개):
  - `package.json` — `devDependencies` 에 `harfbuzzjs`·`fontverter` 추가, `scripts.build` 앞에 `node scripts/verify-deps.mjs &&` 추가
  - `package-lock.json` — `npm install --package-lock-only` 로 갱신(`dev: true` 유지 확인)
  - `scripts/verify-deps.mjs` (신규) — 위 2) 의 검사. 기존 `scripts/verify-build.mjs` 의 주석·오류 출력 스타일(한국어 주석 머리말, `✗`/`✓` 출력, `process.exit(1)`)을 그대로 따른다.

- 검증 명령 (이 저장소에 실제로 있는 것):
  ```bash
  npm install --package-lock-only     # lock 갱신
  node scripts/verify-deps.mjs        # 새 검사 — 통과해야 함
  npm run build                       # subset-font → astro check → astro build → verify-build (실제 빌드, 수십 초)
  ```
  선택(시간 여유가 있을 때만, 몇 분 걸린다): `npm run test:build && npm test`

- 위험과 피할 것:
  - **버전을 올리지 말 것.** 지금 lock 에 있는 `harfbuzzjs@1.6.2`, `fontverter@2.0.0` 을 그대로 캐럿 범위로 적는다. `npm install harfbuzzjs fontverter` 를 그냥 돌리면 최신 버전을 끌어와 `verify-build.mjs` 의 `new HbFace(new HbBlob(buf), 0).collectUnicodes()` / `convert(buf, 'truetype')` 호출이 깨질 수 있다. `--package-lock-only` 로 lock 만 갱신하고 `node_modules` 는 그대로 두는 편이 안전하다.
  - `scripts/verify-deps.mjs` 가 **거짓 실패를 내지 않게** 할 것: `src/**/*.astro` 는 이번 범위에서 빼고(.ts 만), `astro:`·`node:` 접두사와 상대경로(`./`, `../`), 그리고 `@scope/name/subpath` 형태를 패키지 이름(`@scope/name`)으로 정규화하는 처리를 넣어야 한다. 스코프 없는 `pkg/sub` 도 `pkg` 로 자른다. type-only import(`import type { X } from 'y'`)도 같은 규칙으로 본다.
  - `scripts/capture-*.mjs`, `scripts/screenshots.mjs`, `scripts/make-images.mjs` 는 `playwright` 를 쓰는데 이것은 이미 선언되어 있다 — 추가 선언 필요 없음.
  - **`worker/`·`src/lib/inquiry.ts` 의 동작은 건드리지 말 것.** 문의 접수·메일 전달(Turnstile·Resend·Svix 서명)은 운영 중인 보호 경로다. 이 과제는 거기 손댈 이유가 없다.
  - `verify-build.mjs` 의 기존 검사 로직(사이트맵 페이지 목록, BANNED 문자열, 글꼴 커버리지)은 이번에 손대지 말 것 — 차선 후보의 영역이다.
  - `npm run build` 는 `PUBLIC_TURNSTILE_SITE_KEY` 없이도 도는지 확인되지 않았다(미확인). 실패하면 `package.json` 의 `test:build` 가 쓰는 테스트 키를 그대로 써서 `PUBLIC_TURNSTILE_SITE_KEY=1x00000000000000000000BB npm run build` 로 돌릴 것.

- 차선 후보: **서비스 페이지 목록 드리프트 감지** — `scripts/verify-build.mjs` 하단의 사이트맵 확인 목록(17개 경로 하드코딩)과 `tests/site.spec.ts:3` 의 `PAGES` 배열이 같은 목록을 두 군데에 손으로 적어 두고 있다. 최근 커밋(cd1a0a8, e2af87e)에서 서비스 16개가 늘면서 둘 다 수동으로 갱신됐다. **두 목록을 하나로 합치지 말고**(각자 다른 계약을 지킨다 — 하나는 dist 파일, 하나는 HTTP 응답), `verify-build.mjs` 에 "dist 의 모든 `services/*.html` 이 사이트맵 목록과 `tests/site.spec.ts` 의 `PAGES` 에 들어 있는지" 확인하는 **감지만** 추가한다. 파일 1개(`scripts/verify-build.mjs`), 검증은 `npm run build`.

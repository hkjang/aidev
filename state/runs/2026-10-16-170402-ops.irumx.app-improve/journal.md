# 회차 노트 2026-10-16-170402-ops.irumx.app-improve — ops.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:04] base pinned — main@e85afae
- [러너 17:04] autonomy release — 

## 정찰 노트
- 자동 배정된 수정 과제라 새로 고르지 않았다. 핵심 관찰: 직전 두 회차 구현자가 모두 `test:unit`+`check` 만 돌렸고 `npm test` 는 아무도 돌린 적이 없다 — 그래서 과제서를 "원인 확정부터" 로 짰다.
- **확신 없음(가장 중요)**: 샌드박스가 `npm install` 을 막아 `npm test` 를 재현하지 못했다. 1순위 가설(생 `wrangler dev` 가 `wrangler.jsonc:9` 의 `assets` 에 `directory` 가 없어 못 뜬다 / `dist` 없음)과 2순위(api-c 급증 시험의 `age-resources` 전제)는 **읽기만으로 세운 것**이다. `--assets` 플래그가 맞는지도 미확인.
- 읽기로 **배제한** 것: migration 0004 는 열 이름이 스키마와 전부 맞고 db-local 이 매번 DB 를 지우고 적용한다. `first_seen` 은 삽입 때만 쓰여 age-resources 가 덮이지 않는다. 즉 2순위는 논리적으로 통과해야 한다 → 1순위를 먼저 보라고 적었다.
- 구현자가 조심할 것: 포트 8940~8943 에 낡은 서버가 남아 있으면 `reuseExistingServer: true` 가 엉뚱한 결과를 준다. `wrangler.jsonc` 는 `verify-build.mjs` 가 검사하므로 자산 경로는 시험 쪽 명령에서 줄 것.
- 다른 후보(ui 스모크+axe, `do:*` 라벨)를 제친 이유: 그물 자체가 깨진 상태에서 시험을 더 얹거나 기능을 고치면 실패 원인이 섞인다.
- [러너 17:09] scout done — `npm test`(전체 = unit + api) 가 두 회차 연속 exit 1 — 먼저 실패 지점을 재현해 확정하고, api 묶음이 기동/통과

## 구현 노트
- 무엇을 왜: `package.json` 에 `pretest: npm run build` 한 줄 + `playwright.config.ts` 머리 주석에 원인 기록. 실패는 시험 코드가 아니라 **빌드 전제가 명령에 없던 것**이다 — `wrangler.jsonc` 의 `assets` 에 `directory` 가 없고(의도된 것: 배포는 vite 플러그인이 채운다) 그 값을 채운 `dist/irumx_ops/wrangler.json` 은 `vite build` 산출물인데 webServer 는 생 `npx wrangler dev` 를 띄운다. 프로덕션 코드 변경 0.
- 확신 있는 곳(실측): `dist`·`.wrangler` 지운 트리에서 고치기 전 `npm test --silent` → exit 1(assets/directory 오류), 고친 뒤 → `90 passed (15.6s)` exit 0. `npm run check` exit 0. `npm run test:unit` 49 passed(빌드 출력 없음 = `pretest` 가 `test:unit` 에 안 붙는다 확인). 정찰이 말한 기준선 49 가 맞다(51 아님).
- 확신 없는 곳: wrangler 가 `dist/irumx_ops/wrangler.json` 을 **어떤 경로로** 자동 발견하는지는 끝까지 파지 않았다(`.wrangler` 를 지워도 동작했으므로 redirect 파일은 아니다). 발견 기제가 wrangler 메이저 업그레이드에서 바뀌면 이 방식이 다시 깨질 수 있다 — 그때 증상은 또 "webServer 기동 실패" 다. `dist/client` 만 있으면 안 되고 생성된 설정이 필요하다는 것은 직접 지워 보고 확인했다.
- 일부러 하지 않은 것: 과제서가 제안한 `--assets dist/client` (불필요 — 필요한 건 생성된 설정이고 어차피 빌드가 선행된다), `wrangler.jsonc` 수정(`scripts/verify-build.mjs` 가 검사하는 배포 설정), 차선 후보(`do:*` 라벨)·ui 시험 추가(범위를 1순위에 묶었다).
- 다음 역할이 조심할 것: `npm test` 는 이제 **스스로 빌드한다** — 느려졌다(빌드 ~10초 + 시험 ~16초). 포트 8940~8943 이 비어 있어야 한다(`reuseExistingServer: true` 라 낡은 서버에 붙으면 엉뚱하게 통과/실패한다). `npm install` 은 `package-lock.json` 에 플랫폼 잡음(optional dep 의 `dev: true`) 75줄을 만든다 — 이번엔 되돌려 커밋에서 뺐으니 다시 보이면 그 잡음이다.
- [러너 17:15] brief accepted — 채택 — 1순위 가설(A, webServer 기동 실패)이 증상까지 정확히 맞았고 지정한 `pretest` 가 바로 해법이었다. 다만 과제서가 �
- [러너 17:16] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- 판정 approve(risk low). 읽기만 하지 않고 직접 돌렸다 — dist·.wrangler 지운 트리에서 고치기 전 `npx playwright test` → webServer 기동 실패 exit 1(주석에 적힌 `assets`/`directory` 오류 문구 그대로), 고친 뒤 `npm test` → 90 passed exit 0, `npm run test:unit` → 49 passed·dist 생성 없음(`pretest` 가 안 붙는다). 구현 노트의 실측과 전부 일치.
- 구현자가 못 판 "자동 발견 기제" 를 확정했다: `vite build` 가 쓰는 `.wrangler/deploy/config.json` = `{"configPath":"../../dist/irumx_ops/wrangler.json"}` 가 redirect 이고 그 생성 설정에 `assets.directory: "../client"` 가 있다. `.wrangler` 를 지워도 된 건 빌드가 다시 만들기 때문 — redirect 가 아니라는 노트의 추측은 틀렸다.
- 남는 우려(릴리즈 노트): ① 빌드 뒤에 `.wrangler` 만 지우면 같은 증상이 돌아온다 ② `npm run shots`·맨손 `npx playwright test` 엔 빌드 훅이 없다(오늘은 `shots.spec.ts`·`ui-*.spec.ts` 가 없어 무해 — 브라우저 시험을 넣는 회차가 `preshots` 를 같이 넣을 것) ③ `npm test` 가 매번 폰트 서브셋+tsc 4회+vite build 를 선행해 느려졌다.
- 못 본 것: 브라우저·접근성 경로(파일 자체가 없다), 프로덕션 로직(이 diff 가 건드리지 않는다). 보안·법무 차단 사유 없음 — 인증·결제·삭제·마이그레이션·새 의존성·비밀값·개인정보 경로 전부 무관.
- [러너 17:19] review approved — 리뷰 승인 (risk=low)
- [러너 17:19] pr created — https://github.com/hkjang/ops.irumx.app/pull/1
- [러너 17:24] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)

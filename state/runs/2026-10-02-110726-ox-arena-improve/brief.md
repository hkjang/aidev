# 과제서 — 2026-10-02-110726-ox-arena-improve

- 과제: Vitest 도입 + 순수 게임 로직(rankPlayers / derivePhase / 집계) 회귀 테스트 (가치 4 / 위험 2 / 작업량 M)

- 왜: 이 저장소에는 테스트가 **하나도 없다**(`package.json` 에 `test` 스크립트 없음, `src` 아래 `*.test.*` 파일 0개, CI `.github/workflows/pages.yml` 은 main push 시 `npm run build` 후 Pages 배포만 한다). 점수·공동순위·게임 단계 판정 같은 핵심 순수 로직이 전부 수동 확인에만 의존하고 있어, 다음 회차 구현자가 "고쳤다"를 증명할 명령이 `tsc --noEmit` 밖에 없다. 러너 한 대 분량의 작은 테스트 하네스를 깔아 두면 이번 회차뿐 아니라 이후 모든 회차가 실제 동작을 근거로 검증할 수 있게 된다.

- 수용 기준:
  1) `npm test` 가 저장소 루트에서 돌고 전부 통과한다(새 스크립트). `npm run build`(= `tsc --noEmit && vite build`)도 그대로 통과한다 — 테스트 파일이 `tsconfig.json` 의 `include: ["src"]` 안에 들어가므로 타입체크도 같이 통과해야 한다.
  2) `rankPlayers`(`src/lib/types.ts:45`) 에 대해 **공동 순위 건너뛰기**가 고정된다: 점수 100/100/50 인 세 명이 `1, 1, 3` 랭크를 받는다(2가 아니라 3). 동점 그룹 내부 정렬이 `correct` 내림차순 → `nickname` 오름차순임도 함께 고정한다.
  3) `derivePhase`(`src/lib/useRoom.ts:31`) 가 `room === null` → `'waiting'`, `status='playing'` 이고 `now < started_at` → `'countdown'`, `started_at <= now < ends_at` → `'choosing'`, `now >= ends_at` → `'locked'`, `status='ready'|'revealed'|'finished'` → 그대로 통과, 를 각각 증명한다. 경계값(`now === started_at`, `now === ends_at`)도 한 케이스씩 넣는다.
  4) `useRoom` 의 집계(`src/lib/useRoom.ts:120-125` 의 `counts` useMemo)를 **순수 함수로 추출**해 테스트한다: 탈락자(`eliminated: true`)는 `alive` 와 `none` 분모에서 빠지고, `none` 은 음수가 되지 않는다(`Math.max(0, …)`).
  5) 기존 런타임 동작은 바뀌지 않는다 — 추출은 순수 이동이고, `useRoom` 의 `useMemo` 는 새 함수를 호출하도록만 바꾼다(손으로 만든 대역이 아니라 프로덕션에서 실제로 쓰이는 그 함수를 테스트할 것).

- 건드릴 파일 (프로덕션 2개 + 테스트 2~3개):
  - `package.json` — `devDependencies` 에 `vitest` 추가, `scripts` 에 `"test": "vitest run"` 추가. (`package-lock.json` 은 `npm i` 가 자동 갱신 — 커밋에 포함)
  - `src/lib/useRoom.ts` — `counts` useMemo 본문을 `export function computeCounts(choices: Record<string, Choice>, players: Player[])` 로 추출하고 useMemo 가 그것을 호출하게 한다. **다른 변경 금지**(폴링·채널·syncClock 손대지 말 것).
  - `src/lib/useRoom.test.ts` (신규) — `derivePhase`, `computeCounts` 테스트.
  - `src/lib/types.test.ts` (신규) — `rankPlayers` 테스트.
  - (선택) `src/lib/util.test.ts` (신규) — `errMsg`(`src/lib/util.ts:13`) 의 매핑/패스스루. `joinUrl` 은 `location` 에 의존하므로 **테스트하지 말 것**(jsdom 환경을 끌어오면 과제가 커진다).
  - 설정 파일은 만들지 않는 것을 권한다: Vitest 기본 `include` 가 `**/*.{test,spec}.?(c|m)[jt]s?(x)` 이고 기본 environment 가 `node` 라, 위 순수 함수 테스트는 무설정으로 돈다. `vite.config.ts` 에 `test:` 키를 넣으면 타입 경고가 생길 수 있으니 건드리지 말 것.

- 검증 명령:
  ```
  npm i                 # vitest 설치 (이 정찰 환경에서는 npm 이 차단되어 실행 못 해 봄 — 미확인)
  npm test              # 신규. 전부 통과해야 함
  npm run build         # 기존. tsc --noEmit && vite build — 테스트 파일 타입 에러까지 잡힘
  ```
  - **미확인**: 위 두 명령 모두 이 정찰 세션에서 실제로 실행해 보지 못했다(샌드박스에서 `npm ci` 가 거부됨). `package.json` 에 `build` 스크립트가 정의되어 있고 CI 가 같은 명령을 쓰는 것만 확인했다. 구현자는 **먼저 `npm i && npm run build` 를 돌려 기준선이 녹색인지 확인한 뒤** 작업을 시작할 것.
  - vitest 버전은 vite `^5.4.0` 과 맞춰야 한다. `vitest@^2` 계열이 vite 5 대응이다(**미확인** — 설치 후 `npm test` 와 `npm run build` 가 모두 도는 것으로 확인할 것. 안 되면 `npm i -D vitest@2` 로 고정).

- 위험과 피할 것:
  - **`supabase/migrations/**` 를 건드리지 말 것.** 보호 경로이고, 이 환경에는 Supabase 인스턴스도 `supabase/config.toml` 도 없어 SQL 변경을 검증할 방법이 전혀 없다. 아래 차선 후보가 SQL 변경을 요구하더라도 이번 회차에서는 미룬다.
  - **`.github/workflows/pages.yml` 을 건드리지 말 것.** 릴리즈(Pages 배포) 경로다. CI 에 test 단계를 더하는 것은 별도 회차 과제로 남긴다(ideas.json 참조).
  - `rankPlayers` 의 현재 동작(정렬은 `score → correct → nickname`, 랭크는 **점수만** 보고 부여)은 README 의 "동점 공동순위" 와 일치한다. **테스트는 현재 동작을 고정하는 것이지 바꾸는 것이 아니다.** `correct` 를 랭크 기준에 넣지 말 것.
  - `computeCounts` 추출 시 `Choice`/`Player` 타입을 `src/lib/types.ts` 에서 import 할 것. 테스트에서 `Player` 를 만들 때 **필수 필드를 전부 채운 실제 `Player` 타입 객체**를 쓸 것 — `as any` 나 부분 객체 캐스팅으로 때우면 타입 계약이 깨져도 테스트가 통과한다.
  - 테스트 파일이 `src` 안에 있으므로 `vite build` 결과물에 섞이지 않는지 확인할 것(엔트리에서 import 되지 않으니 섞이지 않아야 정상).

- 차선 후보: **탈락 모드에서 정답률·미선택 분모가 틀린 문제** — `supabase/migrations/0003_elimination.sql:114-115` 의 `v_total` 은 방의 **모든** 플레이어를 세는데(탈락자 포함), `src/pages/Display.tsx:50` 이 그 `rv.total` 로 정답률을 계산한다. 반면 같은 화면의 `Field` 는 `useRoom` 의 클라이언트 `counts`(생존자 기준, `src/lib/useRoom.ts:123`)를 쓴다. 같은 개념을 두 경로가 다른 분모로 읽고 있어, 100명 중 50명이 탈락한 상태에서 생존자 전원 정답이면 정답률이 100%가 아니라 50%로 표시된다. 고치려면 `admin_reveal` 이 생존자 기준 `alive`/`none` 을 함께 내려주고 Display 가 그것을 쓰도록 **두 경로를 end-to-end 로 맞춰야** 한다 — 마이그레이션(보호 경로) + 검증 수단 부재 때문에 이번 회차에서는 뺐다. 1순위가 성립하지 않으면 이것을 하되, 서버/클라이언트 양쪽 분모가 같은 입력에서 같은 값을 내는지 반드시 함께 확인할 것.

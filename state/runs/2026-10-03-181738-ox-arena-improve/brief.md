- 과제: PR 검증용 CI 워크플로 추가 (`.github/workflows/ci.yml` 신규 1파일) (가치 4 / 위험 1 / 작업량 S)

- 왜: 이 저장소의 워크플로는 `.github/workflows/pages.yml` 하나뿐이고 트리거가 `push: {branches:[main]}` + `workflow_dispatch` 라서(직접 열어 확인함) **PR 에는 자동 검증이 전혀 없다** — 타입 오류가 있는 PR 이 머지되면 그 시점에 비로소 유일한 릴리즈 경로(Pages 배포)가 빨개진다. PR 단계에서 `tsc --noEmit && vite build` 를 돌리면 릴리즈 경로를 한 글자도 건드리지 않고 회귀를 머지 전에 잡는다.

- 수용 기준:
  1) `.github/workflows/ci.yml` 이 새로 생기고, `pull_request` 와 `push: {branches:[main]}` 에서 `npm ci` → `npm run test --if-present` → `npm run build` 를 순서대로 돈다.
  2) `.github/workflows/pages.yml` 의 diff 가 0줄이다 (`git diff --stat` 으로 보일 것). 유일한 릴리즈 경로이므로 손대지 않는다.
  3) 로컬에서 `npm ci && npm run build` 를 실제로 실행해 녹색 출력을 증거로 남긴다 (CI 가 돌릴 명령과 동일한 명령).
  4) **이 PR 자체에서 ci.yml 이 녹색으로 도는 것을 확인한다** — PR 이 열리면 `gh pr checks` 또는 `gh run list --workflow=ci.yml` 로 결론(success)을 확인하고 그 출력을 남긴다. 워크플로는 로컬에서 실행할 수 없으므로 이것이 유일한 end-to-end 증거다. 빨개지면 고쳐서 녹색을 만든 뒤 끝낸다.
  5) 현재 `package.json` 에는 `test` 스크립트가 **없다**(확인함 — scripts 는 dev/build/preview 뿐). 그래도 CI 가 깨지지 않아야 한다 → `npm run test --if-present` 로 쓸 것(`npm test` 단독은 스크립트가 없으면 실패한다). 심사 중인 다른 PR 이 vitest + `test` 스크립트를 추가하면 같은 줄이 자동으로 테스트를 돌리게 된다.

- 건드릴 파일:
  - `.github/workflows/ci.yml` — 신규. 유일하게 추가·수정하는 파일.
  - (선택) `README.md` — CI 배지 한 줄. 꼭 필요하지 않으면 생략해 1파일로 끝낼 것.

  권장 내용(그대로 써도 되고, 동등하면 바꿔도 된다):
  ```yaml
  name: CI
  on:
    pull_request:
    push: { branches: [main] }
  permissions: { contents: read }
  concurrency: { group: ci-${{ github.ref }}, cancel-in-progress: true }
  jobs:
    verify:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-node@v4
          with: { node-version: 22, cache: npm }
        - run: npm ci
        - run: npm run test --if-present
        - run: npm run build
  ```
  - `node-version: 22` 는 `pages.yml` 과 맞춘 값이다(pages.yml 을 열어 확인). 낮추지 말 것 — `@supabase/supabase-js` 가 Node 20 에서 engine 경고를 낸다.
  - `cache: npm` 은 `package-lock.json` 이 있어야 동작한다 — 리포 루트에 있고 git 에 추적되고 있음(`git ls-files` 로 확인).
  - **환경변수는 넣지 않아도 된다.** `src/lib/supabase.ts:4-5` 가 `import.meta.env.VITE_SUPABASE_URL || '<하드코딩 fallback>'` 형태라 env 없이도 빌드가 통과한다. pages.yml 처럼 `vars.VITE_*` 를 넣으면 fork PR 에서 비게 되므로 오히려 혼란스럽다.

- 검증 명령:
  - `npm ci && npm run build`  ← `build` 는 `tsc --noEmit && vite build` 다(package.json 확인).
  - `git diff --stat origin/main -- .github/workflows/pages.yml`  → 출력이 비어야 한다.
  - PR 생성 후: `gh pr checks` (또는 `gh run list --workflow=ci.yml --limit 3`) → CI 가 success.
  - **미확인 / 주의**: 정찰 샌드박스에서는 `npm ci` 가 권한 거부로 막혀 실행하지 못했다(2026-10-02 정찰도 동일). `node_modules` 도 없다. 따라서 **기준선이 녹색인지 확인되지 않았다** — 구현자는 아무것도 바꾸기 전에 `npm ci && npm run build` 를 먼저 돌려 기준선을 확인하고 시작할 것. 기준선이 이미 빨갛다면 그것이 이번 회차의 과제다(그 경우 ci.yml 추가는 보류하고 빌드부터 고칠 것).

- 위험과 피할 것:
  - `.github/workflows/pages.yml` **절대 수정 금지**. 유일한 릴리즈 경로이고, 운영자가 "릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과하는 것을 확인할 것" 을 반복해 지적했다. 별도 파일로만 추가한다.
  - `permissions` 를 넓히지 말 것(`contents: read` 로 충분). `pages: write` / `id-token: write` 는 배포 워크플로에만 필요하다.
  - `supabase/migrations/**` 와 인증 경로(`admin_*`, `player_tokens`, RLS)는 이번 과제와 무관 — 열지도 말 것.
  - 스코프를 넓히지 말 것: lint 도구 도입(eslint 신규 설정), 의존성 업그레이드, 테스트 프레임워크 추가를 여기에 끼워 넣지 말 것. 특히 **vitest 를 추가하지 말 것** — 2026-10-02 회차가 vitest + `src/lib/phase.ts` 추출을 이미 만들어 심사 중(review-pending)이고, 지금 base(`main@f68a6fa`)에는 머지되어 있지 않다. 다시 추가하면 `package.json`/`package-lock.json` 이 충돌한다. `--if-present` 가 그 PR 과 이 PR 을 둘 다 성립시키는 이유다.
  - 워크플로 파일은 보호 경로로 취급되는 환경이 있다. 승인 프롬프트가 뜨면 거부하지 말고 사람 판단을 기다릴 것.

- 차선 후보: **countdown 단계에서 O/X 버튼이 활성인데 눌러도 아무 일이 없는 문제 (가치 3 / 위험 1 / 작업량 S)** — `src/pages/Play.tsx:79` 의 `canChoose = !out && (phase === 'choosing' || phase === 'countdown')` 는 countdown 에도 버튼을 켜지만, `pick()` 첫 줄 `src/pages/Play.tsx:88` 이 `if (phase !== 'choosing') return;` 로 조용히 삼킨다 → 참가자가 카운트다운 중에 탭하면 선택 표시(`setLocal`)조차 되지 않고 피드백이 없다. 서버(`supabase/migrations/0003_elimination.sql:33-37`)는 `started_at - 300ms` 부터 받아 주므로 "countdown 에도 받아 준다" 와 "choosing 부터 받는다" 중 하나로 **두 줄을 일치**시키면 된다. 권장: `canChoose` 에서 `'countdown'` 을 빼서 버튼을 비활성으로 두는 쪽(서버 왕복 없이 로컬만 고치면 되고 되돌리기 쉽다). 건드릴 파일은 `src/pages/Play.tsx` 1개. 검증은 `npm run build` + 사람 눈. 자동 테스트 수단이 base 에 없으므로 증거는 약하다 — 1순위가 막혔을 때만 고를 것.

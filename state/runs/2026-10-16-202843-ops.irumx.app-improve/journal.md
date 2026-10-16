# 회차 노트 2026-10-16-202843-ops.irumx.app-improve — ops.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:28] base pinned — main@e85afae
- [러너 20:28] autonomy release — 
- [러너 20:28] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 무엇을 왜: `.github/workflows/ci.yml` 한 파일 추가(45줄). 이 저장소에는 `.github` 가 아예 없어 러너 PR 이 "CI 검사 없음" 으로 막혀 있었다. 러너가 실제로 쓰는 순서를 그대로 옮겼다 — `npm ci` → `npm run build` → `npm test`, node 22(`engines: >=22.12.0`), `cache: npm`, `push`·`pull_request`, concurrency 취소, `permissions: contents: read`, `timeout-minutes: 20`. 정적분석은 `npm run build` 안의 `npm run check`(tsc 4개)·`verify-build.mjs` 가 이미 포함하므로 단계를 쪼개지 않았다.
- 확신 없는 곳·검증 못 한 것: **GitHub Actions 러너에서 실제로 돌려 보지 못했다**(push 권한 없음). 로컬 증거는 깨끗한 트리(`node_modules`·`dist`·`.wrangler`·`test-results` 삭제)에서 `CI=true WRANGLER_SEND_METRICS=false` 로 세 명령을 실행해 전부 exit 0, `npm test` 90 passed(unit 49 + api 41)인 것과, YAML 이 `yaml.safe_load` 로 파싱되는 것까지다. actionlint 같은 스키마 검사기는 이 환경에 없어 돌리지 못했다. 특히 미확인인 것: ubuntu-latest 에서 `wrangler dev --local`(workerd) 이 같은 15초에 뜨는지, `actions/setup-node@v4` 의 `cache: npm` 이 첫 실행에서 경고 없이 지나가는지. 둘 다 실패하면 증상은 webServer 기동 실패(= api 41개가 한꺼번에 죽음)로 나타난다.
- 일부러 하지 않은 것: (1) `npx playwright install` — `ui-*.spec.ts`·`shots.spec.ts` 가 하나도 없어 지금은 브라우저가 필요 없다(불필요한 1분). 다만 1.63 의 `playwright` 패키지는 install 스크립트가 없어 `npm ci` 로 브라우저가 깔리지 않으므로(확인함), ui 시험을 넣는 회차에는 이 단계가 반드시 같이 필요하다 — ci.yml 주석에 적어 뒀다. (2) `npm audit` 단계 — 새 CVE 가 뜨면 무관한 PR 이 전부 빨개진다. (3) `package.json` 의 `pretest` 보충 — 명세가 파일 하나로 못 박았다(main 에 아직 없다는 사실은 ideas.json 에 올렸다). (4) CHANGELOG 한 줄 — 이 저장소에 CHANGELOG 파일이 없다.
- 다음 역할이 조심할 것: 이 PR 은 `.github/workflows/` 라 사람 승인이 필요하다(예상된 일). 커밋은 `5b19674` 하나, 프로덕션 코드 변경 0. `npm run build` 는 추적 파일 `public/fonts/*.woff2`·`src/client/styles/fonts.css` 를 다시 쓰지만 바이트가 동일해 `git status` 는 깨끗하다 — 검증하다 디프가 보이면 빌드 산출물이니 커밋하지 말 것. 전체 시험은 `npm run build` 선행이 필수다(main 에 `pretest` 가 없다).
- [러너 20:34] verify passed — 검증 4개 통과 (auto)
- [러너 20:35] pr created — https://github.com/hkjang/ops.irumx.app/pull/2
- [러너 20:35] guard held — .github/workflows/ci.yml 

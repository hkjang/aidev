# 수정 과제 (오류 대응 회차 — 새 아이디어 금지)

- 과제: 릴리즈 검증 `cd web && npm test --silent` 실패 복구 — 실제 `vite build` 를 돌리는 `keep-dist-placeholder` 테스트 경로 (수정 과제 / 가치 4 / 위험 2 / 작업량 S)
- 왜: 마지막 회차가 `cd web && npm test --silent` (exit 1) 로 verify-failed 했고 릴리즈 워크플로가 같은 이유로 두 번 실패했다. 고치지 않으면 다음 회차도 같은 자리에서 멈추고, `.github/workflows/ci.yml` 의 web 잡(`npm ci → lint → test → build`)도 main 에서 계속 빨간 상태다.

## 정찰이 실행으로 확인한 것 (여기서 출발하라)
1. **exit 127 ≠ exit 1 — "node_modules 없음" 은 원인이 아니다.** 이 워크트리(`/home/hkjang/.cache/auto-improve-wt/dataworks`)에는 `web/node_modules` 가 없고, 그 상태로 `cd web && npm test --silent` 를 실제로 돌리면 `sh: 1: vitest: not found` / **exit 127** 이 난다. 러너가 보고한 값은 **exit 1** 이므로 `vitest run` 은 정상 기동했고 **테스트가 실제로 실패**했다(vitest 는 테스트 실패 시 1). 설치 누락 쪽으로 시간을 쓰지 말 것.
2. **`npm ci` 는 lock 과 어긋나지 않았다.** `web/package-lock.json`(lockfileVersion 3)의 루트 `packages[""]` 의 dependencies/devDependencies 가 `web/package.json` 과 글자까지 일치한다(두 파일 직접 대조). `@rolldown/binding-*` 플랫폼 바이너리도 `linux-x64-gnu`·`linux-x64-musl` 포함 16종이 모두 lock 에 있다(package-lock.json:970~1242). 즉 EUSAGE·플랫폼 바이너리 누락도 아니다.
3. **web 소스는 2026-09-29 이후 바뀌지 않았다.** `git log -- web/` 의 최근 커밋은 전부 `chore: release vX` 버전 범프뿐이고, 마지막 실제 web 변경은 `eb6ed09 fix(web): keep tracked web/dist/.gitkeep after vite build` 다. 2026-09-29 회차는 `npm test` 를 9파일/27테스트 통과로 기록했다 → **환경·툴체인에 의존하는 테스트가 범인일 확률이 높다.**
4. **9개 테스트 파일 중 8개는 환경 의존이 없다(전부 읽었다).** `src/api/dataworks.test.ts`(fetch 스텁), `src/lib/labels.ko.test.ts`(순수 문자열), `src/stores/auth-store.test.ts`, `src/features/auth/silent-sso.test.ts`(sessionStorage·history 스파이), `src/features/publish-gate/publish-gate-card.test.tsx`, `src/features/lifecycle/lifecycle-stepper.test.tsx`, `src/pages/products/products-page.test.tsx`, `src/pages/settings/role-management.test.tsx` — 시계·네트워크·파일시스템 의존이 없고 날짜 리터럴은 `created_at: '2026-08-31T00:00:00Z'` 처럼 단정에 쓰이지 않는 고정 fixture 뿐이다. 임포트 대상(`src/test/dataworks-fixtures.ts`, `src/test/setup.ts`)도 실제로 존재한다.
5. **남은 1개가 유일하게 외부에 의존한다: `web/src/test/keep-dist-placeholder.test.ts`.** 이 파일은 `// @vitest-environment node` 로 돌며 vite 의 JS API `build()` 를 **실제로** 호출한다 — 실제 `web/vite.config.ts` 를 `configFile` 로 읽고, `os.tmpdir()` 에 fixture 를 만들고, `emptyOutDir: true` 로 비운 뒤 `keepDistPlaceholder()` 의 `closeBundle` 이 `.gitkeep` 을 되살리는지 파일로 검사한다. 즉 Vite 8(Rolldown) 번들러·플러그인 훅·tmpdir 쓰기에 결과가 달려 있다.

**→ 1순위 가설(소거법, 미확인):** `keep-dist-placeholder` 2사례 중 하나 이상이 `vite build` 단계에서 실패한다. 정찰 환경은 `npm ci`/`npm`/`node -e` 가 샌드박스에 막혀 **이 가설을 실행으로 확인하지 못했다(미확인)**. 구현자는 가설을 믿지 말고 아래 1단계로 먼저 **진짜 실패 메시지를 확보**하라.

## 작업 순서
1. **재현 먼저.** `cd web && npm ci && npm test` 를 돌려 실패한 파일·사례·메시지를 그대로 받아 적는다. 이 출력이 과제의 근거다.
2. **진단.** 실패가 `keep-dist-placeholder.test.ts` 라면 그 테스트의 `build({ ... logLevel: 'silent' })` 가 vite 의 실제 오류를 삼키고 있다 — 진단 중에만 `logLevel` 을 `'info'` 로 바꿔(또는 지워) 원인을 보고, 다 끝나면 `'silent'` 로 되돌린다. 확인할 지점: `web/vite.config.ts:keepDistPlaceholder` 의 `configResolved`(placeholder 경로·원본 내용 읽기)와 `closeBundle`(되쓰기)이 Vite 8 의 build 경로에서 실제로 호출되는지, `emptyOutDir` 이 tmpdir 의 절대 outDir 에 어떻게 적용되는지, fixture `index.html` 에 module script 가 없는 입력에서 번들이 어떻게 처리되는지.
3. **원인 쪽을 고친다.** 플러그인(`web/vite.config.ts`)의 동작이 Vite 8 에서 깨졌으면 플러그인을 고친다. 테스트가 더는 성립하지 않는 방식으로 vite 를 호출하고 있으면 **같은 것을 계속 증명하는 형태로** 테스트 호출만 고친다(아래 금지 사항 참조). 실패가 다른 파일이면 그 파일의 실제 원인을 고친다.
4. **같은 검증을 로컬에서 재현해 통과를 확인**하고 원장에 '수정 과제' 로 기록한다.

## 수용 기준
1. `cd web && npm test --silent` 가 **exit 0**. 러너의 검증 명령과 글자까지 같은 형태로 돌려 확인한다(`--silent` 포함).
2. 수정 전 실패가 **어떤 파일·어떤 사례·어떤 메시지**였는지 원장에 그대로 남는다. 프로덕션 쪽(예: `web/vite.config.ts`)을 고쳤다면 그 파일만 되돌린 상태에서 같은 실패가 다시 나는 것을 실행으로 확인한다.
3. `web/dist/.gitkeep` 이 지켜진다는 원래 보장이 그대로 증명된다: `emptyOutDir` 이 stale 산출물을 여전히 지우고(`stale-chunk.js` 없음), `.gitkeep` 이 **바이트 동일**하게 되살아나며, outDir 에 placeholder 가 없었을 때는 기본 문구로 생성된다. 테스트 2사례의 단정이 약해지지 않아야 한다.
4. `npm run lint`, `npm run build` 통과. `npm run build` 뒤 **`git status --short` 가 깨끗**해야 한다(`web/dist/.gitkeep` 이 삭제·변경 상태로 남지 않음 — 이 테스트가 지키려는 바로 그 불변식이다).
5. 루트에서 `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `go run ./cmd/api-surface-audit`(누락 0) 통과. web/dist 를 임베드하는 `web/embed.go` 때문에 Go 빌드까지 확인해야 한다.

## 건드릴 파일 (프로덕션 2개 이내로 끝낼 것)
- `web/vite.config.ts` : `keepDistPlaceholder()` 의 `configResolved`(21~31행 부근, placeholder 경로 계산과 원본 내용 읽기)·`closeBundle`(33~36행 부근, 되쓰기). Vite 8 에서 훅이 기대대로 돌지 않는 부분만 고친다. `emptyOutDir` 을 끄는 방향은 금지(주석에 이유가 명시돼 있다 — stale 청크가 바이너리에 임베드된다).
- `web/src/test/keep-dist-placeholder.test.ts` : `buildFixture()` 의 `build()` 호출 형태, `logLevel` 되돌리기. 단정 자체는 유지·강화만.
- (재현 결과가 다른 파일을 가리키면 그 파일 + 그 테스트. 그래도 프로덕션 파일 2개 이내로 쪼갤 것.)
- 원장·`docs/OPERATIONS.md` 외에 다른 문서는 손대지 않는다.

## 위험과 피할 것
- **워크플로를 느슨하게 만들어 통과시키는 것은 금지.** `.github/workflows/ci.yml` 의 web 잡(`npm ci`/`lint`/`test`/`build` 4단계)과 `web/package.json` 의 `test: "vitest run"` 은 그대로 둔다. `continue-on-error`, `--passWithNoTests`, `vitest.config.ts` 의 `include` 축소, `it.skip`/`describe.skip`/`test.todo`, 테스트 파일 삭제 — 전부 반려 사유다.
- **의존성 메이저 업그레이드·`package-lock.json` 재생성 금지.** lock 은 package.json 과 이미 동기이고(위 2번) 플랫폼 바이너리도 다 있다. `npm install` 로 lock 을 흔들면 릴리즈 경로를 더 깨뜨린다. `npm ci` 만 쓸 것.
- **릴리즈·빌드 경로를 건드리는 변경이다** — 운영자 규칙상 릴리즈까지 통과하는 것을 확인해야 한다. 수용 기준 4·5(특히 `npm run build` 뒤 `git status` 깨끗함, `go build`)를 빠뜨리지 말 것.
- **빌드 산출물을 커밋하지 말 것.** `web/dist/*` 는 `.gitignore` 되고 `!web/dist/.gitkeep` 만 추적된다. `npm run build` 후 dist 하위 파일이 diff 에 끼지 않았는지 확인.
- 프로필의 함정: `gofmt -l` 은 저장소의 기존 CRLF 때문에 HEAD 에서도 파일명을 출력할 수 있다 — 줄 끝을 건드리지 말 것. 실행 환경 Node 는 22.23.1 이고 **CI 는 Node 24** 다. Node 버전 차이로 재현되지 않으면 그 사실 자체를 원장에 적고, 버전에 의존하지 않는 수정을 택할 것.
- 보호 경로(`internal/proxy/keycloak*.go`·`mcp_oauth.go`, `internal/store` 마이그레이션, `.github/workflows/`)는 이번 과제에서 열 이유가 없다.

## 차선 후보
- **`cd web && npm ci && npm test` 가 실제로 통과해 재현이 안 되면**: 그 사실(명령·출력·Node/npm 버전)을 원장에 '재현 불가' 로 명확히 남기고, 재현 불가를 가장 잘 설명하는 환경 의존을 제거한다 — 1순위는 `keep-dist-placeholder.test.ts` 가 `os.tmpdir()` 에 쓰는 대신 `web/` 아래 임시 디렉터리를 쓰게 하고(`afterEach` 정리 유지), `logLevel: 'silent'` 로 vite 오류를 삼키지 않도록 실패 시 원인이 드러나게 바꾸는 것. 테스트가 지키는 불변식(수용 기준 3)은 그대로 유지한다. 이 경우에도 `npm test --silent`·`npm run build`·`go build ./...` 전체를 돌려 둘 것.

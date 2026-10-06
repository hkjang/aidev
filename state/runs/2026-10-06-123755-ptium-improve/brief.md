# 과제서 — 2026-10-06 (수정 과제)

- 과제: CI 의 `npm audit --audit-level=high` 가 막는 권고를 **락파일/overrides 범위에서** 해소해 PR #44 를 녹색으로 돌리기 (가치 4 / 위험 2 / 작업량 S)
- 왜: 지난 회차 변경 `cb30ae5` 는 `docs/USER_GUIDE.md` **한 파일**뿐인데 CI 가 두 번 실패했다 — 변경이 원인일 수 없다. `.github/workflows/ci.yml:40` 의 `npm audit --audit-level=high` 는 레지스트리의 **살아 있는** 권고 목록을 조회하므로 락파일이 한 글자도 안 바뀌어도 권고가 새로 올라온 날 통과→실패로 뒤집힌다. 수리 시도 `d174036`(`vitest ^3.2.7 → ^5.0.3` 메이저 2단 + `vitest.config.ts` 를 `projects` API 로 재작성)도 실패했으므로, 이번엔 **가장 작은 버전 이동**부터 차례로 올라가며 매 단계를 실제로 돌려 확인한다.

## 수용 기준
1. `cd web && npm ci` 뒤 `npm audit --audit-level=high` 가 **exit 0** 이다 (출력을 과제 기록에 그대로 붙일 것 — 어떤 권고였고 어떤 버전으로 넘겼는지).
2. `npm run typecheck` 와 `npm run build` 가 통과한다.
3. `npm test`(= `vitest run`)가 통과한다. CI 에는 vitest 단계가 **없지만** 이 저장소의 웹 단위 테스트는 로컬 관례이므로 깨뜨리지 않는다.
4. `cd server && go test -race ./...` 와 `go vet ./...` 가 통과한다(이 과제는 Go 를 건드리지 않으므로 회귀 없음 확인용).
5. `.github/workflows/ci.yml` 이 **한 글자도 바뀌지 않았다**(`git diff --stat` 으로 증명). 게이트를 느슨하게 만들어 통과시키는 것은 금지.
6. 바뀐 프로덕션 파일이 **3개 이하**다(`web/package-lock.json` + 필요하면 `web/package.json`, 최후에만 `web/vitest.config.ts`).

## 절차 (이 순서대로, 각 단계에서 멈추고 실제로 돌릴 것)

**0단계 — 사실 확보 (이 정찰이 못 한 것).** 이 샌드박스는 `npm` 실행이 차단되어 **어떤 권고가 걸렸는지 미확인**이다. 먼저:
```
cd web
npm ci
npm audit --audit-level=high
```
출력에서 ① 권고 ID/제목 ② 심각도 ③ 취약 패키지와 **의존 경로**(`node_modules/...` 전체 경로) ④ `fix available` 줄을 적어 둘 것. 이후 단계는 전부 이 출력에 달려 있다.

**1단계 — 패치만으로 끝나면 끝.** `npm audit fix`(**`--force` 금지**) → 다시 `npm audit --audit-level=high`. exit 0 이면 끝. 바뀐 파일은 `web/package-lock.json` 하나다. 이것이 1순위이고 가장 안전하다.

**2단계 — `npm audit fix` 가 "breaking change" 를 요구하면, 경로를 보고 갈라진다.**
- 경로가 `node_modules/vite-node/node_modules/vite` 또는 `node_modules/vitest/node_modules/vite` 면 그것은 **중첩된 vite 7.3.6** 이다(실측: `package-lock.json:3423`, `:3598`). 앱 자신의 vite 는 최상위 8.2.0 인데(`:3324`), `vitest` 3.2.7 이 vite 를 **peer 가 아니라 일반 dependency** 로 `"^5.0.0 || ^6.0.0 || ^7.0.0-0"` 범위로 선언하기 때문에(`:3525`) 트리에 vite 가 두 벌 들어와 있다. 최상위 `vite` 를 올려도 이 사본은 안 바뀐다.
  - 먼저 `web/package.json` 에 `overrides` 로 **패치 범위 안의 vite 7.x** 를 박아 본다(예: `"overrides": { "vite": "^7.3.7" }` — 실제 패치 버전은 0단계 출력의 `fix available` 이 알려 준다). `npm ls vite` 로 두 사본의 버전을 확인할 것.
  - 패치된 7.x 가 없어서 8 로 가야만 한다면, `overrides` 로 vite 8 을 박지 **말 것** — `:3525` 의 선언 범위를 거짓으로 만든다. 그때는 `vitest` 를 **vite 8.2.0 을 포함하는 선언 범위를 가진 가장 낮은 버전**으로 올린다(`npm view vitest@<ver> dependencies.vite` 로 범위를 먼저 확인). `^5.0.3` 으로 두 단계 뛰지 말 것 — `d174036` 이 그렇게 했고 두 번 실패했다.
  - vitest 를 올려 `environmentMatchGlobs` 가 사라진 버전이 되면 `web/vitest.config.ts` 를 고쳐야 한다. 그때만 그 파일을 건드리고, `npm test` 가 **두 환경(node 용 `src/**/*.test.ts`, jsdom 용 `src/**/*.test.tsx`) 모두 실제로 돌았는지** 출력의 파일 수로 확인할 것 — 설정을 잘못 쓰면 한쪽이 조용히 0개가 된다.
- 경로가 `node_modules/postcss`(실측 8.5.25, `:2926`) 처럼 **vite 가 끌어오는 공용 패키지**면 `web/package.json` 의 `overrides` 로 패치 버전을 박는 것이 정석이다. vitest 를 건드릴 이유가 없다.
- `No fix available` 이고 dev 전용 경로라면: **ci.yml 을 고치지 말고** `overrides` 로 패치본을 강제하는 길을 먼저 다 시도하고, 그래도 길이 없으면 **고치지 않고 멈추고** 0단계 출력 전문을 노트에 남길 것(게이트 완화는 운영자가 반려한다 — 판단은 사람에게 넘긴다).

## 건드릴 파일
- `web/package-lock.json` — `npm audit fix` / `npm install` 이 다시 쓴 결과. **손으로 편집 금지**(integrity 해시가 깨지면 `npm ci` 가 실패한다).
- `web/package.json` — 필요한 경우에만 `overrides` 한 블록, 또는 `devDependencies.vitest` 한 줄.
- `web/vitest.config.ts` — 2단계에서 vitest 메이저가 불가피할 때만.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
cd web && npm ci
cd web && npm run typecheck
cd web && npm run build
cd web && npm audit --audit-level=high      # ← 이것이 이번 과제의 본 게이트, exit 0 이어야 한다
cd web && npm test
cd server && go test -race ./...
cd server && go vet ./...
cd server && go test -count=1 ./internal/docs
git diff --stat                             # .github/ 와 server/ 에 변경 0 이어야 한다
git diff --check
```
`docker build --build-arg VERSION=ci --tag ptium:ci .` 는 CI 의 마지막 단계다. 도커가 있으면 돌려 보고, 없으면 "미실시" 로 적을 것.

## 위험과 피할 것
- **`.github/workflows/ci.yml` 은 보호 경로다.** `--omit=dev`·`--audit-level=critical`·`|| true`·`continue-on-error` 중 어느 것도 넣지 말 것. 운영자가 되풀이해 말한 "워크플로를 느슨하게 해 통과시키는 것은 금지" 가 정확히 이 자리다.
- **`d174036` 의 접근(vitest 3→5 + config 재작성)을 그대로 다시 내지 말 것.** 두 번 실패했다.
- 버전 4곳(`VERSION`·`api/openapi.yaml`·`deploy/kubernetes.yaml`·`docs/offline-deployment.md`)과 릴리즈 노트는 건드리지 말 것 — `server/internal/config/stamped_test.go` 가 어긋남을 잡는다.
- `web/vite.config.ts`(앱 빌드 설정)와 `web/tsconfig.*` 은 건드리지 말 것. 참고로 `tsconfig.node.json:11` 은 `vite.config.ts` 만 include 하므로 **`vitest.config.ts` 는 `tsc -b` 가 보지 않는다** — 그 파일의 오류는 typecheck 로 안 잡히고 `npm test` 로만 잡힌다.
- `npm audit` 은 네트워크 의존이다. 오프라인에서 "통과" 를 주장하지 말 것 — 돌지 않았으면 "미실시" 로 적을 것.
- 커밋은 락파일 변경 이유를 한 문장으로 적을 것(어떤 권고를 어떤 버전으로 넘겼는지). 소스 문자열 검사(grep)를 증거로 내지 말 것 — 증거는 `npm audit` 의 exit 코드와 출력이다.

## 차선 후보
목록 안 인용(`- > 인용`)의 `>` 도 떼기 (2/2/S) — `prose.go:281` 의 `isListLine` case 에서 `withoutListMarker` 뒤에 `withoutQuoteMarker` 를 한 번만 적용하는 좁은 수정. 1순위가 0단계에서 "권고가 하나도 없다"(즉 CI 실패가 이미 저절로 풀렸다)로 끝나면 이것을 하되, **그때는 `npm audit` 출력을 노트에 붙여 실패가 사라졌음을 증명할 것**.

## 이 과제서에서 추측인 것 (구현자는 0단계로 전부 대체할 것)
- 어떤 권고가 걸렸는지: **미확인**. 이 샌드박스는 `npm` 실행과 `WebSearch` 가 모두 차단되어 확인하지 못했다. `d174036` 의 커밋 제목("Lift vitest and postcss past the advisories the audit gate stops on")에서 vitest 계열과 postcss 두 갈래를 추정한 것이다.
- 중첩 vite 7.3.6 이 걸린 패키지라는 것: **추정**. 다만 중첩 사본의 존재와 버전, `vitest` 의 vite 의존 범위는 락파일에서 **실측**했다.
- `d174036` 이 두 번째로도 `npm audit` 때문에 실패했다는 것: **추정**. CI 로그를 못 봤다. `vitest.config.ts` 가 `tsc -b` 대상이 아니라는 점(`tsconfig.node.json:11`)은 실측이므로 typecheck 실패는 아니다.

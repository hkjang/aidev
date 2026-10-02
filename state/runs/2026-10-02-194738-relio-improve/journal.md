# 회차 노트 2026-10-02-194738-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:47] base pinned — main@494d00f
- [러너 19:47] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 우선 과제(verify-failed 수정)가 배정돼 새 후보를 고르지 않았다. 확정한 구조적 원인 하나: `Makefile:test` 는 `npm ci && npm run typecheck && npm test` 만 돌고 **`npm run build` 를 전혀 돌리지 않아**, 두 회차가 `make test` exit 0 을 증거로 내고도 CI/release 가 실제로 돌리는 `npm run typecheck && npm run build`(ci.yml:29-31, release.yml:33-39)에서 죽었다.
- **이번 세션은 재현을 0회 했다** — `npm ci`·`node -e` 가 권한으로 막혔고 이 워크트리에 `web/node_modules` 가 없다. 따라서 과제서의 원인 후보 4개(v20 node 그림자 / `tsc -b` + composite 부재 / `emptyOutDir` / 네이티브 선택 의존성)는 **전부 미확인 가설**이며 과제서에 그렇게 못 박았다.
- 오프라인으로 확인한 것만: `tsconfig.app.json` 에 `composite` 없음, `package-lock.json` 에 linux rollup/esbuild 바이너리 둘 다 있음(1025·541), vite engines `^20.19.0 || >=22.12.0`(1705), `ls -l /home/hkjang/node_modules/.bin/node` → `/home/hkjang/node_modules/node/bin/node` (v20 그림자 생존).
- 구현자가 조심할 것: 지난 회차가 "차선" 판정을 받은 이유가 **셸의 node 버전을 npm 스크립트 안의 것과 같다고 가정한 오진**이다. 반드시 재현 출력부터 캡처하고, 과제서 가설과 다르면 가설을 버려라. 워크플로 느슨화 금지(.github/workflows 0줄).
- `npm run build` 는 `internal/webui/dist` 를 비운다 — 돌린 뒤 `dist/README` 생존과 `git status --short` 를 반드시 확인할 것.
- [러너 19:53] scout done — 수정 과제 — 두 회차 연속 verify 를 죽이는 `npm run typecheck && npm run build` 를 **먼저 재현해 원인을 확정한 뒤

## 구현 노트
- 무엇/왜: `Makefile:test` 끝에 `npm run build` 를 이어 붙여(ed0352e 전례) 로컬 검증 범위를 CI 의 "Frontend typecheck and build" 와 맞췄고, `web/package.json:test` 의 bare `node` 를 `$npm_node_execpath` 로 고정했다. 2파일·2줄, `.github/workflows/` 0줄. 커밋 7111cbe.
- **과제서의 원인 진단은 전부 틀렸다**: 본 명령 `cd web && npm ci && npm run typecheck && npm run build` 는 이 워크트리에서 **exit 0**(TC=0, BUILD=0). 러너의 exit 1 은 **저장소 루트 cwd** 에서 왔다 — 루트에 `package.json` 이 없어 npm 이 상위 `/home/hkjang/package.json`(scripts `{}`)을 집고 `Missing script: "typecheck"` 로 죽는다. CI/release 는 각각 `working-directory: web`·`--prefix web` 이라 무죄다. **이 커밋은 러너의 verify 를 통과시키지 못할 수 있다** — 원인이 저장소 밖이라 저장소에서 고칠 수 없다. 러너/환경 쪽 후보 2개를 ideas.json 에 적었다.
- 확신 없는 곳: `$npm_node_execpath` 는 이 환경(nvm v22.23.1 npm)과 지난 회차의 Node 25 npm 에서만 확인했고 **GitHub Actions 의 setup-node@v4 node 24 에서는 돌려 보지 못했다** — npm 표준 변수이고 release.yml 도 같은 npm 을 쓰므로 동작할 것으로 보지만 CI 실측은 없다. 또 이 변수는 `sh -c` 전개에 의존하므로 Windows 셸에서는 깨진다(이 프로젝트는 Linux/Docker 전용이라 범위 밖).
- 일부러 하지 않은 것: `typecheck`·`build` 스크립트는 재현에서 둘 다 exit 0 이라 **손대지 않았다**(과제서 가설 ① 이 제안한 `$npm_node_execpath` 고정 포함) — 증명된 원인이 없는 변경은 하지 않는다는 원칙. `tsconfig.app.json` 의 composite 부재도 `tsc -b` 가 통과하므로 기각(ideas.json 에 rejected).
- 다음 역할이 조심할 것: `make test`·`npm run build` 는 `internal/webui/dist` 를 **비운다** — 빌드가 실패한 채로 두면 `dist/README` 가 사라져 `//go:embed dist/*` 가 깨진다. 섭동 실험 뒤 반드시 `npm run build` 를 다시 돌리고 `cmp web/public/README internal/webui/dist/README` 로 확인할 것(이번에 그렇게 복구했고 IDENTICAL).
- 증거: `make test` exit 0(go test → go vet → npm ci → typecheck → npm test 27/27 → 새 npm run build) · `go test ./...` 22 ok/FAIL 0 · `go build`·`go vet` exit 0 · webui 앵커 테스트 PASS · `check-env-contract.sh`·`check-static-assets.sh` 통과 · `git status --short` 2파일 · `git diff --check` 무출력.
- [러너 20:00] brief fallback — 차선 — 과제서가 "먼저 재현하라" 고 못 박은 것이 정확히 옳았다: 재현해 보니 본 명령은 exit 0 이고 원인 후보 ①~④(v20 
- [러너 20:00] verify failed — 실패한 검증: npm run typecheck && npm run build   # CI에서 가져옴 (exit 1)

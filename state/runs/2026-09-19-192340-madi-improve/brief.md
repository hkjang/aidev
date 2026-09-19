# 과제서 — 2026-09-19-192340-madi-improve (수정 과제)

- 과제: 깨끗한 체크아웃에서 `go build ./...` 이 통과하도록 `web/dist` 자리표시자(`.gitkeep`)를 커밋하고 Vite 빌드가 그것을 지우지 않게 하기 (가치 4 / 위험 1 / 작업량 S)

- 왜: 러너의 자동 검증(`verify.json` source=auto)은 web 빌드보다 **먼저** `go build ./...` 을 돌리는데, `web/embed.go:7` 의 `//go:embed all:dist` 가 gitignore 된 `web/dist/` 없이는 `pattern all:dist: no matching files found` 로 실패한다 — 2026-09-19 11:22(shepherd PR #6)와 17:33(improve, 커밋 83239b9) 두 회차가 정확히 이 한 줄로 verify-failed 됐다(`…/2026-09-19-172338-madi-improve/verify.txt` 확인). `release.yml`·`ci.yml`·`Dockerfile` 은 모두 `npm run build` 뒤에 Go 를 빌드하므로 워크플로 자체는 결함이 없고, 워크플로는 손대지 않는다(느슨하게 금지). 저장소 쪽에서 빈 자리표시자 하나만 추적하면 순서와 무관하게 Go 모듈이 항상 컴파일된다.

- 수용 기준:
  1) `git clean -xfd web/dist` (또는 새 worktree) 직후 `go build ./...` 과 `go vet ./...` 이 exit 0 — 지금은 `web/embed.go:7:12: pattern all:dist: no matching files found` 로 exit 1.
  2) `cd web && npm run build` 를 돌린 **뒤에도** `git status --short web/dist` 가 비어 있다(추적된 `web/dist/.gitkeep` 이 삭제·변경으로 뜨지 않는다). Vite 는 `emptyOutDir` 기본값으로 `dist/` 를 통째로 비우므로 그냥 파일만 커밋하면 빌드 때마다 "deleted" 로 뜬다 — 반드시 빌드가 다시 만들어 주게 할 것.
  3) 실제 빌드 산출물이 있는 상태에서도 `go build ./...` 통과, 그리고 `.gitkeep` 이 서비스워커 캐시 목록(`vite.config.ts` `offlineAssets` 의 `ASSETS`)에 들어가지 않는다 — 필터가 `assets/` 접두사와 `offline.html` 만 고르므로 지금 구조에서 자동으로 제외되지만 `web/dist/sw.js` 를 열어 확인할 것.
  4) 테스트가 증명할 것: 단위 테스트 추가는 필수 아님(빌드 자체가 증거). 대신 위 1)·2) 의 실제 명령 출력을 회차 노트에 남길 것. 러너 검증(`go build ./...` → `go vet ./...` → `go test ./...` → `cd web && npm run build`)이 이 순서 그대로 깨끗한 트리에서 통과해야 한다.

- 건드릴 파일:
  - `web/dist/.gitkeep` — 새 빈 파일(0바이트). 이것이 `all:dist` 패턴을 만족시킨다.
  - `.gitignore` 4행 `web/dist/` — 바로 아래에 `!web/dist/.gitkeep` 추가. (주의: git 은 부모 디렉터리가 통째로 무시되면 자식 예외가 안 먹는다. `web/dist/` 규칙을 `web/dist/*` 로 바꾸고 `!web/dist/.gitkeep` 를 두는 형태가 맞다. `git check-ignore -v web/dist/.gitkeep` 이 아무것도 출력하지 않아야 하고, `git check-ignore -v web/dist/index.html` 은 여전히 무시돼야 한다.)
  - `web/vite.config.ts` `offlineAssets()` 플러그인의 `generateBundle` 핸들러 — 기존 `this.emitFile({type:"asset", fileName:"sw.js", …})` 옆에 `this.emitFile({ type: "asset", fileName: ".gitkeep", source: "" })` 한 줄 추가(또는 같은 파일에 `keepDistPlaceholder()` 소형 플러그인을 따로 두고 `plugins` 배열에 등록). 이렇게 하면 Vite 가 dist 를 비운 뒤 다시 `.gitkeep` 를 써서 2) 가 성립한다. 주석 한 줄로 이유(`go:embed all:dist` 가 빈 디렉터리를 허용하지 않음)를 적을 것.
  - `.dockerignore` 는 `web/dist` 를 이미 제외하지만 `Dockerfile` 15행이 web 스테이지에서 `dist` 를 복사하므로 변경 불필요. **Dockerfile·workflows 무변경.**
  - `docs/` — 선택. `README`/`docs/admin-guide.md` 에 빌드 절이 있으면 "web/dist/.gitkeep 는 Go 임베드용 자리표시자, 삭제 금지" 한 줄. 없으면 생략(문서 재생성 `node scripts/build-docs.mjs` 비용을 피할 것).

- 검증 명령(이 순서로, 모두 저장소 루트에서):
  1. `git clean -xfd web/dist && go build ./... && go vet ./...` — 수정 전엔 exit 1(재현), 수정 후 exit 0.
  2. `git check-ignore -v web/dist/.gitkeep; echo "exit=$?"` → exit 1(무시 안 됨). `git check-ignore -v web/dist/index.html` → 무시됨.
  3. `cd web && ([ -d node_modules ] || npm ci --no-audit --no-fund) && npm run build` (tsc -b + vite build, 약 20초) 그리고 루트에서 `git status --short web/dist` → 빈 출력. `test -f web/dist/.gitkeep`.
  4. `grep -c gitkeep web/dist/sw.js` → 0.
  5. `go build ./... && go vet ./...` (실제 산출물 포함 상태) → exit 0.
  6. `go test -count=1 ./web/ ./cmd/...` — web 패키지·cmd 컴파일만 확인(수 초). **전체 `go test ./...`·`-race`·브라우저 시험은 돌리지 말 것**(러너가 `go test ./...` 을 자체 검증으로 돌리므로 중복이고, 09-19 두 shepherd 회차가 예산 홀드로 끝난 전례).
  7. `cd web && npx prettier --check vite.config.ts` — 기존 드리프트가 없다면 통과 상태 유지.

- 위험과 피할 것:
  - `web/embed.go` 의 패턴을 바꾸거나 빌드 태그로 임베드를 조건화하지 말 것(오프라인 배포 바이너리 형태가 바뀜). `cmd/madi/main.go:52` `fs.Sub(web.Assets,"dist")` 도 그대로.
  - `.gitkeep` 은 `http.FileServer`(server.go:442)가 `/.gitkeep` 으로 빈 200 을 낼 수 있다 — 실제 산출물 위에서는 정보 노출이 없으므로 허용. 별도 차단 코드를 넣지 말 것(효과 없는 변경 금지 원칙).
  - `emptyOutDir: false` 로 우회하지 말 것 — 오래된 해시 파일이 dist 에 쌓여 임베드 바이너리가 커진다.
  - `web/public/.gitkeep` 으로 복사시키는 방법은 Vite 의 dotfile 복사 여부가 미확인 — emitFile 이 확실하다. Rollup 이 점으로 시작하는 `fileName` 을 거부하면(미확인) 파일명을 `keep` 같은 비-dot 이름으로 바꾸고 .gitignore 예외도 맞출 것.
  - 워크플로(`.github/workflows/*`)·`Dockerfile`·`scripts/release-image.sh`·`scripts/verify-*.sh` 는 읽기만 하고 수정 금지.
  - 이전 회차 커밋 83239b9(SAML return_to)는 이 브랜치(main@fd2c3b5)에 없다. 되살리지 말 것 — 이번 회차는 빌드 수정 하나만.
  - 원장(ledger)에는 '수정 과제' 로 기록: 원인 = 러너 자동 검증 순서(Go 먼저) × gitignore 된 embed 대상 디렉터리; 워크플로 결함 아님.

- 차선 후보: 「로그인 화면이 비로그인 도착 경로를 SAML/OIDC 시작 링크에도 return_to 로 넘기기」(가치 3 / 위험 1 / S, `web/src/IdentityLogin.tsx:25`·`App.tsx:402`) — 단, 1순위가 성립하지 않는 경우란 거의 없다(원인이 verify.txt 한 줄로 확정). 1순위를 먼저 끝내지 않으면 어떤 과제든 같은 verify 에서 다시 실패한다.

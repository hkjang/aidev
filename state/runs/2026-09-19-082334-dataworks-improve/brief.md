# 정찰 과제서 — dataworks (2026-09-19)

## 우선 과제 판정: "릴리즈 워크플로 실패" 는 저장소 결함이 아님 (수정할 코드 없음)

러너가 "마지막 회차 error / hold: budget, 릴리즈 워크플로가 같은 이유로 두 번 실패" 로 배정했지만, 확인 결과 다음과 같다.

- `state/runs/2026-09-18-163401-dataworks-improve/evidence.json`: `outcome: error`, `result: "hold: budget"`, stage `improve` 가 `state: hold`, reason **"회차 예산($22)이 오늘 남은 상한을 넘음"**, `base_sha`·`head_sha`·`pr` 모두 빈 값, `usage: []`. 즉 코드 체크아웃도 하기 전에 러너의 일일 예산 상한에서 멈춘 회차이며 GitHub Actions 는 실행되지 않았다.
- 마지막 실제 릴리즈 회차 `2026-09-17-183336-dataworks-improve`: `release.json` 은 `status: released, tag: v0.9.56, github_release: true`. `ci-0c425619aa0b.json` 의 5개 체크(`Go build / vet / test`, `Web lint / test / build`, `build`, `deploy`, `report-build-status`) 모두 `conclusion: success`.
- 저장소의 워크플로는 `.github/workflows/ci.yml` 하나뿐(go 잡: build/vet/test/api-surface-audit, web 잡: npm ci/lint/test/build). 릴리즈 전용 워크플로 파일은 없고 릴리즈는 러너(`scripts/release.sh`·`gh_release.ps1`)가 수행한다. "같은 이유로 두 번 실패" 한 워크플로 단계·스크립트·테스트는 저장소 어디에도 기록이 없다(미확인이 아니라 **존재하지 않음**).

따라서 워크플로/스크립트/테스트를 고칠 대상이 없다. 워크플로를 느슨하게 만들 일도 없다. 원장에는 아래를 '수정 과제' 가 아닌 **"오류 대응: 러너 예산 홀드였음, 코드 결함 아님"** 으로 기록하고, 이번 회차는 아래 과제를 수행한다. (러너의 예산 상한 자체는 aidev 쪽 설정이라 이 저장소 밖.)

---

- 과제: `npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 지우는 문제를 vite 플러그인으로 해결 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/embed.go` 는 `//go:embed all:dist` 로 `web/dist` 를 임베드하므로 소스 체크아웃에 `web/dist/.gitkeep`(추적됨, `.gitignore` 15–16행 `web/dist/*` + `!web/dist/.gitkeep`)이 반드시 있어야 `go build ./...` 가 된다. 그런데 `web/vite.config.ts` 에 아무 조치가 없어 vite 의 기본 `emptyOutDir` 이 `.gitkeep` 을 삭제하고, 최근 6회차(2026-09-11·12·13·15·17·17) 모두 "빌드가 지운 `web/dist/.gitkeep` 복원" 을 손으로 반복했다. 빌드 뒤 작업 트리가 더러워지고, 복원을 잊으면 `go build` 가 깨진다.
- 수용 기준:
  1) `cd web && npm run build` 직후 `git status --porcelain web/dist` 출력이 비어 있다(`.gitkeep` 이 삭제 상태(` D`)로 뜨지 않음). 빌드 산출물(`index.html`, `assets/*`)은 종전과 같이 생성되고 stale 파일은 여전히 비워진다(`emptyOutDir` 을 끄지 않는다).
  2) 빌드 뒤 `go build ./...` 와 `go test ./internal/proxy/ -run SPA` 류 기존 테스트가 그대로 통과한다(임베드 트리에 `.gitkeep` 이 추가로 들어가도 SPA 핸들러 동작은 변하지 않음 — 이미 소스 체크아웃 빌드에서는 `.gitkeep` 만 임베드되고 있었다).
  3) 증명: 실제 `vite build` 를 돌린 뒤 파일 존재를 확인하는 것이 증거다(빌드를 돌리지 않는 소스 문자열 검사·설정 객체 스냅샷 테스트는 증거로 삼지 않는다). vitest 단위 테스트를 추가한다면 플러그인의 `closeBundle`(또는 `writeBundle`) 훅을 실제 임시 outDir 로 호출해 파일이 생기는지 보는 정도로 충분하며, 없어도 1)의 명령 결과를 원장에 남기면 된다.
- 건드릴 파일:
  - `web/vite.config.ts` — `plugins` 에 소형 인라인 플러그인 추가: `closeBundle()` 에서 `node:fs` 로 `path.join(outDir, '.gitkeep')` 을 빈 파일로 다시 쓴다(`outDir` 은 `configResolved` 에서 `config.build.outDir` 로 읽음). `build.emptyOutDir: false` 로 우회하지 말 것(stale 청크가 임베드됨).
  - (선택) `web/embed.go` 주석 — ".gitkeep 은 Vite 빌드 뒤에도 플러그인이 다시 만든다" 한 줄. 코드 변경은 없음.
  - (선택) `docs/RELEASE_GUIDE.md` 또는 `README` 의 웹 빌드 절에 한 줄. 내용 확인은 안 했으므로(미확인) 관련 절이 있을 때만.
- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build && git status --porcelain web/dist` (마지막 출력이 비어야 함; 이번 정찰에서는 web 빌드를 실행하지 않았다 — 구현자가 첫 실행)
  - `go build ./... && go vet ./... && go test ./...` (약 2분, CI 와 동일)
  - `go run ./cmd/api-surface-audit` (gap 0 유지)
  - `gofmt -l` 은 Go 변경이 없으면 불필요
- 위험과 피할 것:
  - `build.emptyOutDir: false` 금지(stale 산출물이 `go:embed all:dist` 로 바이너리에 들어감).
  - `.gitignore`·`web/embed.go` 의 embed 지시문·`Dockerfile` 22행(`COPY --from=web-build /src/web/dist ./web/dist`) 은 건드리지 말 것. Docker 빌드에서는 `.gitkeep` 이 생겨도 무해하다.
  - `vite.config.ts` 의 `base: '/dataworks/'`·proxy·alias 는 그대로 둘 것(SPA 라우팅·dev 프록시 영향).
  - 실제 출력이 바뀌지 않는 수정을 넣지 말 것(운영자 반복 지시) — 이 과제는 빌드 뒤 작업 트리 상태가 실제로 달라지므로 해당 없음.
  - auth/migrations/workflows 는 건드리지 않는다.
- 차선 후보: `internal/dataworks` 도메인 함수 단위 테스트 보강 (가치 3 / 위험 1 / 작업량 M) — `internal/dataworks/domain.go:163 EvaluatePublishGateV2` 의 strict/non-strict 분기와 `domain.go:415 EvaluateRetirementCandidate` 의 조건 조합(비용·워터마크 정체·fit 점수·활성 엔타이틀먼트 유무)을 `domain_test.go` 에 표 테스트로 추가. 현재 `domain_test.go` 는 7개 테스트 220줄이며 `EvaluatePublishGateV2` 직접 테스트는 없음(`TestEvaluatePublishGate*` 는 V1 `EvaluatePublishGate` 대상). 검증: `go test ./internal/dataworks/ -v`. 실제 함수를 실제 `store` 타입으로 호출하는 테스트만 인정(대역 금지).

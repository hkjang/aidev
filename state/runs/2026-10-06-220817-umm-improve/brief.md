- 과제: `make test-go` 가 CI 의 Go 게이트 다섯 개 중 한 개만 돌린다 — 지난 회차가 `test-web` 에서 닫은 드리프트의 나머지 절반 (가치 3 / 위험 1 / 작업량 S)
- 왜: `.github/workflows/ci.yml` 의 `--- Go ---` 절은 게이트 다섯 개(`go vet`, `go test -p 1 ./...`, `go test -race -count=1 ./internal/intelligence`, `govulncheck ./...`, 임베딩 품질 리포트)를 돌리는데 `Makefile:7-9` 의 `test-go` 는 `go vet ./...` + `go test ./...` 둘뿐이다. 특히 `govulncheck` 은 **커밋이 아니라 취약점 DB 가 판정을 바꾸는** 게이트라서, 어제 초록이던 트리가 아무도 손대지 않아도 오늘 빨개진다 — 2026-10-06 회차(`daf50ae`)가 `npm audit` 을 빼먹은 `test-web` 때문에 PR #167 을 잃고 수리 회차를 한 번 더 쓴 것과 **글자 그대로 같은 실패 모드**이고, 그 회차는 Makefile 주석에 "Keep this list in step with the Frontend section of .github/workflows/ci.yml" 를 적어 두고도 Go 쪽 절반을 열어 두었다.
- 수용 기준:
  1) `make test-go` 가 `ci.yml` 의 `--- Go ---` 절과 **같은 게이트를 같은 순서로** 돌린다. make 자신의 명령 에코로 다섯 줄이 실제로 RUN 되는 것을 보이고 각각의 종료 코드를 보고한다(테스트가 아니라 make 타깃이므로 증거는 실행 출력이다).
  2) `go test ./...` → `go test -p 1 ./...` 로 바뀌어 로컬도 CI 와 같은 직렬 실행을 한다. **`-p 1` 의 근거를 새로 측정할 필요는 없다** — `ci.yml:52-56` 이 이미 측정값("measured 2 failures in 3 runs, 0 in 3 when serialised", 116s→120s)을 적어 두었으므로 Makefile 주석은 그 자리를 가리키면 된다.
  3) 임계값·범위를 **느슨하게 만들지 않는다**: `-race` 대상은 `./internal/intelligence` 그대로, govulncheck 은 `./...` 그대로, 임베딩 리포트의 `-run 'Quality|Vocabulary|Threshold|Semantic' -v` 그대로. 로컬에만 검사를 **더하기만** 한다.
  4) 새 외부 의존(네트워크·DB)을 주석으로 정직하게 적는다: govulncheck 은 `go run ...@latest` 라 네트워크가 필요하고, `-p 1 ./...` 의 통합 테스트는 `POSTGRES_DSN` 이 없으면 SKIP 된다. **DSN 없는 PASS 를 "돌았다" 고 쓰지 말 것.**
- 건드릴 파일:
  - `Makefile:7-9` `test-go` 타깃 — 레시피를 `ci.yml` 의 Go 절 다섯 줄과 같은 순서로 맞추고, `test-web` 타깃(`Makefile:11-31`)에 이미 있는 것과 같은 톤의 주석을 단다(왜 `-p 1` 인지는 `ci.yml` 을 가리키고, govulncheck 이 커밋과 무관하게 판정이 바뀌는 게이트라는 것을 적는다). **프로덕션 파일 1개.**
  - (선택, 같은 파일이므로 비용 0) `test-web` 주석이 "Frontend section" 만 가리키므로, 두 타깃이 같은 규칙을 따른다는 것이 읽히게 하는 선에서만 손댈 것. 문장을 다시 쓰려고 하지 말 것.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `make -n test-go` — 레시피가 의도한 다섯 줄인지 먼저 눈으로 확인(빠름).
  - `make test-go` — 다섯 게이트 전부. 격리 PostgreSQL 17 을 띄워 `POSTGRES_DSN` 을 주고 돌릴 것(선례: `docker run --rm -d -e POSTGRES_PASSWORD=... -p 15436:5432 postgres:17`, 검증 뒤 컨테이너 삭제). DSN 없이도 돌지만 그때는 통합 테스트가 SKIP 이므로 그렇게 보고할 것.
  - `make -n test` 로 `test: test-go test-web` 두 반쪽이 그대로인지 확인.
  - `scripts/check-version.sh` (버전은 올리지 않는다 — 개선 회차 관례).
  - 릴리즈 경로 확인: `docker build --build-arg VERSION=0.76.8 .` (운영자 규칙 4번. `Dockerfile` 은 `make` 를 쓰지 않으므로 영향이 없을 것으로 **예상**하지만 지난 회차가 같은 파일에서 이것을 돌렸으므로 같이 돌릴 것. 검증 이미지는 삭제.)
  - Go·웹 소스 변경이 0줄이므로 `npm` 게이트는 돌리지 않아도 된다(`make test-web` 은 이미 지난 회차가 닫았다).
- 위험과 피할 것:
  - **`.github/workflows/ci.yml` 을 고치지 말 것.** 이번 변경은 로컬을 CI 에 맞추는 것이고 그 반대가 아니다. 릴리즈를 반복해 깨뜨린 구역이다(운영자 규칙 4번).
  - **`npm ci` 를 넣지 말 것** — `test-web` 이 같은 이유로 일부러 빼 두었다(설치는 `make web` 의 일, 테스트 타깃이 되돌릴 수 없는 환경 변경을 하면 안 된다. 운영자 규칙 10번).
  - **govulncheck 이 지금 트리에서 실패할 수 있다** — 그러면 그것은 이 변경의 결함이 아니라 CI 도 똑같이 빨갛다는 뜻이다(`main` 의 최신 CI 결과를 `gh run list`/`gh run view` 로 확인해 보고할 것). 실패를 숨기려고 `-tags`·대상 축소·`|| true` 로 느슨하게 만들지 말 것. 실패하면 그 사실을 보고하고 락파일이 아니라 Go 의존성 쪽 수리는 **다른 과제**로 남길 것.
  - `-race` 는 이 환경에서 **확인했다**: 정찰이 `go test -race -count=1 ./internal/intelligence` 를 실제로 돌려 `ok github.com/hkjang/umm/internal/intelligence 5.791s` — cgo/gcc 문제 없고 통과한다(5.8초, 느린 게이트가 아니다).
  - **govulncheck 은 이 환경에서 돌려 보지 못했다 — 미확인.** 정찰 세션의 샌드박스가 `go run ...@latest`(네트워크 모듈 가져오기)를 승인하지 않아 실행이 막혔다. 즉 "지금 트리에서 govulncheck 이 통과한다" 는 보증이 없다. 구현자는 이것을 **가장 먼저** 돌려 보고, 실패하면 그 출력(GHSA 번호·모듈)을 그대로 보고할 것. 환경이 네트워크를 못 쓰면 그 사실을 적고 나머지 네 게이트만 증거로 낼 것 — 돌리지 않은 것을 돌았다고 쓰지 말 것.
  - `-count=1` 은 CI 의 `go test -p 1 ./...` 에 **없다**(race 단계에만 있다). CI 에 없는 플래그를 로컬에만 더해 드리프트를 반대 방향으로 만들지 말 것 — 캐시 끄기를 하고 싶으면 그것은 별 과제다.
  - 보호 경로(`internal/auth/`, `migrations/`, `.github/workflows/`)와 `VERSION`·`web/package.json` 은 건드리지 않는다.
- 차선 후보: `web/src/lib/edge-vocabulary.ts` 의 알려진 라벨 12개를 실제 타입(`web/src/api.ts` 의 `EdgeRelation`/`EdgeOrigin`)·실제 `setLocale`/`translate`·실제 `en.ts` 로 못 박기 — `web/src/lib/` 에서 유일하게 테스트가 없는 모듈(파일 목록으로 확인). **버그 수정이 아니라 테스트 공백 보강 + 드리프트 방어**로 정직하게 적을 것: `?? relation` 폴백은 `migrations/010_memory_graph.sql:56`(relation 6값)·`:60`(origin 6값)의 CHECK 제약이 값을 묶어 두어 현재 도달 불가다 — 정찰이 두 줄을 직접 열어 사전의 12키와 글자 그대로 일치하는 것을 **확인했다**. 시험 파일 1개, 프로덕션 0파일.

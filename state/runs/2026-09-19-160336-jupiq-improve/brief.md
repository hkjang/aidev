# 과제서 (정찰 2026-09-19, base main@001e637)

## 진단 먼저 — "릴리즈 워크플로 실패"의 실체
- 지난 회차(2026-09-19-084840-jupiq-shepherd)의 journal.md에는 수리·심사 노트만 있고 심사 권고는 **merge, 결함 없음**이다. 회차가 'error' 로 끝난 사유는 `hold: budget` — 러너의 USD 예산 소진 홀드이지, 저장소의 `.github/workflows/release.yml` 이 실패한 기록은 어디에도 없다.
- `release.yml` 은 c21e108 이후 변경이 없고 그 파일 그대로 v1.7.0 릴리스가 성공했다(태그 v1.7.0 = VERSION 1.7.0). GitHub Actions 실행 이력은 이 세션에서 `gh` 호출이 승인되지 않아 **미확인**.
- 로컬 재현: `go test ./...` 전 패키지 통과(main@001e637). check-version/check-screenshots/vet 는 명령 승인 문제로 이 세션에서 **미실행**(구현자가 먼저 돌릴 것).
- 결론: 고칠 코드 결함이 재현되지 않는다. 운영자 규칙("실제 동작이 바뀌지 않는 수정은 넣지 말 것")에 따라 워크플로·테스트를 억지로 손대지 않는다. 대신 **같은 이유(예산)로 다시 멈추지 않도록 릴리즈 검증의 반복 비용을 줄이는 저장소 측 과제**를 수정 과제로 잡는다 — 러너의 릴리즈 역할이 매번 release.yml 의 "소스 검사와 테스트" 단계를 손으로 여러 명령으로 나눠 치고 그 출력을 읽는 것이 예산을 태우는 구조이므로, 그 단계를 한 명령으로 묶고 출력을 실패 시에만 보이게 한다. 워크플로 자체는 그대로 두고(느슨하게 만들지 않음) 워크플로가 새 타깃을 부르게 하지도 않는다.

- 과제: 릴리즈 검증 단계를 한 번에 재현하는 `make test-integration`·`make release-check` 타깃과 실행법 문서화 [수정 과제] (가치 3 / 위험 1 / 작업량 S)
- 왜: release.yml 71~86행(go mod verify → vet → govulncheck → go test → 통합 테스트 → check-screenshots → npm ci/audit/lint/test/build)을 로컬에서 재현하려면 지금은 명령 11개를 순서대로 손으로 쳐야 하고, 통합 테스트는 DSN 환경변수를 매번 손으로 조립해야 한다(2026-09-16·09-17 회차 모두 임시 컨테이너를 손으로 띄웠다). 한 타깃으로 묶으면 릴리즈 역할이 도구 호출 1~2번으로 같은 검증을 끝내 예산 홀드로 끊길 가능성이 줄고, 사람도 같은 절차를 쓴다.
- 수용 기준:
  1) `make test-integration` 이 `JUPIQ_INTEGRATION_TEST_DSN` 이 비어 있으면 "DSN 없음, 예: docker run … postgres:16-alpine … / export JUPIQ_INTEGRATION_TEST_DSN=…" 안내를 출력하고 **비0으로 종료**한다(조용히 skip 금지 — 지금 go test 는 DSN 없으면 조용히 skip 한다). DSN 이 있으면 `go test -count=1 -p=1 -run Integration ./internal/store ./internal/api` 를 그대로 실행한다.
  2) `make release-check` 가 release.yml 71~86행과 **같은 명령·같은 순서**를 실행한다(`go mod verify`, `go vet ./...`, `govulncheck ./...`(설치돼 있지 않으면 `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0` 로 대체), `go test ./...`, `$(MAKE) test-integration`, `node scripts/check-screenshots.mjs`, `cd web && npm ci && bash ../scripts/npm-audit-retry.sh high && npm run lint && npm test && npm run build`). 첫 실패에서 멈추고(`set -euo pipefail` 또는 make 의 기본 동작), 성공 시 마지막 줄에 `release-check OK` 를 찍는다. 기존 `lint`/`test` 타깃은 그대로 둔다.
  3) `make help` 목록에 두 타깃이 한 줄 설명과 함께 나타난다(Makefile 8행 `.PHONY` 와 `## ` 주석 관례).
  4) docs/ADMIN_GUIDE.md(또는 README 의 개발 절 — 어느 쪽이 개발 절차를 담고 있는지 열어 보고 정본 하나에만) 에 "릴리스 전 로컬 검증" 소절: postgres:16-alpine 컨테이너 한 줄(`docker run -d --name jupiq-it -e POSTGRES_DB=jupiq_test -e POSTGRES_USER=jupiq -e POSTGRES_PASSWORD=… -p 5432:5432 postgres:16-alpine`), DSN 한 줄, `make release-check` 한 줄, 그리고 `internal/auth/authtest` 가짜 제공자가 **비loopback 로컬 인터페이스**를 요구해 없으면 해당 테스트가 skip 된다는 주의 한 줄. PDF 재생성은 하지 않는다(가이드 본문 변경이 개발자용 한 소절이면 PDF 는 다음 릴리즈에서 함께 굽는다 — 이전 회차에도 같은 전례, 심사 노트 12행).
  5) 테스트가 증명할 것: (a) DSN 없이 `make test-integration` → 비0 종료 + 안내 문구, (b) 임시 postgres:16-alpine 을 띄우고 DSN 을 주면 `make test-integration` 이 통합 테스트를 실제로 돌려 통과(출력에 `ok  	github.com/hkjang/jupiq/internal/store` 와 `internal/api` 가 있고 `no tests to run` 이 아님), (c) `make release-check` 전체 통과와 마지막 줄 `release-check OK`. 세 결과를 회차 노트에 출력 꼬리와 함께 적을 것.
- 건드릴 파일:
  - `Makefile` — `.PHONY` 에 `test-integration release-check` 추가; `test-integration` 타깃(DSN 검사 → go test -run Integration); `release-check` 타깃(위 순서). `VERSION` 변수·`lint`·`test`·`build` 는 그대로.
  - `docs/ADMIN_GUIDE.md` 또는 `README.md` — 개발/릴리스 절차가 이미 있는 쪽에 "릴리스 전 로컬 검증" 소절 추가. 둘 다에 쓰지 말 것(정본 하나).
  - 건드리지 않음: `.github/workflows/*.yml`, `scripts/*.sh`, `scripts/check-version.sh`(VERSION 은 1.7.0 그대로 — 버전 올리는 것은 릴리즈 역할의 일).
- 검증 명령:
  - `make help` (두 타깃 표시)
  - `make test-integration` (DSN 없이 → 비0, 안내)
  - `docker run -d --name jupiq-it -e POSTGRES_DB=jupiq_test -e POSTGRES_USER=jupiq -e POSTGRES_PASSWORD=it -p 5432:5432 postgres:16-alpine` → `JUPIQ_INTEGRATION_TEST_DSN='postgres://jupiq:it@127.0.0.1:5432/jupiq_test?sslmode=disable' make test-integration`
  - `JUPIQ_INTEGRATION_TEST_DSN=… make release-check` (전체, 수 분 소요 — npm ci·govulncheck 포함)
  - `./scripts/check-version.sh`, `node scripts/check-screenshots.mjs`, `go vet ./...`, `gofmt -l .`(변경 없음 확인), `git status` 로 `web/dist`·`bin`·컨테이너 정리(`docker rm -f jupiq-it`).
- 위험과 피할 것:
  - release.yml 을 새 타깃을 호출하도록 바꾸지 말 것(워크플로 보호 경로, 그리고 이 과제의 목적은 로컬 재현이지 CI 변경이 아님). ci.yml 도 손대지 않는다.
  - `govulncheck` 는 네트워크(vuln DB)가 필요하다 — 오프라인이면 실패하는 것이 정상이며 그 단계를 건너뛰는 옵션을 만들지 말 것(느슨화 금지). 로컬에서 실패하면 그대로 노트에 적는다.
  - `npm-audit-retry.sh` 는 `NPM_CI_LOG` 환경변수를 기대한다(release.yml 82~83행) — Makefile 에서 `npm ci 2>&1 | tee` 로 로그 파일을 만들고 `NPM_CI_LOG` 를 넘길 것. `NPM_AUDIT_SKIP` 은 비워 둔다(우회 금지).
  - `make test` 가 이미 `go test ./...` 를 돌리므로 `release-check` 에서 `make test` 를 재사용하면 npm test 가 npm ci 전에 돌 수 있다 — 순서를 release.yml 과 같게 직접 나열할 것.
  - 이번 회차에 코드(Go/TS) 변경은 없다. 코드 변경 충동이 들면 과제 밖이다.
- 차선 후보: queryInt 가 잘못된 page·page_size·limit 을 조용히 기본값으로 바꾸는 부분을 400 으로 (가치 2 / 위험 2 / S) — `internal/api/helpers.go:queryInt`, 프런트는 항상 숫자를 보냄(09-17 확인). 1순위가 성립하지 않는 경우(예: Makefile 변경이 러너 정책상 보호 경로로 막힐 때)에만.

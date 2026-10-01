# 과제서 (정찰, 2026-10-02) — jupiq / base main@92ff88c (v1.8.7)

- 과제: CI·릴리즈 워크플로가 `npm ci` 실패를 `| tee` 로 삼켜 버려 **엉뚱하게 `npm run lint` 가 exit 1 로 터지는 것**을 고친다 (가치 4 / 위험 2 / 작업량 S)
- 왜: `.github/workflows/ci.yml:78` 과 `.github/workflows/release.yml:82` 는 `npm ci 2>&1 | tee "${RUNNER_TEMP}/npm-ci.log"` 를 **`shell:` 지정 없이** 쓴다. GitHub Actions 의 리눅스 기본 셸은 `bash -e {0}` 로 `pipefail` 이 없어 파이프라인의 종료 코드는 마지막 명령(`tee`, 항상 0)이 되고, `npm ci` 가 실패해도 "의존성 설치" 단계는 초록으로 지나간다. 그러면 node_modules 가 반쯤 깔린 상태로 다음 단계가 돌아 **실패가 `npm run lint`(tsc -b, 타입 오류 exit 1) 자리로 밀려서 보고**된다 — 이번 회차에 적재된 증상(`npm run lint` exit 1, 릴리즈 2회 연속 같은 이유)과 정확히 같은 모양이고, 같은 계열의 실패가 2026-09-29·2026-10-02 두 회차에서 "로컬에서는 재현되지 않음" 으로 끝난 이유도 설명한다(로컬은 `npm ci` 가 성공하므로 lint 가 깨끗하다).
- 고치면 실패가 원인 단계에서 원인 메시지로 멈추고, 정찰·수리가 매번 존재하지 않는 타입 오류를 찾아 헤매지 않는다.

## 0단계 (먼저, 15분 상한) — 적재된 실패의 실제 로그를 확인한다
추측으로 고치지 말 것. 아래 순서로 **무엇이 exit 1 을 냈는지** 먼저 본다.
1. `gh run list --workflow release.yml --limit 10` / `gh run list --workflow ci.yml --limit 20 --json databaseId,conclusion,headSha,displayTitle`
2. 실패한 run 의 로그: `gh run view <id> --log-failed` (또는 `--log | grep -n -A40 "타입·린트 검사"`)
3. 판정:
   - 로그가 **`TS2307: Cannot find module …`, `Cannot find name 'process'`, `error TS2688: Cannot find type definition file`, `tsc: not found`** 처럼 **모듈/타입이 없다**고 말하면 → 이 과제서의 진단(설치 실패가 lint 로 밀렸다)이 맞다. 1단계로 간다.
   - 로그가 **실제 소스의 타입 오류**(우리 코드의 파일·행을 짚는 `TS2322`/`TS2345` 류)를 말하면 → **그 타입 오류를 고치는 것이 이번 과제**다. 그때는 아래 1단계 대신 해당 `web/src/**` 파일을 고치고, 1단계는 차선 후보로 남긴다.
   - `gh` 가 없거나 run 을 못 가져오면("미확인") → 1단계를 그대로 진행한다. 1단계는 로그 없이도 성립하는 실제 결함이다.
4. 어느 쪽이든 로컬 기준선을 한 번 찍어 둔다: `npm --prefix web ci` → `npm --prefix web run lint` → `npm --prefix web test`.
   **주의: 이번 정찰은 이 명령을 돌리지 못했다 — 샌드박스가 `npm ci`(네트워크)와 `bash -c` 를 모두 거부했고, 워크트리에 `web/node_modules` 가 없다. 따라서 "base 의 lint 가 깨끗하다" 는 이번 회차에는 미확인이다**(직전 회차는 exit 0 으로 실측했다).

## 1단계 — 파이프라인 종료 코드 마스킹 제거
- 수용 기준:
  1) `.github/workflows/ci.yml` 의 "의존성 설치" 단계와 `.github/workflows/release.yml` 의 "소스 검사와 테스트" 단계에서 **`npm ci` 의 비정상 종료가 그 단계에서 즉시 실패**한다. 즉 `| tee` 가 종료 코드를 덮지 않는다.
  2) 로그 파일(`${RUNNER_TEMP}/npm-ci.log`)은 여전히 만들어진다 — `scripts/npm-audit-retry.sh` 의 장애 폴백이 `NPM_CI_LOG` 에서 `^found 0 vulnerabilities` 를 grep 하므로 **로그 생성을 없애면 안 된다**.
  3) 테스트(증거)가 증명할 것: 실패하는 `npm` 을 흉내낸 스텁을 PATH 앞에 두고 **수정 전 형태는 exit 0, 수정 후 형태는 exit != 0** 임을 보인다. 아래 "검증 명령" 의 harness 를 그대로 쓰면 된다.
  4) 어떤 단계도 느슨해지지 않는다: `continue-on-error`, `|| true`, `set +e`, 검사 삭제, audit 레벨 하향, `NPM_AUDIT_SKIP` 사용 금지. 이번 변경은 **엄격해지는 방향뿐**이다.
  5) `npm --prefix web ci && npm --prefix web run lint && npm --prefix web test && npm --prefix web run build` 가 통과한다(0단계에서 타입 오류가 드러났다면 그것을 고친 뒤).

- 건드릴 파일 (3개 안쪽):
  - `.github/workflows/ci.yml:77-78` — "의존성 설치" 단계에 **`shell: bash` 를 명시**한다. GitHub Actions 는 `shell: bash` 를 명시하면 `bash --noprofile --norc -eo pipefail {0}` 로 돌려 `pipefail` 이 켜진다(지정하지 않으면 `bash -e {0}`). 이 저장소는 이미 같은 관례를 쓴다 — `ci.yml:105`, `release.yml:51·105·113` 이 `shell: bash` 를 명시한다. 명시가 통하지 않는다고 판단되면 대안으로 블록 첫 줄에 `set -o pipefail` 을 넣는다(둘 중 하나만, 중복 금지).
  - `.github/workflows/release.yml:71-86` — "소스 검사와 테스트" 단계도 같은 처리. 이 블록은 `npm ci | tee` 외에 `go test`·`govulncheck`·`npm run lint` 를 한 덩어리로 돌리므로, `pipefail` 이 없으면 파이프를 쓰는 줄의 실패가 전부 묻힌다.
  - (선택, 회귀 방어) `scripts/check-workflow-pipefail.sh` — `.github/workflows/*.yml` 의 `run:` 블록 중 파이프(`|` 명령 파이프)를 쓰면서 `shell: bash` 도 `set -o pipefail` 도 없는 단계를 찾아 비정상 종료. 만들었으면 `ci.yml` 의 Go 검사 job 에 한 단계로만 끼운다(프런트 job 은 건드리지 말 것). YAML 파싱은 하지 말고 단순 텍스트 검사로 충분하다 — 과설계 금지.
  - 프로덕션 Go/TS 코드는 건드리지 않는다. `openapi/`, `migrations/`, `internal/auth/` 무변경.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 마스킹 재현/회귀 harness(네트워크 불필요, 임시 디렉터리에서):
    - 수정 전 형태: `bash -e -c 'false 2>&1 | tee /tmp/x.log; echo status=$?'` → `status=0` 이 나오는 것이 결함이다.
    - 수정 후 형태: `bash --noprofile --norc -eo pipefail -c 'false 2>&1 | tee /tmp/x.log'; echo $?` → `1` 이어야 한다.
    - 더 충실하게: `PATH` 앞에 `exit 1` 하는 `npm` 스텁을 두고 두 단계 본문을 그대로 추출해 각각 `bash -e` / `bash -eo pipefail` 로 실행해 종료 코드를 비교한다.
  - 워크플로 문법: `gh workflow view ci.yml` 가 되면 쓰고, 안 되면 `python3 -c "import yaml,sys;[yaml.safe_load(open(f)) for f in sys.argv[1:]]" .github/workflows/ci.yml .github/workflows/release.yml` 로 파싱만 확인한다. (`actionlint` 가 설치되어 있으면 `actionlint` 가 가장 좋다 — 설치 여부 미확인.)
  - 프런트: `npm --prefix web ci` → `npm --prefix web run lint` → `npm --prefix web test` → `npm --prefix web run build`.
  - 백엔드 무회귀(코드를 안 건드렸음을 확인): `gofmt -l .`(무출력) · `go vet ./...` · `go test -count=1 ./...` · `go build ./...` · `git diff --check`.
  - `./scripts/check-version.sh` (1.8.7) · `node scripts/check-screenshots.mjs` (30/30/21).

- 위험과 피할 것:
  - **보호 경로다**: `.github/workflows` 를 건드린다. 그래서 변경은 두 단계의 **셸 지정 한 줄씩**으로 최소화하고, 단계 이름·순서·명령 문자열은 그대로 둔다. 프로필의 "빌드·릴리스 경로를 건드렸으면 릴리스까지 확인" 교훈이 여기 걸린다.
  - `shell: bash` 를 `defaults.run.working-directory: web`(ci.yml:66) 와 혼동하지 말 것 — 프런트 job 은 이미 `working-directory` 가 web 이고, `shell` 은 단계 수준에 추가하는 별개 키다.
  - `pipefail` 을 켜면 **지금까지 묻혀 있던 실패가 드러나 CI 가 빨개질 수 있다**. 그것은 성공이지 회귀가 아니다 — 드러난 실패는 느슨하게 덮지 말고 원인을 적어 보고한다. 특히 `npm ci` 가 레지스트리/감사 문제로 실패하는 중이라면, 이제 "의존성 설치" 단계에서 멈춘다.
  - `scripts/npm-audit-retry.sh` 는 `set -uo pipefail` 로 이미 `pipefail` 을 쓰고 있다 — 그 파일은 건드리지 말 것.
  - `tee` 를 없애고 `npm ci > log 2>&1` 로 바꾸는 것도 종료 코드는 살지만 **라이브 로그가 사라진다**. 굳이 바꾸지 말고 `shell: bash` 로 해결하라.

- 차선 후보: **`/audit` 의 `page_size` 상한(200)을 실제 HTTP 응답으로 고정** — 직전 회차가 `internal/api/users_page_size_integration_test.go` 에 만든 `documentedPageSizeMaximum(t, path)` 헬퍼(openapi.yaml 에서 maximum 을 파싱)를 재사용해 `/audit` 서브테스트를 더한다. `openapi.yaml:559` 는 `maximum: 200`, `ListAudit`(internal/store/users.go:863-864)도 `pageBounds`(상한 200)를 타므로 **지금은 문서와 동작이 일치**한다 — 결함 수정이 아니라 회귀 방어다(가치 2 / 위험 1 / 작업량 S). fixture 는 audit_logs 260건이 필요하고 `/audit` 에는 `search` 파라미터가 없어 전역 격리가 안 되니 **행 수만 보고 `meta.total` 은 단정하지 말 것**.

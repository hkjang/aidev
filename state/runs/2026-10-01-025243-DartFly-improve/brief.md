- 과제: 수정 과제 — PR #18(a772071, 관리자 변경 이력 '저장된 결과' 탭 페이저)의 CI 실패를 실제 실패 로그로 특정해 고치기 (가치 5 / 위험 2 / 작업량 S~M)
- 왜: 지난 회차가 `verify-failed` 로 끝났고 같은 이유로 두 번 실패했습니다(러너 보고). 릴리스 게이트인 `.github/workflows/ci.yml` 이 빨간 동안에는 v2.79.0 이 나가지 못하고, 다음 회차의 변경까지 같은 빨간 위에 쌓입니다.

## 먼저 알아야 할 것 (정찰이 확인/미확인한 것)
- **정찰은 실패 로그를 못 봤습니다.** 이 세션은 샌드박스가 `gh`(네트워크)를 막고, 작업 디렉터리가 바뀐 뒤에는 `git show` 까지 승인 요구로 막혔습니다. 그래서 **아래 원인 후보는 가설이며, 구현자는 추측으로 고치지 말고 반드시 실제 로그부터 읽으세요.**
- **확인한 것**(정찰이 직접 읽음):
  - `.github/workflows/ci.yml` 은 이 저장소의 유일한 워크플로이고 잡 3개(`test`/`livedb`/`smoke`)입니다. 트리거는 `push:[main]` + `pull_request` + `workflow_dispatch`.
  - 실패한 변경 `a772071` 은 **프로덕션 파일 1개**(`internal/webui/js/admin-history.js`, +75/-20)와 **테스트 1개**(`test/js/admin-history.test.mjs`, +121)뿐입니다. Go·워크플로·스모크 하네스는 손대지 않았습니다.
  - `internal/webui/js/admin-history.js` 의 변경 내용: `EXEC_PAGE`→`PAGE` 로 통일, `savedOffset` 신설, `fetchSaved(at)` 신설, `loadSaved` 가 페이지내 검색(`searchEl.value`)·`{from}-{to} / 총 {total}건`·빈 페이지 재요청을 하고, `execPager(count,total)` 를 `pagerBar(at,count,total,go)` 로 일반화, 탭 전환/검색 입력에서 `savedOffset=0` 리셋, 저장 탭 placeholder 추가.
  - `test/smoke/pages.py` 는 `internal/webui/assets.go` 의 라우팅 표에서 경로를 읽어 **로그인 후 각 페이지를 열어 콘솔 오류·4xx 응답만** 봅니다. 탭을 클릭하지 않고, `admin-history` 전용 단언은 없습니다.
  - `test/smoke/run.sh`·`editor.py` 에도 `admin/history`·pager 관련 단언은 없습니다(`grep` 으로 확인 — 문자열 부재 확인이므로 "이 경로가 안 돈다"의 증명은 아닙니다).
  - 현재 워크트리는 `main@8591080` 이고 **a772071 을 담고 있지 않습니다**(브랜치 `auto/2026-10-01-0252` = main). 커밋 객체는 로컬에 있습니다.
- **미확인**: 어느 잡·어느 단계가 실패했는지, 실패 메시지, PR #18 의 head 브랜치 이름(로컬 브랜치 `auto/2026-10-01-0142` 이 유력하지만 확인 필요), livedb/smoke 가 최근 CI 에서 초록이었는지.

## 0단계 — 실패 로그부터 (이것을 건너뛰면 과제 실패)
```
gh pr view 18 --json number,headRefName,state,url
gh pr checks 18
gh run list --branch "$(gh pr view 18 --json headRefName -q .headRefName)" --limit 10
gh run view <RUN_ID> --log-failed | tail -120
```
두 번 실패한 두 run 의 `--log-failed` 를 **둘 다** 읽고 같은 단계·같은 메시지인지 확인하세요. 실패한 **잡 이름과 단계 이름**을 과제 기록에 그대로 옮겨 적으세요.

## 원인 후보 (정찰의 사전순위 — 로그가 말하는 것이 이깁니다)
1. **`test` 잡의 `go test -race ./...` 안 JS 회귀** — `internal/webui/jstest_test.go` 가 `test/js/*.test.mjs` 를 glob 으로 묶어 Node 로 돕니다. 구현자는 로컬 Node 로 18개 통과를 봤지만 CI 에는 `setup-node` 가 없어 **러너 기본 Node** 로 돕니다. 새 테스트 7개 중 타이머(debounce 250ms)·`URLSearchParams` 순서·`structuredClone` 같은 환경 의존이 섞였는지 `test/js/admin-history.test.mjs` 의 신규 7개를 다시 읽으세요. 로컬 재현: `node test/js/admin-history.test.mjs` 와 `go test -count=1 -v -run TestBrowserModuleTests ./internal/webui/...`(Skip 없이 PASS 가 출력에 보이는지 확인).
2. **`smoke` 잡** — `pip install --quiet playwright` / `playwright install --with-deps chromium`(러너 이미지 변화로 깨지는 단계) 또는 `bash test/smoke/run.sh` / `bash test/smoke/artifact.sh`. 이 경우 a772071 과 무관한 인프라 실패일 수 있습니다. 실패 시 `tail -50 /tmp/dartfly-smoke.log` 단계 출력이 로그에 있으니 먼저 보세요.
3. **`livedb` 잡** — `bash test/livedb/setup.sh --fast` 의 MariaDB 기동이 TLS 시각 문제로 실패하는 알려진 오탐(프로필의 '검증 함정'). **두 번 연속 같은 자리라면 오탐이 아니라 진짜 회귀일 수 있으니** 로그의 실패 테스트 이름을 확인하세요.
4. **`test` 잡의 `gofmt -w . && git diff --exit-code`** — a772071 에 Go 변경이 없어 가능성 낮음. 로그가 이것을 가리키면 워크트리에 커밋되지 않은 생성물이 있다는 뜻입니다.

## 수용 기준
1. 실패한 잡·단계·메시지를 로그에서 인용해 기록에 남겼다(가설이 아니라 인용).
2. 그 원인을 **워크플로를 느슨하게 하지 않고** 고쳤다 — `ci.yml` 에서 단계 삭제·`continue-on-error`·`|| true`·`DF_SMOKE_REQUIRE_BROWSER` 해제·테스트 단정 약화는 금지. (테스트의 기대값을 **새 계약으로 옮기는 것**은 허용되지만, 왜 그 값이 옳은지 커밋 메시지에 쓸 것.)
3. 같은 검증을 로컬에서 재현해 **고치기 전 red / 고친 뒤 green** 을 출력으로 보였다. 원인이 인프라(2·3번)라 로컬에서 red 를 못 만들면, 그 사실과 대신 확인한 것을 명시할 것.
4. `a772071` 의 기능(저장된 결과 탭 페이저·페이지내 검색)이 **그대로 남아 있다** — CI 를 통과시키려고 되돌리지 말 것.
5. 최종: `go test -count=1 -race ./...` · `go vet ./...` · `gofmt -l .`(무출력) · `go build ./cmd/dartfly` · `node test/js/admin-history.test.mjs` 전부 통과.

## 건드릴 파일 (총 3개 안쪽 목표)
- `test/js/admin-history.test.mjs` — 후보 1이면 환경 의존 단정을 고정값·가짜 타이머로 바꾸기(하네스 `test/js/load.mjs` 의 기존 패턴 재사용).
- `internal/webui/js/admin-history.js` — 테스트가 드러낸 실제 결함이 있을 때만. `loadSaved`/`pagerBar`/탭 전환 핸들러가 후보.
- `test/smoke/run.sh` 또는 `test/livedb/setup.sh` — 후보 2·3이 진짜 원인일 때만. **릴리스 검증 경로이므로 여기를 고치면 로컬에서 스모크 전체를 끝까지 돌려 통과를 보일 것**(운영자 반복 지시).
- `.github/workflows/ci.yml` — 러너 이미지 변화 같은 환경 원인이 로그로 증명됐을 때만, **검증 범위를 줄이지 않는 방향으로만**(예: `pip install` 을 venv/`--break-system-packages` 로 고치는 것은 허용, 단계 삭제는 금지).

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
# 작업 시작: 실패한 변경을 현재 브랜치에 올려놓고 시작 (워크트리는 main@8591080 이라 a772071 이 없습니다)
git cherry-pick a772071          # 충돌 없어야 정상(부모가 8591080)

go test -count=1 -race ./...
go vet ./... && gofmt -l . && go build ./cmd/dartfly
node test/js/admin-history.test.mjs
go test -count=1 -v -run TestBrowserModuleTests ./internal/webui/...   # Skip 없이 PASS 인지 출력 확인

# 화면·릴리스 경로를 건드렸을 때만 (수 분, Docker·Chromium 필요)
DF_SMOKE_REQUIRE_BROWSER=1 PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright bash test/smoke/run.sh
bash test/smoke/artifact.sh

# livedb 가 원인일 때만
bash test/livedb/setup.sh --fast
set -a && . /tmp/dartfly-livedb.env && set +a && go test -tags livedb ./...
```

## 위험과 피할 것
- **워크플로를 느슨하게 만들어 통과시키는 것은 명시적 금지**입니다(러너 지시). 단계 제거·`continue-on-error`·타임아웃 늘리기로 덮지 마세요.
- `internal/auth`·세션·SSO·`internal/store/mariadb/migrations`·`deploy/build-release.sh` 는 이번 과제와 무관합니다 — 열지 마세요.
- 스모크를 돌리는 **명령줄에 `/tmp/dartfly-smoke` 문자열을 넣지 마세요**(2026-09-26·09-28 에 exit 144 로 회차가 죽었습니다. 6b01d7b 로 하네스는 고쳐졌지만 습관은 유지).
- `PLAYWRIGHT_BROWSERS_PATH` 없이 `run.sh` 를 돌리면 이 환경(HOME 이 임시)에서는 브라우저 단계만 긴 파이썬 트레이스백으로 죽습니다 — CI 실패로 오인하지 마세요.
- vm+DOM 대역(`admin-history.test.mjs`)은 CSP·인라인 style 함정을 못 봅니다. 화면 동작을 바꿨다면 실제 바이너리로 확인하세요(운영자 반복 지시: 손으로 만든 대역으로 증명 금지).
- CI 로그가 "a772071 과 무관한 인프라 실패"라고 말하면, **그것이 결론이어도 됩니다** — 다만 재실행만으로 초록이 되는지 `gh run rerun --failed <RUN_ID>` 로 확인하고, 두 번 연속 같은 자리면 하네스를 고치세요.

## 차선 후보
실패가 순수 인프라 flake 로 판명되어 30분 안에 고칠 것이 없다면: **`test/smoke/run.sh` 가 브라우저 실행 파일을 못 찾을 때 `PLAYWRIGHT_BROWSERS_PATH` 를 안내하기** (가치 2 / 위험 1 / 작업량 S). `run.sh:609` 의 `import playwright` 검사는 통과하지만 브라우저 바이너리가 없으면 `pages.py`/`editor.py` 가 트레이스백으로 죽고 원인이 마지막 한 줄에만 있습니다. 브라우저 단계 직전에 launch 가능 여부를 확인해 한 줄 안내를 내보내면 회차마다 같은 시간을 잃지 않습니다. 릴리스 검증 경로이므로 고친 뒤 스모크 전체 통과를 보일 것.

# PR 처리기 노트 2026-10-01-040230-DartFly-shepherd — DartFly PR #19
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-01-025243-DartFly-improve)
# 회차 노트 2026-10-01-025243-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:52] base pinned — main@8591080
- [러너 02:52] autonomy release — 

## 정찰 노트
- 러너가 수정 과제(PR #18 CI 실패)를 배정해 새 후보를 고르지 않았습니다. 확인한 것: a772071 은 admin-history.js + admin-history.test.mjs 두 파일뿐이고, ci.yml 은 test/livedb/smoke 3잡이며 pages.py 는 라우팅 표 크롤 + 콘솔오류/4xx 만 봅니다(탭 클릭 없음) — 즉 스모크가 이 변경 때문에 깨질 경로는 보이지 않습니다.
- **확신 없는 곳**: 어느 잡·단계가 실패했는지 못 봤습니다. 샌드박스가 `gh pr checks 18` 을 승인 요구로 막았고, 작업 디렉터리가 바뀐 뒤에는 `git show` 까지 막혀 a772071 의 테스트 파일 본문도 못 읽었습니다. 과제서의 원인 후보 4개는 전부 가설입니다.
- 구현자가 조심할 것: ① `gh run view --log-failed` 로 두 실패 run 을 모두 읽고 원인을 **인용**한 뒤 고칠 것 — 가설로 고치면 세 번째 실패가 됩니다. ② 워크플로 완화(단계 삭제·continue-on-error·단정 약화)는 금지. ③ 워크트리는 main@8591080 이라 a772071 이 없습니다 — `git cherry-pick a772071` 로 시작하고 기능을 되돌리지 말 것.
- 프로필은 0일 전 것이고 이번에 읽은 ci.yml·pages.py·smoke 구조와 어긋나지 않아 다시 쓰지 않았습니다.
- [러너 02:58] scout done — 수정 과제 — PR #18(a772071, 관리자 변경 이력 '저장된 결과' 탭 페이저)의 CI 실패를 실제 실패 로그로 특정�

## 구현 노트
- PR #18 CI 실패는 **코드 결함이 아닙니다**. 세 잡 모두 `steps: []`(시작조차 못 함)이고 annotation 원문이 `The job was not started because recent account payments have failed or your spending limit needs to be increased`. rerun + workflow_dispatch 로 3회 연속 동일 — 저장소 코드로는 초록이 되지 않습니다. `gh` 는 `GH_CONFIG_DIR=/home/hkjang/.config/gh` 를 주면 이 샌드박스에서도 됩니다. `--log-failed` 가 빈 출력이면 로그가 아니라 `gh api repos/hkjang/DartFly/check-runs/<jobId>/annotations` 를 보세요.
- a772071 은 체리픽해 뒀고(978a809) 기능을 되돌리지 않았습니다. CI 가 못 돌린 세 잡을 로컬에서 대신 돌려 초록을 확인했습니다. 그 위에 과제서의 차선 후보를 구현했습니다(0b156d6): 스모크가 브라우저를 못 띄울 때 84줄 트레이스백 대신 원인 첫 줄 + 받는 법 + `PLAYWRIGHT_BROWSERS_PATH` 의 지금 값을 넉 줄로. **검사 범위 불변** — 못 띄우면 여전히 FAIL=1·exit 1 이고 건너뛰기로 바꾸지 않았습니다.
- **확신 없는 곳·검증 못 한 것**: ① CI 가 실제로 초록이 되는 것은 **확인 못 했습니다** — 결제가 풀려야 알 수 있습니다. 로컬 재현이 CI 와 같다는 보장은 러너 이미지 차이만큼 남습니다(특히 `smoke` 잡의 `pip install --quiet playwright`·`playwright install --with-deps chromium` 단계는 로컬에서 돌지 않았습니다 — 이미 설치된 것을 썼습니다). ② livedb 잡은 `internal/catalog TestLiveGeneratedDDLActuallyRuns` 하나가 실패하는데, 이는 이 샌드박스의 임시 HOME 때문에 node 경고가 DDL 에 섞이는 알려진 로컬 전용 오탐입니다(`ddl_live_test.go` 의 `command.Stderr = &out`). origin/main 에서 같은 실패가 난다는 것은 이번 회차에 **직접 재확인하지 않았고** 메모리 기록에 의존했습니다. ③ 새 프로브가 실패하는 다른 경로(의존 공유 라이브러리 누락 등)는 실행 파일 누락만 실제로 재현했고 나머지는 같은 except 로 잡힐 것이라는 추론입니다.
- 일부러 하지 않은 것: ci.yml 은 한 글자도 건드리지 않았습니다(원인이 워크플로가 아니고, 완화는 금지). `ddl_live_test.go` 의 stderr 혼입은 진짜 결함이지만 이번 과제 밖이라 ideas.json 에 3/1/S 로 올려 뒀습니다.
- 다음 역할이 조심할 것: 이 브랜치에는 커밋이 둘입니다(978a809 = a772071 체리픽, 0b156d6 = 하네스). 스모크를 돌릴 때는 `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 를 주세요 — 안 주면 이제 넉 줄짜리 안내와 함께 실패합니다(그게 이번 변경입니다). livedb 재현에는 Docker 가 필요하고 끝나면 `docker rm -f df-pg df-maria df-meta` 로 정리하세요.
- [러너 03:10] brief accepted — 채택(차선) — 과제서의 0단계 지시가 정확히 옳았습니다: 원인 후보 1~4는 전부 빗나갔고 실제 원인은 후보에 없던 "잡이 
- [러너 03:11] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인한 것: main 의 admin-history.js 를 /tmp 스크래치 트리에 놓고 같은 테스트를 돌려 저장된 결과 탭 8개가 **전부 실패**(pass 10/fail 8), HEAD 에서 18개 통과 — 테스트가 실제로 바뀐 경로를 지납니다. `go test -v -run TestBrowserModuleTests ./internal/webui/...` 에서 admin-history.test.mjs 가 Skip 없이 PASS, gofmt 무출력. run.sh 프로브는 경로 어긋남에서 rc=1+한 줄, 올바른 경로에서 rc=0+무출력으로 양방향 재현했고 launch 인수가 pages.py·editor.py 와 동일하며 `set -e` 가 없어 `probe=$(...)` 도 안전합니다. 서버측 /api/v1/query/saved 는 limit<=500·offset>=0 클램프 + 완전 바인딩 + `(? OR r.user_seq = ?)` 권한 조건이라 페이저가 권한면을 넓히지 않습니다.
- 못 본 것: CI 가 실제로 초록이 되는지(계정 결제 문제로 세 잡이 시작조차 못 함), smoke 잡의 playwright 설치 단계, livedb 잡 전체. 러너 노트의 "ts 파일" 경고는 오탐입니다 — diff 에 .ts 는 없고 .js/.mjs 는 제가 직접 돌렸습니다.
- 승인이어도 남는 우려 ①: admin-history.js 는 이동 바를 `#list` 안에 append 하는데 `empty()` 가 그것을 덮어써, 페이지내 검색이 0건이면 이동 바가 사라집니다(saved.js 는 별도 `#saved-pager` 로 피함). 검색이 savedOffset 을 0 으로 되돌리기까지 해서 저장된 결과 탭 검색은 사실상 첫 100건만 봅니다 — 다음 회차 후보.
- 승인이어도 남는 우려 ②: run.sh 의 `str(exc).strip().splitlines()[0]` 는 예외 메시지가 비면 IndexError(FAIL=1 은 유지, 안내만 트레이스백으로 퇴화). 릴리스 노트에 'CI 초록' 을 쓰면 안 됩니다.
- [러너 03:15] review approved — 리뷰 승인 (risk=low)
- [러너 03:15] pr created — https://github.com/hkjang/DartFly/pull/19
- [러너 03:16] ci failed — 성공이 아닌 검사: livedb=failure, smoke=failure, test=failure

## 수리 노트
- **틀린 지적**: "CI 3잡 실패". 세 잡 모두 `steps: 0`·3초이고 annotation 원문이 계정 결제 문제 — 잡이 시작조차 못 했습니다. 저장소 코드로 고칠 수 있는 것이 아니어서 ci.yml 은 한 글자도 안 건드렸습니다. 대신 세 잡이 할 일을 로컬에서 실제로 돌려 전부 초록을 확인했습니다(gofmt/vet/test -race/build, admin-history.test.mjs Skip 없이 PASS, run.sh·artifact.sh 둘 다 `== 통과 ==`).
- **맞은 지적**: 비평 노트 우려 ②. 프로브의 `...splitlines()[0]` 가 빈 메시지 예외에서 IndexError. 빈 줄을 걸러 첫 내용 줄을 쓰고, 없으면 예외 종류 이름을 쓰게 고쳤습니다(7fa00ad).
- **고쳤다는 증거**: run.sh 에서 프로브 본문을 추출해 playwright 스텁으로 예외 주입 — 고치기 전 빈 메시지에서 16줄 이중 트레이스백, 고친 뒤 한 줄(`_Boom`), 둘 다 rc=1. 여러 줄 메시지 출력은 전후 동일(회귀 없음). 진짜 크로미움으로도 양방향 재현.
- **여전히 확신 없는 곳**: ① CI 가 실제로 초록이 되는 것은 결제가 풀려야 알 수 있습니다 — 릴리스 노트에 'CI 초록' 을 쓰면 안 됩니다. ② smoke 잡의 `pip install playwright`·`playwright install --with-deps chromium` 단계는 로컬에서 돌지 않았습니다(이미 받아 둔 것을 씀). ③ livedb 잡은 이번에 돌리지 않았습니다 — 이번 diff 는 test/smoke/run.sh 한 줄뿐이라 무관하다고 판단했습니다.
- **남겨 둔 것**: 우려 ①(admin-history.js 이동 바를 `empty()` 가 덮어씀)은 비평가가 다음 회차 후보로 분류해 범위 밖으로 두었습니다. 프로브 회귀는 자동 테스트로 고정하지 못했습니다(run.sh 용 하네스가 저장소에 없음).

## 심사 노트
- 확인한 것: 새 테스트가 origin/main 소스에서 8개 실패·HEAD 에서 18/18 통과(직접 돌림), jstest_test.go 경유 Skip 없이 PASS. 프로브는 실제 크로미움으로 양방향(rc=0·무출력 / rc=1·한 줄) + 빈 메시지 예외 회귀(수리 전 11줄 이중 트레이스백 → 수리 후 한 줄, 둘 다 rc=1). 실패 시 FAIL=1·exit 1 로 검사 범위 불변.
- 확인한 것: 실제 바이너리 스모크 통과(31개 라우트 문제 0개, /admin/history 포함 — 임베드 모듈이 진짜 브라우저에서 초기화됨), gofmt·vet 무출력, go test -race ./... exit 0, build 성공. 서버 코드 0줄 변경이고 /api/v1/query/saved 의 COUNT·SELECT 가 같은 권한 술어에 전부 바인딩이라 total 기반 이동 바가 권한면을 넓히지 않습니다.
- 못 본 것: CI 초록(세 잡 steps 0 — 계정 결제 문제로 시작 못 함), CI 전용 pip/playwright install 단계, livedb 컨테이너 setup(이번 diff 에 .go 0개라 Go 동작은 main 과 동일). 릴리스 노트에 'CI 초록' 금지.
- 남는 우려(차단 아님): admin-history.js:105-108 — 페이지내 검색 0건이면 empty() 가 이동 바를 덮어써 그 페이지에서 이동 불가(검색어를 지우면 복구, 갇히지 않음). admin-history.js:86-93 — COUNT/SELECT 불일치 시 이동 바 없는 빈 페이지(실행 감사 탭에 이미 배포된 모양, 회귀 아님).
- 권고 근거: approve/merge, risk=low. 결함을 못 찾았고 인증·권한·데이터 경계가 그대로이며 되돌리기는 revert 로 끝납니다(마이그레이션·외부 상태 변경 없음). 스모크 하네스 변경이 페이저와 무관하게 섞인 것은 묶음 취향 문제로 봤습니다.

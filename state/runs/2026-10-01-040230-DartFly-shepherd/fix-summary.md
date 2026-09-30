# 수리 요약 — PR #19

- **CI 세 잡 실패는 코드 결함이 아닙니다.** run 36757310307 의 test/livedb/smoke 모두 `steps: 0`·3초·failure 이고, `gh api .../check-runs/<id>/annotations` 원문이 세 잡 모두 `The job was not started because recent account payments have failed or your spending limit needs to be increased`. 잡이 시작조차 못 했으므로 저장소 변경으로는 초록이 되지 않습니다 — 결제가 풀려야 합니다. ci.yml 은 건드리지 않았습니다(완화 금지).
- 대신 CI 가 못 돌린 것을 로컬에서 실제로 돌려 전부 초록을 확인했습니다: `gofmt -l .` 무출력, `go vet ./...` 무출력, `go test -race ./...` 전부 ok, `go build ./cmd/dartfly` 성공, `admin-history.test.mjs` 가 Skip 없이 PASS, `DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh` = `== 통과 ==`(브라우저 점검 31페이지 문제 0개), `bash test/smoke/artifact.sh` = `== 통과 ==`.
- **비평 노트의 우려 ② 는 맞았고 고쳤습니다**(7fa00ad). run.sh 프로브의 `str(exc).strip().splitlines()[0]` 는 메시지가 빈 예외에서 IndexError 를 내, 트레이스백을 없애려던 코드가 오히려 트레이스백 두 개를 내놓았습니다. 빈 줄을 걸러 첫 "내용 있는" 줄을 고르고, 그런 줄이 없으면 예외 종류 이름을 쓰도록 바꿨습니다.
- 그 고침을 **양방향으로 재현·검증**했습니다: run.sh 에서 프로브 본문을 추출해 playwright 스텁으로 예외를 주입 — 고치기 전 빈 메시지에서 16줄 이중 트레이스백, 고친 뒤 `_Boom` 한 줄(둘 다 rc=1). 여러 줄 메시지의 출력은 전후 동일. 진짜 크로미움으로도 경로 맞음=rc0·무출력, 경로 어긋남=rc1·한 줄.
- 남은 것: 비평 노트의 우려 ①(admin-history.js 이동 바를 `empty()` 가 덮어씀 → 페이지내 검색 0건이면 이동 바 사라짐)은 비평가가 "다음 회차 후보" 로 분류했고 이번 지적 범위 밖이라 손대지 않았습니다. 프로브 회귀를 자동 테스트로 고정하지는 못했습니다(run.sh 의 셸·python 을 도는 하네스가 저장소에 없고, 새로 만들면 지적 범위를 넘습니다).

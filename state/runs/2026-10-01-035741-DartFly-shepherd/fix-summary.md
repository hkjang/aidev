# 수리 결과 — 고칠 결함 없음 (커밋 없음)

- CI 3잡(test/livedb/smoke)은 코드 때문이 아니라 **잡이 시작조차 못 해서** 빨갛습니다: 세 잡 모두 `steps: 0`, 기동 3~4초, annotation 이 `The job was not started because recent account payments have failed or your spending limit needs to be increased` (`gh api repos/hkjang/DartFly/check-runs/{110024328526,110024328857,110024328997}/annotations`).
- a772071 에서 CI 3잡을 로컬로 그대로 재현: `gofmt -w . && git diff --exit-code` 통과, `go vet ./...` 통과, `go test -race ./...` 전 패키지 ok, `go build ./cmd/dartfly` 통과. JS 회귀는 Skip 아니라 실제 PASS(`TestBrowserModuleTests/admin-history.test.mjs` PASS, 15개 모듈 전부 PASS).
- smoke: `PLAYWRIGHT_BROWSERS_PATH=... DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh` → 브라우저 31페이지 0개 문제·편집기 점검 포함 `== 통과 ==`, 이어서 `bash test/smoke/artifact.sh` 도 `== 통과 ==`.
- livedb: `setup.sh --fast` 후 `go test -tags livedb ./...` 는 `internal/catalog TestLiveGeneratedDDLActuallyRuns` 하나만 실패 — node stderr 가 DDL 에 섞이는 로컬 전용 현상이고, **origin/main(8591080)을 별도 워크트리로 꺼내 같은 명령으로 동일 실패를 확인**했습니다(이 변경의 회귀 아님, 변경 파일과 무관).
- 따라서 코드로 초록을 만들 방법이 없어 커밋하지 않았습니다. 결제가 풀린 뒤 `gh workflow run CI --ref <branch>` 로 재실행하면 됩니다.

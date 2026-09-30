# 수리 결과 — 고칠 결함 없음 (커밋 없음)

- CI 세 잡(test/smoke/livedb)은 **시작조차 하지 않았습니다**: `gh api actions/runs/36770672415/jobs` 가 셋 다 `steps: 0`(3~5초 만료), `check-runs/<id>/annotations` 의 failure 메시지가 `The job was not started because recent account payments have failed or your spending limit needs to be increased`. 저장소 코드·워크플로의 결함이 아니라 GitHub 결제 문제입니다. ci.yml 은 손대지 않았습니다(규칙상 워크플로 수정도 금지).
- 대신 브랜치 HEAD(d61a5f0) 그대로 CI 세 잡을 로컬에서 전부 돌려 초록을 확인했습니다: `gofmt -w . && git diff --exit-code` OK / `go vet ./...` OK / `go test -race ./...` 전부 ok / `go build ./cmd/dartfly` OK.
- livedb: `set -a; . /tmp/dartfly-livedb.env; go test -count=1 -tags livedb ./...` → 실패 0(이번 회차가 고친 `TestGeneratedDDLIgnoresNodeStderr`·`TestLiveGeneratedDDLActuallyRuns` 포함).
- smoke: `DF_SMOKE_REQUIRE_BROWSER=1 PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright bash test/smoke/run.sh` → 통과(브라우저 점검 31페이지 문제 0, 편집기 점검 통과), `bash test/smoke/artifact.sh` → 통과. 비평가가 남긴 "브라우저 실제 스모크 미확인" 공백이 이것으로 메워졌습니다. JS 저장본 테스트는 Skip 아님을 `-v` 로 확인(13/18 파일 PASS 출력).
- 남는 확신 없는 곳: CI 가 실제로 초록이 되는지는 결제 복구 전까지 확인 불가입니다. 로컬 `$HOME` 이 런 디렉터리로 바뀌어 있어 `PLAYWRIGHT_BROWSERS_PATH=$HOME/...` 는 실패합니다 — 절대경로를 쓰세요(코드 문제 아님).

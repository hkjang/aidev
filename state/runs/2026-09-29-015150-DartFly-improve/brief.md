- 과제: 수정 과제 — 스모크(릴리스 검증) 하네스의 `pkill -f "$BIN"` 이 회차 자신의 에이전트 프로세스를 죽이는 문제 제거, 서버 PID 파일로 교체 (가치 5 / 위험 1 / 작업량 S)
- 왜: `test/smoke/run.sh` 의 `cleanup()`(:34)과 서버 기동 직전(:100)이 `pkill -f "$BIN"`(기본 `/tmp/dartfly-smoke`)을 호출하는데, `pkill -f` 는 그 문자열을 **모든 프로세스의 전체 명령줄**에 맞춥니다. 이 회차 안에서 실제로 확인했습니다 — `pgrep -af 'dartfly-smoke'` 가 지금 이 회차의 러너 프로세스(`1489011 timeout -k 30 600 claude -p …`), 에이전트 프로세스(`1489019 claude -p …`), 그리고 호출 셸(`1491599 /bin/bash -c …`)을 모두 잡습니다. 러너·에이전트가 걸리는 이유는 프롬프트 본문(원장의 보류 아이디어에 `go build -o /tmp/dartfly-smoke` 라는 문구가 들어 있음)이 그들의 명령줄이기 때문입니다. 즉 **릴리스 검증을 돌리는 순간 하네스가 자기를 실행한 에이전트를 SIGTERM 으로 죽입니다** — 마지막 회차가 `agent produced no result ()` 로 끝난 것과 2026-09-26·2026-09-28 두 회차가 exit 144 에 걸린 것이 같은 원인입니다. 서버 PID 를 파일에 적고 그 PID 만(그것이 실제로 dartfly 바이너리일 때만) 종료하면 검증 범위는 그대로인데 회차가 죽지 않습니다.
- 수용 기준:
  1) 고친 뒤 `test/smoke/run.sh` 어디에도 `pkill`/`pgrep -f` 같은 **전체 명령줄 매칭**이 남지 않는다(`grep -n 'pkill\|pgrep' test/smoke/run.sh` 무출력).
  2) 재현: 명령줄에 `/tmp/dartfly-smoke` 문자열이 있는 셸에서 `bash test/smoke/run.sh --down` 을 부르면 — 고치기 전에는 그 셸이 SIGTERM 으로 죽어 뒤 명령이 실행되지 않고, 고친 뒤에는 살아남아 `SURVIVED=0` 을 찍는다(정찰이 남긴 `probe.sh`, 아래 검증 명령 참고).
  3) `DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh` 가 전과 같이 끝까지 돌아 `== 통과 ==`·exit 0 이고, 종료 후 포트 9911 을 듣는 dartfly 프로세스가 남지 않는다. `--keep` 은 서버를 남기고, 이어지는 `bash test/smoke/run.sh --down` 이 그 서버를 정리한다(정리 실패 시 조용히 넘어가지 말고 메시지를 남길 것).
  4) 연속 실행이 여전히 안전하다: 앞 회차 서버가 떠 있는 상태에서 다시 돌려도 새 서버가 포트를 잡고 점검이 새 바이너리를 상대로 돈다(`ensurePortFree` 의 포트 주인 검사를 없애지 말 것 — 세 번 걸린 함정의 방지 장치).
  5) PID 재사용 안전장치: PID 파일의 값이 죽었거나 `/proc/<pid>/exe` 가 dartfly 가 아니면 죽이지 않는다.
- 건드릴 파일 (프로덕션 코드 0개 — 검증 하네스만, 파일 1개):
  - `test/smoke/run.sh:15-20` — `PIDFILE=${DF_SMOKE_PIDFILE:-/tmp/dartfly-smoke.pid}` 추가.
  - `test/smoke/run.sh:33-37 cleanup()` — `pkill -f "$BIN"` 을 PID 파일 기반 종료로 교체. ① PID 파일이 있으면 읽어 `readlink -f /proc/$pid/exe` 가 `$BIN` 의 실제 경로(또는 `*dartfly*`)일 때만 `kill "$pid"`, 최대 ~10초 대기 후에만 `kill -9`, ② PID 파일이 없거나 못 죽였으면 이미 있는 `portPID`/`portName`(:23-31)으로 `$PORT` 주인이 `*dartfly*` 일 때만 종료, ③ `rm -f "$PIDFILE"`. `docker rm -f "$CONTAINER"`·`rm -f "$JAR"` 는 그대로.
  - `test/smoke/run.sh:99-109` — 기동 직전의 `pkill -f "$BIN"`(:100)도 같은 PID 파일 기반 종료로 바꾸고, `nohup "$BIN" … &`(:109) 바로 뒤에 `echo $! > "$PIDFILE"` 를 추가. `sleep 1` · `ensurePortFree || exit 1`(:101-102) 순서는 유지.
  - `test/smoke/README.md` — 정리 방식이 PID 파일이라는 한 줄과 `DF_SMOKE_PIDFILE` 를 적어 두면 다음 사람이 헤매지 않습니다(README 63줄, 미정독 — 기존 환경변수 표 형식에 맞추세요).
  - `test/smoke/artifact.sh` 는 손댈 필요 없습니다 — 컨테이너 이름(`df-artifact-app`/`df-artifact-meta`)으로만 정리하고 `pkill` 이 없는 것을 확인했습니다(:25-30).
- 검증 명령:
  - 재현(고치기 전 FAIL → 고친 뒤 PASS): `bash /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-015150-DartFly-improve/probe.sh` — 명령줄에 `/tmp/dartfly-smoke` 를 담은 자식 셸에서 `run.sh --down` 을 부르고 그 셸이 살아남는지 봅니다. **이 스크립트를 부르는 당신의 셸 명령줄·heredoc 에 `/tmp/dartfly-smoke` 문자열을 넣지 마세요** — 고치기 전 상태에서는 당신 셸(과 이 회차의 claude 프로세스)이 같이 죽습니다. 이 정찰 세션은 샌드박스 권한 때문에 probe.sh 를 **실행하지 못했습니다(미실행)** — 대신 `pgrep -af 'dartfly-smoke'` 로 매칭 대상(러너/에이전트/셸 PID 3개)을 직접 확인했습니다. 구현자는 probe.sh 를 실제로 돌려 before/after 를 보이세요.
  - 전체(= CI `smoke` 잡이 돌리는 그 명령, `.github/workflows/ci.yml`): `DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh` (Docker·Chromium, 수 분) → 이어서 `bash test/smoke/artifact.sh`.
  - 회귀: `go test -race ./...` · `go vet ./...` · `gofmt -l .` · `bash -n test/smoke/run.sh`.
- 위험과 피할 것:
  - **워크플로를 느슨하게 만들어 통과시키는 것은 금지**: `DF_SMOKE_REQUIRE_BROWSER=1` 의 "playwright 없으면 실패" 분기(run.sh 끝), `ensurePortFree` 의 포트 주인 검사, CI 의 `artifact.sh` 단계를 제거·완화하지 마세요. 바꾸는 것은 "누구를 죽이는가" 하나뿐입니다.
  - `pkill -o`·`pgrep -f`·`ps | grep` 처럼 **전체 명령줄을 문자열로 맞추는 다른 방법으로 바꾸는 것은 같은 결함**입니다(여전히 프롬프트를 명령줄에 담은 에이전트가 걸립니다). PID 파일 + `/proc/<pid>/exe` 확인으로 가세요.
  - `kill -9` 를 곧바로 쓰지 말 것: SIGTERM 종료 로그(`/tmp/dartfly-smoke.log`)가 CI 의 "실패 시 서버 로그" 단계에서 쓰입니다.
  - `.github/workflows/ci.yml` 은 수정 불필요(이미 `bash test/smoke/run.sh` 를 부름) — 보호 경로이니 건드리지 마세요. `internal/**`·`cmd/**` 도 이번 과제 범위 밖입니다.
  - 다른 세션이 9911/53307 을 쥐고 있을 수 있습니다(`DF_SMOKE_PORT`·`DF_SMOKE_META_PORT` 로 피할 수 있음).
- 차선 후보: `admin-history.js` 저장결과 탭의 100건 벽 — `/saved` 의 pager 관례를 `loadSaved(admin-history.js:71)` 에 붙이기(가치 2 / 위험 2 / 작업량 M). 1순위가 이미 고쳐져 있을 때만 고르세요.

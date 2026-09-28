# 회차 노트 2026-09-29-015150-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:51] base pinned — main@6209bc4
- [러너 01:51] autonomy release —

## 정찰 노트
- 우선 과제(릴리스 검증이 같은 이유로 두 번 실패)의 원인을 찾았습니다: `test/smoke/run.sh:34,100` 의 `pkill -f "$BIN"` 이 **회차 자신의 프로세스를 죽입니다**. `pgrep -af 'dartfly-smoke'` 가 `1489011 timeout -k 30 600 claude -p …`·`1489019 claude -p …`·호출 셸을 잡는 것을 직접 봤습니다 — 프롬프트 본문에 `/tmp/dartfly-smoke` 문구가 있어 그것이 그들의 명령줄이기 때문입니다. `agent produced no result ()` 와 2026-09-26·09-28 의 exit 144 가 같은 뿌리이므로 다른 후보(저장결과 pager 등)는 제쳤습니다.
- 추측으로 적은 것: probe.sh 를 **실행하지 못했습니다**(샌드박스가 승인 요구) — before/after 관찰은 구현자가 해야 합니다. `test/smoke/README.md`(63줄)는 정독하지 않았고, exit 코드가 143 이 아니라 144 로 관측된 이유(timeout `-k 30` 이 끼는 것으로 추정)도 미확인입니다.
- 구현자 주의: 고치기 전 상태에서 `/tmp/dartfly-smoke` 문자열을 **당신 셸 명령줄·heredoc 에 넣지 마세요**(당신과 이 회차가 같이 죽습니다). `pgrep -f`/`ps|grep` 로 바꾸는 것은 같은 결함입니다. `ensurePortFree` 의 포트 주인 검사와 `DF_SMOKE_REQUIRE_BROWSER=1` 의 실패 분기는 절대 완화하지 마세요(워크플로 느슨화 금지).
- [러너 01:57] scout done — 수정 과제 — 스모크(릴리스 검증) 하네스의 `pkill -f "$BIN"` 이 회차 자신의 에이전트 프로세스를 죽이는 문

## 구현 노트
- `test/smoke/run.sh` 의 `cleanup()`·기동 직전이 BIN 경로 문자열로 프로세스를 싹 죽이던 것을 **서버 PID 파일**(`DF_SMOKE_PIDFILE`, 기본 `/tmp/dartfly-smoke.pid`) + `/proc/<pid>/exe` 확인으로 바꿨습니다. 프로덕션 코드 0개, 파일 2개(하네스 + README). 커밋 6b01d7b.
- **확신 없는 곳**: ① 생사 판정을 `kill -0` 이 아니라 `/proc/<pid>/exe` 유무로 합니다 — 정리 대상이 run.sh 의 자식이라 SIGTERM 뒤 좀비가 되고 `kill -0` 은 좀비에도 성공하기 때문인데, 좀비가 아닌 상황(다른 셸이 띄운 서버)까지 같은 판정으로 묶었습니다. ② 포트 주인 정리는 `ss` 가 PID 를 보여 줄 때만 됩니다(다른 사용자의 프로세스면 빈 값 → 조용히 건너뜀). 이건 `ensurePortFree` 의 기존 성질을 그대로 따른 것이고 새로 만든 구멍은 아닙니다.
- **검증 못 한 것**: 실제 GitHub Actions 러너 위에서는 돌려 보지 못했습니다(로컬 WSL 만). 정찰이 관측한 exit 코드가 143 이 아니라 144 였던 이유(`timeout -k 30` 추정)도 여전히 미확인입니다.
- **일부러 하지 않은 것**: `ensurePortFree` 의 포트 주인 검사, `DF_SMOKE_REQUIRE_BROWSER=1` 의 playwright 필수 분기, `.github/workflows/ci.yml`, `artifact.sh` 는 손대지 않았습니다(느슨화 금지·보호 경로). `artifact.sh` 에 같은 포트 주인 검사를 옮기는 건 ideas.json 에 다음 후보로 적었습니다.
- **다음 역할이 조심할 것**: 정리 방식을 명령줄 문자열 매칭으로 되돌리면 같은 사고가 그대로 재발합니다(회차 러너의 argv 에 프롬프트 본문이 있어 매칭됩니다). 이 하네스 검증은 Docker·Go·Chromium 이 있어야 돕니다 — 이 세션의 HOME 이 임시라 `python3 -m playwright install chromium`(114MB)을 먼저 받아야 브라우저 점검이 돕니다. 재현 스크립트 세 개(`probe-isolated.sh`·`pidreuse-test.sh`·`keep-down-test.sh`)는 run 디렉터리에 있고 저장소에는 넣지 않았습니다.
- [러너 02:09] brief accepted — 채택 — 지목한 두 자리(:34, :100)와 근거(러너·에이전트가 매칭 대상이라는 것)가 실제와 정확히 맞았고, 지시한 대로 PID 
- [러너 02:09] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인한 것: diff 2파일 전부 정독, main 판 run.sh 를 임시 디렉터리에 꺼내 probe-isolated.sh 로 red(자식 셸 SIGTERM 143) → HEAD 에서 green(SURVIVED=0) 을 직접 재현했고, pidreuse-test.sh 3항목도 다시 돌려 통과. bash -n 통과. 구현 노트의 "확신 없는 곳" 둘(좀비 판정, ss 가 PID 를 못 보는 경우)은 근거가 맞습니다.
- 못 본 것: 전체 스모크·artifact 는 제가 다시 돌리지 않고 smoke-full2.log·artifact.log(둘 다 EXIT=0, 브라우저 31페이지 포함)로 갈음했습니다. GitHub Actions 러너 위 실행은 여전히 미검증.
- 승인이어도 남는 우려: `isDartfly`(run.sh:41)의 정확 경로 비교는 연속 실행에서 **절대** 맞지 않습니다 — go build -o 가 옛 바이너리를 unlink 해 exe 가 `<경로> (deleted)` 가 되기 때문입니다(직접 확인). 기본 BIN 은 `*dartfly*` 부분일치가 구해 주지만 이번에 새로 문서화한 `DF_SMOKE_BIN` 이 'dartfly' 없는 경로면 정리가 거부돼 다음 회차가 ensurePortFree 에서 멈춥니다. `' (deleted)'` 접미사를 떼고 비교하면 사라지는 좁은 퇴행이라 차단하지 않았습니다.
- 부수: stopServer 가 포트 주인을 먼저 조용히 죽여 ensurePortFree 의 "이전에 남아 있던 서버를 정리합니다" 진단이 더는 찍히지 않습니다(run.sh:112 주석과 실제 흐름이 어긋남). README:49 의 `!!` 약속과 PID 재사용 거부 메시지 접두사도 다릅니다.
- 릴리즈: 사용자 영향 없음(하네스 전용). 보안·법무 차단 사유 없음 — 프로덕션 코드 0줄, 인증·비밀값·개인정보 경로 무관, 새 의존성 없음.
- [러너 02:14] review approved — 리뷰 승인 (risk=low)
- [러너 02:14] pr created — https://github.com/hkjang/DartFly/pull/17
- [러너 02:19] ci passed — 검사 3개 모두 success
- [러너 02:19] merge done — 6b01d7b
- [러너 02:24] release published — v2.78.0
- [러너 02:24] gh-release created — GitHub Release v2.78.0
- [러너 02:24] manifest ok — dartfly-v2.78.0.tar.gz dartfly-v2.78.0.tar.gz.sha256 
- [러너 02:24] assets uploaded — 2개
- [러너 02:24] assets verified — v2.78.0 자산 2개 (이전 v2.77.0: 2)

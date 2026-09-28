#!/usr/bin/env bash
# probe.sh 와 같은 재현이지만 BIN 토큰만 격리한 판.
#
# 정찰이 남긴 probe.sh 는 기본 BIN(/tmp/dartfly-smoke)을 씁니다. 그런데 이 회차의
# 러너(`timeout -k 30 2700 claude -p <프롬프트>`)와 에이전트 프로세스의 **명령줄에
# 그 문자열이 들어 있습니다**(프롬프트 본문이 argv). 그래서 고치기 전 상태에서
# probe.sh 를 돌리면 cleanup 의 `pkill -f "$BIN"` 이 회차 자신을 SIGTERM 으로 죽여
# baseline 을 볼 사람이 남지 않습니다. 그래서 여기서는 DF_SMOKE_BIN 을 아무도 쓰지
# 않는 토큰으로 바꿔 **같은 기계장치**(pkill -f 가 명령줄 전체를 맞춘다)만 재현합니다.
# 토큰은 이 파일 안에만 두세요 — 호출하는 셸의 명령줄에 넣으면 그 셸이 죽습니다.
#
# 고치기 전: 자식 셸이 SIGTERM 으로 죽어 SURVIVED 가 찍히지 않습니다  -> FAIL
# 고친 뒤:   SURVIVED=0 이 찍힙니다                                     -> PASS
set -u
REPO=${1:-/home/hkjang/.cache/auto-improve-wt/DartFly}
cd "$REPO" || exit 1
TOKEN=/tmp/df-selfmatch-probe-bin

out=$(bash -c ": $TOKEN ; DF_SMOKE_BIN=$TOKEN bash test/smoke/run.sh --down >/dev/null 2>&1 ; echo SURVIVED=\$?" 2>&1)
rc=$?
echo "-- 자식 셸 출력: [$out]"
echo "-- 자식 셸 종료코드: $rc"
case "$out" in
  *SURVIVED*) echo "PASS: 호출자 셸이 살아남았습니다." ; exit 0 ;;
  *) echo "FAIL: 호출자 셸이 pkill -f 에 죽었습니다(자기 매칭)." ; exit 1 ;;
esac

#!/usr/bin/env bash
# test/smoke/run.sh 의 `pkill -f "$BIN"` 자기 매칭 재현.
#
# 자식 셸의 **명령줄**에 BIN 문자열(/tmp/dartfly-smoke)을 담고, 그 안에서
# `bash test/smoke/run.sh --down`(= cleanup 만 실행) 을 부릅니다.
# 고치기 전: cleanup 의 pkill -f 가 그 자식 셸 자신을 SIGTERM 으로 죽여
#            SURVIVED 가 찍히지 않고 종료코드 143(또는 144) 이 나옵니다.
# 고친 뒤:   SURVIVED=0 이 찍히고 probe 는 "PASS" 로 끝납니다.
#
# 주의: 이 스크립트를 호출하는 당신의 셸 명령줄에는 /tmp/dartfly-smoke 를
#       넣지 마세요(같은 함정에 당신 셸이 걸립니다).
set -u
REPO=${1:-/home/hkjang/.cache/auto-improve-wt/DartFly}
cd "$REPO" || exit 1

out=$(bash -c ': /tmp/dartfly-smoke ; bash test/smoke/run.sh --down >/dev/null 2>&1 ; echo SURVIVED=$?' 2>&1)
rc=$?
echo "-- 자식 셸 출력: [$out]"
echo "-- 자식 셸 종료코드: $rc"
case "$out" in
  *SURVIVED*) echo "PASS: 호출자 셸이 살아남았습니다." ; exit 0 ;;
  *) echo "FAIL: 호출자 셸이 pkill -f 에 죽었습니다(자기 매칭)." ; exit 1 ;;
esac

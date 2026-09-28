#!/usr/bin/env bash
# PID 재사용 안전장치 확인: PID 파일의 값이 dartfly 가 아니면 죽이지 않아야 합니다.
# (수용 기준 5)
set -u
REPO=${1:-/home/hkjang/.cache/auto-improve-wt/DartFly}
cd "$REPO" || exit 1
RC=0
PF=$(mktemp /tmp/df-pidreuse-XXXXXX.pid)
# 아무도 안 쓰는 포트를 줘서 포트 주인 정리 경로가 남의 서버를 건드리지 않게 합니다.
FREEPORT=39117

echo "== 1) PID 파일이 dartfly 아닌 살아 있는 프로세스를 가리킬 때 =="
sleep 300 &
victim=$!
echo "$victim" > "$PF"
out=$(DF_SMOKE_PIDFILE="$PF" DF_SMOKE_PORT="$FREEPORT" bash test/smoke/run.sh --down 2>&1)
if kill -0 "$victim" 2>/dev/null; then
  echo "  PASS: 무관한 프로세스(pid $victim)가 살아남았습니다."
else
  echo "  FAIL: 무관한 프로세스를 죽였습니다."; RC=1
fi
case "$out" in
  *"dartfly 가 아닙니다"*) echo "  PASS: 조용히 넘어가지 않고 메시지를 남겼습니다." ;;
  *) echo "  FAIL: 메시지가 없습니다: [$out]"; RC=1 ;;
esac
kill "$victim" 2>/dev/null; wait "$victim" 2>/dev/null
[ -f "$PF" ] && { echo "  FAIL: PID 파일이 남았습니다."; RC=1; }

echo "== 2) PID 파일이 이미 죽은 PID 를 가리킬 때 =="
sleep 0.1 & dead=$!; wait "$dead" 2>/dev/null
echo "$dead" > "$PF"
out=$(DF_SMOKE_PIDFILE="$PF" DF_SMOKE_PORT="$FREEPORT" bash test/smoke/run.sh --down 2>&1)
case "$out" in
  *"정리했습니다."*) echo "  PASS: 조용히 정리하고 끝냈습니다." ;;
  *) echo "  FAIL: [$out]"; RC=1 ;;
esac
[ -f "$PF" ] && { echo "  FAIL: PID 파일이 남았습니다."; RC=1; }

echo "== 3) PID 파일이 아예 없을 때 =="
out=$(DF_SMOKE_PIDFILE="$PF" DF_SMOKE_PORT="$FREEPORT" bash test/smoke/run.sh --down 2>&1; echo "rc=$?")
case "$out" in
  *"rc=0"*) echo "  PASS: 정상 종료." ;;
  *) echo "  FAIL: [$out]"; RC=1 ;;
esac

rm -f "$PF"
[ "$RC" = 0 ] && echo "== PID 재사용 안전장치 통과 ==" || echo "== 실패 =="
exit $RC

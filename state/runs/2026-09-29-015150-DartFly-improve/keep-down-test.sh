#!/usr/bin/env bash
# 수용 기준 3(--keep 이 서버를 남기고 --down 이 그것을 정리)과
# 4(앞 회차 서버가 떠 있어도 새 서버가 포트를 잡고 점검이 새 바이너리를 상대로 돈다).
set -u
REPO=${1:-/home/hkjang/.cache/auto-improve-wt/DartFly}
cd "$REPO" || exit 1
PIDFILE=/tmp/dartfly-smoke.pid
PORT=9911
RC=0
owner() { ss -tlnpH "sport = :$PORT" 2>/dev/null | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2; }
exeOf() { readlink -f "/proc/$1/exe" 2>/dev/null; }

echo "=== A) --keep 이 서버를 남기는가 ==="
DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh --keep > /tmp/df-keepdown-A.log 2>&1
a_rc=$?
tail -2 /tmp/df-keepdown-A.log
pid1=$(cat "$PIDFILE" 2>/dev/null)
echo "  run --keep rc=$a_rc / PIDFILE=$pid1 / 포트주인=$(owner) / exe=$(exeOf "${pid1:-0}")"
[ "$a_rc" = 0 ] || { echo "  FAIL: --keep 실행이 실패했습니다"; RC=1; }
grep -q '== 통과 ==' /tmp/df-keepdown-A.log || { echo "  FAIL: 통과가 아닙니다"; RC=1; }
[ -n "$pid1" ] && [ "$(owner)" = "$pid1" ] \
  && echo "  PASS: 서버가 남아 포트를 듣고 있고 PID 파일과 일치합니다." \
  || { echo "  FAIL: --keep 이 서버를 남기지 못했습니다"; RC=1; }

echo "=== B) 앞 서버가 떠 있는 채로 다시 돌리기(연속 실행) ==="
DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh --keep > /tmp/df-keepdown-B.log 2>&1
b_rc=$?
tail -2 /tmp/df-keepdown-B.log
pid2=$(cat "$PIDFILE" 2>/dev/null)
echo "  run --keep rc=$b_rc / 새 PID=$pid2 / 포트주인=$(owner) / exe=$(exeOf "${pid2:-0}")"
[ "$b_rc" = 0 ] || { echo "  FAIL: 연속 실행이 실패했습니다"; RC=1; }
grep -q '== 통과 ==' /tmp/df-keepdown-B.log || { echo "  FAIL: 통과가 아닙니다"; RC=1; }
[ -n "$pid2" ] && [ "$pid2" != "$pid1" ] \
  && echo "  PASS: 새 프로세스가 떴습니다(옛 $pid1 -> 새 $pid2)." \
  || { echo "  FAIL: 옛 서버가 그대로입니다"; RC=1; }
[ "$(owner)" = "$pid2" ] \
  && echo "  PASS: 포트를 새 서버가 잡았습니다 — 점검은 새 바이너리를 상대로 돌았습니다." \
  || { echo "  FAIL: 포트 주인이 새 서버가 아닙니다"; RC=1; }
[ -z "$(exeOf "${pid1:-0}")" ] \
  && echo "  PASS: 옛 서버($pid1)는 정리됐습니다." \
  || { echo "  FAIL: 옛 서버가 살아 있습니다"; RC=1; }

echo "=== C) --down 이 --keep 이 남긴 서버를 정리하는가 ==="
out=$(bash test/smoke/run.sh --down 2>&1); d_rc=$?
echo "  --down 출력: [$out] rc=$d_rc"
[ -z "$(owner)" ] && [ -z "$(exeOf "${pid2:-0}")" ] \
  && echo "  PASS: 서버가 정리됐고 포트가 비었습니다." \
  || { echo "  FAIL: 서버가 남았습니다(포트주인=$(owner))"; RC=1; }
[ -f "$PIDFILE" ] && { echo "  FAIL: PID 파일이 남았습니다"; RC=1; } \
  || echo "  PASS: PID 파일이 지워졌습니다."

echo "=== D) 이 스크립트(호출자)가 살아남았는가 ==="
echo "  PASS: SURVIVED pid=$$ — run.sh 를 세 번 부르고도 살아 있습니다."

[ "$RC" = 0 ] && echo "=== keep/down/연속실행 전부 통과 ===" || echo "=== 실패 ==="
exit $RC

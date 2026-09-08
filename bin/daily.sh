#!/usr/bin/env bash
# 자율 개선 에이전트 정기 실행 래퍼 — Windows 작업 스케줄러가 wsl.exe 로 이 파일을 부른다.
# 비대화 셸이라 PATH 를 직접 잡고, flock 으로 겹침 실행을 막는다.
export HOME=/home/hkjang
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.nvm/versions/node/v22.23.1/bin:$HOME/miniconda3/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:/mnt/c/WINDOWS/system32:/mnt/c/WINDOWS/System32/WindowsPowerShell/v1.0"
export LANG=C.UTF-8
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$HERE/.." && pwd)"
mkdir -p "$REPO_DIR/logs" "$HOME/.auto-improve"
exec >>"$REPO_DIR/logs/cron.log" 2>&1
echo "===== $(date '+%F %T') start (args: ${*:---count 1})"
# 러너·프롬프트·원장이 원격에서 바뀌었을 수 있으니 먼저 따라간다
git -C "$REPO_DIR" pull -q --ff-only origin main || echo "warn: aidev pull failed, running with local copy"
# bash 는 스크립트를 실행하면서 읽으므로, 회차 도중 git pull 로 run.sh 가 바뀌면 깨질 수 있다 → 복사본으로 실행
export AIDEV_BIN="$HERE"
tmp=$(mktemp /tmp/aidev-run.XXXXXX.sh) && cp "$HERE/run.sh" "$tmp"
# 회차를 별도 세션(setsid)에서 돌린다 — wsl.exe 나 작업 스케줄러가 이 래퍼를 끊어도 회차는 살아남는다.
# (2026-09-08: 10분 넘는 회차가 매번 다음 트리거 시각에 통째로 사라져 개선 단계가 한 건도 완료되지 않았다.)
setsid flock -n "$HOME/.auto-improve/run.lock" bash -c 'trap "rm -f $0" EXIT; bash "$0" "$@"' "$tmp" "${@:---count 1}" &
child=$!
# 래퍼는 자식이 끝날 때까지 붙어 있는다(정상 종료 시 작업도 정상 종료). 끊기면 자식만 계속 돈다.
wait "$child"

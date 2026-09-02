#!/usr/bin/env bash
# 자율 개선 에이전트 정기 실행 래퍼 — Windows 작업 스케줄러가 wsl.exe 로 이 파일을 부른다.
# 비대화 셸이라 PATH 를 직접 잡고, flock 으로 겹침 실행을 막는다.
export HOME=/home/hkjang
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:$HOME/miniconda3/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
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
exec flock -n "$HOME/.auto-improve/run.lock" bash -c 'trap "rm -f $0" EXIT; bash "$0" "$@"' "$tmp" "${@:---count 1}"

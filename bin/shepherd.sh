#!/usr/bin/env bash
# PR 처리기(shepherd) — 검토 대기로 멈춘 러너 PR 을 시간마다 살펴 원인을 진단하고, 고칠 수 있는
# 것은 고쳐 머지·릴리즈 경로로 돌려보낸다.
#
# 메인 러너(bin/daily.sh)는 새 개선을 돌리고 fixer.sh 는 오류 회차를 다시 잡는다. 그 사이에서
# "PR 은 열렸는데 아무도 보지 않는" 것이 쌓였다 — 보호 파일에 걸려 사람 승인을 기다리는 것,
# 독립 리뷰가 거절한 것, CI 가 실패한 것, base 와 충돌한 것. 2026-09-17 기준 190건이 넘었다.
#
#   흐름: run.sh --shepherd
#     대상 수집(최근 14일 review-pending·verify-failed PR) → 진단(충돌/CI 실패/리뷰 거절/보호 파일/자율화 단계)
#     → 충돌은 리베이스, CI 실패·리뷰 거절은 PR 브랜치 위에서 수정 세션 → 재검증 → 푸시
#     → 엄격한 심사 세션(shepherd-review-prompt.md) → approve & risk≠high 면 aidev-approved 라벨 + 승인 기록
#     → 승인 스윕(approvals)이 CI 확인 뒤 승인 커밋에만 머지하고 릴리즈
#
# 사람만 승인할 수 있는 것(워크플로·비밀값·결제·LICENSE 를 건드린 PR, 자율화 단계가 approve 이하인
# 프로젝트, 심사 risk=high, 두 번 고쳐도 안 되는 것)은 건드리지 않고 진단만 남긴다 → 작업함에 표시.
#
# 락: 메인 러너·fixer 와 같은 run.lock 을 공유한다. 회차가 도는 중이면 끝날 때까지 기다린다
# (최대 SHEPHERD_LOCK_WAIT 초, 기본 50분). 스케줄: Windows 작업 스케줄러 AutoImproveShepherd (매시간).
#   끄기: state/NO-SHEPHERD 파일을 만들면 아무것도 하지 않는다.
#   설정: state/shepherd.env (SHEPHERD_MAX, SHEPHERD_DAILY_BUDGET, SHEPHERD_TRIES, SHEPHERD_NEVER_RE …)
set -uo pipefail
export HOME=/home/hkjang
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.nvm/versions/node/v22.23.1/bin:$HOME/miniconda3/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:/mnt/c/WINDOWS/system32:/mnt/c/WINDOWS/System32/WindowsPowerShell/v1.0"
export LANG=C.UTF-8
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; export AIDEV_BIN="$HERE"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; LOGS="$REPO_DIR/logs"
mkdir -p "$LOGS" "$HOME/.auto-improve"
exec >>"$LOGS/shepherd.log" 2>&1
echo "===== $(date '+%F %T') shepherd start"
[ -f "$STATE/NO-SHEPHERD" ] && { echo "NO-SHEPHERD — 중지 상태, 아무것도 하지 않는다"; exit 0; }
git -C "$REPO_DIR" pull -q --ff-only origin main || echo "warn: aidev pull 실패, 로컬로 진행"
# bash 는 스크립트를 실행하면서 읽으므로 복사본으로 돌린다 (daily.sh 와 같은 이유)
tmp=$(mktemp /tmp/aidev-shepherd.XXXXXX.sh) && cp "$HERE/run.sh" "$tmp"
wait_s=${SHEPHERD_LOCK_WAIT:-3000}
setsid flock -w "$wait_s" "$HOME/.auto-improve/run.lock" bash -c 'trap "rm -f $0" EXIT; bash "$0" --shepherd "$@"' "$tmp" "$@" &
child=$!; wait "$child" || echo "shepherd: 락을 ${wait_s}초 안에 잡지 못했거나 처리 실패 — 다음 시간에"
echo "===== $(date '+%F %T') shepherd done"

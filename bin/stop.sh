#!/usr/bin/env bash
# 긴급 중지 — 파일 하나로 러너를 안전한 경계에서 멈춘다. 다음 예약 실행뿐 아니라 진행 중인 회차도
# 단계 경계(에이전트 시작 전 / 머지 전 / 릴리즈 전)에서 확인한다.
#   bin/stop.sh all|merge|release|<프로젝트> on|off [사유]      bin/stop.sh status
#   all      : 새 회차를 시작하지 않고, 진행 중 회차는 머지·릴리즈를 하지 않는다
#   merge    : 머지만 중지 (PR 은 만든다)        release : 릴리즈·자산 게시만 중지
#   <프로젝트>: 그 프로젝트만 후보에서 제외하고 진행 중이면 머지·릴리즈를 하지 않는다
# 파일은 aidev 저장소에 커밋되므로 GitHub 에서 직접 만들어도 된다 (state/STOP, state/STOP-merge, state/STOP-release, state/STOP-<프로젝트>).
set -uo pipefail
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"; REPO_DIR="$(cd "$HERE/.." && pwd)"; STATE="$REPO_DIR/state"
what=${1:-status}; onoff=${2:-on}; why=${3:-}
if [ "$what" = status ]; then
  ls "$STATE"/STOP* 2>/dev/null | while read -r f; do printf '%s  %s\n' "$(basename "$f")" "$(cat "$f")"; done
  [ -z "$(ls "$STATE"/STOP* 2>/dev/null)" ] && echo "중지 없음"; exit 0
fi
case "$what" in all) f="$STATE/STOP";; merge|release) f="$STATE/STOP-$what";; *) f="$STATE/STOP-$what";; esac
if [ "$onoff" = on ]; then printf '%s by %s: %s\n' "$(date -Iseconds)" "$USER" "${why:-(사유 없음)}" > "$f"; echo "중지 설정: $(basename "$f")"
else rm -f "$f"; echo "중지 해제: $(basename "$f")"; fi
( cd "$REPO_DIR" && git add -A state && git commit -qm "stop: $what $onoff ${why:+— $why}" && git pull -q --rebase --autostash origin main && git push -q origin main ) >/dev/null 2>&1 && echo "aidev 에 반영됨" || echo "(로컬에만 반영 — 원격 푸시 실패)"

#!/usr/bin/env bash
# 오래된 러너 PR 정리 — 열린 채 잊힌 PR 을 세고, 원하면 닫으면서 일감은 ideas.json 으로 회수한다.
#
# 왜: 2026-09-24 기준 열린 러너 PR 132건 중 50건이 일주일을 넘겼다. 그 위에 새 PR 이 계속 쌓여
# madi(11건)·SecCheck(10건)·sqlon(10건)은 일주일 동안 머지 0건에 $100 을 썼다. 닫지 않으면
# WIP 상한(run.sh) 때문에 그 프로젝트는 영영 개선이 안 잡힌다 — 여는 쪽이 아니라 닫는 쪽을 자동화한다.
#
#   bin/pr-gc.sh              보고만 (기본) — state/stale-prs.md 를 쓰고 텔레그램으로 요약
#   bin/pr-gc.sh --close      STALE_PR_DAYS(기본 10)일 넘은 PR 을 닫고 제목을 ideas.json 으로 회수
#   끄기: state/NO-PR-GC
set -uo pipefail
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"; REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"
STALE_PR_DAYS=${STALE_PR_DAYS:-10}
CLOSE=0; [ "${1:-}" = "--close" ] && CLOSE=1
[ -f "$STATE/NO-PR-GC" ] && { echo "pr-gc: state/NO-PR-GC — 건너뜀"; exit 0; }
command -v gh >/dev/null 2>&1 || { echo "pr-gc: gh 없음"; exit 0; }

cutoff=$(date -d "-$STALE_PR_DAYS days" +%s)
tmp=$(mktemp); trap 'rm -f "$tmp"' EXIT
gh search prs --author=@me --state=open --limit 200 --json repository,number,title,createdAt,url > "$tmp" 2>/dev/null \
  || { echo "pr-gc: gh search 실패"; exit 0; }
total=$(jq 'length' "$tmp"); stale=0; closed=0
out="$STATE/stale-prs.md"; { echo "# 오래된 열린 PR ($(date '+%F %H:%M') 기준, ${STALE_PR_DAYS}일 초과)"; echo; } > "$out"

while IFS=$'\t' read -r proj num title url created; do
  [ -n "$num" ] || continue
  c=$(date -d "$created" +%s 2>/dev/null || echo 0); [ "$c" -lt "$cutoff" ] || continue
  age=$(( ($(date +%s) - c) / 86400 )); stale=$((stale+1))
  printf -- '- %s #%s (%s일) %s — %s\n' "$proj" "$num" "$age" "$url" "$title" >> "$out"
  [ "$CLOSE" -eq 1 ] || continue
  # 일감 회수: 아직 안 한 아이디어로 남겨 두고 닫는다 — 닫는다고 필요가 없어진 것은 아니다.
  f="$STATE/$proj.ideas.json"; [ -s "$f" ] || echo '[]' > "$f"
  jq --arg t "$title" --arg n "$url (${age}일 방치돼 닫힘 — 다시 하려면 처음부터)" \
     'if any(.[]?; .title==$t) then . else . + [{title:$t,value:3,risk:2,size:"M",status:"pending",note:$n}] end' \
     "$f" > "$f.tmp" 2>/dev/null && mv "$f.tmp" "$f" || rm -f "$f.tmp"
  if gh pr close "$url" --comment "자동 정리: ${age}일 동안 머지되지 않아 닫습니다. 필요하면 다시 엽니다 — 일감은 aidev state/$proj.ideas.json 에 남겼습니다." >/dev/null 2>&1; then
    closed=$((closed+1)); echo "닫음: $url (${age}일)"
  fi
done < <(jq -r '.[] | [.repository.name, (.number|tostring), .title, .url, .createdAt] | @tsv' "$tmp")

echo "pr-gc: 열림 $total · ${STALE_PR_DAYS}일 초과 $stale · 닫음 $closed"
if [ "$stale" -gt 0 ]; then
  msg="🧹 오래된 러너 PR ${stale}건 (전체 ${total}건 열림, ${STALE_PR_DAYS}일 초과)"
  if [ "$CLOSE" -eq 1 ]; then msg="$msg — ${closed}건 닫고 일감은 ideas 로 회수했습니다"
  else msg="$msg
닫으려면: bin/pr-gc.sh --close (목록 state/stale-prs.md)"; fi
  "$HERE/tg.sh" "$msg" >/dev/null 2>&1 || true
fi

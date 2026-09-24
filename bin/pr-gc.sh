#!/usr/bin/env bash
# 열린 러너 PR 정리 — 끝난 것을 닫고, 일감은 ideas.json 으로 회수한다.
#
# 왜: 러너는 여는 일만 자동이었다. 2026-09-24 기준 열린 PR 134건 중 46건은 PR 처리기가 이미
# "더는 못 한다"(수정·심사 N회 실패, 리베이스로 안 풀리는 충돌)고 판정하고 사람을 부른 것인데,
# 아무도 오지 않아 그대로 쌓였다. 그 위에 새 PR 이 또 얹히고, 시간이 지나면 충돌로 죽는다.
# 판정이 난 것은 닫는 것이 정직하다 — 필요는 ideas 로 남으니 다음 회차가 처음부터 다시 한다.
#
#   bin/pr-gc.sh                  보고 + 종결 판정 PR 정리 (기본; STUCK_PR_DAYS 기본 3일)
#   bin/pr-gc.sh --dry-run        무엇을 닫을지만 보여 준다
#   bin/pr-gc.sh --close          위에 더해 STALE_PR_DAYS(기본 10)일 넘은 PR 도 닫는다
#   bin/pr-gc.sh --report         아무것도 닫지 않고 목록만 (health.sh 가 쓰던 옛 동작)
#   끄기: state/NO-PR-GC · 개별 PR 보호: aidev-keep / aidev-approved / risk-accepted 라벨
set -uo pipefail
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"; REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; SHEPHERD_LOG="$STATE/shepherd.jsonl"
STALE_PR_DAYS=${STALE_PR_DAYS:-10}       # 그냥 오래된 PR (--close 를 줄 때만 닫는다)
STUCK_PR_DAYS=${STUCK_PR_DAYS:-3}        # 처리기가 종결 판정한 뒤 사람을 기다리는 날수
MAX_CLOSE=${MAX_CLOSE:-20}               # 한 번에 닫는 상한 — 잘못 잡았을 때 피해를 자른다
CLOSE_STALE=0; DRY=0; REPORT=0
for a in "$@"; do case "$a" in --close) CLOSE_STALE=1;; --dry-run) DRY=1;; --report) REPORT=1;; esac; done
[ -f "$STATE/NO-PR-GC" ] && { echo "pr-gc: state/NO-PR-GC — 건너뜀"; exit 0; }
command -v gh >/dev/null 2>&1 || { echo "pr-gc: gh 없음"; exit 0; }

now=$(date +%s)
tmp=$(mktemp); trap 'rm -f "$tmp"' EXIT
gh search prs --author=@me --state=open --limit 200 --json repository,number,title,createdAt,url,labels > "$tmp" 2>/dev/null \
  || { echo "pr-gc: gh search 실패"; exit 0; }
total=$(jq 'length' "$tmp"); stale=0; stuck=0; closed=0; relabel=0
out="$STATE/stale-prs.md"; { echo "# 열린 러너 PR ($(date '+%F %H:%M') 기준)"; echo; } > "$out"

# 일감 회수 — 닫는다고 필요가 없어진 것은 아니다. 제목을 pending 아이디어로 남긴다.
recover_idea(){ # $1=프로젝트 $2=제목 $3=메모
  local f="$STATE/$1.ideas.json"; [ -s "$f" ] || echo '[]' > "$f"
  jq --arg t "$2" --arg n "$3" --arg d "$(date +%F)" \
     'if any(.[]?; .title==$t) then . else . + [{title:$t,value:3,risk:2,size:"M",status:"pending",updated:$d,note:$n}] end' \
     "$f" > "$f.tmp" 2>/dev/null && mv "$f.tmp" "$f" || rm -f "$f.tmp"
}

close_pr(){ # $1=url $2=프로젝트 $3=제목 $4=사유 한 줄
  [ "$REPORT" -eq 1 ] && return 0
  if [ "$DRY" -eq 1 ]; then echo "[dry-run] 닫음: $1 — $4"; closed=$((closed+1)); return 0; fi
  [ "$closed" -ge "$MAX_CLOSE" ] && { echo "상한 ${MAX_CLOSE}건 도달 — 나머지는 다음 회차에"; return 1; }
  recover_idea "$2" "$3" "$1 ($4) 로 닫힘 — 다시 하려면 처음부터"
  if gh pr close "$1" --comment "자동 정리: $4

일감은 aidev 의 state/$2.ideas.json 에 남겼습니다 — 다음 회차가 처음부터 다시 시도합니다.
계속 살려 두려면 다시 열고 aidev-keep 라벨을 붙여 주세요." >/dev/null 2>&1; then
    closed=$((closed+1)); echo "닫음: $1 — $4"; return 0
  fi
  return 1
}

while IFS=$'\t' read -r proj num title url created labels; do
  [ -n "$num" ] || continue
  case ",$labels," in *,aidev-keep,*|*,aidev-approved,*|*,risk-accepted,*) continue;; esac
  c=$(date -d "$created" +%s 2>/dev/null || echo "$now"); age=$(( (now - c) / 86400 ))

  # 자가 치유: 승인 기록은 있는데 라벨이 없는 PR — 승인 스윕은 라벨만 보므로 영영 머지되지 않는다.
  # 2026-09-24 에 gh pr edit 가 깨져 55건이 이 상태로 쌓였다. 기록의 SHA 가 지금 head 와 같을 때만 붙인다.
  asha=$(jq -r --arg pr "$url" 'select(.pr==$pr) | (.sha // "")' "$STATE/approvals.jsonl" 2>/dev/null | tail -1)
  if [ -n "$asha" ]; then
    ahead=$(gh pr view "$url" --json headRefOid --jq .headRefOid 2>/dev/null)
    if [ "$asha" = "$ahead" ] && [ "$DRY" -eq 0 ] && [ "$REPORT" -eq 0 ]; then
      slug=$(sed -E 's#^https?://[^/]+/([^/]+/[^/]+)/pull/[0-9]+.*$#\1#' <<<"$url")
      if gh api -X POST "repos/$slug/issues/$num/labels" -f "labels[]=aidev-approved" >/dev/null 2>&1; then
        relabel=$((relabel+1)); echo "라벨 복구: $url (승인 기록은 있는데 라벨이 없었다)"
      fi
    elif [ "$asha" = "$ahead" ]; then
      relabel=$((relabel+1)); echo "[dry-run] 라벨 복구: $url"
    fi
    continue
  fi

  # 1) PR 처리기가 종결 판정한 것 — 판정 뒤 STUCK_PR_DAYS 일이 지나도 사람이 안 오면 닫는다
  last=$(jq -c --arg pr "$url" 'select(.pr==$pr)' "$SHEPHERD_LOG" 2>/dev/null | tail -1)
  if [ -n "$last" ] && [ "$(jq -r '.action // ""' <<<"$last")" = needs-human ]; then
    detail=$(jq -r '.detail // ""' <<<"$last")
    case "$detail" in
      *"시도했지만 통과하지 못함"*|*"자동 리베이스로 풀리지 않음"*)
        vts=$(jq -r '.ts // ""' <<<"$last"); v=$(date -d "$vts" +%s 2>/dev/null || echo "$now")
        vage=$(( (now - v) / 86400 ))
        if [ "$vage" -ge "$STUCK_PR_DAYS" ]; then
          stuck=$((stuck+1))
          printf -- '- [종결] %s #%s (판정 %s일 전) %s — %s\n' "$proj" "$num" "$vage" "$url" "$detail" >> "$out"
          close_pr "$url" "$proj" "$title" "PR 처리기가 ${vage}일 전에 더 진행할 수 없다고 판정했고(${detail%% —*}) 그 뒤 사람 손이 닿지 않았습니다"
          continue
        fi;;
    esac
  fi

  # 2) 그냥 오래된 것 — 목록만 남기고, --close 를 줄 때만 닫는다
  if [ "$age" -ge "$STALE_PR_DAYS" ]; then
    stale=$((stale+1))
    printf -- '- [오래됨] %s #%s (%s일) %s — %s\n' "$proj" "$num" "$age" "$url" "$title" >> "$out"
    [ "$CLOSE_STALE" -eq 1 ] && close_pr "$url" "$proj" "$title" "${age}일 동안 머지되지 않았습니다"
  fi
done < <(jq -r '.[] | [.repository.name, (.number|tostring), .title, .url, .createdAt, ([.labels[]?.name]|join(","))] | @tsv' "$tmp")

echo "pr-gc: 열림 $total · 종결 판정 $stuck · ${STALE_PR_DAYS}일 초과 $stale · 닫음 $closed · 라벨 복구 $relabel"
if [ "$closed" -gt 0 ] || [ "$stale" -gt 0 ]; then
  msg="🧹 러너 PR 정리 — 열림 ${total}건 중 종결 판정 ${stuck}건, ${STALE_PR_DAYS}일 초과 ${stale}건"
  [ "$closed" -gt 0 ] && msg="$msg
${closed}건을 닫고 일감은 ideas 로 회수했습니다."
  [ "$stale" -gt 0 ] && [ "$CLOSE_STALE" -eq 0 ] && msg="$msg
오래된 것까지 닫으려면: bin/pr-gc.sh --close (목록 state/stale-prs.md)"
  [ "$DRY" -eq 0 ] && [ "$REPORT" -eq 0 ] && { "$HERE/tg.sh" "$msg" >/dev/null 2>&1 || true; }
fi

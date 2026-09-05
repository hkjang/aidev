#!/usr/bin/env bash
# 주간 "사람 결정 필요" 다이제스트 — 월요일 첫 회차에 한 번, 지난 7일 동안 사람 손이 필요해진 것들을 이슈 하나로 묶는다.
#   열린 PR(리뷰 보류·보호 파일·CI 실패), 롤백 PR, 릴리즈 관례 없어 skipped 된 프로젝트, 수정 큐, 새 교훈
set -uo pipefail
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; RUNS="$REPO_DIR/docs/data/runs.jsonl"; SITE="https://hkjang.github.io/aidev"
week=$(date +%G-W%V); stamp="$STATE/.digest-$week"
[ "${1:-}" = "--force" ] || [ "$(date +%u)" = 1 ] || exit 0
[ -f "$stamp" ] && exit 0
since=$(date -d '-7 days' +%F)
sec(){ local t=$1; shift; local body; body=$("$@" 2>/dev/null); [ -n "$body" ] && printf '### %s\n%s\n\n' "$t" "$body"; }
open_prs(){ jq -r --arg s "$since" 'select(.date >= $s) | select(.result|test("PR open")) | "- \(.date) **\(.project)** — \(.result)"' "$RUNS"; }
rollbacks(){ jq -r --arg s "$since" 'select(.date >= $s) | select(.result|test("rollback")) | "- \(.date) **\(.project)** — \(.result)"' "$RUNS"; }
skipped(){ jq -r --arg s "$since" 'select(.date >= $s) | select(.result|test("release skipped")) | .project' "$RUNS" | sort -u | sed 's/^/- /'; }
fixq(){ [ -s "$STATE/fix-queue.tsv" ] && awk -F'\t' '{print "- **"$1"** — "$2}' "$STATE/fix-queue.tsv"; }
lessons(){ jq -r --arg s "$since" 'select(.date >= $s) | "- \(.date) **\(.project)** [\(.kind)] \(.detail)"' "$STATE/lessons.jsonl" 2>/dev/null; }
totals=$(jq -s --arg s "$since" '[.[] | select(.date >= $s)] | {runs:length, released:([.[]|select(.result|test("released v?[0-9]"))]|length), merged:([.[]|select(.result|test("merged"))]|length)}' "$RUNS" 2>/dev/null)
body="지난 7일($since ~ $(date +%F)) 요약: $(jq -r '"회차 \(.runs) · 릴리즈 \(.released) · 머지 \(.merged)"' <<<"$totals")

$(sec "🧐 사람이 판단할 PR (리뷰 보류·보호 파일·CI 실패)" open_prs)$(sec "⏪ 롤백 PR" rollbacks)$(sec "🏷️ 릴리즈 관례가 없어 건너뛴 프로젝트 — 관례를 정해 주세요" skipped)$(sec "🛠️ 수정 과제 큐" fixq)$(sec "📚 새 교훈" lessons)
[대시보드]($SITE/) · [주간 보고]($SITE/weekly/$week/) · [교훈]($SITE/lessons/)"
cd "$REPO_DIR" || exit 0
gh label create digest --color 5319E7 --description "주간 결정 필요 다이제스트" >/dev/null 2>&1 || true
gh issue create --title "📋 결정 필요 주간 다이제스트 $week" --label digest --body "$body" >/dev/null 2>&1 && { touch "$stamp"; echo "digest: issue created for $week"; }
exit 0

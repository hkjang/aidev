#!/usr/bin/env bash
# 수동 트리거 — aidev 저장소에 라벨 `run` 이 붙은 열린 이슈(제목 "run: <프로젝트>" 또는 본문 첫 줄)를
# state/run-queue.tsv 에 넣는다. 러너는 수정 큐 다음으로 이 큐를 먼저 집고, 끝나면 이슈에 결과를 달고 닫는다.
# 폰에서 GitHub 앱으로 이슈 하나 만들면 다음 회차에 그 프로젝트가 돈다.
set -uo pipefail
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
Q="$REPO_DIR/state/run-queue.tsv"; touch "$Q"
cd "$REPO_DIR" || exit 0
gh label create run --color 0E8A16 --description "이 프로젝트를 다음 회차에 우선 실행" >/dev/null 2>&1 || true
gh issue list --label run --state open --json number,title,body --jq '.[] | "\(.number)\t\(.title)\t\(.body // "" | split("\n")[0])"' 2>/dev/null \
| while IFS=$'\t' read -r num title first; do
  p=$(sed -nE 's/^[[:space:]]*(run|실행)[[:space:]]*:[[:space:]]*([A-Za-z0-9_.-]+).*/\2/p' <<<"$title")
  [ -n "$p" ] || p=$(sed -nE 's/^[[:space:]]*(run|실행)[[:space:]]*:[[:space:]]*([A-Za-z0-9_.-]+).*/\2/p' <<<"$first")
  [ -n "$p" ] || { gh issue comment "$num" --body "제목을 \`run: <프로젝트>\` 형식으로 적어 주세요. 예: \`run: weekly\`" >/dev/null 2>&1; gh issue edit "$num" --remove-label run >/dev/null 2>&1; continue; }
  grep -q -P "^$p\t" "$Q" || { printf '%s\t%s\t%s\n' "$p" "이슈 #$num: $title" "$num" >> "$Q"; echo "inbox: queued $p (#$num)"; }
done
exit 0
